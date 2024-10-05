package controller

import (
	"simctl/db"
	"simctl/gateway"
	"simctl/model"
	"slices"
	"sync"
	"time"
)

type GatewayEngine struct {
	lastId         int64
	gwUser         *model.GatewayUser
	qryItems       []*model.Sim
	qryFunsCounter map[string]int
}

func (ge *GatewayEngine) GetSims() []model.Sim {
	sims := make([]model.Sim, 0)
	if db.Engine.Where("gwuser_id = ? AND id > ?", ge.gwUser.Id, ge.lastId).OrderBy("id").Limit(10).Find(&sims); len(sims) > 0 {
		ge.lastId = sims[len(sims)-1].Id
	} else {
		ge.lastId = 0
	}
	return sims
}

func (ge *GatewayEngine) Run() {
	ge.qryFunsCounter = make(map[string]int)
	for {
		if ge.lastId == 0 {
			time.Sleep(3 * time.Second)
		}
		for {
			sims := ge.GetSims()
			if len(sims) == 0 {
				break
			}
			if count := ge.initItems(sims); count == 0 {
				continue
			}
			if ge.isEnough() {
				break
			}
		}
		if len(ge.qryItems) == 0 {
			continue
		}
		ge.qry()
		for _, item := range ge.qryItems {
			item.QryComplete()
		}
		ge.qryItems = nil
	}
}

func (ge *GatewayEngine) initItems(sims []model.Sim) int {
	var count int
	for _, sim := range sims {
		if qryFuns := sim.QryInit(); len(qryFuns) > 0 {
			ge.qryItems = append(ge.qryItems, &sim)
			count++
			ge.statsQryFuns(qryFuns)
		}
	}
	return count
}

func (ge *GatewayEngine) statsQryFuns(qryFuns []string) {
	for _, qryFun := range qryFuns {
		if count, ok := ge.qryFunsCounter[qryFun]; ok {
			ge.qryFunsCounter[qryFun] = count + 1
		} else {
			ge.qryFunsCounter[qryFun] = 1
		}
	}
}

func (ge *GatewayEngine) isEnough() bool {
	switch ge.gwUser.Gateway.(type) {
	case *gateway.Unicom:
		if count, ok := ge.qryFunsCounter["QryDtls"]; ok && count > 20 {
			return true
		}
	case *gateway.Mobile:
		for _, count := range ge.qryFunsCounter {
			if count > 300 {
				return true
			}
		}
	case *gateway.Telecom:
		if count, ok := ge.qryFunsCounter["MtFlows"]; ok && count > 50 {
			return true
		}
		if count, ok := ge.qryFunsCounter["QryStsMore"]; ok && count > 300 {
			return true
		}
		if count, ok := ge.qryFunsCounter["QryAuthStses"]; ok && count > 50 {
			return true
		}
	}
	return false
}

func (ge *GatewayEngine) qry() {
	var simers []gateway.Simer
	switch gateway := ge.gwUser.Gateway.(type) {
	case *gateway.Unicom:
		if count, ok := ge.qryFunsCounter["QryDtls"]; ok && count > 0 {
			sims := ge.getQryFunSims("QryDtls")
			for _, sim := range sims {
				simers = append(simers, sim)
			}
			gateway.QryDtls(simers)
		}
	case *gateway.Mobile:
		for qryFun := range ge.qryFunsCounter {
			if qryFun == "QrySts" {
				ge.qryConcurt(qryFun, gateway.QrySts, 30)
			}
			if qryFun == "QryAuthSts" {
				ge.qryConcurt(qryFun, gateway.QryAuthSts, 30)
			}
			if qryFun == "QryCmunt" {
				ge.qryConcurt(qryFun, gateway.QryCmunt, 30)
			}
			if qryFun == "MtFlow" {
				ge.qryConcurt(qryFun, gateway.MtFlow, 30)
			}
		}
	case *gateway.Telecom:
		for qryFun := range ge.qryFunsCounter {
			if qryFun == "QryAuthStses" {
				sims := ge.getQryFunSims(qryFun)
				for _, sim := range sims {
					simers = append(simers, sim)
				}
				gateway.QryAuthStses(simers)
			}
			if qryFun == "MtFlows" {
				sims := ge.getQryFunSims(qryFun)
				for _, sim := range sims {
					simers = append(simers, sim)
				}
				gateway.MtFlows(simers)
			}
			if qryFun == "QryStsMore" {
				ge.qryConcurt(qryFun, gateway.QryStsMore, 10)
			}
			simers = nil
		}
	}
}

func (ge *GatewayEngine) qryConcurt(qryFun string, qry func(sim gateway.Simer) error, size int) {
	sims := ge.getQryFunSims(qryFun)
	var start, end int
	for {
		if start == len(sims) {
			break
		}
		end = start + size
		if end > len(sims) {
			end = len(sims)
		}
		bSims := sims[start:end]
		var wg sync.WaitGroup
		for _, sim := range bSims {
			wg.Add(1)
			go func(sim *model.Sim) {
				defer wg.Done()
				qry(sim)
			}(sim)
		}
		wg.Wait()
		start = end
	}
}

func (ge *GatewayEngine) getQryFunSims(qryFun string) []*model.Sim {
	var sims []*model.Sim
	for _, item := range ge.qryItems {
		if exist := slices.Contains(item.QryFuns, qryFun); exist {
			sims = append(sims, item)
		}
	}
	return sims
}
