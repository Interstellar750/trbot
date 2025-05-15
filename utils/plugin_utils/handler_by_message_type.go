package plugin_utils

import (
	"trbot/utils/handler_structs"
	"trbot/utils/type_utils"

	"github.com/go-telegram/bot/models"
)

type HandlerByMessageTypeFor map[models.ChatType]HandlerByMessageTypeList

type HandlerByMessageTypeList map[type_utils.MessageTypeList]HandlerByMessageTypeFunctions

type HandlerByMessageTypeFunctions map[string]HandlerByMessageType

func (funcs HandlerByMessageTypeFunctions) BuildSelectKeyboard() models.ReplyMarkup {
	var msgTypeItems [][]models.InlineKeyboardButton

	for name := range funcs {
		msgTypeItems = append(msgTypeItems, []models.InlineKeyboardButton{{
			Text: name,
			CallbackData: "handler_by_message_type_" + name,
		}})
	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: msgTypeItems,
	}
}

type HandlerByMessageType struct {
	Name        string
	ChatType    models.ChatType
	MessageType type_utils.MessageTypeList
	Handler     func(*handler_structs.SubHandlerParams)
}

func AddHandlerByMessageTypePlugin(plugins ...HandlerByMessageType) {
	if AllPlugins.HandlerByMessageTypeFor == nil {
		AllPlugins.HandlerByMessageTypeFor = HandlerByMessageTypeFor{}
	}

	for _, plugin := range plugins {
		if AllPlugins.HandlerByMessageTypeFor[plugin.ChatType] == nil {
			AllPlugins.HandlerByMessageTypeFor[plugin.ChatType] = HandlerByMessageTypeList{}
		}

		if AllPlugins.HandlerByMessageTypeFor[plugin.ChatType][plugin.MessageType] == nil {
			AllPlugins.HandlerByMessageTypeFor[plugin.ChatType][plugin.MessageType] = HandlerByMessageTypeFunctions{}
		}
		_, isExist := AllPlugins.HandlerByMessageTypeFor[plugin.ChatType][plugin.MessageType][plugin.Name]

		if !isExist {
			AllPlugins.HandlerByMessageTypeFor[plugin.ChatType][plugin.MessageType][plugin.Name] = plugin
		}
	}

	
}

// func SelectHandlerByMessageTypeHandlerCallback(opts *handler_structs.SubHandlerParams) {
// 	opts.Update.CallbackQuery.
// 	if AllPlugins.HandlerByMessageTypeFor[opts.ChatType][opts.MessageType][opts.Name] != nil {
// 		AllPlugins.HandlerByMessageTypeFor[opts.ChatType][opts.MessageType][opts.Name].Handler(opts)
// 	}
// }
