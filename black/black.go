package black

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"spbk/etc"
	"spbk/models"
	"spbk/statics"
	"time"

	"github.com/beego/beego/v2/client/httplib"
	"github.com/beego/beego/v2/core/logs"
)

var (
	gServConf    etc.BlackServ
	gChannelcode int
	gAccessKey   string
	gHostUrl     string

	gPacketLen int           = minRequestNumber
	gTimerSend time.Duration = 100 * time.Millisecond
)

const (
	minRequestNumber        = 50
	resultCodeSuccSafe      = 1
	resultCodeSuccDanger    = 2
	resultCodeErrParam      = 10
	resultCodeErrSign       = 11
	resultCodeErrLimit      = 12
	resultCodeErrNeedCharge = 13
	resultCodeErrLimitDay   = 14
	resultCodeErrIvalidIp   = 15

	mobile_len = 11

	businese_type_voice = "voice"
	businese_type_sms   = "sms"
)

func Config(packNum, delay int) {

	gPacketLen = packNum
	if gPacketLen <= 0 {
		gPacketLen = 10
	}

	if delay <= 0 {
		delay = 100
	}

	gTimerSend = time.Duration(delay) * time.Millisecond
	logs.Debug("设置 打包数量：%v,超时:%v ", gPacketLen, gTimerSend)
}

type rqInfo struct {
	mobile       string
	busineseType string
	result       chan bool
}

var gRequestChan chan *rqInfo

type blackArgs struct {
	ChannelCode int      `json:"channelCode"` //渠道编码，由业务支撑平台提供
	TimeStamp   int64    `json:"tm"`          //是 unix 时间戳，13 位长度
	Mobiles     []string `json:"mobiles"`     //是 需校验数组清单，最长 1000
	Sign        string   `json:"sign"`        //签名，详细算法参考示例
	Mold        int      `json:"mold"`        //默认 1
}

type blackResp struct {
	Code int      `json:"code"`  //响应码，具体参考文末说明
	Msg  string   `json:"msg"`   //简单描述
	Data []string `json:"datas"` //风险数据数组
}

func HashString(text string) string {

	signByte := []byte(text)
	hash := md5.New()
	hash.Write(signByte)
	return hex.EncodeToString(hash.Sum(nil))
}

func sign(args blackArgs) string {

	signStr := fmt.Sprintf("%d%d%s", args.ChannelCode, args.TimeStamp, gAccessKey)

	for _, number := range args.Mobiles {
		signStr += number
	}

	return HashString(signStr)
}

func signWithAccessKey(args blackArgs, acceessKey string) string {

	signStr := fmt.Sprintf("%d%d%s", args.ChannelCode, args.TimeStamp, acceessKey)

	for _, number := range args.Mobiles {
		signStr += number
	}

	return HashString(signStr)
}

func Init(blackServer etc.BlackServ, redisAddr, pwd string) {

	redisInit(redisAddr, pwd)

	if blackServer.MaxNumReq > minRequestNumber {
		gPacketLen = blackServer.MaxNumReq
	}

	logs.Info("每次请求打包大小:%v", gPacketLen)

	if len(blackServer.Url) > 0 {

		gServConf = blackServer
		gHostUrl = blackServer.Url

		gAccessKey = blackServer.Voice.AccessKey
		gChannelcode = blackServer.Voice.AccessId
		logs.Debug("初始化黑名单信息:%v....,", blackServer)

		gRequestChan = make(chan *rqInfo, 10000)

		go requestResultTask()
	} else {
		logs.Warn("url 为空，黑名单未启动:%v", blackServer)
	}
}

func Delete(mobile string) error {

	if len(mobile) != mobile_len {
		return fmt.Errorf("ErrNum")
	}

	err := cacheDelete(mobile)
	if err != nil {
		logs.Error("[%v] 从缓存中删除失败", mobile)
	}
	err = models.DeleteBLackMobile(mobile)
	return err
}

func Verify(number string) (bool, error) {

	mobile := number
	if len(number) > mobile_len {
		mobile = number[len(number)-mobile_len:]
	}

	isBlack, hitten := CacheCheck(mobile)
	if hitten {
		return isBlack, nil
	}

	reqTime := time.Now()
	isBlack, err := verifyIsBlackFromRemote(mobile)
	if err != nil {
		logs.Error("从远端获取[%v]名单报错:%v", mobile, err)
		return false, err
	} else {
		logs.Debug("[%v] 返回结果，黑名单?:%v,耗时:%v .", mobile, isBlack, time.Since(reqTime))
		if isBlack {
			models.AddBLackMobile(mobile)
		}
	}

	SaveToRedis(mobile, isBlack)
	return isBlack, err
}

