package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"github.com/iammeizu/webrtc-demo/middleware"
	"github.com/iammeizu/webrtc-demo/signalserver"
	"log"
)

var (
	p = flag.String("port", ":9000", "")
	workerAddr = flag.String("worker address", "127.0.0.1:9001", "")
)


func main()  {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(middleware.RequestIDMiddleWare)
	r.Use(middleware.AuthMiddleWare)

	r.GET("/signal", func(ctx *gin.Context) {
		ctx.Keys["worker"] = *workerAddr
		signalserver.Serve(ctx)
	})

	err := r.Run(*port)
	if err != nil {
		log.Println("Signal Server boot failed")
	} else {
		log.Println("Signal Server run in port: ", *p)
	}
}
