package wechat

import (
	"log"
	"simctl/config"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/miniProgram"
)

var (
	MiniClient *miniProgram.MiniProgram
	err        error
)

func init() {
	MiniClient, err = miniProgram.NewMiniProgram(&miniProgram.UserConfig{
		AppID:     config.AppID,  // 小程序appid
		Secret:    config.Secret, // 小程序app secret
		HttpDebug: false,
	})
	if err != nil {
		log.Fatal("load merchant private key error")
	}
}
