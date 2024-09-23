package model

import "simctl/db"

type Agent struct {
	Id         int64  `json:"id"`
	Name       string `json:"name"`
	SuperiorId int64  `json:"superiorId"`
	Superior   *Agent `json:"superior"`
}

func (a *Agent) LoadSuperior() {
	a.Superior = new(Agent)
	db.Engine.ID(a.SuperiorId).Get(a.Superior)
}
