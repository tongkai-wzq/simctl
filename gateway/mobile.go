package gateway

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"simctl/db"
	"strconv"
	"time"
)

const mUrl = "https://api.iot.10086.cn/v5"

type mResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Mobile struct {
	gateway
	Appid, Password string
}

func (m *Mobile) getTransid() string {
	now := time.Now()
	timeStr := now.Format("20060102150405")
	transid := fmt.Sprintf("%v%v%v", m.Appid, timeStr, strconv.Itoa(rand.New(rand.NewSource(now.UnixNano())).Intn(90000000)+10000000))
	return transid
}

func (m *Mobile) getToken() (string, error) {
	data := map[string]any{
		"appid":    m.Appid,
		"password": m.Password,
		"refresh":  0,
	}
	var resp struct {
		mResp
		Result []struct {
			Token string `json:"token"`
			Ttl   int    `json:"ttl"`
		} `json:"result"`
	}
	if err := m.post("/ec/get/token", data, &resp); err != nil {
		return "", fmt.Errorf("request token err %w", err)
	}
	return resp.Result[0].Token, nil
}

func (m *Mobile) post(uri string, data map[string]any, resp any) error {
	data["transid"] = m.getTransid()
	if uri != "/ec/get/token" {
		if token, err := m.getToken(); err == nil {
			data["token"] = token
		} else {
			return err
		}
	}
	req := gwClient.Post(mUrl + uri)
	req.SetBodyJsonMarshal(data)
	reader, err := m.send(req)
	if err != nil {
		return err
	}
	if err := json.NewDecoder(reader).Decode(resp); err != nil {
		return err
	}
	return nil
}

func (m *Mobile) QrySts(simer Simer) error {
	data := map[string]any{
		"iccid": simer.GetIccid(),
	}
	var resp struct {
		mResp
		Result []struct {
			CardStatus     string `json:"cardStatus"`
			LastChangeDate string `json:"lastChangeDate"`
		} `json:"result"`
	}
	if err := m.post("/ec/query/sim-status", data, &resp); err != nil {
		return fmt.Errorf("request sim-status err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request sim-status err : %v %v", resp.Status, resp.Message)
	}
	if status, err := strconv.Atoi(resp.Result[0].CardStatus); err == nil {
		simer.SetStatus(int8(status))
		db.Engine.Cols("status").Update(simer)
	} else {
		return fmt.Errorf("parse cardStatus err %w", err)
	}
	return nil
}

func (m *Mobile) QryAuthSts(simer Simer) error {
	data := map[string]any{
		"iccid": simer.GetIccid(),
	}
	var resp struct {
		mResp
		Result []struct {
			RealNameStatus string `json:"realNameStatus"`
			Reason         string `json:"reason"`
		} `json:"result"`
	}
	if err := m.post("/ec/query/sim-real-name-status", data, &resp); err != nil {
		return fmt.Errorf("request sim-real-name-status err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request sim-real-name-status err : %v %v", resp.Status, resp.Message)
	}
	if resp.Result[0].RealNameStatus == "1" {
		simer.SetAuth(true)
	} else {
		simer.SetAuth(false)
	}
	db.Engine.Cols("auth").Update(simer)
	return nil
}

func (m *Mobile) QryCmunt(simer Simer) error {
	data := map[string]any{
		"iccid": simer.GetIccid(),
	}
	var resp struct {
		mResp
		Result []struct {
			ServiceTypeList []struct {
				ServiceType   string `json:"serviceType"`
				ServiceStatus string `json:"serviceStatus"`
			} `json:"serviceTypeList"`
		} `json:"result"`
	}
	if err := m.post("/ec/query/sim-communication-function-status", data, &resp); err != nil {
		return fmt.Errorf("request sim-communication-function-status err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request sim-communication-function-status err : %v %v", resp.Status, resp.Message)
	}
	for _, service := range resp.Result[0].ServiceTypeList {
		if service.ServiceType == "11" {
			status, _ := strconv.Atoi(service.ServiceStatus)
			simer.SetFlowOn(int8(status))
		}
	}
	db.Engine.Cols("flow_on").Update(simer)
	return nil
}

func (m *Mobile) GetAuthLink(simer Simer) (string, error) {
	data := map[string]any{
		"iccid": simer.GetIccid(),
	}
	var resp struct {
		mResp
		Result []struct {
			BusiSeq string `json:"busiSeq"`
			Url     string `json:"url"`
		} `json:"result"`
	}
	if err := m.post("/ec/secure/sim-real-name-reg", data, &resp); err != nil {
		return "", fmt.Errorf("request sim-real-name-reg err : %w", err)
	} else if resp.Status != "0" {
		return "", fmt.Errorf("request sim-real-name-reg err : %v %v", resp.Status, resp.Message)
	}
	return resp.Result[0].Url, nil
}

