package gateway

import (
	"crypto/md5"
	"encoding/xml"
	"fmt"
	"simctl/db"
	"sort"
	"strconv"
	"strings"
	"time"
)

const tUrl = "https://cmp-api.ctwing.cn:20164"

type Telecom struct {
	gateway
	AppKey     string
	SecretKey  string
	CustNumber string // 群组管理->企业id
}

func (t *Telecom) getSign(data map[string]any, timestamp string) string {
	var keys []string
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var str string
	for k, v := range keys {
		if k == 0 {
			str += fmt.Sprintf("%v=%v", v, data[v])
		} else {
			str += fmt.Sprintf("&%v=%v", v, data[v])
		}
	}
	str = str + t.SecretKey + timestamp
	sign16 := md5.Sum([]byte(str))
	hexDigits := [16]string{
		"0", "1", "2", "3", "4", "5", "6", "7", "8", "9", "a", "b", "c", "d", "e", "f",
	}
	var sign32 string
	for _, v := range sign16 {
		sign32 += hexDigits[v>>4&0xf]
		sign32 += hexDigits[v&0xf]
	}
	return fmt.Sprintf("%x", []byte(sign32))
}

func (t *Telecom) post(uri string, data map[string]any, resp any) error {
	timestamp := time.Now().Format("20060102150405")
	req := gwClient.Post(tUrl + uri)
	req.SetHeader("AppKey", t.AppKey)
	req.SetHeader("Timestamp", timestamp)
	req.SetHeader("Sign", strings.ToUpper(t.getSign(data, timestamp)))
	req.SetBody(data)
	if err := req.Do().Into(resp); err != nil { // "/5gcmp/openapi/v1/common/singleCutNet"  toXml
		return err
	}
	return nil
}

// status auth flowOn
// ：1：可激活2：测试激活3：测试去激活4：在用5：停机6：运营商管理状态
func (t *Telecom) QryStsMore(simer Simer) error {
	data := map[string]any{
		"access_number": simer.GetMsisdn(),
	}
	var resp struct {
		Result             string `json:"result"`
		ResultMsg          string `json:"resultMsg"`
		NetBlockStatusName string `json:"netBlockStatusName"`
		AuthStatus         string `json:"authStatus"`
		ProductInfo        []struct {
			ProductMainStatusCd          string `json:"productMainStatusCd"`
			ProductMainStatusName        string `json:"productMainStatusName"`
			OperatorDefinitionStatusName string `json:"operatorDefinitionStatusName"`
		} `json:"productInfo"`
	}
	if err := t.post("/openapi/v1/prodinst/queryCardMainStatus", data, &resp); err != nil {
		return fmt.Errorf("request queryCardMainStatus err : %w", err)
	} else if resp.Result != "0" {
		return fmt.Errorf("request queryCardMainStatus err : %v %v", resp.Result, resp.ResultMsg)
	}
	if status, err := strconv.Atoi(resp.ProductInfo[0].ProductMainStatusCd); err == nil {
		simer.SetStatus(int8(status))
	}
	if resp.AuthStatus == "1" {
		simer.SetAuth(true)
	} else if resp.AuthStatus == "0" {
		simer.SetAuth(false)
	}
	if resp.NetBlockStatusName == "未断网" {
		simer.SetFlowOn(1)
	} else if resp.NetBlockStatusName == "已断网" {
		simer.SetFlowOn(0)
	}
	simer.SetSyncAt()
	db.Engine.Cols("status", "auth", "flow_on", "sync_at").Update(simer)
	return nil
}

func (t *Telecom) QryAuthStses(simers []Simer) error {
	var msisdns []string
	for _, simer := range simers {
		msisdns = append(msisdns, simer.GetMsisdn())
	}
	data := map[string]any{
		"accessNumber": msisdns,
	}
	type stsItem struct {
		AccessNumber string `json:"accessNumber"`
		Status       string `json:"status"`
	}
	var resp struct {
		Code int    `json:"code"`
		Msg  string `json:"msg"`
		Data struct {
			StatusList []stsItem `json:"statusList"`
		} `json:"data"`
	}
	if err := t.post("/api/v1/realName/qryRealNameStatus", data, &resp); err != nil {
		return fmt.Errorf("request qryRealNameStatus err : %w", err)
	} else if resp.Code != 0 {
		return fmt.Errorf("request qryRealNameStatus err : %v %v", resp.Code, resp.Msg)
	}
	for _, simer := range simers {
		for _, item := range resp.Data.StatusList {
			if item.AccessNumber == simer.GetMsisdn() {
				if item.Status == "1" {
					simer.SetAuth(true)
				} else {
					simer.SetAuth(false)
				}
				simer.SetSyncAt()
				db.Engine.Cols("auth", "sync_at").Update(simer)
				break
			}
		}
	}
	return nil
}

