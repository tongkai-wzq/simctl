package model

import (
	"simctl/db"
	"time"
)

type SaleMeal struct {
	MealId      int64   `json:"mealId"`
	Title       string  `json:"title"`
	Base        bool    `json:"base"`
	AcrossMonth bool    `json:"acrossMonth"`
	Price       float64 `json:"price"`
	Once        bool    `json:"once"`
	AcMthAble   bool    `json:"acMthAble"`
}

type Group struct {
	Id    int64   `json:"id"`
	Name  string  `json:"name"`
	Meals []*Meal `xorm:"-" json:"meals"`
}

func (g *Group) LoadMeals() {
	g.Meals = make([]*Meal, 0, 30)
	db.Engine.Where("group_id = ?", g.Id).Find(&g.Meals)
}

func (g *Group) GetSaleMeals() []*SaleMeal {
	var saleMeals []*SaleMeal
	for _, meal := range g.Meals {
		saleMeal := SaleMeal{
			MealId:      meal.Id,
			Title:       meal.Title,
			Base:        meal.Base,
			AcrossMonth: meal.AcrossMonth,
			Price:       meal.Price,
			Once:        meal.Once,
		}
		saleMeals = append(saleMeals, &saleMeal)
	}
	return saleMeals
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

func (m *Meal) LoadGroup() {
	m.Group = new(Group)
	db.Engine.ID(m.GroupId).Get(m.Group)
}

func (m *Meal) GetBeginAt(baseExpiredAt *time.Time, nextMonth bool) time.Time {
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

func (m *Meal) AlignPackets(startAt time.Time) []*Packet {
	var packets []*Packet
	packet := Packet{
		StartAt: startAt,
		Base:    m.Base,
		Kb:      m.Mb * 1024,
		KbCft:   m.MbCft,
	}
	if m.Base {
		for i := 1; i <= int(m.MonthNber); i++ {
			var first bool
			if i == 1 {
				first = true
			}
			packet.ExpiredAt = m.getExpiredAt(packet.StartAt, first)
			newPacket := packet
			packets = append(packets, &newPacket)
			packet.StartAt = packet.ExpiredAt.Add(time.Second)
		}
		return packets
	} else {
		if m.Day > 0 {
			packet.ExpiredAt = packet.StartAt.AddDate(0, 0, int(m.Day))
		} else {
			packet.ExpiredAt = m.getExpiredAt(packet.StartAt, true)
		}
		packets = append(packets, &packet)
		return packets
	}
}

func (m *Meal) getExpiredAt(startAt time.Time, first bool) time.Time {
	expiredAt := startAt.AddDate(0, 1, 0)
	if !m.AcrossMonth {
		if first {
			expiredAt = expiredAt.AddDate(0, 0, -expiredAt.Day()+1)
			expiredAt = expiredAt.Add(-time.Duration(expiredAt.Hour()) * time.Hour).Add(-time.Duration(expiredAt.Minute()) * time.Minute).Add(-time.Duration(expiredAt.Second()) * time.Second)
		}
		expiredAt = expiredAt.Add(-time.Second)
	}
	return expiredAt
}
