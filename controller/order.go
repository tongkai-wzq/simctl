package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"simctl/config"
	"simctl/db"
	"simctl/model"
	"simctl/wechat"
	"time"

	"github.com/gorilla/websocket"
	"github.com/wechatpay-apiv3/wechatpay-go/core"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments"
	"github.com/wechatpay-apiv3/wechatpay-go/services/payments/jsapi"
)

var buyWidgets map[string]*Buy = make(map[string]*Buy)

func NewBuy(w http.ResponseWriter, r *http.Request) {
	var (
		buy   *Buy
		exist bool
	)
	outTradeNo := r.URL.Query().Get("outTradeNo")
	if buy, exist = buyWidgets[outTradeNo]; outTradeNo == "" || !exist {
		buy = new(Buy)
		if user := AuthUser(r); user != nil {
			buy.user = user
		} else {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
	}
	if conn, err := upgrader.Upgrade(w, r, nil); err == nil {
		buy.SetConn(conn)
		if simId := r.URL.Query().Get("simId"); outTradeNo == "" && simId != "" {
			go buy.init(simId)
		}
		buy.Read(buy)
	}
}

type Buy struct {
	widget
	user      *model.User
	saleMeals []*model.SaleMeal
	order     *model.Order
	prepay    *jsapi.PrepayWithRequestPaymentResponse
}

func (b *Buy) GetHandleMap() map[string]func(bMsg []byte) {
	return map[string]func(bMsg []byte){
		"submit": b.OnSubmit,
		"unify":  b.OnUnify,
	}
}

type buyInitResp struct {
	OutTradeNo string `json:"outTradeNo"`
	message
	simDtl
	SaleMeals []*model.SaleMeal `json:"saleMeals"`
}

func (b *Buy) init(simId string) {
	sim := new(model.Sim)
	db.Engine.ID(simId).Get(sim)
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
	iResp.OutTradeNo = b.order.OutTradeNo
	iResp.Id = sim.Id
	iResp.Iccid = sim.Iccid
	iResp.Msisdn = sim.Msisdn
	iResp.MapNber = sim.MapNber
	iResp.Auth = sim.Auth
	iResp.FlowOn = sim.FlowOn
	iResp.Status = sim.Status
	iResp.Operator = sim.GetGwUser().Operator
	iResp.Handle = "init"
	sim.LoadGroup()
	if saleMeals, err := sim.PreSaleMeals(); err == nil {
		b.saleMeals = saleMeals
		iResp.SaleMeals = b.saleMeals
	} else {
		iResp.Code = 4004
		iResp.Msg = err.Error()
	}
	if data, err := json.Marshal(&iResp); err == nil {
		b.conn.WriteMessage(websocket.TextMessage, data)
	}
}

type buySubmitMsg struct {
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
		log.Println(err.Error())
	}
	b.order.MealId = b.saleMeals[sMsg.MealKey].MealId
	b.order.LoadMeal()
	b.order.Title = b.order.Meal.Title
	b.order.NextMonth = sMsg.NextMonth
	b.order.Price = b.saleMeals[sMsg.MealKey].Price
	b.order.PrePackets()
	var sResp buySubmitResp
	sResp.Handle = "submit"
	for _, packet := range b.order.Packets {
		sResp.Packets = append(sResp.Packets, buyPacket{
			Base:      packet.Base,
			StartAt:   packet.StartAt,
			ExpiredAt: packet.ExpiredAt,
			Kb:        packet.Kb,
		})
	}
	if data, err := json.Marshal(&sResp); err == nil {
		b.conn.WriteMessage(websocket.TextMessage, data)
	}
}

type unifyResp struct {
	message
	Data *jsapi.PrepayWithRequestPaymentResponse `json:"data"`
}

func (b *Buy) OnUnify(bMsg []byte) {
	if b.order.Status == 0 && b.prepay != nil {
		wechat.CloseOrder(b.order.OutTradeNo)
	}
	var uResp unifyResp
	uResp.Handle = "unify"
	if resp, err := wechat.Prepay(jsapi.PrepayRequest{
		Description: core.String(regexp.MustCompile(`[^\w\p{Han}]+`).ReplaceAllString(b.order.Title, "")),
		OutTradeNo:  core.String(b.order.OutTradeNo),
		Attach:      core.String(fmt.Sprintf("原价%v", b.order.Price)),
		NotifyUrl:   core.String(fmt.Sprintf("%v/payNotify", config.Domain)),
		Amount: &jsapi.Amount{
			Total: core.Int64(int64(b.order.Price * 100)),
		},
		Payer: &jsapi.Payer{
			Openid: core.String(b.order.User.Openid),
		},
		SettleInfo: &jsapi.SettleInfo{
			ProfitSharing: core.Bool(true),
		},
	}); err == nil {
		b.prepay = resp
		uResp.Data = b.prepay
	} else {
		uResp.Code = 4002
		uResp.Msg = err.Error()
	}
	if data, err := json.Marshal(&uResp); err == nil {
		b.conn.WriteMessage(websocket.TextMessage, data)
	}
}

func PayNotify(w http.ResponseWriter, r *http.Request) {
	content := new(payments.Transaction)
	_, err := wechat.NotifyHandle.ParseNotifyRequest(context.Background(), r, content)
	if err != nil {
		log.Println("ParseNotifyRequest", err)
		w.Write(nil)
		return
	}
	b := buyWidgets[*content.OutTradeNo]
	if b.order.Status == 1 {
		w.Write(nil)
		return
	}
	b.order.TransactionId = *content.TransactionId
	b.order.Amount = float64(*content.Amount.PayerTotal) / 100
	b.order.Status = 1
	db.Engine.Insert(b.order)
	b.order.SavePackets()
	if err := b.order.GiveRbt(); err != nil {
		log.Println(err.Error())
	}
	w.Write(nil)
}

func (b *Buy) End() {
	if b.order.Status == 0 && b.prepay != nil {
		wechat.CloseOrder(b.order.OutTradeNo)
	}
	delete(buyWidgets, b.order.OutTradeNo)
}
