package main

import (
	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/rtp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
	"io"
	//"os"
)

type Parser struct {
	IsClosed	   bool
	isSeenKeyframe bool
	videoTimestamp uint32
	sampleBuilder  *samplebuilder.SampleBuilder
	videoWriter    webm.BlockWriteCloser
	inputPipe      io.WriteCloser
	outputPipe     io.Reader
}

