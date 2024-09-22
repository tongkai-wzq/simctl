package model

import (
	"simctl/db"
)

type Order struct {
	Id        int64
	Title     string `json:"title"`
	AgentId   int64  `json:"agentId"`
	Agent     *Agent `xorm:"-" json:"agent"`
	SimId     int64
	Sim       *Sim `xorm:"-" json:"sim"`
	MealId    int64
	Meal      *Meal   `xorm:"-" json:"meal"`
	NextMonth bool    `json:"nextMonth"`
	Amount    float64 `json:"amount"`
	RefundAmt float64 `json:"refundAmt"`
	Status    int64   `json:"status"`
}

func (o *Order) LoadAgent() {
	o.Agent = new(Agent)
	db.Engine.ID(o.AgentId).Get(o.Agent)
}

func (o *Order) LoadMeal() {
	o.Meal = new(Meal)
	db.Engine.ID(o.MealId).Get(o.Meal)
}

func (o *Order) PrePackets() []*Packet {
	beginAt := o.Meal.RsvBeginAt(o.Sim.GetBaseExpired(), o.NextMonth)
	packets := o.Meal.AgtPackets(beginAt)
	for _, packet := range packets {
		packet.SimId = o.SimId
	}
	return packets
}

func (o *Order) SavePackets(packets []*Packet) {
	for _, packet := range packets {
		packet.OrderId = o.Id
	}
	db.Engine.Insert(&packets)
}
