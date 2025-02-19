package main

import (
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func startHandler(opts *subHandlerOpts) {
	if len(opts.fields) > 1 {
		if strings.HasPrefix(opts.fields[1], "via-inline") {
			inlineArgument := strings.Split(opts.fields[1], "_")
			if inlineArgument[1] == "test" {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text: "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "点击加入测试群组",
						URL: "https://t.me/+BomkHuFsjqc3ZGE1",
					}}}},
				})
			} else if inlineArgument[1] == "noreply" {
				return
			} else if inlineArgument[1] == "savedmessage-help" {
				saveMessageHandler(opts)
				return
			}
			return
		} else if opts.fields[1] == "savedmessage_privacy_policy" {
			SendPrivacyPolicy(opts)
			return
		} else if opts.fields[1] == "savedmessage_privacy_policy_agree" {
			AgreePrivacyPolicy(opts)
			return
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