func (m *Mobile) MtFlow(simer Simer) error {
	data := map[string]any{
		"iccid": simer.GetIccid(),
	}
	var resp struct {
		mResp
		Result []struct {
			DataAmount string `json:"dataAmount"`
		} `json:"result"`
	}
	if err := m.post("/ec/query/sim-data-usage", data, &resp); err != nil {
		return fmt.Errorf("request sim-data-usage err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request sim-data-usage err : %v %v", resp.Status, resp.Message)
	}
	if flow, err := strconv.ParseFloat(resp.Result[0].DataAmount, 64); err == nil {
		simer.SetMonthKb(int64(flow))
		db.Engine.Cols("month_kb", "month_at").Update(simer)
	} else {
		return fmt.Errorf("parse sim-data-usage err %w", err)
	}
	return nil
}

func (m *Mobile) MtVoice(simer Simer) error {
	data := map[string]any{
		"iccid": simer.GetIccid(),
	}
	var resp struct {
		mResp
		Result []struct {
			VoiceAmount string `json:"voiceAmount"`
		} `json:"result"`
	}
	if err := m.post("/ec/query/sim-voice-usage", data, &resp); err != nil {
		return fmt.Errorf("request sim-voice-usage err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request sim-voice-usage err : %v %v", resp.Status, resp.Message)
	}
	if voice, err := strconv.ParseFloat(resp.Result[0].VoiceAmount, 64); err == nil {
		log.Println(voice)
	} else {
		return fmt.Errorf("parse sim-voice-usage err %w", err)
	}
	return nil
}

// 0:申请停机(已激活转已停机) 1:申请复机(已停机转已激活) 2:库存转已激活 3:可测试转库存 4:可测试转待激活 5:可测试转已激活 6:待激活转已激活
func (m *Mobile) ChgLfcy(simer Simer, status int8) error {
	data := map[string]any{
		"iccid":    simer.GetIccid(),
		"operType": status,
	}
	var resp struct {
		mResp
		Result []struct {
			Iccid string `json:"iccid"`
		} `json:"result"`
	}
	if err := m.post("/ec/change/sim-status", data, &resp); err != nil {
		return fmt.Errorf("request change-sim-status err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request change-sim-status err : %v %v", resp.Status, resp.Message)
	}
	simer.SetStatus(status)
	db.Engine.Cols("status").Update(simer)
	return nil
}

func (m *Mobile) SwtFlowOn(simer Simer, flowOn int8) error {
	operType := 1
	if flowOn == 1 {
		operType = 0
	}
	data := map[string]any{
		"iccid":    simer.GetIccid(),
		"operType": operType,
		"apnName":  "CMIOT",
	}
	var resp struct {
		mResp
		Result []struct {
			OrderNum string `json:"orderNum"`
		} `json:"result"`
	}
	if err := m.post("/ec/operate/sim-apn-function", data, &resp); err != nil {
		return fmt.Errorf("request operate-sim-apn-function err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request operate-sim-apn-function err : %v %v", resp.Status, resp.Message)
	}
	simer.SetFlowOn(flowOn)
	db.Engine.Cols("flow_on").Update(simer)
	return nil
}

func (m *Mobile) SwtVoiceOn(simer Simer, voiceOn int8) error {
	operType := 1
	if voiceOn == 1 {
		operType = 0
	}
	data := map[string]any{
		"iccid":    simer.GetIccid(),
		"operType": operType,
	}
	var resp struct {
		mResp
		Result []struct {
			OrderNum string `json:"orderNum"`
		} `json:"result"`
	}
	if err := m.post("/ec/operate/sim-call-function", data, &resp); err != nil {
		return fmt.Errorf("request operate-sim-call-function err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request operate-sim-call-function err : %v %v", resp.Status, resp.Message)
	}
	return nil
}

// 1:速率恢复 91:APN-AMBR=2Mbps（月初不自动恢复）92:APN-AMBR=1Mbps（月初不自动恢复) 93:APN-AMBR=512Kbps（月初不自动恢复) 94:APN-AMBR=128Kbps（月初不自动恢复)
func (m *Mobile) LitRate(simer Simer, MB uint8) error {
	rateMap := map[uint8]int8{
		0: 1, 2: 91, 1: 92,
	}
	data := map[string]any{
		"iccid":   simer.GetIccid(),
		"apnName": "CMIOT",
	}
	if serviceUsageState, ok := rateMap[MB]; ok {
		data["serviceUsageState"] = serviceUsageState
	} else {
		data["serviceUsageState"] = 1
	}
	var resp struct {
		mResp
	}
	if err := m.post("/ec/operate/network-speed", data, &resp); err != nil {
		return fmt.Errorf("request operate-network-speed err : %w", err)
	} else if resp.Status != "0" {
		return fmt.Errorf("request operate-network-speed err : %v %v", resp.Status, resp.Message)
	}
	return nil
}
