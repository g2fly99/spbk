package controllers

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
)

const (
	callback = "http://121.201.115.60:18184/chout/notify/CMCCgdyd/DeliveryInfoNotification/sip:10650515001089007@botplatform.rcs.chinamobile.com"
)

// Operations about Users
type M5gController struct {
	beego.Controller
}

func (this *M5gController) URLMapping() {
	this.Mapping("send", this.Send)
	this.Mapping("option", this.Option)
	this.Mapping("getlocation", this.GetIpLocation)
}

type response struct {
	Code      string `json:"code"`
	Msg       string `json:"msg"`
	MessageId string `json:"messageId"`
}

type option struct {
	SendSpeed int    `json:"send_speed"` //每秒发送速率
	Success   int    `json:"success"`    //成功比例
	Gurl      string `json:"url"`        //推送地址
}

type SendMessageT struct {
	XMLName xml.Name `xml:"outboundMessageRequest"`
	Phone   []string `xml:"destinationAddress"`
}

type deleveryResultT struct {
	phone []string
	msgId string
}

var gOption option
var resultChan chan *deleveryResultT

// @router /send [post]
func (this *M5gController) Send() {

	res := response{
		Code:      "00000",
		Msg:       "投递成功",
		MessageId: createNewMsgId(),
	}
	logs.Debug("获取变量:%v", this.Ctx.Input.Param(":splat"))

	if len(this.Ctx.Input.RequestBody) == 0 {
		res.Msg = "无数据"
		res.Code = "00001"
		res.MessageId = ""
		this.Data["json"] = res
		this.ServeJSON()

		logs.Error("body中无数据....")
		return
	}

	logs.Debug("收到新的请求:%v", string(this.Ctx.Input.RequestBody))
	msg := &SendMessageT{}

	err := xml.Unmarshal(this.Ctx.Input.RequestBody, msg)
	if err != nil {
		logs.Error("xml 解析失败:%v", err)
		res.Msg = "xml解析失败"
		this.Data["json"] = res
		this.ServeJSON()
		return
	}

	logs.Debug("解析成功,号码个数:%v", len(msg.Phone))
	this.Data["json"] = res
	this.ServeJSON()

	result := &deleveryResultT{
		phone: msg.Phone,
		msgId: res.MessageId,
	}
	resultChan <- result
	return
}

// @router /option [post]
func (this *M5gController) Option() {

	logs.Debug("收到新的请求:%v", string(this.Ctx.Input.RequestBody))
	err := json.Unmarshal(this.Ctx.Input.RequestBody, &gOption)
	if err != nil {
		logs.Error("配置失败：%v", err)
	} else {
		logs.Debug("新的配置：%v", gOption)
	}

	if gOption.SendSpeed == 0 {
		gOption.SendSpeed = 50
	}

	if gOption.Success == 0 {
		gOption.Success = 50
	}

	if len(gOption.Gurl) == 0 {
		gOption.Gurl = callback
	}

	this.Data["json"] = gOption
	this.ServeJSON()

}

func getIPaddreesLocation(ip string) (string, error) {

	type resultInfo struct {
		Message string `json:"msg"`
		Ret     int    `json:"ret"`
		Data    struct {
			City       string `json:"city"`
			Region     string `json:"region"`
			Contry     string `json:"country"`
			Country_id string `json:"country_id"`
			District   string `json:"district"`
		} `json:"data"`
	}

	start := time.Now()
	url := fmt.Sprintf("https://api01.aliyun.venuscn.com/ip?ip=%s", ip)
	resp, err := httplib.Get(url).
		Header("Authorization", "APPCODE e10ae64a863e4441b1f75b6c191bd415").
		Header("Content-Type", "application/json;charset=UTF-8").
		SetUserAgent("danmi").
		SetTimeout(2*time.Second, 3*time.Second).
		Retries(3).
		Bytes()

	if err != nil {
		logs.Error("IP 请求失败:%v", err)
		return "", errors.New(fmt.Sprintf("请求失败:%v", err.Error()))
	}

	res := resultInfo{}
	err = json.Unmarshal(resp, &res)
	if err != nil {
		logs.Error("收到请求结果:%v,错误：%v", string(resp), err)
		return "", errors.New(fmt.Sprintf("结果解析错误:%v,%v", err.Error(), string(resp)))
	}

	if res.Ret != 200 {
		logs.Error("请求[%v] 返回错误:%v", ip, res.Message)
		return res.Message, nil
	}

	logs.Debug("收到结果:%v", string(resp))
	//默认中国区取省，市
	area := res.Data.Region + "," + res.Data.City
	if res.Data.Country_id != "CN" || len(res.Data.City) == 0 {
		area = res.Data.Contry
		//国外，仅带省
		if len(res.Data.Region) > 0 {
			area += "," + res.Data.Region
		}
	} else if len(res.Data.District) > 0 {
		//如果有区，则把区带上
		area += "," + res.Data.District
	}

	logs.Debug("获取[%v] 结果:[%v], 耗时:%v", ip, area, time.Since(start))
	return area, nil
}

// @router /getlocation [get]
func (this *M5gController) GetIpLocation() {

	type result struct {
		Ip   string `json:"ci"`
		Area string `json:"cname"`
	}

	ip := strings.Split(this.Ctx.Request.RemoteAddr, ":")
	logs.Debug("收到新的请求,from:%v", ip)
	if len(ip) > 1 {

		area, err := getIPaddreesLocation(ip[0])
		res := &result{
			Ip:   ip[0],
			Area: area,
		}

		if err != nil {
			res.Area = err.Error()
		}
		this.Data["json"] = res
	} else {
		logs.Error("请求错误:%v ", ip)
		this.Data["json"] = &result{this.Ctx.Request.RemoteAddr, "IP地址错误"}
	}

	this.ServeJSON()

	return
}

