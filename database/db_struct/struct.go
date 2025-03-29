package db_struct

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

	// latest data
	LatestMessage           string `yaml:"LatestMessage,omitempty"`
	LatestInlineQuery       string `yaml:"LatestInlineQuery,omitempty"`
	LatestInlineResult      string `yaml:"LatestInlineResult,omitempty"`
	LatestCallbackQueryData string `yaml:"LatestCallbackQueryData,omitempty"`

	// usage count
	InlineRequest        int `yaml:"InlineRequest,omitempty"`
	InlineResult         int `yaml:"InlineResult,omitempty"`
	MessageNormal        int `yaml:"MessageNormal,omitempty"`
	MessageCommand       int `yaml:"MessageCommand,omitempty"`
	CallbackQuery        int `yaml:"CallbackQuery,omitempty"`
	StickerDownloaded    int `yaml:"StickerDownloaded,omitempty"`
	StickerSetDownloaded int `yaml:"StickerSetDownloaded,omitempty"`
}

type ChatInfoField_CustomFlag string
const (
	DefaultInlinePlugin ChatInfoField_CustomFlag = "DefaultInlinePlugin"
)

type ChatInfoField_Status string
const (
	HasPendingCallbackQuery ChatInfoField_Status = "HasPendingCallbackQuery"
)

type ChatInfoField_LatestData string
const (
	LatestMessage ChatInfoField_LatestData = "LatestMessage"

	LatestInlineQuery  ChatInfoField_LatestData = "LatestInlineQuery"
	LatestInlineResult ChatInfoField_LatestData = "LatestInlineResult"

	LatestCallbackQueryData ChatInfoField_LatestData = "LatestCallbackQueryData"
	
)

type ChatInfoField_UsageCount string
const (
	InlineRequest ChatInfoField_UsageCount = "InlineRequest"
	InlineResult  ChatInfoField_UsageCount = "InlineResult"
	
	MessageNormal  ChatInfoField_UsageCount = "MessageNormal"
	MessageCommand ChatInfoField_UsageCount = "MessageCommand"
	
	CallbackQuery ChatInfoField_UsageCount = "CallbackQuery"
	
	StickerDownloaded    ChatInfoField_UsageCount = "StickerDownloaded"
	StickerSetDownloaded ChatInfoField_UsageCount = "StickerSetDownloaded"
)
