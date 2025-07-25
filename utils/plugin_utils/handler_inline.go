package plugin_utils

import (
	"fmt"
	"trbot/database/db_struct"
	"trbot/utils/configs"
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot/models"
)

type InlineHandlerAttr struct {
	IsHideInCommandList bool
	IsCantBeDefault     bool
	IsOnlyAllowAdmin    bool
}

type InlineCommandList struct {
	Command     string
	Attr        InlineHandlerAttr
	Description string
}

// 需要返回一个列表，将由程序的分页函数来控制分页和输出
type InlineHandler struct {
	Command       string
	Description   string
	Attr          InlineHandlerAttr
	InlineHandler func(*handler_params.InlineQuery) []models.InlineQueryResult
}

func AddInlineHandlerHandlers(handlers ...InlineHandler) int {
	if AllPlugins.InlineHandler == nil { AllPlugins.InlineHandler = []InlineHandler{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.Command == "" { continue }
		AllPlugins.InlineCommandList = append(AllPlugins.InlineCommandList, InlineCommandList{Command: handler.Command, Attr: handler.Attr, Description: handler.Description})
		AllPlugins.InlineHandler = append(AllPlugins.InlineHandler, handler)
		handlerCount++
	}
	return handlerCount
}

// 完全由插件自行控制输出
type InlineManualHandler struct {
	Command     string
	Description string
	Attr        InlineHandlerAttr
	InlineHandler func(*handler_params.InlineQuery) error
}

func AddInlineManualHandlerHandlers(handlers ...InlineManualHandler) int {
	if AllPlugins.InlineManualHandler == nil { AllPlugins.InlineManualHandler = []InlineManualHandler{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.Command == "" { continue }
		AllPlugins.InlineCommandList = append(AllPlugins.InlineCommandList, InlineCommandList{Command: handler.Command, Attr: handler.Attr, Description: handler.Description})
		AllPlugins.InlineManualHandler = append(AllPlugins.InlineManualHandler, handler)
		handlerCount++
	}
	return handlerCount
}

// 符合命令前缀，完全由插件自行控制输出
type InlinePrefixHandler struct {
	PrefixCommand string
	Description   string
	Attr          InlineHandlerAttr
	InlineHandler func(*handler_params.InlineQuery) error
}

func AddInlinePrefixHandlerPlugins(handlers ...InlinePrefixHandler) int {
	if AllPlugins.InlinePrefixHandler == nil { AllPlugins.InlinePrefixHandler = []InlinePrefixHandler{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.PrefixCommand == "" { continue }
		AllPlugins.InlineCommandList = append(AllPlugins.InlineCommandList, InlineCommandList{Command: handler.PrefixCommand, Attr: handler.Attr, Description: handler.Description})
		AllPlugins.InlinePrefixHandler = append(AllPlugins.InlinePrefixHandler, handler)
		handlerCount++
	}
	return handlerCount
}

// 构建一个用于选择 Inline 模式下默认命令的按钮键盘
func BuildDefaultInlineCommandSelectKeyboard(chatInfo *db_struct.ChatInfo) models.ReplyMarkup {
	var inlinePlugins [][]models.InlineKeyboardButton
	for _, v := range AllPlugins.InlineCommandList {
		if v.Attr.IsCantBeDefault {
			continue
		}
		if chatInfo.DefaultInlinePlugin == v.Command {
			inlinePlugins = append(inlinePlugins, []models.InlineKeyboardButton{{
				Text: fmt.Sprintf("✅ [ %s%s ] - %s", configs.BotConfig.InlineSubCommandSymbol, v.Command, v.Description),
				CallbackData: "inline_default_" + v.Command,
			}})
		} else {
			inlinePlugins = append(inlinePlugins, []models.InlineKeyboardButton{{
				Text: fmt.Sprintf("[ %s%s ] - %s", configs.BotConfig.InlineSubCommandSymbol, v.Command, v.Description),
				CallbackData: "inline_default_" + v.Command,
			}})
		}
	}

	inlinePlugins = append(inlinePlugins, []models.InlineKeyboardButton{
		{
			Text:         "取消默认命令",
			CallbackData: "inline_default_none",
		},
		{
			Text:                         "浏览 inline 命令菜单",
			SwitchInlineQueryCurrentChat: configs.BotConfig.InlineSubCommandSymbol,
		},
	})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: inlinePlugins,
	}

	return kb
}
