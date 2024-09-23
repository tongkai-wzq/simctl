package model

import "time"

type Packet struct {
	Id        int64 `json:"id"`
	OrderId   int64
	Order     *Order `xorm:"-" json:"order"`
	SimId     int64
	Sim       *Sim      `xorm:"-" json:"sim"`
	Base      bool      `json:"base"`
	StartAt   time.Time `json:"startAt"`
	ExpiredAt time.Time `json:"expiredAt"`
	Kb        int64     `json:"Kb"`
	KbCft     float64   `json:"kbCft"`
	Used      int64     `json:"used"`
	Invalid   bool      `json:"invalid"`
}
