package statics

import (
	"github.com/beego/beego/v2/core/logs"
	"spbk/models"
	"time"
)

type statics_info struct {
	dataType string
	number   uint64
	blacks   uint64
}

type StaticsCountInfo struct {
	Request   uint64 `json:"request"`
	Response  uint64 `json:"response"`
	Cache     uint64 `json:"cache"`
	Black     uint64 `json:"black"`
	TimeOut   uint64 `json:"timeOut"`
	TotalReq  uint64 `json:"total_req"`
	TotalResp uint64 `json:"total_resp"`
}

const (
	statics_type_sms   = "sms"
	statics_type_voice = "voice"
)

var (
	gCountChan    chan statics_info
	gReqCountChan chan statics_info
	gCacheChan    chan statics_info
	gTimeOutChan  chan statics_info
	gSmsStatics   StaticsCountInfo
	gVoiceStatics StaticsCountInfo
)

func AddVoiceRqStatics(number uint64) {

	ns := statics_info{
		number:   number,
		dataType: statics_type_voice,
	}
	gReqCountChan <- ns
}

func AddSmsRqStatics(number uint64) {

	ns := statics_info{
		dataType: statics_type_sms,
		number:   number,
	}

	gReqCountChan <- ns
}

func AddResponeStatics(number, blacks uint64, dataType string) {

	ns := statics_info{
		number:   number,
		blacks:   blacks,
		dataType: dataType,
	}

	gCountChan <- ns
}

func AddCacheStatics(number uint64, dataType string) {

	ns := statics_info{
		number:   number,
		dataType: dataType,
	}

	gCacheChan <- ns
}

func AddTimeOutStatics(number uint64, dataType string) {

	ns := statics_info{
		number:   number,
		dataType: dataType,
	}

	gTimeOutChan <- ns
}

func staticsTask() {

	logs.Info("启动统计线程。。。。")
	printStopCount := 0
	printVoiceStopCount := 0
	timeOutChan := make(chan bool, 2)
	timeOutChan <- true
	for {
		select {
		case stcs := <-gCountChan:
			if stcs.dataType == statics_type_sms {
				gSmsStatics.Response += stcs.number
				gSmsStatics.TotalResp += uint64(stcs.number)
				gSmsStatics.Black += stcs.blacks
			} else {
				gVoiceStatics.Response += stcs.number
				gVoiceStatics.Black += stcs.blacks
				gVoiceStatics.TotalResp += uint64(stcs.number)
			}
		case cache := <-gCacheChan:
			if cache.dataType == statics_type_sms {
				gSmsStatics.Cache += cache.number
			} else {
				gVoiceStatics.Cache += cache.number
			}

		case stcs := <-gReqCountChan:

			if stcs.dataType == statics_type_sms {
				gSmsStatics.Request += stcs.number
				gSmsStatics.TotalReq += uint64(stcs.number)
			} else {
				gVoiceStatics.Request += stcs.number
				gVoiceStatics.TotalReq += uint64(stcs.number)
			}
		case stcs := <-gTimeOutChan:
			if stcs.dataType == statics_type_sms {
				gSmsStatics.TimeOut += stcs.number
			} else {
				gVoiceStatics.TimeOut += stcs.number
			}
		case <-timeOutChan:

			//如果无数据，则不打印
			if gSmsStatics.Request > 0 || gSmsStatics.Response > 0 {
				printStopCount = 0
			} else {
				printStopCount++
			}

			if printStopCount < 2 {
				logs.Info("短信请求流速:[%4v]响应流速:[%4v]总请求数:[%8v]总响应数:[%8v]缓存数:[%6v]超时:[%4v]黑名单数:[%6v]",
					gSmsStatics.Request, gSmsStatics.Response, gSmsStatics.TotalReq, gSmsStatics.TotalResp,
					gSmsStatics.Cache, gSmsStatics.TimeOut, gSmsStatics.Black)
			}

			if gVoiceStatics.Request > 0 || gVoiceStatics.Response > 0 {
				printVoiceStopCount = 0
			} else {
				printVoiceStopCount++
			}

			if printVoiceStopCount < 2 {
				logs.Info("语音请求流速:[%4v]响应流速:[%4v]总请求数:[%8v]总响应数:[%8v]缓存数:[%6v]超时:[%4v]黑名单数:[%6v]",
					gVoiceStatics.Request, gVoiceStatics.Response, gVoiceStatics.TotalReq, gVoiceStatics.TotalResp,
					gVoiceStatics.Cache, gVoiceStatics.TimeOut, gVoiceStatics.Black)
			}

			gSmsStatics.Request = 0
			gSmsStatics.Response = 0

			gVoiceStatics.Request = 0
			gVoiceStatics.Response = 0

			time.AfterFunc(1*time.Second, func() { timeOutChan <- true })
			now := time.Now()
			if now.Hour() == 23 && now.Minute() == 59 && now.Second() < 59 && now.Second() > 55 {

				if gSmsStatics.TotalResp > 0 {
					//保存统计信息
					models.AddStaics(statics_type_sms, gSmsStatics.Cache, gSmsStatics.Black, gSmsStatics.TimeOut,
						gSmsStatics.TotalReq, gSmsStatics.TotalResp)

					logs.Info("统计信息 短信总请求数:[%8v]总响应数:[%8v]缓存数:[%6v]超时:[%4v]黑名单数:[%6v]",
						gSmsStatics.TotalReq, gSmsStatics.TotalResp,
						gSmsStatics.Cache, gSmsStatics.TimeOut, gSmsStatics.Black)
				}

				if gSmsStatics.TotalResp > 0 {

					models.AddStaics(statics_type_voice, gVoiceStatics.Cache, gVoiceStatics.Black, gVoiceStatics.TimeOut,
						gVoiceStatics.TotalReq, gVoiceStatics.TotalResp)

					logs.Info("统计信息 语音总请求数:[%8v]总响应数:[%8v]缓存数:[%6v]超时:[%4v]黑名单数:[%6v]",
						gVoiceStatics.TotalReq, gVoiceStatics.TotalResp, gVoiceStatics.Cache,
						gVoiceStatics.TimeOut, gVoiceStatics.Black)
				}

				gSmsStatics = StaticsCountInfo{}
				gVoiceStatics = StaticsCountInfo{}
			}
		}
	}
}

func Init() {

	gCountChan = make(chan statics_info, 1000)
	gReqCountChan = make(chan statics_info, 1000)
	gCacheChan = make(chan statics_info, 1000)
	gTimeOutChan = make(chan statics_info, 1000)

	go staticsTask()
}

func Statics() (data interface{}) {

	type staticData struct {
		Sms   StaticsCountInfo `json:"sms"`
		Voice StaticsCountInfo `json:"voice"`
	}
	res := staticData{
		Sms:   gSmsStatics,
		Voice: gVoiceStatics,
	}

	return res
}
