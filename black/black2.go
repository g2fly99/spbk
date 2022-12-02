package black

import (
	"crypto/md5"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"spbk/models"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
)

type request struct {
	SpId      string   `json:"spId"`
	Passwd    string   `json:"passwd"`
	Phones    []string `json:"phones"`
	Timestamp string   `json:"timestamp"`
	Sign      string   `json:"sign"`
}

func HashStringForApp(text string) string {

	signByte := []byte(text)
	hash := md5.New()
	hash.Write(signByte)
	return hex.EncodeToString(hash.Sum(nil))
}

func signForApp(apId, paasswd, timeStamp string) string {

	pwd := HashString(paasswd)
	//签名， (spId+passwd+ timestamp)进行 base64 编码 后，再生成 MD5  转 HEX 小写字母
	encoded := base64.StdEncoding.EncodeToString([]byte(apId + pwd + timeStamp))
	return HashStringForApp(encoded)
}

type Response struct {
	Status int               `json:"status"` //0 表示成功， 其他代表失败， 详见代码说明
	Msg    string            `json:"msg"`    //msg	string	文字说明， OK 代表发送成功，其他见代码数码
	Filter map[string]string `json:"filter"` //1：高危、 2：中危、 3：低危； 不在filter中为安全号码
}

func testBlack() {

	newReq := request{
		SpId:   "MzIRbm",
		Passwd: "dd40ac0a900cef6df9d46b6457c56776",
	}

	blackChan := make(chan ([]*models.IpccBlackList), 1000)
	pageSize := 500
	for i := 0; i < 10; i++ {

		black, err := models.GetBlackList(pageSize, pageSize*i)
		if err != nil {
			logs.Error("读取数据失败:%v", err)
			return
		}

		if len(black) > 0 {
			blackChan <- black
			logs.Debug("已取批次:%v", len(blackChan))
		} else {
			logs.Debug("没有新的黑名单,当前序号:%v", i)
			break
		}
	}

	for {
		black := <-blackChan
		go func() {
			reqPhones := make([]string, 0, 0)
			for _, phone := range black {
				reqPhones = append(reqPhones, phone.Mobile)
			}
			newReq.Phones = reqPhones

			start := time.Now()
			newReq.Timestamp = fmt.Sprintf("%d", start.UnixNano()/1e6)
			newReq.Sign = signForApp(newReq.SpId, newReq.Passwd, newReq.Timestamp)

			req, err := json.Marshal(newReq)
			if err != nil {
				logs.Error("请求编码失败:%v", err)
				return
			}

			logs.Debug("请求校验:签名[%v],时间戳[%v],账号[%v],密码[%v]"+
				"号码数量[%v]", newReq.Sign, newReq.Timestamp,
				newReq.SpId, newReq.Passwd, len(newReq.Phones))

			resp, err := httplib.Post("http://43.138.10.238:2509/check").
				Header("Content-Type", "application/json").
				Debug(true).
				SetUserAgent("qingmayun").
				Body(req).
				SetTimeout(5*time.Second, 5*time.Second).
				Bytes()
			if err != nil {
				logs.Error("请求失败:%v,请求内容:%v,", err, string(req))
				return
			}

			result := Response{}
			err = json.Unmarshal(resp, &result)
			if err != nil {
				logs.Error("请求内容:%v,结果解析失败:%v,返回内容:%v", string(req), err, string(resp))
			} else {
				logs.Debug("收到结果:%v ,耗时:%v", result.Filter, time.Since(start))
			}
		}()
	}
}

func DetectBlack(phones []string) (result Response) {

	if len(phones) == 0 {
		logs.Warn("无数据")
		result.Msg = "无数据"
		return
	}

	newReq := request{
		SpId:   "MzIRbm",
		Passwd: "dd40ac0a900cef6df9d46b6457c56776",
	}

	newReq.Phones = phones

	start := time.Now()
	newReq.Timestamp = fmt.Sprintf("%d", start.UnixNano()/1e6)
	newReq.Sign = signForApp(newReq.SpId, newReq.Passwd, newReq.Timestamp)

	req, err := json.Marshal(newReq)
	if err != nil {
		logs.Error("请求编码失败:%v", err)
		return
	}

	logs.Debug("请求校验:签名[%v],时间戳[%v],账号[%v],密码[%v]"+
		"号码数量[%v]", newReq.Sign, newReq.Timestamp,
		newReq.SpId, newReq.Passwd, len(newReq.Phones))

	resp, err := httplib.Post("http://43.138.10.238:2509/check").
		Header("Content-Type", "application/json").
		Debug(true).
		SetUserAgent("qingmayun").
		Body(req).
		SetTimeout(5*time.Second, 5*time.Second).
		Bytes()
	if err != nil {
		logs.Error("请求失败:%v,请求内容:%v,", err, string(req))
		result.Msg = err.Error()
		return
	}

	err = json.Unmarshal(resp, &result)
	if err != nil {
		logs.Error("请求内容:%v,结果解析失败:%v,返回内容:%v", string(req), err, string(resp))
	} else {
		logs.Debug("收到结果:%v ,耗时:%v", result.Filter, time.Since(start))
	}

	return
}
