package main

import (
	"archive/zip"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/image/webp"
)


func echoSticker(opts *subHandlerOpts) (io.Reader, bool, error) {
	var fileSuffix string // `.webp`, `.webm`, `.tgs`, `.tgs`
	// 根据贴纸类型设置文件扩展名
	if opts.update.Message.Sticker.IsVideo {
		fileSuffix = "webm"
	} else if opts.update.Message.Sticker.IsAnimated {
		fileSuffix = "tgs"
	} else {
		fileSuffix = "webp"
	}

	var isCustomSticker bool = false

	var stickerfileName string // `CAACAgUA.` or `duck_2_video 5 CAACAgUA.`
	var stickerSetName  string // `-custom` or `duck_2_video`
	// 检查一下贴纸是否有 packName，没有的话就是自定义贴纸
	if opts.update.Message.Sticker.SetName != "" {
		var stickerIndex int // 存放贴纸在贴纸包中的索引
		// 获取贴纸包信息
		stickerSet, err := opts.thebot.GetStickerSet(opts.ctx, &bot.GetStickerSetParams{ Name: opts.update.Message.Sticker.SetName })
		if err == nil {
			// 寻找贴纸在贴纸包中的索引并赋值
			for i, n := range stickerSet.Stickers {
				if n.FileID == opts.update.Message.Sticker.FileID {
					stickerIndex = i
					break
				}
			}

			// 在这个条件下，贴纸包名和贴纸索引都存在，赋值完整的贴纸文件名
			stickerSetName = opts.update.Message.Sticker.SetName
			stickerfileName = fmt.Sprintf("%s %d %s.", opts.update.Message.Sticker.SetName, stickerIndex, opts.update.Message.Sticker.FileID)
		} else {
			// 到这里是因为用户发送的贴纸对应的贴纸包已经被删除了，但贴纸中的信息还有对应的 SetName，会触发查询，但因为贴纸包被删了就查不到，将 index 值设为 -1，缓存后当作自定义贴纸继续
			log.Println("error getting sticker set:", err)
			isCustomSticker = true
			stickerSetName = opts.update.Message.Sticker.SetName
			stickerfileName = fmt.Sprintf("%s %d %s.", opts.update.Message.Sticker.SetName, -1, opts.update.Message.Sticker.FileID)
		}
	} else {
		// 自定义贴纸，防止与普通贴纸包冲突，将贴纸包名设置为 `-custom`，文件名仅有 FileID 用于辨识
		isCustomSticker = true
		stickerSetName = "-custom"
		stickerfileName = fmt.Sprintf("%s.", opts.update.Message.Sticker.FileID)
	}

	// 保存贴纸源文件的目录 .cache/sticker/setName/
	var filePath       string = stickerCache_path + stickerSetName + "/"
	// 到贴纸文件的完整目录 .cache/sticker/setName/stickerFileName.webp
	var originFullPath string = filePath + stickerfileName + fileSuffix

	// 转码后为 png 格式的目录 .cache/sticker_png/setName/
	var filePathPNG   string = stickerCachePNG_path + stickerSetName + "/"
	// 转码后到 png 格式贴纸的完整目录 .cache/sticker_png/setName/stickerFileName.png
	var toPNGFullPath string = filePathPNG + stickerfileName + "png"

	_, err := os.Stat(originFullPath) // 检查贴纸源文件是否已缓存
	if os.IsNotExist(err) { // 文件不存在，进行下载
		// 从服务器获取文件信息
		fileinfo, err := opts.thebot.GetFile(opts.ctx, &bot.GetFileParams{ FileID: opts.update.Message.Sticker.FileID })
		if err != nil {
			return nil, false, fmt.Errorf("error getting fileinfo %s: %v", opts.update.Message.Sticker.FileID, err)
		}

		// 日志提示该文件没被缓存，正在下载
		if IsDebugMode {
			log.Printf("file [%s] doesn't exist, downloading %s", originFullPath, fileinfo.FilePath)
		}

		// 组合链接下载贴纸源文件
		resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, fileinfo.FilePath))
		if err != nil {
			return nil, false, fmt.Errorf("error downloading file %s: %v", fileinfo.FilePath, err)
		}
		defer resp.Body.Close()

		// 创建保存贴纸的目录
		err = os.MkdirAll(filePath, 0755)
		if err != nil {
			return nil, false, fmt.Errorf("error creating directory %s: %w", filePath, err)
		}

		// 创建贴纸空文件
		downloadedSticker, err := os.Create(originFullPath)
		if err != nil {
			return nil, false, fmt.Errorf("error creating file %s: %w", originFullPath, err)
		}
		defer downloadedSticker.Close()

		// 将下载的原贴纸写入空文件
		_, err = io.Copy(downloadedSticker, resp.Body)
		if err != nil {
			return nil, false, fmt.Errorf("error writing to file %s: %w", originFullPath, err)
		}
	} else if IsDebugMode { // 文件已存在，跳过下载
		log.Printf("file %s already exists", originFullPath)
	}

	// 存放最后读取并发送的文件完整目录 .cache/sticker/setName/stickerFileName.webp
	var finalFullPath string

	// 如果贴纸类型不是视频和矢量，进行转换
	if !opts.update.Message.Sticker.IsVideo && !opts.update.Message.Sticker.IsAnimated {
		_, err = os.Stat(toPNGFullPath) // 使用目录提前检查一下是否已经转换过
		if os.IsNotExist(err) { // 提示不存在，进行转换
			// 日志提示该文件没转换，正在转换
			if IsDebugMode {
				log.Printf("file [%s] does not exist, converting", toPNGFullPath)
			}
			// 创建保存贴纸的目录
			err = os.MkdirAll(filePathPNG, 0755)
			if err != nil {
				return nil, false, fmt.Errorf("error creating directory %s: %w", filePathPNG, err)
			}
			// 读取原贴纸文件，转码后存储到 png 格式贴纸的完整目录
			err = ConvertWebPToPNG(originFullPath, toPNGFullPath)
			if err != nil {
				return nil, false, fmt.Errorf("error converting webp to png %s: %w", originFullPath, err)
			}
		} else if IsDebugMode { // 文件存在，跳过转换
			log.Printf("file [%s] already converted", toPNGFullPath)
		}
		// 处理完成，将最后要读取的目录设为转码后 png 格式贴纸的完整目录
		finalFullPath = toPNGFullPath
	} else {
		// 不需要转码，直接读取原贴纸文件
		finalFullPath = originFullPath
	}

	// 逻辑完成，读取最后的文件，返回给上一级函数
	cachedfile, err := os.Open(finalFullPath)
	if err != nil {
		return nil, false, fmt.Errorf("error opening file %s: %w", finalFullPath, err)
	}

	if isCustomSticker {
		log.Printf("sticker [%s] is downloaded", finalFullPath)
	}

	return cachedfile, isCustomSticker, nil
}

