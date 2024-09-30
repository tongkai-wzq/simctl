package model

import "simctl/gateway"

var GateWays map[int64]gateway.GateWayer = make(map[int64]gateway.GateWayer)

type GatewayUser struct {
	Id       int64
	Name     string            `json:"name"`
	Operator string            `json:"operator"`
	GwType   string            `json:"gwType"`
	Params   map[string]string `xorm:"json" json:"params"`
}

func (gu *GatewayUser) BuildGateWay() gateway.GateWayer {
	if gateway, ok := GateWays[gu.Id]; ok {
		return gateway
	}
	switch gu.GwType {
	case "mobile":
		GateWays[gu.Id] = &gateway.Mobile{
			Appid:    gu.Params["appid"],
			Password: gu.Params["password"],
		}
		GateWays[gu.Id].SetGwUserId(gu.Id)
	case "unicom":
		GateWays[gu.Id] = &gateway.Unicom{
			AppId:     gu.Params["appId"],
			AppSecret: gu.Params["appSecret"],
			OpenId:    gu.Params["openId"],
		}
		GateWays[gu.Id].SetGwUserId(gu.Id)
	case "telecom":
		GateWays[gu.Id] = &gateway.Telecom{
			AppKey:     gu.Params["appKey"],
			SecretKey:  gu.Params["secretKey"],
			CustNumber: gu.Params["custNumber"],
		}
		GateWays[gu.Id].SetGwUserId(gu.Id)
	default:
		return nil
	}
	return GateWays[gu.Id]
}
