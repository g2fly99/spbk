package black

import (
	"fmt"
	"github.com/beego/beego/v2/core/logs"
	"spbk/statics"
	"time"
)

type BlackMethod interface {
	Verify(mobiles []string, busineseType string) ([]string, bool, error)
}

type Blackhandle struct {
	redisTrhead int //缓存获取线程
	taskChan    chan *rqInfo
	requestChan chan *rqInfo
	request     func(mobiles []string, busineseType string) ([]string, bool, error)
}

func NewHanlde(reqfunc func(mobiles []string, busineseType string) ([]string, bool, error),
	redisThreads int) *Blackhandle {

	if redisThreads == 0 {
		redisThreads = 5
	}

	handler := &Blackhandle{
		redisTrhead: redisThreads,
		taskChan:    make(chan *rqInfo, 10000),
		requestChan: make(chan *rqInfo, 10000),
		request:     reqfunc,
	}

	go handler.cacheCheckTask()
	go handler.requestResultTask()

	return handler
}

func (h *Blackhandle) VerifyWithTypePacket(args []*rqInfo, busineseType string) {

	reqNums := []string{}
	reqMap := make(map[string]*rqInfo)

	for _, r := range args {
		reqNums = append(reqNums, r.mobile)
		reqMap[r.mobile] = r
	}
	reqCount := len(reqNums)
	//logs.Debug("开始请求:%v", len(args))
	reqTime := time.Now()
	risknums, isBlack, err := h.request(reqNums, busineseType)
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

func (h *Blackhandle) cacheCheckTask() {

	logs.Debug("启动[%v]个缓存线程....", h.redisTrhead)
	for i := 0; i < h.redisTrhead; i++ {
		go func(taskId int) {
			timer := time.Now()
			for {
				req := <-h.taskChan
				timer = time.Now()
				_, isBlack, hitten := CacheCheckWithType(req.mobile, req.busineseType)
				//4s以内获取到结果，认为是OK的
				if time.Since(timer) < 4*time.Second {
					if hitten {
						logs.Debug("[%v]从Redis获取结果:%v", req.mobile, isBlack)
						req.result <- isBlack
						statics.AddCacheStatics(1, req.busineseType)
						continue
					}
					h.requestChan <- req
				} else {
					logs.Error("[%v] 从redis查询超时:%4v.", req.mobile, time.Since(timer))
				}
			}
		}(i)
	}
}

func (h *Blackhandle) requestResultTask() {

	logs.Info("启动打包并发任务。。。。")
	reqNumber := []*rqInfo{}
	reqVoiceNumber := []*rqInfo{}
	timeOutChan := make(chan bool, 2)
	smsTimer := time.Now()
	voiceTimer := time.Now()
	timeOutChan <- true

	for {

		select {
		case req := <-h.requestChan:

			//logs.Debug("新的校验请求:%v", req)
			if req.busineseType == businese_type_sms {

				reqNumber = append(reqNumber, req)
				//每10个请求一次
				if len(reqNumber) == gPacketLen {
					//logs.Debug("启动请求:%v", reqNumber)
					go h.VerifyWithTypePacket(reqNumber, businese_type_sms)
					smsTimer = time.Now()
					reqNumber = []*rqInfo{}
				}
			} else {
				reqVoiceNumber = append(reqVoiceNumber, req)
				//logs.Debug("[%v] 加入到待发送队列,长度：[%v]", req.mobile, len(reqVoiceNumber))
				if len(reqVoiceNumber) == 10 {
					//logs.Debug("voice 启动请求:%v", reqNumber)
					go h.VerifyWithTypePacket(reqVoiceNumber, businese_type_voice)
					voiceTimer = time.Now()
					reqVoiceNumber = []*rqInfo{}
				}
			}
		case <-timeOutChan:

			if len(reqNumber) > 0 && time.Since(smsTimer) > gTimerSend {
				logs.Debug("短信超时，预请求数:%v,上次发送时间:%v", len(reqNumber), smsTimer.Format("15:04:05.000"))
				go h.VerifyWithTypePacket(reqNumber, businese_type_sms)
				reqNumber = []*rqInfo{}
				smsTimer = time.Now()
			}

			//logs.Debug("语音超时校验，预请求数:%v,上次发送时间:%v", len(reqVoiceNumber), voiceTimer.Format("15:04:05"))
			if len(reqVoiceNumber) > 0 && time.Since(voiceTimer) > gTimerSend {
				logs.Debug("语音超时，预请求数:%v,上次发送时间:%v", len(reqVoiceNumber), voiceTimer.Format("15:04:05.000"))
				go h.VerifyWithTypePacket(reqVoiceNumber, businese_type_voice)
				reqVoiceNumber = []*rqInfo{}
				voiceTimer = time.Now()
			}

			time.AfterFunc(gTimerSend, func() { timeOutChan <- true })
		}
	}
}

func (h *Blackhandle) AddVerify(number, busineseType string) (bool, error) {

	nr := &rqInfo{
		mobile:       number,
		busineseType: busineseType,
		result:       make(chan bool, 20),
	}

	h.taskChan <- nr

	timeOutChan := make(chan bool, 1)
	send := true
	time.AfterFunc(10*time.Second, func() {
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
			logs.Debug("[%v] 响应超时.", number)
			statics.AddTimeOutStatics(1, busineseType)
			return true, fmt.Errorf("timeout")
		}
	}
}
