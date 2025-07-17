package plugin_utils

import (
	"fmt"
	"strings"
	"trbot/utils"
	"trbot/utils/flate"
	"trbot/utils/handler_params"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HandlerByMessageTypes map[models.ChatType]map[message_utils.MessageTypeList]map[int64]map[string]ByMessageTypeHandler

func (funcs HandlerByMessageTypes) BuildSelectKeyboard(chatType models.ChatType, msgType message_utils.MessageTypeList, chatID int64) ([][]models.InlineKeyboardButton, int) {
	var msgTypeItems [][]models.InlineKeyboardButton
	var handlerCount int

	// handler for some chat id
	for _, handler := range funcs[chatType][msgType][chatID] {
		msgTypeItems = append(msgTypeItems, []models.InlineKeyboardButton{{
			Text: handler.PluginName,
			CallbackData: "HBMT_" + handler.PluginName,
		}})
		handlerCount++
	}

	// handler for any chat id
	for _, handler := range funcs[chatType][msgType][0] {
		msgTypeItems = append(msgTypeItems, []models.InlineKeyboardButton{{
			Text: handler.PluginName,
			CallbackData: "HBMT_" + handler.PluginName,
		}})
		handlerCount++
	}

	return msgTypeItems, handlerCount
}

type ByMessageTypeHandler struct {
	PluginName       string
	ChatType         models.ChatType
	ForChatID        int64           // 0 for all
	MessageType      message_utils.MessageTypeList
	AllowAutoTrigger bool // Allow auto trigger when there is only one handler of the same type
	/*
		Only update type handler can register, if there is more than
		one handler of the same type, the bot will send a keyboard
		to let the user choose which handler they want to use.

		If that, the update type that trigger this function
		will be `update.CallbackQuery`, not `update.Message`.

		To register this type of handler, make sure the
		function can handle both update types.

		You can try to simply copy the message data from
		`update.CallbackQuery.Message.Message.ReplyToMessage`
		to `update.Message` at the beginning of the handler as follow:

		```
		if opts.Update.Message == nil && opts.Update.CallbackQuery != nil && strings.HasPrefix(opts.Update.CallbackQuery.Data, "HBMT_") && opts.Update.CallbackQuery.Message.Message != nil && opts.Update.CallbackQuery.Message.Message.ReplyToMessage != nil {
			opts.Update.Message = opts.Update.CallbackQuery.Message.Message.ReplyToMessage
		}
		````
	*/
	UpdateHandler func(*handler_params.Update) error // with full access to `update`.
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
func AddHandlerByMessageTypeHandlers(handlers ...ByMessageTypeHandler) int {
	if AllPlugins.HandlerByMessageType == nil { AllPlugins.HandlerByMessageType = HandlerByMessageTypes{} }

	var handlerCount int
	for _, handler := range handlers {
		if AllPlugins.HandlerByMessageType[handler.ChatType] == nil { AllPlugins.HandlerByMessageType[handler.ChatType] = map[message_utils.MessageTypeList]map[int64]map[string]ByMessageTypeHandler{} }
		if AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType] == nil { AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType] = map[int64]map[string]ByMessageTypeHandler{} }
		if AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType][handler.ForChatID] == nil { AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType][handler.ForChatID] = map[string]ByMessageTypeHandler{} }

		_, isExist := AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType][handler.ForChatID][handler.PluginName]
		if isExist {
			log.Error().
				Str("funcName", "AddHandlerByMessageTypePlugins").
				Str("chatType", string(handler.ChatType)).
				Str("messageType", string(handler.MessageType)).
				Int64("forChatID", handler.ForChatID).
				Str("name", handler.PluginName).
				Msgf("Duplicate plugin exists, registration skipped")
		} else {
			AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType][handler.ForChatID][handler.PluginName] = handler
			handlerCount++
		}
	}

	return handlerCount
}

