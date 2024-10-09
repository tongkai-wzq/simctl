package model

import (
	"fmt"
	"log"
	"simctl/db"
	"simctl/wechat"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
)

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

func (r *Rebates) ToAccount() {
	var receivers []profitsharing.CreateOrderReceiver
	receivers = append(receivers, profitsharing.CreateOrderReceiver{
		Account:     core.String(r.Agent.Openid),
		Amount:      core.Int64(int64(r.Amount * 100)),
		Description: core.String(fmt.Sprintf("套餐订单(%v)返佣", r.Order.OutTradeNo)),
		Type:        core.String("PERSONAL_OPENID"),
	})
	err := wechat.ProfitSharing(profitsharing.CreateOrderRequest{
		OutOrderNo:      core.String(fmt.Sprintf("%v-%v", r.Order.OutTradeNo, r.Id)),
		Receivers:       receivers,
		TransactionId:   core.String(r.Order.TransactionId),
		UnfreezeUnsplit: core.Bool(true),
	})
	if err == nil {
		r.Status = 1
		db.Engine.Cols("status").Update(r)
	} else {
		log.Printf("%v %v %v \n", r.Order.OutTradeNo, r.Agent.Name, err.Error())
	}
}
