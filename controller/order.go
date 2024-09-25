package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"simctl/config"
	"simctl/db"
	"simctl/model"
	"simctl/wechat"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/gorilla/websocket"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
)

func NewBuy(w http.ResponseWriter, r *http.Request) {
	_, claims, _ := jwtauth.FromContext(r.Context())
	var user model.User
	if has, err := db.Engine.ID(int64(claims["userId"].(float64))).Get(&user); err != nil || !has {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	if conn, err := upgrader.Upgrade(w, r, nil); err == nil {
		var buy Buy
		buy.Conn = conn
		buy.user = &user
		buy.Run(buy.GetHandleMap())
	} else {
		println(err.Error())
	}
}

type buyInitMsg struct {
	message
	SimId int64 `json:"simId"`
}

type buyInitResp struct {
	message
	Iccid     string            `json:"iccid"`
	Msisdn    string            `json:"msisdn"`
	MapNber   string            `json:"mapNber"`
	SaleMeals []*model.SaleMeal `json:"saleMeals"`
}

type buySubmitMsg struct {
	message
	MealKey   int64 `json:"mealKey"`
	NextMonth bool  `json:"nextMonth"`
}

type buyPacket struct {
	Base      bool      `json:"base"`
	StartAt   time.Time `json:"startAt"`
	ExpiredAt time.Time `json:"expiredAt"`
	Kb        int64     `json:"Kb"`
}

type buySubmitResp struct {
	message
	Packets []buyPacket `json:"packets"`
}

type Buy struct {
	widget
	user      *model.User
	saleMeals []*model.SaleMeal
	order     *model.Order
	packets   []*model.Packet
}

func (b *Buy) GetHandleMap() map[string]func(bMsg []byte) {
	return map[string]func(bMsg []byte){
		"init":   b.OnInit,
		"submit": b.OnSubmit,
	}
}

func (b *Buy) OnInit(bMsg []byte) {
	var iMsg buyInitMsg
	json.Unmarshal(bMsg, &iMsg)
	sim := new(model.Sim)
	db.Engine.ID(iMsg.SimId).Get(sim)
	b.order = &model.Order{
		UserId:  b.user.Id,
		User:    b.user,
		SimId:   sim.Id,
		Sim:     sim,
		AgentId: sim.AgentId,
	}
	b.order.LoadAgent()
	var iResp buyInitResp
	iResp.Iccid = sim.Iccid
	iResp.Msisdn = sim.Msisdn
	iResp.MapNber = sim.MapNber
	iResp.Handle = "init"
	b.saleMeals = sim.PreSaleMeals()
	iResp.SaleMeals = b.saleMeals
	if data, err := json.Marshal(&iResp); err == nil {
		b.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (b *Buy) OnSubmit(bMsg []byte) {
	var sMsg buySubmitMsg
	if err := json.Unmarshal(bMsg, &sMsg); err != nil {
		fmt.Println(err.Error())
	}
	b.order.MealId = b.saleMeals[sMsg.MealKey].MealId
	b.order.LoadMeal()
	b.order.NextMonth = sMsg.NextMonth
	b.order.Price = b.saleMeals[sMsg.MealKey].Price
	b.packets = b.order.PrePackets()
	var sResp buySubmitResp
	sResp.Handle = "submit"
	for _, packet := range b.packets {
		sResp.Packets = append(sResp.Packets, buyPacket{
			Base:      packet.Base,
			StartAt:   packet.StartAt,
			ExpiredAt: packet.ExpiredAt,
			Kb:        packet.Kb,
		})
	}
	if data, err := json.Marshal(&sResp); err == nil {
		b.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

func (b *Buy) OnUnify(bMsg []byte) {
	svc := jsapi.JsapiApiService{Client: wechat.PayClient}
	resp, result, err := svc.PrepayWithRequestPayment(context.Background(),
		jsapi.PrepayRequest{
			Appid:       core.String(config.AppID),
			Mchid:       core.String(config.MchID),
			Description: core.String("Image形象店-深圳腾大-QQ公仔"),
			OutTradeNo:  core.String("1217752501201407033233368018"),
			Attach:      core.String("自定义数据说明"),
			NotifyUrl:   core.String("https://www.weixin.qq.com/wxpay/pay.php"),
			Amount: &jsapi.Amount{
				Total: core.Int64(100),
			},
			Payer: &jsapi.Payer{
				Openid: core.String("oUpF8uMuAJO_M2pxb1Q9zNjWeS6o"),
			},
		},
	)

	if err == nil {
		log.Println(resp, result)
	} else {
		log.Println(err)
	}
}

func (b *Buy) Pay() {
	b.order.Status = 1
	db.Engine.Insert(b.order)
	b.order.SavePackets(b.packets)
}
