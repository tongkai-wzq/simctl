package model

import (
	"errors"
	"fmt"
	"simctl/db"
	"simctl/wechat"
	"time"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
)

type Order struct {
	Id            int64  `json:"id"`
	OutTradeNo    string `json:"outTradeNo"`
	TransactionId string `json:"transactionId"`
	Title         string `json:"title"`
	AgentId       int64  `json:"agentId"`
	Agent         *Agent `xorm:"-" json:"agent"`
	SimId         int64
	Sim           *Sim `xorm:"-" json:"sim"`
	UserId        int64
	User          *User `xorm:"-" json:"user"`
	MealId        int64
	Meal          *Meal     `xorm:"-" json:"meal"`
	NextMonth     bool      `json:"nextMonth"`
	Price         float64   `json:"price"`
	Amount        float64   `json:"amount"`
	RefundAmt     float64   `json:"refundAmt"`
	Packets       []*Packet `xorm:"-"`
	Status        int8      `json:"status"`
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
	o.Packets = nil
	beginAt := o.Meal.RsvBeginAt(o.Sim.GetBaseExpired(), o.NextMonth)
	for _, packet := range o.Meal.AgtPackets(beginAt) {
		packet.SimId = o.SimId
		o.Packets = append(o.Packets, &packet)
	}
	return o.Packets
}

func (o *Order) SavePackets() {
	for _, packet := range o.Packets {
		packet.OrderId = o.Id
	}
	db.Engine.Insert(o.Packets)
}

func (o *Order) GetRbtPca() float64 {
	return o.Amount / o.Price
}

func (o *Order) GiveRbt() error {
	var rebates []*Rebates
	var agent, subAgent *Agent
	agent = o.Agent
	for {
		var agtMeal, subAgtMeal AgentMeal
		var amount float64
		if has, err := db.Engine.Where("agent_id = ? AND meal_id = ?", agent.Id, o.MealId).Get(&agtMeal); err != nil || !has {
			return errors.New("agentMeal no found")
		}
		if subAgent == nil {
			if agtMeal.Price > 0 {
				amount = (agtMeal.Price - agtMeal.StlPrice) * o.GetRbtPca()
			} else {
				amount = (o.Price - agtMeal.StlPrice) * o.GetRbtPca()
			}
		} else {
			if has, err := db.Engine.Where("agent_id = ? AND meal_id = ?", subAgent.Id, o.MealId).Get(&subAgtMeal); err != nil || !has {
				return errors.New("agentMeal no found")
			}
			amount = (subAgtMeal.StlPrice - agtMeal.StlPrice) * o.GetRbtPca()
		}
		if amount > 0 {
			rebate := Rebates{
				Amount: amount,
				Status: 0,
			}
			rebate.AgentId = o.AgentId
			rebate.Agent = o.Agent
			rebate.OrderId = o.Id
			rebate.Order = o
			rebates = append(rebates, &rebate)
		}
		if agent.SuperiorId > 0 {
			agent.LoadSuperior()
			subAgent = agent
			agent = agent.Superior
		} else {
			break
		}
	}
	var receivers []profitsharing.CreateOrderReceiver
	for _, rebate := range rebates {
		db.Engine.Insert(rebate)
		receivers = append(receivers, profitsharing.CreateOrderReceiver{
			Account:     core.String(rebate.Agent.Openid),
			Amount:      core.Int64(int64(rebate.Amount * 100)),
			Description: core.String(fmt.Sprintf("%v返佣", rebate.Order.OutTradeNo)),
			Type:        core.String("PERSONAL_OPENID"),
		})
	}
	go func() {
		time.Sleep(15 * time.Second)
		err := wechat.ProfitSharing(profitsharing.CreateOrderRequest{
			OutOrderNo:      core.String(fmt.Sprintf("%v-S", o.OutTradeNo)),
			Receivers:       receivers,
			TransactionId:   core.String(o.TransactionId),
			UnfreezeUnsplit: core.Bool(true),
		})
		if err == nil {
			for _, rebate := range rebates {
				rebate.Status = 1
				db.Engine.Cols("status").Update(rebate)
			}
		}
	}()
	return nil
}
