package main

import (
	"github.com/pion/webrtc/v3"
)

type workerHandler struct {
	pc *webrtc.PeerConnection
	parser
}