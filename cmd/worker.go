package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/iammeizu/webrtc-demo/middleware"
	"github.com/iammeizu/webrtc-demo/worker"
	"log"
)

var (
	port = flag.String("port", ":9001", "")
)

func main() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.RequestIDMiddleWare)

	r.GET("/signal", worker.Serve)

	err := r.Run(*port)
	if err != nil {
		log.Println("Worker boot failed")
	} else {
		log.Println("Worker running in port: ", *port)
	}
}
