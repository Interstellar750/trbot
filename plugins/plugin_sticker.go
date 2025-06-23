package plugins

import (
	"archive/zip"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"trbot/database"
	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/handler_structs"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
	"golang.org/x/image/webp"
)

var StickerCache_path    string = filepath.Join(consts.CacheDirectory, "sticker/")
var StickerCachePNG_path string = filepath.Join(consts.CacheDirectory, "sticker_png/")
var StickerCacheGIF_path string = filepath.Join(consts.CacheDirectory, "sticker_gif/")
var StickerCacheZip_path string = filepath.Join(consts.CacheDirectory, "sticker_zip/")

func init() {
	plugin_utils.AddCallbackQueryCommandPlugins([]plugin_utils.CallbackQuery{
		{
			// 不转换格式，打包下载整个贴纸包
			CommandChar: "s",
			Handler: DownloadStickerPackCallBackHandler,
		},
		{
			// 将贴纸包中的静态贴纸全部转换为 PNG 格式并打包
			CommandChar: "S",
			Handler: DownloadStickerPackCallBackHandler,
		},
	}...)
	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "贴纸下载",
		Description: "直接向机器人发送任意贴纸来下载转换后的 PNG 格式图片\n\n<blockquote expandable>仅限静态贴纸会被转换，动画和视频贴纸将会以原文件形式发送\n若您发送的贴纸为一个贴纸包中的贴纸，您可以点击消息中的按钮来下载整个贴纸包</blockquote>",
		ParseMode:   models.ParseModeHTML,
	})
	plugin_utils.AddHandlerByMessageTypePlugins(plugin_utils.HandlerByMessageType{
		PluginName:      "StickerDownload",
		ChatType:         models.ChatTypePrivate,
		MessageType:      message_utils.Sticker,
		AllowAutoTrigger: true,
		Handler:          EchoStickerHandler,
	})
}

type stickerDatas struct {
	Data            io.Reader
	IsConverted     bool
	IsCustomSticker bool
	StickerCount    int
	StickerIndex    int
	StickerSetName  string // 贴纸包的 urlname
	StickerSetTitle string // 贴纸包名称
}

