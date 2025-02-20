package main

import (
	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Plugin_dashStart struct {
	handler           []dashStartHandler
	withPrefixHandler []dashStartWithPrefixHandler
}

type dashStartHandler struct {
	argument string
	handler  func(*subHandlerOpts)
}

type dashStartWithPrefixHandler struct {
	prefix   string
	argument string
	handler  func(*subHandlerOpts)
}

var dashStart = Plugin_dashStart{}

func addPluginHandlers() {
	dashStart.handler = []dashStartHandler{
		{
			"savedmessage_privacy_policy",
			SendPrivacyPolicy,
		},
		{
			"savedmessage_privacy_policy_agree",
			AgreePrivacyPolicy,
		},
	}
	dashStart.withPrefixHandler = []dashStartWithPrefixHandler{
		{
			"via-inline",
			"test",
			func(opts *subHandlerOpts) {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:          opts.update.Message.Chat.ID,
					Text:            "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
					ReplyParameters: &models.ReplyParameters{MessageID: opts.update.Message.ID},
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "点击加入测试群组",
						URL:  "https://t.me/+BomkHuFsjqc3ZGE1",
					}}}},
				})
			},
		},
		{
			"via-inline",
			"noreply",
			nil,
		},
		{
			"via-inline",
			"savedmessage-help",
			saveMessageHandler,
		},
	}

	// dashStartWithPrefixHandlers := []dashStartWithPrefixHandler{}

	// for _, handler := range dashStartHandlers {
	// 	pendingdashStartHandlers = append(pendingdashStartHandlers, handler)
	// }
}

// func initPlugins() {
// 	for _, handler := range pendingdashStartHandlers {
// 		dashStart.start = append(dashStart.start, handler)
// 	}
// 	for _, handler := range pendingdashStartWithPrefixHandlers {
// 		dashStart.startWithPrefix = append(dashStart.startWithPrefix, handler)
// 	}
// }
