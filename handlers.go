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

var currentOptions = []bool{false, false, false}
var forwardonlylist = &forwardMetadata{}

func catchAllHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {

	var botMessage *models.Message // 存放 bot 发送的信息
	// log.Printf("%s send a message: [%s]", update.Message.From.Username, update.Message.Text)
	// fmt.Println(update.Message.Chat.ID)

	var commandFields []string // 存储以 / 开头的命令字段切片，由 strings.Fields() 在后续进行分割再赋值
	var currentChatType = update.Message.Chat.Type // 获取发送请求的聊天类型
	// log.Println(currentChatType)
	if currentChatType == models.ChatTypeChannel {
		thebot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text: "get channel messages!",
		})
	}

	// 首先判断聊天类型，这里处理私聊、群组和超级群组的消息
	if AnyContains(currentChatType, models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 检测如果消息开头是 / 符号，作为命令来处理
		if strings.HasPrefix(update.Message.Text, "/") {
			commandFields = strings.Fields(update.Message.Text) // 对消息开头是 / 符号的命令进行分割，类似命令行参数
			// 预设的多个命令
			if commandMaybeWithSuffixUsername(commandFields, "/start") {
				startHandler(ctx, thebot, update, commandFields)
				return
			} else if commandMaybeWithSuffixUsername(commandFields, "/forwardonly") {
				addToWriteListHandler(ctx, thebot, update)
				return
			} else if commandMaybeWithSuffixUsername(commandFields, "/chatinfo") {
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
					Text: fmt.Sprintf("类型: [<code>%v</code>]\nID: [<code>%v</code>]\n用户名:[<code>%v</code>]", update.Message.Chat.Type, update.Message.Chat.ID, update.Message.Chat.Username),
					ParseMode: models.ParseModeHTML,
				})
				return
			} else if commandMaybeWithSuffixUsername(commandFields, "/test") {
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
					ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{ { {
						Text: "点击加入测试群组",
						URL: "https://t.me/+BomkHuFsjqc3ZGE1",
					}}}},
				})
				return
			} else if strings.HasSuffix(commandFields[0], "@" + botMe.Username) {
				// 注意，此段应该保持在此 if-else 语句的末尾，否则后续的命令将无法触发
				// 为防止与其他 bot 的命令冲突，默认不会处理不在命令列表中的命令
				// 如果消息以 /xxx@examplebot 的形式指定此 bot 回应，且 /xxx 不在预设的命令中时，才发送该命令不可用的提示
				botMessage, _ = thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
					Text:      "不存在的命令",
				})
				time.Sleep(time.Second * 10)
				thebot.DeleteMessages(ctx, &bot.DeleteMessagesParams{
					ChatID:     update.Message.Chat.ID,
					MessageIDs: []int{
						update.Message.ID,
						botMessage.ID,
					},
				})
				return
			}
		}

		// 不符合上方条件，即消息开头不是 / 符号的信息
		if currentChatType == models.ChatTypePrivate {
			// 如果用户发送的是贴纸，下载并返回贴纸源文件给用户
			if update.Message.Sticker != nil {
				// echoStickerHandler(ctx, thebot, update)
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					Text:      "本 bot 获取贴纸文件的功能出了点问题，暂不可用",
					ReplyParameters: &models.ReplyParameters{
						MessageID: update.Message.ID,
					},
				})
				return
			}

			// 不匹配上面项目的则提示不可用
			if strings.HasPrefix(update.Message.Text, "/") {
				// 非冗余条件，在私聊状态下应处理用户发送的所有开头为 / 的命令
				// 与群组中不同，群组中命令末尾不指定此 bot 回应的命令无须处理，以防与群组中的其他 bot 冲突
				botMessage, _ = thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
					Text:      "不存在的命令",
				})
				if private_log { privateLogToChat(ctx, thebot, update) }
			} else {
				// 非命令消息，提示无操作可用
				botMessage, _ = thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:    update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
					Text:      "无操作可用",
				})
				if private_log { privateLogToChat(ctx, thebot, update) }

				// thebot.ForwardMessages(ctx, &bot.ForwardMessagesParams{
				// 	ChatID:     logChat_ID,
				// 	FromChatID: update.Message.Chat.ID,
				// 	MessageIDs: []int{
				// 		update.Message.ID - 1,
				// 		update.Message.ID,
				// 	},
				// })
			}

			// 等待五秒删除请求信息和回复信息
			time.Sleep(time.Second * 10)
			thebot.DeleteMessages(ctx, &bot.DeleteMessagesParams{
				ChatID:     update.Message.Chat.ID,
				MessageIDs: []int{
					update.Message.ID,
					botMessage.ID,
				},
			})
		} else if AnyContains(currentChatType, models.ChatTypeGroup, models.ChatTypeSupergroup) {
			if forwardonlylist != nil {
				// 处理消息删除逻辑，只有当群组启用该功能时才处理
				if fwdonly_IsGroupEnabled(update.Message.Chat.ID, forwardonlylist) && (
					getMessageType(update.Message) == MessageTypeText ||
					getMessageType(update.Message) == MessageTypeVoice ||
					getMessageType(update.Message) == MessageTypeSticker) {
					_, err := thebot.DeleteMessage(ctx, &bot.DeleteMessageParams{
						ChatID:    update.Message.Chat.ID,
						MessageID: update.Message.ID,
					})
					if err != nil {
						log.Printf("Failed to delete message: %v", err)
					} else {
						log.Printf("Deleted message from %d in %d: %s\n", update.Message.From.ID, update.Message.Chat.ID, update.Message.Text)
					}
				}
			}
		}
	}

}

