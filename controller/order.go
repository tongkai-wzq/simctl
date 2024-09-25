package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"simctl/config"
	"simctl/db"
	"simctl/model"
	"simctl/wechat"
	"time"

	"github.com/go-chi/jwtauth/v5"
	"github.com/gorilla/websocket"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/core/notify"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
)

var buyWidgets map[string]*Buy = make(map[string]*Buy)

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
		"unify":  b.OnUnify,
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

func (b *Buy) OnInit(bMsg []byte) {
	var iMsg buyInitMsg
	json.Unmarshal(bMsg, &iMsg)
	sim := new(model.Sim)
	db.Engine.ID(iMsg.SimId).Get(sim)
	now := time.Now()
	b.order = &model.Order{
		OutTradeNo: fmt.Sprintf("F%v%v", now.Format("200601021504"), rand.New(rand.NewSource(now.UnixNano())).Intn(9000)+1000),
		UserId:     b.user.Id,
		User:       b.user,
		SimId:      sim.Id,
		Sim:        sim,
		AgentId:    sim.AgentId,
	}
	b.order.LoadAgent()
	buyWidgets[b.order.OutTradeNo] = b
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

func (b *Buy) OnSubmit(bMsg []byte) {
	var sMsg buySubmitMsg
	if err := json.Unmarshal(bMsg, &sMsg); err != nil {
		fmt.Println(err.Error())
	}
	b.order.MealId = b.saleMeals[sMsg.MealKey].MealId
	b.order.LoadMeal()
	b.order.Title = b.order.Meal.Title
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

type unifyResp struct {
	message
	Data *jsapi.PrepayWithRequestPaymentResponse `json:"data"`
}

func (b *Buy) OnUnify(bMsg []byte) {
	svc := jsapi.JsapiApiService{Client: wechat.PayClient}
	resp, _, err := svc.PrepayWithRequestPayment(context.Background(),
		jsapi.PrepayRequest{
			Appid:       core.String(config.AppID),
			Mchid:       core.String(config.MchID),
			Description: core.String(regexp.MustCompile(`[^\w\p{Han}]+`).ReplaceAllString(b.order.Title, "")),
			OutTradeNo:  core.String(b.order.OutTradeNo),
			Attach:      core.String(fmt.Sprintf("原价%v", b.order.Price)),
			NotifyUrl:   core.String("https://api.ruiheiot.com/payNotify"),
			Amount: &jsapi.Amount{
				Total: core.Int64(int64(b.order.Price * 100)),
			},
			Payer: &jsapi.Payer{
				Openid: core.String(b.order.User.Openid),
			},
			SettleInfo: &jsapi.SettleInfo{
				ProfitSharing: core.Bool(true),
			},
		},
	)
	var uResp unifyResp
	uResp.Handle = "unify"
	if err == nil {
		uResp.Data = resp
	} else {
		uResp.Code = 4002
		uResp.Msg = err.Error()
	}
	if data, err := json.Marshal(&uResp); err == nil {
		b.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

func PayNotify(w http.ResponseWriter, r *http.Request) {
	var handler notify.Handler
	content := new(payments.Transaction)
	notifyReq, err := handler.ParseNotifyRequest(context.Background(), r, content)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(notifyReq.Summary)
	fmt.Println(content)
	b := buyWidgets[*content.OutTradeNo]
	if b.order.Status == 1 {
		w.Write(nil)
		return
	}
	b.order.TransactionId = *content.TransactionId
	b.order.Status = 1
	db.Engine.Insert(b.order)
	b.order.SavePackets(b.packets)
	w.Write(nil)
}
