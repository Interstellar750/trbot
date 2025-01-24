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
	fields   []string // 根据请求的类型，可能是消息文本，也可能是 inline 的 query
}

func defaultHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	
	var opts = subHandlerOpts{
		ctx:      ctx,
		thebot:   thebot,
		update:   update,
	}

	if update.Message != nil { // 消息
		opts.fields = strings.Fields(update.Message.Text)
		opts.chatInfo, opts.DBIndex = getIDInfoAndIndex(&update.Message.Chat.ID)

		if opts.DBIndex == -1 && AddChatInfo(&update.Message.Chat) {
			log.Printf("add (message)%s \"%s\"[%d] in database", update.Message.Chat.Type, showChatName(&update.Message.Chat), update.Message.Chat.ID)
			opts.chatInfo, opts.DBIndex = getIDInfoAndIndex(&update.Message.Chat.ID)
		}

		opts.chatInfo.MessageCount++
		log.Printf("message from: \"%s\"(%s)[%d], message: [%s]", showUserName(update.Message.From), update.Message.From.Username, update.Message.From.ID, update.Message.Text)

		messageHandler(&opts)
	} else if update.EditedMessage != nil { // 私聊或群组消息被编辑
		log.Printf("edited from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], edited message [%d] to [%s]", 
			showUserName(update.EditedMessage.From), update.EditedMessage.From.Username, update.EditedMessage.From.ID,
			showChatName(&update.EditedMessage.Chat), update.EditedMessage.Chat.Username, update.EditedMessage.Chat.ID,
			update.EditedMessage.ID, update.EditedMessage.Text,
		)
	} else if update.InlineQuery != nil { // inline 查询
		opts.fields = strings.Fields(update.InlineQuery.Query)
		opts.chatInfo, opts.DBIndex = getIDInfoAndIndex(&update.InlineQuery.From.ID)

		if opts.DBIndex == -1 && AddUserInfo(update.InlineQuery.From) {
			log.Printf("add (inline)private \"%s\"[%d] in database", update.InlineQuery.From.Username, update.InlineQuery.From.ID)
			opts.chatInfo, opts.DBIndex = getIDInfoAndIndex(&update.InlineQuery.From.ID)
		}

		opts.chatInfo.InlineCount++
		log.Printf("inline from: \"%s\"(%s)[%d], query: [%s]", 
			showUserName(update.InlineQuery.From), update.InlineQuery.From.Username, update.InlineQuery.From.ID,
			update.InlineQuery.Query,
		)

		inlineHandler(&opts)
	} else if update.ChosenInlineResult != nil { // inline 查询结果被选择
		log.Printf("chosen inline from \"%s\"(%s)[%d], ID: [%s] query: [%s]",
			showUserName(&update.ChosenInlineResult.From), update.ChosenInlineResult.From.Username, update.ChosenInlineResult.From.ID,
			update.ChosenInlineResult.ResultID, update.ChosenInlineResult.Query,
		)
	} else if update.CallbackQuery != nil { // replymarkup 回调
		log.Printf("callback from \"%s\"(%s)[%d] in \"%s\"(%s)[%d] query: [%s]",
			showUserName(&update.CallbackQuery.From), update.CallbackQuery.From.Username, update.CallbackQuery.From.ID,
			showChatName(&update.CallbackQuery.Message.Message.Chat), update.CallbackQuery.Message.Message.Chat.Username, update.CallbackQuery.Message.Message.Chat.ID,
			update.CallbackQuery.Data,
		)
	} else if update.MessageReaction != nil { // 私聊或群组表情回应
		log.Printf("reaction from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
			showUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
			showChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
			update.MessageReaction.MessageID,
		)
	} else if update.MessageReactionCount != nil { // 频道消息表情回应数量
		log.Printf("reaction count from in \"%s\"(%s)[%d], to message [%d], reactions: %v",
			showChatName(&update.MessageReactionCount.Chat), update.MessageReactionCount.Chat.Username, update.MessageReactionCount.Chat.ID,
			update.MessageReactionCount.MessageID, update.MessageReactionCount.Reactions,
		)
	} else if update.ChannelPost != nil { // 频道信息
		if update.ChannelPost.From != nil { // 在频道中使用户身份发送
			log.Printf("channel post from user \"%s\"(%s)[%d], in \"%s\"(%s)[%d], message [%s]",
				showUserName(update.ChannelPost.From), update.ChannelPost.From.Username, update.ChannelPost.From.ID,
				showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
				update.ChannelPost.Text,
			)
		} else if update.ChannelPost.SenderBusinessBot != nil { // 在频道中由商业 bot 发送
			log.Printf("channel post from businessbot \"%s\"(%s)[%d], in \"%s\"(%s)[%d], message [%s]",
				showUserName(update.ChannelPost.SenderBusinessBot), update.ChannelPost.SenderBusinessBot.Username, update.ChannelPost.SenderBusinessBot.ID,
				showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
				update.ChannelPost.Text,
			)
		} else if update.ChannelPost.ViaBot != nil { // 在频道中由 bot 发送
			log.Printf("channel post from bot \"%s\"(%s)[%d], in \"%s\"(%s)[%d], message [%s]",
				showUserName(update.ChannelPost.ViaBot), update.ChannelPost.ViaBot.Username, update.ChannelPost.ViaBot.ID,
				showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
				update.ChannelPost.Text,
			)
		} else if update.ChannelPost.SenderChat != nil { // 在频道中使用其他频道身份发送
			if update.ChannelPost.SenderChat.ID == update.ChannelPost.Chat.ID { // 在频道中由频道自己发送
				log.Printf("channel post in \"%s\"(%s)[%d], message [%s]",
					showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.Text,
				)
			} else {
				log.Printf("channel post from another channel \"%s\"(%s)[%d], in \"%s\"(%s)[%d], message [%s]",
					showChatName(update.ChannelPost.SenderChat), update.ChannelPost.SenderChat.Username, update.ChannelPost.SenderChat.ID,
					showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.Text,
				)
			}
		} else { // 没有身份信息
			log.Printf("channel post from nobody in \"%s\"(%s)[%d], message [%s]",
				// showUserName(update.ChannelPost.From), update.ChannelPost.From.Username, update.ChannelPost.From.ID,
				showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
				update.ChannelPost.Text,
			)
		}
		return
	} else if update.EditedChannelPost != nil { // 频道中编辑过的消息
		log.Printf("edited channel post in \"%s\"(%s)[%d], message [%s]",
			showChatName(&update.EditedChannelPost.Chat), update.EditedChannelPost.Chat.Username, update.EditedChannelPost.Chat.ID,
			update.EditedChannelPost.Text,
		)
	} else { // 没有加入的
		log.Printf("unknown update type: %v", update)
		// thebot.CopyMessage(ctx, &bot.CopyMessageParams{
		// })
	}
}

