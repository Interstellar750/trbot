package plugin_utils

import (
	"trbot/utils/handler_utils"

	"github.com/go-telegram/bot/models"
)

type InlineHandlerAttr struct {
	IsHideInCommandList bool
	IsCantBeDefault     bool
	IsOnlyAllowAdmin    bool
}

type InlineCommandList struct {
	Command string
	Attr InlineHandlerAttr
	Description string
}

// 需要返回一个列表，将由程序的分页函数来控制分页和输出
type InlineHandler struct {
	Command string
	Attr InlineHandlerAttr
	Handler func(*handler_utils.SubHandlerOpts) []models.InlineQueryResult
	Description string
}

func AddInlineHandlerPlugins(InlineHandlerPlugins ...InlineHandler) int {
	if AllPlugins.InlineHandler == nil { AllPlugins.InlineHandler = []InlineHandler{} }
	var pluginCount int
	for _, originPlugin := range InlineHandlerPlugins {
		if originPlugin.Command == "" { continue }
		AllPlugins.InlineCommandList = append(AllPlugins.InlineCommandList, InlineCommandList{Command: originPlugin.Command, Attr: originPlugin.Attr, Description: originPlugin.Description})
		AllPlugins.InlineHandler = append(AllPlugins.InlineHandler, originPlugin)
		pluginCount++
	}
	return pluginCount
}

// 完全由插件自行控制输出
type InlineManualHandler struct {
	Command string
	Attr InlineHandlerAttr
	Handler func(*handler_utils.SubHandlerOpts)
	Description string
}

func AddInlineManualHandlerPlugins(InlineManualHandlerPlugins ...InlineManualHandler) int {
	if AllPlugins.InlineManualHandler == nil { AllPlugins.InlineManualHandler = []InlineManualHandler{} }
	var pluginCount int
	for _, originPlugin := range InlineManualHandlerPlugins {
		if originPlugin.Command == "" { continue }
		AllPlugins.InlineCommandList = append(AllPlugins.InlineCommandList, InlineCommandList{Command: originPlugin.Command, Attr: originPlugin.Attr, Description: originPlugin.Description})
		AllPlugins.InlineManualHandler = append(AllPlugins.InlineManualHandler, originPlugin)
		pluginCount++
	}
	return pluginCount
}

// 符合命令前缀，完全由插件自行控制输出
type InlinePrefixHandler struct {
	PrefixCommand string
	Attr InlineHandlerAttr
	Handler func(*handler_utils.SubHandlerOpts)
	Description string
}

func AddInlinePrefixHandlerPlugins(InlineManualHandlerPlugins ...InlinePrefixHandler) int {
	if AllPlugins.InlinePrefixHandler == nil { AllPlugins.InlinePrefixHandler = []InlinePrefixHandler{} }
	var pluginCount int
	for _, originPlugin := range InlineManualHandlerPlugins {
		if originPlugin.PrefixCommand == "" { continue }
		AllPlugins.InlineCommandList = append(AllPlugins.InlineCommandList, InlineCommandList{Command: originPlugin.PrefixCommand, Attr: originPlugin.Attr, Description: originPlugin.Description})
		AllPlugins.InlinePrefixHandler = append(AllPlugins.InlinePrefixHandler, originPlugin)
		pluginCount++
	}
	return pluginCount
}
