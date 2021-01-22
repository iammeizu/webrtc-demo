package signalserver

import (
	"context"
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"net/url"
)

const (
	SignalStatusStart = iota
	SignalStatusSdp
	SignalStatusCandidate

	SignalTimeout = 30
)

var (
	upgrader = &websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
)

type SignalHandler struct {
	status int
	worker string
	isUpClosed bool
	isDownClosed bool
	upCon *websocket.Conn
	downCon *websocket.Conn
}

type Message struct {
	key string		`json:"key"`
	value string	`json:"value"`
}

func NewSignalHandler () SignalHandler {
	sh := SignalHandler{
		status: SignalStatusStart,
		isUpClosed: false,
		isDownClosed: true,
		upCon:    nil,
		downCon:  nil,
	}
	return sh
}


func Serve (ctx *gin.Context) {
	sh := NewSignalHandler()
	sh.worker = ctx.Keys["worker"].(string)
	c, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		log.Println("Upgrade err: ", err)
	}
	sh.upCon = c
	ctx1, cancel := context.WithTimeout(ctx, SignalTimeout)

	defer func() {
		err = sh.upCon.Close()
		if err != nil {
			log.Println("Up connection close err: ", err)
		}
		sh.isUpClosed = true
		cancel()
	}()

	go sh.Listen(ctx1)
	go sh.Run(ctx1)

	select {
	case <- ctx1.Done():
		log.Println("Request close ", ctx.Request.Header["request_id"])
		return
	}

}

func (sh *SignalHandler) CheckStatus(from, to int) bool {
	StatusConvertMap := map[int]int{
		SignalStatusStart: SignalStatusSdp,
		SignalStatusSdp: SignalStatusCandidate,
		SignalStatusCandidate: SignalStatusCandidate,
	}

	if v, ok := StatusConvertMap[from]; ok && v == to {
		return true
	}
	return false
}

func (sh *SignalHandler) Run(ctx context.Context) {
	msg := Message{}
	for {
		_, message, _ := sh.upCon.ReadMessage()
		err := json.Unmarshal(message, &msg)
		if err != nil {
			log.Println("Invalid Message err: ", err)
			continue
		}

		switch msg.key {
		case "sdp":
			if sh.CheckStatus(SignalStatusStart, SignalStatusSdp){
				sh.status = SignalStatusSdp
				sh.Send(message)
			} else {
				log.Println("Invalid Process, current status: ", sh.status)
			}
		case "candidate":
			if sh.CheckStatus(SignalStatusSdp, SignalStatusCandidate) || sh.CheckStatus(SignalStatusCandidate, SignalStatusCandidate) {
				sh.status = SignalStatusCandidate
				sh.Send(message)
			} else {
				log.Println("Invalid Process, current status: ", sh.status)
			}
		}

		select {
		case <- ctx.Done():
			log.Println("Signal handler closed")
			return
		default:
			continue
		}
	}
}

func (sh *SignalHandler) Listen(ctx context.Context) {
	u := url.URL{Scheme:"ws", Host: sh.worker, Path:"/signal"}
	log.Printf("connecting to %s", u.String())
	var err error

	sh.downCon, _, err = websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("Dail err: ", err)
	}
	sh.isDownClosed = false

	defer func() {
		err := sh.downCon.Close()
		if err != nil {
			log.Println("Down connection close err: ", err)
		}
		sh.isDownClosed = true
	}()

	go func(ctx context.Context) {
		for {
			_, message, err := sh.downCon.ReadMessage()
			if err != nil {
				log.Println("read from worker err: ", err)
				return
			}
			err = sh.upCon.WriteMessage(websocket.TextMessage, message)
			log.Println("write ", string(message))
			if err != nil {
				break
			}
			select {
			case <- ctx.Done():
				return
			default:
				continue
			}
		}
	}(ctx)

	select {
	case <-ctx.Done():
		return
	}
}

func (sh *SignalHandler) Send (payload []byte) {
	if !sh.isDownClosed {
		err := sh.downCon.WriteMessage(websocket.TextMessage, payload)
		if err != nil {
			log.Println("Websocket write msg err: ", err)
		}
	} else {
		log.Println("Down connection is closed!")
	}
}