func getStickerPack(opts *subHandlerOpts, stickerSet *models.StickerSet, isOnlyPNG bool) (io.Reader, int, error) {

	filePath    := stickerCache_path + stickerSet.Name + "/"
	filePathPNG := stickerCachePNG_path + stickerSet.Name + "/"

	var allCached    bool = true
	var allConverted bool = true

	var stickerCount_webm int
	var stickerCount_tgs int
	var stickerCount_webp int

	var stickersInZip int

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

		var originFullPath string = filePath + stickerfileName + fileSuffix
		var toPNGFullPath  string = filePathPNG + stickerfileName + "png"

		_, err := os.Stat(originFullPath) // 检查单个贴纸是否已缓存
		if os.IsNotExist(err) {
			allCached = false

			// 从服务器获取文件内容
			fileinfo, err := opts.thebot.GetFile(opts.ctx, &bot.GetFileParams{ FileID: sticker.FileID })
			if err != nil {
				return nil, 0, fmt.Errorf("error getting file info %s: %v", sticker.FileID, err)
			}

			if IsDebugMode {
				log.Printf("file [%s] does not exist, downloading %s", originFullPath, fileinfo.FilePath)
			}

			// 下载贴纸文件
			resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, fileinfo.FilePath))
			if err != nil {
				return nil, 0, fmt.Errorf("error downloading file %s: %v", fileinfo.FilePath, err)
			}
			defer resp.Body.Close()

			err = os.MkdirAll(filePath, 0755)
			if err != nil {
				return nil, 0, fmt.Errorf("error creating directory %s: %w", filePath, err)
			}

			// 创建文件并保存
			downloadedSticker, err := os.Create(originFullPath)
			if err != nil {
				return nil, 0, fmt.Errorf("error creating file %s: %w", originFullPath, err)
			}
			defer downloadedSticker.Close()

			// 将下载的内容写入文件
			_, err = io.Copy(downloadedSticker, resp.Body)
			if err != nil {
				return nil, 0, fmt.Errorf("error writing to file %s: %w", originFullPath, err)
			}
		} else if IsDebugMode {
			// 存在跳过下载过程
			log.Printf("file [%s] already exists", originFullPath)
		}

		// 仅需要 PNG 格式时进行转换
		if isOnlyPNG && !sticker.IsVideo && !sticker.IsAnimated {
			_, err = os.Stat(toPNGFullPath)
			if os.IsNotExist(err) {
				allConverted = false
				if IsDebugMode {
					log.Printf("file [%s] does not exist, converting", toPNGFullPath)
				}
				// 创建保存贴纸的目录
				err = os.MkdirAll(filePathPNG, 0755)
				if err != nil {
					return nil, 0, fmt.Errorf("error creating directory %s: %w", filePathPNG, err)
				}
				// 将 webp 转换为 png
				err = ConvertWebPToPNG(originFullPath, toPNGFullPath)
				if err != nil {
					return nil, 0, fmt.Errorf("error converting webp to png %s: %w", originFullPath, err)
				}
			} else if IsDebugMode {
				log.Printf("file [%s] already converted", toPNGFullPath)
			}
		}
	}

	var zipFileName string
	var compressFolderPath string

	var isZiped bool = true

	// 根据要下载的类型设置压缩包的文件名和路径以及压缩包中的贴纸数量
	if isOnlyPNG {
		stickersInZip = stickerCount_webp
		zipFileName = fmt.Sprintf("%s(%d)_png.zip", stickerSet.Name, stickersInZip)
		compressFolderPath = filePathPNG
	} else {
		stickersInZip = stickerCount_webp + stickerCount_webm + stickerCount_tgs
		zipFileName = fmt.Sprintf("%s(%d).zip", stickerSet.Name, stickersInZip)
		compressFolderPath = filePath
	}

	_, err := os.Stat(stickerCacheZip_path + zipFileName) // 检查压缩包文件是否存在
	if os.IsNotExist(err) {
		isZiped = false
		err = os.MkdirAll(stickerCacheZip_path, 0755)
		if err != nil {
			return nil, 0, fmt.Errorf("error creating directory %s: %w", stickerCacheZip_path, err)
		}
		err = zipFolder(compressFolderPath, stickerCacheZip_path + zipFileName)
		if err != nil {
			return nil, 0, fmt.Errorf("error zipping folder %s: %w", compressFolderPath, err)
		} else if IsDebugMode {
			log.Println("successfully zipped folder", stickerCacheZip_path + zipFileName)
		}
	} else if IsDebugMode {
		log.Println("zip file already exists", stickerCacheZip_path + zipFileName)
	}

	// 读取压缩后的贴纸包
	zipedSet, err := os.Open(stickerCacheZip_path + zipFileName)
	if err != nil {
		return nil, 0, fmt.Errorf("error opening zip file %s: %w", stickerCacheZip_path + zipFileName, err)
	}

	if isZiped { // 存在已经完成压缩的贴纸包
		log.Printf("sticker pack \"%s\"[%s](%d) is already zipped", stickerSet.Title, stickerSet.Name, stickersInZip)
	} else if isOnlyPNG && allConverted { // 仅需要 PNG 格式，且贴纸包完全转换成 PNG 格式，但尚未压缩
		log.Printf("sticker pack \"%s\"[%s](%d) is already converted", stickerSet.Title, stickerSet.Name, stickersInZip)
	} else if allCached { // 贴纸包中的贴纸已经全部缓存了
		log.Printf("sticker pack \"%s\"[%s](%d) is already cached", stickerSet.Title, stickerSet.Name, stickersInZip)
	} else { // 新下载的贴纸包（如果有部分已经下载了也是这个）
		log.Printf("sticker pack \"%s\"[%s](%d) is downloaded", stickerSet.Title, stickerSet.Name, stickersInZip)
	}

	return zipedSet, stickersInZip, nil
}

