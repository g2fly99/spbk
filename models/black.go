package models

import (
	"time"

	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	_ "github.com/go-sql-driver/mysql"
)

const gChannelMaxLen = 10000

type black struct {
	mobile       string
	busineseType string
}

var gBlackListChan chan black

type IpccBlackList struct {
	Id         int64
	Mobile     string
	Voice      string    `orm:"description(产品类型);null"`
	Sms        string    `orm:"description(产品类型);null"`
	CreateTime time.Time `orm:"description(添加时间);auto_now_add;type(datetime)"`
	ReadTime   time.Time `orm:"description(导出时间);null"`
}

func AddBLackMobile(mobile string) {

	if len(gBlackListChan) < gChannelMaxLen {
		a := black{
			mobile:       mobile,
			busineseType: "voice",
		}
		gBlackListChan <- a
	} else {
		logs.Error("mobile:%v ,drop .channel is full.", mobile)
	}
}

func AddBLackMobileWithType(mobile, busineseType string) {

	if len(gBlackListChan) < gChannelMaxLen {
		a := black{
			mobile:       mobile,
			busineseType: busineseType,
		}
		gBlackListChan <- a
	} else {
		logs.Error("mobile:%v ,drop .channel is full.", mobile)
	}
}

func DeleteBLackMobile(mobile string) error {
	delInfo := &IpccBlackList{
		Mobile: mobile,
	}
	o := orm.NewOrm()
	_, err := o.Delete(delInfo, "Mobile")
	return err
}

func GetBlackList(number, offset int) ([]*IpccBlackList, error) {

	var result []*IpccBlackList

	o := orm.NewOrm()
	_, err := o.QueryTable("IpccBlackList").
		Filter("CreateTime__gte", time.Now().Format("2006-01-02")).
		Filter("Sms", "true").
		Offset(offset).
		Limit(number).
		All(&result)

	if err != nil {
		return nil, err
	}
	return result, nil
}

func dbProcessRoutine() {

	blaskList := make([]IpccBlackList, 0, 0)
	timeOutChan := make(chan bool, 5)
	time.AfterFunc(50*time.Second, func() { timeOutChan <- true })

	logs.Info("数据库插入任务启动....")
	for {
		select {

		case <-timeOutChan:

			if len(blaskList) > 0 {

				o := orm.NewOrm()
				_, err := o.InsertMulti(len(blaskList), blaskList)
				if err != nil {
					logs.Error("number:%v 新增黑名单失败:%v", blaskList, err)
				}
				blaskList = make([]IpccBlackList, 0, 0)
			}

			time.AfterFunc(50*time.Second, func() { timeOutChan <- true })

		case info := <-gBlackListChan:

			newBlack := IpccBlackList{
				Mobile:     info.mobile,
				CreateTime: time.Now(),
				Voice:      "",
				Sms:        "",
			}

			switch info.busineseType {
			case "voice":
				newBlack.Voice = "true"
			case "sms":
				newBlack.Sms = "true"
			}

			blaskList = append(blaskList, newBlack)
			if len(blaskList) > 100 {
				o := orm.NewOrm()
				_, err := o.InsertMulti(len(blaskList), blaskList)
				if err != nil {
					logs.Error("number:%v 新增黑名单失败:%v", blaskList, err)
				}
				blaskList = make([]IpccBlackList, 0, 0)
			}
		}
	}
}

func RegisterBlackInfo() {

	orm.RegisterModel(new(IpccBlackList))
	gBlackListChan = make(chan black, gChannelMaxLen)
	go dbProcessRoutine()
}
