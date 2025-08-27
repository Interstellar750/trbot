package internal_plugin

import (
	"fmt"
	"strings"
	"trbot/utils"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

func startHandler(params *handler_params.Message) error {
	defer utils.PanicCatcher(params.Ctx, "startHandler")
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetChatDict(&params.Message.Chat)).
		Dict(utils.GetUserDict(params.Message.From)).
		Str("text", params.Message.Text).
		Logger()

	if len(params.Fields) > 1 {
		// todo: move to RunSlashStartWithPrefixCommandHandlers
		for _, plugin := range plugin_utils.AllPlugins.SlashStartCommand.WithPrefixHandler {
			if strings.HasPrefix(params.Fields[1], plugin.PrefixArgument) {
				slogger := logger.With().
					Str("handlerPrefixArgument", plugin.PrefixArgument).
					Str("handlerName", plugin.Name).
					Logger()

				if plugin.MessageHandler != nil {
					slogger.Info().Msg("Hit by prefix /start command handler")
					err := plugin.MessageHandler(params)
					if err != nil {
						slogger.Error().
							Err(err).
							Msg("Error in by prefix /start command handler")
					}
					return err
				} else {
					slogger.Warn().Msg("Hit by prefix /start command handler, but this handler function is nil, skip")
				}
			}
		}
		// todo: move to RunSlashStartCommandHandlers
		for _, plugin := range plugin_utils.AllPlugins.SlashStartCommand.Handler {
			if params.Fields[1] == plugin.Argument {
				slogger := logger.With().
					Str("handlerArgument", plugin.Argument).
					Str("handlerName", plugin.Name).
					Logger()

				if plugin.MessageHandler != nil {
					slogger.Info().Msg("Hit by argument /start command handler")
					err := plugin.MessageHandler(params)
					if err != nil {
						slogger.Error().
							Err(err).
							Msg("Error in by argument /start command handler")
					}
					return err
				} else {
					slogger.Warn().Msg("Hit by argument /start command handler, but this handler function is nil, skip")
				}
			}
		}
	}

	var err error
	if params.Message.Chat.Type == models.ChatTypePrivate {
		_, err = params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
			ChatID:          params.Message.Chat.ID,
			Text:            fmt.Sprintf("Hello, *%s*\n\n您可以向此处发送一个贴纸，您会得到一张转换后的 PNG 或 GIF 图片\n\n您可以打开左下角的命令菜单或发送 /help 命令来查看更多帮助信息", utils.ShowUserName(params.Message.From)),
			ParseMode:       models.ParseModeMarkdownV1,
			ReplyParameters: &models.ReplyParameters{ MessageID: params.Message.ID },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "bot welcome").
				Msg(flaterr.SendMessage.Str())
		}
	}

	return err
}

func helpHandler(params *handler_params.Message) error {
	defer utils.PanicCatcher(params.Ctx, "helpHandler")

	_, err := params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
		ChatID:          params.Message.Chat.ID,
		Text:            fmt.Sprintf("当前 bot 中有 %d 个帮助文档", len(plugin_utils.AllPlugins.HandlerHelp)),
		ReplyParameters: &models.ReplyParameters{ MessageID: params.Message.ID },
		ReplyMarkup:     plugin_utils.BuildHandlerHelpKeyboard(),
	})
	if err != nil {
		zerolog.Ctx(params.Ctx).Error().
			Err(err).
			Str(utils.GetCurrentFuncName()).
			Dict(utils.GetChatDict(&params.Message.Chat)).
			Dict(utils.GetUserDict(params.Message.From)).
			Str("content", "bot help keyboard").
			Msg(flaterr.SendMessage.Str())
	}
	return err
}

func helpCallbackHandler(params *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(&params.CallbackQuery.From)).
		Dict(utils.GetChatDict(&params.CallbackQuery.Message.Message.Chat)).
		Logger()

	if strings.HasPrefix(params.CallbackQuery.Data, "help-handler_") {
		handlerName := strings.TrimPrefix(params.CallbackQuery.Data, "help-handler_")
		for _, handler := range plugin_utils.AllPlugins.HandlerHelp {
			if handler.Name == handlerName {
				var replyMarkup models.ReplyMarkup

				// 如果帮助函数有自定的 ReplyMarkup，则使用它，否则显示默认的按钮
				if handler.ReplyMarkup != nil {
					replyMarkup = handler.ReplyMarkup
				} else {
					replyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text:         "返回",
							CallbackData: "help",
						},
						{
							Text:         "关闭",
							CallbackData: "delete_this_message",
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
						Msg(flaterr.EditMessageText.Str())
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
				Msg(flaterr.AnswerCallbackQuery.Str())
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
			Msg(flaterr.EditMessageText.Str())
	}
	return err
}
