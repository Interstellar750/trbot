package plugins

import (
	"archive/zip"
	"context"
	"fmt"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"trbot/database"
	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"trbot/utils/type/message_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
	"golang.org/x/image/webp"
)

var stickerCollect CollectedSticker
var stickerCollectPath string = filepath.Join(consts.YAMLDataBaseDir, "collectsticker/", consts.YAMLFileName)

var StickerCache_path    string = filepath.Join(consts.CacheDirectory, "sticker/")
var StickerCachePNG_path string = filepath.Join(consts.CacheDirectory, "sticker_png/")
var StickerCacheGIF_path string = filepath.Join(consts.CacheDirectory, "sticker_gif/")
var StickerCacheZip_path string = filepath.Join(consts.CacheDirectory, "sticker_zip/")

var MP4Cache_path string = filepath.Join(consts.CacheDirectory, "mp4/")
var GIFCache_path string = filepath.Join(consts.CacheDirectory, "mp4_GIF/")

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "collectsitcker",
		Func: readCollectStickerList,
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "collectsitcker",
		Saver:  saveCollectStickerList,
		Loader: readCollectStickerList,
	})
	plugin_utils.AddCallbackQueryHandlers([]plugin_utils.CallbackQuery{
		{
			// 不转换格式，打包下载整个贴纸包
			CallbackDataPrefix: "s",
			CallbackQueryHandler: DownloadStickerPackCallbackHandler,
		},
		{
			// 将贴纸包中的静态贴纸全部转换为 PNG 格式并打包
			CallbackDataPrefix: "S",
			CallbackQueryHandler: DownloadStickerPackCallbackHandler,
		},
		{
			// 下载贴纸包中的所有贴纸并打包发送到收藏频道
			CallbackDataPrefix: "c",
			CallbackQueryHandler: collectStickerSet,
		},
	}...)
	plugin_utils.AddFullCommandHandlers([]plugin_utils.FullCommand{
		{
			FullCommand:    "https://t.me/addstickers/",
			ForChatType:    []models.ChatType{models.ChatTypePrivate},
			MessageHandler: getStickerPackInfo,
		},
		{
			FullCommand:    "t.me/addstickers/",
			ForChatType:    []models.ChatType{models.ChatTypePrivate},
			MessageHandler: getStickerPackInfo,
		},
	}...)
	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "贴纸下载",
		Description: "直接向机器人发送任意贴纸来下载转换后的 PNG 格式图片\n\n<blockquote expandable>仅限静态贴纸会被转换，动画和视频贴纸将会以原文件形式发送\n若您发送的贴纸为一个贴纸包中的贴纸，您可以点击消息中的按钮来下载整个贴纸包</blockquote>",
		ParseMode:   models.ParseModeHTML,
	})
	plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
		PluginName:       "下载贴纸",
		ChatType:         models.ChatTypePrivate,
		MessageType:      message_utils.Sticker,
		AllowAutoTrigger: true,
		MessageHandler:   EchoStickerHandler,
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "cachedsticker",
		MessageHandler: showCachedStickers,
	})
	plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
		PluginName:       "MP4 转 GIF",
		ChatType:         models.ChatTypePrivate,
		MessageType:      message_utils.Animation,
		AllowAutoTrigger: true,
		MessageHandler:   convertMP4ToGifHandler,
	})
}

type CollectedSticker struct {
	ChannelID  int64        `yaml:"ChannelID"`
	StickerSet []stickerSetInfo `yaml:"StickerSet"` // 已收藏的贴纸包列表
}

func (cs CollectedSticker) GetStickerSetByName(name string) *stickerSetInfo {
	for i, set := range cs.StickerSet {
		if set.Name == name {
			return &cs.StickerSet[i]
		}
	}
	return nil
}

type stickerSetInfo struct {
	Title  string `yaml:"Title"`  // 贴纸包的名称
	Name   string `yaml:"Name"`   // 贴纸包的 urlname
	MsgID  int    `yaml:"MsgID"`  // 发送到频道的消息 ID
	Count  int    `yaml:"Count"`  // 贴纸包中的贴纸数量
}

type stickerDatas struct {
	Data            io.Reader
	IsConverted     bool
	IsCustomSticker bool
	StickerCount    int
	StickerIndex    int
	StickerSetName  string // 贴纸包的 urlname
	StickerSetTitle string // 贴纸包名称

	WebP int
	WebM int
	tgs  int
}

