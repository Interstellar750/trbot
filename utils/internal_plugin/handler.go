package internal_plugin

import (
	"fmt"
	"strings"
	"trbot/utils"
	"trbot/utils/handler_structs"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)


func startHandler(params *handler_structs.SubHandlerParams) error {
	defer utils.PanicCatcher(params.Ctx, "startHandler")
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "startHandler").
		Logger()

	if len(params.Fields) > 1 {
		for _, n := range plugin_utils.AllPlugins.SlashStart.WithPrefixHandler {
			if strings.HasPrefix(params.Fields[1], n.Prefix) {
				inlineArgument := strings.Split(params.Fields[1], "_")
				if inlineArgument[1] == n.Argument {
					if n.Handler == nil {
						logger.Debug().
							Dict(utils.GetUserDict(params.Update.Message.From)).
							Str("handlerPrefix", n.Prefix).
							Str("handlerArgument", n.Argument).
							Str("handlerName", n.Name).
							Str("fullCommand", params.Update.Message.Text).
							Msg("tigger /start command handler by prefix, but this handler function is nil, skip")
						continue
					}
					err := n.Handler(params)
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(params.Update.Message.From)).
							Str("handlerPrefix", n.Prefix).
							Str("handlerArgument", n.Argument).
							Str("handlerName", n.Name).
							Str("fullCommand", params.Update.Message.Text).
							Msg("Error in /start command handler by prefix tigger")
					}
					return err
				}
			}
		}
		for _, n := range plugin_utils.AllPlugins.SlashStart.Handler {
			if params.Fields[1] == n.Argument {
				err := n.Handler(params)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(params.Update.Message.From)).
						Str("handlerArgument", n.Argument).
						Str("handlerName", n.Name).
						Str("fullCommand", params.Update.Message.Text).
						Msg("Error in /start command handler by argument")
				}
				return err
			}
		}
	}

	_, err := params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
		ChatID:             params.Update.Message.Chat.ID,
		Text:               fmt.Sprintf("Hello, *%s %s*\n\n您可以向此处发送一个贴纸，您会得到一张转换后的 png 图片\n\n您也可以使用 [inline](https://telegram.org/blog/inline-bots?setln=en) 模式进行交互，点击下方的按钮来使用它", params.Update.Message.From.FirstName, params.Update.Message.From.LastName),
		ParseMode:          models.ParseModeMarkdownV1,
		ReplyParameters:    &models.ReplyParameters{ MessageID: params.Update.Message.ID },
		LinkPreviewOptions: &models.LinkPreviewOptions{ IsDisabled: bot.True() },
		ReplyMarkup:        &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text:                         "尝试 Inline 模式",
			SwitchInlineQueryCurrentChat: " ",
		}}}},
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetChatDict(&params.Update.Message.Chat)).
			Dict(utils.GetUserDict(params.Update.Message.From)).
			Msg("Failed to send `bot welcome` message")
	}

	return err
}

func helpHandler(params *handler_structs.SubHandlerParams) error {
	defer utils.PanicCatcher(params.Ctx, "helpHandler")
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "helpHandler").
		Logger()

	_, err := params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
		ChatID:             params.Update.Message.Chat.ID,
		Text:               fmt.Sprintf("当前 bot 中有 %d 个帮助文档", len(plugin_utils.AllPlugins.HandlerHelp)),
		ParseMode:          models.ParseModeMarkdownV1,
		ReplyParameters:    &models.ReplyParameters{MessageID: params.Update.Message.ID},
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: bot.True()},
		ReplyMarkup:        plugin_utils.BuildHandlerHelpKeyboard(),
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetChatDict(&params.Update.Message.Chat)).
			Dict(utils.GetUserDict(params.Update.Message.From)).
			Msg("Failed to send `bot help keyboard` message")
	}
	return err
}

func helpCallbackHandler(params *handler_structs.SubHandlerParams) error {
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "helpCallbackHandler").
		Logger()
	
	if params.Update.CallbackQuery.Data == "help-close" {
		_, err := params.Thebot.DeleteMessage(params.Ctx, &bot.DeleteMessageParams{
			ChatID:    params.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: params.Update.CallbackQuery.Message.Message.ID,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Dict(utils.GetChatDict(&params.Update.CallbackQuery.Message.Message.Chat)).
			Msg("Failed to delete `bot help keyboard` message")
		}
		return err
	} else if strings.HasPrefix(params.Update.CallbackQuery.Data, "help-handler_") {
		handlerName := strings.TrimPrefix(params.Update.CallbackQuery.Data, "help-handler_")
		for _, handler := range plugin_utils.AllPlugins.HandlerHelp {
			if handler.Name == handlerName {
				var replyMarkup models.ReplyMarkup

				// 如果帮助函数有自定的 ReplyMarkup，则使用它，否则显示默认的按钮
				if handler.ReplyMarkup != nil {
					replyMarkup = handler.ReplyMarkup
				} else {
					replyMarkup = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text:         "返回",
							CallbackData: "help",
						},
						{
							Text:         "关闭",
							CallbackData: "help-close",
						},
					}}}
				}

				_, err := params.Thebot.EditMessageText(params.Ctx, &bot.EditMessageTextParams{
					ChatID:      params.Update.CallbackQuery.Message.Message.Chat.ID,
					MessageID:   params.Update.CallbackQuery.Message.Message.ID,
					Text:        handler.Description,
					ParseMode:   handler.ParseMode,
					ReplyMarkup: replyMarkup,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetChatDict(&params.Update.CallbackQuery.Message.Message.Chat)).
						Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
						Str("pluginName", handler.Name).
						Msg("Edit messag to `plugin help message` failed")
				}
				return err
			}
		}
		_, err := params.Thebot.AnswerCallbackQuery(params.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: params.Update.CallbackQuery.ID,
			Text:            "您请求查看的帮助页面不存在，可能是机器人管理员已经移除了这个插件",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Msg("Failed to send `help page is not exist` callback answer")
		}
	}
	
	_, err := params.Thebot.EditMessageText(params.Ctx, &bot.EditMessageTextParams{
		ChatID:      params.Update.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   params.Update.CallbackQuery.Message.Message.ID,
		Text:        fmt.Sprintf("当前 bot 中有 %d 个帮助文档", len(plugin_utils.AllPlugins.HandlerHelp)),
		ReplyMarkup: plugin_utils.BuildHandlerHelpKeyboard(),
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetChatDict(&params.Update.CallbackQuery.Message.Message.Chat)).
			Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
			Msg("Edit messag to `bot help keyboard` failed")
	}
	return err
}
