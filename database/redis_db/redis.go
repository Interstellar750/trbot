package redis_db

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/configs"

	"github.com/go-telegram/bot/models"
	"github.com/redis/go-redis/v9"
)

var UserDB *redis.Client // 用户数据

func InitializeDB(ctx context.Context) error {
	if configs.BotConfig.RedisURL != "" {
		if configs.BotConfig.RedisDatabaseID != -1 {
			UserDB = redis.NewClient(&redis.Options{
				Addr:     configs.BotConfig.RedisURL,
				Password: configs.BotConfig.RedisPassword,
				DB:       configs.BotConfig.RedisDatabaseID,
			})
			err := UserDB.Ping(ctx).Err()
			if err != nil {
				return fmt.Errorf("failed to ping Redis [%d] database: %w", configs.BotConfig.RedisDatabaseID, err)
			}
		}

		return nil
	} else {
		return fmt.Errorf("RedisURL is empty")
	}
}

// 保存用户信息
func SaveChatInfo(ctx context.Context, chatInfo *db_struct.ChatInfo) error {
	if chatInfo == nil {
		return fmt.Errorf("failed to save chat info: chatInfo is nil")
	}

	key := strconv.FormatInt(chatInfo.ID, 10)
	v := reflect.ValueOf(*chatInfo)
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
		return nil, fmt.Errorf("failed to get chat info: %w", err)
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
		return fmt.Errorf("failed to get chat info: %w", err)
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
			return fmt.Errorf("failed to init new user: %w", err)
		}
	}
	return nil
}

func InitChat(ctx context.Context, chat *models.Chat) error {
	chatData, err := GetChatInfo(ctx, chat.ID)
	if err != nil {
		return fmt.Errorf("failed to get chat info: %w", err)
	}
	if chatData == nil {
		var newChat = db_struct.ChatInfo{
			ID:       chat.ID,
			ChatName: utils.ShowChatName(chat),
			ChatType: chat.Type,
			AddTime:  time.Now().Format(time.RFC3339),
		}

		err = SaveChatInfo(ctx, &newChat)
		if err != nil {
			return fmt.Errorf("failed to init new chat: %w", err)
		}
	}
	return nil
}

func IncrementalUsageCount(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_UsageCount) error {
	count, err := UserDB.HGet(ctx, strconv.FormatInt(chatID, 10), string(fieldName)).Int()
	if err != nil {
		if err == redis.Nil {
			err = UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), string(fieldName), 0).Err()
			if err != nil {
				return fmt.Errorf("failed to create empty [%s] key: %w", string(fieldName), err)
			}
		} else {
			return fmt.Errorf("failed to get [%s] usage count: %w", string(fieldName), err)
		}
	}

	err = UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), string(fieldName), count + 1).Err()
	if err != nil {
		return fmt.Errorf("failed to incrementing [%s] usage count: %w", string(fieldName), err)
	}

	return nil
}

func RecordLatestData(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_LatestData, value string) error {
	err := UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), string(fieldName), value).Err()
	if err != nil {
		return fmt.Errorf("failed to record latest [%s] data: %w", string(fieldName), err)
	}

	return nil
}

func UpdateOperationStatus(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_Status, value bool) error {
	err := UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), string(fieldName), value).Err()
	if err != nil {
		return fmt.Errorf("failed to update operation [%s] status: %w", string(fieldName), err)
	}

	return nil
}

func SetCustomFlag(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_CustomFlag, value string) error {
	err := UserDB.HSet(ctx, strconv.FormatInt(chatID, 10), string(fieldName), value).Err()
	if err != nil {
		return fmt.Errorf("failed to set custom [%s] flag: %w", string(fieldName), err)
	}

	return nil
}
