package model

import "simctl/gateway"

type GatewayUser struct {
	Id       int64
	Name     string            `json:"name"`
	Operator string            `json:"operator"`
	GwType   string            `json:"gwType"`
	Params   map[string]string `xorm:"json" json:"params"`
	gateway  gateway.GateWayer `xorm:"-" json:"-"`
}

func (gu *GatewayUser) GateWay() gateway.GateWayer {
	if gu.gateway != nil {
		return gu.gateway
	}
	switch gu.GwType {
	case "mobile":
		gu.gateway = &gateway.Mobile{
			Appid:    gu.Params["appid"],
			Password: gu.Params["password"],
		}
		gu.gateway.SetGwUserId(gu.Id)
	case "unicom":
		gu.gateway = &gateway.Unicom{
			AppId:     gu.Params["appId"],
			AppSecret: gu.Params["appSecret"],
			OpenId:    gu.Params["openId"],
		}
		gu.gateway.SetGwUserId(gu.Id)
	case "telecom":
		gu.gateway = &gateway.Telecom{
			AppKey:     gu.Params["appKey"],
			SecretKey:  gu.Params["secretKey"],
			CustNumber: gu.Params["custNumber"],
		}
		gu.gateway.SetGwUserId(gu.Id)
	default:
		return nil
	}
	return gu.gateway
}
