package internal_plugin

import (
	"fmt"
	"strings"
	"trbot/utils"
	"trbot/utils/err_template"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

func startHandler(params *handler_params.Update) error {
	defer utils.PanicCatcher(params.Ctx, "startHandler")
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "startHandler").
		Dict(utils.GetChatDict(&params.Update.Message.Chat)).
		Dict(utils.GetUserDict(params.Update.Message.From)).
		Str("text",    params.Update.Message.Text).
		Logger()

	var messageParams = handler_params.Message{
		Ctx:      params.Ctx,
		Thebot:   params.Thebot,
		Message:  params.Update.Message,
		ChatInfo: params.ChatInfo,
		Fields:   strings.Fields(params.Update.Message.Text),
	}

	if len(messageParams.Fields) > 1 {
		for _, plugin := range plugin_utils.AllPlugins.SlashStartCommand.WithPrefixHandler {
			if strings.HasPrefix(messageParams.Fields[1], plugin.Prefix) {
				inlineArgument := strings.Split(messageParams.Fields[1], "_")
				if inlineArgument[1] == plugin.Argument {
					slogger := logger.With().
						Str("handlerPrefix", plugin.Prefix).
						Str("handlerArgument", plugin.Argument).
						Str("handlerName", plugin.Name).
						Logger()

					slogger.Info().Msg("Hit /start command handler by prefix")

					var err error
					switch {
					case plugin.MessageHandler != nil:
						err = plugin.MessageHandler(&messageParams)
					case plugin.UpdateHandler != nil:
						err = plugin.UpdateHandler(params)
					default:
						slogger.Warn().Msg("Hit /start command handler by prefix, but this handler all function is nil, skip")
						continue
					}
					if err != nil {
						slogger.Error().
							Err(err).
							Msg("Error in /start command handler by prefix trigger")
					}
					return err
				}
			}
		}
		for _, plugin := range plugin_utils.AllPlugins.SlashStartCommand.Handler {
			if messageParams.Fields[1] == plugin.Argument {
				slogger := logger.With().
					Str("handlerArgument", plugin.Argument).
					Str("handlerName", plugin.Name).
					Logger()

				slogger.Info().Msg("Hit /start command handler by argument")

				var err error
				switch {
				case plugin.MessageHandler != nil:
					err = plugin.MessageHandler(&messageParams)
				case plugin.UpdateHandler != nil:
					err = plugin.UpdateHandler(params)
				default:
					slogger.Warn().Msg("Hit /start command handler by argument, but this handler all function is nil, skip")
					continue
				}
				if err != nil {
					slogger.Error().
						Err(err).
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
			Str("content", "bot welcome").
			Msg(err_template.SendMessage)
	}

	return err
}

func helpHandler(params *handler_params.Message) error {
	defer utils.PanicCatcher(params.Ctx, "helpHandler")
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "helpHandler").
		Logger()

	_, err := params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
		ChatID:             params.Message.Chat.ID,
		Text:               fmt.Sprintf("当前 bot 中有 %d 个帮助文档", len(plugin_utils.AllPlugins.HandlerHelp)),
		ParseMode:          models.ParseModeMarkdownV1,
		ReplyParameters:    &models.ReplyParameters{MessageID: params.Message.ID},
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: bot.True()},
		ReplyMarkup:        plugin_utils.BuildHandlerHelpKeyboard(),
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetChatDict(&params.Message.Chat)).
			Dict(utils.GetUserDict(params.Message.From)).
			Str("content", "bot help keyboard").
			Msg(err_template.SendMessage)
	}
	return err
}

func helpCallbackHandler(params *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "helpCallbackHandler").
		Dict(utils.GetUserDict(&params.CallbackQuery.From)).
		Dict(utils.GetChatDict(&params.CallbackQuery.Message.Message.Chat)).
		Logger()

	if params.CallbackQuery.Data == "help-close" {
		_, err := params.Thebot.DeleteMessage(params.Ctx, &bot.DeleteMessageParams{
			ChatID:    params.CallbackQuery.Message.Message.Chat.ID,
			MessageID: params.CallbackQuery.Message.Message.ID,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "bot help keyboard").
				Msg(err_template.DeleteMessage)
		}
		return err
	} else if strings.HasPrefix(params.CallbackQuery.Data, "help-handler_") {
		handlerName := strings.TrimPrefix(params.CallbackQuery.Data, "help-handler_")
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
					ChatID:      params.CallbackQuery.Message.Message.Chat.ID,
					MessageID:   params.CallbackQuery.Message.Message.ID,
					Text:        handler.Description,
					ParseMode:   handler.ParseMode,
					ReplyMarkup: replyMarkup,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("pluginName", handler.Name).
						Str("content", "plugin help message").
						Msg(err_template.EditMessageText)
				}
				return err
			}
		}
		_, err := params.Thebot.AnswerCallbackQuery(params.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: params.CallbackQuery.ID,
			Text:            "您请求查看的帮助页面不存在，可能是机器人管理员已经移除了这个插件",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "help page is not exist").
				Msg(err_template.AnswerCallbackQuery)
		}
	}

	_, err := params.Thebot.EditMessageText(params.Ctx, &bot.EditMessageTextParams{
		ChatID:      params.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   params.CallbackQuery.Message.Message.ID,
		Text:        fmt.Sprintf("当前 bot 中有 %d 个帮助文档", len(plugin_utils.AllPlugins.HandlerHelp)),
		ReplyMarkup: plugin_utils.BuildHandlerHelpKeyboard(),
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("content", "bot help keyboard").
			Msg(err_template.EditMessageText)
	}
	return err
}
