package download

import (
	"archive/zip"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
		StickerSetHash:  hashStickerSet(stickerSet),
	}

	var compressDir  string
	var originDir    string = filepath.Join(config.CachedDir,    stickerSet.Name)
	var convertedDir string = filepath.Join(config.ConvertedDir, stickerSet.Name)

	// 根据要下载的类型设置压缩包的文件名和路径以及压缩包中的贴纸数量
	if needConvert {
		data.StickerSetFileName = fmt.Sprintf("%s(%d)%s", stickerSet.Name, data.StickerCount, common.StickerSetConvertedSuffix)
		compressDir = convertedDir
	} else {
		data.StickerSetFileName = fmt.Sprintf("%s(%d)%s", stickerSet.Name, data.StickerCount, common.StickerSetSuffix)
		compressDir = originDir
	}

	var isCompressed  bool = true
	var allDownloaded bool = true
	var allConverted  bool = true

	zipFilePath := filepath.Join(config.CompressedDir, fmt.Sprintf("%s_%s", data.StickerSetHash, data.StickerSetFileName))
	fileinfo, err := os.Stat(zipFilePath) // 检查压缩包文件是否存在
	if err != nil {
		if os.IsNotExist(err) {
			isCompressed = false

			for i, sticker := range stickerSet.Stickers {
				// 根据贴纸类型设置文件扩展名和统计贴纸数量
				switch {
				case sticker.IsVideo:
					data.StickerSuffix = ".webm"
					data.StickerConvertedSuffix = ".gif"
					data.WebM++
				case sticker.IsAnimated:
					data.StickerSuffix = ".tgs"
					data.StickerConvertedSuffix = ".gif"
					data.TGS++
				default:
					data.StickerSuffix = ".webp"
					data.StickerConvertedSuffix = ".png"
					data.WebP++
				}

				originStickerPath := filepath.Join(originDir, sticker.FileID + data.StickerSuffix)

				// 检查贴纸是否已缓存
				_, err := os.Stat(originStickerPath)
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

				// 是否需要转换
				if !config.Config.DisableConvert && needConvert {
					convertedStickerPath := filepath.Join(convertedDir, sticker.FileID + data.StickerConvertedSuffix)

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

			err = os.MkdirAll(config.CompressedDir, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("directory", config.CompressedDir).
					Msg("Failed to create zip file directory")
				return nil, fmt.Errorf("failed to create zip file directory [%s]: %w", config.CompressedDir, err)
			}
			err = compressStickerPack(stickerSet, compressDir, zipFilePath, needConvert)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", zipFilePath).
					Msg("Failed to compress sticker pack")
				return nil, fmt.Errorf("failed to compress sticker pack [%s]: %w", zipFilePath, err)
			}

			fileinfo, err := os.Stat(zipFilePath)
			if err != nil {
				logger.Error().
					Err(err).
					Str("zipFilePath", zipFilePath).
					Msg("Failed to read compressed sticker set zip file info")
				return nil, fmt.Errorf("failed to read compressed sticker set zip file [%s] info: %w", zipFilePath, err)
			}

			data.StickerSetSize = fileinfo.Size()
			logger.Debug().
				Str("compressDir", compressDir).
				Str("zipFilePath", zipFilePath).
				Msg("Compress sticker pack successfully")

		} else {
			logger.Error().
				Err(err).
				Str("zipFilePath", zipFilePath).
				Msg("Failed to read compressed sticker set zip file info")
			return nil, fmt.Errorf("failed to read compressed sticker set zip file [%s] info: %w", zipFilePath, err)
		}
	} else {
		data.StickerSetSize = fileinfo.Size()
		logger.Debug().
			Str("zipFilePath", zipFilePath).
			Msg("sticker set zip file already compressed")
	}

	// 读取压缩后的贴纸包
	data.Data, err = os.Open(zipFilePath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("zipFilePath", zipFilePath).
			Msg("Failed to open compressed sticker set zip file")
		return nil, fmt.Errorf("failed to open compressed sticker set zip file [%s]: %w", zipFilePath, err)
	}

	if isCompressed {
		// 存在已经完成压缩的贴纸包（原始格式或已转换）
		logger.Info().
			Str("zipFilePath", zipFilePath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("Sticker set already compressed")
	} else if needConvert && allConverted {
		// 需要转换，且已经全部转换过了，只是没有被打包
		logger.Info().
			Str("zipFilePath", zipFilePath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("Sticker set already converted")
	} else if allDownloaded {
		// 贴纸包中的贴纸已经全部缓存了
		logger.Info().
			Str("zipFilePath", zipFilePath).
			Dict("stickerSet", zerolog.Dict().
				Str("title", data.StickerSetTitle).
				Str("name", data.StickerSetName).
				Int("count", data.StickerCount),
			).
			Msg("Sticker set already cached")
	} else {
		// 新下载的贴纸包（如果有部分已经下载了也是这个）
		logger.Info().
			Str("zipFilePath", zipFilePath).
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
		zipFileWriter, err := zipWriter.Create(fmt.Sprintf("%s_%03d%s", sticker.SetName, i, dotFileSuffix))
		if err != nil { return err }

		// 复制文件内容到 ZIP
		_, err = io.Copy(zipFileWriter, file)
		if err != nil { return err }
	}
	return nil
}

func hashStickerSet(stickerSet *models.StickerSet) string {
	var sb strings.Builder
	for _, sticker := range stickerSet.Stickers {
		sb.WriteString(sticker.FileID)
	}
	return fmt.Sprintf("%x", md5.Sum([]byte(sb.String())))
}
