package controller

import (
	"encoding/json"
	"fmt"
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
		HandshakeTimeout: time.Second * 10,
		ReadBufferSize:   1024,
		WriteBufferSize:  1024,
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}
}

func AuthUser(w http.ResponseWriter, r *http.Request) *model.User {
	_, claims, _ := jwtauth.FromContext(r.Context())
	var user model.User
	if has, err := db.Engine.ID(int64(claims["userId"].(float64))).Get(&user); err != nil || !has {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return nil
	}
	return &user
}

type message struct {
	Code   int64  `json:"code"`
	Msg    string `json:"msg"`
	Handle string `json:"handle"`
}

type Widgeter interface {
	GetHandleMap() map[string]func(bMsg []byte)
	Close()
}

type widget struct {
	Conn *websocket.Conn
}

func (w *widget) Run(cc Widgeter) {
	go w.keep(cc)
	for {
		msgType, bMsg, err := w.Conn.ReadMessage()
		if err != nil {
			fmt.Println("err msg", err, msgType)
			break
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

func (w *widget) keep(cc Widgeter) {
	w.Conn.SetReadDeadline(time.Now().Add(9 * time.Second))
	w.Conn.SetPongHandler(func(appData string) error {
		w.Conn.SetReadDeadline(time.Now().Add(9 * time.Second))
		return nil
	})
	w.Conn.SetCloseHandler(func(code int, text string) error {
		cc.Close()
		w.Conn.Close()
		fmt.Println("close handle", code, text)
		return nil
	})
	for {
		time.Sleep(3 * time.Second)
		if err := w.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
			break
		}
	}
}
