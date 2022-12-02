package controllers

import (
	"encoding/json"
	"fmt"
	"spbk/black"
	"spbk/etc"
	"spbk/models"
	"spbk/statics"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
)

// Operations about black
type BlackController struct {
	beego.Controller
}

func (this *BlackController) URLMapping() {

	this.Mapping("add", this.Add)
	this.Mapping("delete", this.Delete)
	this.Mapping("verify", this.SmsVerifys)
	this.Mapping("config", this.Config)
	this.Mapping("statics", this.Statics)
	this.Mapping("detectblack", this.DetectBlack)
}

// @router /detectblack [post]
func (this *BlackController) DetectBlack() {
	logs.Debug("收到新的请求:%v", string(this.Ctx.Input.RequestBody))

	type blackInfo struct {
		Mobile []string `json:"mobile"`
	}

	var req blackInfo
	err := json.Unmarshal(this.Ctx.Input.RequestBody, &req)
	if err != nil {
		logs.Debug("json解析失败:%v", err)
		this.Data["json"] = "json错误"
		this.ServeJSON()
		return
	}

	res := map[string]string{}
	for _, mobile := range req.Mobile {
		res[mobile] = "安全"
	}

	result := black.DetectBlack(req.Mobile)
	for number, value := range result.Filter {
		switch value {
		case "1":
			res[number] = "高危"
		case "2":
			res[number] = "中危"
		case "3":
			res[number] = "低危"
		}

	}

	this.Data["json"] = res
	this.ServeJSON()

}

// @router /verifyo [post]
func (this *BlackController) Verify() {

	logs.Debug("收到新的请求:%v", string(this.Ctx.Input.RequestBody))
	type blackInfo struct {
		BusinesesType string `json:"businesesType"`
		Mobile        []string
	}
	var req blackInfo
	err := json.Unmarshal(this.Ctx.Input.RequestBody, &req)
	if err != nil {
		logs.Error("请求解析json失败:%v", err)
		this.Data["json"] = "json错误"
		this.ServeJSON()
	}

	if len(req.BusinesesType) == 0 {
		req.BusinesesType = "voice"
	}
	result := make(map[string]bool)
	for _, mobile := range req.Mobile {
		isBlack, _ := black.VerifyWithType(mobile, req.BusinesesType)
		result[mobile] = isBlack
	}

	this.Data["json"] = result
	this.ServeJSON()
}

// @router /smsverify [post]
func (this *BlackController) SmsVerify() {

	logs.Debug("收到新请求：%v.", string(this.Ctx.Input.RequestBody))

	type dataInfo struct {
		Score int  `json:"score"`
		Hit   bool `json:"hit"`
	}

	type response struct {
		Status int      `json:"status"`
		Msg    string   `json:"msg"`
		Data   dataInfo `json:"data"`
	}

	result := response{
		Status: 0,
		Msg:    "Success",
		Data: dataInfo{
			Score: 99,
			Hit:   true,
		},
	}

	hashTaype, _ := this.GetInt("hash_type", 0)
	accessId := this.GetString("mch_id")
	if hashTaype != 1 || accessId != "IWmfSOq9" { //|| accessId != etc.Conf.BlackSv.Sms.AccessKey {
		result.Status = 110
		result.Msg = "服务未激活或未购买"
		this.Data["json"] = result
		this.ServeJSON()

		logs.Error("sms校验失败,hashTaype[%v]; accessId[%v];confaccess[%v] ",
			hashTaype, accessId, etc.Conf.BlackSv.Sms.AccessKey)
		return
	}

	//添加统计请求
	statics.AddSmsRqStatics(1)
	number := this.GetString("receive_number")
	isBlack := false
	var err error

	if etc.Conf.Api.Start {
		isBlack, err = black.AppStoreSmsVerify(number, "sms")
	} else {
		isBlack, err = black.VerifyWithTypeByIndex(number, "sms")
	}

	if err != nil {
		result.Data.Hit = false
		result.Data.Score = 0
		this.Data["json"] = result
		this.ServeJSON()
		logs.Debug("[%v]sms超时返回结果:%v ,to [%v]", number, result, this.Ctx.Request.RemoteAddr)
		return
	}

	if isBlack {
		result.Data.Score = 99
	} else {
		result.Data.Score = 0
	}

	logs.Debug("[%v]sms返回结果:%v,to [%v],", number, result, this.Ctx.Request.RemoteAddr)
	this.Data["json"] = result
	this.ServeJSON()

}

type dataInfo struct {
	Mobile string `json:"mobile"`
	Score  int    `json:"score"`
	Hit    bool   `json:"hit"`
}

