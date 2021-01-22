package parser

import (
	"github.com/at-wat/ebml-go/webm"
	"github.com/pion/rtp"
	"github.com/pion/webrtc/v3/pkg/media/samplebuilder"
	"io"
	"log"
	"os"
	"os/exec"
)

type Parser struct {
	Width  uint64
	Height uint64

	IsClosed       bool
	isSeenKeyframe bool
	videoTimestamp uint32
	audioTimestamp uint32
	fileTimestamp  uint32

	videoBuilder *samplebuilder.SampleBuilder
	audioBuilder *samplebuilder.SampleBuilder
	audioWriter   webm.BlockWriteCloser
	pipeWriter    webm.BlockWriteCloser
	fileWriter    webm.BlockWriteCloser
	inputPipe     io.WriteCloser
	outputPipe    io.Reader
	ffmpeg        *exec.Cmd
}

const (
	filename = "/tmp/test.webm"
)

func NewParser(w, h int) *Parser {
	in, out, ffmpeg := RunFFmpeg(w, h)
	p := Parser{
		Width:          uint64(w),
		Height:         uint64(h),
		IsClosed:       false,
		isSeenKeyframe: false,
		videoTimestamp: 0,
		videoBuilder:   nil,
		audioBuilder:   nil,
		pipeWriter:     nil,
		inputPipe:      in,
		outputPipe:     out,
		ffmpeg:         ffmpeg,
	}
	return &p
}

func (p *Parser) Close() {
	p.ClosePipe()
	p.CloseFile()
	p.CloseFFmpeg()
	p.IsClosed = true
}

func (p *Parser) InitPipeWriter() {
	w, err := webm.NewSimpleBlockWriter(p.inputPipe, []webm.TrackEntry{{
		Name:            "Video",
		TrackNumber:     2,
		TrackUID:        67890,
		CodecID:         "V_VP8",
		TrackType:       1,
		DefaultDuration: 33333333,
		Video: &webm.Video{
			PixelWidth:  p.Width,
			PixelHeight: p.Height,
		},
	}})
	if err != nil {
		log.Println("Parser init pipe writer err", err)
	}
	p.pipeWriter = w[0]
}

func (p *Parser) ClosePipe() {
	if p.pipeWriter != nil {
		if err := p.pipeWriter.Close(); err != nil {
			log.Println("Parser pipe close err", err)
		}
		p.pipeWriter = nil
	}
}

func (p *Parser) InitFileWriter() {
	f, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0666)
	if err != nil {
		log.Println("open file err", err)
	}
	w, err := webm.NewSimpleBlockWriter(f, []webm.TrackEntry{
		{
			Name:            "Video",
			TrackNumber:     2,
			TrackUID:        67890,
			CodecID:         "V_VP8",
			TrackType:       1,
			DefaultDuration: 33333333,
			Video: &webm.Video{
				PixelWidth:  p.Width,
				PixelHeight: p.Height,
			},
		},{
			Name:            "Audio",
			TrackNumber:     1,
			TrackUID:        12345,
			CodecID:         "A_OPUS",
			TrackType:       2,
			DefaultDuration: 20000000,
			Audio: &webm.Audio{
				SamplingFrequency: 48000.0,
				Channels:          2,
			},
	},
	})

	if err != nil {
		log.Println("Parser init file writer err", err)
	}
	p.fileWriter = w[0]
	p.audioWriter = w[1]
}

func (p *Parser) CloseFile() {
	if p.fileWriter != nil {
		if err := p.fileWriter.Close(); err != nil {
			log.Println("Parser file close err", err)
		}
		p.fileWriter = nil
	}

	if p.audioWriter != nil {
		if err := p.audioWriter.Close(); err != nil {
			log.Println("Parser audio close err")
		}
		p.audioWriter = nil
	}
}

func (p *Parser) CloseFFmpeg() {
	if p.ffmpeg != nil {
		if err := p.ffmpeg.Wait(); err != nil {
			log.Println("ffmpeg close err", err)
		}
	}
}

func (p *Parser) PushAudio(rtpPacket *rtp.Packet) {
	p.audioBuilder.Push(rtpPacket)

	for {
		sample := p.audioBuilder.Pop()
		if sample == nil {
			return
		}
		if p.audioWriter != nil {
			p.audioTimestamp += sample.Samples
			t := p.audioTimestamp / 48
			if _, err := p.audioWriter.Write(true, int64(t), sample.Data); err != nil {
				panic(err)
			}
		}
	}
}

func (p *Parser) PushVideo(rtpPacket *rtp.Packet) {
	p.videoBuilder.Push(rtpPacket)

	for {
		sample := p.videoBuilder.Pop()
		if sample == nil {
			return
		}
		isKeyframe := sample.Data[0]&0x1 == 0
		if isKeyframe && !p.isSeenKeyframe {
			raw := uint(sample.Data[6]) | uint(sample.Data[7]<<8) | uint(sample.Data[8]<<16) | uint(sample.Data[9])<<24
			p.Width = uint64(raw & 0x3FFF)
			p.Height = uint64((raw >> 16) & 0x3FFF)
			p.InitPipeWriter()
			p.InitFileWriter()
			p.isSeenKeyframe = true
		}

		if p.pipeWriter != nil {
			p.videoTimestamp += sample.Samples
			t := p.videoTimestamp / 90
			if _, err := p.pipeWriter.Write(isKeyframe, int64(t), sample.Data); err != nil {
				log.Println("pipe write err", err)
			}
		}

		if p.fileWriter != nil {
			p.fileTimestamp += sample.Samples
			t := p.fileTimestamp / 90
			if _, err := p.fileWriter.Write(isKeyframe, int64(t), sample.Data); err != nil {
				log.Println("file write err", err)
			}
		}
	}
}

func (p *Parser) PopImage(size int64) []byte {
	buf := make([]byte, size)
	if _, err := io.ReadFull(p.outputPipe, buf); err != nil {
		log.Println("pop image err", err)
	} else {
		log.Println("pop image success")
	}
	return buf
}