func EchoStickerHandler(opts *handler_params.Message) error {
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

	stickerData, err := EchoSticker(opts)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error when downloading sticker")
		handlerErr.Addf("error when downloading sticker: %w", err)

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Message.From.ID,
			Text:      fmt.Sprintf("下载贴纸时发生了一些错误\n<blockquote expandable>Failed to download sticker: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
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
			documentParams.Caption = "<blockquote>see <a href=\"https://core.telegram.org/stickers#animated-stickers\">stickers/animated-stickers</a></blockquote>"
			stickerFileSuffix = "tgs.file"
		} else {
			stickerFileSuffix = "png"
		}

		if stickerData.IsCustomSticker {
			stickerFilePrefix = "sticker"
		} else {
			var button [][]models.InlineKeyboardButton = [][]models.InlineKeyboardButton{
				{{ Text: "下载贴纸包中的静态贴纸", CallbackData: fmt.Sprintf("S_%s", opts.Message.Sticker.SetName) }},
				{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", opts.Message.Sticker.SetName) }},
			}

			if stickerCollect.ChannelID != 0 && contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...) {
				for _, set := range stickerCollect.StickerSet {
					if set.Name == opts.Message.Sticker.SetName {
						button = append(button, [][]models.InlineKeyboardButton{{
							{
								Text: fmt.Sprintf("🔁 更新? (%d)>(%d)", set.Count, stickerData.StickerCount),
								CallbackData: fmt.Sprintf("c_%s", stickerData.StickerSetName),
							},
							{
								Text: "✅ 已收藏至频道",
								URL: utils.MsgLinkPrivate(stickerCollect.ChannelID, set.MsgID),
							},
						}}...)
						break
					}
				}
				if len(button) == 2 {
					button = append(button, []models.InlineKeyboardButton{{
						Text: "⭐️ 收藏至频道",
						CallbackData: fmt.Sprintf("c_%s", stickerData.StickerSetName),
					}})
				}
			}

			stickerFilePrefix = fmt.Sprintf("%s_%d", stickerData.StickerSetName, stickerData.StickerIndex)

			// 仅在不为自定义贴纸时显示下载整个贴纸包按钮
			documentParams.Caption += fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包中一共有 %d 个贴纸\n", stickerData.StickerSetName, stickerData.StickerSetTitle, stickerData.StickerCount)
			documentParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: button }
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

func EchoSticker(opts *handler_params.Message) (*stickerDatas, error) {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var data stickerDatas
	var fileSuffix string // `webp`, `webm`, `tgs`

	// 根据贴纸类型设置文件扩展名
	if opts.Message.Sticker.IsVideo {
		fileSuffix = "webm"
	} else if opts.Message.Sticker.IsAnimated {
		fileSuffix = "tgs"
	} else {
		fileSuffix = "webp"
	}

	var stickerFileNameWithDot string // `CAACAgUA.` or `duck_2_video 5 CAACAgUA.`
	var stickerSetNamePrivate  string // `-custom` or `duck_2_video`

	// 检查一下贴纸是否有 packName，没有的话就是自定义贴纸
	if opts.Message.Sticker.SetName != "" {
		// 获取贴纸包信息
		stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: opts.Message.Sticker.SetName })
		if err != nil {
			// this sticker has a setname, but that sticker set has been deleted
			// it may also be because of an error in obtaining sticker there are no static stickers in the sticker set information, so just let the user try again.
			logger.Warn().
				Err(err).
				Str("setName", opts.Message.Sticker.SetName).
				Msg("Failed to get sticker set info, download it as a custom sticker")

			// 到这里是因为用户发送的贴纸对应的贴纸包已经被删除了，但贴纸中的信息还有对应的 SetName，会触发查询，但因为贴纸包被删了就查不到，将 index 值设为 -1，缓存后当作自定义贴纸继续
			data.IsCustomSticker   = true
			stickerSetNamePrivate  = opts.Message.Sticker.SetName
			stickerFileNameWithDot = fmt.Sprintf("%s %d %s.", opts.Message.Sticker.SetName, -1, opts.Message.Sticker.FileID)
		} else {
			// sticker is in a sticker set
			data.StickerCount    = len(stickerSet.Stickers)
			data.StickerSetName  = stickerSet.Name
			data.StickerSetTitle = stickerSet.Title

			// 寻找贴纸在贴纸包中的索引并赋值
			for i, n := range stickerSet.Stickers {
				if n.FileID == opts.Message.Sticker.FileID {
					data.StickerIndex = i
					break
				}
			}

			// 在这个条件下，贴纸包名和贴纸索引都存在，赋值完整的贴纸文件名
			stickerSetNamePrivate  = opts.Message.Sticker.SetName
			stickerFileNameWithDot = fmt.Sprintf("%s %d %s.", opts.Message.Sticker.SetName, data.StickerIndex, opts.Message.Sticker.FileID)
		}
	} else {
		// this sticker doesn't have a setname, so it is a custom sticker
		// 自定义贴纸，防止与普通贴纸包冲突，将贴纸包名设置为 `-custom`，文件名仅有 FileID 用于辨识
		data.IsCustomSticker   = true
		stickerSetNamePrivate  = "-custom"
		stickerFileNameWithDot = fmt.Sprintf("%s.", opts.Message.Sticker.FileID)
	}

	var stickerFileDir string = filepath.Join(StickerCache_path, stickerSetNamePrivate)      // 保存贴纸源文件的目录 .cache/sticker/setName/
	var originFullPath string = filepath.Join(stickerFileDir, stickerFileNameWithDot + fileSuffix) // 到贴纸文件的完整目录 .cache/sticker/setName/stickerFileName.webp
	var finalFullPath  string // 存放最后读取并发送的文件完整目录 .cache/sticker/setName/stickerFileName.webp

	_, err := os.Stat(originFullPath) // 检查贴纸源文件是否已缓存
	if err != nil {
		// 如果文件不存在，进行下载，否则返回错误
		if os.IsNotExist(err) {
			// 日志提示该文件没被缓存，正在下载
			logger.Debug().
				Str("originFullPath", originFullPath).
				Msg("Sticker file not cached, downloading")

			// 从服务器获取文件信息
			fileinfo, err := opts.Thebot.GetFile(opts.Ctx, &bot.GetFileParams{ FileID: opts.Message.Sticker.FileID })
			if err != nil {
				logger.Error().
					Err(err).
					Str("fileID", opts.Message.Sticker.FileID).
					Str("content", "sticker file info").
					Msg(flaterr.GetFile.Str())
				return nil, fmt.Errorf(flaterr.GetFile.Fmt(), opts.Message.Sticker.FileID, err)
			}

			// 组合链接下载贴纸源文件
			resp, err := http.Get(opts.Thebot.FileDownloadLink(fileinfo))
			if err != nil {
				logger.Error().
					Err(err).
					Str("filePath", fileinfo.FilePath).
					Msg("Failed to download sticker file")
				return nil, fmt.Errorf("failed to download sticker file [%s]: %w", fileinfo.FilePath, err)
			}
			defer resp.Body.Close()

			// 创建保存贴纸的目录
			err = os.MkdirAll(stickerFileDir, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("directory", stickerFileDir).
					Msg("Failed to create sticker directory to save sticker")
				return nil, fmt.Errorf("failed to create directory [%s] to save sticker: %w", stickerFileDir, err)
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
		logger.Debug().
			Str("originFullPath", originFullPath).
			Msg("Sticker file already cached")
	}

	if opts.Message.Sticker.IsAnimated {
		// tgs
		// 不需要转码，直接读取原贴纸文件
		finalFullPath = originFullPath
	} else if opts.Message.Sticker.IsVideo {
		if configs.BotConfig.FFmpegPath != "" {
			// webm, convert to GIF
			var GIFFilePath   string = filepath.Join(StickerCacheGIF_path, stickerSetNamePrivate) // 转码后为 png 格式的目录 .cache/sticker_png/setName/
			var toGIFFullPath string = filepath.Join(GIFFilePath, stickerFileNameWithDot + "GIF") // 转码后到 png 格式贴纸的完整目录 .cache/sticker_png/setName/stickerFileName.png

			_, err = os.Stat(toGIFFullPath) // 使用目录提前检查一下是否已经转换过
			if err != nil {
				// 如果提示不存在，进行转换
				if os.IsNotExist(err) {
					// 日志提示该文件没转换，正在转换
					logger.Debug().
						Str("toGIFFullPath", toGIFFullPath).
						Msg("Sticker file does not convert, converting")

					// 创建保存贴纸的目录
					err = os.MkdirAll(GIFFilePath, 0755)
					if err != nil {
						logger.Error().
							Err(err).
							Str("GIFFilePath", GIFFilePath).
							Msg("Failed to create directory to convert file")
						return nil, fmt.Errorf("failed to create directory [%s] to convert sticker file: %w", GIFFilePath, err)
					}

					// 读取原贴纸文件，转码后存储到 GIF 格式贴纸的完整目录
					err = convertWebMToGif(originFullPath, toGIFFullPath)
					if err != nil {
						logger.Error().
							Err(err).
							Str("originFullPath", originFullPath).
							Msg("Failed to convert WebM to GIF")
						return nil, fmt.Errorf("failed to convert webm [%s] to GIF: %w", originFullPath, err)
					}
				} else {
					// 其他错误
					logger.Error().
						Err(err).
						Str("toGIFFullPath", toGIFFullPath).
						Msg("Failed to read converted GIF file info")
					return nil, fmt.Errorf("failed to read converted GIF sticker file [%s] info: %w", toGIFFullPath, err)
				}
			} else {
				// 文件存在，跳过转换
				logger.Debug().
					Str("toGIFFullPath", toGIFFullPath).
					Msg("Sticker file already converted to GIF")
			}

			// 处理完成，将最后要读取的目录设为转码后 GIF 格式贴纸的完整目录
			data.IsConverted = true
			finalFullPath = toGIFFullPath
		} else {
			// 没有 ffmpeg 能用来转码，直接读取原贴纸文件
			finalFullPath = originFullPath
		}
	} else {
		// webp, need convert to png
		var PNGFilePath   string = filepath.Join(StickerCachePNG_path, stickerSetNamePrivate) // 转码后为 png 格式的目录 .cache/sticker_png/setName/
		var toPNGFullPath string = filepath.Join(PNGFilePath, stickerFileNameWithDot + "png") // 转码后到 png 格式贴纸的完整目录 .cache/sticker_png/setName/stickerFileName.png

		_, err = os.Stat(toPNGFullPath) // 使用目录提前检查一下是否已经转换过
		if err != nil {
			// 如果提示不存在，进行转换
			if os.IsNotExist(err) {
				// 日志提示该文件没转换，正在转换
				logger.Debug().
					Str("toPNGFullPath", toPNGFullPath).
					Msg("Sticker file does not convert, converting")

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
						Msg("Failed to convert WebP to PNG")
					return nil, fmt.Errorf("failed to convert WebP [%s] to PNG: %w", originFullPath, err)
				}
			} else {
				// 其他错误
				logger.Error().
					Err(err).
					Str("toPNGFullPath", toPNGFullPath).
					Msg("Failed to read converted PNG sticker file info")
				return nil, fmt.Errorf("failed to read converted PNG sticker file [%s] info : %w", toPNGFullPath, err)
			}
		} else {
			// 文件存在，跳过转换
			logger.Debug().
				Str("toPNGFullPath", toPNGFullPath).
				Msg("Sticker file already converted to PNG")
		}

		// 处理完成，将最后要读取的目录设为转码后 PNG 格式贴纸的完整目录
		data.IsConverted = true
		finalFullPath = toPNGFullPath
	}

	// 逻辑完成，读取最后的文件，返回给上一级函数
	data.Data, err = os.Open(finalFullPath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("finalFullPath", finalFullPath).
			Msg("Failed to open downloaded sticker file")
		return nil, fmt.Errorf("failed to open downloaded sticker file [%s]: %w", finalFullPath, err)
	}

	return &data, nil
}

func DownloadStickerPackCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var handlerErr flaterr.MultErr

	botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
		Text:   "已请求下载，请稍候",
		ParseMode: models.ParseModeMarkdownV1,
		DisableNotification: true,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
			Str("content", "start download stickerset notice").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "start download stickerset notice", err)
	}

	err = database.IncrementalUsageCount(opts.Ctx, opts.CallbackQuery.Message.Message.Chat.ID, db_struct.StickerSetDownloaded)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
			Msg("Failed to incremental sticker set download count")
		handlerErr.Addf("failed to incremental sticker set download count: %w", err)
	}

	var setName  string
	var isOnlyPNG bool
	if opts.CallbackQuery.Data[0:2] == "S_" {
		setName = strings.TrimPrefix(opts.CallbackQuery.Data, "S_")
		isOnlyPNG = true
	} else {
		setName = strings.TrimPrefix(opts.CallbackQuery.Data, "s_")
		isOnlyPNG = false
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
			ChatID: opts.CallbackQuery.From.ID,
			Text:   fmt.Sprintf("获取贴纸包时发生了一些错误\n<blockquote expandable>Failed to get sticker set info: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
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

		stickerData, err := getStickerPack(opts.Ctx, opts.Thebot, stickerSet, isOnlyPNG)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Msg("Failed to download sticker set")
			handlerErr.Addf("failed to download sticker set: %w", err)

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.CallbackQuery.From.ID,
				Text:   fmt.Sprintf("下载贴纸包时发生了一些错误\n<blockquote expandable>Failed to download sticker set: %s</blockquote>", err),
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
				ChatID: opts.CallbackQuery.From.ID,
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
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("content", "sticker set zip file").
					Msg(flaterr.SendDocument.Str())
				handlerErr.Addt(flaterr.SendDocument, "sticker set zip file", err)
			}

			_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
				ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: botMessage.ID,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("content", "start download stickerset notice").
					Msg(flaterr.DeleteMessage.Str())
				handlerErr.Addt(flaterr.DeleteMessage, "start download sticker set notice", err)
			}
		}
	}

	return handlerErr.Flat()
}

