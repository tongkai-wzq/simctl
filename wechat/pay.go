package wechat

import (
	"context"
	"log"
	"simctl/config"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/auth/verifiers"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
	"github.com/wechatpay-apiv3/wechatpay-go/services/profitsharing"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

var (
	PayClient          *core.Client
	CertificateVisitor core.CertificateVisitor
	NotifyHandle       *notify.Handler
)

func init() {
	mchPrivateKey, err := utils.LoadPrivateKeyWithPath("./apiclient_key.pem")
	if err != nil {
		log.Fatal("load merchant private key error")
	}
	ctx := context.Background()
	opts := []core.ClientOption{
		option.WithWechatPayAutoAuthCipher(config.MchID, config.MchCertificateSerialNumber, mchPrivateKey, config.MchAPIv3Key),
	}
	PayClient, err = core.NewClient(ctx, opts...)
	if err != nil {
		log.Fatalf("new wechat pay client err:%s", err)
	}
	downloader.MgrInstance().RegisterDownloaderWithPrivateKey(ctx, mchPrivateKey, config.MchCertificateSerialNumber, config.MchID, config.MchAPIv3Key)
	CertificateVisitor = downloader.MgrInstance().GetCertificateVisitor(config.MchID)
	NotifyHandle = notify.NewNotifyHandler(config.MchAPIv3Key, verifiers.NewSHA256WithRSAVerifier(CertificateVisitor))
}

func Prepay(prepayRequest jsapi.PrepayRequest) (*jsapi.PrepayWithRequestPaymentResponse, error) {
	svc := jsapi.JsapiApiService{Client: PayClient}
	prepayRequest.Appid = core.String(config.AppID)
	prepayRequest.Mchid = core.String(config.MchID)
	resp, _, err := svc.PrepayWithRequestPayment(context.Background(), prepayRequest)
	return resp, err
}

func CloseOrder(outTradeNo string) error {
	svc := jsapi.JsapiApiService{Client: PayClient}
	_, err := svc.CloseOrder(context.Background(), jsapi.CloseOrderRequest{
		OutTradeNo: &outTradeNo,
		Mchid:      core.String(config.MchID),
	})
	return err
}

func ProfitSharing(createOrderRequest profitsharing.CreateOrderRequest) error {
	svc := profitsharing.OrdersApiService{Client: PayClient}
	createOrderRequest.Appid = core.String(config.AppID)
	_, _, err := svc.CreateOrder(context.Background(), createOrderRequest)
	return err
}
