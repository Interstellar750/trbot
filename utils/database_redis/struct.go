package database_redis

import "github.com/go-telegram/bot/models"

// Hash
type BaseInfo struct {
	ChatName string
	ChatType models.ChatType
	AddTime  string
}

// Hash
type LatestContent struct {
	LatestMessage           string
	LatestInlineQuery       string
	LatestInlineResult      string
	LatestCallbackQueryData string
}

// Zset
type UsageCount struct {
	InlineRequst       int
	InlineCallback     int
	MessageNormal      int
	MessageCommand     int
	StickerDownload    int
	StickerSetDownload int
}
