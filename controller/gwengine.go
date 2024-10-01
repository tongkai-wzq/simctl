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
	gwUser         model.GatewayUser
	qryItems       []*geItem
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
	for {
		if ge.lastId == 0 {
			time.Sleep(30 * time.Second)
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
			item.complete()
		}
		ge.qryItems = nil
	}
}

func (ge *GatewayEngine) initItems(sims []model.Sim) uint8 {
	var count uint8
	for _, sim := range sims {
		geItem := geItem{
			sim: sim,
		}
		if qryFuns := geItem.init(); len(qryFuns) > 0 {
			ge.qryItems = append(ge.qryItems, &geItem)
			count++
		}
	}
	if count > 0 {
		ge.statsQryFuns()
	}
	return count
}

func (ge *GatewayEngine) statsQryFuns() {
	counter := make(map[string]int, 10)
	for _, item := range ge.qryItems {
		for _, qryFun := range item.qryFuns {
			if count, ok := counter[qryFun]; ok {
				counter[qryFun] = count + 1
			} else {
				counter[qryFun] = 1
			}
		}
	}
	if len(counter) > 0 {
		ge.qryFunsCounter = counter
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
		sims := ge.getQryFunSims("QryDtls")
		for _, sim := range sims {
			simers = append(simers, sim)
		}
		gateway.QryDtls(simers)
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

func (ge *GatewayEngine) qryConcurt(qryFun string, callback func(sim gateway.Simer) error, size int) {
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
				callback(sim)
			}(sim)
		}
		wg.Wait()
		start = end
	}
}

func (ge *GatewayEngine) getQryFunSims(qryFun string) []*model.Sim {
	var sims []*model.Sim
	for _, item := range ge.qryItems {
		slices.Sort(item.qryFuns)
		if _, exist := slices.BinarySearch(item.qryFuns, qryFun); exist {
			sims = append(sims, &item.sim)
		}
	}
	return sims
}

type geItem struct {
	sim     model.Sim
	qryFuns []string
}

func (gei *geItem) init() []string {
	return nil
}

func (gei *geItem) complete() {

}
