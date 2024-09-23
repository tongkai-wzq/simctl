package model

type Rebates struct {
	Id        int64
	AgentId   int64  `json:"agentId"`
	Agent     *Agent `xorm:"-" json:"agent"`
	OrderId   int64
	Order     *Order  `xorm:"-" json:"order"`
	Amount    float64 `json:"amount"`
	RefundAmt float64 `json:"refundAmt"`
	Status    int64   `json:"status"`
}
