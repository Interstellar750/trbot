package plugin_utils

import (
	"fmt"
	"strings"
	"trbot/utils"
	"trbot/utils/handler_structs"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
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
	MessageType      message_utils.MessageTypeList
	AllowAutoTrigger bool // Allow auto trigger when there is only one handler of the same type
	Handler          func(*handler_structs.SubHandlerParams) error
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
	if AllPlugins.HandlerByMessageType == nil { AllPlugins.HandlerByMessageType = map[models.ChatType]map[message_utils.MessageTypeList]HandlerByMessageTypeFunctions{} }

	var pluginCount int
	for _, plugin := range plugins {
		if AllPlugins.HandlerByMessageType[plugin.ChatType] == nil { AllPlugins.HandlerByMessageType[plugin.ChatType] = map[message_utils.MessageTypeList]HandlerByMessageTypeFunctions{} }
		if AllPlugins.HandlerByMessageType[plugin.ChatType][plugin.MessageType] == nil { AllPlugins.HandlerByMessageType[plugin.ChatType][plugin.MessageType] = HandlerByMessageTypeFunctions{} }

		_, isExist := AllPlugins.HandlerByMessageType[plugin.ChatType][plugin.MessageType][plugin.PluginName]
		if !isExist {
			AllPlugins.HandlerByMessageType[plugin.ChatType][plugin.MessageType][plugin.PluginName] = plugin
			pluginCount++
		}
	}

	return pluginCount
}

func RemoveHandlerByMessageTypePlugin(chatType models.ChatType, messageType message_utils.MessageTypeList, pluginName string) {
	if AllPlugins.HandlerByMessageType == nil { return }

	_, isExist := AllPlugins.HandlerByMessageType[chatType][messageType][pluginName]
	if isExist {
		delete(AllPlugins.HandlerByMessageType[chatType][messageType], pluginName)
	}
}

func SelectHandlerByMessageTypeHandlerCallback(opts *handler_structs.SubHandlerParams) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("funcName", "SelectHandlerByMessageTypeHandlerCallback").
		Logger()

	var chatType, messageType, pluginName string
	var chatTypeMessageTypeAndPluginName  string

	if strings.HasPrefix(opts.Update.CallbackQuery.Data, "HBMT_") {
		chatTypeMessageTypeAndPluginName =  strings.TrimPrefix(opts.Update.CallbackQuery.Data, "HBMT_")
		chatTypeAndPluginNameList        := strings.Split(chatTypeMessageTypeAndPluginName, "_")
		if len(chatTypeAndPluginNameList) < 3 {
			err := fmt.Errorf("no enough fields")
			logger.Error().
				Err(err).
				Dict("user", zerolog.Dict().
					Str("name", utils.ShowUserName(&opts.Update.CallbackQuery.From)).
					Str("username", opts.Update.CallbackQuery.From.Username).
					Int64("ID", opts.Update.CallbackQuery.From.ID),
				).
				Str("CallbackQuery", opts.Update.CallbackQuery.Data).
				Msg("User selected callback query doesn't have enough fields")
			return err
		}
		chatType, messageType, pluginName = chatTypeAndPluginNameList[0], chatTypeAndPluginNameList[1], chatTypeAndPluginNameList[2]
		handler, isExist := AllPlugins.HandlerByMessageType[models.ChatType(chatType)][message_utils.MessageTypeList(messageType)][pluginName]
		if isExist {
			logger.Debug().
				Dict("user", zerolog.Dict().
					Str("name", utils.ShowUserName(&opts.Update.CallbackQuery.From)).
					Str("username", opts.Update.CallbackQuery.From.Username).
					Int64("ID", opts.Update.CallbackQuery.From.ID),
				).
				Str("messageType", messageType).
				Str("pluginName", pluginName).
				Str("chatType", chatType).
				Msg("User selected a handler by message")
			// if opts.Update.CallbackQuery.Message.Message.ReplyToMessage != nil {
			// 	opts.Update.Message = opts.Update.CallbackQuery.Message.Message.ReplyToMessage
			// }
			err := handler.Handler(opts)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
					Str("handerChatType", string(handler.ChatType)).
					Str("handlerMessageType", string(handler.MessageType)).
					Bool("allowAutoTrigger", handler.AllowAutoTrigger).
					Str("handlerName", handler.PluginName).
					Msg("Error in handler by message type")
				return err
			}
			_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
				ChatID:    opts.Update.CallbackQuery.From.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
					Dict(utils.GetChatDict(&opts.Update.CallbackQuery.Message.Message.Chat)).
				Msg("Failed to delete `select handler by message type keyboard` message")
				return err
			}
		} else {
			_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: opts.Update.CallbackQuery.ID,
				Text: fmt.Sprintf("此功能 [ %s ] 不可用，可能是管理员已经移除了这个功能", pluginName),
				ShowAlert: true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
					Dict(utils.GetChatDict(&opts.Update.CallbackQuery.Message.Message.Chat)).
				Msg("Failed to send `handler by message type is not exist` callback answer")
				return err
			}
		}
	}
	logger.Warn().
		Str("callbackQuery", opts.Update.CallbackQuery.Data).
		Msg("Receive an invalid callback query, it should start with `HBMT_`")
	return nil
}
