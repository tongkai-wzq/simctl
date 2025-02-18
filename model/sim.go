package model

import (
	"errors"
	"simctl/db"
	"simctl/gateway"
	"time"

	"xorm.io/builder"
)

type Sim struct {
	Id       int64        `json:"id"`
	GwuserId int64        `json:"gwuserId"`
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

func (s *Sim) SetSyncAt() {
	now := time.Now()
	s.SyncAt = &now
}

func (s *Sim) LoadAgent() {
	s.Agent = new(Agent)
	db.Engine.ID(s.AgentId).Get(s.Agent)
}

func (s *Sim) LoadGroup() {
	s.Group = new(Group)
	db.Engine.ID(s.GroupId).Get(s.Group)
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

func (s *Sim) GetPacket() *Packet {
	packet := new(Packet)
	if has, err := db.Engine.Where("sim_id = ? AND invalid IS FALSE AND used/kb_cft < kb", s.Id).OrderBy("expired_at,id").Get(packet); err == nil && has {
		return packet
	}
	return nil
}

func (s *Sim) PreSaleMeals() ([]*SaleMeal, error) {
	s.Group.LoadMeals()
	saleMeals := s.Group.GetSaleMeals()
	saleMeals = s.onceRemove(saleMeals)
	if s.AgentId > 0 {
		var agentGroup AgentGroup
		if has, err := db.Engine.Where("agent_id = ? AND group_id = ?", s.AgentId, s.GroupId).Get(&agentGroup); err == nil && has {
			if agentGroup.Rebates {
				agentGroup.LoadAgentMeals()
				saleMeals = agentGroup.AttachPrice(saleMeals)
			}
		} else {
			return nil, errors.New("未代理此套餐")
		}
	}
	baseExpiredAt := s.GetBaseExpired()
	for _, saleMeal := range saleMeals {
		if !saleMeal.AcrossMonth && saleMeal.Base && (baseExpiredAt == nil || baseExpiredAt.Before(time.Now())) {
			saleMeal.AcMthAble = true
		} else if !saleMeal.AcrossMonth {
			saleMeal.AcMthAble = true
		}
	}
	return saleMeals, nil
}

func (s *Sim) onceRemove(saleMeals []*SaleMeal) []*SaleMeal {
	var newSaleMeals []*SaleMeal
	for _, saleMeal := range saleMeals {
		if saleMeal.Once {
			if exist, err := db.Engine.Exist(&Order{SimId: s.Id, MealId: saleMeal.MealId, Status: 1}); err == nil && exist {
				continue
			}
		}
		newSaleMeals = append(newSaleMeals, saleMeal)
	}
	return newSaleMeals
}

func (s *Sim) QryInit() ([]string, bool, *int64, *Packet) {
	gwUser := s.GetGwUser()
	var (
		qryFuns []string
		must    bool
		lastKb  *int64
		packet  *Packet
	)
	packet = s.GetPacket()
	if packet == nil || gwUser.Gateway.IsCycleNear(gwUser.Gateway) {
		must = false
		lastKb = nil
	} else if s.MonthAt == nil || !gwUser.Gateway.IsCurtCycle(gwUser.Gateway, *s.MonthAt) {
		must = true
		lastKb = nil
	} else if time.Since(*s.MonthAt) > 15*time.Minute {
		must = true
		lastKb = &s.MonthKb
	}
	switch Gateway := gwUser.Gateway.(type) {
	case *gateway.Unicom:
		if packet == nil && s.Status == 2 {
			go Gateway.ChgLfcy(s, 3)
		} else if packet != nil && s.Status == 3 {
			go Gateway.ChgLfcy(s, 2)
		}
		if s.Auth {
			if s.SyncAt == nil || time.Since(*s.SyncAt) > 24*time.Hour || must {
				qryFuns = append(qryFuns, "QryDtls")
			}
		} else {
			if s.SyncAt == nil || time.Since(*s.SyncAt) > 8*time.Hour {
				qryFuns = append(qryFuns, "QryDtls")
			}
		}
	case *gateway.Mobile:
		if packet == nil && s.Status == 2 && s.FlowOn == 1 {
			go Gateway.SwtFlowOn(s, 0)
		} else if packet != nil && s.Status == 2 && s.FlowOn == 0 {
			go Gateway.SwtFlowOn(s, 1)
		}
		if s.Auth {
			if s.SyncAt == nil || time.Since(*s.SyncAt) > 24*time.Hour {
				qryFuns = append(qryFuns, "QryAuthSts", "QrySts", "QryCmunt")
			}
			if must {
				qryFuns = append(qryFuns, "MtFlow")
			}
		} else {
			if s.SyncAt == nil || time.Since(*s.SyncAt) > 8*time.Hour {
				qryFuns = append(qryFuns, "QryAuthSts")
			}
		}
	case *gateway.Telecom:
		if packet == nil && s.Status == 4 && s.FlowOn == 1 {
			go Gateway.SwtFlowOn(s, 0)
		} else if packet != nil && s.Status == 4 && s.FlowOn == 0 {
			go Gateway.SwtFlowOn(s, 1)
		}
		if s.Auth {
			if s.SyncAt == nil || time.Since(*s.SyncAt) > 24*time.Hour {
				qryFuns = append(qryFuns, "QryStsMore")
			}
			if must {
				qryFuns = append(qryFuns, "MtFlows")
			}
		} else {
			if s.SyncAt == nil || time.Since(*s.SyncAt) > 8*time.Hour {
				qryFuns = append(qryFuns, "QryAuthStses")
			}
		}
	}
	return qryFuns, must, lastKb, packet
}
