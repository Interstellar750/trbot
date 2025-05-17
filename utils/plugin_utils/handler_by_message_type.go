package plugin_utils

import (
	"fmt"
	"log"
	"strings"
	"trbot/utils/consts"
	"trbot/utils/handler_structs"
	"trbot/utils/type_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type HandlerByMessageTypeFunctions map[string]HandlerByMessageType

func (funcs HandlerByMessageTypeFunctions) BuildSelectKeyboard() models.ReplyMarkup {
	var msgTypeItems [][]models.InlineKeyboardButton

	for name := range funcs {
		msgTypeItems = append(msgTypeItems, []models.InlineKeyboardButton{{
			Text: name,
			CallbackData: fmt.Sprintf("HBMT_%s_%s_%s", funcs[name].ChatType, funcs[name].MessageType, name),
		}})
	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: msgTypeItems,
	}
}

type HandlerByMessageType struct {
	PluginName       string
	ChatType         models.ChatType
	MessageType      type_utils.MessageTypeList
	AllowAutoTrigger bool // Allow auto trigger when there is only one handler of the same type
	Handler          func(*handler_structs.SubHandlerParams)
}

/*
	If more than one such plugin is registered
	or the `AllowAutoTrigger`` flag is not `true`
	
	The bot will reply to the message that triggered
	this plugin and send a keyboard to let the
	user choose which plugin they want to use.

	In this case, the data that the plugin needs
	to process will change from `update.Message` to
	in `opts.Update.CallbackQuery.Message.Message.ReplyToMessage`.
	
	But I'm not sure whether this field will be empty,
	so need to manually judge it in the plugin.

	You can try to simply copy the data to `update.Message`
	at the beginning of the plugin as follow:

	```
	if opts.Update.Message == nil && opts.Update.CallbackQuery != nil && strings.HasPrefix(opts.Update.CallbackQuery.Data, "HBMT_") && opts.Update.CallbackQuery.Message.Message != nil && opts.Update.CallbackQuery.Message.Message.ReplyToMessage != nil {
		opts.Update.Message = opts.Update.CallbackQuery.Message.Message.ReplyToMessage
	}
	```
*/
func AddHandlerByMessageTypePlugins(plugins ...HandlerByMessageType) int {
	if AllPlugins.HandlerByMessageType == nil { AllPlugins.HandlerByMessageType = map[models.ChatType]map[type_utils.MessageTypeList]HandlerByMessageTypeFunctions{} }

	var pluginCount int
	for _, plugin := range plugins {
		if AllPlugins.HandlerByMessageType[plugin.ChatType] == nil { AllPlugins.HandlerByMessageType[plugin.ChatType] = map[type_utils.MessageTypeList]HandlerByMessageTypeFunctions{} }
		if AllPlugins.HandlerByMessageType[plugin.ChatType][plugin.MessageType] == nil { AllPlugins.HandlerByMessageType[plugin.ChatType][plugin.MessageType] = HandlerByMessageTypeFunctions{} }

		_, isExist := AllPlugins.HandlerByMessageType[plugin.ChatType][plugin.MessageType][plugin.PluginName]
		if !isExist {
			AllPlugins.HandlerByMessageType[plugin.ChatType][plugin.MessageType][plugin.PluginName] = plugin
			pluginCount++
		}
	}

	return pluginCount
}

func RemoveHandlerByMessageTypePlugin(chatType models.ChatType, messageType type_utils.MessageTypeList, pluginName string) {
	if AllPlugins.HandlerByMessageType == nil { return }

	_, isExist := AllPlugins.HandlerByMessageType[chatType][messageType][pluginName]
	if isExist {
		delete(AllPlugins.HandlerByMessageType[chatType][messageType], pluginName)
	}
}

func SelectHandlerByMessageTypeHandlerCallback(opts *handler_structs.SubHandlerParams) {
	var chatType, messageType, pluginName string
	var chatTypeMessageTypeAndPluginName string
	if strings.HasPrefix(opts.Update.CallbackQuery.Data, "HBMT_") {
		chatTypeMessageTypeAndPluginName = strings.TrimPrefix(opts.Update.CallbackQuery.Data, "HBMT_")
		chatTypeAndPluginNameList := strings.Split(chatTypeMessageTypeAndPluginName, "_")
		if len(chatTypeAndPluginNameList) < 3 { return }
		chatType, messageType, pluginName = chatTypeAndPluginNameList[0], chatTypeAndPluginNameList[1], chatTypeAndPluginNameList[2]
		handler, isExist := AllPlugins.HandlerByMessageType[models.ChatType(chatType)][type_utils.MessageTypeList(messageType)][pluginName]
		if isExist {
			if consts.IsDebugMode {
				log.Printf("select handler by message type [%s] plugin [%s] for chat type [%s]", messageType, pluginName, chatType)
			}
			// if opts.Update.CallbackQuery.Message.Message.ReplyToMessage != nil {
			// 	opts.Update.Message = opts.Update.CallbackQuery.Message.Message.ReplyToMessage
			// }
			handler.Handler(opts)
			opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
				ChatID:    opts.Update.CallbackQuery.From.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			})
		} else {
			opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: opts.Update.CallbackQuery.ID,
				Text: fmt.Sprintf("此功能 [ %s ] 不可用，可能是管理员已经移除了这个功能", pluginName),
				ShowAlert: true,
			})
		}
	}
}
