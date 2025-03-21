package database_struct

import "github.com/go-telegram/bot/models"

type ChatInfo struct {
	// normal data
	ID       int64           `yaml:"ID"`
	ChatName string          `yaml:"ChatName"`
	ChatType models.ChatType `yaml:"ChatType"`
	AddTime  string          `yaml:"AddTime"`

	// flags
	DefaultInlinePlugin string `yaml:"DefaultInlinePlugin,omitempty"`

	// status
	HasPendingCallbackQuery bool `yaml:"HasPendingCallbackQuery,omitempty"`

	// message
	LatestMessage           string `yaml:"LatestMessage,omitempty"`
	// inline
	LatestInlineQuery       string `yaml:"LatestInlineQuery,omitempty"`
	LatestInlineResult      string `yaml:"LatestInlineResult,omitempty"`
	// callbackquery
	LatestCallbackQueryData string `yaml:"LatestCallbackQueryData,omitempty"`

	// inline
	InlineRequest        int `yaml:"InlineRequest,omitempty"`
	InlineResult         int `yaml:"InlineResult,omitempty"`
	// message
	MessageNormal        int `yaml:"MessageNormal,omitempty"`
	MessageCommand       int `yaml:"MessageCommand,omitempty"`
	// callback query
	CallbackQuery        int `yaml:"CallbackQuery,omitempty"`
	// sticker
	StickerDownloaded    int `yaml:"StickerDownloaded,omitempty"`
	StickerSetDownloaded int `yaml:"StickerSetDownloaded,omitempty"`
}