// @router /verify [post]
func (this *BlackController) SmsVerifys() {

	logs.Debug("收到新请求：%v.", string(this.Ctx.Input.RequestBody))

	type response struct {
		Status int        `json:"status"`
		Msg    string     `json:"msg"`
		Data   []dataInfo `json:"data"`
	}

	result := response{
		Status: 0,
		Msg:    "Success",
	}

	hashTaype, _ := this.GetInt("hash_type", 0)
	accessId := this.GetString("mch_id")
	if hashTaype != 1 || accessId != etc.Conf.BlackSv.Sms.AccessKey {
		result.Status = 110
		result.Msg = "服务未激活或未购买"
		this.Data["json"] = result
		this.ServeJSON()
		return
	}

	numbers := this.GetStrings("receive_number")
	//添加统计请求
	statics.AddSmsRqStatics(uint64(len(numbers)))
	risknums, isBlack, err := black.VerifyIsBlacksWithBusineseType(numbers, "sms")
	if err != nil || false == isBlack {
		res := check(numbers, nil)
		result.Data = res

		result.Status = 110
		result.Msg = "服务未激活或未购买"
		this.Data["json"] = result
		this.ServeJSON()
		return
	}

	res := check(numbers, risknums)
	result.Data = res
	logs.Debug("sms返回结果:%v,to [%v]", result.Data, this.Ctx.Request.RemoteAddr)
	this.Data["json"] = result
	this.ServeJSON()

}

func check(request, risk []string) []dataInfo {

	result := []dataInfo{}
	reqMap := make(map[string]bool)

	for _, number := range request {
		reqMap[number] = true
	}

	logs.Debug("黑名单数量:[%v]", len(risk))
	for _, number := range risk {
		delete(reqMap, number)
		black.SaveToRedisWithType("", number, "sms", true)
		models.AddBLackMobileWithType(number, "sms")

		res := dataInfo{
			Mobile: number,
			Hit:    true,
			Score:  99,
		}
		result = append(result, res)
	}

	logs.Debug("安全名单数量:[%v]", len(reqMap))
	for number, _ := range reqMap {

		res := dataInfo{
			Mobile: number,
			Hit:    true,
			Score:  0,
		}
		result = append(result, res)
	}

	return result

}

// @router /add [post]
func (this *BlackController) Add() {

	type blackInfo struct {
		AccessKey string   `json:"accessKey"`
		Mobile    []string `json:"mobile"`
	}

	type Resp struct {
		Code    int
		Message string
	}

	var req blackInfo
	err := json.Unmarshal(this.Ctx.Input.RequestBody, &req)
	if err != nil {
		this.Data["json"] = Resp{403, fmt.Sprintf("参数错误:%v", err.Error())}
		this.ServeJSON()
		return
	}

	if req.AccessKey != etc.Conf.Base.AccessKey {
		this.Data["json"] = Resp{404, "accessKey非法"}
		this.ServeJSON()
		return
	}

	for _, mobile := range req.Mobile {
		models.AddBLackMobile(mobile)
		black.SaveToRedis(mobile, true)
	}

	this.Data["json"] = Resp{0, "success"}
	this.ServeJSON()
}

// @router /delete [post]
func (this *BlackController) Delete() {

	type blackInfo struct {
		Mobile []string
	}
	var req blackInfo
	json.Unmarshal(this.Ctx.Input.RequestBody, &req)

	result := make(map[string]string)
	for _, mobile := range req.Mobile {

		err := black.Delete(mobile)
		if err != nil {
			result[mobile] = err.Error()
		} else {
			result[mobile] = "删除成功"
		}
	}

	this.Data["json"] = result
	this.ServeJSON()
}

// @router /config [post]
func (this *BlackController) Config() {

	logs.Debug("收到新请求：%v.", string(this.Ctx.Input.RequestBody))

	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
	}

	type param struct {
		Packnum int `json:"packnum"`
		Delay   int `json:"delay"`
	}

	result := response{
		Status: 0,
		Msg:    "Success",
	}

	var args param

	err := json.Unmarshal(this.Ctx.Input.RequestBody, &args)
	if err != nil {
		logs.Error("解析错误:%v,data:%v", err, string(this.Ctx.Input.RequestBody))
	}

	black.Config(args.Packnum, args.Delay)

	this.Data["json"] = result
	this.ServeJSON()

}

// @Title 统计信息
// @router /statics [get]
func (this *BlackController) Statics() {

	type response struct {
		Status int    `json:"status"`
		Msg    string `json:"msg"`
		Data   interface{}
	}

	s := statics.Statics()
	result := response{
		Status: 0,
		Msg:    "单位为条/秒;total单位为 条/天",
		Data:   s,
	}

	this.Data["json"] = result
	this.ServeJSON()

}
