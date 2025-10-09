package download

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"trbot/plugins/sticker_download/common"
	"trbot/plugins/sticker_download/config"
	"trbot/plugins/sticker_download/convert"
	"trbot/plugins/sticker_download/lock"
	"trbot/utils"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog"
)

// 下载单个贴纸
func GetSticker(opts *handler_params.Message) (*common.StickerDatas, error) {
	// if !lock.Download.TryLock() {}
	lock.Download.Lock()
	defer lock.Download.Unlock()

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var data common.StickerDatas

	// 根据贴纸类型设置文件扩展名
	var dotFileSuffix           string // `.webp`, `.webm`, `.tgs`
	var dotFileSuffix_converted string // `.gif`,  `.gif`,  `.png`

	switch {
	case opts.Message.Sticker.IsVideo:
		dotFileSuffix = ".webm"
		dotFileSuffix_converted = ".gif"
	case opts.Message.Sticker.IsAnimated:
		dotFileSuffix = ".tgs"
		dotFileSuffix_converted = ".gif"
	default:
		dotFileSuffix = ".webp"
		dotFileSuffix_converted = ".png"
	}

	// 检查一下贴纸是否有 packName，没有的话就是自定义贴纸
	if opts.Message.Sticker.SetName != "" {
		// 仅在允许下载贴纸包时获取贴纸包信息
		if config.Config.AllowDownloadStickerSet {
			stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: opts.Message.Sticker.SetName })
			if err != nil {
				// this sticker has a setname, but that sticker set has been deleted
				logger.Warn().
					Err(err).
					Str("setName", opts.Message.Sticker.SetName).
					Msg("Failed to get sticker set info, download it as a custom sticker")

				// 到这里是因为用户发送的贴纸对应的贴纸包已经被删除了，但贴纸中的信息还有对应的 SetName，会触发查询，但因为贴纸包被删了就查不到，将 index 值设为 -1，缓存后当作自定义贴纸继续
				data.IsCustomSticker = true
				data.StickerSetName  = "-custom"
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
			}
		} else {
			data.StickerSetName = opts.Message.Sticker.SetName
		}
	} else {
		// this sticker doesn't have a setname, so it is a custom sticker
		// 自定义贴纸，防止与普通贴纸包冲突，将贴纸包名设置为 `-custom`
		data.IsCustomSticker = true
		data.StickerSetName  = "-custom"
	}

	var originStickerDir  string = filepath.Join(config.CachedDir, data.StickerSetName)
	var originStickerPath string = filepath.Join(originStickerDir, opts.Message.Sticker.FileID + dotFileSuffix)

	_, err := os.Stat(originStickerPath) // 检查贴纸源文件是否已缓存
	if err != nil {
		// 如果文件不存在，进行下载，否则返回错误
		if os.IsNotExist(err) {
			// 日志提示该文件没被缓存，正在下载
			logger.Debug().
				Str("path", originStickerPath).
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
			err = os.MkdirAll(originStickerDir, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("directory", originStickerDir).
					Msg("Failed to create sticker directory to save sticker")
				return nil, fmt.Errorf("failed to create directory [%s] to save sticker: %w", originStickerDir, err)
			}

			// 创建贴纸空文件
			downloadedSticker, err := os.Create(originStickerPath)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", originStickerPath).
					Msg("Failed to create empty sticker file")
				return nil, fmt.Errorf("failed to create empty sticker file [%s]: %w", originStickerPath, err)
			}
			defer downloadedSticker.Close()

			// 将下载的原贴纸写入空文件
			_, err = io.Copy(downloadedSticker, resp.Body)
			if err != nil {
				logger.Error().
					Err(err).
					Str("fullPath", originStickerPath).
					Msg("Failed to writing sticker data to file")
				return nil, fmt.Errorf("failed to writing sticker data to file [%s]: %w", originStickerPath, err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("fullPath", originStickerPath).
				Msg("Failed to read cached sticker file info")
			return nil, fmt.Errorf("failed to read cached sticker file [%s] info: %w", originStickerPath, err)
		}
	} else {
		// 文件已存在，跳过下载
		logger.Debug().
			Str("fullPath", originStickerPath).
			Msg("Sticker file already cached")
	}

	if !config.Config.DisableConvert {
		convertedStickerDir  := filepath.Join(config.ConvertedDir, data.StickerSetName)
		convertedStickerPath := filepath.Join(convertedStickerDir, opts.Message.Sticker.FileID + dotFileSuffix_converted)

		_, err = os.Stat(convertedStickerPath)
		if err != nil {
			if os.IsNotExist(err) {
				// 日志提示该文件没转换，正在转换
				logger.Debug().
					Str("path", convertedStickerPath).
					Msg("Sticker file does not convert, converting")

				// 创建保存贴纸的目录
				err = os.MkdirAll(convertedStickerDir, 0755)
				if err != nil {
					logger.Error().
						Err(err).
						Str("path", convertedStickerDir).
						Msg("Failed to create directory to convert sticker")
					return nil, fmt.Errorf("failed to create directory [%s] to convert sticker: %w", convertedStickerDir, err)
				}

				switch {
				case opts.Message.Sticker.IsVideo:
					if config.Config.FFmpegPath != "" {
						err = convert.WebMToGif(originStickerPath, convertedStickerPath)
						if err != nil {
							logger.Error().
								Err(err).
								Str("path", originStickerPath).
								Msg("Failed to convert WebM to GIF")
							return nil, fmt.Errorf("failed to convert WebM [%s] to GIF: %w", originStickerPath, err)
						}
						data.IsConverted = true
					}
				case opts.Message.Sticker.IsAnimated:
					if config.Config.LottieToPNGPath != "" && config.Config.GifskiPath != "" {
						err = convert.TGSToGif(opts.Message.Sticker.FileID, originStickerPath, convertedStickerPath)
						if err != nil {
							logger.Error().
								Err(err).
								Str("path", originStickerPath).
								Msg("Failed to convert TGS to GIF")
							return nil, fmt.Errorf("failed to convert TGS [%s] to GIF: %w", originStickerPath, err)
						}
						data.IsConverted = true
					}
				default:
					err = convert.WebPToPNG(originStickerPath, convertedStickerPath)
					if err != nil {
						logger.Error().
							Err(err).
							Str("path", originStickerPath).
							Msg("Failed to convert WebP to PNG")
						return nil, fmt.Errorf("failed to convert WebP [%s] to PNG: %w", originStickerPath, err)
					}
					data.IsConverted = true
				}
			} else {
				// 其他错误
				logger.Error().
					Err(err).
					Str("path", convertedStickerPath).
					Msg("Failed to read converted sticker file info")
				return nil, fmt.Errorf("failed to read converted sticker file info: %w", err)
			}
		} else {
			data.IsConverted = true
		}

		if data.IsConverted {
			data.Data, err = os.Open(convertedStickerPath)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", convertedStickerPath).
					Msg("Failed to open converted sticker file")
				return nil, fmt.Errorf("failed to open converted sticker file [%s]: %w", convertedStickerPath, err)
			}
		}
	}

	// 有时可能会因为没有填写转换工具 PATH 导致跳过转换，此时直接返回原始文件
	if !data.IsConverted {
		data.Data, err = os.Open(originStickerPath)
		if err != nil {
			logger.Error().
				Err(err).
				Str("path", originStickerPath).
				Msg("Failed to open downloaded sticker file")
			return nil, fmt.Errorf("failed to open downloaded sticker file [%s]: %w", originStickerPath, err)
		}
	}

	// 逻辑完成，读取最后的文件，返回给上一级函数
	return &data, nil
}
