package controller

import (
	"encoding/json"
	"log"
	"net/http"
	"simctl/db"
	"simctl/model"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/gorilla/websocket"
)

var upgrader websocket.Upgrader

func init() {
	upgrader = websocket.Upgrader{
		HandshakeTimeout: time.Second * 15,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func AuthUser(r *http.Request) *model.User {
	_, claims, _ := jwtauth.FromContext(r.Context())
	var user model.User
	if exist, err := db.Engine.ID(int64(claims["userId"].(float64))).Get(&user); err == nil || exist {
		return &user
	}
	return nil
}

type message struct {
	Code   int64  `json:"code"`
	Msg    string `json:"msg"`
	Handle string `json:"handle"`
}

type Widgeter interface {
	GetHandleMap() map[string]func(bMsg []byte)
	End()
}

type widget struct {
	conn  *websocket.Conn
	timer *time.Timer
}

func (w *widget) SetConn(conn *websocket.Conn) {
	if w.conn != nil {
		w.conn.Close()
	}
	w.conn = conn
}

func (w *widget) Read(cc Widgeter) {
	if w.timer == nil {
		w.timer = time.AfterFunc(900*time.Second, func() {
			cc.End()
		})
	}
	conn := w.conn
	for {
		msgType, bMsg, err := conn.ReadMessage()
		if err != nil {
			log.Println("ReadMessage", err.Error(), msgType)
			break
		} else {
			w.timer.Reset(900 * time.Second)
		}
		var msg message
		json.Unmarshal(bMsg, &msg)
		for key, handle := range cc.GetHandleMap() {
			if key == msg.Handle {
				handle(bMsg)
				break
			}
		}
	}
}
