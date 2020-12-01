package main

import (
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3"
	"github.com/gorilla/websocket"
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
	worker Worker
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