func EchoStickerHandler(opts *handler_structs.SubHandlerParams) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str("funcName", "EchoStickerHandler").
		Logger()
	
	if opts.Update.Message == nil && opts.Update.CallbackQuery != nil && strings.HasPrefix(opts.Update.CallbackQuery.Data, "HBMT_") && opts.Update.CallbackQuery.Message.Message != nil && opts.Update.CallbackQuery.Message.Message.ReplyToMessage != nil {
		// if this handler tigger by `handler by message type`, copy `update.CallbackQuery.Message.Message.ReplyToMessage` to `update.Message`
		opts.Update.Message = opts.Update.CallbackQuery.Message.Message.ReplyToMessage
		logger.Debug().
			Str("callbackQueryData", opts.Update.CallbackQuery.Data).
			Msg("copy `update.CallbackQuery.Message.Message.ReplyToMessage` to `update.Message`")
	}

	logger.Debug().
		Str("emoji", opts.Update.Message.Sticker.Emoji).
		Str("setName", opts.Update.Message.Sticker.SetName).
		Msg("start download sticker")

	err := database.IncrementalUsageCount(opts.Ctx, opts.Update.Message.From.ID, db_struct.StickerDownloaded)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(opts.Update.Message.From)).
			Msg("Incremental sticker download count error")
	}

	stickerData, err := EchoSticker(opts)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(opts.Update.Message.From)).
			Msg("Failed to download sticker")

		_, msgerr := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Update.Message.From.ID,
			Text:      fmt.Sprintf("下载贴纸时发生了一些错误\n<blockquote expandable>Failed to download sticker: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
		if msgerr != nil {
			logger.Error().
				Err(msgerr).
				Dict(utils.GetUserDict(opts.Update.Message.From)).
				Msg("Failed to send `sticker download error` message")
		}
		return err
	}

	documentParams := &bot.SendDocumentParams{
		ChatID:                      opts.Update.Message.From.ID,
		ParseMode:                   models.ParseModeHTML,
		ReplyParameters:             &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
		DisableNotification:         true,
		DisableContentTypeDetection: true, // Prevent the server convert gif to mp4
	}

	var stickerFilePrefix, stickerFileSuffix string

	if opts.Update.Message.Sticker.IsVideo {
		if stickerData.IsConverted {
			stickerFileSuffix = "gif"
		} else {
			documentParams.Caption = "<blockquote>see <a href=\"https://wikipedia.org/wiki/WebM\">wikipedia/WebM</a></blockquote>"
			stickerFileSuffix = "webm"
		}
	} else if opts.Update.Message.Sticker.IsAnimated {
		documentParams.Caption = "<blockquote>see <a href=\"https://core.telegram.org/stickers#animated-stickers\">stickers/animated-stickers</a></blockquote>"
		stickerFileSuffix = "tgs.file"
	} else {
		stickerFileSuffix = "png"
	}

	if stickerData.IsCustomSticker {
		stickerFilePrefix = "sticker"
	} else {
		stickerFilePrefix = fmt.Sprintf("%s_%d", stickerData.StickerSetName, stickerData.StickerIndex)
	
		// 仅在不为自定义贴纸时显示下载整个贴纸包按钮
		documentParams.Caption += fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包中一共有 %d 个贴纸\n", stickerData.StickerSetName, stickerData.StickerSetTitle, stickerData.StickerCount)
		documentParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{ Text: "下载贴纸包中的静态贴纸", CallbackData: fmt.Sprintf("S_%s", opts.Update.Message.Sticker.SetName) },
			},
			{
				{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", opts.Update.Message.Sticker.SetName) },
			},
		}}
	}

	documentParams.Document = &models.InputFileUpload{ Filename: fmt.Sprintf("%s.%s", stickerFilePrefix, stickerFileSuffix), Data: stickerData.Data }

	_, err = opts.Thebot.SendDocument(opts.Ctx, documentParams)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(opts.Update.Message.From)).
			Msg("Failed to send sticker file to user")
		return err
	}

	return nil
}

