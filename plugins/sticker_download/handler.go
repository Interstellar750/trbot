package sticker_download

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"trle5.xyz/gopkg/trbot/database"
	"trle5.xyz/gopkg/trbot/database/db_struct"
	"trle5.xyz/gopkg/trbot/plugins/sticker_download/collect"
	"trle5.xyz/gopkg/trbot/plugins/sticker_download/config"
	"trle5.xyz/gopkg/trbot/plugins/sticker_download/download"
	"trle5.xyz/gopkg/trbot/plugins/sticker_download/lock"
	"trle5.xyz/gopkg/trbot/plugins/sticker_download/mpeg4gif"
	"trle5.xyz/gopkg/trbot/utils"
	"trle5.xyz/gopkg/trbot/utils/configs"
	"trle5.xyz/gopkg/trbot/utils/flaterr"
	"trle5.xyz/gopkg/trbot/utils/handler_params"
	"trle5.xyz/gopkg/trbot/utils/plugin_utils"
	"trle5.xyz/gopkg/trbot/utils/task"
	"trle5.xyz/gopkg/trbot/utils/type/contain"
	"trle5.xyz/gopkg/trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
	"github.com/rs/zerolog"
)

func Init() {
	plugin_utils.AddCallbackQueryHandlers([]plugin_utils.CallbackQuery{
		{
			// 不转换格式，打包下载整个贴纸包
			CallbackDataPrefix: "s_",
			CallbackQueryHandler: stickerPackCallbackHandler,
		},
		{
			// 将贴纸包中的静态贴纸全部转换为 PNG 格式并打包
			CallbackDataPrefix: "S_",
			CallbackQueryHandler: stickerPackCallbackHandler,
		},
	}...)
	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "贴纸下载",
		Description: "向机器人发送任意贴纸或视频 GIF 来下载它\n向机器人发送贴纸包链接来下载整个贴纸包\n\n注意：\n静态贴纸会被转换为 PNG 格式\n视频贴纸和视频 GIF 会被转换为 GIF 格式\n动画贴纸（.tgs 格式）不会被转换，将会添加 .file 后缀以文件形式发回",
	})
	plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
		PluginName:       "下载贴纸",
		ChatType:         models.ChatTypePrivate,
		MessageType:      message_utils.Sticker,
		AllowAutoTrigger: true,
		MessageHandler:   stickerHandler,
	})
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "sticker_download",
		Func: func(ctx context.Context, thebot *bot.Bot) error{
			err := config.ReadStickerConfig(ctx)
			if err != nil {
				return fmt.Errorf("failed to read sticker config: %w", err)
			}

			err = task.ScheduleTask(ctx, task.Task{
				Name:    "save_sticker_config",
				Group:   "sticker",
				Job:     job.NewFunctionJobWithDesc(
					func(ctx context.Context) (int, error) {
						config.SaveStickerConfig(ctx)
						return 0, nil
					},
					"Save sticker config every 10 minutes",
				),
				Trigger: quartz.NewSimpleTrigger(10 * time.Minute),
			})
			if err != nil {
				return fmt.Errorf("failed to add auto save sticker config task: %w", err)
			}

			if config.Config.AllowDownloadStickerSet || config.Config.UseCollcetSticker {
				plugin_utils.AddFullCommandHandlers([]plugin_utils.FullCommand{
					{
						FullCommand:    "https://t.me/addstickers/",
						ForChatType:    []models.ChatType{models.ChatTypePrivate},
						MessageHandler: stickerLinkHandler,
					},
					{
						FullCommand:    "t.me/addstickers/",
						ForChatType:    []models.ChatType{models.ChatTypePrivate},
						MessageHandler: stickerLinkHandler,
					},
				}...)
			}

			if config.Config.UseCollcetSticker {
				err = collect.ReadCollectStickerList(ctx)
				if err != nil {
					return fmt.Errorf("failed to init collect sticker: %w", err)
				}
				plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
					Name:   "collectsitcker",
					Saver:  collect.SaveCollectStickerList,
					Loader: collect.ReadCollectStickerList,
				})
				plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
					// 下载贴纸包中的所有贴纸并打包发送到收藏频道（不转换格式）
					CallbackDataPrefix:   "c_",
					CallbackQueryHandler: collect.CollectStickerSet,
				})
			}

			if !config.Config.DisableConvert && config.Config.FFmpegPath != "" {
				plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
					PluginName:       "MP4 转 GIF",
					ChatType:         models.ChatTypePrivate,
					MessageType:      message_utils.Animation,
					AllowAutoTrigger: true,
					MessageHandler:   mpeg4gif.ConvertMP4ToGifHandler,
				})
			}
			return nil
		},
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "sticker_download",
		Saver:  config.SaveStickerConfig,
		Loader: config.ReadStickerConfig,
	})
}

func stickerHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	logger.Info().
		Str("emoji", opts.Message.Sticker.Emoji).
		Str("setName", opts.Message.Sticker.SetName).
		Msg("Start download sticker")

	err := database.IncrementalUsageCount(opts.Ctx, opts.Message.From.ID, db_struct.StickerDownloaded)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to incremental sticker download count")
		handlerErr.Addf("failed to incremental sticker download count: %w", err)
	}

	if lock.Download.TryLock() {
		// 成功获取到锁，直接继续下载不发送提示信息
		defer lock.Download.Unlock()
	} else {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.From.ID,
			Text:            "已加入贴纸下载队列，请稍候...",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "sticker download in progress notice").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "sticker download in progress notice", err)
		}

		// 发送信息提示等待后，阻断等待锁
		lock.Download.Lock()
		defer lock.Download.Unlock()
	}

	_, err = opts.Thebot.SendChatAction(opts.Ctx, &bot.SendChatActionParams{
		ChatID: opts.Message.From.ID,
		Action: models.ChatActionChooseSticker,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Msg("failed to send chat action")
	}

	stickerData, err := download.GetSticker(opts)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error when downloading sticker")
		handlerErr.Addf("error when downloading sticker: %w", err)

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.From.ID,
			Text:            fmt.Sprintf("下载贴纸时发生了一些错误\n<blockquote expandable>Failed to download sticker: %s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "sticker download error").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "sticker download error", err)
		}
	} else {
		if stickerData.ZipFile != nil {
			defer stickerData.ZipFile.Close()
		}
		documentParams := &bot.SendDocumentParams{
			ChatID:                      opts.Message.From.ID,
			ParseMode:                   models.ParseModeHTML,
			ReplyParameters:             &models.ReplyParameters{ MessageID: opts.Message.ID },
			DisableNotification:         true,
			DisableContentTypeDetection: true, // Prevent the server convert GIF to MP4
		}

		var stickerFilePrefix, stickerFileSuffix string
		if opts.Message.Sticker.IsVideo {
			if stickerData.IsConverted {
				stickerFileSuffix = "gif"
			} else {
				documentParams.Caption = "<blockquote>see <a href=\"https://wikipedia.org/wiki/WebM\">wikipedia/WebM</a></blockquote>"
				stickerFileSuffix = "webm"
			}
		} else if opts.Message.Sticker.IsAnimated {
			if stickerData.IsConverted {
				stickerFileSuffix = "gif"
			} else {
				documentParams.Caption = "<blockquote>see <a href=\"https://core.telegram.org/stickers#animated-stickers\">stickers/animated-stickers</a></blockquote>"
				stickerFileSuffix = "tgs.file"
			}
		} else {
			if stickerData.IsConverted {
				stickerFileSuffix = "png"
			} else {
				documentParams.Caption = "<blockquote>see <a href=\"https://wikipedia.org/wiki/WebP\">wikipedia/WebP</a></blockquote>"
				stickerFileSuffix = "webp"
			}
		}

		if stickerData.IsCustomSticker {
			stickerFilePrefix = "sticker"
		} else {
			var button [][]models.InlineKeyboardButton
			var suffix string = stickerData.StickerSetName
			if len(stickerData.StickerSetName) > 62 {
				id := config.Config.GetOversizeSetIDByName(stickerData.StickerSetName)
				if id == 0 {
					config.Config.OversizeSetCount++
					config.Config.OversizeSets = append(config.Config.OversizeSets, config.OversizeSet{
						SetName: stickerData.StickerSetName,
						SetID:   config.Config.OversizeSetCount,
					})
					suffix = fmt.Sprintf("_%d", config.Config.OversizeSetCount)
				} else {
					suffix = fmt.Sprintf("_%d", id)
				}
			}

			if config.Config.AllowDownloadStickerSet {
				if config.Config.DisableConvert {
					button = [][]models.InlineKeyboardButton{
						{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", suffix) }},
					}
				} else {
					button = [][]models.InlineKeyboardButton{
						{{ Text: "下载转换后的贴纸包", CallbackData: fmt.Sprintf("S_%s", suffix) }},
						{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", suffix) }},
					}
				}
				documentParams.Caption += fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包中一共有 %d 个贴纸\n", stickerData.StickerSetName, stickerData.StickerSetTitle, stickerData.StickerCount)
				stickerFilePrefix = fmt.Sprintf("%s_%d", stickerData.StickerSetName, stickerData.StickerIndex)
			} else {
				stickerFilePrefix = stickerData.StickerSetName
			}

			if config.Config.UseCollcetSticker {
				collect.AddButton(suffix, stickerData.StickerCount, contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...), &button)
			}

			// 仅在不为自定义贴纸时显示下载整个贴纸包按钮
			if len(button) > 0 {
				documentParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: button }
			}
		}

		documentParams.Document = &models.InputFileUpload{ Filename: fmt.Sprintf("%s.%s", stickerFilePrefix, stickerFileSuffix), Data: stickerData.Data }

		_, err = opts.Thebot.SendDocument(opts.Ctx, documentParams)
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "sticker file").
				Msg(flaterr.SendDocument.Str())
			handlerErr.Addt(flaterr.SendDocument, "sticker file", err)
		}
	}

	return handlerErr.Flat()
}

func stickerPackCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if !config.Config.AllowDownloadStickerSet {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "机器人当前禁用了贴纸包下载功能",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "stickerset download disabled notice").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "stickerset download disabled notice", err)
		}
		return handlerErr.Flat()
	}

	err := database.IncrementalUsageCount(opts.Ctx, opts.CallbackQuery.From.ID, db_struct.StickerSetDownloaded)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to incremental sticker set download count")
		handlerErr.Addf("failed to incremental sticker set download count: %w", err)
	}

	var setName     string
	var needConvert bool
	if opts.CallbackQuery.Data[0:2] == "S_" {
		setName = strings.TrimPrefix(opts.CallbackQuery.Data, "S_")
		needConvert = true
		if config.Config.DisableConvert {
			_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: opts.CallbackQuery.ID,
				Text:            "机器人当前禁用了贴纸包转换功能",
				ShowAlert:       true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "stickerset convert disabled notice").
					Msg(flaterr.AnswerCallbackQuery.Str())
				handlerErr.Addt(flaterr.AnswerCallbackQuery, "stickerset convert disabled notice", err)
			}
			return handlerErr.Flat()
		}
	} else {
		setName = strings.TrimPrefix(opts.CallbackQuery.Data, "s_")
		needConvert = false
	}

	// 有下划线开头即代表这是一个超出了长度的贴纸包，下划线后面的数据是一个 ID
	// 解析这个 ID 并从 config.Config.OversizeSets 中拿到对应的贴纸包名称
	if setName[0:1] == "_" {
		setID, err := strconv.Atoi(strings.TrimPrefix(setName, "_"))
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to parser oversize sticker set ID")
			handlerErr.Addf("failed to parser oversize sticker set ID [%s]: %w", setName, err)
			_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: opts.CallbackQuery.ID,
				Text:            "解析贴纸包 ID 失败，请重新尝试发送此贴纸再点击按钮",
				ShowAlert:       true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "failed to parser oversize sticker set ID notice").
					Msg(flaterr.AnswerCallbackQuery.Str())
				handlerErr.Addt(flaterr.AnswerCallbackQuery, "failed to parser oversize sticker set ID notice", err)
			}
			return handlerErr.Flat()
		}
		setName = config.Config.GetOversizeSetNameByID(setID)
		if setName == "" {
			logger.Error().
				Int("setID", setID).
				Msg("Failed to find oversize sticker set")
			handlerErr.Addf("failed to find oversize sticker set [%d]", setID)
			_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: opts.CallbackQuery.ID,
				Text:            "未找到 ID 对应的贴纸包，请重新尝试发送此贴纸再点击按钮",
				ShowAlert:       true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "no sticker set found for set ID notice").
					Msg(flaterr.AnswerCallbackQuery.Str())
				handlerErr.Addt(flaterr.AnswerCallbackQuery, "no sticker set found for set ID notice", err)
			}
			return handlerErr.Flat()
		}
	}

	if lock.Download.TryLock() {
		// 成功获取到锁，直接继续下载不发送提示信息
		defer lock.Download.Unlock()
	} else {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "您的下载请求已提交，但由于当前有其他用户正在下载贴纸，可能需要等待一段时间后才能开始下载",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "stickerset download in progress notice").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "stickerset download in progress notice", err)
		}

		// 发送信息提示等待后，阻断等待锁
		lock.Download.Lock()
		defer lock.Download.Unlock()
	}

	// 通过贴纸的 packName 获取贴纸集
	stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: setName })
	if err != nil {
		logger.Error().
			Err(err).
			Str("setName", setName).
			Msg(flaterr.GetStickerSet.Str())
		handlerErr.Addt(flaterr.GetStickerSet, setName, err)

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.CallbackQuery.From.ID,
			Text:            fmt.Sprintf("获取贴纸包信息时发生了一些错误\n<blockquote expandable>Failed to get sticker set info: %s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.CallbackQuery.Message.Message.ID },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "get sticker set info error").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "get sticker set info error", err)
		}
	} else {
		logger.Info().
			Bool("needConvert", needConvert).
			Dict("stickerSet", zerolog.Dict().
				Str("title", stickerSet.Title).
				Str("name", stickerSet.Name).
				Int("allCount", len(stickerSet.Stickers)),
			).
			Msg("Start download sticker set")

		if len(stickerSet.Stickers) > 120 {
			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:             opts.CallbackQuery.From.ID,
				Text:               fmt.Sprintf("抱歉，<a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包中的贴纸数量过多 (%d)，无法使用打包下载功能<blockquote expandable>您可以单独发送要下载的贴纸</blockquote>", stickerSet.Name, stickerSet.Title, len(stickerSet.Stickers)),
				ReplyParameters:    &models.ReplyParameters{ MessageID: opts.CallbackQuery.Message.Message.ID },
				LinkPreviewOptions: &models.LinkPreviewOptions{ IsDisabled: bot.True()},
				ParseMode:          models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "too many stickers").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "too many stickers", err)
			}
			return handlerErr.Flat()
		}

		if opts.CallbackQuery.Message.Message.Text != "" {
			_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text:      fmt.Sprintf("正在下载%s <a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包，请稍候...", utils.TextForTrueOrFalse(needConvert, "并转换", ""), stickerSet.Name, stickerSet.Title),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "start download stickerset notice").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "start download stickerset notice", err)
			}
		} else {
			_, err = opts.Thebot.EditMessageCaption(opts.Ctx, &bot.EditMessageCaptionParams{
				ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Caption:   fmt.Sprintf("正在下载%s <a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包，请稍候...", utils.TextForTrueOrFalse(needConvert, "并转换", ""), stickerSet.Name, stickerSet.Title),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "start download stickerset notice").
					Msg(flaterr.EditMessageCaption.Str())
				handlerErr.Addt(flaterr.EditMessageCaption, "start download stickerset notice", err)
			}
		}

		stickerData, err := download.GetStickerPack(opts.Ctx, opts.Thebot, stickerSet, needConvert)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to download sticker set")
			handlerErr.Addf("failed to download sticker set: %w", err)

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:    opts.CallbackQuery.From.ID,
				Text:      fmt.Sprintf("下载贴纸包时发生了一些错误\n<blockquote expandable>Failed to download sticker set: %s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "download sticker set error").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "download sticker set error", err)
			}
		} else {
			var pendingMessage string

			if needConvert {
				pendingMessage = fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 已下载\n包含 %d 个贴纸（经过转换）", stickerData.StickerSetName, stickerData.StickerSetTitle, stickerData.StickerCount)
			} else {
				pendingMessage = fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 已下载\n包含 %d 个贴纸", stickerData.StickerSetName, stickerData.StickerSetTitle, stickerData.StickerCount)
			}

			// 获取 stickerData.Data 的文件大小
			if stickerData.StickerSetSize > 50 * 1024 * 1024 {
				_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.CallbackQuery.From.ID,
					Text:      pendingMessage + "<blockquote>由于文件大小超过 50 MB，请点击下方按钮下载\n注意：此链接并不会一直有效</blockquote>",
					ParseMode: models.ParseModeHTML,
					LinkPreviewOptions: &models.LinkPreviewOptions{ IsDisabled: bot.True() },
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.CallbackQuery.Message.Message.ID },
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "打开浏览器下载",
						URL:  fmt.Sprintf("https://alist.trle5.xyz/d/cache/sticker_compressed/%s_%s", stickerData.StickerSetHash, stickerData.StickerSetFileName),
					}}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "download sticker set error").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "download sticker set error", err)
				}
			} else {
				_, err = opts.Thebot.SendDocument(opts.Ctx, &bot.SendDocumentParams{
					ChatID:    opts.CallbackQuery.From.ID,
					Document:  &models.InputFileUpload{Filename: stickerData.StickerSetFileName, Data: stickerData.Data},
					Caption:   pendingMessage,
					ParseMode: models.ParseModeHTML,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.CallbackQuery.Message.Message.ID },
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "sticker set zip file").
						Msg(flaterr.SendDocument.Str())
					handlerErr.Addt(flaterr.SendDocument, "sticker set zip file", err)
				}
			}
		}
	}

	return handlerErr.Flat()
}

