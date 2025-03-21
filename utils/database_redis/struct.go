package database_redis

import "github.com/go-telegram/bot/models"

type ChatInfo struct {
	// normal data
	ID       int64
	ChatName string
	ChatType models.ChatType
	AddTime  string

	// flags
	DefaultInlinePlugin string

	// status
	HasPendingCallbackQuery bool

	// message
	LatestMessage           string
	// inline
	LatestInlineQuery       string
	LatestInlineResult      string
	// callbackquery
	LatestCallbackQueryData string

	// inline
	InlineRequest        int
	InlineResult         int
	// message
	MessageNormal        int
	MessageCommand       int
	// callback query
	CallbackQuery        int
	// sticker
	StickerDownloaded    int
	StickerSetDownloaded int
}
