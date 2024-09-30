package model

type Rebates struct {
	Id        int64  `json:"id"`
	AgentId   int64  `json:"agentId"`
	Agent     *Agent `xorm:"-" json:"agent"`
	OrderId   int64
	Order     *Order  `xorm:"-" json:"order"`
	Amount    float64 `json:"amount"`
	RefundAmt float64 `json:"refundAmt"`
	Status    int8    `json:"status"`
}
