package redis_db

import (
	"context"
	"fmt"
	"log"
	"reflect"
	"strconv"
	"time"

	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/configs"

	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
)

var MainDB *redis.Client // 配置文件
var UserDB *redis.Client // 用户数据

var ctxbg = context.Background()

func InitializeDB() (bool, error) {
	if configs.BotConfig.RedisURL != "" {
		if configs.BotConfig.RedisMainDB != -1 {
			MainDB = redis.NewClient(&redis.Options{
				Addr:     configs.BotConfig.RedisURL,
				Password: configs.BotConfig.RedisPassword,
				DB:       configs.BotConfig.RedisMainDB,
			})
			err := PingRedis(ctxbg, MainDB)
			if err != nil {
				return false, fmt.Errorf("error ping Redis MainDB: %s", err)
			}
		}
		if configs.BotConfig.RedisUserInfoDB != -1 {
			UserDB = redis.NewClient(&redis.Options{
				Addr:     configs.BotConfig.RedisURL,
				Password: configs.BotConfig.RedisPassword,
				DB:       configs.BotConfig.RedisUserInfoDB,
			})
			err := PingRedis(ctxbg, UserDB)
			if err != nil {
				return false, fmt.Errorf("error ping Redis UserDB: %s", err)
			}
		}

		return true, nil
	} else {
		return false, fmt.Errorf("RedisURL is empty")
	}
}

func PingRedis(ctx context.Context, db *redis.Client) error {
	_, err := db.Ping(ctx).Result()
	return err
}

// 保存用户信息
func SaveChatInfo(ctx context.Context, chatInfo *db_struct.ChatInfo) error {
	if chatInfo == nil {
		return fmt.Errorf("chatInfo 不能为空")
	}

	key := strconv.FormatInt(chatInfo.ID, 10)
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
func GetChatInfo(ctx context.Context, chatID int64) (*db_struct.ChatInfo, error) {
	key := strconv.FormatInt(chatID, 10)
	data, err := UserDB.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	user := &db_struct.ChatInfo{}
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


func InitUser(ctx context.Context, user *models.User) error {
	chatData, err := GetChatInfo(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("[UserDB] Error getting chat info from Redis: %v", err)
	}
	if chatData == nil {
		var newUser = db_struct.ChatInfo{
			ID:       user.ID,
			ChatName: utils.ShowUserName(user),
			ChatType: models.ChatTypePrivate,
			AddTime:  time.Now().Format(time.RFC3339),
		}

		err = SaveChatInfo(ctx, &newUser)
		if err != nil {
			return fmt.Errorf("[UserDB] Error saving user info to Redis: %v", err)
		}
		log.Printf("newUser: \"%s\"(%d)\n", newUser.ChatName, user.ID)
		return nil
	} else {
		log.Printf("oldUser: \"%s\"(%d)\n", chatData.ChatName, chatData.ID)
		return nil
	}
}

func InitChat(ctx context.Context, chat *models.Chat) error {
	chatData, err := GetChatInfo(ctx, chat.ID)
	if err != nil {
		return fmt.Errorf("[UserDB] Error getting chat info from Redis: %v", err)
	}
	if chatData == nil {
		var newChat = db_struct.ChatInfo{
			ID:       chat.ID,
			ChatName: utils.ShowChatName(chat),
			ChatType: models.ChatTypePrivate,
			AddTime:  time.Now().Format(time.RFC3339),
		}

		err = SaveChatInfo(ctx, &newChat)
		if err != nil {
			return fmt.Errorf("[UserDB] Error saving chat info to Redis: %v", err)
		}
		log.Printf("newChat: \"%s\"(%d)\n", newChat.ChatName, newChat.ID)
		return nil
	} else {
		log.Printf("oldChat: \"%s\"(%d)\n", chatData.ChatName, chatData.ID)
		return nil
	}
}

func IncrementalUsageCount(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_UsageCount) error {
	count, err := UserDB.HGet(ctx, strconv.FormatInt(chatID, 10), string(fieldName)).Int()
	if err == nil {
		err = UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), fieldName, count + 1).Err()
		if err == nil {
			return nil
		}
	} else if err == redis.Nil {
		err = UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), fieldName, 1).Err()
		if err == nil {
			log.Printf("[UserDB] Key %s not found, creating in Redis\n", fieldName)
			return nil
		}
	}

	return fmt.Errorf("[UserDB] Error incrementing usage count to Redis: %v", err)
}

func RecordLatestData(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_LatestData, value string) error {
	err := UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), fieldName, value).Err()
	if err == nil {
		return nil
	}
	
	return fmt.Errorf("[UserDB] Error saving chat info to Redis: %v", err)
}

func UpdateOperationStatus(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_Status, value bool) error {
	err := UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), fieldName, value).Err()
	if err == nil {
		return nil
	}

	return fmt.Errorf("[UserDB] Error update operation status to Redis: %v", err)
}

func SetCustomFlag(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_CustomFlag, value string) error {
    err := UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), fieldName, value).Err()
	if err == nil {
		return nil
	}

	return fmt.Errorf("[UserDB] Error setting custom flag to Redis: %v", err)
}
