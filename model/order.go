package model

import (
	"errors"
	"simctl/db"
	"time"
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
	Meal          *Meal      `xorm:"-" json:"meal"`
	NextMonth     bool       `json:"nextMonth"`
	Price         float64    `json:"price"`
	Amount        float64    `json:"amount"`
	RefundAmt     float64    `json:"refundAmt"`
	Packets       []*Packet  `xorm:"-" json:"-"`
	Rebates       []*Rebates `xorm:"-" json:"-"`
	Status        int8       `json:"status"`
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
	beginAt := o.Meal.GetBeginAt(o.Sim.GetBaseExpired(), o.NextMonth)
	for _, packet := range o.Meal.AlignPackets(beginAt) {
		packet.SimId = o.SimId
		o.Packets = append(o.Packets, packet)
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
	var agent, subAgent *Agent
	agent = o.Agent
	for {
		var agtMeal, subAgtMeal AgentMeal
		var amount float64
		var agentGroup AgentGroup
		if has, err := db.Engine.Where("agent_id = ? AND group_id = ?", agent.Id, o.Sim.GroupId).Get(&agentGroup); err == nil && !has {
			return errors.New("未分配此套餐组")
		} else if err != nil {
			return err
		} else if agentGroup.Rebates {
			if has, err := db.Engine.Where("agent_id = ? AND meal_id = ?", agent.Id, o.MealId).Get(&agtMeal); err == nil && !has {
				return errors.New("agentMeal no found")
			} else if err != nil {
				return err
			}
			if subAgent == nil {
				if agtMeal.Price > 0 {
					amount = (agtMeal.Price - agtMeal.StlPrice) * o.GetRbtPca()
				} else {
					amount = (o.Price - agtMeal.StlPrice) * o.GetRbtPca()
				}
			} else {
				if has, err := db.Engine.Where("agent_id = ? AND meal_id = ?", subAgent.Id, o.MealId).Get(&subAgtMeal); err == nil && !has {
					return errors.New("agentMeal no found")
				} else if err != nil {
					return err
				}
				amount = (subAgtMeal.StlPrice - agtMeal.StlPrice) * o.GetRbtPca()
			}
			o.Rebates = append(o.Rebates, &Rebates{
				AgentId: agent.Id,
				Agent:   agent,
				OrderId: o.Id,
				Order:   o,
				Amount:  amount,
				Status:  0,
			})
		}
		if agent.SuperiorId > 0 {
			agent.LoadSuperior()
			subAgent = agent
			agent = agent.Superior
		} else {
			break
		}
	}
	if len(o.Rebates) > 0 {
		for _, rebate := range o.Rebates {
			db.Engine.Insert(rebate)
		}
		go o.RbtToAccount()
	}
	return nil
}

func (o *Order) RbtToAccount() {
	time.Sleep(15 * time.Second)
	for _, rebate := range o.Rebates {
		rebate.ToAccount()
	}
}
