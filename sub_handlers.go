package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func startHandler(opts *subHandlerOpts) {
	if len(opts.fields) > 1 && strings.HasPrefix(opts.fields[1], "via-inline") {
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
		}
	} else {
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID:    opts.update.Message.Chat.ID,
			Text:      fmt.Sprintf("Hello, *%s %s*\n\nThis robot doesn't currently support chat mode, please use [inline mode](https://telegram.org/blog/inline-bots?setln=en) for interactive operations.", opts.update.Message.From.FirstName, opts.update.Message.From.LastName),
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			LinkPreviewOptions: &models.LinkPreviewOptions{ PreferSmallMedia: bot.True() },
			ParseMode: models.ParseModeMarkdownV1,
		})
	}
}

func addToWriteListHandler(opts *subHandlerOpts) {
	if opts.update.Message.Chat.Type == "private" {
		botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text:   "仅限转发模式被设计为仅在群组中可用",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
		})
		time.Sleep(time.Second * 10)
		opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			ChatID:     opts.update.Message.Chat.ID,
			MessageIDs: []int{
				opts.update.Message.ID,
				botMessage.ID,
			},
		})
	} else if userIsAdmin(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, opts.update.Message.From.ID) {
		if !opts.chatInfo.IsEnableForwardonly && strings.HasSuffix(opts.update.Message.Text, fmt.Sprint(opts.update.Message.Chat.ID)) {
			if opts.chatInfo.ID != opts.update.Message.Chat.ID {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "发送的群组 ID 与当前群组的 ID 不符，请先运行 `/forwardonly`",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.chatInfo.IsEnableForwardonly = true
				savenow <- true
				log.Println("Turn forwardonly on for group", opts.update.Message.Chat.ID)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "仅限转发模式已启用",
					ParseMode: models.ParseModeMarkdownV1,
				})
			}
		} else if opts.update.Message.Text == "/forwardonly disable" {
			if !opts.chatInfo.IsEnableForwardonly {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "此群组并没有开启仅限转发模式哦",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.chatInfo.IsEnableForwardonly = false
				savenow <- true
				log.Println("Turn forwardonly off for group", opts.update.Message.Chat.ID)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   fmt.Sprintf("仅限转发模式已关闭，重新启用请发送 `/forwardonly %d`", opts.update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
			}
		} else if strings.HasPrefix(opts.update.Message.Text, "/forwardonly") {
			if userIsAdmin(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, botMe.ID) && userHavePermissionDeleteMessage(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, botMe.ID) {
				if opts.chatInfo.IsEnableForwardonly {
					botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
						ChatID: opts.update.Message.Chat.ID,
						Text:   "仅限转发模式已启用，无须重复开启，若要关闭，请发送 `/forwardonly disable` 来关闭它",
						ParseMode: models.ParseModeMarkdownV1,
					})
					time.Sleep(time.Second * 5)
					opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
						ChatID:     opts.update.Message.Chat.ID,
						MessageIDs: []int{
							opts.update.Message.ID,
							botMessage.ID,
						},
					})
					return
				}
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   fmt.Sprintf("请求已确定，继续发送 `/forwardonly %d` 以启用仅限转发模式", opts.update.Message.Chat.ID),
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
			ChatID:     opts.update.Message.Chat.ID,
			MessageIDs: []int{
				opts.update.Message.ID,
				botMessage.ID,
			},
		})
	}
}

func echoStickerHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	// 下载 webp 格式的贴纸
	file, err := thebot.GetFile(ctx, &bot.GetFileParams{ FileID: update.Message.Sticker.FileID })
	if err != nil { log.Printf("Error getting file: %v", err) }
	if update.Message.Sticker.IsVideo {
		thebot.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:   update.Message.Chat.ID,
			Caption: "see [wikipedia/WebM](https://wikipedia.org/wiki/WebM)",
			Document: &models.InputFileUpload{Filename: "sticker.webm", Data: echoSticker(file.FilePath)},
			// Document: &models.InputFileString{Data: file.FilePath},
			ParseMode: models.ParseModeMarkdownV1,
		})
	} else if update.Message.Sticker.IsAnimated {
		thebot.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:   update.Message.Chat.ID,
			Caption: "see [stickers/animated-stickers](https://core.telegram.org/stickers#animated-stickers)",
			Document: &models.InputFileUpload{Filename: "sticker.tgs.file", Data: echoSticker(file.FilePath)},
			ParseMode: models.ParseModeMarkdownV1,
		})
	} else {
		thebot.SendDocument(ctx, &bot.SendDocumentParams{
			ChatID:   update.Message.Chat.ID,
			Caption: "see [wikipedia/WebP](https://wikipedia.org/wiki/WebP)",
			Document: &models.InputFileUpload{Filename: "sticker.webp.png", Data: echoSticker(file.FilePath)},
			// Document: &models.InputFileString{ Data: update.Message.Sticker.FileID }, // 没法以文件形式发送
			ParseMode: models.ParseModeMarkdownV1,
		})
	}
}

var currentOptions = []bool{false, false, false}

func callbackHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	// answering callback query first to let Telegram know that we received the callback query,
	// and we're handling it. Otherwise, Telegram might retry sending the update repetitively
	// as it thinks the callback query doesn't reach to our application. learn more by
	// reading the footnote of the https://core.telegram.org/bots/api#callbackquery type.
	thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	switch update.CallbackQuery.Data {
	case "btn_opt1":
		currentOptions[0] = !currentOptions[0]
	case "btn_opt2":
		currentOptions[1] = !currentOptions[1]
	case "btn_opt3":
		currentOptions[2] = !currentOptions[2]
	case "btn_select":
		thebot.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:   fmt.Sprintf("Selected options: %v", currentOptions),
		})
		// b.SendMessage(ctx, &bot.SendMessageParams{
		// 	ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		// })
		return
	}

	thebot.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ReplyMarkup: buildKeyboard(),
	})
}

func buildKeyboard() models.ReplyMarkup {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: buttonText("Option 1", currentOptions[0]), CallbackData: "btn_opt1"},
				{Text: buttonText("Option 2", currentOptions[1]), CallbackData: "btn_opt2"},
				{Text: buttonText("Option 3", currentOptions[2]), CallbackData: "btn_opt3"},
			},
			{
				{Text: "Select", CallbackData: "btn_select"},
			},
		},
	}

	return kb
}

func buttonText(text string, opt bool) string {
	if opt { return "✅ " + text }
	return "❌ " + text
}

func commandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Select multiple options",
		ReplyMarkup: buildKeyboard(),
	})
}
