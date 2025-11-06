package database

import (
	"context"

	"trbot/database/db_struct"

	"github.com/go-telegram/bot/models"
)


func InitChat(ctx context.Context, chat *models.Chat) error {
	return database.InitChat(ctx, chat)
}

func InitUser(ctx context.Context, user *models.User) error {
	return database.InitUser(ctx, user)
}

func GetChatInfo(ctx context.Context, chatID int64) (data *db_struct.ChatInfo, err error) {
	return database.GetChatInfo(ctx, chatID)
}

func IncrementalUsageCount(ctx context.Context, chatID int64, fieldName db_struct.UsageCount) error {
	return database.IncrementalUsageCount(ctx, chatID, fieldName)
}

func RecordLatestData(ctx context.Context, chatID int64, fieldName db_struct.LatestData, data string) error {
	return database.RecordLatestData(ctx, chatID, fieldName, data)
}

func UpdateOperationStatus(ctx context.Context, chatID int64, fieldName db_struct.Status, value bool) error {
	return database.UpdateOperationStatus(ctx, chatID, fieldName, value)
}

func SetCustomFlag(ctx context.Context, chatID int64, fieldName db_struct.Flag, value string) error {
	return database.SetCustomFlag(ctx, chatID, fieldName, value)
}

func SaveDatabase(ctx context.Context) error {
	return database.SaveDatabase(ctx)
}

func ReadDatabase(ctx context.Context) error {
	return database.ReadDatabase(ctx)
}
