package mpeg4gif

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"trbot/plugins/sticker_download/convert"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var MP4Cache_path string = filepath.Join(configs.CacheDir, "mp4/")
var GIFCache_path string = filepath.Join(configs.CacheDir, "mp4_GIF/")


func ConvertMP4ToGifHandler(opts *handler_params.Message) error {
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
			Text:      fmt.Sprintf("下载 GIF 时发生了一些错误\n<blockquote expandable>Failed to download MP4: %s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
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
			Document:                    &models.InputFileUpload{Filename: strings.Split(opts.Message.Animation.FileName, ".")[0] + ".GIF", Data: GIFFile},
			ParseMode:                   models.ParseModeHTML,
			ReplyParameters:             &models.ReplyParameters{MessageID: opts.Message.ID},
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

	var originFullPath string = filepath.Join(MP4Cache_path, opts.Message.Animation.FileID+".MP4")
	var toGifFullPath string = filepath.Join(GIFCache_path, opts.Message.Animation.FileID+".GIF")

	_, err := os.Stat(originFullPath) // 检查是否已缓存
	if err != nil {
		// 如果文件不存在，进行下载，否则返回错误
		if os.IsNotExist(err) {
			// 日志提示该文件没被缓存，正在下载
			logger.Debug().
				Str("originFullPath", originFullPath).
				Msg("GIF file not cached, downloading")

			// 从服务器获取文件信息
			fileinfo, err := opts.Thebot.GetFile(opts.Ctx, &bot.GetFileParams{FileID: opts.Message.Animation.FileID})
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
			err = convert.MP4ToGif(originFullPath, toGifFullPath)
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
