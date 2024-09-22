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
	AcrossMonth bool    `json:"acrossMonth"`
	Price       float64 `json:"price"`
}

func (s *Sim) PreSaleMeals() []*SaleMeal {
	saleMeals := make([]*SaleMeal, 0, 15)
	if s.AgentId > 0 {
		sql := "select m.id as meal_id,m.title,m.across_month,if(am.price>0,am.price,m.price) as price from meal as m left join agent_meal as am on m.id=am.meal_id where m.group_id = ? and am.agent_id = ?"
		if err := db.Engine.SQL(sql, s.GroupId, s.AgentId).Find(&saleMeals); err != nil {
			fmt.Println(err.Error())
		}
	} else {
		sql := "select id as meal_id,title,across_month,price from meal where group_id = ?"
		db.Engine.SQL(sql, s.GroupId).Find(&saleMeals)
	}
	return saleMeals
}
