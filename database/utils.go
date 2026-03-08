package database

import (
	"trle5.xyz/trbot/database/db_struct"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/type/update_utils"

	"github.com/rs/zerolog"
)

func RecordData(params *handler_params.Update, updateType update_utils.Update) {
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Logger()

	switch {
	case updateType.Message:
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
		if params.Update.Message.Caption != "" {
			err = RecordLatestData(params.Ctx, params.Update.Message.Chat.ID, db_struct.LatestMessage, params.Update.Message.Caption)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&params.Update.Message.Chat)).
					Msg("Failed to record latest `caption` data")
			}
		} else if params.Update.Message.Text != "" {
			err = RecordLatestData(params.Ctx, params.Update.Message.Chat.ID, db_struct.LatestMessage, params.Update.Message.Text)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&params.Update.Message.Chat)).
					Msg("Failed to record latest `message text` data")
			}
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
	case updateType.ChannelPost:
		err := InitChat(params.Ctx, &params.Update.ChannelPost.Chat)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.ChannelPost.Chat)).
				Msg("Failed to init channel")
		}
		err = IncrementalUsageCount(params.Ctx, params.Update.ChannelPost.Chat.ID, db_struct.MessageNormal)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.ChannelPost.Chat)).
				Msg("Failed to incremental `message` usage count")
		}
		if params.Update.ChannelPost.Caption != "" {
			err = RecordLatestData(params.Ctx, params.Update.ChannelPost.Chat.ID, db_struct.LatestMessage, params.Update.ChannelPost.Caption)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&params.Update.ChannelPost.Chat)).
					Msg("Fail to record latest `caption` data")
			}
		} else if params.Update.ChannelPost.Text != "" {
			err = RecordLatestData(params.Ctx, params.Update.ChannelPost.Chat.ID, db_struct.LatestMessage, params.Update.ChannelPost.Text)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&params.Update.ChannelPost.Chat)).
					Msg("Failed to record latest `message text` data")
			}
		}
		params.ChatInfo, err = GetChatInfo(params.Ctx, params.Update.ChannelPost.Chat.ID)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&params.Update.ChannelPost.Chat)).
				Msg("Failed to get channel info")
		}
	case updateType.InlineQuery:
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
