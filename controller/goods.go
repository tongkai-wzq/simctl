package controller

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"simctl/config"
	"simctl/wechat"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
)

func Meals(w http.ResponseWriter, r *http.Request) {
	svc := profitsharing.ReceiversApiService{Client: wechat.PayClient}
	resp, result, err := svc.AddReceiver(context.Background(),
		profitsharing.AddReceiverRequest{
			Account:        core.String("oxpvg5YTFhmE_lgAsjQDUUPNPPnU"),
			Appid:          core.String(config.AppID),
			CustomRelation: core.String(fmt.Sprintf("代理商%v", "wzq")),
			RelationType:   profitsharing.RECEIVERRELATIONTYPE_DISTRIBUTOR.Ptr(),
			Type:           profitsharing.RECEIVERTYPE_PERSONAL_OPENID.Ptr(),
		},
	)

	if err != nil {
		// 处理错误
		log.Printf("call AddReceiver err:%s", err)
	} else {
		// 处理返回结果
		log.Printf("status=%d resp=%s", result.Response.StatusCode, resp)
	}
	w.Write([]byte("Hello World!"))
}
