package gateway

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"time"
)

var gwClient *http.Client

func init() {
	gwClient = &http.Client{
		Timeout: 15 * time.Second,
	}
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

type GateWayer interface {
	GetGwUserId() int64
	SetGwUserId(gwUserId int64)
	ChgLfcy(simer Simer, status int8) error
	IsCycleNear(gateway GateWayer) bool
	IsCurtCycle(gateway GateWayer, at time.Time) bool
}

type SwtFlowOner interface {
	SwtFlowOn(simer Simer, flowOn int8) error
}

type gateway struct {
	gwUserId int64
}

func (gw *gateway) GetGwUserId() int64 {
	return gw.gwUserId
}

func (gw *gateway) SetGwUserId(gwUserId int64) {
	gw.gwUserId = gwUserId
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

func (gw *gateway) send(req *http.Request) (io.ReadCloser, error) {
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Accept-Encoding", "gzip,deflate,sdch")
	if response, err := gwClient.Do(req); err == nil && response.StatusCode == http.StatusOK {
		defer response.Body.Close()
		var reader io.ReadCloser
		if response.Header.Get("Content-Encoding") == "gzip" {
			if reader, err = gzip.NewReader(response.Body); err != nil {
				return nil, err
			}
			defer reader.Close()
		} else {
			reader = response.Body
		}
		return reader, nil
	} else if err == nil {
		defer response.Body.Close()
		return nil, fmt.Errorf("req StatusCode %v", response.StatusCode)
	} else {
		return nil, err
	}
}
