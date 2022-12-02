package black

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
)

type apiStoreHandle struct {
	apiKey string
	token  string
	url    string
	BlackMethod
	handle *Blackhandle
}

var gApiStoreHandle apiStoreHandle

func AppStoreSmsVerify(number, busineseType string) (bool, error) {
	if gApiStoreHandle.handle == nil {
		return false, errors.New("not init")
	}
	return gApiStoreHandle.handle.AddVerify(number, busineseType)
}

func NewApiStoreHandle(apiKey, token, url string) {

	logs.Debug("启动apistore黑名单校验...")
	gApiStoreHandle.apiKey = apiKey
	gApiStoreHandle.token = token
	gApiStoreHandle.url = url
	gApiStoreHandle.handle = NewHanlde(gApiStoreHandle.Verify, 5)
}

type respInfo struct {
	Code int      `json:"code"` //状态码（0：正常；其他异常）
	Msg  string   `json:"msg"`
	Data []string `json:"data"` //"13783393028;1", [0：非黑名单；1：黑名单；2：未知]
}

func (h apiStoreHandle) parseResponse(resp []string) ([]string, bool) {

	returnRes := []string{}
	haveBlack := false
	//"13783393028;1",
	for _, res := range resp {
		result := strings.Split(res, ";")
		if len(result) == 2 {
			//[0：非黑名单；1：黑名单；2：未知]
			if result[1] == "1" {
				returnRes = append(returnRes, result[0])
			}
			haveBlack = true
		}
	}
	return returnRes, haveBlack
}

func (h apiStoreHandle) Verify(mobiles []string, busineseType string) ([]string, bool, error) {

	url := "/api/detection/blacklist?"
	//其中apiKey即平台提供的apiKey，token即平台提供的token
	args := fmt.Sprintf("apiKey=%s&token=%s", h.apiKey, h.token)

	riskNums := ""
	for _, mobile := range mobiles {
		riskNums += "," + mobile
	}

	uri := h.url + url + args + "&" + "phone=" + riskNums[1:]
	logs.Debug("请求uri:%v ", uri)
	result := respInfo{}
	resp, err := httplib.Post(uri).
		Header("Content-Type", "application/x-www-form-urlencoded").
		Debug(true).
		SetUserAgent("qingmayun").
		SetTimeout(2*time.Second, 5*time.Second).
		Bytes()
	if err != nil {
		logs.Error("请求失败:%v,请求内容:%v,", err, riskNums[1:])
		return mobiles, false, fmt.Errorf("eReq")
	}

	logs.Debug("请求成功,返回内容:%v,", string(resp))
	err = json.Unmarshal(resp, &result)
	if err != nil {
		logs.Error("请求内容:%v,结果解析失败:%v,返回内容:%v", riskNums[1:], err, string(resp))
		return mobiles, false, fmt.Errorf("eReq2")
	}

	if result.Code != 0 {
		return mobiles, false, fmt.Errorf("eResp")
	}

	//logs.Debug("[%v]收到结果:%v ", mobiles, len(result.Data))
	risckNum, haveBlack := h.parseResponse(result.Data)
	return risckNum, haveBlack, nil
}