func ConvertWebPToPNG(webpPath, pngPath string) error {
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

func zipFolder(srcDir, zipFile string) error {
	// 创建 ZIP 文件
	outFile, err := os.Create(zipFile)
	if err != nil {
		return err
	}
	defer outFile.Close()

	// 创建 ZIP 写入器
	zipWriter := zip.NewWriter(outFile)
	defer zipWriter.Close()

	// 遍历文件夹并添加文件到 ZIP
	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 计算文件在 ZIP 中的相对路径
		relPath, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}

		// 如果是目录，则跳过
		if info.IsDir() {
			return nil
		}

		// 打开文件
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		// 创建 ZIP 内的文件
		zipFileWriter, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		// 复制文件内容到 ZIP
		_, err = io.Copy(zipFileWriter, file)
		return err
	})

	return err
}

func echoStickerHandler(opts *subHandlerOpts) {
	// 下载 webp 格式的贴纸
	if IsDebugMode {
		fmt.Println(opts.update.Message.Sticker)
	}

	stickerdata, isCustomSticker, err := echoSticker(opts)
	if err != nil {
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID:    opts.update.Message.Chat.ID,
			Text:      fmt.Sprintf("下载贴纸时发生了一些错误\n<blockquote>Error downloading sticker: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
	}

	if stickerdata == nil {
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID:    opts.update.Message.Chat.ID,
			Text:      "未能获取到贴纸",
			ParseMode: models.ParseModeMarkdownV1,
		})
		return
	}

	documentParams := &bot.SendDocumentParams{
		ChatID:          opts.update.Message.Chat.ID,
		ParseMode:       models.ParseModeMarkdownV1,
		ReplyParameters: &models.ReplyParameters{MessageID: opts.update.Message.ID},
	}

	// 仅在不为自定义贴纸时显示下载整个贴纸包按钮
	if !isCustomSticker {
		documentParams.ReplyMarkup = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "下载贴纸包中的静态贴纸", CallbackData: fmt.Sprintf("S_%s", opts.update.Message.Sticker.SetName)},
			},
			{
				{Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", opts.update.Message.Sticker.SetName)},
			},
		}}
		// if opts.update.Message.Sticker.IsVideo {
		// 	documentParams.Caption  = fmt.Sprintf("[%s](https://t.me/addstickers/%s)\nsee [wikipedia/WebM](https://wikipedia.org/wiki/WebM)", opts.update.Message.Sticker.Title, opts.update.Message.Sticker.SetName)
		// 	documentParams.Document = &models.InputFileUpload{Filename: "sticker.webm", Data: stickerdata}
		// } else if opts.update.Message.Sticker.IsAnimated {
		// 	documentParams.Caption  = "see [stickers/animated-stickers](https://core.telegram.org/stickers#animated-stickers)"
		// 	documentParams.Document = &models.InputFileUpload{Filename: "sticker.tgs.file", Data: stickerdata}
		// } else {
		// 	documentParams.Document = &models.InputFileUpload{Filename: "sticker.png", Data: stickerdata}
		// }
	}

	if opts.update.Message.Sticker.IsVideo {
		documentParams.Caption = "see [wikipedia/WebM](https://wikipedia.org/wiki/WebM)"
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.webm", Data: stickerdata}
	} else if opts.update.Message.Sticker.IsAnimated {
		documentParams.Caption = "see [stickers/animated-stickers](https://core.telegram.org/stickers#animated-stickers)"
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.tgs.file", Data: stickerdata}
	} else {
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.png", Data: stickerdata}
	}

	opts.thebot.SendDocument(opts.ctx, documentParams)
}
