package models

import (
	"fmt"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	"spbk/etc"
	"time"
)

type IpccBlackStatics struct {
	Id          uint64
	ProductType string    `orm:"description(产品类型);null"`
	Cache       uint64    `orm:"description(缓存次数);null"`
	Black       uint64    `orm:"description(黑名单数);null"`
	TimeOut     uint64    `orm:"description(超时数);null"`
	TotalReq    uint64    `orm:"description(总请求);null"`
	TotalResp   uint64    `orm:"description(总响应);null"`
	Service     string    `orm:"description(服务名称);null"`
	CreateTime  time.Time `orm:"description(添加时间);auto_now_add;type(datetime)"`
}

func AddStaics(productType string, cache, blacks, timeout, request, response uint64) {

	statics := &IpccBlackStatics{
		ProductType: productType,
		Cache:       cache,
		Black:       blacks,
		TimeOut:     timeout,
		TotalReq:    request,
		TotalResp:   response,
		Service:     fmt.Sprintf("%d", etc.Conf.Base.Dsid),
	}

	o := orm.NewOrm()
	_, err := o.Insert(statics)
	if err != nil {
		logs.Error("新增统计信息失败:%v,%v", statics, err)
	}
}

func RegisterStatics() {

	orm.RegisterModel(new(IpccBlackStatics))
}
