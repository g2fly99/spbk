package main

import (
	"os"
	"spbk/black"
	"spbk/controllers"
	"spbk/database"
	"spbk/etc"
	"spbk/models"
	_ "spbk/routers"
	"spbk/sip_server"
	"spbk/statics"

	"github.com/beego/beego/v2/core/logs"
	beego "github.com/beego/beego/v2/server/web"
	"github.com/beego/beego/v2/server/web/filter/cors"
)

func main() {

	if beego.BConfig.RunMode == "dev" {
		beego.BConfig.WebConfig.DirectoryIndex = true
		beego.BConfig.WebConfig.StaticDir["/swagger"] = "swagger"
	} else {
		err := logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/spbk.log","maxdays":30}`)
		if err != nil {
			logs.Error("create file adapter err:%v", err)
			panic("log fail")
		}
	}

	pwd, _ := os.Getwd()
	logs.Debug("当前系统路径:%v", pwd)
	err := etc.InitConfig("conf/config.yml")
	if err != nil {
		logs.Error("配置文件加载失败: %v", err)
		os.Exit(1)
	}

	models.RegisterBlackInfo()
	models.RegisterStatics()
	//mysql初始化
	err = database.InitMysql(etc.Conf.Mysql.MysqlDBs.User,
		etc.Conf.Mysql.MysqlDBs.Password,
		etc.Conf.Mysql.MysqlDBs.IP,
		etc.Conf.Mysql.MysqlDBs.Db,
		etc.Conf.Mysql.MysqlDBs.Port,
		etc.Conf.Mysql.Debug)
	if err != nil {
		logs.Warn("mysql connect err:%v", err)
	}

	if etc.Conf.BlackSv.Voice.Start {
		sip_server.Init()
	}

	black.Init(etc.Conf.BlackSv, etc.Conf.Redis.IP, etc.Conf.Redis.Password)
	if etc.Conf.Api.Start {
		black.NewApiStoreHandle(etc.Conf.Api.AccessId, etc.Conf.Api.AccessKey, etc.Conf.Api.Url)
	}

	controllers.M5gInit()

	statics.Init()

	//InsertFilter是提供一个过滤函数
	beego.InsertFilter("*", beego.BeforeRouter, cors.Allow(&cors.Options{
		// 允许访问所有源
		AllowAllOrigins: true,
		// 可选参数"GET", "POST", "PUT", "DELETE", "OPTIONS" (*为所有)
		AllowMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		// 指的是允许的Header的种类
		AllowHeaders: []string{"Origin", "Authorization", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		// 公开的HTTP标头列表
		ExposeHeaders: []string{"Content-Length", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "Content-Type"},
		// 如果设置，则允许共享身份验证凭据，例如cookie
		AllowCredentials: true,
	}))

	beego.Run()
}