/*
<?xml version="1.0" encoding="UTF-8"?>
<msg:deliveryInfoNotification xmlns:msg="urn:oma:xml:rest:netapi:messaging:1">
<deliveryInfo>
    <address>tel:+8619585550103</address>
    <messageId>5eae954c-42ca-4181-9ab4-9c0ef2e2ac55</messageId>
    <deliveryStatus>DeliveryImpossible</deliveryStatus>
    <description>SVC0002</description>
    <text>AO msg service capability invalid</text>
</deliveryInfo>
<link rel="OutboundMessageRequest"    href="https://example.com/exampleAPI/messaging/v1/outbound/sip%3A10086%40botplatform.rcs.chinamobile.com/requests/5eae954c-42ca-4181-9ab4-9c0ef2e2ac55"/>
</msg:deliveryInfoNotification>
*/

var gCount int64 = 0

func createNewMsgId() string {
	gCount++
	return fmt.Sprintf("%v", time.Now().UnixNano()+gCount)
}

func makeFaileXml(phone, msgId string) string {

	message := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
	message += "<msg:deliveryInfoNotification xmlns:msg=\"urn:oma:xml:rest:netapi:messaging:1\">"
	message += "<deliveryInfo>"
	message += "<address>tel:+86" + phone + "</address>"
	message += "<messageId>\"" + msgId + "</messageId>"
	message += "<deliveryStatus>DeliveryImpossible</deliveryStatus>"
	message += "<description>SVC0002</description>"
	message += "<text>AO msg service capability invalid</text>"
	message += "</deliveryInfo>"
	message += "<link rel=\"OutboundMessageRequest\"    href=\"https://example.com/exampleAPI/messaging/v1/outbound/sip%3A10086%40botplatform.rcs.chinamobile.com/requests/" + msgId + "\"/>"
	message += "</msg:deliveryInfoNotification>"

	return message
}

/*
<?xml version="1.0" encoding="UTF-8"?>
<msg:deliveryInfoNotification xmlns:msg="urn:oma:xml:rest:netapi:messaging:1">
<deliveryInfo>
    <address>tel:+8619585550103</address>
    <messageId>5eae954c-42ca-4181-9ab4-9c0ef2e2ac55</messageId>
    <deliveryStatus>DeliveredToTerminal</deliveryStatus>
<description>UP2</description>

</deliveryInfo>
<link rel="OutboundMessageRequest"    href="https://example.com/exampleAPI/messaging/v1/outbound/sip%3A10086%40botplatform.rcs.chinamobile.com/requests/5eae954c-42ca-4181-9ab4-9c0ef2e2ac55"/>
</msg:deliveryInfoNotification>
*/

func makeSuccessXml(phone, msgId string) string {

	message := "<?xml version=\"1.0\" encoding=\"UTF-8\"?>"
	message += "<msg:deliveryInfoNotification xmlns:msg=\"urn:oma:xml:rest:netapi:messaging:1\">"
	message += "<deliveryInfo>"
	message += "<address>tel:+86" + phone + "</address>"
	message += "<messageId>\"" + msgId + "</messageId>"
	message += "<deliveryStatus>DeliveredToTerminal</deliveryStatus>"
	message += "<description>UP2</description>"
	message += "<text>AO msg service capability invalid</text>"
	message += "</deliveryInfo>"
	message += "<link rel=\"OutboundMessageRequest\"    href=\"https://example.com/exampleAPI/messaging/v1/outbound/sip%3A10086%40botplatform.rcs.chinamobile.com/requests/" + msgId + "\"/>"
	message += "</msg:deliveryInfoNotification>"

	return message
}

func sendRoutine() {

	total := 0
	timeStart := time.Now()

	logs.Debug("模拟推送结果启动....")
	for {
		result := <-resultChan
		//logs.Debug(":%v", newResult)
		if timeStart.IsZero() {
			//开始计时
			timeStart = time.Now()
		}

		for _, phone := range result.phone {
			rand.Seed(time.Now().UnixNano())
			//随机生成100以内的正整数
			rate := rand.Intn(100)
			if rate > gOption.Success {

				go sendDelevery(false, phone, result.msgId)
			} else {

				go sendDelevery(true, phone, result.msgId)
			}
			total++

			//达到限速速率
			if total >= gOption.SendSpeed {

				logs.Debug("发送条数:%v,计时时间:%v,偏移:%v", total, timeStart.Format("15:04:05"), time.Since(timeStart))
				cost := time.Since(timeStart)
				//未到1秒钟则需要停止发送
				if cost <= time.Second {
					time.Sleep(time.Second - cost)
				}
				//重新开始计时
				timeStart = time.Now()
				total = 0
			}

		}

	}
}

func sendDelevery(success bool, phone, messageId string) {

	body := ""
	if success {
		logs.Debug("发送成功,%v", phone)
		body = makeSuccessXml(phone, messageId)
	} else {
		logs.Debug("发送失败,%v", phone)
		body = makeFaileXml(phone, messageId)
	}

	resp, err := httplib.Post(gOption.Gurl).
		Header("Content-Type", "application/json").
		SetUserAgent("qingmayun").
		SetTimeout(5*time.Second, 5*time.Second).
		Body(body).
		String()

	if err != nil {
		logs.Error("请求失败:%v，url:%v", err, gOption.Gurl)
		return
	}

	logs.Debug("receive:%v", (resp))

}

func M5gInit() {

	gOption.SendSpeed = 10
	gOption.Success = 50
	gOption.Gurl = callback
	resultChan = make(chan *deleveryResultT, 100000)

	go sendRoutine()
}
