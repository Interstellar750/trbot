package db_struct

import (
	"github.com/go-telegram/bot/models"
)

type ChatInfo struct {
	// normal data
	ID       int64           `yaml:"ID"`
	ChatName string          `yaml:"ChatName"`
	ChatType models.ChatType `yaml:"ChatType"`
	AddTime  string          `yaml:"AddTime"`

	Flag   map[Flag  ]string `yaml:"Flag,omitempty"`
	Status map[Status]bool   `yaml:"Status,omitempty"`

	LatestData map[LatestData]string `yaml:"LatestData,omitempty"`
	UsageCount map[UsageCount]int    `yaml:"UsageCount,omitempty"`
}

type Flag string
const (
	DefaultInlinePlugin Flag = "DefaultInlinePlugin"
)

type Status string
const (
	IsBanned Status = "IsBanned"
)

type LatestData string
const (
	LatestMessage LatestData = "LatestMessage"

	LatestInlineQuery  LatestData = "LatestInlineQuery"
	LatestInlineResult LatestData = "LatestInlineResult"

	LatestCallbackQueryData LatestData = "LatestCallbackQueryData"
)

type UsageCount string
const (
	MessageNormal  UsageCount = "MessageNormal"
	MessageCommand UsageCount = "MessageCommand"
	CallbackQuery  UsageCount = "CallbackQuery"
	InlineRequest  UsageCount = "InlineRequest"
	InlineResult   UsageCount = "InlineResult"

	StickerDownloaded    UsageCount = "StickerDownloaded"
	StickerSetDownloaded UsageCount = "StickerSetDownloaded"
)
