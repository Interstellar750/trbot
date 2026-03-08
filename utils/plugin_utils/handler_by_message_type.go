package plugin_utils

import (
	"fmt"
	"strings"

	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/flaterr"
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HandlerByMessageTypes map[models.ChatType]map[message_utils.Type]map[int64]map[string]ByMessageTypeHandler

func (funcs HandlerByMessageTypes) BuildSelectKeyboard(chatType models.ChatType, msgType message_utils.Type, chatID int64) ([][]models.InlineKeyboardButton, int) {
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
	MessageType      message_utils.Type
	AllowAutoTrigger bool // Allow auto trigger when there is only one handler of the same type
	/*
		Only Message type handler can register, if there is more than
		one handler of the same type, the bot will send a keyboard
		to let the user choose which handler they want to use.

		In this case, the `params.Message` in the handler parameters is
		actually `update.CallbackQuery.Message.Message.ReplyToMessage`,
		you don't need to handle the `CallbackQuery``, but note that
		the `Message.From`` field in this message will always be bot.
	*/
	MessageHandler func(*handler_params.Message) error
}

/*
	If more than one such plugin is registered
	or the `AllowAutoTrigger` flag is not `true`.

	The bot will reply to the message that triggered
	this plugin and send a keyboard to let the
	user choose which plugin they want to use.

	In this case, the `params.Message` in the handler parameters is
	actually `update.CallbackQuery.Message.Message.ReplyToMessage`,
	you don't need to handle the `CallbackQuery``, but note that
	the `Message.From`` field in this message will always be bot.
*/
func AddHandlerByMessageTypeHandlers(handlers ...ByMessageTypeHandler) int {
	if AllPlugins.HandlerByMessageType == nil { AllPlugins.HandlerByMessageType = HandlerByMessageTypes{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.PluginName == "" || handler.ChatType == "" || handler.MessageType == "" || handler.MessageHandler == nil {
			log.Error().
				Str(utils.GetCurrentFuncName()).
				Str("pluginName", handler.PluginName).
				Str("chatType", string(handler.ChatType)).
				Str("messageType", string(handler.MessageType)).
				Int64("forChatID", handler.ForChatID).
				Msgf("Not enough parameters, skip this handler")
			continue
		}
		if AllPlugins.HandlerByMessageType[handler.ChatType] == nil { AllPlugins.HandlerByMessageType[handler.ChatType] = map[message_utils.Type]map[int64]map[string]ByMessageTypeHandler{} }
		if AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType] == nil { AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType] = map[int64]map[string]ByMessageTypeHandler{} }
		if AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType][handler.ForChatID] == nil { AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType][handler.ForChatID] = map[string]ByMessageTypeHandler{} }

		_, isExist := AllPlugins.HandlerByMessageType[handler.ChatType][handler.MessageType][handler.ForChatID][handler.PluginName]
		if isExist {
			log.Error().
				Str(utils.GetCurrentFuncName()).
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

func RemoveHandlerByMessageTypeHandler(chatType models.ChatType, messageType message_utils.Type, chatID int64, handlerName string) {
	if AllPlugins.HandlerByMessageType == nil { return }
	delete(AllPlugins.HandlerByMessageType[chatType][messageType][chatID], handlerName)
}

func RunByMessageTypeHandlers(params *handler_params.Message) error {
	if AllPlugins.HandlerByMessageType == nil { return nil }
	var handlerErr  flaterr.MultErr

	msgType := message_utils.GetMessageType(params.Message).AsType()
	logger := zerolog.Ctx(params.Ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Str("messageType", msgType.Str()).
		Str("chatType", string(params.Message.Chat.Type)).
		Int64("chatID", params.Message.Chat.ID).
		Logger()

	var needBuildSelectKeyboard bool = true

	if AllPlugins.HandlerByMessageType[params.Message.Chat.Type] != nil {
		handlerCount    := len(AllPlugins.HandlerByMessageType[params.Message.Chat.Type][msgType][params.Message.Chat.ID])
		anyHandlerCount := len(AllPlugins.HandlerByMessageType[params.Message.Chat.Type][msgType][0])
		if handlerCount + anyHandlerCount > 0 {
			if handlerCount + anyHandlerCount == 1 {
				// 总共只有一个 handler
				var targetChatID int64 = 0
				if handlerCount == 1 {
					// 如果这唯一一个 handler 是针对此 chat 的，就将 targetChatID 设置为此 chat ID
					// 否则就会触发任意 chat ID 可用的 handler
					targetChatID = params.Message.Chat.ID
				}
				// 虽然是遍历，但实际上只能遍历一次
				for name, handler := range AllPlugins.HandlerByMessageType[params.Message.Chat.Type][msgType][targetChatID] {
					if handler.AllowAutoTrigger {
						// 允许自动触发的 handler
						slogger := logger.With().
							Str("handlerName", name).
							Int64("forChatID", handler.ForChatID).
							Logger()

						if handler.MessageHandler != nil {
							slogger.Info().Msg("Hit by message type handler")
							err := handler.MessageHandler(params)
							if err != nil {
								slogger.Error().
									Err(err).
									Msg("Error in by message type handler")
								handlerErr.Addf("Error in by message type handler [%s]: %w", name, err)
							}
							needBuildSelectKeyboard = false
						} else {
							slogger.Warn().Msg("Hit by message type handler, but this handler function is nil, skip")
							handlerErr.Addf("hit by message type handler [%s], but this handler function is nil, skip", name)
						}

					}
				}
			}

			// handler 多于一个或许没有允许自动触发的 handler，就会发送选择键盘
			if needBuildSelectKeyboard {
				handlerKeyboard, count := AllPlugins.HandlerByMessageType.BuildSelectKeyboard(params.Message.Chat.Type, msgType, params.Message.Chat.ID)
				if len(handlerKeyboard) > 0 {
					slogger := logger.With().
						Int("handlerCount", count).
						Logger()

					slogger.Info().Msg("Send a handler by message type select keyboard to user")
					// 多个 handler 自动回复一条带按钮的消息让用户手动操作
					_, err := params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
						ChatID:    params.Message.Chat.ID,
						Text:      fmt.Sprintf("请选择一个 [ %s ] 类型消息的功能", msgType),
						ReplyParameters: &models.ReplyParameters{ MessageID: params.Message.ID },
						ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: handlerKeyboard },
					})
					if err != nil {
						slogger.Error().
							Err(err).
							Str("content", "handler by message type select keyboard").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "handler by message type select keyboard", err)
					}
				}
			}
		}
	}

	return handlerErr.Flat()
}

func SelectByMessageTypeHandlerCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Dict(utils.GetChatDict(&opts.CallbackQuery.Message.Message.Chat)).
		Str("callbackQueryData", opts.CallbackQuery.Data).
		Logger()

	var handlerErr flaterr.MultErr

	if strings.HasPrefix(opts.CallbackQuery.Data, "HBMT_") {
		pluginName := strings.TrimPrefix(opts.CallbackQuery.Data, "HBMT_")
		if pluginName == "" {
			err := fmt.Errorf("user selected callback query doesn't have enough fields")
			logger.Error().
				Err(err).
				Msg("Failed to trigger by message type handler")
			handlerErr.Addf("Failed to trigger by message type handler: %w", err)
		} else {
			messageType := message_utils.GetMessageType(opts.CallbackQuery.Message.Message.ReplyToMessage).AsType()
			handler, isExist := AllPlugins.HandlerByMessageType[opts.CallbackQuery.Message.Message.Chat.Type][messageType][opts.CallbackQuery.Message.Message.Chat.ID][pluginName]
			if isExist {
				logger.Debug().
					Str("chatType", string(opts.CallbackQuery.Message.Message.Chat.Type)).
					Str("messageType", messageType.Str()).
					Int64("chatID", opts.CallbackQuery.Message.Message.Chat.ID).
					Str("pluginName", pluginName).
					Msg("User selected a by message type handler")
				if handler.MessageHandler == nil {
					logger.Warn().Msg("Hit by message type handler, but this handler function is nil, ignore")
					return handlerErr.Addf("hit by message type handler [%s], but this handler function is nil, ignore", pluginName).Flat()
				}
				err := handler.MessageHandler(&handler_params.Message{
					Ctx:      opts.Ctx,
					Thebot:   opts.Thebot,
					Message:  opts.CallbackQuery.Message.Message.ReplyToMessage,
					ChatInfo: opts.ChatInfo,
					Fields:   strings.Fields(opts.CallbackQuery.Message.Message.Text),
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict("handler", zerolog.Dict().
							Str("chatType", string(handler.ChatType)).
							Str("messageType", handler.MessageType.Str()).
							Int64("forChatID", handler.ForChatID).
							Bool("allowAutoTrigger", handler.AllowAutoTrigger).
							Str("name", handler.PluginName),
						).
						Msg("Error in by message type handler")
					handlerErr.Addf("error in by message type handler: %w", err)

					_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:    opts.CallbackQuery.From.ID,
						Text:      fmt.Sprintf("调用 %s 功能时发生了一些错误\n<blockquote expandable>%s</blockquote>", pluginName, err),
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "error in by message type handler notice").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "error in by message type handler notice", err)
					}
				} else {
					_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
						ChatID:    opts.CallbackQuery.From.ID,
						MessageID: opts.CallbackQuery.Message.Message.ID,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "select by message type handler keyboard").
							Msg(flaterr.DeleteMessages.Str())
						handlerErr.Addt(flaterr.DeleteMessage, "select by message type handler keyboard", err)
					}
				}
			} else {
				handler, isExist := AllPlugins.HandlerByMessageType[opts.CallbackQuery.Message.Message.Chat.Type][messageType][0][pluginName]
				if isExist {
					logger.Debug().
						Str("chatType", string(opts.CallbackQuery.Message.Message.Chat.Type)).
						Str("messageType", messageType.Str()).
						Int64("chatID", opts.CallbackQuery.Message.Message.Chat.ID).
						Str("pluginName", pluginName).
						Msg("User selected a by message type handler")
					if handler.MessageHandler == nil {
						logger.Warn().Msg("Hit by message type handler, but this handler function is nil, ignore")
						return handlerErr.Addf("hit by message type handler [%s], but this handler function is nil, ignore", pluginName).Flat()
					}
					err := handler.MessageHandler(&handler_params.Message{
						Ctx:      opts.Ctx,
						Thebot:   opts.Thebot,
						Message:  opts.CallbackQuery.Message.Message.ReplyToMessage,
						ChatInfo: opts.ChatInfo,
						Fields:   strings.Fields(opts.CallbackQuery.Message.Message.Text),
					})
					if err != nil {
						logger.Error().
							Err(err).
							Dict("handler", zerolog.Dict().
								Str("chatType", string(handler.ChatType)).
								Str("messageType", handler.MessageType.Str()).
								Int64("forChatID", handler.ForChatID).
								Bool("allowAutoTrigger", handler.AllowAutoTrigger).
								Str("name", handler.PluginName),
							).
							Msg("Error in by message type handler")
						handlerErr.Addf("error in by message type handler: %w", err)

						_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID:    opts.CallbackQuery.From.ID,
							Text:      fmt.Sprintf("调用 %s 功能时发生了一些错误\n<blockquote expandable>%s</blockquote>", pluginName, err),
							ParseMode: models.ParseModeHTML,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Str("content", "error in by message type handler notice").
								Msg(flaterr.SendMessage.Str())
							handlerErr.Addt(flaterr.SendMessage, "error in by message type handler notice", err)
						}
					} else {
						_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
							ChatID:    opts.CallbackQuery.From.ID,
							MessageID: opts.CallbackQuery.Message.Message.ID,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Str("content", "select by message type handler keyboard").
								Msg(flaterr.DeleteMessages.Str())
							handlerErr.Addt(flaterr.DeleteMessage, "select by message type handler keyboard", err)
						}
					}
				} else {
					_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
						CallbackQueryID: opts.CallbackQuery.ID,
						Text: fmt.Sprintf("此功能 [ %s ] 不可用，可能是管理员已经移除了这个功能", pluginName),
						ShowAlert: true,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "this by message type handler is not exist").
							Msg(flaterr.AnswerCallbackQuery.Str())
						handlerErr.Addt(flaterr.AnswerCallbackQuery, "this by message type handler is not exist", err)
					}
				}
			}
		}
	}

	return handlerErr.Flat()
}