// full command for "t.me/addstickers/" or "https://t.me/addstickers/"
func stickerLinkHandler(opts *handler_params.Message) error {
	if opts.Message == nil || opts.Message.Text == "" {
		return nil
	}

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var handlerErr flaterr.MultErr
	var stickerSetName string

	if strings.HasPrefix(opts.Message.Text, "https://t.me/addstickers/") {
		stickerSetName = strings.TrimPrefix(opts.Message.Text, "https://t.me/addstickers/")
	} else if strings.HasPrefix(opts.Message.Text, "t.me/addstickers/") {
		stickerSetName = strings.TrimPrefix(opts.Message.Text, "t.me/addstickers/")
	}

	if stickerSetName != "" {
		stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{Name: stickerSetName})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(opts.Message.From)).
				Str("stickerSetName", stickerSetName).
				Msg(flaterr.GetStickerSet.Str())
			handlerErr.Addt(flaterr.GetStickerSet, stickerSetName, err)

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.From.ID,
				Text:            fmt.Sprintf("获取贴纸包信息时发生了一些错误\n<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
				ParseMode:       models.ParseModeHTML,
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Message.From)).
					Str("content", "get sticker set info error").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "get sticker set info error", err)
			}
		} else {
			var suffix string = stickerSet.Name
			if len(stickerSet.Name) > 62 {
				id := config.Config.GetOversizeSetIDByName(stickerSet.Name)
				if id == 0 {
					config.Config.OversizeSetCount++
					config.Config.OversizeSets = append(config.Config.OversizeSets, config.OversizeSet{
						SetName: stickerSet.Name,
						SetID:   config.Config.OversizeSetCount,
					})
					suffix = fmt.Sprintf("_%d", config.Config.OversizeSetCount)
				} else {
					suffix = fmt.Sprintf("_%d", id)
				}
			}
			var button [][]models.InlineKeyboardButton

			if config.Config.AllowDownloadStickerSet {
				if config.Config.DisableConvert {
					button = [][]models.InlineKeyboardButton{
						{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", suffix) }},
					}
				} else {
					button = [][]models.InlineKeyboardButton{
						{{ Text: "下载转换后的贴纸包", CallbackData: fmt.Sprintf("S_%s", suffix) }},
						{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", suffix) }},
					}
				}
			}

			if config.Config.UseCollcetSticker {
				collect.AddButton(stickerSet.Name, len(stickerSet.Stickers), contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...), &button)
			}

			var ReplyMarkup models.ReplyMarkup
			if len(button) > 0 {
				ReplyMarkup = &models.InlineKeyboardMarkup{
					InlineKeyboard: button,
				}
			}

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:              opts.Message.From.ID,
				Text:                fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包中一共有 %d 个贴纸\n", stickerSet.Name, stickerSet.Title, len(stickerSet.Stickers)),
				ParseMode:           models.ParseModeHTML,
				DisableNotification: true,
				ReplyParameters:     &models.ReplyParameters{MessageID: opts.Message.ID},
				ReplyMarkup:         ReplyMarkup,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Message.From)).
					Str("content", "sticker set info").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "sticker set info", err)
			}
		}
	} else {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.From.ID,
			Text:            "请发送一个有效的贴纸链接",
			ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(opts.Message.From)).
				Str("content", "empty sticker link notice").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "empty sticker link notice", err)
		}
	}

	return handlerErr.Flat()
}
