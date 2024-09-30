package model

type GatewayUser struct {
	Id       int64
	Name     string            `json:"name"`
	Operator string            `json:"operator"`
	GwType   string            `json:"gwType"`
	Params   map[string]string `xorm:"json" json:"params"`
}
