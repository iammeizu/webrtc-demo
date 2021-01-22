package main

import (
	"flag"
	"github.com/gin-gonic/gin"
	"log"
)

var (
	port = flag.String("port", ":9001", "")
)

func main() {
	r := gin.New()
	r.Use(gin.Logger())
	r.Use(gin.Recovery())
	r.Use(RequestIDMiddleWare)

	r.GET("/signal", Serve)

	err := r.Run(*port)
	if err != nil {
		log.Println("Worker boot failed")
	} else {
		log.Println("Worker running in port: ", *port)
	}
}
