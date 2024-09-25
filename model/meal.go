package model

import "time"

type Group struct {
	Id   int64  `json:"id"`
	Name string `json:"name"`
}

type Meal struct {
	Id          int64   `json:"id"`
	GroupId     int64   `json:"groupId"`
	Group       *Group  `xorm:"-" json:"group"`
	Title       string  `json:"title"`
	Base        bool    `json:"base"`
	MonthNber   int64   `json:"monthNber"`
	AcrossMonth bool    `json:"acrossMonth"`
	Day         int64   `json:"day"`
	Price       float64 `json:"price"`
	Mb          int64   `json:"Mb"`
	MbCft       float64 `json:"mbCft"`
	Once        bool    `json:"once"`
}

func (m *Meal) RsvBeginAt(baseExpiredAt *time.Time, nextMonth bool) time.Time {
	now := time.Now()
	next := now.AddDate(0, 1, -now.Day()+1).Add(-time.Duration(now.Hour()) * time.Hour).Add(-time.Duration(now.Minute()) * time.Minute).Add(-time.Duration(now.Second()) * time.Second)
	nOrN := func() time.Time {
		if !m.AcrossMonth && nextMonth {
			return next
		} else {
			return now
		}
	}
	if m.Base {
		if baseExpiredAt == nil {
			return nOrN()
		} else if baseExpiredAt.After(now) {
			return baseExpiredAt.Add(time.Second)
		} else {
			return nOrN()
		}
	} else {
		return nOrN()
	}
}

func (m *Meal) AgtPackets(startAt time.Time) []Packet {
	var packets []Packet
	if m.Base {
		packet := Packet{
			StartAt: startAt,
			Base:    m.Base,
			Kb:      m.Mb * 1024,
			KbCft:   m.MbCft,
		}
		for i := 0; i < int(m.MonthNber); i++ {
			if i == 0 {
				packet.ExpiredAt = m.rsvCcExpiredAt(packet.StartAt, false)
			} else {
				packet.ExpiredAt = m.rsvCcExpiredAt(packet.StartAt, true)
			}
			packets = append(packets, packet)
			packet.StartAt = packet.ExpiredAt.Add(time.Second)
		}
		return packets
	} else {
		packet := Packet{
			StartAt: startAt,
			Base:    m.Base,
			Kb:      m.Mb * 1024,
			KbCft:   m.MbCft,
		}
		if m.Day > 0 {
			packet.ExpiredAt = packet.StartAt.AddDate(0, 0, int(m.Day))
		} else {
			packet.ExpiredAt = m.rsvCcExpiredAt(packet.StartAt, false)
		}
		packets = append(packets, packet)
		return packets
	}
}

func (m *Meal) rsvCcExpiredAt(startAt time.Time, ccNext bool) time.Time {
	var expiredAt time.Time
	if m.AcrossMonth {
		expiredAt = startAt.AddDate(0, 1, 0)
	} else {
		expiredAt = startAt.AddDate(0, 1, 0)
		if !ccNext {
			expiredAt = expiredAt.AddDate(0, 0, -expiredAt.Day()+1)
			expiredAt = expiredAt.Add(-time.Duration(expiredAt.Hour()) * time.Hour).Add(-time.Duration(expiredAt.Minute()) * time.Minute).Add(-time.Duration(expiredAt.Second()) * time.Second)
		}
		expiredAt = expiredAt.Add(-time.Second)
	}
	return expiredAt
}
