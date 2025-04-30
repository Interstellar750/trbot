package plugins

import (
	"archive/zip"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"trbot/database"
	"trbot/database/db_struct"
	"trbot/utils/consts"
	"trbot/utils/handler_utils"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"golang.org/x/image/webp"
)

var StickerCache_path    string = consts.Cache_path + "sticker/"
var StickerCachePNG_path string = consts.Cache_path + "sticker_png/"
var StickerCacheZip_path string = consts.Cache_path + "sticker_zip/"

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
}

type stickerDatas struct {
	Data            io.Reader
	IsCustomSticker bool
	StickerCount    int
	StickerIndex    int
	StickerSetName  string // 贴纸包的 urlname
	StickerSetTitle string // 贴纸包名称
}

func EchoSticker(opts *handler_utils.SubHandlerOpts) (*stickerDatas, error) {
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
		if err == nil {
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
		} else {
			// 到这里是因为用户发送的贴纸对应的贴纸包已经被删除了，但贴纸中的信息还有对应的 SetName，会触发查询，但因为贴纸包被删了就查不到，将 index 值设为 -1，缓存后当作自定义贴纸继续
			log.Println("error getting sticker set:", err)
			data.IsCustomSticker   = true
			stickerSetNamePrivate  = opts.Update.Message.Sticker.SetName
			stickerFileNameWithDot = fmt.Sprintf("%s %d %s.", opts.Update.Message.Sticker.SetName, -1, opts.Update.Message.Sticker.FileID)
		}
	} else {
		// 自定义贴纸，防止与普通贴纸包冲突，将贴纸包名设置为 `-custom`，文件名仅有 FileID 用于辨识
		data.IsCustomSticker   = true
		stickerSetNamePrivate  = "-custom"
		stickerFileNameWithDot = fmt.Sprintf("%s.", opts.Update.Message.Sticker.FileID)
	}

	var filePath       string = StickerCache_path + stickerSetNamePrivate + "/"       // 保存贴纸源文件的目录 .cache/sticker/setName/
	var originFullPath string = filePath + stickerFileNameWithDot + fileSuffix // 到贴纸文件的完整目录 .cache/sticker/setName/stickerFileName.webp

	var filePathPNG   string = StickerCachePNG_path + stickerSetNamePrivate + "/"  // 转码后为 png 格式的目录 .cache/sticker_png/setName/
	var toPNGFullPath string = filePathPNG + stickerFileNameWithDot + "png" // 转码后到 png 格式贴纸的完整目录 .cache/sticker_png/setName/stickerFileName.png

	_, err := os.Stat(originFullPath) // 检查贴纸源文件是否已缓存
	// 如果文件不存在，进行下载，否则跳过
	if os.IsNotExist(err) {
		// 从服务器获取文件信息
		fileinfo, err := opts.Thebot.GetFile(opts.Ctx, &bot.GetFileParams{ FileID: opts.Update.Message.Sticker.FileID })
		if err != nil { return nil, fmt.Errorf("error getting fileinfo %s: %v", opts.Update.Message.Sticker.FileID, err) }

		// 日志提示该文件没被缓存，正在下载
		if consts.IsDebugMode { log.Printf("file [%s] doesn't exist, downloading %s", originFullPath, fileinfo.FilePath) }

		// 组合链接下载贴纸源文件
		resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", consts.BotToken, fileinfo.FilePath))
		if err != nil { return nil, fmt.Errorf("error downloading file %s: %v", fileinfo.FilePath, err) }
		defer resp.Body.Close()

		// 创建保存贴纸的目录
		err = os.MkdirAll(filePath, 0755)
		if err != nil { return nil, fmt.Errorf("error creating directory %s: %w", filePath, err) }

		// 创建贴纸空文件
		downloadedSticker, err := os.Create(originFullPath)
		if err != nil { return nil, fmt.Errorf("error creating file %s: %w", originFullPath, err) }
		defer downloadedSticker.Close()

		// 将下载的原贴纸写入空文件
		_, err = io.Copy(downloadedSticker, resp.Body)
		if err != nil { return nil, fmt.Errorf("error writing to file %s: %w", originFullPath, err) }
	} else if consts.IsDebugMode {
		// 文件已存在，跳过下载
		log.Printf("file %s already exists", originFullPath)
	}

	var finalFullPath string // 存放最后读取并发送的文件完整目录 .cache/sticker/setName/stickerFileName.webp

	// 如果贴纸类型不是视频和矢量，进行转换
	if !opts.Update.Message.Sticker.IsVideo && !opts.Update.Message.Sticker.IsAnimated {
		_, err = os.Stat(toPNGFullPath) // 使用目录提前检查一下是否已经转换过
		// 如果提示不存在，进行转换
		if os.IsNotExist(err) {
			// 日志提示该文件没转换，正在转换
			if consts.IsDebugMode { log.Printf("file [%s] does not exist, converting", toPNGFullPath) }

			// 创建保存贴纸的目录
			err = os.MkdirAll(filePathPNG, 0755)
			if err != nil { return nil, fmt.Errorf("error creating directory %s: %w", filePathPNG, err) }

			// 读取原贴纸文件，转码后存储到 png 格式贴纸的完整目录
			err = ConvertWebPToPNG(originFullPath, toPNGFullPath)
			if err != nil { return nil, fmt.Errorf("error converting webp to png %s: %w", originFullPath, err) }
		} else if consts.IsDebugMode {
			// 文件存在，跳过转换
			log.Printf("file [%s] already converted", toPNGFullPath)
		}
		// 处理完成，将最后要读取的目录设为转码后 png 格式贴纸的完整目录
		finalFullPath = toPNGFullPath
	} else {
		// 不需要转码，直接读取原贴纸文件
		finalFullPath = originFullPath
	}

	// 逻辑完成，读取最后的文件，返回给上一级函数
	data.Data, err = os.Open(finalFullPath)
	if err != nil { return nil, fmt.Errorf("error opening file %s: %w", finalFullPath, err) }

	if data.IsCustomSticker { log.Printf("sticker [%s] is downloaded", finalFullPath) }

	return &data, nil
}

