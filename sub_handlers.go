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
			LinkPreviewOptions: &models.LinkPreviewOptions{ IsDisabled: bot.True() },
			ParseMode: models.ParseModeMarkdownV1,
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "Try Inline Mode",
				SwitchInlineQueryCurrentChat: " ",
			}}}},
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
					Text:   "发送的群组 ID 与当前群组的 ID 不符，请先发送 `/forwardonly`",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.chatInfo.IsEnableForwardonly = true
				log.Println("Turn forwardonly on for group", opts.update.Message.Chat.ID)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "仅限转发模式已启用",
					ParseMode: models.ParseModeMarkdownV1,
				})
				DB_savenow <- true
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
				log.Println("Turn forwardonly off for group", opts.update.Message.Chat.ID)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   fmt.Sprintf("仅限转发模式已关闭，重新启用请发送 `/forwardonly %d`", opts.update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
				DB_savenow <- true
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

func echoStickerHandler(opts *subHandlerOpts) {
	// 下载 webp 格式的贴纸
	fmt.Println(opts.update.Message.Sticker)

	stickerdata, err := echoSticker(opts)
	if err != nil {
		log.Println("Error downloading sticker:", err)
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text:   "下载贴纸时发生了一些错误",
			ParseMode: models.ParseModeMarkdownV1,
		})
	}

	if stickerdata == nil {
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text:   "下载贴纸时发生了一些错误",
			ParseMode: models.ParseModeMarkdownV1,
		})
		return
	}

	documentParams := &bot.SendDocumentParams{
		ChatID:   opts.update.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdownV1,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "下载贴纸包中的静态贴纸", CallbackData: fmt.Sprintf("S_%s", opts.update.Message.Sticker.SetName)},
			},
			{
				{Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", opts.update.Message.Sticker.SetName)},
			},
		}},
	}

	if opts.update.Message.Sticker.IsVideo {
		documentParams.Caption  = "see [wikipedia/WebM](https://wikipedia.org/wiki/WebM)"
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.webm", Data: stickerdata}
	} else if opts.update.Message.Sticker.IsAnimated {
		documentParams.Caption  = "see [stickers/animated-stickers](https://core.telegram.org/stickers#animated-stickers)"
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.tgs.file", Data: stickerdata}
	} else {
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.png", Data: stickerdata}
	}

	opts.thebot.SendDocument(opts.ctx, documentParams)
	
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

func udoneseHandler(opts *subHandlerOpts) {
	udon, err := AdditionalDatas.Udonese, AdditionalDatas.UdoneseErr
	if err != nil {
		log.Println("some error in while read udonese list: ", err)
	}

	// 统计词使用次数
	for i, n := range udon.OnlyWord() {
		if n == opts.update.Message.Text || strings.HasPrefix(opts.update.Message.Text, n) {
			udon.List[i].Used++
			err = SaveYamlDB(udon_path, metadataFileName, *udon)
			if err != nil {
				log.Println("get some error when add used count:", err)
			}
			// fmt.Println(udon.List[i].Word, "+1", udon.List[i].Used)
		} 
	}

	if opts.fields[0] == "sms" {
		// 参数过少，提示用法
		if len(opts.fields) < 2 {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				Text:   "使用方法：发送 `sms <词>` 来查看对应的意思",
				ParseMode: models.ParseModeMarkdownV1,
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击浏览全部词与意思",
					SwitchInlineQueryCurrentChat: ":sms ",
				}}}},
			})
			return
		}

		// 在数据库循环查找这个词
		for _, word := range udon.List {
			if strings.EqualFold(word.Word, opts.fields[1]) && len(word.MeaningList) > 0 {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ParseMode: models.ParseModeHTML,
				})
				return
			}
		}

		// 到这里就是没找到，提示没有
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
		})
		return
	} else if opts.fields[0] == "udonese" {
		if len(opts.fields) < 2 {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID:    opts.update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				Text: "使用 `udonese <词> <单个意思>` 来添加记录",
				ParseMode: models.ParseModeMarkdownV1,
			})
			return
		}

		meaning := strings.TrimSpace(opts.update.Message.Text[len(opts.fields[0])+len(opts.fields[1])+2:])

		var fromName string
		var fromID   int64
		var viaName  string
		var viaID    int64

		if opts.update.Message.ReplyToMessage == nil {
			fromName = showUserName(opts.update.Message.From)
			fromID = opts.update.Message.From.ID
		} else {
			fromName = showUserName(opts.update.Message.ReplyToMessage.From)
			fromID = opts.update.Message.ReplyToMessage.From.ID
			viaName = showUserName(opts.update.Message.From)
			viaID = opts.update.Message.From.ID
		}

		var pendingMessage string
		var botMessage *models.Message

		oldMeaning := addUdonese(udon, &UdoneseWord{
			Word: opts.fields[1],
			MeaningList: []UdoneseMeaning{ {
				Meaning: meaning,
				FromID: fromID,
				FromName: fromName,
				ViaID: viaID,
				ViaName: viaName,
			}},
		})
		if oldMeaning != nil {
			pendingMessage += fmt.Sprintf("[%s] 意思已存在于 [%s] 中:\n", meaning, oldMeaning.Word)
			for i, s := range oldMeaning.MeaningList {
				if meaning == s.Meaning {
					if s.ViaID != 0 { // 通过回复添加
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] From <a href=\"https://t.me/@id%d\">%s</a> Via <a href=\"https://t.me/@id%d\">%s</a>\n",
							i + 1, s.Meaning, s.FromID, s.FromName, s.ViaID, s.ViaName,
						)
					} else if s.FromID == 0 { // 有添加人信息
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] From <a href=\"https://t.me/@id%d\">%s</a>\n",
							i + 1, s.Meaning, s.FromID, s.FromName,
						)
					} else { // 只有意思
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s]\n", i + 1, s.Meaning)
					}
				}
			}
		} else {
			err = SaveYamlDB(udon_path, metadataFileName, *udon)
			if err != nil {
				pendingMessage += fmt.Sprintln("保存语句时似乎发生了一些错误:\n", err)
			} else {
				pendingMessage += fmt.Sprintf("已添加 [<code>%s</code>]\n", opts.fields[1])
				if viaID != 0 { // 通过回复添加
					pendingMessage += fmt.Sprintf("[%s] From <a href=\"https://t.me/@id%d\">%s</a> Via <a href=\"https://t.me/@id%d\">%s</a>\n",
						meaning, fromID, fromName, viaID, viaName,
					)
				} else if fromID != 0 { // 普通命令添加
					pendingMessage += fmt.Sprintf("[%s] From <a href=\"https://t.me/@id%d\">%s</a>\n",
						meaning, fromID, fromName,
					)
				}
			}
		}

		pendingMessage += fmt.Sprintln("<blockquote>发送的消息与此消息将在十秒后删除</blockquote>")
		botMessage, _= opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text: pendingMessage,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			ParseMode: models.ParseModeHTML,
		})
		if err == nil {
			time.Sleep(time.Second * 10)
			opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
				ChatID: opts.update.Message.Chat.ID,
				MessageIDs: []int{
					opts.update.Message.ID,
					botMessage.ID,
				},
			})
		}
		return
	} else if len(opts.fields) > 1 && strings.HasSuffix(opts.update.Message.Text, "ssm") {
		// 在数据库循环查找这个词
		for _, word := range udon.List {
			if strings.EqualFold(word.Word, opts.fields[1]) && len(word.MeaningList) > 0 {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ParseMode: models.ParseModeHTML,
				})
				return
			}
		}

		// 到这里就是没找到，提示没有
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
		})
	}
}
