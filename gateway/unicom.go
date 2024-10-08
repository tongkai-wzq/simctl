package gateway

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"simctl/db"
	"strconv"
	"time"

	"github.com/tjfoc/gmsm/sm3"
)

const uUrl = "https://gwapi.10646.cn/api"

type uResp struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

type Unicom struct {
	gateway
	AppId     string
	AppSecret string
	OpenId    string
}

func (u *Unicom) getParams() map[string]any {
	now := time.Now()
	msec := now.Format(".000")
	transId := fmt.Sprintf("%v%v%v", now.Format("20060102150405"), msec[1:], strconv.Itoa(rand.New(rand.NewSource(now.UnixNano())).Intn(900000)+100000))
	params := map[string]any{
		"app_id":    u.AppId,
		"timestamp": fmt.Sprintf("%v %v", now.Format("2006-01-02 15:04:05"), msec[1:]),
		"trans_id":  transId,
	}
	var paramsStr string
	for k, v := range params {
		paramsStr = fmt.Sprintf("%v%v%v", paramsStr, k, v)
	}
	params["token"] = hex.EncodeToString(sm3.Sm3Sum([]byte(fmt.Sprintf("%v%v", paramsStr, u.AppSecret))))
	return params
}

func (u *Unicom) post(uri string, data map[string]any, resp any) error {
	params := u.getParams()
	data["messageId"] = "simctl"
	data["openId"] = u.OpenId
	data["version"] = "1.0"
	params["data"] = data
	req := gwClient.Post(uUrl + uri)
	req.SetBody(params)
	if err := req.Do().Into(resp); err != nil {
		return err
	}
	return nil
}

func (u *Unicom) ChgLfcy(simer Simer, status int8) error {
	if err := u.chgAttr(simer, "3", strconv.Itoa(int(status))); err == nil {
		simer.SetStatus(status)
		db.Engine.Cols("status").Update(simer)
	} else {
		return err
	}
	return nil
}

func (u *Unicom) chgAttr(simer Simer, changeType string, targetValue string) error {
	data := map[string]any{
		"asynchronous": "0",
		"iccid":        simer.GetIccid(),
		"changeType":   changeType,
		"targetValue":  targetValue,
	}
	var resp struct {
		uResp
		Data struct {
			Iccid      string `json:"iccid"`
			ResultCode string `json:"resultCode"`
		} `json:"data"`
	}
	if err := u.post("/wsEditTerminal/V1/1Main/vV1.1", data, &resp); err != nil {
		return fmt.Errorf("request wsEditTerminal err : %w", err)
	} else if resp.Status != "0000" {
		return fmt.Errorf("request wsEditTerminal err : %v %v", resp.Status, resp.Message)
	}
	return nil
}

type terminalDetail struct {
	Iccid                 string `json:"iccid"`
	MonthToDateUsage      string `json:"monthToDateUsage"`
	MonthToDateDataUsage  string `json:"monthToDateDataUsage"`
	MonthToDateVoiceUsage string `json:"monthToDateVoiceUsage"`
	SimStatus             string `json:"simStatus"`
	RealNameStatus        string `json:"realNameStatus"`
	Imei                  string `json:"imei"`
}

func (u *Unicom) QryDtls(simers []Simer) error {
	var iccids []string
	for _, simer := range simers {
		iccids = append(iccids, simer.GetIccid())
	}
	data := map[string]any{
		"iccids": iccids,
	}
	var resp struct {
		uResp
		Data struct {
			Terminals []terminalDetail `json:"terminals"`
		} `json:"data"`
	}
	if err := u.post("/wsGetTerminalDetails/V1/1Main/vV1.1", data, &resp); err != nil {
		return fmt.Errorf("request wsGetTerminalDetails err : %w", err)
	} else if resp.Status != "0000" {
		return fmt.Errorf("request wsGetTerminalDetails err : %v %v", resp.Status, resp.Message)
	}
	for _, terminal := range resp.Data.Terminals {
		for _, simer := range simers {
			if terminal.Iccid != simer.GetIccid() {
				continue
			}
			if status, err := strconv.Atoi(terminal.SimStatus); err == nil {
				simer.SetStatus(int8(status))
			}
			if terminal.RealNameStatus == "2" || terminal.RealNameStatus == "3" {
				simer.SetAuth(true)
			} else {
				simer.SetAuth(false)
			}
			if monthToDateUsage, err := strconv.ParseFloat(terminal.MonthToDateDataUsage, 64); err == nil {
				simer.SetMonthKb(int64(monthToDateUsage * 1024))
			}
			simer.SetSyncAt()
			db.Engine.Cols("status", "auth", "month_kb", "month_at", "sync_at").Update(simer)
			break
		}
	}
	return nil
}