func RemoveHandlerByMessageTypeHandler(chatType models.ChatType, messageType message_utils.MessageTypeList, chatID int64, handlerName string) {
	if AllPlugins.HandlerByMessageType == nil { return }

	// _, isExist := AllPlugins.HandlerByMessageType[chatType][messageType][chatID][handlerName]
	// if isExist {
		delete(AllPlugins.HandlerByMessageType[chatType][messageType][chatID], handlerName)
	// }
}

func RunByMessageTypeHandlers(params *handler_params.Update) (bool, string, error) {
	var isProcessed bool
	var handlerErr  flate.MultErr

	msgType := message_utils.GetMessageType(params.Update.Message).AsValue()
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "RunByChatIDHandlers").
		Str("messageType", string(msgType)).
		Logger()

	var needBuildSelectKeyboard bool = true

	if AllPlugins.HandlerByMessageType[params.Update.Message.Chat.Type] != nil {
		handlerCount    := len(AllPlugins.HandlerByMessageType[params.Update.Message.Chat.Type][msgType][params.Update.Message.Chat.ID])
		anyHandlerCount := len(AllPlugins.HandlerByMessageType[params.Update.Message.Chat.Type][msgType][0])
		if handlerCount + anyHandlerCount > 0 {
			if handlerCount + anyHandlerCount == 1 {
				var handlerID int64 = 0
				if handlerCount == 1 {
					handlerID = params.Update.Message.Chat.ID
				}
				// 虽然是遍历，但实际上只能遍历一次
				for name, handler := range AllPlugins.HandlerByMessageType[params.Update.Message.Chat.Type][msgType][handlerID] {
					if handler.AllowAutoTrigger {
						// 允许自动触发的 handler
						logger.Info().
							Str("handlerName", name).
							Int64("forChatID", handler.ForChatID).
							Msg("Hit by message type handler")

						if handler.UpdateHandler == nil {
							logger.Warn().Msg("Hit by message type handler, but this handler function is nil, skip")
							continue
						}
						err := handler.UpdateHandler(params)
						if err != nil {
							logger.Error().
								Err(err).
								Str("handlerName", name).
								Int64("forChatID", handler.ForChatID).
								Msg("Error in by message type handler")
						}
						isProcessed = true
						needBuildSelectKeyboard = false
					}
				}
			}

			if needBuildSelectKeyboard {
				handlerKeyboard, count := AllPlugins.HandlerByMessageType.BuildSelectKeyboard(params.Update.Message.Chat.Type, msgType, params.Update.Message.Chat.ID)
				if len(handlerKeyboard) > 0 {
					isProcessed = true
					// 多个 handler 自动回复一条带按钮的消息让用户手动操作
					_, err := params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
						ChatID:    params.Update.Message.Chat.ID,
						Text:      fmt.Sprintf("请选择一个 [ %s ] 类型消息的功能", msgType),
						ReplyParameters: &models.ReplyParameters{ MessageID: params.Update.Message.ID },
						ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: handlerKeyboard },
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int("handlerCount", count).
							Str("content", "handler by message type select keyboard").
							Msg(flate.SendMessage.Str())
					}
				}
			}
		}
	}

	return isProcessed, string(msgType), handlerErr.Flat()
}

