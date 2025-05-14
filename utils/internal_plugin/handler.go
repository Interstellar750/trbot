package internal_plugin

import (
	"fmt"
	"log"
	"strings"
	"trbot/utils"
	"trbot/utils/handler_structs"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)


func startHandler(opts *handler_structs.SubHandlerParams) {
	defer utils.PanicCatcher("startHandler")

	if len(opts.Fields) > 1 {
		for _, n := range plugin_utils.AllPlugins.SlashStart.WithPrefixHandler {
			if strings.HasPrefix(opts.Fields[1], n.Prefix) {
				inlineArgument := strings.Split(opts.Fields[1], "_")
				if inlineArgument[1] == n.Argument {
					if n.Handler == nil {
						continue
					}
					n.Handler(opts)
					return
				}
			}
		}
		for _, n := range plugin_utils.AllPlugins.SlashStart.Handler {
			if opts.Fields[1] == n.Argument {
				n.Handler(opts)
				return
			}
		}
	}

	opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:             opts.Update.Message.Chat.ID,
		Text:               fmt.Sprintf("Hello, *%s %s*\n\n您可以向此处发送一个贴纸，您会得到一张转换后的 png 图片\n\n您也可以使用 [inline](https://telegram.org/blog/inline-bots?setln=en) 模式进行交互，点击下方的按钮来使用它", opts.Update.Message.From.FirstName, opts.Update.Message.From.LastName),
		ParseMode:          models.ParseModeMarkdownV1,
		ReplyParameters:    &models.ReplyParameters{MessageID: opts.Update.Message.ID},
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: bot.True()},
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text:                         "尝试 Inline 模式",
			SwitchInlineQueryCurrentChat: " ",
		}}}},
	})
}

func helpHandler(opts *handler_structs.SubHandlerParams) {
	defer utils.PanicCatcher("helpHandler")

	opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:             opts.Update.Message.Chat.ID,
		Text:               fmt.Sprintf("当前 bot 中有 %d 个帮助文档", len(plugin_utils.AllPlugins.HandlerHelp)),
		ParseMode:          models.ParseModeMarkdownV1,
		ReplyParameters:    &models.ReplyParameters{MessageID: opts.Update.Message.ID},
		LinkPreviewOptions: &models.LinkPreviewOptions{IsDisabled: bot.True()},
		ReplyMarkup:        plugin_utils.BuildHandlerHelpKeyboard(),
	})
}

func helpCallbackHandler(opts *handler_structs.SubHandlerParams) {
	if opts.Update.CallbackQuery.Data == "help-close" {
		opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
			ChatID:    opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
		})
		return
	} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "help-handler_") {
		handlerName := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "help-handler_")
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

				_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID:      opts.Update.CallbackQuery.Message.Message.Chat.ID,
					MessageID:   opts.Update.CallbackQuery.Message.Message.ID,
					Text:        handler.Description,
					ParseMode:   handler.ParseMode,
					ReplyMarkup: replyMarkup,
				})
				if err != nil {
					log.Println("[helpCallbackHandler] error when build handler help message:",err)
				}
				return
			}
		}
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.Update.CallbackQuery.ID,
			Text:            "您请求查看的帮助页面不存在，可能是机器人管理员已经移除了这个插件",
			ShowAlert:       true,
		})
		if err != nil {
			log.Println("[helpCallbackHandler] error when send no this plugin message:", err)
		}
	}
	
	_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
		ChatID:      opts.Update.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   opts.Update.CallbackQuery.Message.Message.ID,
		Text:        fmt.Sprintf("当前 bot 中有 %d 个帮助文档", len(plugin_utils.AllPlugins.HandlerHelp)),
		ReplyMarkup: plugin_utils.BuildHandlerHelpKeyboard(),
	})
	if err != nil {
		log.Println("[helpCallbackHandler] error when rebuild help keyboard:",err)
	}
}
