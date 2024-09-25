package model

import (
	"context"
	"errors"
	"log"
	"simctl/db"
	"simctl/wechat"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
)

type Order struct {
	Id         int64  `json:"id"`
	OutTradeNo string `json:"outTradeNo"`
	Title      string `json:"title"`
	AgentId    int64  `json:"agentId"`
	Agent      *Agent `xorm:"-" json:"agent"`
	SimId      int64
	Sim        *Sim `xorm:"-" json:"sim"`
	UserId     int64
	User       *User `xorm:"-" json:"user"`
	MealId     int64
	Meal       *Meal   `xorm:"-" json:"meal"`
	NextMonth  bool    `json:"nextMonth"`
	Price      float64 `json:"price"`
	Amount     float64 `json:"amount"`
	RefundAmt  float64 `json:"refundAmt"`
	Status     int64   `json:"status"`
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
	aPackets := o.Meal.AgtPackets(beginAt)
	var packets []*Packet
	for _, packet := range aPackets {
		packet.SimId = o.SimId
		packets = append(packets, &packet)
	}
	return packets
}

func (o *Order) SavePackets(packets []*Packet) {
	for _, packet := range packets {
		packet.OrderId = o.Id
	}
	db.Engine.Insert(&packets)
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
		if amount == 0 {
			continue
		}
		rebate := Rebates{
			Amount: amount,
			Status: 0,
		}
		rebate.AgentId = o.AgentId
		rebate.Agent = o.Agent
		rebate.OrderId = o.Id
		rebate.Order = o
		rebates = append(rebates, &rebate)
		if agent.SuperiorId > 0 {
			agent.LoadSuperior()
			subAgent = agent
			agent = agent.Superior
		} else {
			break
		}
	}
	for _, rebate := range rebates {
		db.Engine.Insert(rebate)
	}
	go func() {
		svc := profitsharing.OrdersApiService{Client: wechat.PayClient}

		var receivers []profitsharing.CreateOrderReceiver
		receivers = append(receivers, profitsharing.CreateOrderReceiver{
			Account:     core.String("86693852"),
			Amount:      core.Int64(888),
			Description: core.String("分给商户A"),
			Name:        core.String("hu89ohu89ohu89o"),
			Type:        core.String("MERCHANT_ID"),
		})

		resp, result, err := svc.CreateOrder(context.Background(),
			profitsharing.CreateOrderRequest{
				Appid:           core.String("wx8888888888888888"),
				OutOrderNo:      core.String("P20150806125346"),
				Receivers:       receivers,
				SubAppid:        core.String("wx8888888888888889"),
				SubMchid:        core.String("1900000109"),
				TransactionId:   core.String("4208450740201411110007820472"),
				UnfreezeUnsplit: core.Bool(true),
			},
		)

		if err != nil {
			// 处理错误
			log.Printf("call CreateOrder err:%s", err)
		} else {
			// 处理返回结果
			log.Printf("status=%d resp=%s", result.Response.StatusCode, resp)
		}
	}()
	return nil
}
