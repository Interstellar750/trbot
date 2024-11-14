package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func echoSticker(filePath string) (*io.PipeReader) {
	fmt.Printf("https://api.telegram.org/file/bot%s/%s\n", botToken, filePath)
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, filePath))
	if err != nil { log.Printf("error downloading file: %v", err) }
	// defer resp.Body.Close()
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		_, err := io.Copy(writer, resp.Body)
		if err != nil {
			fmt.Println("Error copying to pipe:", err)
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
func GetMessageType(message *models.Message) MessageType {
	switch {
	case message.ForwardOrigin != nil:
		return MessageTypeForwarded
	case message.Text != "":
		return MessageTypeText
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
	default:
		return MessageTypeUnknown
	}
}


// 检查用户是否是管理员
// chat type: “private”, “group”, “supergroup”, or “channel”
// not work for "private" chats
func checkIfAdmin(ctx context.Context, thebot *bot.Bot, chatID, userID int64) bool {
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
