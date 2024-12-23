package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

func echoSticker(filePath string) *io.PipeReader {
	log.Printf("https://api.telegram.org/file/bot%s/%s\n", botToken, filePath)
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, filePath))
	if err != nil { log.Printf("error downloading file: %v", err) }
	defer resp.Body.Close()
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		_, err := io.Copy(writer, resp.Body)
		if err != nil {
			log.Println("Error copying to pipe:", err)
		}
	}()

	return reader
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

// 判断消息的类型
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
func userIsAdmin(ctx context.Context, thebot *bot.Bot, chatID, userID int64) bool {
	admins, err := thebot.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{
		ChatID: chatID,
	})
	if err != nil {
		log.Printf("Failed to get chat administrators: %v", err)
		return false
	}
	for _, admin := range admins {
		// fmt.Println(admin.Administrator.User.ID, userID)
		// fmt.Println(admin.Owner.User.ID, userID)
		if admin.Administrator != nil && admin.Administrator.User.ID == userID {
			return true
		}
		if admin.Owner != nil && admin.Owner.User.ID == userID {
			return true
		}
	}
	return false
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

func setUpWebhook(ctx context.Context, thebot *bot.Bot, url string) {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != url {
		if webHookInfo.URL == "" {
			log.Println("Webhook is not setup, setting up now...")
		} else {
			log.Printf("unsame Webhook URL [%s], save it and setting up new URL...", webHookInfo.URL)
			logToFile(time.Now().Format("2006/01/02 15:04:05") + " (unsame) old Webhook URL: " + webHookInfo.URL)
		}
		success, err := thebot.SetWebhook(ctx, &bot.SetWebhookParams{
			URL: url,
		})
		if err != nil { log.Println(err) }
		if success { log.Println("Webhook is set up successfully") }

	} else {
		log.Println("Webhook is already setup")
	}
}

func saveAndCleanRemoteWebhookURL(ctx context.Context, thebot *bot.Bot) {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != "" {
		log.Printf("found Webhook URL [%s] set at api server, save and clean it...", webHookInfo.URL)
		logToFile(time.Now().Format("2006/01/02 15:04:05") + " (remote) old Webhook URL: " + webHookInfo.URL)
		thebot.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
			DropPendingUpdates: true,
		})
	}
}

func logToFile(message string) {
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

func readLog() []string {
	// 打开日志文件
	file, err := os.Open(logfile_path)
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

func AnyContains(query string, chars ...string) bool {
	for _, char := range chars {
		if strings.Contains(char, query) {
			return true
		}
	}
	return false
}
