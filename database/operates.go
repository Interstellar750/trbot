package database

import (
	"context"
	"fmt"
	"trbot/database/db_struct"

	"github.com/go-telegram/bot/models"
)

func InitChat(ctx context.Context, chat *models.Chat) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.InitChat(ctx, chat)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.InitChat(ctx, chat)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func InitUser(ctx context.Context, user *models.User) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.InitUser(ctx, user)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.InitUser(ctx, user)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func GetChatInfo(ctx context.Context, chatID int64) (*db_struct.ChatInfo, error) {
	// 优先从高优先级数据库获取数据
	for _, db := range DBBackends {
		return db.GetChatInfo(ctx, chatID)
	}
	for _, db := range DBBackends_LowLevel {
		return db.GetChatInfo(ctx, chatID)
	}
	return nil, fmt.Errorf("no database available")
}

func IncrementalUsageCount(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_UsageCount) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.IncrementalUsageCount(ctx, chatID, fieldName)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.IncrementalUsageCount(ctx, chatID, fieldName)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func RecordLatestData(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_LatestData, data string) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.RecordLatestData(ctx, chatID, fieldName, data)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.RecordLatestData(ctx, chatID, fieldName, data)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func UpdateOperationStatus(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_Status, value bool) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.UpdateOperationStatus(ctx, chatID, fieldName, value)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.UpdateOperationStatus(ctx, chatID, fieldName, value)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func SetCustomFlag(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_CustomFlag, value string) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.SetCustomFlag(ctx, chatID, fieldName, value)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.SetCustomFlag(ctx, chatID, fieldName, value)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func SaveDatabase(ctx context.Context) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.SaveDatabase(ctx)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.SaveDatabase(ctx)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func ReadDatabase(ctx context.Context) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.ReadDatabase(ctx)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.ReadDatabase(ctx)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}