func VerifyWithTypeByIndex(number, busineseType string) (bool, error) {

	nr := &rqInfo{
		mobile:       number,
		busineseType: busineseType,
		result:       make(chan bool, 20),
	}

	gRequestChan <- nr

	timeOutChan := make(chan bool, 1)
	send := true
	time.AfterFunc(10*time.Second, func() {
		//超时未收到消息，返回失败
		if send {
			timeOutChan <- true
		}
	})

	for {
		select {
		case result := <-nr.result:
			send = false
			return result, nil
		case <-timeOutChan:
			logs.Debug("[%v]响应超时", number)
			statics.AddTimeOutStatics(1, busineseType)
			return true, errors.New("timeout")
		}
	}
}

func requestResultTask() {

	logs.Info("启动打包并发任务。。。。")
	reqNumber := []*rqInfo{}
	reqVoiceNumber := []*rqInfo{}
	timeOutChan := make(chan bool, 2)
	smsTimer := time.Now()
	voiceTimer := time.Now()
	timeOutChan <- true

	for {

		select {
		case req := <-gRequestChan:

			_, isBlack, hitten := CacheCheckWithType(req.mobile, req.busineseType)
			if hitten {
				logs.Debug("[%v]从Redis获取结果:%v", req.mobile, isBlack)
				req.result <- isBlack
				statics.AddCacheStatics(1, req.busineseType)
				continue
			}

			//logs.Debug("新的校验请求:%v", req)
			if req.busineseType == businese_type_sms {

				reqNumber = append(reqNumber, req)
				//每10个请求一次
				if len(reqNumber) == gPacketLen {
					//logs.Debug("启动请求:%v", reqNumber)
					go VerifyWithTypePacket(reqNumber, businese_type_sms)
					smsTimer = time.Now()
					reqNumber = []*rqInfo{}
				}
			} else {
				reqVoiceNumber = append(reqVoiceNumber, req)
				//logs.Debug("[%v] 加入到待发送队列,长度：[%v]", req.mobile, len(reqVoiceNumber))
				if len(reqVoiceNumber) == gPacketLen {
					//logs.Debug("voice 启动请求:%v", reqNumber)
					go VerifyWithTypePacket(reqVoiceNumber, businese_type_voice)
					voiceTimer = time.Now()
					reqVoiceNumber = []*rqInfo{}
				}
			}
		case <-timeOutChan:

			if len(reqNumber) > 0 && time.Since(smsTimer) > gTimerSend {
				logs.Debug("短信超时，预请求数:%v,上次发送时间:%v", len(reqNumber), smsTimer.Format("15:04:05.000"))
				go VerifyWithTypePacket(reqNumber, businese_type_sms)
				reqNumber = []*rqInfo{}
				smsTimer = time.Now()
			}

			//logs.Debug("语音超时校验，预请求数:%v,上次发送时间:%v", len(reqVoiceNumber), voiceTimer.Format("15:04:05"))
			if len(reqVoiceNumber) > 0 && time.Since(voiceTimer) > gTimerSend {
				logs.Debug("语音超时，预请求数:%v,上次发送时间:%v", len(reqVoiceNumber), voiceTimer.Format("15:04:05.000"))
				go VerifyWithTypePacket(reqVoiceNumber, businese_type_voice)
				reqVoiceNumber = []*rqInfo{}
				voiceTimer = time.Now()
			}

			time.AfterFunc(gTimerSend, func() { timeOutChan <- true })
		}

	}
}

func sendResult(numbers []string, requets map[string]*rqInfo, busineseType string) {

	//logs.Debug("黑名单数量:[%v]", len(numbers))
	for _, number := range numbers {

		r := requets[number]
		if r != nil {
			r.result <- true
			delete(requets, number)
			models.AddBLackMobileWithType(number, r.busineseType)
			SaveToRedisWithType("", number, r.busineseType, true)
		} else {
			logs.Error("[%v]号码不存在", number)
		}
	}

	//logs.Debug("安全名单数量:[%v]", len(requets))
	if busineseType == "sms" {
		for _, r := range requets {
			r.result <- false
			SaveToRedisWithType("", r.mobile, r.busineseType, true)
		}

	} else {
		for _, r := range requets {
			//time.Sleep(10 * time.Millisecond)
			r.result <- false
			SaveToRedisWithType("", r.mobile, r.busineseType, true)
		}
	}
}

