package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/pion/webrtc/v3"
	"log"
	"net/http"
	"time"
)

const (
	WebsocketConTimeout = 30
	RtcpPLIInterval     = 1
	CandidateTimeout    = 3
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

func Serve(ctx *gin.Context) {
	var err error
	ctx1, cancel1 := context.WithTimeout(ctx, WebsocketConTimeout)
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
		cancel1()
	}()

	wh.pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		log.Println("candidate: ", c)
		wh.candidates <- c
	})

	wh.pc.OnTrack(wh.worker.OnTrack)

	wh.pc.OnDataChannel(wh.worker.OnDataChannel)

	wh.pc.OnICEConnectionStateChange(func(connectionState webrtc.ICEConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", connectionState.String())

		switch connectionState {
		case webrtc.ICEConnectionStateDisconnected:
			return
		case webrtc.ICEConnectionStateConnected:
			go wh.worker.Run(ctx)
			cancel1()
		}
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
			case <-wh.candidates:
				c = <-wh.candidates
			case <-t.C:
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
