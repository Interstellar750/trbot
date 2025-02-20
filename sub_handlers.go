package main

import (
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func startHandler(opts *subHandlerOpts) {
	if len(opts.fields) > 1 {
		for _, n := range dashStart.withPrefixHandler {
			if strings.HasPrefix(opts.fields[1], n.prefix) {
				inlineArgument := strings.Split(opts.fields[1], "_")
				if inlineArgument[1] == n.argument {
					n.handler(opts)
					return
				}
			}
		}
		for _, n := range dashStart.handler {
			if opts.fields[1] == n.argument {
				n.handler(opts)
				return
			}
		}
	}

	opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
		ChatID:    opts.update.Message.Chat.ID,
		Text:      fmt.Sprintf("Hello, *%s %s*\n\n您可以向此处发送一个贴纸，您将会得到一张转换后的 png 图片\n\n您也可以使用 [inline](https://telegram.org/blog/inline-bots?setln=en) 模式进行交互，点击下方的按钮来使用它", opts.update.Message.From.FirstName, opts.update.Message.From.LastName),
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
		LinkPreviewOptions: &models.LinkPreviewOptions{ IsDisabled: bot.True() },
		ParseMode: models.ParseModeMarkdownV1,
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "尝试 Inline 模式",
			SwitchInlineQueryCurrentChat: " ",
		}}}},
	})
}
