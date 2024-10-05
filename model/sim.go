package model

import (
	"log"
	"simctl/db"
	"time"

	"xorm.io/builder"
)

type Sim struct {
	Id       int64        `json:"id"`
	GwuserId int64        ` json:"gwuserId"`
	GwUser   *GatewayUser `xorm:"-" json:"gwUser"`
	AgentId  int64        `json:"agentId"`
	Agent    *Agent       `xorm:"-" json:"agent"`
	GroupId  int64
	Group    *Group     `xorm:"-" json:"group"`
	Iccid    string     `json:"iccid"`
	Msisdn   string     `json:"msisdn"`
	MapNber  string     `json:"mapNber"`
	Auth     bool       `json:"auth"`
	FlowOn   int8       `json:"flowOn"`
	Status   int8       `json:"status"`
	SyncAt   *time.Time `json:"syncAt"`
	MonthKb  int64      `json:"monthKb"`
	MonthAt  *time.Time `json:"monthAt"`
	Packet   *Packet    `xorm:"-" json:"-"`
}

func (s *Sim) GetIccid() string {
	return s.Iccid
}

func (s *Sim) GetMsisdn() string {
	return s.Msisdn
}

func (s *Sim) GetStatus() int8 {
	return s.Status
}

func (s *Sim) SetStatus(status int8) {
	s.Status = status
}

func (s *Sim) GetAuth() bool {
	return s.Auth
}

func (s *Sim) SetAuth(auth bool) {
	s.Auth = auth
}

func (s *Sim) GetFlowOn() int8 {
	return s.FlowOn
}

func (s *Sim) SetFlowOn(flonOn int8) {
	s.FlowOn = flonOn
}

func (s *Sim) GetMonthKb() int64 {
	return s.MonthKb
}

func (s *Sim) SetMonthKb(monthKb int64) {
	s.MonthKb = monthKb
	now := time.Now()
	s.MonthAt = &now
}

func (s *Sim) GetMonthAt() *time.Time {
	return s.MonthAt
}

func (s *Sim) LoadAgent() {
	s.Agent = new(Agent)
	db.Engine.ID(s.AgentId).Get(s.Agent)
}

func (s *Sim) GetGwUser() *GatewayUser {
	return GatewayUsers[s.GwuserId]
}

func (s *Sim) GetBaseExpired() *time.Time {
	var (
		expiredAt time.Time
		packet    Packet
	)
	cond := builder.Eq{"sim_id": s.Id, "base": true, "invalid": false}
	if has, err := db.Engine.Where(cond).OrderBy("expired_at desc").Limit(1).Get(&packet); err == nil && has {
		expiredAt = packet.ExpiredAt
		return &expiredAt
	}
	return nil
}

func (s *Sim) LoadPacket() {
	packet := new(Packet)
	if has, err := db.Engine.Where("sim_id = ? AND invalid IS FALSE AND used/kb_cft < kb", s.Id).OrderBy("expired_at,id").Get(s.Packet); err == nil && has {
		s.Packet = packet
	}
}

func (s *Sim) MustSync() (bool, *int64) {
	gwUser := s.GetGwUser()
	if gwUser.Gateway.IsCycleNear(gwUser.Gateway) {
		return false, nil
	}
	if s.Packet == nil {
		return false, nil
	}
	if s.MonthAt == nil || !gwUser.Gateway.IsCurtCycle(gwUser.Gateway, *s.MonthAt) {
		return true, nil
	} else if time.Since(*s.MonthAt) > 15*time.Minute {
		return true, &s.MonthKb
	}
	return false, nil
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
	cond1 := builder.Eq{"group_id": s.GroupId, "once": true}
	db.Engine.Table("meal").Where(cond1).Cols("id").Find(&oneIds)
	var mIds []int64
	cond2 := builder.Eq{"sim_id": s.Id, "status": 1}.And(builder.In("meal_id", oneIds))
	db.Engine.Table("order").Where(cond2).Cols("meal_id").Find(&mIds)
	saleMeals := make([]*SaleMeal, 0, 15)
	if s.AgentId > 0 {
		sql := builder.Select("m.id as meal_id,m.title,m.base,m.across_month,if(am.price>0,am.price,m.price) as price,m.once").From("meal as m")
		sql.InnerJoin("agent_group as ag", "m.group_id=ag.group_id").InnerJoin("agent_meal as am", "m.id=am.meal_id")
		sql.Where(builder.Eq{"ag.group_id": s.GroupId, "ag.rebates": true, "am.agent_id": s.AgentId}.And(builder.NotIn("m.id", mIds)))
		if err := db.Engine.SQL(sql).Find(&saleMeals); err != nil {
			log.Println(err.Error())
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