func VerifyWithTypePacket(args []*rqInfo, busineseType string) {

	reqNums := []string{}
	reqMap := make(map[string]*rqInfo)

	for _, r := range args {
		reqNums = append(reqNums, r.mobile)
		reqMap[r.mobile] = r
	}
	reqCount := len(reqNums)
	//logs.Debug("开始请求:%v", len(args))
	reqTime := time.Now()
	risknums, isBlack, err := VerifyIsBlacksWithBusineseType(reqNums, busineseType)
	if err != nil {
		logs.Error("从远端获取名单报错:%v,耗时:%v .", err, time.Since(reqTime))
		return
	} else {
		risks := len(risknums)
		logs.Debug("[%v]黑名单数量 [%v/%v] 返回结果，黑名单?:%v,耗时:%v.", busineseType, risks, reqCount, isBlack, time.Since(reqTime))
		//添加统计请求
		statics.AddResponeStatics(uint64(reqCount), uint64(risks), busineseType)
	}
	sendResult(risknums, reqMap, busineseType)
}

func VerifyWithType(number, busineseType string) (bool, error) {

	mobile := number
	if len(number) > mobile_len {
		mobile = number[len(number)-mobile_len:]
	}

	cache, isBlack, hitten := CacheCheckWithType(mobile, busineseType)
	if hitten {
		logs.Debug("[%v]从Redis获取结果:%v", number, isBlack)
		return isBlack, nil
	}

	reqTime := time.Now()
	isBlack, err := verifyIsBlackWithBusineseType(mobile, busineseType)
	if err != nil {
		logs.Error("[%v]从远端获取名单报错:%v,耗时:%v.", mobile, err, time.Since(reqTime))
		return false, err
	} else {
		logs.Debug("[%v]返回结果，黑名单?:%v,耗时:%v.", mobile, isBlack, time.Since(reqTime))
		if isBlack {
			models.AddBLackMobileWithType(mobile, busineseType)
		}
	}

	SaveToRedisWithType(cache, mobile, busineseType, isBlack)
	return isBlack, err
}

func verifyIsBlackFromRemote(mobile string) (bool, error) {

	mobiles := []string{mobile}

	newVerify := blackArgs{
		ChannelCode: gChannelcode,
		TimeStamp:   time.Now().UnixNano() / 1e6, //毫秒
		Mobiles:     mobiles,
		Mold:        1,
	}

	newVerify.Sign = sign(newVerify)

	request, err := json.Marshal(newVerify)
	if err != nil {
		logs.Error("josn 编码失败:%v,内容:%v", err, newVerify)
		return false, fmt.Errorf("eSys")
	}

	result := blackResp{}
	resp, err := httplib.Post(gHostUrl).
		Header("Content-Type", "application/json").
		Debug(true).
		SetUserAgent("qingmayun").
		Body(request).
		SetTimeout(5*time.Second, 15*time.Second).
		Bytes()
	if err != nil {
		logs.Error("请求失败:%v,请求内容:%v,", err, string(request))
		return false, fmt.Errorf("eReq")
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		logs.Error("请求内容:%v,结果解析失败:%v,返回内容:%v", string(request), err, string(resp))
		return false, fmt.Errorf("eReq2")
	} else {
		logs.Debug("[%v]收到结果:%v ", mobile, string(resp))
	}

	switch result.Code {
	case resultCodeSuccSafe:

		return false, nil
	case resultCodeSuccDanger:
		if len(result.Data) == 0 {
			logs.Error("code:%v ,结果返回数据错误:%v,data:%v", result.Code, result.Msg, result.Data)
			return false, fmt.Errorf("eData")
		} else {
			return true, nil
		}
	case resultCodeErrParam:
		logs.Error("参数错误:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eParam")
	case resultCodeErrSign:
		logs.Error("签名错误:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eParam")
	case resultCodeErrLimitDay:
		logs.Error("请求受限:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eLimit")
	case resultCodeErrLimit:
		logs.Error("请求受限:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eLimit")
	case resultCodeErrNeedCharge:
		logs.Error("余额不足:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eCharge")
	default:
		logs.Error("结果返回失败:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eResp")
	}
}