func getStickerPack(opts *handler_utils.SubHandlerOpts, stickerSet *models.StickerSet, isOnlyPNG bool) (*stickerDatas, error) {
	var data stickerDatas = stickerDatas{
		IsCustomSticker: false,
		StickerSetName:  stickerSet.Name,
		StickerSetTitle: stickerSet.Title,
	}

	filePath    := StickerCache_path + stickerSet.Name + "/"
	filePathPNG := StickerCachePNG_path + stickerSet.Name + "/"

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

		var originFullPath string = filePath    + stickerfileName + fileSuffix
		var toPNGFullPath  string = filePathPNG + stickerfileName + "png"

		_, err := os.Stat(originFullPath) // 检查单个贴纸是否已缓存
		if os.IsNotExist(err) {
			allCached = false

			// 从服务器获取文件内容
			fileinfo, err := opts.Thebot.GetFile(opts.Ctx, &bot.GetFileParams{ FileID: sticker.FileID })
			if err != nil { return nil, fmt.Errorf("error getting file info %s: %v", sticker.FileID, err) }

			if consts.IsDebugMode {
				log.Printf("file [%s] does not exist, downloading %s", originFullPath, fileinfo.FilePath)
			}

			// 下载贴纸文件
			resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", consts.BotToken, fileinfo.FilePath))
			if err != nil { return nil, fmt.Errorf("error downloading file %s: %v", fileinfo.FilePath, err) }
			defer resp.Body.Close()

			err = os.MkdirAll(filePath, 0755)
			if err != nil { return nil, fmt.Errorf("error creating directory %s: %w", filePath, err) }

			// 创建文件并保存
			downloadedSticker, err := os.Create(originFullPath)
			if err != nil { return nil, fmt.Errorf("error creating file %s: %w", originFullPath, err) }
			defer downloadedSticker.Close()

			// 将下载的内容写入文件
			_, err = io.Copy(downloadedSticker, resp.Body)
			if err != nil { return nil, fmt.Errorf("error writing to file %s: %w", originFullPath, err) }
		} else if consts.IsDebugMode {
			// 存在跳过下载过程
			log.Printf("file [%s] already exists", originFullPath)
		}

		// 仅需要 PNG 格式时进行转换
		if isOnlyPNG && !sticker.IsVideo && !sticker.IsAnimated {
			_, err = os.Stat(toPNGFullPath)
			if os.IsNotExist(err) {
				allConverted = false
				if consts.IsDebugMode {
					log.Printf("file [%s] does not exist, converting", toPNGFullPath)
				}
				// 创建保存贴纸的目录
				err = os.MkdirAll(filePathPNG, 0755)
				if err != nil {
					return nil, fmt.Errorf("error creating directory %s: %w", filePathPNG, err)
				}
				// 将 webp 转换为 png
				err = ConvertWebPToPNG(originFullPath, toPNGFullPath)
				if err != nil {
					return nil, fmt.Errorf("error converting webp to png %s: %w", originFullPath, err)
				}
			} else if consts.IsDebugMode {
				log.Printf("file [%s] already converted", toPNGFullPath)
			}
		}
	}

	var zipFileName string
	var compressFolderPath string

	var isZiped bool = true

	// 根据要下载的类型设置压缩包的文件名和路径以及压缩包中的贴纸数量
	if isOnlyPNG {
		if stickerCount_webp == 0 {
			return nil, fmt.Errorf("there are no static stickers in the sticker pack")
		}
		data.StickerCount = stickerCount_webp
		zipFileName = fmt.Sprintf("%s(%d)_png.zip", stickerSet.Name, data.StickerCount)
		compressFolderPath = filePathPNG
	} else {
		data.StickerCount = stickerCount_webp + stickerCount_webm + stickerCount_tgs
		zipFileName = fmt.Sprintf("%s(%d).zip", stickerSet.Name, data.StickerCount)
		compressFolderPath = filePath
	}

	_, err := os.Stat(StickerCacheZip_path + zipFileName) // 检查压缩包文件是否存在
	if os.IsNotExist(err) {
		isZiped = false
		err = os.MkdirAll(StickerCacheZip_path, 0755)
		if err != nil {
			return nil, fmt.Errorf("error creating directory %s: %w", StickerCacheZip_path, err)
		}
		err = zipFolder(compressFolderPath, StickerCacheZip_path + zipFileName)
		if err != nil {
			return nil, fmt.Errorf("error zipping folder %s: %w", compressFolderPath, err)
		} else if consts.IsDebugMode {
			log.Println("successfully zipped folder", StickerCacheZip_path + zipFileName)
		}
	} else if consts.IsDebugMode {
		log.Println("zip file already exists", StickerCacheZip_path + zipFileName)
	}

	// 读取压缩后的贴纸包
	data.Data, err = os.Open(StickerCacheZip_path + zipFileName)
	if err != nil {
		return nil, fmt.Errorf("error opening zip file %s: %w", StickerCacheZip_path + zipFileName, err)
	}

	if isZiped { // 存在已经完成压缩的贴纸包
		log.Printf("sticker pack \"%s\"[%s](%d) is already zipped", stickerSet.Title, stickerSet.Name, data.StickerCount)
	} else if isOnlyPNG && allConverted { // 仅需要 PNG 格式，且贴纸包完全转换成 PNG 格式，但尚未压缩
		log.Printf("sticker pack \"%s\"[%s](%d) is already converted", stickerSet.Title, stickerSet.Name, data.StickerCount)
	} else if allCached { // 贴纸包中的贴纸已经全部缓存了
		log.Printf("sticker pack \"%s\"[%s](%d) is already cached", stickerSet.Title, stickerSet.Name, data.StickerCount)
	} else { // 新下载的贴纸包（如果有部分已经下载了也是这个）
		log.Printf("sticker pack \"%s\"[%s](%d) is downloaded", stickerSet.Title, stickerSet.Name, data.StickerCount)
	}

	return &data, nil
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

