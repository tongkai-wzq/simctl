package model

import (
	"fmt"
	"simctl/db"
	"time"

	"xorm.io/builder"
)

type Sim struct {
	Id      int64  `json:"id"`
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
	db.Engine.Table("order").Where(builder.Eq{"sim_id": s.Id, "status": 1}.And(builder.In("meal_id", oneIds))).Cols("meal_id").Find(&mIds)
	saleMeals := make([]*SaleMeal, 0, 15)
	if s.AgentId > 0 {
		sql := builder.Select("m.id as meal_id,m.title,m.base,m.across_month,if(am.price>0,am.price,m.price) as price,m.once")
		sql.From("meal as m").LeftJoin("agent_meal as am", "m.id=am.meal_id").Where(builder.Eq{"m.group_id": s.GroupId, "am.agent_id": s.AgentId}.And(builder.NotIn("m.id", mIds)))
		if err := db.Engine.SQL(sql).Find(&saleMeals); err != nil {
			fmt.Println(err.Error())
		}
	} else {
		sql := builder.Select("id as meal_id,title,base,across_month,price,once").From("meal").Where(builder.Eq{"group_id": s.GroupId}.And(builder.NotIn("id", mIds)))
		db.Engine.SQL(sql).Find(&saleMeals)
	}
	baseExpiredAt := s.GetBaseExpired()
	for _, saleMeal := range saleMeals {
		if !saleMeal.AcrossMonth && saleMeal.Base && (baseExpiredAt == nil || baseExpiredAt.Before(time.Now())) {
			saleMeal.AcMthAble = true
		} else if !saleMeal.AcrossMonth {
			saleMeal.AcMthAble = true
		}
	}
	return saleMeals
}
