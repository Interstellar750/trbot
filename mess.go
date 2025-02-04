package main

import (
	"archive/zip"
	"bufio"
	"context"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
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

// 定义消息类型枚举
type MessageType int
const (
	MessageTypeText MessageType = iota
	MessageTypePhoto
	MessageTypeVideo
	MessageTypeVoice
	MessageTypeDocument
	MessageTypeAudio
	MessageTypeForwarded
	MessageTypeSticker
	MessageTypeUnknown
)

// 判断消息的类型 需要重写
func getMessageType(message *models.Message) MessageType {
	switch {
	case message.ForwardOrigin != nil:
		return MessageTypeForwarded
	case message.Photo != nil:
		return MessageTypePhoto
	case message.Video != nil:
		return MessageTypeVideo
	case message.Voice != nil:
		return MessageTypeVoice
	case message.Document != nil:
		return MessageTypeDocument
	case message.Audio != nil:
		return MessageTypeAudio
	case message.Sticker != nil:
		return MessageTypeSticker
	case message.Text != "":
		return MessageTypeText
	default:
		return MessageTypeUnknown
	}
}

// 检查用户是否是管理员
// chat type: "private", "group", "supergroup", or "channel"
// not work for "private" chats
func userIsAdmin(ctx context.Context, thebot *bot.Bot, chatID, userID any) bool {
	admins, err := thebot.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{ ChatID: chatID })
	if err != nil {
		log.Printf("Failed to get chat administrators: %v", err)
		return false
	}

	var admins_usernames []string
	var admins_userIDs []int64

	for _, admin := range admins {
		if admin.Owner != nil {
		    admins_userIDs = append(admins_userIDs, admin.Owner.User.ID)
			if admin.Owner.User.Username != "" {
		        admins_usernames = append(admins_usernames, admin.Owner.User.Username)
		    }
		}
		if admin.Administrator != nil {
		    admins_userIDs = append(admins_userIDs, admin.Administrator.User.ID)
			if admin.Administrator.User.Username != "" {
		        admins_usernames = append(admins_usernames, admin.Administrator.User.Username)
		    }
		}
	}

	switch value := userID.(type) {
	case int:
		return AnyContains(value, admins_userIDs)
	case int64:
		// fmt.Println(value)
		return AnyContains(value, admins_userIDs)
	case string:
		// fmt.Println(value)
		if strings.ContainsAny(value, "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ_") {
			return AnyContains(value, admins_usernames)
		} else {
			int_userID, _ := strconv.Atoi(value)
			return AnyContains(int64(int_userID), admins_userIDs)
		}
	default:
		log.Println("userID type not supported")
		return false
	}
}
func userHavePermissionDeleteMessage(ctx context.Context, thebot *bot.Bot, chatID, userID any) bool {
	admins, err := thebot.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{
		ChatID: chatID,
	})
	if err != nil {
		log.Printf("Failed to get chat administrators: %v", err)
		return false
	}

	var adminshavepermission_usernames []string
	var adminshavepermission_userIDs []int64

	for _, admin := range admins {
		// owner allways have all permission
		if admin.Administrator != nil && admin.Administrator.CanDeleteMessages {
		    adminshavepermission_userIDs = append(adminshavepermission_userIDs, admin.Administrator.User.ID)
			if admin.Administrator.User.Username != "" {
		        adminshavepermission_usernames = append(adminshavepermission_usernames, admin.Administrator.User.Username)
		    }
		}
	}
	switch value := userID.(type) {
	case int:
		return AnyContains(value, adminshavepermission_userIDs)
	case int64:
		// fmt.Println(value)
		return AnyContains(value, adminshavepermission_userIDs)
	case string:
		// fmt.Println(value)
		if strings.ContainsAny(value, "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ_") {
			return AnyContains(value, adminshavepermission_usernames)
		} else {
			int_userID, _ := strconv.Atoi(value)
			return AnyContains(int64(int_userID), adminshavepermission_userIDs)
		}
	default:
		log.Println("userID type not supported")
		return false
	}
}
// 查找 bot token，优先级为 环境变量 > .env 文件
func whereIsBotToken() string {
	botToken = os.Getenv("BOT_TOKEN")
	if botToken == "" {
		// log.Printf("No bot token in environment, trying to read it from the .env file")
		godotenv.Load()
		botToken = os.Getenv("BOT_TOKEN")
		if botToken == "" {
			log.Fatalln("No bot token in environment and .env file, try create a bot from @botfather https://core.telegram.org/bots/tutorial#obtain-your-bot-token")
		}
		log.Printf("Get token from .env file: %s", showBotID())
	} else {
		log.Printf("Get token from environment: %s", showBotID())
	}
	return botToken
}

