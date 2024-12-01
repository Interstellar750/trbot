package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"unicode"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

func echoSticker(filePath string) (*io.PipeReader) {
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
	botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Printf("No bot token in environment, trying to read it from the .env file")
		if godotenv.Load() != nil { log.Fatalln("Can't loading .env file") }
		botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
		if botToken == "" { log.Fatalln("No bot token in .env file, try create a bot from @botfather https://core.telegram.org/bots/tutorial#obtain-your-bot-token") }
		log.Printf("Get token from .env file: %s", showBotID())
	} else {
		log.Printf("Get token from environment: %.10s", botToken)
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
