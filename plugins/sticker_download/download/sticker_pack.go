package download

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"trbot/plugins/sticker_download/common"
	"trbot/plugins/sticker_download/config"
	"trbot/plugins/sticker_download/convert"
	"trbot/utils"
	"trbot/utils/flaterr"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

func GetStickerPack(ctx context.Context, thebot *bot.Bot, stickerSet *models.StickerSet, needConvert bool) (*common.StickerDatas, error) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var data = common.StickerDatas{
		IsCustomSticker: false,
		StickerCount:    len(stickerSet.Stickers),
		StickerSetName:  stickerSet.Name,
		StickerSetTitle: stickerSet.Title,
	}

	logger.Info().
		Dict("stickerSet", zerolog.Dict().
			Str("title", data.StickerSetTitle).
			Str("name", data.StickerSetName).
			Int("allCount", len(stickerSet.Stickers)),
		).
		Bool("needConvert", needConvert).
		Msg("Start download sticker set")

	originDir    := filepath.Join(config.CachedDir,    stickerSet.Name)
	convertedDir := filepath.Join(config.ConvertedDir, stickerSet.Name)

	var allDownloaded bool = true
	var allConverted  bool = true

	for i, sticker := range stickerSet.Stickers {
		var dotFileSuffix           string
		var dotFileSuffix_converted string

		// 根据贴纸类型设置文件扩展名和统计贴纸数量
		switch {
		case sticker.IsVideo:
			dotFileSuffix = ".webm"
			dotFileSuffix_converted = ".gif"
			data.WebM++
		case sticker.IsAnimated:
			dotFileSuffix = ".tgs"
			dotFileSuffix_converted = ".gif"
			data.TGS++
		default:
			dotFileSuffix = ".webp"
			dotFileSuffix_converted = ".png"
			data.WebP++
		}

		originStickerPath := filepath.Join(originDir, sticker.FileID + dotFileSuffix)

		_, err := os.Stat(originStickerPath) // 检查单个贴纸是否已缓存
		if err != nil {
			if os.IsNotExist(err) {
				allDownloaded = false
				logger.Debug().
					Str("path", originStickerPath).
					Str("setName", data.StickerSetName).
					Int("index", i).
					Msg("Sticker file not cached, downloading")

				// 从服务器获取文件内容
				fileinfo, err := thebot.GetFile(ctx, &bot.GetFileParams{FileID: sticker.FileID})
				if err != nil {
					logger.Error().
						Err(err).
						Int("index", i).
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
						Int("index", i).
						Str("filePath", fileinfo.FilePath).
						Msg("Failed to download sticker file")
					return nil, fmt.Errorf("failed to download sticker file [%s]: %w", fileinfo.FilePath, err)
				}
				defer resp.Body.Close()

				err = os.MkdirAll(originDir, 0755)
				if err != nil {
					logger.Error().
						Err(err).
						Int("index", i).
						Str("filePath", originDir).
						Msg("Failed to creat directory to save sticker")
					return nil, fmt.Errorf("failed to create directory [%s] to save sticker: %w", originDir, err)
				}

				// 创建空白贴纸文件
				stickerfile, err := os.Create(originStickerPath)
				if err != nil {
					logger.Error().
						Err(err).
						Int("index", i).
						Str("path", originStickerPath).
						Msg("Failed to create sticker file")
					return nil, fmt.Errorf("failed to create sticker file [%s]: %w", originStickerPath, err)
				}
				defer stickerfile.Close()

				// 将下载的内容写入文件
				_, err = io.Copy(stickerfile, resp.Body)
				if err != nil {
					logger.Error().
						Err(err).
						Int("index", i).
						Str("path", originStickerPath).
						Msg("Failed to writing sticker data to file")
					return nil, fmt.Errorf("failed to writing sticker data to file [%s]: %w", originStickerPath, err)
				}
			} else {
				logger.Error().
					Err(err).
					Int("index", i).
					Str("path", originStickerPath).
					Msg("Failed to read cached sticker file info")
				return nil, fmt.Errorf("failed to read cached sticker file [%s] info: %w", originStickerPath, err)
			}
		} else {
			// 存在跳过下载过程
			logger.Trace().
				Int("index", i).
				Str("path", originStickerPath).
				Msg("Sticker file already exists")
		}

		if !config.Config.DisableConvert && needConvert {
			convertedStickerPath := filepath.Join(convertedDir, sticker.FileID + dotFileSuffix_converted)

			_, err = os.Stat(convertedStickerPath)
			if err != nil {
				if os.IsNotExist(err) {
					allConverted = false
					logger.Debug().
						Int("index", i).
						Str("path", convertedStickerPath).
						Msg("File does not convert, converting")
					// 创建保存贴纸的目录
					err = os.MkdirAll(convertedDir, 0755)
					if err != nil {
						logger.Error().
							Err(err).
							Int("index", i).
							Str("directory", convertedDir).
							Msg("Failed to create directory to convert file")
						return nil, fmt.Errorf("failed to create directory [%s] to convert file: %w", convertedDir, err)
					}

					switch {
					case sticker.IsVideo:
						err = convert.WebMToGif(originStickerPath, convertedStickerPath)
						if err != nil {
							logger.Error().
								Err(err).
								Int("index", i).
								Str("path", originStickerPath).
								Msg("Failed to convert WebM to GIF")
							return nil, fmt.Errorf("failed to convert WebM [%s] to GIF: %w", originStickerPath, err)
						}
					case sticker.IsAnimated:
						err = convert.TGSToGif(sticker.FileID, originStickerPath, convertedStickerPath)
						if err != nil {
							logger.Error().
								Err(err).
								Int("index", i).
								Str("path", originStickerPath).
								Msg("Failed to convert WebP to PNG")
							return nil, fmt.Errorf("failed to convert WebP [%s] to PNG: %w", originStickerPath, err)
						}
					default:
						err = convert.WebPToPNG(originStickerPath, convertedStickerPath)
						if err != nil {
							logger.Error().
								Err(err).
								Int("index", i).
								Str("path", originStickerPath).
								Msg("Failed to convert WebP to PNG")
							return nil, fmt.Errorf("failed to convert WebP [%s] to PNG: %w", originStickerPath, err)
						}
					}
				} else {
					// 其他错误
					logger.Error().
						Err(err).
						Int("index", i).
						Str("path", convertedStickerPath).
						Msg("Failed to read converted sticker file info")
					return nil, fmt.Errorf("failed to read converted sticker file info: %w", err)
				}
			} else {
				logger.Debug().
					Str("path", convertedStickerPath).
					Msg("Sticker already converted")
			}
		}
	}

	var compressFolderPath string
	var isZiped bool = true

	// 根据要下载的类型设置压缩包的文件名和路径以及压缩包中的贴纸数量
	if needConvert {
		data.StickerSetFileName = fmt.Sprintf("%s(%d)_converted.zip", stickerSet.Name, data.StickerCount)
		compressFolderPath = convertedDir
	} else {
		data.StickerSetFileName = fmt.Sprintf("%s(%d).zip", stickerSet.Name, data.StickerCount)
		compressFolderPath = originDir
	}

	zipFileFullPath := filepath.Join(config.CompressedDir, data.StickerSetFileName)

	_, err := os.Stat(zipFileFullPath) // 检查压缩包文件是否存在
	if err != nil {
		if os.IsNotExist(err) {
			isZiped = false
			err = os.MkdirAll(config.CompressedDir, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("directory", config.CompressedDir).
					Msg("Failed to create zip file directory")
				return nil, fmt.Errorf("failed to create zip file directory [%s]: %w", config.CompressedDir, err)
			}
			err = compressStickerPack(stickerSet, compressFolderPath, zipFileFullPath, needConvert)
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

	fileinfo, err := os.Stat(zipFileFullPath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("zipFileFullPath", zipFileFullPath).
			Msg("Failed to read compressed sticker set zip file info")
		return nil, fmt.Errorf("failed to read compressed sticker set zip file [%s] info: %w", zipFileFullPath, err)
	} else {
		data.StickerSetSize = fileinfo.Size()
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
	} else if needConvert && allConverted {
		// 仅需要 PNG 格式，且贴纸包完全转换成 PNG 格式，但尚未压缩
		logger.Info().
			Str("zipFileFullPath", zipFileFullPath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("Sticker set already converted")
	} else if allDownloaded {
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
			Msg("Sticker set downloaded")
	}

	return &data, nil
}

func compressStickerPack(stickerSet *models.StickerSet, srcDir, zipFile string, converted bool) error {
	// 创建 ZIP 文件
	outFile, err := os.Create(zipFile)
	if err != nil { return err }
	defer outFile.Close()

	// 创建 ZIP 写入器
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	for i, sticker := range stickerSet.Stickers {
		var dotFileSuffix string

		if converted {
			if sticker.IsVideo || sticker.IsAnimated {
				dotFileSuffix = ".gif"
			} else {
				dotFileSuffix = ".png"
			}
		} else {
			switch {
			case sticker.IsVideo:
				dotFileSuffix = ".webm"
			case sticker.IsAnimated:
				dotFileSuffix = ".tgs"
			default:
				dotFileSuffix = ".webp"
			}
		}

		file, err := os.Open(filepath.Join(srcDir, sticker.FileID + dotFileSuffix))
		if err != nil { return err }
		defer file.Close()

		// 创建 ZIP 内的文件
		zipFileWriter, err := zipWriter.Create(fmt.Sprintf("%s_%02d%s", sticker.SetName, i, dotFileSuffix))
		if err != nil { return err }

		// 复制文件内容到 ZIP
		_, err = io.Copy(zipFileWriter, file)
		if err != nil { return err }
	}
	return nil
}
