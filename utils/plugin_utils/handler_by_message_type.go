package plugin_utils

import (
	"fmt"
	"strings"
	"trbot/utils"
	"trbot/utils/err_template"
	"trbot/utils/flat_err"
	"trbot/utils/handler_params"
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
	/*
		Only update type handler can register, if there is more than
		one handler of the same type, the bot will send a keyboard
		to let the user choose which plugin they want to use.

		If that, the update type that trigger this function
		will be `update.CallbackQuery`, not `update.Message`.

		To register this type of plugin, make sure the
		function can handle both update types.

		You can try to simply copy the data to `update.Message`
		at the beginning of the plugin as follow:

		```
		if opts.Update.Message == nil && opts.Update.CallbackQuery != nil && strings.HasPrefix(opts.Update.CallbackQuery.Data, "HBMT_") && opts.Update.CallbackQuery.Message.Message != nil && opts.Update.CallbackQuery.Message.Message.ReplyToMessage != nil {
			opts.Update.Message = opts.Update.CallbackQuery.Message.Message.ReplyToMessage
		}
		````
	*/
	UpdateHandler func(*handler_params.Update) error
}

/*
	If more than one such plugin is registered
	or the `AllowAutoTrigger` flag is not `true`.

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

func SelectByMessageTypeHandlerCallback(opts *handler_params.Update) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("funcName", "SelectHandlerByMessageTypeHandlerCallback").
		Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
		Dict(utils.GetChatDict(&opts.Update.CallbackQuery.Message.Message.Chat)).
		Str("CallbackQuery", opts.Update.CallbackQuery.Data).
		Logger()

	var chatType, messageType, pluginName string
	var chatTypeMessageTypeAndPluginName  string

	var handlerErr flat_err.Errors

	if strings.HasPrefix(opts.Update.CallbackQuery.Data, "HBMT_") {
		chatTypeMessageTypeAndPluginName =  strings.TrimPrefix(opts.Update.CallbackQuery.Data, "HBMT_")
		chatTypeAndPluginNameList        := strings.Split(chatTypeMessageTypeAndPluginName, "_")
		if len(chatTypeAndPluginNameList) < 3 {
			err := fmt.Errorf("user selected callback query doesn't have enough fields")
			logger.Error().
				Err(err).
				Msg("Failed to trigger by message type handler")
			handlerErr.Addf("Failed to trigger by message type handler: %w", err)
		} else {
			chatType, messageType, pluginName = chatTypeAndPluginNameList[0], chatTypeAndPluginNameList[1], chatTypeAndPluginNameList[2]
			handler, isExist := AllPlugins.HandlerByMessageType[models.ChatType(chatType)][message_utils.MessageTypeList(messageType)][pluginName]
			if isExist {
				logger.Debug().
					Str("messageType", messageType).
					Str("pluginName", pluginName).
					Str("chatType", chatType).
					Msg("User selected a by message type handler")
				// if opts.Update.CallbackQuery.Message.Message.ReplyToMessage != nil {
				// 	opts.Update.Message = opts.Update.CallbackQuery.Message.Message.ReplyToMessage
				// }
				err := handler.UpdateHandler(opts)
				if err != nil {
					logger.Error().
						Err(err).
						Dict("handler", zerolog.Dict().
							Str("chatType", string(handler.ChatType)).
							Str("messageType", string(handler.MessageType)).
							Bool("allowAutoTrigger", handler.AllowAutoTrigger).
							Str("name", handler.PluginName),
						).
						Msg("Error in by message type handler")

					_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:    opts.Update.CallbackQuery.From.ID,
						Text:      fmt.Sprintf("调用 %s 功能时发生了一些错误\n<blockquote expandable>Failed to download sticker: %s</blockquote>", pluginName, err),
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "error in by message type handler notice").
							Msg(err_template.SendMessage)
						handlerErr.Addf("failed to send `error in by message type handler notice` message: %w", err)
					}
				} else {
					_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
						ChatID:    opts.Update.CallbackQuery.From.ID,
						MessageID: opts.Update.CallbackQuery.Message.Message.ID,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "select by message type handler keyboard").
							Msg(err_template.DeleteMessages)
						handlerErr.Addf("failed to delete `select by message type handler keyboard` message: %w", err)
					}
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
						Str("content", "this by message type handler is not exist").
						Msg(err_template.AnswerCallbackQuery)
					handlerErr.Addf("failed to send `this by message type handler is not exist` callback answer: %w", err)
				}
			}
		}
	}

	return handlerErr.Flat()
}
