package database

import (
	"fmt"
	"github.com/beego/beego/v2/client/orm"
	"github.com/beego/beego/v2/core/logs"
	_ "github.com/go-sql-driver/mysql"
)

// 初始化mysql
func InitMysql(user, pwd, ip, db string, port int, debug bool) error {

	orm.Debug = debug
	err := orm.RegisterDriver("mysql", orm.DRMySQL)
	if err != nil {

		return err
	}

	alias := "default"
	auth := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&loc=Local", user, pwd, ip, port, db)

	err = orm.RegisterDataBase(alias, "mysql", auth)
	if err != nil {
		logs.Error("register dataBase failed:%v,auth:%v", err, auth)
		return err
	}

	err = orm.RunSyncdb(alias, false, true)
	if err != nil {
		logs.Error("RunSyncdb failed:%v", err)
		return err
	}

	logs.Debug("数据库连接成功....")
	return nil
}
