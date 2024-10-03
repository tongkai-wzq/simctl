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

func (gu *GatewayUser) GetId() int64 {
	return gu.Id
}
func (gu *GatewayUser) GetName() string {
	return gu.Name
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
		gu.Gateway.SetGwUser(gu)
	case "unicom":
		gu.Gateway = &gateway.Unicom{
			AppId:     gu.Params["appId"],
			AppSecret: gu.Params["appSecret"],
			OpenId:    gu.Params["openId"],
		}
		gu.Gateway.SetGwUser(gu)
	case "telecom":
		gu.Gateway = &gateway.Telecom{
			AppKey:     gu.Params["appKey"],
			SecretKey:  gu.Params["secretKey"],
			CustNumber: gu.Params["custNumber"],
		}
		gu.Gateway.SetGwUser(gu)
	default:
		return nil
	}
	return gu.Gateway
}