func verifyIsBlackWithBusineseType(mobile, busineseType string) (bool, error) {

	mobiles := []string{mobile}

	newVerify := blackArgs{
		ChannelCode: gServConf.Voice.AccessId,
		TimeStamp:   time.Now().UnixNano() / 1e6, //毫秒
		Mobiles:     mobiles,
		Mold:        1,
	}

	accessKey := gServConf.Voice.AccessKey
	if busineseType == "sms" {
		newVerify.ChannelCode = gServConf.Sms.AccessId
		accessKey = gServConf.Sms.AccessKey
	}

	newVerify.Sign = signWithAccessKey(newVerify, accessKey)

	request, err := json.Marshal(newVerify)
	if err != nil {
		logs.Error("josn 编码失败:%v,内容:%v", err, newVerify)
		return false, fmt.Errorf("eSys")
	}
	logs.Debug("请求内容:%v", string(request))
	result := blackResp{}
	resp, err := httplib.Post(gHostUrl).
		Header("Content-Type", "application/json").
		Debug(true).
		SetUserAgent("qingmayun").
		Body(request).
		SetTimeout(5*time.Second, 5*time.Second).
		Bytes()
	if err != nil {
		logs.Error("请求失败:%v,请求内容:%v,", err, string(request))
		return false, fmt.Errorf("eReq")
	}
	err = json.Unmarshal(resp, &result)
	if err != nil {
		logs.Error("请求内容:%v,结果解析失败:%v,返回内容:%v", string(request), err, string(resp))
		return false, fmt.Errorf("eReq2")
	} else {
		logs.Debug("[%v]收到结果:%v ", mobile, string(resp))
	}

	switch result.Code {
	case resultCodeSuccSafe:

		return false, nil
	case resultCodeSuccDanger:
		if len(result.Data) == 0 {
			logs.Error("code:%v ,结果返回数据错误:%v,data:%v", result.Code, result.Msg, result.Data)
			return false, fmt.Errorf("eData")
		} else {
			return true, nil
		}
	case resultCodeErrParam:
		logs.Error("参数错误:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eParam")
	case resultCodeErrSign:
		logs.Error("签名错误:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eParam")
	case resultCodeErrLimitDay:
		logs.Error("请求受限:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eLimit")
	case resultCodeErrLimit:
		logs.Error("请求受限:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eLimit")
	case resultCodeErrNeedCharge:
		logs.Error("余额不足:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eCharge")
	default:
		logs.Error("结果返回失败:%v,失败信息:%v", result.Code, result.Msg)
		return false, fmt.Errorf("eResp")
	}
}

func VerifyIsBlacksWithBusineseType(mobiles []string, busineseType string) ([]string, bool, error) {

	riskNums := []string{}
	newVerify := blackArgs{
		ChannelCode: gServConf.Voice.AccessId,
		TimeStamp:   time.Now().UnixNano() / 1e6, //毫秒
		Mobiles:     mobiles,
		Mold:        1,
	}

	accessKey := gServConf.Voice.AccessKey
	if busineseType == "sms" {
		newVerify.ChannelCode = gServConf.Sms.AccessId
		accessKey = gServConf.Sms.AccessKey
	}

	newVerify.Sign = signWithAccessKey(newVerify, accessKey)

	request, err := json.Marshal(newVerify)
	if err != nil {
		logs.Error("josn 编码失败:%v,内容:%v", err, newVerify)
		return riskNums, false, fmt.Errorf("eSys")
	}
	logs.Debug("请求内容:%v", string(request))
	result := blackResp{}
	resp, err := httplib.Post(gHostUrl).
		Header("Content-Type", "application/json").
		Debug(true).
		SetUserAgent("qingmayun").
		Body(request).
		SetTimeout(2*time.Second, 5*time.Second).
		Bytes()
	if err != nil {
		logs.Error("请求失败:%v,请求内容:%v,", err, string(request))
		return riskNums, false, fmt.Errorf("eReq")
	}

	err = json.Unmarshal(resp, &result)
	if err != nil {
		logs.Error("请求内容:%v,结果解析失败:%v,返回内容:%v", string(request), err, string(resp))
		return riskNums, false, fmt.Errorf("eReq2")
	} else {
		logs.Debug("[%v]收到结果:%v ", mobiles, string(resp))
	}

	switch result.Code {
	case resultCodeSuccSafe:

		return result.Data, false, nil
	case resultCodeSuccDanger:
		if len(result.Data) == 0 {
			logs.Error("code:%v ,结果返回数据错误:%v,data:%v", result.Code, result.Msg, result.Data)
			return result.Data, false, fmt.Errorf("eData")
		} else {
			return result.Data, true, nil
		}
	case resultCodeErrParam:
		logs.Error("参数错误:%v,失败信息:%v", result.Code, result.Msg)
		return result.Data, false, fmt.Errorf("eParam")
	case resultCodeErrSign:
		logs.Error("签名错误:%v,失败信息:%v", result.Code, result.Msg)
		return result.Data, false, fmt.Errorf("eParam")
	case resultCodeErrLimitDay:
		logs.Error("请求受限:%v,失败信息:%v", result.Code, result.Msg)
		return result.Data, false, fmt.Errorf("eLimit")
	case resultCodeErrLimit:
		logs.Error("请求受限:%v,失败信息:%v", result.Code, result.Msg)
		return result.Data, false, fmt.Errorf("eLimit")
	case resultCodeErrNeedCharge:
		logs.Error("余额不足:%v,失败信息:%v", result.Code, result.Msg)
		return result.Data, false, fmt.Errorf("eCharge")
	default:
		logs.Error("结果返回失败:%v,失败信息:%v", result.Code, result.Msg)
		return result.Data, false, fmt.Errorf("eResp")
	}
}