// 处理所有信息请求的处理函数，触发条件为任何消息
func messageHandler(opts *subHandlerOpts) {
	var botMessage *models.Message // 存放 bot 发送的信息

	// log.Printf("%s send a message: [%s]", opts.update.Message.From.Username, opts.update.Message.Text)
	// fmt.Println(opts.update.Message.Chat.ID)

	if opts.update.Message.Chat.Type == models.ChatTypeChannel {
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text: "get channel messages!",
		})
	}

	// 首先判断聊天类型，这里处理私聊、群组和超级群组的消息
	if AnyContains(opts.update.Message.Chat.Type, models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 检测如果消息开头是 / 符号，作为命令来处理
		if strings.HasPrefix(opts.update.Message.Text, "/") {
			// 预设的多个命令
			if commandMaybeWithSuffixUsername(opts.fields, "/start") {
				startHandler(opts)
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/forwardonly") {
				addToWriteListHandler(opts)
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/chatinfo") {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					Text: fmt.Sprintf("类型: [<code>%v</code>]\nID: [<code>%v</code>]\n用户名:[<code>%v</code>]", opts.update.Message.Chat.Type, opts.update.Message.Chat.ID, opts.update.Message.Chat.Username),
					ParseMode: models.ParseModeHTML,
				})
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/test") {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text: "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{ { {
						Text: "点击加入测试群组",
						URL: "https://t.me/+BomkHuFsjqc3ZGE1",
					}}}},
				})
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/version") && AnyContains(opts.update.Message.From.ID, logMan_IDs) {
				// info, err := opts.thebot.GetWebhookInfo(ctx)
				// fmt.Println(info)
				// return
				botMessage, _ = opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text: outputVersionInfo(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ParseMode: models.ParseModeMarkdownV1,
				})
				time.Sleep(time.Second * 20)
				opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
					ChatID: opts.update.Message.Chat.ID,
					MessageIDs: []int{
						opts.update.Message.ID,
						botMessage.ID,
					},
				})
				return
			} else if strings.HasSuffix(opts.fields[0], "@" + botMe.Username) {
				// 注意，此段应该保持在此 if-else 语句的末尾，否则后续的命令将无法触发
				// 为防止与其他 bot 的命令冲突，默认不会处理不在命令列表中的命令
				// 如果消息以 /xxx@examplebot 的形式指定此 bot 回应，且 /xxx 不在预设的命令中时，才发送该命令不可用的提示
				botMessage, _ = opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					Text:      "不存在的命令",
				})
				time.Sleep(time.Second * 10)
				opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
					ChatID:     opts.update.Message.Chat.ID,
					MessageIDs: []int{
						opts.update.Message.ID,
						botMessage.ID,
					},
				})
				return
			}
		} else if opts.update.Message.Chat.ID == udonGroupID && len(opts.fields) > 0 {
			udoneseHandler(opts)
			return
		}

		// 不符合上方条件，即消息开头不是 / 符号的信息
		if opts.update.Message.Chat.Type == models.ChatTypePrivate {
			// 如果用户发送的是贴纸，下载并返回贴纸源文件给用户
			if opts.update.Message.Sticker != nil {
				// echoStickerHandler(opts.ctx, thebot, update)

				// opts.thebot.GetStickerSet(opts.ctx, &bot.GetStickerSetParams{
				// 	Name: opts.update.Message.Sticker.SetName,
				// })
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					Text:      "本 bot 获取贴纸文件的功能出了点问题，暂不可用",
					ReplyParameters: &models.ReplyParameters{
						MessageID: opts.update.Message.ID,
					},
				})
				return
			}

			// 不匹配上面项目的则提示不可用
			if strings.HasPrefix(opts.update.Message.Text, "/") {
				// 非冗余条件，在私聊状态下应处理用户发送的所有开头为 / 的命令
				// 与群组中不同，群组中命令末尾不指定此 bot 回应的命令无须处理，以防与群组中的其他 bot 冲突
				botMessage, _ = opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					Text:      "不存在的命令",
				})
				if private_log { privateLogToChat(opts.ctx, opts.thebot, opts.update) }
			} else {
				// 非命令消息，提示无操作可用
				botMessage, _ = opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					Text:      "无操作可用",
				})
				if private_log { privateLogToChat(opts.ctx, opts.thebot, opts.update) }

				// opts.thebot.ForwardMessages(opts.ctx, &bot.ForwardMessagesParams{
				// 	ChatID:     logChat_ID,
				// 	FromChatID: opts.update.Message.Chat.ID,
				// 	MessageIDs: []int{
				// 		opts.update.Message.ID - 1,
				// 		opts.update.Message.ID,
				// 	},
				// })
			}

			// 等待五秒删除请求信息和回复信息
			time.Sleep(time.Second * 10)
			opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
				ChatID:     opts.update.Message.Chat.ID,
				MessageIDs: []int{
					opts.update.Message.ID,
					botMessage.ID,
				},
			})
		} else if AnyContains(opts.update.Message.Chat.Type, models.ChatTypeGroup, models.ChatTypeSupergroup) {
			// 处理消息删除逻辑，只有当群组启用该功能时才处理
			if opts.chatInfo.IsEnableForwardonly && (
				getMessageType(opts.update.Message) == MessageTypeText ||
				getMessageType(opts.update.Message) == MessageTypeVoice ||
				getMessageType(opts.update.Message) == MessageTypeSticker) {
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
}

// 处理 inline 模式下的请求
func inlineHandler(opts *subHandlerOpts) {
	if strings.HasPrefix(opts.update.InlineQuery.Query, InlineSubCommandSymbol) {
		switch opts.fields[0][1:] { // 添加 [1:] 抛弃第一个字符是因为子命令需要一个触发符号
		// 普通命令添加到 switch 的 case 语句中，任何人都能用
		case "uaav":
			if len(opts.fields) < 2 {
				_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.update.InlineQuery.ID,
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
			} else if len(opts.fields) == 2 {
				if strings.HasPrefix(opts.fields[1], "https://") {
					_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultVoice{
								ID: "custom",
								Title: "Custom voice",
								VoiceURL: opts.fields[1],
							},
						},
						IsPersonal: true,
					})
					if err != nil {
						log.Println("Error when answering inline query: ", err)
					}
				} else {
					_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.update.InlineQuery.ID,
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
				_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.update.InlineQuery.ID,
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
		case "sms":
			var udoneseResultList []models.InlineQueryResult
			if len(opts.fields) < 2 || len(opts.fields) == 2 && strings.HasPrefix(opts.fields[len(opts.fields)-1], InlinePaginationSymbol) {
				for _, data := range AdditionalDatas.Udonese.List {
					udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
						ID:    data.Word,
						Title: data.Word,
						Description: fmt.Sprintf("有 %d 个意思: %s...", len(data.MeaningList), data.MeaningList[0].Meaning),
						InputMessageContent: models.InputTextMessageContent{
							MessageText: data.OutputMeanings(),
							ParseMode: models.ParseModeHTML,
						},
					})
				}
			} else {
				for _, data := range AdditionalDatas.Udonese.List {
					// 通过词查找意思
					if InlineQueryMatchMultKeyword(opts.fields, []string{data.Word}, true) {
						udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
							ID:    data.Word,
							Title: data.Word,
							Description: fmt.Sprintf("有 %d 个意思: %s...", len(data.MeaningList), data.MeaningList[0].Meaning),
							InputMessageContent: models.InputTextMessageContent{
								MessageText: data.OutputMeanings(),
								ParseMode: models.ParseModeHTML,
							},
						})
					}
					// 通过意思查找词
					if InlineQueryMatchMultKeyword(opts.fields, data.OnlyMeaning(), true) {
						for _, n := range data.MeaningList {
							if InlineQueryMatchMultKeyword(opts.fields, []string{n.Meaning}, true) {
								udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
									ID:    n.Meaning,
									Title: n.Meaning,
									Description: fmt.Sprintf("%s 对应的词是 %s", n.Meaning, data.Word),
									InputMessageContent: models.InputTextMessageContent{
										MessageText: fmt.Sprintf("%s 对应的词是 <code>%s</code>", n.Meaning, data.Word),
										ParseMode: models.ParseModeHTML,
									},
								})
							}
						}
					}
				}
				if len(udoneseResultList) == 0 {
					udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
						ID:       "none",
						Title:    "没有符合关键词的内容",
						Description: fmt.Sprintf("没有找到包含 %s 的词或意思", opts.fields[1:]),
						InputMessageContent: models.InputTextMessageContent{
							MessageText: "没有这个词，使用 <code>udonese <词> <意思> </code> 来添加吧",
							ParseMode: models.ParseModeHTML,
						},
					})
				}
			}

			_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.update.InlineQuery.ID,
				Results:       InlineResultPagination(opts.fields, udoneseResultList),
			})
			if err != nil {
				log.Println("Error when answering inline :sms command", err)
			}
			return
		default:
			// default 中设定一些管理员命令和无命令提示
			if AnyContains(opts.update.InlineQuery.From.ID, logMan_IDs) {
				if strings.HasPrefix(opts.update.InlineQuery.Query, InlineSubCommandSymbol + "log") {
					logs := readLog()
					if logs != nil {
						log_count := len(logs)
						var log_all string
						for index, log := range logs {
							log_all = fmt.Sprintf("%s\n%02d %s", log_all, index, log)
						}
						_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
							InlineQueryID: opts.update.InlineQuery.ID,
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
							IsPersonal: true,
							CacheTime: 0,
						})
						if err != nil {
							log.Println("Error when answering inline query :log", err)
						}
					} else {
						log.Println("Error when reading log file")
					}
					return
				} else if strings.HasPrefix(opts.update.InlineQuery.Query, InlineSubCommandSymbol + "reload") {
					ADR_reload <- true
					_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:    "reload",
								Title: "已请求更新",
								Description: fmt.Sprintf("last update at %s", time.Now().Format(time.RFC3339)),
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: "???",
									ParseMode: models.ParseModeMarkdownV1,
								},
							},
						},
						IsPersonal: true,
						CacheTime: 0,
					})
					if err != nil {
						log.Println("Error when answering inline query :reload", err)
					}
					return
				} else if strings.HasPrefix(opts.update.InlineQuery.Query, InlineSubCommandSymbol + "savedb") {
					DB_savenow <- true
					_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:    "savedb",
								Title: "已请求保存",
								Description: fmt.Sprintf("last update at %s", time.Now().Format(time.RFC3339)),
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: "???",
									ParseMode: models.ParseModeMarkdownV1,
								},
							},
						},
						IsPersonal: true,
						CacheTime: 0,
					})
					if err != nil {
						log.Println("Error when answering inline query :savedb", err)
					}
					return
				}
			}
			_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.update.InlineQuery.ID,
				Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:    "noinlinecommand",
					Title: fmt.Sprintf("不存在的命令 [%s]", opts.fields[0]),
					Description: "请检查命令是否正确",
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
						ParseMode: models.ParseModeMarkdownV1,
					},
				}},
			})
			if err != nil {
				log.Println("Error when answering inline no command", err)
			}
			return
		}
	}

	if AdditionalDatas.VoiceErr != nil {
		log.Printf("Error when reading metadata files: %v", AdditionalDatas.VoiceErr)
		opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.update.InlineQuery.ID,
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
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: logChat_ID,
			Text:   fmt.Sprintf("%s\nInline Mode: some user get error， %v", time.Now().Format(time.RFC3339), AdditionalDatas.VoiceErr),
		})
		return
	} else if AdditionalDatas.Voices == nil {
		log.Printf("No voices file in voices_path: %s", voice_path)
		opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.update.InlineQuery.ID,
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
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID:    logChat_ID,
			Text:      fmt.Sprintf("%s\nInline Mode: some user can't load voices", time.Now().Format(time.RFC3339)),
		})
		return
	}

	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult

	// 没有查询字符串或使用分页搜索符号，返回所有结果
	if opts.update.InlineQuery.Query == "" || len(opts.fields) == 1 && strings.HasPrefix(opts.fields[len(opts.fields)-1], InlinePaginationSymbol) {
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
				if InlineQueryMatchMultKeyword(opts.fields, []string{voicePack.Name, voice.Title, voice.Caption}, false) {
					results = append(results, &models.InlineQueryResultVoice{
						ID:       voice.ID,
						Title:    voicePack.Name + ": " + voice.Title,
						Caption:  voice.Caption,
						VoiceURL: voice.VoiceURL,
					})
				}
			}
		}
		if len(results) == 0 {
			results = append(results, &models.InlineQueryResultArticle{
				ID:    "none",
				Title: "没有符合关键词的内容",
				Description: fmt.Sprintf("没有找到包含 %s 的内容", opts.update.InlineQuery.Query),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}

	// fmt.Println(opts.fields, len(results))

	_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: opts.update.InlineQuery.ID,
		Results:       InlineResultPagination(opts.fields, results),
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