func SelectByMessageTypeHandlerCallback(opts *handler_params.Update) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("funcName", "SelectHandlerByMessageTypeHandlerCallback").
		Dict(utils.GetUserDict(&opts.Update.CallbackQuery.From)).
		Dict(utils.GetChatDict(&opts.Update.CallbackQuery.Message.Message.Chat)).
		Str("callbackQuery", opts.Update.CallbackQuery.Data).
		Logger()

	var handlerErr flate.MultErr

	if strings.HasPrefix(opts.Update.CallbackQuery.Data, "HBMT_") {
		pluginName := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "HBMT_")
		if pluginName == "" {
			err := fmt.Errorf("user selected callback query doesn't have enough fields")
			logger.Error().
				Err(err).
				Msg("Failed to trigger by message type handler")
			handlerErr.Addf("Failed to trigger by message type handler: %w", err)
		} else {
			messageType := message_utils.GetMessageType(opts.Update.CallbackQuery.Message.Message.ReplyToMessage).AsValue()
			handler, isExist := AllPlugins.HandlerByMessageType[opts.Update.CallbackQuery.Message.Message.Chat.Type][messageType][opts.Update.CallbackQuery.Message.Message.Chat.ID][pluginName]
			if isExist {
				logger.Debug().
					Str("chatType", string(opts.Update.CallbackQuery.Message.Message.Chat.Type)).
					Str("messageType", string(messageType)).
					Int64("chatID", opts.Update.CallbackQuery.Message.Message.Chat.ID).
					Str("pluginName", pluginName).
					Msg("User selected a by message type handler")
				if handler.UpdateHandler == nil {
					logger.Warn().Msg("Hit by message type handler, but this handler function is nil, ignore")
					return handlerErr.Addf("hit by message type handler [%s], but this handler function is nil, ignore", pluginName).Flat()
				}
				err := handler.UpdateHandler(opts)
				if err != nil {
					logger.Error().
						Err(err).
						Dict("handler", zerolog.Dict().
							Str("chatType", string(handler.ChatType)).
							Str("messageType", string(handler.MessageType)).
							Int64("forChatID", handler.ForChatID).
							Bool("allowAutoTrigger", handler.AllowAutoTrigger).
							Str("name", handler.PluginName),
						).
						Msg("Error in by message type handler")
					handlerErr.Addf("error in by message type handler: %w", err)

					_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:    opts.Update.CallbackQuery.From.ID,
						Text:      fmt.Sprintf("调用 %s 功能时发生了一些错误\n<blockquote expandable>%s</blockquote>", pluginName, err),
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "error in by message type handler notice").
							Msg(flate.SendMessage.Str())
						handlerErr.Addf(flate.SendMessage.Fmt(), "error in by message type handler notice", err)
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
							Msg(flate.DeleteMessages.Str())
						handlerErr.Addf(flate.DeleteMessage.Fmt(), "select by message type handler keyboard", err)
					}
				}
			} else {
				handler, isExist := AllPlugins.HandlerByMessageType[opts.Update.CallbackQuery.Message.Message.Chat.Type][messageType][0][pluginName]
				if isExist {
					logger.Debug().
						Str("chatType", string(opts.Update.CallbackQuery.Message.Message.Chat.Type)).
						Str("messageType", string(messageType)).
						Int64("chatID", opts.Update.CallbackQuery.Message.Message.Chat.ID).
						Str("pluginName", pluginName).
						Msg("User selected a by message type handler")
					if handler.UpdateHandler == nil {
						logger.Warn().Msg("Hit by message type handler, but this handler function is nil, ignore")
						return handlerErr.Addf("hit by message type handler [%s], but this handler function is nil, ignore", pluginName).Flat()
					}
					err := handler.UpdateHandler(opts)
					if err != nil {
						logger.Error().
							Err(err).
							Dict("handler", zerolog.Dict().
								Str("chatType", string(handler.ChatType)).
								Str("messageType", string(handler.MessageType)).
								Int64("forChatID", handler.ForChatID).
								Bool("allowAutoTrigger", handler.AllowAutoTrigger).
								Str("name", handler.PluginName),
							).
							Msg("Error in by message type handler")
						handlerErr.Addf("error in by message type handler: %w", err)

						_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID:    opts.Update.CallbackQuery.From.ID,
							Text:      fmt.Sprintf("调用 %s 功能时发生了一些错误\n<blockquote expandable>%s</blockquote>", pluginName, err),
							ParseMode: models.ParseModeHTML,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Str("content", "error in by message type handler notice").
								Msg(flate.SendMessage.Str())
							handlerErr.Addf(flate.SendMessage.Fmt(), "error in by message type handler notice", err)
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
								Msg(flate.DeleteMessages.Str())
							handlerErr.Addf(flate.DeleteMessage.Fmt(), "select by message type handler keyboard", err)
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
							Msg(flate.AnswerCallbackQuery.Str())
						handlerErr.Addf(flate.AnswerCallbackQuery.Fmt(), "this by message type handler is not exist", err)
					}
				}
			}
		}
	}

	return handlerErr.Flat()
}