// 输出 bot 的 ID
func showBotID() string {
	var botID string
	for _, char := range botToken {
		if unicode.IsDigit(char) {
			botID += string(char)
		} else {
			break // 遇到非数字字符停止
		}
	}
	return botID
}

func usingWebhook() bool {
	webhookURL = os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		// 到这里可能变量没在环境里，试着读一下 .env 文件
		godotenv.Load()
		webhookURL = os.Getenv("WEBHOOK_URL")
		if webhookURL == "" {
			// 到这里就是 .env 文件里也没有，不启用
			log.Printf("No Webhook URL in environment and .env file, using getUpdate")
			return false
		}
		// 从 .env 文件中获取到了 URL，启用 Webhook
		log.Printf("Get Webhook URL from .env file: %s", webhookURL)
		return true
	} else {
		// 从环境变量中获取到了 URL，启用 Webhook
		log.Printf("Get Webhook URL from environment: %s", webhookURL)
		return true
	}
}

func setUpWebhook(ctx context.Context, thebot *bot.Bot, params *bot.SetWebhookParams) {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != params.URL {
		if webHookInfo.URL == "" {
			log.Println("Webhook not set, setting it now...")
		} else {
			log.Printf("unsame Webhook URL [%s], save it and setting up new URL...", webHookInfo.URL)
			printLogAndSave(time.Now().Format(time.RFC3339) + " (unsame) old Webhook URL: " + webHookInfo.URL)
		}
		success, err := thebot.SetWebhook(ctx, params)
		if err != nil { log.Panicln("Set Webhook URL err:", err) }
		if success { log.Println("Webhook setup successfully:", params.URL) }

	} else {
		log.Println("Webhook is already set:", webHookInfo.URL)
	}
}

func saveAndCleanRemoteWebhookURL(ctx context.Context, thebot *bot.Bot) *models.WebhookInfo {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != "" {
		log.Printf("found Webhook URL [%s] set at api server, save and clean it...", webHookInfo.URL)
		printLogAndSave(time.Now().Format(time.RFC3339) + " (remote) old Webhook URL: " + webHookInfo.URL)
		thebot.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
			DropPendingUpdates: false,
		})
		return webHookInfo
	}
	return nil
}

func printLogAndSave(message string) {
	log.Println(message)
	// 打开日志文件，如果不存在则创建
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	// 将文本写入日志文件
	_, err = file.WriteString(message + "\n")
	if err != nil {
		log.Println(err)
		return
	}
}

// 从 log.txt 读取文件
func readLog() []string {
	// 打开日志文件
	file, err := os.Open(logFile_path)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer file.Close()

	// 读取文件内容
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
		return nil
	}
	return lines
}

func privateLogToChat(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	thebot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    logChat_ID,
		Text:      fmt.Sprintf("[%s %s](t.me/@id%d) say: \n%s", update.Message.From.FirstName, update.Message.From.LastName, update.Message.Chat.ID, update.Message.Text),
		ParseMode: models.ParseModeMarkdownV1,
	})
}


