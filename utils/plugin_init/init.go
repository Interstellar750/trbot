package plugin_init

import (
	"fmt"
	"log"
	"strings"
	"time"
	"trbot/database"
	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_utils"
	"trbot/utils/mess"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func RegisterPlugins() {
	// 触发：'/start via-inline_test'
	plugin_utils.AddSlashStartWithPrefixCommandPlugins([]plugin_utils.SlashStartWithPrefixHandler{
		{
			Prefix:   "via-inline",
			Argument: "noreply",
			Handler:  nil, // 不回复
		},
		{
			Prefix:   "via-inline",
			Argument: "test",
			Handler: func(opts *handler_utils.SubHandlerOpts) {
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Update.Message.Chat.ID,
					Text:            "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
					ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "点击加入测试群组",
						URL:  "https://t.me/+BomkHuFsjqc3ZGE1",
					}}}},
				})
			},
		},
		{
			Prefix:   "via-inline",
			Argument: "change-inline-command",
			Handler: func(opts *handler_utils.SubHandlerOpts) {
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					Text:   "选择一个 Inline 模式下的默认命令\n<blockquote>由于缓存原因，您可能需要等一会才能看到更新后的结果</blockquote>",
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				})
			},
		},
	}...)
	plugin_utils.AddCallbackQueryCommandPlugins(plugin_utils.CallbackQuery{
		CommandChar: "inline_default_",
		Handler: func(opts *handler_utils.SubHandlerOpts) {
			if opts.Update.CallbackQuery.Data == "inline_default_none" {
				database.SetCustomFlag(opts.Ctx, opts.Update.CallbackQuery.From.ID, db_struct.DefaultInlinePlugin, "")
				opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
					ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.Update.CallbackQuery.Message.Message.ID,
					ReplyMarkup: utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
				})
			}
			callbackField := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "inline_default_")
			for _, inlinePlugin := range plugin_utils.AllPlugins.InlineCommandList {
				if inlinePlugin.Command == callbackField {
					database.SetCustomFlag(opts.Ctx, opts.Update.CallbackQuery.From.ID, db_struct.DefaultInlinePlugin, callbackField)
					opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
						ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
						MessageID: opts.Update.CallbackQuery.Message.Message.ID,
						ReplyMarkup: utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
					})
					break
				}
			}
			consts.SignalsChannel.Database_save <- true
		},
	})

	// 文本消息开头的命令
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "chatinfo",
		Handler: func(opts *handler_utils.SubHandlerOpts) {
			opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
				Text:            fmt.Sprintf("类型: [<code>%v</code>]\nID: [<code>%v</code>]\n用户名:[<code>%v</code>]", opts.Update.Message.Chat.Type, opts.Update.Message.Chat.ID, opts.Update.Message.Chat.Username),
				ParseMode:       models.ParseModeHTML,
			})
		},
	})
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "test",
		Handler: func(opts *handler_utils.SubHandlerOpts) {
			opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Update.Message.Chat.ID,
				Text:            "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击加入测试群组",
					URL:  "https://t.me/+BomkHuFsjqc3ZGE1",
				}}}},
			})
		},
	})
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "fileid",
		Handler: func(opts *handler_utils.SubHandlerOpts) {
			var pendingMessage string
			if opts.Update.Message.ReplyToMessage != nil {
				if opts.Update.Message.ReplyToMessage.Sticker != nil {
					pendingMessage = fmt.Sprintf("Type: [Sticker] \nFileID: [<code>%v</code>]", opts.Update.Message.ReplyToMessage.Sticker.FileID)
				} else if opts.Update.Message.ReplyToMessage.Document != nil {
					pendingMessage = fmt.Sprintf("Type: [Document] \nFileID: [<code>%v</code>]", opts.Update.Message.ReplyToMessage.Document.FileID)
				} else if opts.Update.Message.ReplyToMessage.Photo != nil {
					pendingMessage = "Type: [Photo]\n"
					if len(opts.Fields) > 1 && opts.Fields[1] == "all" { // 如果有 all 指示，显示图片所有分辨率的 File ID
						for i, n := range opts.Update.Message.ReplyToMessage.Photo {
							pendingMessage += fmt.Sprintf("\nPhotoID_%d: W:%d H:%d Size:%d \n[<code>%s</code>]\n", i, n.Width, n.Height, n.FileSize, n.FileID)
						}
					} else { // 否则显示最后一个的 File ID (应该是最高分辨率的)
						pendingMessage += fmt.Sprintf("PhotoID: [<code>%s</code>]\n", opts.Update.Message.ReplyToMessage.Photo[len(opts.Update.Message.ReplyToMessage.Photo)-1].FileID)
					}
				} else if opts.Update.Message.ReplyToMessage.Video != nil {
					pendingMessage = fmt.Sprintf("Type: [Video] \nFileID: [<code>%v</code>]", opts.Update.Message.ReplyToMessage.Video.FileID)
				} else if opts.Update.Message.ReplyToMessage.Voice != nil {
					pendingMessage = fmt.Sprintf("Type: [Voice] \nFileID: [<code>%v</code>]", opts.Update.Message.ReplyToMessage.Voice.FileID)
				} else if opts.Update.Message.ReplyToMessage.Audio != nil {
					pendingMessage = fmt.Sprintf("Type: [Audio] \nFileID: [<code>%v</code>]", opts.Update.Message.ReplyToMessage.Audio.FileID)
				} else {
					pendingMessage = "Unknown message type"
				}
			} else {
				pendingMessage = "Reply to a Sticker, Document or Photo to get its FileID"
			}
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Update.Message.Chat.ID,
				Text:            pendingMessage,
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
				ParseMode:       models.ParseModeHTML,
			})
			if err != nil {
				log.Printf("Error response /fileid command: %v", err)
			}
		},
	})
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "version",
		Handler: func(opts *handler_utils.SubHandlerOpts) {
			// info, err := opts.Thebot.GetWebhookInfo(ctx)
			// fmt.Println(info)
			// return
			botMessage, _ := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Update.Message.Chat.ID,
				Text:            mess.OutputVersionInfo(),
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
				ParseMode:       models.ParseModeMarkdownV1,
			})
			time.Sleep(time.Second * 20)
			success, _ := opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
				ChatID: opts.Update.Message.Chat.ID,
				MessageIDs: []int{
					opts.Update.Message.ID,
					botMessage.ID,
				},
			})
			if !success {
				// 如果不能把用户的消息也删了，就单独删 bot 的消息
				opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					MessageID: botMessage.ID,
				})
			}
			
		},
	})

	// inline 模式自行处理输出的函数
	plugin_utils.AddInlineManualHandlerPlugins(plugin_utils.InlineManualHandler{
		Command: "uaav",
		Attr: plugin_utils.InlineHandlerAttr{
			IsHideInCommandList: true,
			IsCantBeDefault: true,
		},
		Handler: func(opts *handler_utils.SubHandlerOpts) {
			keywords := utils.InlineExtractKeywords(opts.Fields)
			if len(keywords) == 0 {
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "custom voices",
							Title:       "URL as a voice",
							Description: "接着输入一个音频 URL 来其作为语音样式发送（不会转换格式）",
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
								ParseMode:   models.ParseModeMarkdownV1,
							},
						},
					},
				})
				if err != nil {
					mess.PrintLogAndSave(fmt.Sprintln("some error when answer custom voice tips,", err))
				}
			} else if len(keywords) == 1 {
				if strings.HasPrefix(keywords[0], "https://") {
					_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.Update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultVoice{
								ID:       "custom",
								Title:    "Custom voice",
								VoiceURL: keywords[0],
							},
						},
						IsPersonal: true,
					})
					if err != nil {
						log.Println("Error when answering inline query: ", err)
					}
				} else {
					_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.Update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:          "error",
								Title:       "音频 URL 格式错误",
								Description: "请确保音频链接以 https:// 作为开头，若填写完整 URL 后此消息依然存在，请检查 URL 是否有效",
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
									ParseMode:   models.ParseModeMarkdownV1,
								},
							},
						},
					})
					if err != nil {
						log.Println("Error when answering inline query", err)
					}
				}
			} else {
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "error",
							Title:       "参数过多，请注意空格",
							Description: fmt.Sprintf("使用方法：@%s %suaav <单个音频链接>", consts.BotMe.Username, consts.InlineSubCommandSymbol),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
								ParseMode:   models.ParseModeMarkdownV1,
							},
						},
					},
				})
				if err != nil {
					log.Println("Error when answering inline query", err)
				}
			}
		},
		Description: "将一个音频链接作为语音格式发送",
	})

	plugin_utils.AddInlinePrefixHandlerPlugins([]plugin_utils.InlinePrefixHandler{
		{
			PrefixCommand: "log",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault: true,
				IsOnlyAllowAdmin: true,
			},
			Handler: func(opts *handler_utils.SubHandlerOpts) {
				logs := mess.ReadLog()
				if logs != nil {
					log_count := len(logs)
					var log_all string
					for index, log := range logs {
						log_all = fmt.Sprintf("%s\n%02d %s", log_all, index, log)
					}
					_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.Update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:    "log",
								Title: fmt.Sprintf("%d logs update at %s", log_count, time.Now().Format(time.RFC3339)),
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: fmt.Sprintf("last update at %s\n%s", time.Now().Format(time.RFC3339), log_all),
									ParseMode:   models.ParseModeMarkdownV1,
								},
							},
						},
						IsPersonal: true,
						CacheTime:  0,
					})
					if err != nil {
						log.Println("Error when answering inline query :log", err)
					}
				} else {
					log.Println("Error when reading log file")
				}
			},
			Description: "显示日志",
		},
		{
			PrefixCommand: "plugindb_reload",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault: true,
				IsOnlyAllowAdmin: true,
			},
			Handler: func(opts *handler_utils.SubHandlerOpts) {
				consts.SignalsChannel.PluginDB_reload <- true
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "reload",
							Title:       "已请求更新",
							Description: fmt.Sprintf("last update at %s", time.Now().Format(time.RFC3339)),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: "???",
								ParseMode:   models.ParseModeMarkdownV1,
							},
						},
					},
					IsPersonal: true,
					CacheTime:  0,
				})
				if err != nil {
					log.Println("Error when answering inline query :reload", err)
				}
			},
			Description: "重新读取插件数据库",
		},
		{
			PrefixCommand: "plugindb_save",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault: true,
				IsOnlyAllowAdmin: true,
			},
			Handler: func(opts *handler_utils.SubHandlerOpts) {
				consts.SignalsChannel.PluginDB_save <- true
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "reload",
							Title:       "已请求保存",
							Description: fmt.Sprintf("last save at %s", time.Now().Format(time.RFC3339)),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: "???",
								ParseMode:   models.ParseModeMarkdownV1,
							},
						},
					},
					IsPersonal: true,
					CacheTime:  0,
				})
				if err != nil {
					log.Println("Error when answering inline query :reload", err)
				}
			},
			Description: "保存插件数据库",
		},
		{
			PrefixCommand: "savedb",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault: true,
				IsOnlyAllowAdmin: true,
			},
			Handler: func(opts *handler_utils.SubHandlerOpts) {
				consts.SignalsChannel.Database_save <- true
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "savedb",
							Title:       "已请求保存",
							Description: fmt.Sprintf("last update at %s", time.Now().Format(time.RFC3339)),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: "???",
								ParseMode:   models.ParseModeMarkdownV1,
							},
						},
					},
					IsPersonal: true,
					CacheTime:  0,
				})
				if err != nil {
					log.Println("Error when answering inline query :savedb", err)
				}
			},
			Description: "保存数据库",
		},
	}...)
}
