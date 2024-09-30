package model

import "simctl/gateway"

type GatewayUser struct {
	Id       int64
	Name     string            `json:"name"`
	Operator string            `json:"operator"`
	GwType   string            `json:"gwType"`
	Params   map[string]string `xorm:"json" json:"params"`
	Gateway  gateway.GateWayer `xorm:"-" json:"-"`
}

func (gu *GatewayUser) GateWay() gateway.GateWayer {
	if gu.Gateway != nil {
		return gu.Gateway
	}
	switch gu.GwType {
	case "mobile":
		gu.Gateway = &gateway.Mobile{
			Appid:    gu.Params["appid"],
			Password: gu.Params["password"],
		}
		gu.Gateway.SetGwUserId(gu.Id)
	case "unicom":
		gu.Gateway = &gateway.Unicom{
			AppId:     gu.Params["appId"],
			AppSecret: gu.Params["appSecret"],
			OpenId:    gu.Params["openId"],
		}
		gu.Gateway.SetGwUserId(gu.Id)
	case "telecom":
		gu.Gateway = &gateway.Telecom{
			AppKey:     gu.Params["appKey"],
			SecretKey:  gu.Params["secretKey"],
			CustNumber: gu.Params["custNumber"],
		}
		gu.Gateway.SetGwUserId(gu.Id)
	default:
		return nil
	}
	return gu.Gateway
}