// 如果 target 是 candidates 的一部分, 返回 true
// 常规类型会判定值是否相等，字符串如果包含也符合条件，例如 "bc" 在 "abcd" 中
func AnyContains(target any, candidates ...any) bool {
	for _, candidate := range candidates {
		if candidates == nil { continue }
		// fmt.Println(reflect.ValueOf(target).Kind(), reflect.ValueOf(candidate).Kind(), reflect.Array, reflect.Slice)
		targetKind := reflect.ValueOf(target).Kind()
		candidateKind := reflect.ValueOf(candidate).Kind()
		if targetKind != candidateKind && !AnyContains(candidateKind, reflect.Slice, reflect.Array) {
			log.Printf("[Warn] (func)AnyContains: candidate(%v) not match target(%v)", candidateKind, targetKind)
		}
		switch c := candidate.(type) {
		case string:
			if targetKind == reflect.String && strings.Contains(c, target.(string)) {
				return true
			}
		default:
			if reflect.DeepEqual(target, c) {
				return true
			}
			if reflect.ValueOf(c).Kind() == reflect.Slice || reflect.ValueOf(c).Kind() == reflect.Array {
				if checkNested(target, reflect.ValueOf(c)) {
					return true
				}
			}
		}
	}
	return false
}

// 为 AnyContains 的递归函数
func checkNested(target any, value reflect.Value) bool {
	// fmt.Println(reflect.ValueOf(value.Index(0).Interface()).Kind())
	if reflect.TypeOf(target) != reflect.TypeOf(value.Index(0).Interface()) && !AnyContains(reflect.ValueOf(value.Index(0).Interface()).Kind(), reflect.Slice, reflect.Array) {
		log.Printf("[Error] (func)AnyContains: candidates's subitem(%v) not match target(%v), skip this compare", reflect.TypeOf(value.Index(0).Interface()), reflect.TypeOf(target))
		return false
	}
	for i := 0; i < value.Len(); i++ {
		element := value.Index(i).Interface()
		switch c := element.(type) {
		case string:
			if reflect.ValueOf(target).Kind() == reflect.String && strings.Contains(c, target.(string)) {
				return true
			}
		default:
			if reflect.DeepEqual(target, c) {
				return true
			}
			// Check nested slices or arrays
			elemValue := reflect.ValueOf(c)
			if elemValue.Kind() == reflect.Slice || elemValue.Kind() == reflect.Array {
				if checkNested(target, elemValue) {
					return true
				}
			}
		}
	}
	return false
}

// 允许响应带有机器人用户名后缀的命令，例如 /help@examplebot
func commandMaybeWithSuffixUsername(commandFields []string, command string) bool {
	atBotUsername := "@" + botMe.Username
	if commandFields[0] == command || commandFields[0] == command + atBotUsername {
		return true
	}
	return false
}

func outputVersionInfo() string {
	// 获取 git sha 和 commit 时间
	c, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	// 获取 git 分支
	b, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	// 获取 commit 说明
	m, _ := exec.Command("git", "log", "-1", "--pretty=%s").Output()
	r := runtime.Version()
	h, _ := os.Hostname()
	info := fmt.Sprintf("Branch: %sCommit: [%s - %s](https://gitea.trle5.xyz/trle5/trbot/commit/%s)\nRuntime: %s\nHostname: %s", b, m, c[:10], c, r, h)
	return info
}

func showUserName(user *models.User) string {
	if user.LastName != "" {
		return user.FirstName + " " + user.LastName
	} else {
		return user.FirstName
	}
}

func showChatName(chat *models.Chat) string {
	if chat.Title != "" { // 群组
		return chat.Title
	} else if chat.LastName != "" { // 
		return chat.FirstName + " " + chat.LastName
	} else {
		return chat.FirstName
	}
}