func getStickerPack(ctx context.Context, thebot *bot.Bot, stickerSet *models.StickerSet, isOnlyPNG bool) (*stickerDatas, error) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
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
		Msg("Start download sticker set")

	filePath    := filepath.Join(StickerCache_path, stickerSet.Name)
	PNGFilePath := filepath.Join(StickerCachePNG_path, stickerSet.Name)

	var allCached    bool = true
	var allConverted bool = true

	for i, sticker := range stickerSet.Stickers {
		stickerfileName := fmt.Sprintf("%s %d %s.", sticker.SetName, i, sticker.FileID)
		var fileSuffix string

		// 根据贴纸类型设置文件扩展名和统计贴纸数量
		if sticker.IsVideo {
			fileSuffix = "webm"
			data.WebM++
		} else if sticker.IsAnimated {
			fileSuffix = "tgs"
			data.tgs++
		} else {
			fileSuffix = "webp"
			data.WebP++
		}

		var originFullPath string = filepath.Join(filePath, stickerfileName + fileSuffix)

		_, err := os.Stat(originFullPath) // 检查单个贴纸是否已缓存
		if err != nil {
			if os.IsNotExist(err) {
				allCached = false
				logger.Trace().
					Str("originFullPath", originFullPath).
					Str("stickerSetName", data.StickerSetName).
					Int("stickerIndex", i).
					Msg("Sticker file not cached, downloading")

				// 从服务器获取文件内容
				fileinfo, err := thebot.GetFile(ctx, &bot.GetFileParams{ FileID: sticker.FileID })
				if err != nil {
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("fileID", sticker.FileID).
						Str("content", "sticker file info").
						Msg(flaterr.GetFile.Str())
					return nil, fmt.Errorf(flaterr.GetFile.Fmt(), sticker.FileID, err)
				}

				// 下载贴纸文件
				resp, err := http.Get(thebot.FileDownloadLink(fileinfo))
				if err != nil {
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("filePath", fileinfo.FilePath).
						Msg("Failed to download sticker file")
					return nil, fmt.Errorf("failed to download sticker file %s: %w", fileinfo.FilePath, err)
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
				Msg("Sticker file already exists")
		}

		// 仅需要 PNG 格式时进行转换
		if isOnlyPNG && !sticker.IsVideo && !sticker.IsAnimated {
			var toPNGFullPath string = filepath.Join(PNGFilePath, stickerfileName + "png")

			_, err = os.Stat(toPNGFullPath)
			if err != nil {
				if os.IsNotExist(err) {
					allConverted = false
					logger.Trace().
						Int("stickerIndex", i).
						Str("toPNGFullPath", toPNGFullPath).
						Msg("File does not convert, converting")
					// 创建保存贴纸的目录
					err = os.MkdirAll(PNGFilePath, 0755)
					if err != nil {
						logger.Error().
							Err(err).
							Int("stickerIndex", i).
							Str("PNGFilePath", PNGFilePath).
							Msg("Failed to create directory to convert file")
						return nil, fmt.Errorf("failed to create directory [%s] to convert file: %w", PNGFilePath, err)
					}
					// 将 webp 转换为 PNG
					err = convertWebPToPNG(originFullPath, toPNGFullPath)
					if err != nil {
						logger.Error().
							Err(err).
							Int("stickerIndex", i).
							Str("originFullPath", originFullPath).
							Msg("Failed to convert WebP to PNG")
						return nil, fmt.Errorf("failed to converting WebP to PNG [%s]: %w", originFullPath, err)
					}
				} else {
					// 其他错误
					logger.Error().
						Err(err).
						Int("stickerIndex", i).
						Str("toPNGFullPath", toPNGFullPath).
						Msg("Failed to read converted PNG file info")
					return nil, fmt.Errorf("failed to read converted PNG file info: %w", err)
				}
			} else {
				logger.Trace().
					Str("toPNGFullPath", toPNGFullPath).
					Msg("File already converted")
			}
		}
	}

	var zipFileName string
	var compressFolderPath string

	var isZiped bool = true

	// 根据要下载的类型设置压缩包的文件名和路径以及压缩包中的贴纸数量
	if isOnlyPNG {
		if data.WebP == 0 {
			logger.Warn().
				Dict("stickerSet", zerolog.Dict().
					Str("stickerSetName", stickerSet.Name).
					Int("WebP", data.WebP).
					Int("tgs", data.tgs).
					Int("WebM", data.WebM),
				).
				Msg("There are no static stickers in the sticker set")
			return nil, fmt.Errorf("there are no static stickers in the sticker set [%s]", stickerSet.Name)
		}
		data.StickerCount = data.WebP
		zipFileName = fmt.Sprintf("%s(%d)_png.zip", stickerSet.Name, data.StickerCount)
		compressFolderPath = PNGFilePath
	} else {
		data.StickerCount = data.WebP + data.WebM + data.tgs
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
			logger.Debug().
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
		logger.Debug().
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
			Msg("Sticker set already zipped")
	} else if isOnlyPNG && allConverted {
		// 仅需要 PNG 格式，且贴纸包完全转换成 PNG 格式，但尚未压缩
		logger.Info().
			Str("zipFileFullPath", zipFileFullPath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("Sticker set already converted")
	} else if allCached {
		// 贴纸包中的贴纸已经全部缓存了
		logger.Info().
			Str("zipFileFullPath", zipFileFullPath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("Sticker set already cached")
	} else {
		// 新下载的贴纸包（如果有部分已经下载了也是这个）
		logger.Info().
			Str("zipFileFullPath", zipFileFullPath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("Sticker set already downloaded")
	}

	return &data, nil
}

func convertWebPToPNG(webpPath, pngPath string) error {
	// 打开 WebP 文件
	webpFile, err := os.Open(webpPath)
	if err != nil {
		return fmt.Errorf("打开 WebP 文件失败: %w", err)
	}
	defer webpFile.Close()

	// 解码 WebP 图片
	img, err := webp.Decode(webpFile)
	if err != nil {
		return fmt.Errorf("解码 WebP 失败: %w", err)
	}

	// 创建 PNG 文件
	pngFile, err := os.Create(pngPath)
	if err != nil {
		return fmt.Errorf("创建 PNG 文件失败: %w", err)
	}
	defer pngFile.Close()

	// 编码 PNG
	err = png.Encode(pngFile, img)
	if err != nil {
		return fmt.Errorf("编码 PNG 失败: %w", err)
	}

	return nil
}

// use ffmpeg
func convertWebMToGif(webmPath, GIFPath string) error {
	return exec.Command(configs.BotConfig.FFmpegPath, "-i", webmPath, GIFPath).Run()
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

func showCachedStickers(opts *handler_params.Message) error {
	var button     [][]models.InlineKeyboardButton
	var tempButton []models.InlineKeyboardButton

	entries, err := os.ReadDir(StickerCache_path)
	if err != nil { return err }

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "-custom" {
			if len(tempButton) == 4 {
				button = append(button, tempButton)
				tempButton = []models.InlineKeyboardButton{}
			}
			tempButton = append(tempButton, models.InlineKeyboardButton{
				Text: entry.Name(),
				URL:  "https://t.me/addstickers/" + entry.Name(),
			})
		}
	}

	if len(tempButton) > 0 { button = append(button, tempButton) }

	_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Message.Chat.ID,
		Text:   "请选择要查看的贴纸包",
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: button,
		},
		// MessageEffectID: "5104841245755180586",
	})
	return err
}

func collectStickerSet(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Str("callbackQueryData", opts.CallbackQuery.Data).
		Logger()

	var handlerErr flaterr.MultErr

	if stickerCollect.ChannelID == 0 {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "未设置贴纸包收集频道",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "collect channel ID not set").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "collect channel ID not set", err)
		}
	} else {
		stickerSetName := strings.TrimPrefix(opts.CallbackQuery.Data, "c_")

		stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: stickerSetName })
		if err != nil {
			logger.Error().
				Err(err).
				Str("stickerSetName", stickerSetName).
				Msg(flaterr.GetStickerSet.Str())
			handlerErr.Addt(flaterr.GetStickerSet, stickerSetName, err)

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.CallbackQuery.From.ID,
				Text:   fmt.Sprintf("获取贴纸包时发生了一些错误\n<blockquote expandable>Failed to get sticker set info: %s</blockquote>", err),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "get sticker set info error").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "get sticker set info error", err)
			}
		} else {
			_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: opts.CallbackQuery.ID,
				Text:             "已开始下载贴纸包",
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "start downloading sticker pack notice").
					Msg(flaterr.AnswerCallbackQuery.Str())
				handlerErr.Addt(flaterr.AnswerCallbackQuery, "start downloading sticker pack notice", err)
			}
			stickerData, err := getStickerPack(opts.Ctx, opts.Thebot, stickerSet, false)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to download sticker set")
				handlerErr.Addf("failed to download sticker set: %w", err)

				_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.CallbackQuery.From.ID,
					Text:   fmt.Sprintf("下载贴纸包时发生了一些错误\n<blockquote expandable>Failed to download sticker set: %s</blockquote>", err),
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
				var reply *models.ReplyParameters
				collected := stickerCollect.GetStickerSetByName(stickerSetName)
				if collected != nil {
					// 已经收藏过了，更新贴纸包信息
					reply = &models.ReplyParameters{
						MessageID: collected.MsgID,
						AllowSendingWithoutReply: true, // 如果旧消息被删除了也允许发送
					}
				}
				var pendingMessage string
				if stickerData.WebP > 0 { pendingMessage += fmt.Sprintf(" %d(静态)", stickerData.WebP) }
				if stickerData.WebM > 0 { pendingMessage += fmt.Sprintf(" %d(动态)", stickerData.WebM) }
				if stickerData.tgs  > 0 { pendingMessage += fmt.Sprintf(" %d(矢量)", stickerData.tgs) }
				channelMessage, err := opts.Thebot.SendDocument(opts.Ctx, &bot.SendDocumentParams{
					ChatID:          stickerCollect.ChannelID,
					ParseMode:       models.ParseModeMarkdownV1,
					ReplyParameters: reply,
					Caption:         fmt.Sprintf("[%s](https://t.me/addstickers/%s)\n共 %d 个贴纸:%s\n存档时间 %s", stickerData.StickerSetTitle, stickerData.StickerSetName, stickerData.StickerCount, pendingMessage, time.Now().Format(time.RFC3339)),
					Document:        &models.InputFileUpload{ Filename: fmt.Sprintf("%s(%d).zip", stickerData.StickerSetName, stickerData.StickerCount), Data: stickerData.Data },
					ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "查看贴纸包", URL: "https://t.me/addstickers/" + stickerData.StickerSetName },
					}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("channelID", stickerCollect.ChannelID).
						Str("stickerSetName", stickerSetName).
						Str("content", "collect sticker set file").
						Msg(flaterr.SendDocument.Str())
					handlerErr.Addt(flaterr.SendDocument, "collect sticker set", err)
					_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:    opts.CallbackQuery.From.ID,
						Text:      fmt.Sprintf("将贴纸包发送到收藏频道失败: <blockquote expandable>%s</blockquote>", err.Error()),
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("channelID", stickerCollect.ChannelID).
							Str("stickerSetName", stickerSetName).
							Str("content", "collect sticker set failed notice").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "collect sticker set failed notice", err)
					}
				} else {
					if collected == nil {
						stickerCollect.StickerSet = append(stickerCollect.StickerSet, stickerSetInfo{
							Title: stickerSet.Title,
							Name:  stickerSet.Name,
							MsgID: channelMessage.ID,
							Count: stickerData.StickerCount,
						})
					} else {
						// 更新已收藏的贴纸包信息
						collected.Title = stickerSet.Title
						collected.MsgID = channelMessage.ID
					}

					err = saveCollectStickerList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Dict("collectSticker", zerolog.Dict().
								Str("title", collected.Title).
								Str("name", collected.Name),
							).
							Msg("Failed to save collect sticker list")
						handlerErr.Addf("failed to save collect sticker list: %w", err)
					}

					_, err = opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
						ChatID:    opts.CallbackQuery.From.ID,
						MessageID: opts.CallbackQuery.Message.Message.ID,
						ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
							Text: "✅ 已收藏至频道", URL: utils.MsgLinkPrivate(channelMessage.Chat.ID, channelMessage.ID),
						}}}},
					})
					if err != nil {
						logger.Err(err).
							Int64("channelID", stickerCollect.ChannelID).
							Str("stickerSetName", stickerSetName).
							Str("content", "collect sticker set success notice").
							Msg(flaterr.EditMessageReplyMarkup.Str())
						handlerErr.Addt(flaterr.EditMessageReplyMarkup, "collect sticker set success notice", err)
					}
				}
			}
		}

	}

	return handlerErr.Flat()
}

