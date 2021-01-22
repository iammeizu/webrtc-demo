package worker

import (
	"context"
	"fmt"
	"github.com/iammeizu/webrtc-demo/parser"
	"github.com/pion/rtcp"
	"github.com/pion/webrtc/v3"
	"log"
	"time"
)

const (
	Width     = 480
	Height    = 640
	FrameSize = Width * Height * 3

	Bitrate = 2 * 1024 * 1024
)

type WorkerHandler struct {
	resultChan		chan string
	pc     			*webrtc.PeerConnection
	parser 			*parser.Parser
}

func NewWorkerHandler(p *webrtc.PeerConnection) *WorkerHandler {

	wh := WorkerHandler{
		pc:         p,
		parser:     parser.NewParser(Width, Height),
		resultChan: make(chan string, 0),
	}
	return &wh
}

func (wh *WorkerHandler) Run(ctx context.Context) {
	for {
		image := wh.parser.PopImage(FrameSize)
		log.Println("image size", len(image))
		// do something with image
		wh.resultChan <- "pop image success"

		select {
		case <-ctx.Done():
			return
		default:
			continue
		}
	}
}

func (wh *WorkerHandler) OnTrack(track *webrtc.Track, receiver *webrtc.RTPReceiver) {

	go func() {
		ticker := time.NewTicker(time.Second * RtcpPLIInterval)
		for range ticker.C {

			// Send PLI to get a full frame
			errSend := wh.pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: track.SSRC()}})
			if errSend != nil {
				fmt.Println("Send PLI err", errSend)
			}

			// Send REMB to set bitrate
			if writeErr := wh.pc.WriteRTCP([]rtcp.Packet{&rtcp.ReceiverEstimatedMaximumBitrate{Bitrate: Bitrate, SenderSSRC: track.SSRC()}}); writeErr != nil {
				fmt.Println("Send REMB err", writeErr)
			}

		}
	}()

	for {
		rtpPacket, readErr := track.ReadRTP()
		if readErr != nil {
			log.Println("Read rtp err", rtpPacket)
		}

		if wh.parser.IsClosed {
			break
		}

		switch track.Kind() {
		case webrtc.RTPCodecTypeAudio:
			wh.parser.PushAudio(rtpPacket)
		case webrtc.RTPCodecTypeVideo:
			wh.parser.PushVideo(rtpPacket)
		}
	}

}

func (wh *WorkerHandler) OnDataChannel(d *webrtc.DataChannel) {
	log.Printf("New DataChannel %s %d\n", d.Label(), d.ID())

	d.OnOpen(func() {
		log.Printf("Data channel '%s'-'%d' open.\n", d.Label(), d.ID())
		for range time.NewTicker(RtcpPLIInterval * time.Millisecond).C {
			select {
			case <-wh.resultChan:
				message := <-wh.resultChan
				log.Printf("Sending '%s'\n", message)

				sendErr := d.SendText(message)
				if sendErr != nil {
					panic(sendErr)
				}

			default:
				continue
			}
		}
	})

	d.OnMessage(func(msg webrtc.DataChannelMessage) {
		fmt.Printf("Message from DataChannel '%s': '%s'\n", d.Label(), string(msg.Data))
	})
}
