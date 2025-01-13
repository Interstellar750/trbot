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

// 调用子处理函数时的传递的参数，避免重复获取
type subHandlerOpts struct {
	ctx      context.Context
	thebot   *bot.Bot
	update   *models.Update
	chatInfo *IDInfo
	DBIndex  int
	fields   []string
}

// 处理所有信息请求的处理函数，触发条件为任何消息
func catchAllHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	var botMessage *models.Message // 存放 bot 发送的信息

	info, index := getIDInfoAndIndex(update.Message.Chat.ID)
	fields := strings.Fields(update.Message.Text)

	if index == -1 && AddChatID(update.Message.Chat) {
		log.Printf("add group [%d] in database", update.Message.Chat.ID)
		info, index = getIDInfoAndIndex(update.Message.Chat.ID)
	}

	var opts = subHandlerOpts{
		ctx:      ctx,
		thebot:   thebot,
		update:   update,
		chatInfo: info,
		DBIndex:  index,
		fields:   fields,
	}

	// log.Printf("%s send a message: [%s]", update.Message.From.Username, update.Message.Text)
	// fmt.Println(update.Message.Chat.ID)

	if update.Message.Chat.Type == models.ChatTypeChannel {
		thebot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text: "get channel messages!",
		})
	}

	// 首先判断聊天类型，这里处理私聊、群组和超级群组的消息
	if AnyContains(update.Message.Chat.Type, models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 检测如果消息开头是 / 符号，作为命令来处理
		if strings.HasPrefix(update.Message.Text, "/") {
			// 预设的多个命令
			if commandMaybeWithSuffixUsername(fields, "/start") {
				startHandler(&opts)
				return
			} else if commandMaybeWithSuffixUsername(fields, "/forwardonly") {
				addToWriteListHandler(&opts)
				return
			} else if commandMaybeWithSuffixUsername(fields, "/chatinfo") {
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
					Text: fmt.Sprintf("类型: [<code>%v</code>]\nID: [<code>%v</code>]\n用户名:[<code>%v</code>]", update.Message.Chat.Type, update.Message.Chat.ID, update.Message.Chat.Username),
					ParseMode: models.ParseModeHTML,
				})
				return
			} else if commandMaybeWithSuffixUsername(fields, "/test") {
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
			} else if commandMaybeWithSuffixUsername(fields, "/version") && AnyContains(update.Message.From.ID, logMan_IDs) {
				botMessage, _ = thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text: outputVersionInfo(),
					ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
					ParseMode: models.ParseModeMarkdownV1,
				})
				time.Sleep(time.Second * 20)
				thebot.DeleteMessages(ctx, &bot.DeleteMessagesParams{
					ChatID: update.Message.Chat.ID,
					MessageIDs: []int{
						update.Message.ID,
						botMessage.ID,
					},
				})
				return
			} else if strings.HasSuffix(fields[0], "@" + botMe.Username) {
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
		} else if len(fields) > 0 && AnyContains(fields[0], "sms", "udonese") && update.Message.Chat.ID == udonGroupID {
			udoneseHandler(&opts)
			return
		}

		// 不符合上方条件，即消息开头不是 / 符号的信息
		if update.Message.Chat.Type == models.ChatTypePrivate {
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
		} else if AnyContains(update.Message.Chat.Type, models.ChatTypeGroup, models.ChatTypeSupergroup) {
			// 处理消息删除逻辑，只有当群组启用该功能时才处理
			if info.IsEnableForwardonly && (
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

// 默认函数，处理 inline 模式下的请求
func inlinehandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	// 不知道为什么用户选择 AnswerInlineQuery 列表中的参数后，还会再次触发这个函数，而且 update 中的 InlineQuery 还正好为空
	if update.InlineQuery == nil {
		log.Println("InlineQuery is nil")
		return
	}

	log.Printf("inline from: [%s], query: [%s]", update.InlineQuery.From.Username, update.InlineQuery.Query)

	if AnyContains(update.InlineQuery.From.ID, logMan_IDs) {
		if update.InlineQuery.Query == "log" {
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
							Title: fmt.Sprintf("%d logs update at %s", log_count, time.Now().Format(time.RFC3339)),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: fmt.Sprintf("last update at %s\n%s", time.Now().Format(time.RFC3339), log_all),
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
		} else if strings.HasPrefix(update.InlineQuery.Query, ":") {
			if strings.HasPrefix(update.InlineQuery.Query, ":uaav") {
				queryFields := strings.Fields(update.InlineQuery.Query)
				if len(queryFields) < 2 {
					_, err := thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:    "custom voices",
								Title: "URL as a voice",
								Description: "接着输入一个音频 URL 来其作为语音样式发送（不会转换格式）",
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
									ParseMode: models.ParseModeMarkdownV1,
								},
							},
						},
					})
					if err != nil {
						printLogAndSave(fmt.Sprintln("some error when answer custom voice tips,", err))
					}
				} else if len(queryFields) == 2 {
					if strings.HasPrefix(queryFields[1], "https://") {
						_, err := thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
							InlineQueryID: update.InlineQuery.ID,
							Results: []models.InlineQueryResult{
								&models.InlineQueryResultVoice{
									ID: "custom",
									Title: "Custom voice",
									VoiceURL: queryFields[1],
								},
							},
						})
						if err != nil {
							log.Println("Error when answering inline query: ", err)
						}
					} else {
						_, err := thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
							InlineQueryID: update.InlineQuery.ID,
							Results: []models.InlineQueryResult{
								&models.InlineQueryResultArticle{
									ID:    "error",
									Title: "音频 URL 格式错误",
									Description: "请确保音频链接以 https:// 作为开头，若填写完整 URL 后此消息依然存在，请检查 URL 是否有效",
									InputMessageContent: &models.InputTextMessageContent{
										MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
										ParseMode: models.ParseModeMarkdownV1,
									},
								},
							},
						})
						if err != nil {
							log.Println("Error when answering inline query", err)
						}
					}
				} else {
					_, err := thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:    "error",
								Title: "参数过多，请注意空格",
								Description: fmt.Sprintf("使用方法：@%s :uaav <单个音频链接>", botMe.Username),
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
									ParseMode: models.ParseModeMarkdownV1,
								},
							},
						},
					})
					if err != nil {
						log.Println("Error when answering inline query", err)
					}
				}
				return
			} else if strings.HasPrefix(update.InlineQuery.Query, ":reload") {
				ADR_reload <- true
			}
		}
	}

	if AdditionalDatas.VoiceErr != nil {
		// if errors.Is(e)
		log.Printf("Error when reading metadata files: %v", AdditionalDatas.VoiceErr)
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
			Text:      fmt.Sprintf("%s\nInline Mode: some user get error", time.Now().Format(time.RFC3339)),
		})
		return
	} else if AdditionalDatas.Voices == nil {
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
			Text:      fmt.Sprintf("%s\nInline Mode: some user can't load voices", time.Now().Format(time.RFC3339)),
		})
		return
	}

	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult

	if update.InlineQuery.Query == "" {
		for _, voicePack := range AdditionalDatas.Voices {
			for _, voice := range voicePack.Voices {
				results = append(results, &models.InlineQueryResultVoice{
					ID:       voice.ID,
					Title:    voicePack.Name + ": " + voice.Title,
					Caption:  voice.Caption,
					VoiceURL: voice.VoiceURL,
				})
			}
		}
	} else {
		for _, voicePack := range AdditionalDatas.Voices {
			for _, voice := range voicePack.Voices {
				if AnyContains(update.InlineQuery.Query, voicePack.Name, voice.Title, voice.Caption) {
				// if strings.ContainsAny(metadata.VoicesName, update.InlineQuery.Query) || strings.ContainsAny(voice.Title, update.InlineQuery.Query) || strings.ContainsAny(voice.Caption, update.InlineQuery.Query) {
					result := &models.InlineQueryResultVoice{
						ID:       voice.ID,
						Title:    voicePack.Name + ": " + voice.Title,
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


	_, err := thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: update.InlineQuery.ID,
		Results:       results,
		CacheTime:     180,
		// Button: &models.InlineQueryResultsButton{
		// 	Text: "一个测试用的按钮",
		// 	StartParameter: "via-inline_test",
		// },
	})
	if err != nil {
		log.Printf("Error sending inline query response: %v", err)
		return
	}
}
