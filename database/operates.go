package database

import (
	"context"
	"fmt"
	"strings"

	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/handler_structs"
	"trbot/utils/type/update_utils"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
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


func RecordData(params *handler_structs.SubHandlerParams) {
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "RecordData").
		Logger()

	updateType := update_utils.GetUpdateType(params.Update)

	switch {
	case updateType.Message:
		if params.Update.Message.Text != "" {
			params.Fields = strings.Fields(params.Update.Message.Text)
		}
		err := InitChat(params.Ctx, &params.Update.Message.Chat)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.Message.Chat)).
				Msg("Init chat failed")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.Message.Chat.ID, db_struct.MessageNormal)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.Message.Chat)).
				Msg("Incremental message count failed")
		}
		err = RecordLatestData(params.Ctx, params.Update.Message.Chat.ID, db_struct.LatestMessage, params.Update.Message.Text)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.Message.Chat)).
				Msg("Record latest message failed")
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.Message.Chat.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.Message.Chat)).
				Msg("Get chat info failed")
		}
	case updateType.EditedMessage:
		// no ?
	case updateType.InlineQuery:
		if params.Update.InlineQuery.Query != "" {
			params.Fields = strings.Fields(params.Update.InlineQuery.Query)
		}

		err := InitUser(params.Ctx, params.Update.InlineQuery.From)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(params.Update.InlineQuery.From)).
				Msg("Init user failed")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.InlineQuery.From.ID, db_struct.InlineRequest)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(params.Update.InlineQuery.From)).
				Msg("Incremental inline request count failed")
		}
		err = RecordLatestData(params.Ctx, params.Update.InlineQuery.From.ID, db_struct.LatestInlineQuery, params.Update.InlineQuery.Query)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(params.Update.InlineQuery.From)).
				Msg("Record latest inline query failed")
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.InlineQuery.From.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(params.Update.InlineQuery.From)).
				Msg("Get user info failed")
		}
	case updateType.ChosenInlineResult:
		if params.Update.ChosenInlineResult.Query != "" {
			params.Fields = strings.Fields(params.Update.ChosenInlineResult.Query)
		}

		err := InitUser(params.Ctx, &params.Update.ChosenInlineResult.From)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("Init user failed")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.ChosenInlineResult.From.ID, db_struct.InlineResult)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("Incremental inline result count failed")
		}
		err = RecordLatestData(params.Ctx, params.Update.ChosenInlineResult.From.ID, db_struct.LatestInlineResult, params.Update.ChosenInlineResult.ResultID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("Record latest inline result failed")
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.ChosenInlineResult.From.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("Get user info failed")
		}
	case updateType.CallbackQuery:
		err := InitUser(params.Ctx, &params.Update.CallbackQuery.From)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Msg("Init user failed")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.CallbackQuery.From.ID, db_struct.CallbackQuery)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Msg("Incremental callback query count failed")
		}
		err = RecordLatestData(params.Ctx, params.Update.CallbackQuery.From.ID, db_struct.LatestCallbackQueryData, params.Update.CallbackQuery.Data)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Msg("Record latest callback query failed")
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.CallbackQuery.From.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("Get user info failed")
		}
	}
}