func EchoStickerHandler(opts *handler_utils.SubHandlerOpts) {
	// 下载 webp 格式的贴纸
	if consts.IsDebugMode {
		fmt.Println(opts.Update.Message.Sticker)
	}

	database.IncrementalUsageCount(opts.Ctx, opts.Update.Message.Chat.ID, db_struct.StickerDownloaded)

	stickerData, err := EchoSticker(opts)
	if err != nil {
		opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Update.Message.Chat.ID,
			Text:      fmt.Sprintf("下载贴纸时发生了一些错误\n<blockquote expandable>Error downloading sticker: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
	}

	if stickerData == nil || stickerData.Data == nil {
		opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Update.Message.Chat.ID,
			Text:      "未能获取到贴纸",
			ParseMode: models.ParseModeMarkdownV1,
		})
		return
	}

	documentParams := &bot.SendDocumentParams{
		ChatID:          opts.Update.Message.Chat.ID,
		ParseMode:       models.ParseModeHTML,
		ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
	}

	var stickerFilePrefix, stickerFileSuffix string

	if opts.Update.Message.Sticker.IsVideo {
		documentParams.Caption = "<blockquote>see <a href=\"https://wikipedia.org/wiki/WebM\">wikipedia/WebM</a></blockquote>"
		stickerFileSuffix = "webm"
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
		documentParams.ReplyMarkup = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "下载贴纸包中的静态贴纸", CallbackData: fmt.Sprintf("S_%s", opts.Update.Message.Sticker.SetName)},
			},
			{
				{Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", opts.Update.Message.Sticker.SetName)},
			},
		}}
	}

	documentParams.Document = &models.InputFileUpload{Filename: fmt.Sprintf("%s.%s", stickerFilePrefix, stickerFileSuffix), Data: stickerData.Data}

	opts.Thebot.SendDocument(opts.Ctx, documentParams)
}

func DownloadStickerPackCallBackHandler(opts *handler_utils.SubHandlerOpts) {
	botMessage, _ := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
		Text:   "已请求下载，请稍候",
		ParseMode: models.ParseModeMarkdownV1,
	})

	database.IncrementalUsageCount(opts.Ctx, opts.Update.Message.Chat.ID, db_struct.StickerSetDownloaded)

	var packName string
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
		log.Printf("error getting sticker set: %v", err)
		opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.CallbackQuery.From.ID,
			Text:   fmt.Sprintf("获取贴纸包时发生了一些错误\n<blockquote expandable>Error getting sticker set: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
		return
	}

	stickerData, err := getStickerPack(opts, stickerSet, isOnlyPNG)
	if err != nil {
		log.Println("Error downloading sticker:", err)
		opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.CallbackQuery.From.ID,
			Text:   fmt.Sprintf("下载贴纸包时发生了一些错误\n<blockquote expandable>Error download sticker set: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
	}
	if stickerData == nil || stickerData.Data == nil {
		opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.CallbackQuery.From.ID,
			Text:   "未能获取到压缩包",
			ParseMode: models.ParseModeMarkdownV1,
		})
		return
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

	opts.Thebot.SendDocument(opts.Ctx, documentParams)

	opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
		ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: botMessage.ID,
	})

}
