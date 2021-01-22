package main

import (
	"github.com/gorilla/websocket"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
)

type Message struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type WebrtcHandler struct {
	pc         *webrtc.PeerConnection
	rtpChan    chan *rtp.Packet
	dataChan   chan string
	candidates chan *webrtc.ICECandidate

	isClosed bool
	wsCon    *websocket.Conn
	worker   Worker
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

	wh.worker = NewWorkerHandler(wh.pc)
	return wh
}
