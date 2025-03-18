package database_redis

import (
	"bytes"
	"encoding/gob"
	"log"
	"strconv"
	"time"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_utils"

	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
)

var MainDB = redis.NewClient(&redis.Options{
	Addr:     consts.RedisURL,
	Password: consts.RedisPassword,
	DB:       consts.RedisMainDB,
})

var SubDB = redis.NewClient(&redis.Options{
	Addr:     consts.RedisURL,
	Password: consts.RedisPassword,
	DB:       consts.RedisSubDB,
})

func PingRedis(opts *handler_utils.SubHandlerOpts, db *redis.Client) (string, error) {
	pong, err := db.Ping(opts.Ctx).Result()
	if err != nil {
		return "", err
	}
	return pong, nil
}

func gobEncode(thing any) (any, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(thing)
	if err != nil {
		return nil, err
	} else {
		return buf.Bytes(), nil
	}
}

func InitUser(opts *handler_utils.SubHandlerOpts, user *models.User) bool {
	undecode, err := MainDB.HGet(opts.Ctx, "BaseInfo", strconv.FormatInt(user.ID, 10)).Bytes()
	if err != nil {

		var newUser = BaseInfo{
			ChatName: utils.ShowUserName(user),
			ChatType: models.ChatTypePrivate,
			AddTime: time.Now().Format(time.RFC3339),
		}

		err = MainDB.SAdd(opts.Ctx, "UserList", strconv.FormatInt(user.ID, 10)).Err()
		if err != nil { log.Println("Redis 创建 UserList 失败:", err) }

		newUserByte, err := gobEncode(newUser)
		if err != nil { log.Println("Gob 序列化 BaseInfo 失败:", err) }
		err = MainDB.HSet(opts.Ctx, "BaseInfo", strconv.FormatInt(user.ID, 10), newUserByte).Err()
		if err != nil { log.Println("Redis 创建 BaseInfo 失败:", err) }

		var newUserContent = LatestContent{}
		newUserContentByte, err := gobEncode(newUserContent)
		if err != nil { log.Println("Gob 序列化 LatestContent 失败:", err) }
		err = MainDB.HSet(opts.Ctx, "LatestContent", strconv.FormatInt(user.ID, 10), newUserContentByte).Err()
		if err != nil { log.Println("Redis 创建 LatestContent 失败:", err) }

		var newUserUsage = UsageCount{}
		newUserUsageByte, err := gobEncode(newUserUsage)
		if err != nil { log.Println("Gob 序列化 UsageCount 失败:", err) }
		err = MainDB.HSet(opts.Ctx, "UsageCount", strconv.FormatInt(user.ID, 10), newUserUsageByte).Err()
		if err != nil { log.Println("Redis 创建 UsageCount 失败:", err) }

		log.Printf("newUser: \"%s\"(%d)\n", newUser.ChatName, user.ID)
		return true
	} else {
		var oldUser BaseInfo
		dec := gob.NewDecoder(bytes.NewReader(undecode))
		err = dec.Decode(&oldUser)
		if err != nil {
			log.Println("Gob 反序列化 BaseInfo 失败:", err)
		}

		// 打印获取到的结构体
		log.Printf("oldUser: \"%s\"(%d)\n", oldUser.ChatName, user.ID)
		return false
	}
}
