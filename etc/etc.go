package etc

import (
	"github.com/beego/beego/v2/core/logs"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type (
	MysqlDB struct {
		Name     string `yaml:"name"`
		IP       string `yaml:"ip"`
		Port     int    `yaml:"port"`
		Db       string `yaml:"db"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
	}

	Mysql struct {
		Debug    bool    `yaml:"debug"`
		MysqlDBs MysqlDB `yaml:"mysql_dbs"`
	}

	Redis struct {
		IP       string `yaml:"ip"`
		Password string `yaml:"password"`
	}

	common struct {
		Dsid      int    `yaml:"dsid"`
		AccessKey string `yaml:"accessKey"`
	}

	server struct {
		Start     bool   `yaml:"start"`
		AccessId  int    `yaml:"accessId"`
		AccessKey string `yaml:"accessKey"`
	}

	ApiStore struct {
		Start     bool   `yaml:"start"`
		AccessId  string `yaml:"accessId"`
		AccessKey string `yaml:"accessKey"`
		Url       string `yaml:"url"`
	}

	BlackServ struct {
		Voice     server `yaml:"voice"`
		Sms       server `yaml:"sms"`
		Url       string `yaml:"url"`
		MaxNumReq int    `yaml:"number-per-request"`
	}

	oss struct {
		EndPoint   string `yaml:"endPoint"`
		Bucket     string `yaml:"bucket"`
		SecretId   string `yaml:"secretId"`
		SecretKey  string `yaml:"secretKey"`
		UploadTask int    `yaml:"taskNum"`
	}

	config struct {
		Api     ApiStore  `yaml:"api_store"`
		BlackSv BlackServ `yaml:"black_service"`
		Mysql   Mysql     `yaml:"mysql"`
		Storage oss       `yaml:"oss"`
		Redis   Redis     `yaml:"redis"`
		Base    common    `yaml:"common"`
	}
)

var Conf config

// 初始化配置文件
func InitConfig(file string) error {

	bs, err := ioutil.ReadFile(file)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(bs, &Conf)
}

// 初始化配置
func InitConfigFromData(data []byte) error {

	return yaml.Unmarshal(data, &Conf)

}

// 初始化配置
func UpdateConfigFromData(data []byte) {

	newConf := config{}
	err := yaml.Unmarshal(data, &newConf)
	if err != nil {
		logs.Error("解析配置文件失败:%v", err)
		return
	}

	if newConf.Base != Conf.Base {
		logs.Debug("Base 配置变更")
	}

	if newConf.Mysql != Conf.Mysql {
		logs.Debug("Mysql 配置变更,Old:%v,新配置:%v", Conf.Mysql, newConf.Mysql)
	}

	if newConf.Redis != Conf.Redis {
		logs.Debug("Redis 配置变更")
	}

	if newConf.Storage != Conf.Storage {
		logs.Debug("Storage 配置变更")
	}
}
