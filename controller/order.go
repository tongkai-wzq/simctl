package controller

import (
	"encoding/json"
	"net/http"
	"simctl/db"
	"simctl/model"
	"time"

	"github.com/gorilla/websocket"
)

func NewBuy(w http.ResponseWriter, r *http.Request) {
	if conn, err := upgrader.Upgrade(w, r, nil); err == nil {
		var buy Buy
		buy.Conn = conn
		buy.Run(buy.GetHandleMap())
	} else {
		println(err.Error())
	}
}

type buyInitMsg struct {
	message
	SimId int64 `json:"simId"`
}

type buyInitResp struct {
	message
	Iccid     string            `json:"iccid"`
	Msisdn    string            `json:"msisdn"`
	MapNber   string            `json:"mapNber"`
	SaleMeals []*model.SaleMeal `json:"saleMeals"`
}

type buySubmitMsg struct {
	message
	MealKey   int64 `json:"mealKey"`
	NextMonth bool  `json:"nextMonth"`
}

type buyPacket struct {
	Base      bool      `json:"base"`
	StartAt   time.Time `json:"startAt"`
	ExpiredAt time.Time `json:"expiredAt"`
	Kb        int64     `json:"Kb"`
}

type buySubmitResp struct {
	message
	Packets []buyPacket `json:"packets"`
}

type Buy struct {
	widget
	saleMeals []*model.SaleMeal
	order     *model.Order
	packets   []*model.Packet
}

func (b *Buy) GetHandleMap() map[string]func(bMsg []byte) {
	return map[string]func(bMsg []byte){
		"init":   b.OnInit,
		"submit": b.OnSubmit,
	}
}

func (b *Buy) OnInit(bMsg []byte) {
	var iMsg buyInitMsg
	json.Unmarshal(bMsg, &iMsg)
	sim := new(model.Sim)
	db.Engine.ID(iMsg.SimId).Get(sim)
	b.order = new(model.Order)
	b.order.SimId = sim.Id
	b.order.Sim = sim
	b.order.AgentId = sim.AgentId
	b.order.LoadAgent()
	var iResp buyInitResp
	iResp.Iccid = sim.Iccid
	iResp.Msisdn = sim.Msisdn
	iResp.MapNber = sim.MapNber
	iResp.Handle = "init"
	b.saleMeals = sim.PreSaleMeals()
	iResp.SaleMeals = b.saleMeals
	if data, err := json.Marshal(&iResp); err == nil {
		b.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (b *Buy) OnSubmit(bMsg []byte) {
	var sMsg buySubmitMsg
	json.Unmarshal(bMsg, &sMsg)
	b.order.MealId = b.saleMeals[sMsg.MealKey].MealId
	b.order.LoadMeal()
	b.order.NextMonth = sMsg.NextMonth
	b.order.Price = b.saleMeals[sMsg.MealKey].Price
	b.packets = b.order.PrePackets()
	var sResp buySubmitResp
	sResp.Handle = "submit"
	for _, packet := range b.packets {
		sResp.Packets = append(sResp.Packets, buyPacket{
			Base:      packet.Base,
			StartAt:   packet.StartAt,
			ExpiredAt: packet.ExpiredAt,
			Kb:        packet.Kb,
		})
	}
	if data, err := json.Marshal(&sResp); err == nil {
		b.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (b *Buy) OnUnify(bMsg []byte) {

}

func (b *Buy) Pay() {
	b.order.Status = 1
	db.Engine.Insert(b.order)
	b.order.SavePackets(b.packets)
}
