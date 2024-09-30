package controller

import (
	"simctl/db"
	"simctl/model"
)

type GatewayEngine struct {
	lastId         int64
	gatewayUser    model.GatewayUser
	qryItems       []*geItem
	qryFunsCounter map[string]int
}

func (ge *GatewayEngine) GetSims() []model.Sim {
	var sims []model.Sim
	if db.Engine.Where("gwuser_id = ? AND id > ?", ge.gatewayUser.Id, ge.lastId).OrderBy("id").Limit(10).Find(&sims); len(sims) == 0 {
		ge.lastId = 0
	} else {
		ge.lastId = sims[len(sims)-1].Id
	}
	return sims
}

type geItem struct {
	sim     model.Sim
	qryFuns []string
}
