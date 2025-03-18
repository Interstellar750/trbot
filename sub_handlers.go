package main

import (
	"fmt"
	"strings"
	"trbot/utils/handler_utils"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func startHandler(opts *handler_utils.SubHandlerOpts) {
	if len(opts.Fields) > 1 {
		for _, n := range plugin_utils.AllPugins.SlashStart.WithPrefixHandler {
			if strings.HasPrefix(opts.Fields[1], n.Prefix) {
				inlineArgument := strings.Split(opts.Fields[1], "_")
				if inlineArgument[1] == n.Argument {
					n.Handler(opts)
					return
				}
			}
		}
		for _, n := range plugin_utils.AllPugins.SlashStart.Handler {
			if opts.Fields[1] == n.Argument {
				n.Handler(opts)
				return
			}
		}
	}

	opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:    opts.Update.Message.Chat.ID,
		Text:      fmt.Sprintf("Hello, *%s %s*\n\n您可以向此处发送一个贴纸，您将会得到一张转换后的 png 图片\n\n您也可以使用 [inline](https://telegram.org/blog/inline-bots?setln=en) 模式进行交互，点击下方的按钮来使用它", opts.Update.Message.From.FirstName, opts.Update.Message.From.LastName),
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
		LinkPreviewOptions: &models.LinkPreviewOptions{ IsDisabled: bot.True() },
		ParseMode: models.ParseModeMarkdownV1,
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "尝试 Inline 模式",
			SwitchInlineQueryCurrentChat: " ",
		}}}},
	})
}
