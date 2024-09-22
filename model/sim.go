package model

import (
	"fmt"
	"simctl/db"
	"time"
)

type Sim struct {
	Id      int64
	AgentId int64  `json:"agentId"`
	Agent   *Agent `xorm:"-" json:"agent"`
	GroupId int64
	Group   *Group `xorm:"-" json:"group"`
	Iccid   string `json:"iccid"`
	Msisdn  string `json:"msisdn"`
	MapNber string `json:"mapNber"`
}

func (s *Sim) LoadAgent() {
	s.Agent = new(Agent)
	db.Engine.ID(s.AgentId).Get(s.Agent)
}

func (s *Sim) GetBaseExpired() *time.Time {
	var (
		expiredAt time.Time
		packet    Packet
	)
	sql := "select * from packet where sim_id = ? AND base IS TRUE AND invalid IS FALSE order by expired_at desc"
	if has, err := db.Engine.SQL(sql, s.Id).Get(&packet); err == nil && has {
		expiredAt = packet.ExpiredAt
		return &expiredAt
	}
	return nil
}

type SaleMeal struct {
	MealId      int64   `json:"mealId"`
	Title       string  `json:"title"`
	Base        bool    `json:"base"`
	AcrossMonth bool    `json:"acrossMonth"`
	Price       float64 `json:"price"`
	Once        bool    `json:"once"`
	AcMthAble   bool    `json:"acMthAble"`
}

func (s *Sim) PreSaleMeals() []*SaleMeal {
	var oneIds []int64
	db.Engine.Table("meal").Where("group_id = ? AND once IS TRUE", s.GroupId).Cols("id").Find(&oneIds)
	var mIds []int64
	db.Engine.Table("order").Where("sim_id = ? AND meal_id in ? AND status = 1", s.Id, oneIds).Cols("meal_id").Find(&mIds)
	saleMeals := make([]*SaleMeal, 0, 15)
	if s.AgentId > 0 {
		sql := "select m.id as meal_id,m.title,m.base,m.across_month,if(am.price>0,am.price,m.price) as price,m.once from meal as m left join agent_meal as am on m.id=am.meal_id where m.group_id = ? and am.agent_id = ? and m.id not in ?"
		if err := db.Engine.SQL(sql, s.GroupId, s.AgentId, mIds).Find(&saleMeals); err != nil {
			fmt.Println(err.Error())
		}
	} else {
		sql := "select id as meal_id,title,base,across_month,price,once from meal where group_id = ? AND id not in ?"
		db.Engine.SQL(sql, s.GroupId, mIds).Find(&saleMeals)
	}
	baseExpiredAt := s.GetBaseExpired()
	for _, saleMeal := range saleMeals {
		if saleMeal.AcrossMonth && saleMeal.Base && (baseExpiredAt == nil || baseExpiredAt.Before(time.Now())) {
			saleMeal.AcMthAble = true
		} else if saleMeal.AcrossMonth {
			saleMeal.AcMthAble = true
		}
	}
	return saleMeals
}
