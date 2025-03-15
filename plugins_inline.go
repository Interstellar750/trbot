package main

import (
	"github.com/go-telegram/bot/models"
)

// 需要返回一个列表，将由程序的分页函数来控制分页和输出
type Plugin_Inline struct {
	command string
	handler func(*subHandlerOpts) []models.InlineQueryResult
}

func AddInlineHandlerPlugins(InlineHandlerPlugins ...Plugin_Inline) int {
	if AllPugins.Inline == nil { AllPugins.Inline = []Plugin_Inline{} }
	var pluginCount int
	for _, originPlugin := range InlineHandlerPlugins {
		AllPugins.Inline = append(AllPugins.Inline, originPlugin)
		pluginCount++
	}
	return pluginCount
}

// 完全由插件自行控制输出
type Plugin_InlineManual struct {
	command string
	handler func(*subHandlerOpts)
}

func AddInlineManualHandlerPlugins(InlineManualHandlerPlugins ...Plugin_InlineManual) int {
	if AllPugins.InlineManual == nil { AllPugins.InlineManual = []Plugin_InlineManual{} }
	var pluginCount int
	for _, originPlugin := range InlineManualHandlerPlugins {
		AllPugins.InlineManual = append(AllPugins.InlineManual, originPlugin)
		pluginCount++
	}
	return pluginCount
}