/*
{\"code\":0,\"msg\":\"查询成功\",\"data\":[{\"accessNumber\":\"1064948626968\",\"result\":\"1\",\"unit\":\"MB\",\"flowAmount\":\"42118.59\",\"month\":\"202406\",\"status\":\"0\",\"statusDesc\":\"查询成功\"},{\"accessNumber\":\"1064945848612\",\"result\":\"1\",\"unit\":\"MB\",\"flowAmount\":\"35999.65\",\"month\":\"202406\",\"status\":\"0\",\"statusDesc\":\"查询成功\"}]}
*/
func (t *Telecom) MtFlows(simers []Simer) error {
	var msisdns []string
	for _, simer := range simers {
		msisdns = append(msisdns, simer.GetMsisdn())
	}
	data := map[string]any{
		"custNumber":   t.CustNumber,
		"accessNumber": msisdns,
	}
	type flowItem struct {
		AccessNumber string `json:"accessNumber"`
		Result       string `json:"result"`
		Unit         string `json:"unit"`
		FlowAmount   string `json:"flowAmount"`
	}
	var resp struct {
		Code int        `json:"code"`
		Msg  string     `json:"msg"`
		Data []flowItem `json:"data"`
	}
	if err := t.post("/api/v1/batchQry/batchQryFlowByMonth", data, &resp); err != nil {
		return fmt.Errorf("request batchQryFlowByMonth err : %w", err)
	} else if resp.Code != 0 {
		return fmt.Errorf("request batchQryFlowByMonth err : %v %v", resp.Code, resp.Msg)
	}
	for _, simer := range simers {
		for _, item := range resp.Data {
			if item.AccessNumber == simer.GetMsisdn() {
				if flowAmount, err := strconv.ParseFloat(item.FlowAmount, 64); err == nil {
					simer.SetMonthKb(int64(flowAmount * 1024))
					db.Engine.Cols("month_kb", "month_at").Update(simer)
				}
				break
			}
		}
	}
	return nil
}

func (t *Telecom) ChgLfcy(simer Simer, status int8) error {
	return nil
}

func (t *Telecom) SwtFlowOn(simer Simer, flowOn int8) error {
	action := "ADD"
	if flowOn == 1 {
		action = "DEL"
	}
	data := map[string]any{
		"access_number": simer.GetMsisdn(),
		"action":        action,
	}
	var resp struct {
		XMLName  xml.Name `xml:"SvcCont"`
		Response struct {
			RspType string `xml:"RspType"`
			RspCode string `xml:"RspCode"`
			RspDesc string `xml:"RspDesc"`
		} `xml:"Response"`
		GROUPTRANSACTIONID string `xml:"GROUP_TRANSACTIONID"`
	}
	if err := t.post("/5gcmp/openapi/v1/common/singleCutNet", data, &resp); err != nil {
		return fmt.Errorf("request singleCutNet err : %w", err)
	} else if resp.Response.RspCode != "0000" {
		return fmt.Errorf("request singleCutNet err : %v %v", resp.Response.RspCode, resp.Response.RspDesc)
	}
	simer.SetFlowOn(flowOn)
	db.Engine.Cols("flow_on").Update(simer)
	return nil
}

func (t *Telecom) LitRate(simer Simer, MB uint8) error {
	data := map[string]any{
		"access_number": simer.GetMsisdn(),
		"action":        "ADD",
	}
	rateMap := map[uint8]int8{
		1: 12,
	}
	if speedValue, ok := rateMap[MB]; ok {
		data["speedValue"] = speedValue
	} else {
		data["speedValue"] = "10"
		data["action"] = "DEL"
	}
	var resp struct {
		ResultCode string `json:"resultCode"`
		ResultMsg  string `json:"resultMsg"`
	}
	if err := t.post("/5gcmp/openapi/v1/common/speedLimitAction", data, &resp); err != nil {
		return fmt.Errorf("request speedLimitAction err : %w", err)
	} else if resp.ResultCode != "0000" {
		return fmt.Errorf("request speedLimitAction err : %v %v", resp.ResultCode, resp.ResultMsg)
	}
	return nil
}
