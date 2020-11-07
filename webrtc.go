package main

import (
	"encoding/json"
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"github.com/pion/rtp"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"time"
)


const (
	WebsocketConTimeout = 30
	RtcpPLIInterval = 200
	CandidateTimeout = 3
)

var (
	upgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type Message struct {
	Key string		`json:"key"`
	Value string	`json:"value"`
}

type WebrtcHandler struct {
	pc        *webrtc.PeerConnection
	rtpChan   chan *rtp.Packet
	dataChan  chan string
	candidates chan *webrtc.ICECandidate

	isClosed  bool
	wsCon *websocket.Conn

}

func NewWebrtcHandler() WebrtcHandler {
	wh := WebrtcHandler{
		pc:         nil,
		rtpChan:    nil,
		dataChan:   nil,
		candidates: nil,
		isClosed:   false,
		wsCon:      nil,
	}
	return wh
}


func Serve(ctx *gin.Context) {
	var err error
	ctx1, cancel := context.WithTimeout(ctx, WebsocketConTimeout)
	wh := NewWebrtcHandler()
	wh.wsCon, err = upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Printf("error upgrading %s", err)
		return
	}

	defer func() {
		err = wh.wsCon.Close()
		if err != nil {
			log.Println("Worker websocket close err: ", err)
		}
		err = wh.pc.Close()
		if err != nil {
			log.Println("Worker rtp close err: ", err)
		}
		cancel()
	}()

	wh.pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Println("candidate: ", c)
		wh.candidates <- c
	})

	wh.pc.OnTrack(func(track *webrtc.Track, receiver *webrtc.RTPReceiver) {
		go func() {
			ticker := time.NewTicker(time.Millisecond * RtcpPLIInterval)
			for range ticker.C {
				errSend := wh.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}})
				if errSend != nil {
					fmt.Println(errSend)
				}
			}
		}()

		for {
			rtpPacket, readErr := track.ReadRTP()
			if readErr != nil {
				panic(readErr)
			}
			select {
			case wh.rtpChan <- rtpPacket:
				log.Println("rtp packet size: ", len(rtpPacket.Payload))
			default:
			}
		}
	})

	wh.pc.OnDataChannel(func(d *webrtc.DataChannel) {
		log.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

		d.OnOpen(func() {
			log.Printf("Data channel '%s'-'%d' open. Random messages will now be sent to any connected DataChannels every 5 seconds\n", d.Label(), d.ID())
			for range time.NewTicker(RtcpPLIInterval * time.Millisecond).C {
				select {
				case <-wh.dataChan:
					message := <-wh.dataChan
					log.Printf("Sending '%s'\n", message)

					sendErr := d.SendText(message)
					if sendErr != nil {
						panic(sendErr)
					}
				default:
					break
				}
			}
		})

		d.OnMessage(func(msg webrtc.DataChannelMessage) {
			fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
		})
	})

	wh.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())
	})

	go wh.Signal(ctx1)

	select {
	case <-ctx1.Done():
		log.Println("Request Close ", ctx.Request.Header["request_id"])
		return
	}
}

func (wh *WebrtcHandler) Signal(ctx context.Context) {
	msg, resp := Message{}, Message{}
	for {
		mt, message, err := wh.wsCon.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		log.Printf("recv: %s", message)

		err = json.Unmarshal(message, &msg)
		if err != nil {
			log.Println("Invalid message from websocket")
		}
		switch msg.Key {
		case "sdp":
			offer := webrtc.SessionDescription{}
			err = json.Unmarshal([]byte(msg.Value), &offer)
			if err != nil {
				log.Println("remote sdp unmarshal error ", err)
			}
			if err := wh.pc.SetRemoteDescription(offer); err != nil {
				panic(err)
			}
			answer, err := wh.pc.CreateAnswer(nil)
			if err != nil {
				panic(err)
			} else if err = wh.pc.SetLocalDescription(answer); err != nil {
				panic(err)
			}
			response, _ := json.Marshal(*wh.pc.LocalDescription())
			resp.Key, resp.Value = "sdp", string(response)

		case "candidate":
			remoteCandidate := webrtc.ICECandidateInit{}
			err = json.Unmarshal([]byte(msg.Value), &remoteCandidate)

			if err != nil {
				log.Println("remote candidate unmarshal error ", err)
			}
			if candidateErr := wh.pc.AddICECandidate(remoteCandidate); candidateErr != nil {
				log.Println("Add ice candidate err: ", candidateErr)
			}
			t := time.NewTimer(time.Second * CandidateTimeout)
			var c *webrtc.ICECandidate
			select {
			case <- wh.candidates:
				c = <- wh.candidates
			case <- t.C:
				log.Println("Wait candidate timeout ", len(wh.candidates))
			}
			resp.Key, resp.Value = "candidate", c.ToJSON().Candidate
		}
		b, err := json.Marshal(resp)
		err = wh.wsCon.WriteMessage(mt, b)
		if err != nil {
			log.Println("write:", err)
			break
		}

		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}