func startHandler(ctx context.Context, thebot *bot.Bot, update *models.Update, fields []string) {
	if strings.HasPrefix(fields[1], "via-inline") {
		inlineArgument := strings.Split(fields[1], "_")
		if inlineArgument[1] == "test" {
			thebot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text: "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
				ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{ { {
					Text: "点击加入测试群组",
					URL: "https://t.me/+BomkHuFsjqc3ZGE1",
				}}}},
			})
		}
	} else {
		thebot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      fmt.Sprintf("Hello, *%s %s*\n\nThis robot doesn't currently support chat mode, please use [inline mode](https://telegram.org/blog/inline-bots?setln=en) for interactive operations.", update.Message.From.FirstName, update.Message.From.LastName),
			ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
			LinkPreviewOptions: &models.LinkPreviewOptions{ PreferSmallMedia: bot.True() },
			ParseMode: models.ParseModeMarkdownV1,
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

func inlinehandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	// log.Println("inlinehandler working")
	if update.InlineQuery == nil { return }

	log.Printf("inline from: [%s], query: [%s]", update.InlineQuery.From.Username, update.InlineQuery.Query)

	if update.InlineQuery.Query == "log" && ( update.InlineQuery.From.ID == 2074319561 || update.InlineQuery.From.ID == 1086395364 )  {
		logs := readLog()
		if logs != nil {
			log_count := len(logs)
			var log_all string
			for index, log := range logs {
				log_all = fmt.Sprintf("%s\n%02d %s", log_all, index, log)
				// log_all = log_all + "\n" + index + log
			}
			_, err := thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: update.InlineQuery.ID,
				Results: []models.InlineQueryResult{
					&models.InlineQueryResultArticle{
						ID:    "log",
						Title: fmt.Sprintf("%d logs update at %s", log_count, time.Now().Format("2006/01/02 15:04:05")),
						InputMessageContent: &models.InputTextMessageContent{
							MessageText: fmt.Sprintf("last update at %s\n%s", time.Now().Format("2006/01/02 15:04:05"), log_all),
							ParseMode: models.ParseModeMarkdownV1,
						},
					},
				},
				CacheTime: 0,
			})
			if err != nil {
				log.Println("Error when answering inline query", err)
			}
		} else {
			log.Println("Error when reading log file")
		}
		return
	}

	metadataList, err := readMetadataFromDir(voice_path)
	if err != nil {
		// if errors.Is(e)
		log.Printf("Error when reading metadata files: %v", err)
		thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{
				&models.InlineQueryResultVoice{
					ID:       "none",
					Title:    "读取语音文件时发生错误，请联系机器人管理员",
					Caption:  "由于无法读取到语音文件，此处被替换为预设的 `♿otto: 我是说的道理~` ",
					VoiceURL: "https://otto-hzys.huazhiwan.xyz/static/ysddTokens/wssddl.mp3",
					ParseMode: models.ParseModeMarkdownV1,
				},
			},
			CacheTime: 0,
		})
		thebot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    logChat_ID,
			Text:      fmt.Sprintf("%s\nInline Mode: some user get error", time.Now().String()),
		})
		return
	} else if metadataList == nil {
		log.Printf("No voices file in voices_path: %s", voice_path)
		thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{
				&models.InlineQueryResultVoice{
					ID:       "none",
					Title:    "无法读取到语音文件，请联系机器人管理员",
					Caption:  "由于无法读取到语音文件，此处被替换为预设的 `♿otto: 我是说的道理~` ",
					VoiceURL: "https://otto-hzys.huazhiwan.xyz/static/ysddTokens/wssddl.mp3",
					ParseMode: models.ParseModeMarkdownV1,
				},
			},
			CacheTime: 0,
		})
		thebot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    logChat_ID,
			Text:      fmt.Sprintf("%s\nInline Mode: some user can't load voices", time.Now().String()),
		})
		return
	}

	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult

	if update.InlineQuery.Query == "" {
		for _, metadata := range metadataList {
			for _, voice := range metadata.Voices {
				result := &models.InlineQueryResultVoice{
					ID:       voice.ID,
					Title:    metadata.VoicesName + ": " + voice.Title,
					Caption:  voice.Caption,
					VoiceURL: voice.VoiceURL,
				}
				results = append(results, result)
			}
		}
	} else {
		for _, metadata := range metadataList {
			for _, voice := range metadata.Voices {
				if AnyContains(update.InlineQuery.Query, metadata.VoicesName, voice.Title, voice.Caption) {
				// if strings.ContainsAny(metadata.VoicesName, update.InlineQuery.Query) || strings.ContainsAny(voice.Title, update.InlineQuery.Query) || strings.ContainsAny(voice.Caption, update.InlineQuery.Query) {
					result := &models.InlineQueryResultVoice{
						ID:       voice.ID,
						Title:    metadata.VoicesName + ": " + voice.Title,
						Caption:  voice.Caption,
						VoiceURL: voice.VoiceURL,
					}
				results = append(results, result)
				}
			}
		}
		if len(results) == 0 {
			results = append(results, &models.InlineQueryResultArticle{
				ID:       "none",
				Title:    fmt.Sprintf("🈚 没有找到包含 %s 的内容", update.InlineQuery.Query),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "什么都没有",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}

	_, err = thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: update.InlineQuery.ID,
		Results:       results,
		CacheTime:     0,
		Button: &models.InlineQueryResultsButton{
			Text: "一个测试用的按钮",
			StartParameter: "via-inline_test",
		},
	})
	if err != nil {
		log.Printf("Error sending inline query response: %v", err)
		return
	}
}

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
