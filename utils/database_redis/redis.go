package database_redis

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"
	"trbot/utils"
	"trbot/utils/consts"

	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
)

var MainDB = redis.NewClient(&redis.Options{
	Addr:     consts.RedisURL,
	Password: consts.RedisPassword,
	DB:       consts.RedisMainDB,
})

var UserDB = redis.NewClient(&redis.Options{
	Addr:     consts.RedisURL,
	Password: consts.RedisPassword,
	DB:       consts.RedisUserInfoDB,
})

func PingRedis(ctx context.Context, db *redis.Client) (string, error) {
	pong, err := db.Ping(ctx).Result()
	if err != nil {
		return "", err
	}
	return pong, nil
}

// func gobEncode(thing any) (any, error) {
// 	var buf bytes.Buffer
// 	enc := gob.NewEncoder(&buf)
// 	err := enc.Encode(thing)
// 	if err != nil {
// 		return nil, err
// 	} else {
// 		return buf.Bytes(), nil
// 	}
// }

// 保存用户信息
func saveChatInfo(ctx context.Context, chatID int64, chatInfo *ChatInfo) error {
	if chatInfo == nil {
		return fmt.Errorf("chatInfo 不能为空")
	}

	key := strconv.FormatInt(chatID, 10)
	v := reflect.ValueOf(*chatInfo) // 解除指针获取值
	t := reflect.TypeOf(*chatInfo)

	data := make(map[string]interface{})
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		data[field.Name] = fmt.Sprintf("%v", value.Interface())
	}

	return UserDB.HSet(ctx, key, data).Err()
}

// 获取用户信息
func getChatInfo(ctx context.Context, chatID int64) (*ChatInfo, error) {
	key := strconv.FormatInt(chatID, 10)
	data, err := UserDB.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	user := &ChatInfo{}
	v := reflect.ValueOf(user).Elem()
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)
		valueStr, exists := data[field.Name]
		if !exists {
			continue
		}

		fieldValue := v.Field(i)
		if fieldValue.CanSet() {
			switch fieldValue.Kind() {
			case reflect.String:
				fieldValue.SetString(valueStr)
			case reflect.Int, reflect.Int64:
				intValue, _ := strconv.Atoi(valueStr)
				fieldValue.SetInt(int64(intValue))
			}
		}
	}

	return user, nil
}


func InitUser(ctx context.Context, user *models.User) bool {
	chatData, err := getChatInfo(ctx, user.ID)
	if err != nil {
		log.Println("[UserDB] Error getting chat info from Redis:", err)
		return false
	}
	if chatData == nil {
		var newUser = ChatInfo{
			ID:       user.ID,
			ChatName: utils.ShowUserName(user),
			ChatType: models.ChatTypePrivate,
			AddTime: time.Now().Format(time.RFC3339),
		}

		err = saveChatInfo(ctx, user.ID, &newUser)
		if err != nil {
			log.Println("[UserDB] Error saving user info to Redis:", err)
			return false
		}
		log.Printf("newUser: \"%s\"(%d)\n", newUser.ChatName, user.ID)
		return true
	} else {
		log.Printf("oldUser: \"%s\"(%d)\n", chatData.ChatName, chatData.ID)
		return false
	}
}

func InitChat(ctx context.Context, chat *models.Chat) bool {
	chatData, err := getChatInfo(ctx, chat.ID)
	if err != nil {
		log.Println("[UserDB] Error getting chat info from Redis:", err)
		return false
	}
	if chatData == nil {
		var newChat = ChatInfo{
			ID:       chat.ID,
			ChatName: utils.ShowChatName(chat),
			ChatType: models.ChatTypePrivate,
			AddTime: time.Now().Format(time.RFC3339),
		}

		err = saveChatInfo(ctx, chat.ID, &newChat)
		if err != nil {
			log.Println("[UserDB] Error saving chat info to Redis:", err)
			return false
		}
		log.Printf("newChat: \"%s\"(%d)\n", newChat.ChatName, newChat.ID)
		return true
	} else {
		log.Printf("oldChat: \"%s\"(%d)\n", chatData.ChatName, chatData.ID)
		return false
	}
}
