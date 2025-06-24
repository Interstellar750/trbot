package internal_plugin

import (
	"context"
	"fmt"
	"strings"
	"time"
	"trbot/database"
	"trbot/database/db_struct"
	"trbot/plugins"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/handler_structs"
	"trbot/utils/mess"
	"trbot/utils/plugin_utils"
	"trbot/utils/signals"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

// this function run only once in main
func Register(ctx context.Context) {
	// 初始化 plugins/ 中的插件
	plugins.InitPlugins()

	plugin_utils.RunPluginInitializers(ctx)

	// 以 `/` 符号开头的命令
	plugin_utils.AddSlashSymbolCommandPlugins([]plugin_utils.SlashSymbolCommand{
		{
			SlashCommand: "start",
			Handler:      startHandler,
		},
		{
			SlashCommand: "help",
			Handler:      helpHandler,
		},
		{
			SlashCommand: "chatinfo",
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
					Text:            fmt.Sprintf("类型: [<code>%v</code>]\nID: [<code>%v</code>]\n用户名:[<code>%v</code>]", opts.Update.Message.Chat.Type, opts.Update.Message.Chat.ID, opts.Update.Message.Chat.Username),
					ParseMode:       models.ParseModeHTML,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("command", "/chatinfo").
						Msg("send `chat info` message failed")
				}
				return err
			},
		},
		{
			SlashCommand: "test",
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Update.Message.Chat.ID,
					Text:            "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
					ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
					ReplyMarkup:     &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "点击加入测试群组",
						URL:  "https://t.me/+BomkHuFsjqc3ZGE1",
					}}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("command", "/test").
						Msg("send `test group invite link` message failed")
				}
				return err
			},
		},
		{
			SlashCommand: "fileid",
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				var pendingMessage string
				if opts.Update.Message.ReplyToMessage != nil {
					if opts.Update.Message.ReplyToMessage.Sticker != nil {
						pendingMessage = fmt.Sprintf("Type: [Sticker] \nFileID: [<code>%v</code>]", opts.Update.Message.ReplyToMessage.Sticker.FileID)
					} else if opts.Update.Message.ReplyToMessage.Document != nil {
						pendingMessage = fmt.Sprintf("Type: [Document] \nFileID: [<code>%v</code>]", opts.Update.Message.ReplyToMessage.Document.FileID)
					} else if opts.Update.Message.ReplyToMessage.Photo != nil {
						pendingMessage = "Type: [Photo]\n"
						if len(opts.Fields) > 1 && opts.Fields[1] == "all" { // 如果有 all 参数则显示图片所有分辨率的 File ID
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
					logger.Error().
						Err(err).
						Str("command", "/fileid").
						Msg("send `file ID` message failed")
				}
				return err
			},
		},
		{
			SlashCommand: "version",
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				// info, err := opts.Thebot.GetWebhookInfo(ctx)
				// fmt.Println(info)
				// return
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Update.Message.Chat.ID,
					Text:            mess.OutputVersionInfo(),
					ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
					ParseMode:       models.ParseModeMarkdownV1,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("command", "/version").
						Msg("Failed to send `bot version info` message")
					return err
				}
				return nil
			},
		},
	}...)

	// 触发：'/start <Prefix>_<Argument>'，如果是通过消息按钮发送的，用户只会看到自己发送了一个 `/start`
	plugin_utils.AddSlashStartWithPrefixCommandPlugins([]plugin_utils.SlashStartWithPrefixHandler{
		{
			Prefix:   "via-inline",
			Argument: "noreply",
			Handler:  nil, // 不回复
		},
		{
			Prefix:   "via-inline",
			Argument: "change-inline-command",
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Update.Message.Chat.ID,
					Text:            fmt.Sprintf("选择一个 Inline 模式下的默认命令<blockquote>由于缓存原因，您可能需要等一会才能看到更新后的结果</blockquote>无论您是否设定了默认命令，您始终都可以在 inline 模式下输入 <code>%s</code> 号来查看全部可用的命令", configs.BotConfig.InlineSubCommandSymbol),
					ParseMode:       models.ParseModeHTML,
					ReplyMarkup:     plugin_utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				})
				if err != nil {
					logger.Error().
						Err(err).
						Msg("Failed to send `select inline default command keyboard` message")
				}
				return err
			},
		},
	}...)

	// 触发：'/start <Argument>'，如果是通过消息按钮发送的，用户只会看到自己发送了一个 `/start`
	plugin_utils.AddSlashStartCommandPlugins([]plugin_utils.SlashStartHandler{
		{
			Argument: "noreply",
			Handler:  nil, // 不回复
		},
	}...)

	// 通过消息按钮触发的请求
	plugin_utils.AddCallbackQueryCommandPlugins([]plugin_utils.CallbackQuery{
		{
			CommandChar: "inline_default_",
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				if opts.Update.CallbackQuery.Data == "inline_default_none" {
					err := database.SetCustomFlag(opts.Ctx, opts.Update.CallbackQuery.From.ID, db_struct.DefaultInlinePlugin, "")
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
							Msg("Remove inline default command flag failed")
						return err
					}
					// if chatinfo get from redis database, it won't be the newst data, need reload it from database
					opts.ChatInfo, err = database.GetChatInfo(opts.Ctx, opts.Update.CallbackQuery.From.ID)
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
							Msg("Get chat info failed")
					}
					_, err = opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
						ChatID:      opts.Update.CallbackQuery.Message.Message.Chat.ID,
						MessageID:   opts.Update.CallbackQuery.Message.Message.ID,
						ReplyMarkup: plugin_utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
					})
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
							Msg("Edit message to `inline command select keyboard` failed")
						return err
					}
				}
				if strings.HasPrefix(opts.Update.CallbackQuery.Data, "inline_default_noedit_") {
					callbackField := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "inline_default_noedit_")
					for _, inlinePlugin := range plugin_utils.AllPlugins.InlineCommandList {
						if inlinePlugin.Command == callbackField {
							err := database.SetCustomFlag(opts.Ctx, opts.Update.CallbackQuery.From.ID, db_struct.DefaultInlinePlugin, callbackField)
							if err != nil {
								logger.Error().
									Err(err).
									Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
									Msg("Change inline default command flag failed")
								return err
							}
							_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
								CallbackQueryID: opts.Update.CallbackQuery.ID,
								Text:            fmt.Sprintf("已成功将您的 inline 模式默认命令设为 \"%s\"", callbackField),
								ShowAlert:       true,
							})
							if err != nil {
								logger.Error().
									Err(err).
									Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
									Msg("Failed to send `inline command changed` callback answer")
								return err
							}
							break
						}
					}
				} else {
					callbackField := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "inline_default_")
					for _, inlinePlugin := range plugin_utils.AllPlugins.InlineCommandList {
						if inlinePlugin.Command == callbackField {
							err := database.SetCustomFlag(opts.Ctx, opts.Update.CallbackQuery.From.ID, db_struct.DefaultInlinePlugin, callbackField)
							if err != nil {
								logger.Error().
									Err(err).
									Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
									Msg("Change inline default command flag failed")
								return err
							}
							// if chatinfo get from redis database, it won't be the newst data, need reload it from database
							opts.ChatInfo, err = database.GetChatInfo(opts.Ctx, opts.Update.CallbackQuery.From.ID)
							if err != nil {
								logger.Error().
									Err(err).
									Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
									Msg("Get chat info failed")
							}
							_, err = opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
								ChatID:      opts.Update.CallbackQuery.Message.Message.Chat.ID,
								MessageID:   opts.Update.CallbackQuery.Message.Message.ID,
								ReplyMarkup: plugin_utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
							})
							if err != nil {
								logger.Error().
									Err(err).
									Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
									Msg("Edit message to `inline command select keyboard` failed")
								return err
							}
							break
						}
					}
				}
				
				signals.SIGNALS.Database_save <- true
				return nil
			},
		},
		{
			CommandChar: "help",
			Handler:     helpCallbackHandler,
		},
		{
			CommandChar: "HBMT", // Handler By Message Type
			Handler: plugin_utils.SelectHandlerByMessageTypeHandlerCallback,
		},
	}...)

	// inline 模式自行处理输出的函数
	plugin_utils.AddInlineManualHandlerPlugins(plugin_utils.InlineManualHandler{
		Command: "uaav",
		Attr: plugin_utils.InlineHandlerAttr{
			IsHideInCommandList: true,
			IsCantBeDefault: true,
		},
		Handler: func(opts *handler_structs.SubHandlerParams) error {
			logger := zerolog.Ctx(opts.Ctx)
			keywords := utils.InlineExtractKeywords(opts.Fields)
			if len(keywords) == 0 {
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "custom_voices",
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
					logger.Error().
						Err(err).
						Str("command", "uaav").
						Msg("Failed to send `usage tips` inline result")
					return err
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
						logger.Error().
							Err(err).
							Str("query", opts.Update.InlineQuery.Query).
							Str("command", "uaav").
							Msg("Failed to send `valid voice url` inline result")
						return err
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
						logger.Error().
							Err(err).
							Str("query", opts.Update.InlineQuery.Query).
							Str("command", "uaav").
							Msg("Failed to send `URL invalid` inline result")
						return err
					}
				}
			} else {
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "error",
							Title:       "参数过多，请注意空格",
							Description: fmt.Sprintf("使用方法：@%s %suaav <单个音频链接>", consts.BotMe.Username, configs.BotConfig.InlineSubCommandSymbol),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
								ParseMode:   models.ParseModeMarkdownV1,
							},
						},
					},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("query", opts.Update.InlineQuery.Query).
						Str("command", "uaav").
						Msg("Failed to send `too much argumunt` inline result")
					return err
				}
			}
			return nil
		},
		Description: "将一个音频链接作为语音格式发送",
	})

	// inline 模式中以前缀触发的命令，需要自行处理输出。
	plugin_utils.AddInlinePrefixHandlerPlugins([]plugin_utils.InlinePrefixHandler{
		{
			PrefixCommand: "log",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault:     true,
				IsOnlyAllowAdmin:    true,
			},
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				logs, err := mess.ReadLog()
				if err != nil {
					logger.Error().
						Err(err).
						Str("query", opts.Update.InlineQuery.Query).
						Dict(utils.GetUserDict(opts.Update.InlineQuery.From)).
						Str("command", "log").
						Msg("Read log by inline command failed")
					return err
				}
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
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(opts.Update.InlineQuery.From)).
							Str("command", "log").
							Msg("Failed to send `log info` inline result")
							
						return err
					}
				}
				return nil
			},
			Description: "显示日志",
		},
		{
			PrefixCommand: "reloadpdb",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault:     true,
				IsOnlyAllowAdmin:    true,
			},
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				signals.SIGNALS.PluginDB_reload <- true
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "reloadpdb-back",
							Title:       "已请求重新加载插件数据库",
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
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.InlineQuery.From)).
						Str("command", "reloadpdb").
						Msg("Failed to send `reload plugin database info` inline result")
				}
				return err
			},
			Description: "重新读取插件数据库",
		},
		{
			PrefixCommand: "savepdb",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault:     true,
				IsOnlyAllowAdmin:    true,
			},
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				signals.SIGNALS.PluginDB_save <- true
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "savepdb-back",
							Title:       "已请求保存插件数据库",
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
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.InlineQuery.From)).
						Str("command", "savepdb").
						Msg("Failed to send `save plugin database info` inline result")
				}
				return err
			},
			Description: "保存插件数据库",
		},
		{
			PrefixCommand: "savedb",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault:     true,
				IsOnlyAllowAdmin:    true,
			},
			Handler: func(opts *handler_structs.SubHandlerParams) error {
				logger := zerolog.Ctx(opts.Ctx)
				signals.SIGNALS.Database_save <- true
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:          "savedb-back",
							Title:       "已请求保存数据库",
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
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.InlineQuery.From)).
						Str("command", "savedb").
						Msg("Failed to send `save database info` inline result")
				}
				return err
			},
			Description: "保存数据库",
		},
	}...)
}