// full command "t.me/addstickers/" or "https://t.me/addstickers/"
func getStickerPackInfo(opts *handler_params.Message) error {
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
		stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: stickerSetName })
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
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
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

			button = append(button, [][]models.InlineKeyboardButton{
				{{ Text: "下载贴纸包中的静态贴纸", CallbackData: fmt.Sprintf("S_%s", stickerSet.Name) }},
				{{ Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", stickerSet.Name) }},
			}...)

			if stickerCollect.ChannelID != 0 && contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...) {
				for _, set := range stickerCollect.StickerSet {
					if set.Name == stickerSet.Name {
						button = append(button, [][]models.InlineKeyboardButton{{
							{
								Text: fmt.Sprintf("🔁 更新? (%d)>(%d)", set.Count, len(stickerSet.Stickers)),
								CallbackData: fmt.Sprintf("c_%s", stickerSet.Name),
							},
							{
								Text: "✅ 已收藏至频道",
								URL: utils.MsgLinkPrivate(stickerCollect.ChannelID, set.MsgID),
							},
						}}...)
						break
					}
				}
				if len(button) == 2 {
					button = append(button, []models.InlineKeyboardButton{{
						Text: "⭐️ 收藏至频道",
						CallbackData: fmt.Sprintf("c_%s", stickerSet.Name),
					}})
				}
			}
			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:              opts.Message.From.ID,
				Text:                fmt.Sprintf("<a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包中一共有 %d 个贴纸\n", stickerSet.Name, stickerSet.Title, len(stickerSet.Stickers)),
				ParseMode:           models.ParseModeHTML,
				DisableNotification: true,
				ReplyParameters:     &models.ReplyParameters{ MessageID: opts.Message.ID },
				ReplyMarkup:         &models.InlineKeyboardMarkup{ InlineKeyboard: button},
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
			ChatID: opts.Message.From.ID,
			Text:   "请发送一个有效的贴纸链接",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
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

func convertMP4ToGifHandler(opts *handler_params.Message) error {
	if opts.Message == nil || opts.Message.Animation == nil {
		return nil
	}

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	logger.Info().
		Msg("Start download GIF")

	GIFFile, err := downloadGifHandler(opts)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Error when downloading MP4")
		handlerErr.Addf("error when downloading MP4: %w", err)

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Message.From.ID,
			Text:      fmt.Sprintf("下载 GIF 时发生了一些错误\n<blockquote expandable>Failed to download MP4: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "MP4 download error").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "MP4 download error", err)
		}
	} else {
		_, err = opts.Thebot.SendDocument(opts.Ctx, &bot.SendDocumentParams{
			ChatID:                      opts.Message.From.ID,
			Caption:                     fmt.Sprintf("视频长度 %d 秒\n分辨率 %d*%dpx", opts.Message.Animation.Duration, opts.Message.Animation.Width, opts.Message.Animation.Height),
			Document:                    &models.InputFileUpload{ Filename: strings.Split(opts.Message.Animation.FileName, ".")[0] + ".GIF", Data: GIFFile },
			ParseMode:                   models.ParseModeHTML,
			ReplyParameters:             &models.ReplyParameters{ MessageID: opts.Message.ID },
			DisableNotification:         true,
			DisableContentTypeDetection: true, // Prevent the server convert GIF to MP4
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "GIF file").
				Msg(flaterr.SendDocument.Str())
			handlerErr.Addt(flaterr.SendDocument, "GIF file", err)
		}
	}

	return handlerErr.Flat()
}

