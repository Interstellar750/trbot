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

	// latest datas
	LatestMessage           string
	LatestInlineQuery       string
	LatestInlineResult      string
	LatestCallbackQueryData string

	// usage counts
	InlineRequst         int
	InlineCallback       int
	MessageNormal        int
	MessageCommand       int
	StickerDownloaded    int
	StickerSetDownloaded int
}