func EchoSticker(opts *handler_structs.SubHandlerParams) (*stickerDatas, error) {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str("funcName", "EchoSticker").
		Logger()

	var data stickerDatas
	var fileSuffix string // `webp`, `webm`, `tgs`

	// 根据贴纸类型设置文件扩展名
	if opts.Update.Message.Sticker.IsVideo {
		fileSuffix = "webm"
	} else if opts.Update.Message.Sticker.IsAnimated {
		fileSuffix = "tgs"
	} else {
		fileSuffix = "webp"
	}

	var stickerFileNameWithDot string // `CAACAgUA.` or `duck_2_video 5 CAACAgUA.`
	var stickerSetNamePrivate  string // `-custom` or `duck_2_video`

	// 检查一下贴纸是否有 packName，没有的话就是自定义贴纸
	if opts.Update.Message.Sticker.SetName != "" {
		// 获取贴纸包信息
		stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: opts.Update.Message.Sticker.SetName })
		if err != nil {
			// this sticker has a setname, but that sticker set has been deleted
			// it may also be because of an error in obtaining sticker there are no static stickers in the sticker set information, so just let the user try again.
			logger.Warn().
				Err(err).
				Str("setName", opts.Update.Message.Sticker.SetName).
				Msg("Failed to get sticker set info, download it as a custom sticker")
			
			// 到这里是因为用户发送的贴纸对应的贴纸包已经被删除了，但贴纸中的信息还有对应的 SetName，会触发查询，但因为贴纸包被删了就查不到，将 index 值设为 -1，缓存后当作自定义贴纸继续
			data.IsCustomSticker   = true
			stickerSetNamePrivate  = opts.Update.Message.Sticker.SetName
			stickerFileNameWithDot = fmt.Sprintf("%s %d %s.", opts.Update.Message.Sticker.SetName, -1, opts.Update.Message.Sticker.FileID)
		} else {
			// sticker is in a sticker set
			data.StickerCount    = len(stickerSet.Stickers)
			data.StickerSetName  = stickerSet.Name
			data.StickerSetTitle = stickerSet.Title

			// 寻找贴纸在贴纸包中的索引并赋值
			for i, n := range stickerSet.Stickers {
				if n.FileID == opts.Update.Message.Sticker.FileID {
					data.StickerIndex = i
					break
				}
			}

			// 在这个条件下，贴纸包名和贴纸索引都存在，赋值完整的贴纸文件名
			stickerSetNamePrivate  = opts.Update.Message.Sticker.SetName
			stickerFileNameWithDot = fmt.Sprintf("%s %d %s.", opts.Update.Message.Sticker.SetName, data.StickerIndex, opts.Update.Message.Sticker.FileID)
		}
	} else {
		// this sticker doesn't have a setname, so it is a custom sticker
		// 自定义贴纸，防止与普通贴纸包冲突，将贴纸包名设置为 `-custom`，文件名仅有 FileID 用于辨识
		data.IsCustomSticker   = true
		stickerSetNamePrivate  = "-custom"
		stickerFileNameWithDot = fmt.Sprintf("%s.", opts.Update.Message.Sticker.FileID)
	}

	var filePath       string = filepath.Join(StickerCache_path, stickerSetNamePrivate)      // 保存贴纸源文件的目录 .cache/sticker/setName/
	var originFullPath string = filepath.Join(filePath, stickerFileNameWithDot + fileSuffix) // 到贴纸文件的完整目录 .cache/sticker/setName/stickerFileName.webp

	var PNGFilePath   string = filepath.Join(StickerCachePNG_path, stickerSetNamePrivate) // 转码后为 png 格式的目录 .cache/sticker_png/setName/
	var toPNGFullPath string = filepath.Join(PNGFilePath, stickerFileNameWithDot + "png") // 转码后到 png 格式贴纸的完整目录 .cache/sticker_png/setName/stickerFileName.png

	var GIFFilePath   string = filepath.Join(StickerCacheGIF_path, stickerSetNamePrivate) // 转码后为 png 格式的目录 .cache/sticker_png/setName/
	var toGIFFullPath string = filepath.Join(GIFFilePath, stickerFileNameWithDot + "gif") // 转码后到 png 格式贴纸的完整目录 .cache/sticker_png/setName/stickerFileName.png

	_, err := os.Stat(originFullPath) // 检查贴纸源文件是否已缓存
	if err != nil {
		// 如果文件不存在，进行下载，否则返回错误
		if os.IsNotExist(err) {
			// 日志提示该文件没被缓存，正在下载
			logger.Trace().
				Str("originFullPath", originFullPath).
				Msg("sticker file not cached, downloading")

			// 从服务器获取文件信息
			fileinfo, err := opts.Thebot.GetFile(opts.Ctx, &bot.GetFileParams{ FileID: opts.Update.Message.Sticker.FileID })
			if err != nil {
				logger.Error().
					Err(err).
					Str("fileID", opts.Update.Message.Sticker.FileID).
					Msg("Failed to get sticker file info")
				return nil, fmt.Errorf("failed to get sticker file [%s] info: %w", opts.Update.Message.Sticker.FileID, err)
			}

			// 组合链接下载贴纸源文件
			resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", configs.BotConfig.BotToken, fileinfo.FilePath))
			if err != nil {
				logger.Error().
					Err(err).
					Str("filePath", fileinfo.FilePath).
					Msg("Failed to download sticker file")
				return nil, fmt.Errorf("failed to download sticker file [%s]: %w", fileinfo.FilePath, err)
			}
			defer resp.Body.Close()

			// 创建保存贴纸的目录
			err = os.MkdirAll(filePath, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("filePath", filePath).
					Msg("Failed to create sticker directory to save sticker")
				return nil, fmt.Errorf("failed to create directory [%s] to save sticker: %w", filePath, err)
			}

			// 创建贴纸空文件
			downloadedSticker, err := os.Create(originFullPath)
			if err != nil {
				logger.Error().
					Err(err).
					Str("originFullPath", originFullPath).
					Msg("Failed to create sticker file")
				return nil, fmt.Errorf("failed to create sticker file [%s]: %w", originFullPath, err)
			}
			defer downloadedSticker.Close()

			// 将下载的原贴纸写入空文件
			_, err = io.Copy(downloadedSticker, resp.Body)
			if err != nil {
				logger.Error().
					Err(err).
					Str("originFullPath", originFullPath).
					Msg("Failed to writing sticker data to file")
				return nil, fmt.Errorf("failed to writing sticker data to file [%s]: %w", originFullPath, err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("originFullPath", originFullPath).
				Msg("Failed to read cached sticker file info")
			return nil, fmt.Errorf("failed to read cached sticker file [%s] info: %w", originFullPath, err)
		}
	} else {
		// 文件已存在，跳过下载
		logger.Trace().
			Str("originFullPath", originFullPath).
			Msg("sticker file already cached")
	}

	var finalFullPath string // 存放最后读取并发送的文件完整目录 .cache/sticker/setName/stickerFileName.webp

	if opts.Update.Message.Sticker.IsAnimated {
		// tgs
		// 不需要转码，直接读取原贴纸文件
		finalFullPath = originFullPath
	} else if opts.Update.Message.Sticker.IsVideo {
		if configs.BotConfig.FFmpegPath != "" {
			// webm, convert to gif
			_, err = os.Stat(toGIFFullPath) // 使用目录提前检查一下是否已经转换过
			if err != nil {
				// 如果提示不存在，进行转换
				if os.IsNotExist(err) {
					// 日志提示该文件没转换，正在转换
					logger.Trace().
						Str("toGIFFullPath", toGIFFullPath).
						Msg("sticker file does not convert, converting")

					// 创建保存贴纸的目录
					err = os.MkdirAll(GIFFilePath, 0755)
					if err != nil {
						logger.Error().
							Err(err).
							Str("GIFFilePath", GIFFilePath).
							Msg("Failed to create directory to convert file")
						return nil, fmt.Errorf("failed to create directory [%s] to convert sticker file: %w", GIFFilePath, err)
					}

					// 读取原贴纸文件，转码后存储到 png 格式贴纸的完整目录
					err = convertWebmToGif(originFullPath, toGIFFullPath)
					if err != nil {
						logger.Error().
							Err(err).
							Str("originFullPath", originFullPath).
							Msg("Failed to convert webm to gif")
						return nil, fmt.Errorf("failed to convert webm [%s] to gif: %w", originFullPath, err)
					}
				} else {
					// 其他错误
					logger.Error().
						Err(err).
						Str("toGIFFullPath", toGIFFullPath).
						Msg("Failed to read converted file info")
					return nil, fmt.Errorf("failed to read converted sticker file [%s] info: %w", toGIFFullPath, err)
				}
			} else {
				// 文件存在，跳过转换
				logger.Trace().
					Str("toGIFFullPath", toGIFFullPath).
					Msg("sticker file already converted to gif")
			}

			// 处理完成，将最后要读取的目录设为转码后 gif 格式贴纸的完整目录
			data.IsConverted = true
			finalFullPath = toGIFFullPath
		} else {
			// 没有 ffmpeg 能用来转码，直接读取原贴纸文件
			finalFullPath = originFullPath
		}
	} else {
		// webp, need convert to png
		_, err = os.Stat(toPNGFullPath) // 使用目录提前检查一下是否已经转换过
		if err != nil {
			// 如果提示不存在，进行转换
			if os.IsNotExist(err) {
				// 日志提示该文件没转换，正在转换
				logger.Trace().
					Str("toPNGFullPath", toPNGFullPath).
					Msg("sticker file does not convert, converting")

				// 创建保存贴纸的目录
				err = os.MkdirAll(PNGFilePath, 0755)
				if err != nil {
					logger.Error().
						Err(err).
						Str("PNGFilePath", PNGFilePath).
						Msg("Failed to create directory to convert sticker")
					return nil, fmt.Errorf("failed to create directory [%s] to convert sticker: %w", PNGFilePath, err)
				}

				// 读取原贴纸文件，转码后存储到 png 格式贴纸的完整目录
				err = convertWebPToPNG(originFullPath, toPNGFullPath)
				if err != nil {
					logger.Error().
						Err(err).
						Str("originFullPath", originFullPath).
						Msg("Failed to convert webp to png")
					return nil, fmt.Errorf("failed to convert webp [%s] to png: %w", originFullPath, err)
				}
			} else {
				// 其他错误
				logger.Error().
					Err(err).
					Str("toPNGFullPath", toPNGFullPath).
					Msg("Failed to read converted sticker file info")
				return nil, fmt.Errorf("failed to read converted png sticker file [%s] info : %w", toPNGFullPath, err)
			}
		} else {
			// 文件存在，跳过转换
			logger.Trace().
				Str("toPNGFullPath", toPNGFullPath).
				Msg("sticker file already converted to png")
		}

		// 处理完成，将最后要读取的目录设为转码后 png 格式贴纸的完整目录
		data.IsConverted = true
		finalFullPath = toPNGFullPath
	}

	// 逻辑完成，读取最后的文件，返回给上一级函数
	data.Data, err = os.Open(finalFullPath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("finalFullPath", finalFullPath).
			Msg("Failed to open sticker file")
		return nil, fmt.Errorf("failed to open sticker file [%s]: %w", finalFullPath, err)
	}

	return &data, nil
}

func DownloadStickerPackCallBackHandler(opts *handler_structs.SubHandlerParams) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str("funcName", "DownloadStickerPackCallBackHandler").
		Logger()

	botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "已请求下载，请稍候",
		ParseMode: models.ParseModeMarkdownV1,
		DisableNotification: true,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
			Msg("Failed to send `start download stickerset` message")
	}

	err = database.IncrementalUsageCount(opts.Ctx, opts.Update.CallbackQuery.Message.Message.Chat.ID, db_struct.StickerSetDownloaded)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(opts.Update.Message.From)).
			Msg("Incremental sticker set download count error")
	}

	var packName  string
	var isOnlyPNG bool
	if opts.Update.CallbackQuery.Data[0:2] == "S_" {
		packName = strings.TrimPrefix(opts.Update.CallbackQuery.Data, "S_")
		isOnlyPNG = true
	} else {
		packName = strings.TrimPrefix(opts.Update.CallbackQuery.Data, "s_")
		isOnlyPNG = false
	}

	// 通过贴纸的 packName 获取贴纸集
	stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: packName })
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
			Msg("Failed to get sticker set info")

		_, msgerr := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.CallbackQuery.From.ID,
			Text:   fmt.Sprintf("获取贴纸包时发生了一些错误\n<blockquote expandable>Failed to get sticker set info: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
		if msgerr != nil {
			logger.Error().
				Err(msgerr).
				Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
				Msg("Failed to send `get sticker set info error` message")
		}
		return err
	}

	stickerData, err := getStickerPack(opts, stickerSet, isOnlyPNG)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
			Msg("Failed to download sticker set")

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.CallbackQuery.From.ID,
			Text:   fmt.Sprintf("下载贴纸包时发生了一些错误\n<blockquote expandable>Failed to download sticker set: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
				Msg("Failed to send `download sticker set error` message")
		}
		return err
	}

	documentParams := &bot.SendDocumentParams{
		ChatID: opts.Update.CallbackQuery.From.ID,
		ParseMode: models.ParseModeMarkdownV1,
	}

	if isOnlyPNG {
		documentParams.Caption  = fmt.Sprintf("[%s](https://t.me/addstickers/%s) 已下载\n包含 %d 个贴纸（仅转换后的 PNG 格式）", stickerData.StickerSetTitle, stickerData.StickerSetName, stickerData.StickerCount)
		documentParams.Document = &models.InputFileUpload{Filename: fmt.Sprintf("%s(%d)_png.zip", stickerData.StickerSetName, stickerData.StickerCount), Data: stickerData.Data}
	} else {
		documentParams.Caption  = fmt.Sprintf("[%s](https://t.me/addstickers/%s) 已下载\n包含 %d 个贴纸", stickerData.StickerSetTitle, stickerData.StickerSetName, stickerData.StickerCount)
		documentParams.Document = &models.InputFileUpload{Filename: fmt.Sprintf("%s(%d).zip", stickerData.StickerSetName, stickerData.StickerCount), Data: stickerData.Data}
	}

	_, err = opts.Thebot.SendDocument(opts.Ctx, documentParams)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
			Msg("Failed to send sticker set zip file to user")
	}

	_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
		ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: botMessage.ID,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
			Msg("Failed to delete `start download stickerset` message")
	}

	return nil
}

