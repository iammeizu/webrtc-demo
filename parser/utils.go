package parser

import (
	"io"
	"log"
	"os/exec"
	"strconv"
)

func RunFFmpeg(width, height int) (io.WriteCloser, io.Reader, *exec.Cmd) {
	/*
	 start ffmpeg process to handle video stream

	 -i : input
	 -r : output fps
	 -s : output frame resolution
	 -f : output format
	 */

	ffmpeg := exec.Command("ffmpeg", "-i", "pipe:0", "-r", "10", "-pix_fmt", "bgr24", "-s", strconv.Itoa(width)+"x"+strconv.Itoa(height), "-f", "rawvideo", "pipe:1")

	in, _ := ffmpeg.StdinPipe()
	out, _ := ffmpeg.StdoutPipe()

	if err := ffmpeg.Start(); err != nil {
		log.Println("FFmpeg start err", err)
	} else {
		log.Println("FFmpeg process started")
	}
	return in, out, ffmpeg
}
