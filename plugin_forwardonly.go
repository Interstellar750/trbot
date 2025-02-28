package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func forwardOnlyModeHandler(opts *subHandlerOpts) {
	if opts.update.Message.Chat.Type == "private" {
		botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID:          opts.update.Message.Chat.ID,
			Text:            "仅限转发模式被设计为仅在群组中可用",
			ReplyParameters: &models.ReplyParameters{MessageID: opts.update.Message.ID},
		})
		time.Sleep(time.Second * 10)
		opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			ChatID: opts.update.Message.Chat.ID,
			MessageIDs: []int{
				opts.update.Message.ID,
				botMessage.ID,
			},
		})
	} else if userIsAdmin(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, opts.update.Message.From.ID) {
		if !opts.chatInfo.IsEnableForwardonly && strings.HasSuffix(opts.update.Message.Text, fmt.Sprint(opts.update.Message.Chat.ID)) {
			if opts.chatInfo.ID != opts.update.Message.Chat.ID {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					Text:      "发送的群组 ID 与当前群组的 ID 不符，请先发送 `/forwardonly`",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.chatInfo.IsEnableForwardonly = true
				log.Println("Turn forwardonly on for group", opts.update.Message.Chat.ID)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					Text:      "仅限转发模式已启用",
					ParseMode: models.ParseModeMarkdownV1,
				})
				SignalsChannel.Database_save <- true
			}
		} else if opts.update.Message.Text == "/forwardonly disable" {
			if !opts.chatInfo.IsEnableForwardonly {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					Text:      "此群组并没有开启仅限转发模式哦",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.chatInfo.IsEnableForwardonly = false
				log.Println("Turn forwardonly off for group", opts.update.Message.Chat.ID)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					Text:      fmt.Sprintf("仅限转发模式已关闭，重新启用请发送 `/forwardonly %d`", opts.update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
				SignalsChannel.Database_save <- true
			}
		} else if strings.HasPrefix(opts.update.Message.Text, "/forwardonly") {
			if userIsAdmin(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, botMe.ID) && userHavePermissionDeleteMessage(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, botMe.ID) {
				if opts.chatInfo.IsEnableForwardonly {
					botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
						ChatID:    opts.update.Message.Chat.ID,
						Text:      "仅限转发模式已启用，无须重复开启，若要关闭，请发送 `/forwardonly disable` 来关闭它",
						ParseMode: models.ParseModeMarkdownV1,
					})
					time.Sleep(time.Second * 5)
					opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
						ChatID: opts.update.Message.Chat.ID,
						MessageIDs: []int{
							opts.update.Message.ID,
							botMessage.ID,
						},
					})
					return
				}
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					Text:      fmt.Sprintf("请求已确定，继续发送 `/forwardonly %d` 以启用仅限转发模式", opts.update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
			} else {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "启用此功能前，请先将机器人设为管理员\n如果还是提示本消息，请检查机器人是否有删除消息的权限",
				})
			}
		}
	} else {
		botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text:   "抱歉，您不是群组的管理员，无法为群组更改此功能",
		})
		time.Sleep(time.Second * 5)
		opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			ChatID: opts.update.Message.Chat.ID,
			MessageIDs: []int{
				opts.update.Message.ID,
				botMessage.ID,
			},
		})
	}
}

func deleteNotForwardMessage(opts *subHandlerOpts) {
	if AnyContains(opts.update.Message.Chat.Type, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 处理消息删除逻辑，只有当群组启用该功能时才处理
		if opts.chatInfo.IsEnableForwardonly && opts.update.Message.ForwardOrigin == nil {
			_, err := opts.thebot.DeleteMessage(opts.ctx, &bot.DeleteMessageParams{
				ChatID:    opts.update.Message.Chat.ID,
				MessageID: opts.update.Message.ID,
			})
			if err != nil {
				log.Printf("Failed to delete message: %v", err)
			} else {
				log.Printf("Deleted message from %d in %d: %s\n", opts.update.Message.From.ID, opts.update.Message.Chat.ID, opts.update.Message.Text)
			}
		}
	}
}
