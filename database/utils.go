package database

import (
	"strings"
	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/handler_structs"
	"trbot/utils/type/update_utils"

	"github.com/rs/zerolog"
)

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
				Msg("Failed to init chat")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.Message.Chat.ID, db_struct.MessageNormal)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.Message.Chat)).
				Msg("Failed to incremental `message` usage count")
		}
		err = RecordLatestData(params.Ctx, params.Update.Message.Chat.ID, db_struct.LatestMessage, params.Update.Message.Text)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.Message.Chat)).
				Msg("Failed to record latest `message text` data")
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.Message.Chat.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.Message.Chat)).
				Msg("Failed to get chat info")
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
				Msg("Failed to init user")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.InlineQuery.From.ID, db_struct.InlineRequest)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(params.Update.InlineQuery.From)).
				Msg("Failed to incremental `inline request` usage count")
		}
		err = RecordLatestData(params.Ctx, params.Update.InlineQuery.From.ID, db_struct.LatestInlineQuery, params.Update.InlineQuery.Query)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(params.Update.InlineQuery.From)).
				Msg("Failed to record latest `inline query` data")
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.InlineQuery.From.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(params.Update.InlineQuery.From)).
				Msg("Failed to get user info")
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
				Msg("Failed to init user")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.ChosenInlineResult.From.ID, db_struct.InlineResult)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("Failed to incremental `inline result` usage count")
		}
		err = RecordLatestData(params.Ctx, params.Update.ChosenInlineResult.From.ID, db_struct.LatestInlineResult, params.Update.ChosenInlineResult.ResultID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("failed to record latest `inline result` data")
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.ChosenInlineResult.From.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("Failed to get user info")
		}
	case updateType.CallbackQuery:
		err := InitUser(params.Ctx, &params.Update.CallbackQuery.From)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Msg("Failed to init user")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.CallbackQuery.From.ID, db_struct.CallbackQuery)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Msg("Failed to incremental `callback query` usage count")
		}
		err = RecordLatestData(params.Ctx, params.Update.CallbackQuery.From.ID, db_struct.LatestCallbackQueryData, params.Update.CallbackQuery.Data)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Msg("Failed to record latest `callback query` data")
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.CallbackQuery.From.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.ChosenInlineResult.From)).
				Msg("Failed get user info")
		}
	}
}
