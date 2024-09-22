package controller

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

func init() {
	upgrader = websocket.Upgrader{
		HandshakeTimeout: time.Second * 10,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

type message struct {
	Handle string `json:"handle"`
}

type widget struct {
	Conn *websocket.Conn
}

func (w *widget) Run(handleMap map[string]func(bMsg []byte)) {
	for {
		msgType, bMsg, err := w.Conn.ReadMessage()
		if err != nil {
			fmt.Println("err msg", err, msgType)
			return
		}
		var msg message
		json.Unmarshal(bMsg, &msg)
		for key, handle := range handleMap {
			if key == msg.Handle {
				handle(bMsg)
				break
			}
		}
	}
}
