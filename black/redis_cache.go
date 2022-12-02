package black

import (
	"strconv"
	"time"

	"github.com/beego/beego/v2/core/logs"
	"github.com/go-redis/redis"
)

var (
	gRedisClient *redis.Client
)

const (
	black_null       = 0
	black_audio      = 1
	black_sms        = 2
	black_audio_read = 256
	black_sms_read   = 512
)

func redisInit(addr, passwd string) {

	if gRedisClient != nil {
		return
	}

	gRedisClient = redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     passwd,
		PoolSize:     800,
		MinIdleConns: 5,
		IdleTimeout:  180 * time.Second,
		DB:           8,
	})

	res := gRedisClient.Ping()
	if res.Err() != nil {
		logs.Error("redis connect failed:%v", res.String())
		gRedisClient = nil
	} else {
		logs.Info("redis 连接成功......")
	}
}

func CacheCheck(mobile string) (isBlack bool, hit bool) {

	cache := getFromRedis(mobile)
	if len(cache) == 0 {
		return false, false
	}

	hit = true
	if cache == "true" {
		isBlack = true
	} else {
		isBlack = false
	}
	return
}

func cacheDelete(mobile string) error {

	if gRedisClient != nil {

		result := gRedisClient.Del(mobile)
		logs.Debug("[%v]从缓存中删除:%v", mobile, result.String())
		return result.Err()
	} else {
		return nil
	}
}

func SaveToRedisWithType(cache, mobile, busineseType string, isBlack bool) {

	if gRedisClient == nil {
		return
	}

	if len(cache) == 0 {
		//redis本来不存在
		value := 0
		expire := time.Hour * 48
		if isBlack {
			switch busineseType {
			case "voice":
				value |= black_audio
			case "sms":
				value |= black_sms
			}
		} else {
			switch busineseType {
			case "voice":
				value |= black_audio_read
			case "sms":
				value |= black_sms_read
			}
		}

		result := gRedisClient.Set(mobile, value, expire)
		if result.Err() != nil {
			logs.Debug("save [%v] to redis:%v,err:%v", mobile, result.Val(), result.Err())
		}

	} else {
		value, err := strconv.Atoi(cache)
		if err != nil {
			SaveToRedisWithType("", mobile, busineseType, isBlack)
			return
		}

		expire := time.Hour * 48
		if isBlack {

			switch busineseType {
			case "sms":
				value |= black_sms
			case "voice":
				value |= black_audio
			}
		} else {
			switch busineseType {
			case "sms":
				value |= black_sms_read
			case "voice":
				value |= black_audio_read
			}
		}

		result := gRedisClient.Set(mobile, value, expire)
		if result.Err() != nil {
			logs.Debug("save [%v] to redis:%v,err:%v", mobile, result.Val(), result.Err())
		}
	}
}

func SaveToRedis(mobile string, isBlack bool) {

	if gRedisClient == nil {
		return
	}

	value := "false"
	expire := time.Hour * 48
	if isBlack {
		value = "true"
		expire = 0
	}
	gRedisClient.Set(mobile, value, expire)
	//logs.Debug("save [%v] to redis:%v,err:%v", mobile, result.Val(), result.Err())
}

func getFromRedis(mobile string) string {

	if gRedisClient == nil {
		return ""
	}

	result := gRedisClient.Get(mobile)
	logs.Debug("[%v]从redis获取信息:%v", mobile, result.String())
	return result.Val()
}

func CacheCheckWithType(mobile, busineseType string) (result string, isBlack bool, hit bool) {

	cache := getFromRedis(mobile)
	if len(cache) == 0 {
		return "", false, false
	}

	//采用2个字节，第一个字节 1为true,0为false，第二个字节，0为语音，1为短信
	hit = false
	isBlack = false
	result = cache
	value, err := strconv.Atoi(cache)
	if err != nil {
		return "", false, false
	}

	if busineseType == "voice" {
		if value&black_audio > 0 {
			isBlack = true
			hit = true
		} else if value&black_audio_read > 0 {
			hit = true
		}
		return
	} else if busineseType == "sms" {
		if value&black_sms > 0 {
			isBlack = true
			hit = true
		} else if value&black_sms_read > 0 {
			hit = true
		}
		return
	}
	return
}
