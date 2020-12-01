package main

import (
	"context"
	"github.com/pion/webrtc/v3"
)

type Worker interface {
	// Used to do task
	Run(ctx context.Context)

	// Used to handle media data
	OnTrack(track *webrtc.Track, receiver *webrtc.RTPReceiver)

	// Used to handle non-media data
	OnDataChannel(d *webrtc.DataChannel)
}
