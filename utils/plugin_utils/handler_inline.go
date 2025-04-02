package plugin_utils

import (
	"trbot/utils/handler_utils"

	"github.com/go-telegram/bot/models"
)

type Plugin_InlineCommandList struct {
	Command string
	Description string
}

// 需要返回一个列表，将由程序的分页函数来控制分页和输出
type Plugin_Inline struct {
	Command string
	Handler func(*handler_utils.SubHandlerOpts) []models.InlineQueryResult
	Description string
}

func AddInlineHandlerPlugins(InlineHandlerPlugins ...Plugin_Inline) int {
	if AllPugins.Inline == nil { AllPugins.Inline = []Plugin_Inline{} }
	var pluginCount int
	for _, originPlugin := range InlineHandlerPlugins {
		AllPugins.InlineCommandList = append(AllPugins.InlineCommandList, Plugin_InlineCommandList{Command: originPlugin.Command, Description: originPlugin.Description})
		AllPugins.Inline = append(AllPugins.Inline, originPlugin)
		pluginCount++
	}
	return pluginCount
}

// 完全由插件自行控制输出
type Plugin_InlineManual struct {
	Command string
	Handler func(*handler_utils.SubHandlerOpts)
	Description string
}

func AddInlineManualHandlerPlugins(InlineManualHandlerPlugins ...Plugin_InlineManual) int {
	if AllPugins.InlineManual == nil { AllPugins.InlineManual = []Plugin_InlineManual{} }
	var pluginCount int
	for _, originPlugin := range InlineManualHandlerPlugins {
		AllPugins.InlineCommandList = append(AllPugins.InlineCommandList, Plugin_InlineCommandList{Command: originPlugin.Command, Description: originPlugin.Description})
		AllPugins.InlineManual = append(AllPugins.InlineManual, originPlugin)
		pluginCount++
	}
	return pluginCount
}

type Plugin_InlinePrefix struct {
	PrefixCommand string
	Handler func(*handler_utils.SubHandlerOpts)
	Description string
}

func AddInlinePrefixHandlerPlugins(InlineManualHandlerPlugins ...Plugin_InlinePrefix) int {
	if AllPugins.InlinePrefix == nil { AllPugins.InlinePrefix = []Plugin_InlinePrefix{} }
	var pluginCount int
	for _, originPlugin := range InlineManualHandlerPlugins {
		AllPugins.InlineCommandList = append(AllPugins.InlineCommandList, Plugin_InlineCommandList{Command: originPlugin.PrefixCommand, Description: originPlugin.Description})
		AllPugins.InlinePrefix = append(AllPugins.InlinePrefix, originPlugin)
		pluginCount++
	}
	return pluginCount
}
