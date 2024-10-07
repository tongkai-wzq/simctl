package gateway

import (
	"time"

	"github.com/imroc/req/v3"
)

var gwClient *req.Client

func init() {
	gwClient = req.C().SetTimeout(15 * time.Second)
	gwClient.SetCommonHeader("Content-Type", "application/json")
}

type Simer interface {
	GetIccid() string
	GetMsisdn() string
	GetFlowOn() int8
	SetFlowOn(flowOn int8)
	GetStatus() int8
	SetStatus(status int8)
	GetAuth() bool
	SetAuth(auth bool)
	GetMonthKb() int64
	SetMonthKb(monthKb int64)
	GetMonthAt() *time.Time
}

type GwUserer interface {
	GetId() int64
	GetName() string
}

type GateWayer interface {
	GetGwUser() GwUserer
	SetGwUser(gwUser GwUserer)
	ChgLfcy(simer Simer, status int8) error
	IsCycleNear(gateway GateWayer) bool
	IsCurtCycle(gateway GateWayer, at time.Time) bool
}

type SwtFlowOner interface {
	SwtFlowOn(simer Simer, flowOn int8) error
}

type gateway struct {
	gwUser GwUserer
}

func (gw *gateway) GetGwUser() GwUserer {
	return gw.gwUser
}

func (gw *gateway) SetGwUser(gwUser GwUserer) {
	gw.gwUser = gwUser
}

func (gw *gateway) IsCycleNear(gateway GateWayer) bool {
	now := time.Now()
	switch gateway.(type) {
	case *Unicom:
		if now.Day() == 27 && now.Hour() == 0 && now.Minute() < 15 {
			return true
		} else if now.Day() == 26 && now.Hour() == 23 && now.Minute() > 50 {
			return true
		}
	default:
		next := now.AddDate(0, 0, 1)
		if now.Day() == 1 && now.Hour() == 0 && now.Minute() < 15 {
			return true
		} else if next.Day() == 1 && now.Hour() == 23 && now.Minute() > 50 {
			return true
		}
	}
	return false
}

func (gw *gateway) IsCurtCycle(gateway GateWayer, at time.Time) bool {
	now := time.Now()
	switch gateway.(type) {
	case *Unicom:
		if now.Day() < 27 && at.Day() < 27 && now.Month() == at.Month() {
			return true
		} else if now.Day() >= 27 && at.Day() >= 27 && now.Month() == at.Month() {
			return true
		} else if now.Day() < 27 && at.Day() >= 27 && at.AddDate(0, 1, 0).Month() == now.Month() {
			return true
		}
	default:
		if now.Month() == at.Month() {
			return true
		}
	}
	return false
}
