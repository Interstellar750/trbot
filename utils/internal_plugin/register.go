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
	"trbot/utils/flate"
	"trbot/utils/handler_params"
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
	plugin_utils.AddSlashCommandPlugins([]plugin_utils.SlashCommand{
		{
			SlashCommand: "start",
			UpdateHandler: startHandler,
		},
		{
			SlashCommand: "help",
			MessageHandler: helpHandler,
		},
		{
			SlashCommand: "chatinfo",
			MessageHandler: func(opts *handler_params.Message) error {
				logger := zerolog.Ctx(opts.Ctx)
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
					Text:            fmt.Sprintf("类型: [<code>%v</code>]\nID: [<code>%v</code>]\n用户名:[<code>%v</code>]", opts.Message.Chat.Type, opts.Message.Chat.ID, opts.Message.Chat.Username),
					ParseMode:       models.ParseModeHTML,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("command", "/chatinfo").
						Str(flate.Cont("chat info")).
						Msg(flate.SendMessage.Str())
					return fmt.Errorf(flate.SendMessage.Template(), "chat info", err)
				}
				return nil
			},
		},
		{
			SlashCommand: "test",
			MessageHandler: func(opts *handler_params.Message) error {
				logger := zerolog.Ctx(opts.Ctx)
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            "您可以订阅测试频道以查看最近的更新更新内容",
					ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
					ReplyMarkup:     &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "测试频道",
						URL:  "https://t.me/viewtrbot",
					}}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("command", "/test").
						Str(flate.Cont("test channel link")).
						Msg(flate.SendMessage.Str())
					return fmt.Errorf(flate.SendMessage.Template(), "test channel link", err)
				}
				return nil
			},
		},
		{
			SlashCommand: "fileid",
			MessageHandler: func(opts *handler_params.Message) error {
				logger := zerolog.Ctx(opts.Ctx)
				var pendingMessage string
				if opts.Message.ReplyToMessage != nil {
					if opts.Message.ReplyToMessage.Sticker != nil {
						pendingMessage = fmt.Sprintf("Type: [Sticker] \nFileID: [<code>%v</code>]", opts.Message.ReplyToMessage.Sticker.FileID)
					} else if opts.Message.ReplyToMessage.Document != nil {
						pendingMessage = fmt.Sprintf("Type: [Document] \nFileID: [<code>%v</code>]", opts.Message.ReplyToMessage.Document.FileID)
					} else if opts.Message.ReplyToMessage.Photo != nil {
						pendingMessage = "Type: [Photo]\n"
						if len(opts.Fields) > 1 && opts.Fields[1] == "all" { // 如果有 all 参数则显示图片所有分辨率的 File ID
							for i, n := range opts.Message.ReplyToMessage.Photo {
								pendingMessage += fmt.Sprintf("\nPhotoID_%d: W:%d H:%d Size:%d \n[<code>%s</code>]\n", i, n.Width, n.Height, n.FileSize, n.FileID)
							}
						} else { // 否则显示最后一个的 File ID (应该是最高分辨率的)
							pendingMessage += fmt.Sprintf("PhotoID: [<code>%s</code>]\n", opts.Message.ReplyToMessage.Photo[len(opts.Message.ReplyToMessage.Photo)-1].FileID)
						}
					} else if opts.Message.ReplyToMessage.Video != nil {
						pendingMessage = fmt.Sprintf("Type: [Video] \nFileID: [<code>%v</code>]", opts.Message.ReplyToMessage.Video.FileID)
					} else if opts.Message.ReplyToMessage.Voice != nil {
						pendingMessage = fmt.Sprintf("Type: [Voice] \nFileID: [<code>%v</code>]", opts.Message.ReplyToMessage.Voice.FileID)
					} else if opts.Message.ReplyToMessage.Audio != nil {
						pendingMessage = fmt.Sprintf("Type: [Audio] \nFileID: [<code>%v</code>]", opts.Message.ReplyToMessage.Audio.FileID)
					} else {
						pendingMessage = "Unknown message type"
					}
				} else {
					pendingMessage = "Reply to a Sticker, Document or Photo to get its FileID"
				}
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            pendingMessage,
					ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
					ParseMode:       models.ParseModeHTML,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("command", "/fileid").
						Str(flate.Cont("media file ID info")).
						Msg(flate.SendMessage.Str())
					return fmt.Errorf(flate.SendMessage.Template(), "media file ID info", err)
				}
				return nil
			},
		},
		{
			SlashCommand: "version",
			MessageHandler: func(opts *handler_params.Message) error {
				logger := zerolog.Ctx(opts.Ctx)
				// info, err := opts.Thebot.GetWebhookInfo(ctx)
				// fmt.Println(info)
				// return
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            mess.OutputVersionInfo(),
					ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
					ParseMode:       models.ParseModeMarkdownV1,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("command", "/version").
						Str(flate.Cont("bot version info")).
						Msg(flate.SendMessage.Str())
					return fmt.Errorf(flate.SendMessage.Template(), "bot version info", err)
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
			MessageHandler:  nil, // 不回复
		},
		{
			Prefix:   "via-inline",
			Argument: "change-inline-command",
			MessageHandler: func(opts *handler_params.Message) error {
				logger := zerolog.Ctx(opts.Ctx)
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            fmt.Sprintf("选择一个 Inline 模式下的默认命令<blockquote>由于缓存原因，您可能需要等一会才能看到更新后的结果</blockquote>无论您是否设定了默认命令，您始终都可以在 inline 模式下输入 <code>%s</code> 号来查看全部可用的命令", configs.BotConfig.InlineSubCommandSymbol),
					ParseMode:       models.ParseModeHTML,
					ReplyMarkup:     plugin_utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str(flate.Cont("select inline default command keyboard")).
						Str("content", "").
						Msg(flate.SendMessage.Str())
					return fmt.Errorf(flate.SendMessage.Template(), "select inline default command keyboard", err)
				}
				return nil
			},
		},
	}...)

	// 触发：'/start <Argument>'，如果是通过消息按钮发送的，用户只会看到自己发送了一个 `/start`
	plugin_utils.AddSlashStartCommandPlugins([]plugin_utils.SlashStartHandler{
		{
			Argument:       "noreply",
			MessageHandler: nil, // 不回复
		},
	}...)

	// 通过消息按钮触发的请求
	plugin_utils.AddCallbackQueryPlugins([]plugin_utils.CallbackQuery{
		{
			CommandChar: "inline_default_",
			UpdateHandler: func(opts *handler_params.Update) error {
				logger := zerolog.Ctx(opts.Ctx).
					With().
					Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
					Logger()

				var handlerErr flate.MultErr

				if opts.Update.CallbackQuery.Data == "inline_default_none" {
					err := database.SetCustomFlag(opts.Ctx, opts.Update.CallbackQuery.From.ID, db_struct.DefaultInlinePlugin, "")
					if err != nil {
						logger.Error().
							Err(err).
							Msg("Failed to remove inline default command flag")
						handlerErr.Addf("failed to remove inline default command flag: %w", err)
					} else {
						// if chatinfo get from redis database, it won't be the newst data, need reload it from database
						opts.ChatInfo, err = database.GetChatInfo(opts.Ctx, opts.Update.CallbackQuery.From.ID)
						if err != nil {
							logger.Error().
								Err(err).
								Msg("Failed to get chat info")
							handlerErr.Addf("failed to get chat info: %w", err)
						} else {
							_, err = opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
								ChatID:      opts.Update.CallbackQuery.Message.Message.Chat.ID,
								MessageID:   opts.Update.CallbackQuery.Message.Message.ID,
								ReplyMarkup: plugin_utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
							})
							if err != nil {
								logger.Error().Func(flate.Wrapper(&handlerErr).
									Err(err).
									ErrContent("inline command select keyboard",
										flate.EditMessageReplyMarkup,
									).
									DoneAndSend())
								return err
							}
						}
					}
				} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "inline_default_noedit_") {
					callbackField := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "inline_default_noedit_")
					for _, inlinePlugin := range plugin_utils.AllPlugins.InlineCommandList {
						if inlinePlugin.Command == callbackField {
							err := database.SetCustomFlag(opts.Ctx, opts.Update.CallbackQuery.From.ID, db_struct.DefaultInlinePlugin, callbackField)
							if err != nil {
								logger.Error().
									Err(err).
									Msg("Failed to change inline default command flag")
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
									Str("content", "inline command changed").
									Msg(flate.AnswerCallbackQuery.Str())
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
									Msg("Failed to change inline default command flag")
								return err
							}
							// if chatinfo get from redis database, it won't be the latest data, need reload it from database
							opts.ChatInfo, err = database.GetChatInfo(opts.Ctx, opts.Update.CallbackQuery.From.ID)
							if err != nil {
								logger.Error().
									Err(err).
									Msg("Failed to get chat info")
							}
							_, err = opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
								ChatID:      opts.Update.CallbackQuery.Message.Message.Chat.ID,
								MessageID:   opts.Update.CallbackQuery.Message.Message.ID,
								ReplyMarkup: plugin_utils.BuildDefaultInlineCommandSelectKeyboard(opts.ChatInfo),
							})
							if err != nil {
								logger.Error().
									Err(err).
									Str("content", "inline command select keyboard").
									Msg(flate.EditMessageReplyMarkup.Str())
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
			CommandChar:          "help",
			CallbackQueryHandler: helpCallbackHandler,
		},
		{
			CommandChar:   "HBMT", // Handler By Message Type
			UpdateHandler: plugin_utils.SelectByMessageTypeHandlerCallback,
		},
	}...)

	// inline 模式自行处理输出的函数
	plugin_utils.AddInlineManualHandlerPlugins(plugin_utils.InlineManualHandler{
		Command: "uaav",
		Attr: plugin_utils.InlineHandlerAttr{
			IsHideInCommandList: true,
			IsCantBeDefault: true,
		},
		InlineHandler: func(opts *handler_params.InlineQuery) error {
			logger := zerolog.Ctx(opts.Ctx)
			var handlerErr flate.MultErr
			keywords := utils.InlineExtractKeywords(opts.Fields)
			if len(keywords) == 0 {
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.InlineQuery.ID,
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
						Dict(utils.GetUserDict(opts.InlineQuery.From)).
						Str("content", "uaav command usage tips").
						Msg(flate.AnswerInlineQuery.Str())
					handlerErr.Addf("failed to send `uaav command usage tips` inline answer: %w", err)
				}
			} else if len(keywords) == 1 {
				if strings.HasPrefix(keywords[0], "https://") {
					_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.InlineQuery.ID,
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
							Dict(utils.GetUserDict(opts.InlineQuery.From)).
							Str("query", opts.InlineQuery.Query).
							Str("content", "uaav valid voice url").
							Msg(flate.AnswerInlineQuery.Str())
						handlerErr.Addf("failed to send `uaav valid voice url` inline answer: %w", err)
					}
				} else {
					_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.InlineQuery.ID,
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
							Dict(utils.GetUserDict(opts.InlineQuery.From)).
							Str("query", opts.InlineQuery.Query).
							Str("content", "uaav invalid URL").
							Msg(flate.AnswerInlineQuery.Str())
						handlerErr.Addf("failed to send `uaav invalid URL` inline answer: %w", err)
					}
				}
			} else {
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.InlineQuery.ID,
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
						Dict(utils.GetUserDict(opts.InlineQuery.From)).
						Str("query", opts.InlineQuery.Query).
						Str("command", "uaav").
						Str("content", "too much argumunt").
						Msg(flate.AnswerInlineQuery.Str())
					return err
				}
			}
			return handlerErr.Flat()
		},
		Description: "将一个音频链接作为语音格式发送",
	})

	// inline 模式中以前缀触发的命令，需要自行处理输出。
	plugin_utils.AddInlinePrefixHandlerPlugins([]plugin_utils.InlinePrefixHandler{
		{
			PrefixCommand: "panic",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault:     true,
				IsOnlyAllowAdmin:    true,
			},
			InlineHandler: func(opts *handler_params.InlineQuery) error {
				// zerolog.Ctx(ctx).Error().Stack().Err(errors.WithStack(errors.New("test panic"))).Msg("")
				panic("test panic")
			},
			Description: "测试 panic",
		},
		{
			PrefixCommand: "log",
			Attr: plugin_utils.InlineHandlerAttr{
				IsHideInCommandList: true,
				IsCantBeDefault:     true,
				IsOnlyAllowAdmin:    true,
			},
			UpdateHandler: func(opts *handler_params.Update) error {
				logger := zerolog.Ctx(opts.Ctx)
				var handlerErr flate.MultErr
				logs, err := mess.ReadLog()
				if err != nil {
					logger.Error().
						Err(err).
						Str("query", opts.Update.InlineQuery.Query).
						Dict(utils.GetUserDict(opts.Update.InlineQuery.From)).
						Str("command", "log").
						Msg("Failed to read log by inline command")
					handlerErr.Addf("failed to read log: %w", err)
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
							Str("query", opts.Update.InlineQuery.Query).
							Str("content", "log infos").
							Msg(flate.AnswerInlineQuery.Str())
						handlerErr.Addf("failed to send `log infos` inline answer: %w", err)
					}
				}
				return handlerErr.Flat()
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
			UpdateHandler: func(opts *handler_params.Update) error {
				logger := zerolog.Ctx(opts.Ctx)
				var handlerErr flate.MultErr
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
						Str("query", opts.Update.InlineQuery.Query).
						Str("content", "plugin database reloaded").
						Msg(flate.AnswerInlineQuery.Str())
					handlerErr.Addf("failed to send `plugin database reloaded` inline answer: %w", err)
				}
				return handlerErr.Flat()
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
			UpdateHandler: func(opts *handler_params.Update) error {
				logger := zerolog.Ctx(opts.Ctx)
				var handlerErr flate.MultErr
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
						Str("query", opts.Update.InlineQuery.Query).
						Str("content", "plugin database saved").
						Msg(flate.AnswerInlineQuery.Str())
					handlerErr.Addf("failed to send `plugin database saved` inline answer: %w", err)
				}
				return handlerErr.Flat()
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
			UpdateHandler: func(opts *handler_params.Update) error {
				logger := zerolog.Ctx(opts.Ctx)
				var handlerErr flate.MultErr
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
						Str("query", opts.Update.InlineQuery.Query).
						Str("content", "database saved").
						Msg(flate.AnswerInlineQuery.Str())
					handlerErr.Addf("failed to send `database saved` inline answer: %w", err)
				}
				return handlerErr.Flat()
			},
			Description: "保存数据库",
		},
	}...)
}