func InlineResultPagination(queryFields []string, results []models.InlineQueryResult) []models.InlineQueryResult {
	// 当 result 的数量超过 InlineResultsPerPage 时，进行分页
	// fmt.Println(len(results), InlineResultsPerPage)
	if len(results) > InlineResultsPerPage {
		// 获取 update.InlineQuery.Query 末尾的 `-<数字>` 来选择输出第几页
		var pageNow int = 1
		var pageSize = (InlineResultsPerPage -1)

		if len(queryFields) > 0 && strings.HasPrefix(queryFields[len(queryFields)-1], InlinePaginationSymbol) {
			var err error
			pageNow, err = strconv.Atoi(queryFields[len(queryFields)-1][1:])
			if err != nil {
				if queryFields[len(queryFields)-1][1:] != "" {
					return []models.InlineQueryResult{&models.InlineQueryResultArticle{
						ID: "noThisOperation",
						Title: "无效的操作",
						Description: fmt.Sprintf("若您想翻页查看，请尝试输入 `%s2` 来查看第二页", InlinePaginationSymbol),
						InputMessageContent: &models.InputTextMessageContent{
							MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
							ParseMode: models.ParseModeMarkdownV1,
						},
					}}
				} else {
					return []models.InlineQueryResult{&models.InlineQueryResultArticle{
						ID: "keepInputNumber",
						Title: "请继续输入数字",
						Description: fmt.Sprintf("继续输入一个数字来查看对应的页面，当前列表有 %d 页", (len(results) + pageSize - 1) / pageSize),
						InputMessageContent: &models.InputTextMessageContent{
							MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
							ParseMode: models.ParseModeMarkdownV1,
						},
					}}
				}
			}
		}

		start := (pageNow - 1) * pageSize
		end := start + pageSize

		if start >= len(results) {
			return []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID: "wrongPageNumber",
				Title: "错误的页码",
				Description: fmt.Sprintf("您输入的页码 %d 超出范围，当前列表有 %d 页", pageNow, (len(results) + pageSize - 1) / pageSize),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
					ParseMode: models.ParseModeMarkdownV1,
				},
			}}
		}

		if end > len(results) {
			end = len(results)
		}
		pageResults := results[start:end]

		// 添加翻页提示
		if end < len(results) {
			totalPages := (len(results) + pageSize - 1) / pageSize
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID: "paginationPage",
				Title: fmt.Sprintf("当前您在第 %d 页", pageNow),
				Description: fmt.Sprintf("后面还有 %d 页内容，输入 %s%d 查看下一页", totalPages - pageNow, InlinePaginationSymbol, pageNow + 1),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		} else {
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID: "paginationPage",
				Title: fmt.Sprintf("当前您在第 %d 页", pageNow),
				Description: "后面已经没有东西了",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}

		return pageResults
	} else if len(queryFields) > 0 && strings.HasPrefix(queryFields[len(queryFields)-1], InlinePaginationSymbol) {
		return []models.InlineQueryResult{&models.InlineQueryResultArticle{
			ID: "noNeedPagination",
			Title: "没有多余的内容",
			Description: fmt.Sprintf("只有 %d 个条目，你想翻页也没有多的了", len(results)),
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
				ParseMode: models.ParseModeMarkdownV1,
			},
		}}
	} else {
		return results
	}
}

func InlineQueryMatchMultKeyword(queryFields []string, Keyword []string, inSubCommand bool) bool {
	var allkeywords int
	if strings.HasPrefix(queryFields[len(queryFields)-1], InlinePaginationSymbol) {
		queryFields = queryFields[:len(queryFields) -1]
	} else {
		allkeywords = len(queryFields)
	}
	if inSubCommand && len(queryFields) > 0 {
		queryFields = queryFields[1:]
	}
	if allkeywords == 1 {
		if AnyContains(queryFields[0], Keyword) {
			return true
		}
	} else {
		var allMatch bool = true

		for _, n := range queryFields {
			if AnyContains(n, Keyword) {
				// 保持 current 内容，继续过滤
				// continue
			} else {
				// 只要有一个关键词未匹配，返回 false
				allMatch = false
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}
