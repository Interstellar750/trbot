package sticker_download

import (
	"context"
	"fmt"
	"strings"
	"trbot/database"
	"trbot/database/db_struct"
	"trbot/plugins/sticker_download/collect"
	"trbot/plugins/sticker_download/config"
	"trbot/plugins/sticker_download/download"
	"trbot/plugins/sticker_download/mpeg4gif"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
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

	stickerData, err := download.GetSticker(opts)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error when downloading sticker")
		handlerErr.Addf("error when downloading sticker: %w", err)

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.From.ID,
			Text:            fmt.Sprintf("下载贴纸时发生了一些错误\n<blockquote expandable>Failed to download sticker: %s</blockquote>", err),
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

			if config.Config.AllowDownloadStickerSet {
				if config.Config.DisableConvert {
					button = [][]models.InlineKeyboardButton{
						{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", opts.Message.Sticker.SetName) }},
					}
				} else {
					button = [][]models.InlineKeyboardButton{
						{{ Text: "下载转换后的贴纸包", CallbackData: fmt.Sprintf("S_%s", opts.Message.Sticker.SetName) }},
						{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", opts.Message.Sticker.SetName) }},
					}
				}
				documentParams.Caption += fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包中一共有 %d 个贴纸\n", stickerData.StickerSetName, stickerData.StickerSetTitle, stickerData.StickerCount)
				stickerFilePrefix = fmt.Sprintf("%s_%d", stickerData.StickerSetName, stickerData.StickerIndex)
			} else {
				stickerFilePrefix = stickerData.StickerSetName
			}

			if config.Config.UseCollcetSticker {
				collect.AddButton(opts.Message.Sticker.SetName, contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...), &button)
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
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Str("content", "stickerset download disabled notice").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addf(flaterr.AnswerCallbackQuery.Str(), "stickerset download disabled notice", err)
		}
		return handlerErr.Flat()
	}

	err := database.IncrementalUsageCount(opts.Ctx, opts.CallbackQuery.From.ID, db_struct.StickerSetDownloaded)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
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
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("content", "stickerset convert disabled notice").
					Msg(flaterr.AnswerCallbackQuery.Str())
				handlerErr.Addf(flaterr.AnswerCallbackQuery.Str(), "stickerset convert disabled notice", err)
			}
			return handlerErr.Flat()
		}
	} else {
		setName = strings.TrimPrefix(opts.CallbackQuery.Data, "s_")
		needConvert = false
	}

	// 通过贴纸的 packName 获取贴纸集
	stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: setName })
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
			Str("setName", setName).
			Msg(flaterr.GetStickerSet.Str())
		handlerErr.Addt(flaterr.GetStickerSet, setName, err)

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.CallbackQuery.From.ID,
			Text:            fmt.Sprintf("获取贴纸包信息时发生了一些错误\n<blockquote expandable>Failed to get sticker set info: %s</blockquote>", err),
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.CallbackQuery.Message.Message.ID },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Str("content", "get sticker set info error").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "get sticker set info error", err)
		}
	} else {
		logger.Info().
			Dict("stickerSet", zerolog.Dict().
				Str("title", stickerSet.Title).
				Str("name", stickerSet.Name).
				Int("allCount", len(stickerSet.Stickers)),
			).
			Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
			Msg("Start download sticker set")

		_, err = opts.Thebot.EditMessageCaption(opts.Ctx, &bot.EditMessageCaptionParams{
			ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.CallbackQuery.Message.Message.ID,
			Caption:   fmt.Sprintf("正在下载%s <a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包，请稍候...", utils.TextForTrueOrFalse(needConvert, "并转换", ""), stickerSet.Name, stickerSet.Title),
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Str("content", "start download stickerset notice").
				Msg(flaterr.EditMessageCaption.Str())
			handlerErr.Addt(flaterr.EditMessageCaption, "start download stickerset notice", err)
		}

		stickerData, err := download.GetStickerPack(opts.Ctx, opts.Thebot, stickerSet, needConvert)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Msg("Failed to download sticker set")
			handlerErr.Addf("failed to download sticker set: %w", err)

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:    opts.CallbackQuery.From.ID,
				Text:      fmt.Sprintf("下载贴纸包时发生了一些错误\n<blockquote expandable>Failed to download sticker set: %s</blockquote>", err),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("content", "download sticker set error").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "download sticker set error", err)
			}
		} else {
			documentParams := &bot.SendDocumentParams{
				ChatID:    opts.CallbackQuery.From.ID,
				ParseMode: models.ParseModeHTML,
			}

			if needConvert {
				documentParams.Caption  = fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 已下载\n包含 %d 个贴纸（经过转换）", stickerData.StickerSetName, stickerData.StickerSetTitle, stickerData.StickerCount)
				documentParams.Document = &models.InputFileUpload{Filename: fmt.Sprintf("%s(%d)_converted.zip", stickerData.StickerSetName, stickerData.StickerCount), Data: stickerData.Data}
			} else {
				documentParams.Caption  = fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 已下载\n包含 %d 个贴纸", stickerData.StickerSetName, stickerData.StickerSetTitle, stickerData.StickerCount)
				documentParams.Document = &models.InputFileUpload{Filename: fmt.Sprintf("%s(%d).zip", stickerData.StickerSetName, stickerData.StickerCount), Data: stickerData.Data}
			}

			_, err = opts.Thebot.SendDocument(opts.Ctx, documentParams)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("content", "sticker set zip file").
					Msg(flaterr.SendDocument.Str())
				handlerErr.Addt(flaterr.SendDocument, "sticker set zip file", err)
			}
		}
	}

	return handlerErr.Flat()
}

// full command "t.me/addstickers/" or "https://t.me/addstickers/"
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
				Text:            fmt.Sprintf("获取贴纸包信息时发生了一些错误\n<blockquote expandable>%s</blockquote>", err),
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
			var button [][]models.InlineKeyboardButton

			if config.Config.AllowDownloadStickerSet {
				if config.Config.DisableConvert {
					button = [][]models.InlineKeyboardButton{
						{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", stickerSet.Name) }},
					}
				} else {
					button = [][]models.InlineKeyboardButton{
						{{ Text: "下载转换后的贴纸包", CallbackData: fmt.Sprintf("S_%s", stickerSet.Name) }},
						{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", stickerSet.Name) }},
					}
				}
			}

			if config.Config.UseCollcetSticker {
				collect.AddButton(stickerSet.Name, contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...), &button)
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