func downloadGifHandler(opts *handler_params.Message) (io.Reader, error) {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var originFullPath string = filepath.Join(MP4Cache_path, opts.Message.Animation.FileID + ".MP4")
	var toGifFullPath  string = filepath.Join(GIFCache_path, opts.Message.Animation.FileID + ".GIF")

	_, err := os.Stat(originFullPath) // 检查是否已缓存
	if err != nil {
		// 如果文件不存在，进行下载，否则返回错误
		if os.IsNotExist(err) {
			// 日志提示该文件没被缓存，正在下载
			logger.Debug().
				Str("originFullPath", originFullPath).
				Msg("GIF file not cached, downloading")

			// 从服务器获取文件信息
			fileinfo, err := opts.Thebot.GetFile(opts.Ctx, &bot.GetFileParams{ FileID: opts.Message.Animation.FileID })
			if err != nil {
				logger.Error().
					Err(err).
					Str("fileID", opts.Message.Animation.FileID).
					Msg(flaterr.GetFile.Str())
				return nil, fmt.Errorf(flaterr.GetFile.Fmt(), opts.Message.Animation.FileID, err)
			}

			// 组合链接下载贴纸源文件
			resp, err := http.Get(opts.Thebot.FileDownloadLink(fileinfo))
			if err != nil {
				logger.Error().
					Err(err).
					Str("filePath", fileinfo.FilePath).
					Msg("Failed to download GIF file")
				return nil, fmt.Errorf("failed to download GIF file [%s]: %w", fileinfo.FilePath, err)
			}
			defer resp.Body.Close()

			// 创建保存贴纸的目录
			err = os.MkdirAll(MP4Cache_path, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("directory", MP4Cache_path).
					Msg("Failed to create GIF directory to save GIF")
				return nil, fmt.Errorf("failed to create directory [%s] to save GIF: %w", MP4Cache_path, err)
			}

			// 创建贴纸空文件
			downloadedGif, err := os.Create(originFullPath)
			if err != nil {
				logger.Error().
					Err(err).
					Str("originFullPath", originFullPath).
					Msg("Failed to create GIF file")
				return nil, fmt.Errorf("failed to create GIF file [%s]: %w", originFullPath, err)
			}
			defer downloadedGif.Close()

			// 将下载的原贴纸写入空文件
			_, err = io.Copy(downloadedGif, resp.Body)
			if err != nil {
				logger.Error().
					Err(err).
					Str("originFullPath", originFullPath).
					Msg("Failed to writing GIF data to file")
				return nil, fmt.Errorf("failed to writing GIF data to file [%s]: %w", originFullPath, err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("originFullPath", originFullPath).
				Msg("Failed to read cached GIF file info")
			return nil, fmt.Errorf("failed to read cached GIF file [%s] info: %w", originFullPath, err)
		}
	} else {
		// 文件已存在，跳过下载
		logger.Debug().
			Str("originFullPath", originFullPath).
			Msg("MP4 file already cached")
	}

	_, err = os.Stat(toGifFullPath) // 使用目录提前检查一下是否已经转换过
	if err != nil {
		// 如果提示不存在，进行转换
		if os.IsNotExist(err) {
			// 日志提示该文件没转换，正在转换
			logger.Debug().
				Str("toGifFullPath", toGifFullPath).
				Msg("MP4 file does not convert, converting")

			// 创建保存贴纸的目录
			err = os.MkdirAll(GIFCache_path, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("GIFCache_path", GIFCache_path).
					Msg("Failed to create directory to convert GIF")
				return nil, fmt.Errorf("failed to create directory [%s] to convert GIF: %w", GIFCache_path, err)
			}

			// 读取原贴纸文件，转码后存储到 png 格式贴纸的完整目录
			err = convertMP4ToGif(originFullPath, toGifFullPath)
			if err != nil {
				logger.Error().
					Err(err).
					Str("originFullPath", originFullPath).
					Msg("Failed to convert MP4 to GIF")
				return nil, fmt.Errorf("failed to convert MP4 [%s] to GIF: %w", originFullPath, err)
			}
		} else {
			// 其他错误
			logger.Error().
				Err(err).
				Str("toGifFullPath", toGifFullPath).
				Msg("Failed to read converted GIF file info")
			return nil, fmt.Errorf("failed to read converted GIF file [%s] info : %w", toGifFullPath, err)
		}
	} else {
		// 文件存在，跳过转换
		logger.Debug().
			Str("toGifFullPath", toGifFullPath).
			Msg("MP4 file already converted to GIF")
	}


	data, err := os.Open(toGifFullPath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("toGifFullPath", toGifFullPath).
			Msg("Failed to open converted GIF file")
		return nil, fmt.Errorf("failed to open converted GIF file [%s]: %w", toGifFullPath, err)
	}

	return data, nil
}

func convertMP4ToGif(MP4Path, GIFPath string) error {
	return exec.Command(configs.BotConfig.FFmpegPath, "-i", MP4Path, GIFPath).Run()
}

// func convertMP4ToWebM(MP4Path, webmPath string) error {
// 	return exec.Command(configs.BotConfig.FFmpegPath,
// 		"-i", MP4Path,
// 		"-vf", "scale='if(gt(iw,ih),512,-2)':'if(gt(ih,iw),512,-2)',fps=30",
// 		"-c:v", "libvpx-vp9",
// 		"-crf", "40",
// 		"-b:v", "0",
// 		"-an",
// 		"-limit_filesize", "262144",
// 		webmPath,
// 	).Run()
// }

func readCollectStickerList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "sticker").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(stickerCollectPath, &stickerCollect)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", stickerCollectPath).
				Msg("Not found collect sticker list file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(stickerCollectPath, &stickerCollect)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", stickerCollectPath).
					Msg("Failed to create empty collect sticker list file")
				return fmt.Errorf("failed to create empty collect sticker list file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", stickerCollectPath).
				Msg("Failed to load collect sticker list file")
			return fmt.Errorf("failed to load collect sticker list file: %w", err)
		}
	}

	return nil
}

func saveCollectStickerList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "sticker").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.SaveYAML(stickerCollectPath, &stickerCollect)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", stickerCollectPath).
			Msg("Failed to save collect sticker list")
		return fmt.Errorf("failed to save collect sticker list: %w", err)
	}
	return nil
}