func getStickerPack(opts *handler_structs.SubHandlerParams, stickerSet *models.StickerSet, isOnlyPNG bool) (*stickerDatas, error) {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str("funcName", "getStickerPack").
		Logger()

	var data stickerDatas = stickerDatas{
		IsCustomSticker: false,
		StickerSetName:  stickerSet.Name,
		StickerSetTitle: stickerSet.Title,
	}

	logger.Debug().
		Dict("stickerSet", zerolog.Dict().
			Str("title", data.StickerSetTitle).
			Str("name", data.StickerSetName).
			Int("allCount", len(stickerSet.Stickers)),
		).
		Msg("start download sticker set")

	filePath    := filepath.Join(StickerCache_path, stickerSet.Name)
	PNGFilePath := filepath.Join(StickerCachePNG_path, stickerSet.Name)

	var allCached    bool = true
	var allConverted bool = true

	var stickerCount_webm int
	var stickerCount_tgs  int
	var stickerCount_webp int

	for i, sticker := range stickerSet.Stickers {
		stickerfileName := fmt.Sprintf("%s %d %s.", sticker.SetName, i, sticker.FileID)
		var fileSuffix string

		// 根据贴纸类型设置文件扩展名和统计贴纸数量
		if sticker.IsVideo {
			fileSuffix = "webm"
			stickerCount_webm++
		} else if sticker.IsAnimated {
			fileSuffix = "tgs"
			stickerCount_tgs++
		} else {
			fileSuffix = "webp"
			stickerCount_webp++
		}

		var originFullPath string = filepath.Join(filePath,    stickerfileName + fileSuffix)
		var toPNGFullPath  string = filepath.Join(PNGFilePath, stickerfileName + "png")

		_, err := os.Stat(originFullPath) // 检查单个贴纸是否已缓存
		if err != nil {
			if os.IsNotExist(err) {
				allCached = false
				logger.Trace().
					Str("originFullPath", originFullPath).
					Str("stickerSetName", data.StickerSetName).
					Int("stickerIndex", i).
					Msg("sticker file not cached, downloading")

				// 从服务器获取文件内容
				fileinfo, err := opts.Thebot.GetFile(opts.Ctx, &bot.GetFileParams{ FileID: sticker.FileID })
				if err != nil {
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("fileID", opts.Update.Message.Sticker.FileID).
						Msg("error getting sticker file info")
					return nil, fmt.Errorf("error getting file info %s: %v", sticker.FileID, err)
				}

				// 下载贴纸文件
				resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", configs.BotConfig.BotToken, fileinfo.FilePath))
				if err != nil {
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("filePath", fileinfo.FilePath).
						Msg("error downloading sticker file")
					return nil, fmt.Errorf("error downloading file %s: %v", fileinfo.FilePath, err)
				}
				defer resp.Body.Close()

				err = os.MkdirAll(filePath, 0755)
				if err != nil {
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("filePath", filePath).
						Msg("Failed to creat directory to save sticker")
					return nil, fmt.Errorf("failed to create directory [%s] to save sticker: %w", filePath, err)
				}

				// 创建文件并保存
				downloadedSticker, err := os.Create(originFullPath)
				if err != nil {
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("originFullPath", originFullPath).
						Msg("Failed to create sticker file")
					return nil, fmt.Errorf("failed to create sticker file [%s]: %w", originFullPath, err)
				}
				defer downloadedSticker.Close()

				// 将下载的内容写入文件
				_, err = io.Copy(downloadedSticker, resp.Body)
				if err != nil {
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("originFullPath", originFullPath).
						Msg("Failed to writing sticker data to file")

					return nil, fmt.Errorf("failed to writing sticker data to file [%s]: %w", originFullPath, err)
				}
			} else {
				logger.Error().
					Err(err).
					Int("stickerIndex", i).
					Str("originFullPath", originFullPath).
					Msg("Failed to read cached sticker file info")
				return nil, fmt.Errorf("failed to read cached sticker file [%s] info: %w", originFullPath, err)
			}
		} else {
			// 存在跳过下载过程
			logger.Trace().
				Int("stickerIndex", i).
				Str("originFullPath", originFullPath).
				Msg("sticker file already exists")
		}

		// 仅需要 PNG 格式时进行转换
		if isOnlyPNG && !sticker.IsVideo && !sticker.IsAnimated {
			_, err = os.Stat(toPNGFullPath)
			if err != nil {
				if os.IsNotExist(err) {
					allConverted = false
					logger.Trace().
						Int("stickerIndex", i).
						Str("toPNGFullPath", toPNGFullPath).
						Msg("file does not convert, converting")
					// 创建保存贴纸的目录
					err = os.MkdirAll(PNGFilePath, 0755)
					if err != nil {
						logger.Error().
							Err(err).
							Int("stickerIndex", i).
							Str("PNGFilePath", PNGFilePath).
							Msg("error creating directory to convert file")
						return nil, fmt.Errorf("error creating directory %s: %w", PNGFilePath, err)
					}
					// 将 webp 转换为 png
					err = convertWebPToPNG(originFullPath, toPNGFullPath)
					if err != nil {
						logger.Error().
							Err(err).
							Int("stickerIndex", i).
							Str("originFullPath", originFullPath).
							Msg("error converting webp to png")
						return nil, fmt.Errorf("error converting webp to png %s: %w", originFullPath, err)
					}
				} else {
					// 其他错误
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("toPNGFullPath", toPNGFullPath).
						Msg("error when reading converted file info")
					return nil, fmt.Errorf("error when reading converted file info: %w", err)
				}
			} else {
				logger.Trace().
					Str("toPNGFullPath", toPNGFullPath).
					Msg("file already converted")
			}
		}
	}

	var zipFileName string
	var compressFolderPath string

	var isZiped bool = true

	// 根据要下载的类型设置压缩包的文件名和路径以及压缩包中的贴纸数量
	if isOnlyPNG {
		if stickerCount_webp == 0 {
			logger.Warn().
				Dict("stickerSet", zerolog.Dict().
					Str("stickerSetName", stickerSet.Name).
					Int("WebP", stickerCount_webp).
					Int("tgs", stickerCount_tgs).
					Int("WebM", stickerCount_webm),
				).
				Msg("there are no static stickers in the sticker set")
			return nil, fmt.Errorf("there are no static stickers in the sticker set")
		}
		data.StickerCount = stickerCount_webp
		zipFileName = fmt.Sprintf("%s(%d)_png.zip", stickerSet.Name, data.StickerCount)
		compressFolderPath = PNGFilePath
	} else {
		data.StickerCount = stickerCount_webp + stickerCount_webm + stickerCount_tgs
		zipFileName = fmt.Sprintf("%s(%d).zip", stickerSet.Name, data.StickerCount)
		compressFolderPath = filePath
	}

	var zipFileFullPath string = filepath.Join(StickerCacheZip_path, zipFileName)

	_, err := os.Stat(zipFileFullPath) // 检查压缩包文件是否存在
	if err != nil {
		if os.IsNotExist(err) {
			isZiped = false
			err = os.MkdirAll(StickerCacheZip_path, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("StickerCacheZip_path", StickerCacheZip_path).
					Msg("Failed to create zip file directory")
				return nil, fmt.Errorf("failed to create zip file directory [%s]: %w", StickerCacheZip_path, err)
			}
			err = zipFolder(compressFolderPath, zipFileFullPath)
			if err != nil {
				logger.Error().
					Err(err).
					Str("compressFolderPath", compressFolderPath).
					Msg("Failed to compress sticker folder")
				return nil, fmt.Errorf("failed to compress sticker folder [%s]: %w", compressFolderPath, err)
			}
			logger.Trace().
				Str("compressFolderPath", compressFolderPath).
				Str("zipFileFullPath", zipFileFullPath).
				Msg("Compress sticker folder successfully")
		} else {
			logger.Error().
				Err(err).
				Str("zipFileFullPath", zipFileFullPath).
				Msg("Failed to read compressed sticker set zip file info")
			return nil, fmt.Errorf("failed to read compressed sticker set zip file [%s] info: %w", zipFileFullPath, err)
		}
	} else {
		logger.Trace().
			Str("zipFileFullPath", zipFileFullPath).
			Msg("sticker set zip file already compressed")
	}

	// 读取压缩后的贴纸包
	data.Data, err = os.Open(zipFileFullPath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("zipFileFullPath", zipFileFullPath).
			Msg("Failed to open compressed sticker set zip file")
		return nil, fmt.Errorf("failed to open compressed sticker set zip file [%s]: %w", zipFileFullPath, err)
	}

	if isZiped {
		// 存在已经完成压缩的贴纸包（原始格式或已转换）
		logger.Info().
			Str("zipFileFullPath", zipFileFullPath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("sticker set already zipped")
	} else if isOnlyPNG && allConverted {
		// 仅需要 PNG 格式，且贴纸包完全转换成 PNG 格式，但尚未压缩
		logger.Info().
			Str("zipFileFullPath", zipFileFullPath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("sticker set already converted")
	} else if allCached {
		// 贴纸包中的贴纸已经全部缓存了
		logger.Info().
			Str("zipFileFullPath", zipFileFullPath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("sticker set already cached")
	} else {
		// 新下载的贴纸包（如果有部分已经下载了也是这个）
		logger.Info().
			Str("zipFileFullPath", zipFileFullPath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("sticker set already downloaded")
	}

	return &data, nil
}

func convertWebPToPNG(webpPath, pngPath string) error {
	// 打开 WebP 文件
	webpFile, err := os.Open(webpPath)
	if err != nil {
		return fmt.Errorf("打开 WebP 文件失败: %v", err)
	}
	defer webpFile.Close()

	// 解码 WebP 图片
	img, err := webp.Decode(webpFile)
	if err != nil {
		return fmt.Errorf("解码 WebP 失败: %v", err)
	}

	// 创建 PNG 文件
	pngFile, err := os.Create(pngPath)
	if err != nil {
		return fmt.Errorf("创建 PNG 文件失败: %v", err)
	}
	defer pngFile.Close()

	// 编码 PNG
	err = png.Encode(pngFile, img)
	if err != nil {
		return fmt.Errorf("编码 PNG 失败: %v", err)
	}

	return nil
}

// use ffmpeg
func convertWebmToGif(webmPath, gifPath string) error {
	cmd := exec.Command(configs.BotConfig.FFmpegPath, "-i", webmPath, "-vf", "fps=10", gifPath)
	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

func zipFolder(srcDir, zipFile string) error {
	// 创建 ZIP 文件
	outFile, err := os.Create(zipFile)
	if err != nil { return err }
	defer outFile.Close()

	// 创建 ZIP 写入器
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// 遍历文件夹并添加文件到 ZIP
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }

		// 计算文件在 ZIP 中的相对路径
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil { return err }

		// 如果是目录，则跳过
		if info.IsDir() { return nil }

		// 打开文件
		file, err := os.Open(path)
		if err != nil { return err }
		defer file.Close()

		// 创建 ZIP 内的文件
		zipFileWriter, err := zipWriter.Create(relPath)
		if err != nil { return err }

		// 复制文件内容到 ZIP
		_, err = io.Copy(zipFileWriter, file)
		return err
	})

	return err
}
