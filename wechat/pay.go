package wechat

import (
	"context"
	"log"
	"simctl/config"

	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/downloader"
	"github.com/wechatpay-apiv3/wechatpay-go/core/option"
	"github.com/wechatpay-apiv3/wechatpay-go/utils"
)

var (
	PayClient          *core.Client
	CertificateVisitor core.CertificateVisitor
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
}
