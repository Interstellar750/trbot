package plugin_utils

import (
	"trle5.xyz/gopkg/trbot/utils"
	"trle5.xyz/gopkg/trbot/utils/flaterr"
	"trle5.xyz/gopkg/trbot/utils/handler_params"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HandlerByMessageChatID map[int64]map[string]ByMessageChatIDHandler

/*
	It is allowed to set multiple handlers,
	and each handler will be triggered.

	However, due to the nature of the map,
	the execution order cannot be guaranteed.
*/
type ByMessageChatIDHandler struct {
	ForChatID  int64 // 0 for all chats
	PluginName string

	MessageHandler func(*handler_params.Message) error
}

func AddHandlerByMessageChatIDHandlers(handlers ...ByMessageChatIDHandler) int {
	if AllPlugins.HandlerByMessageChatID == nil { AllPlugins.HandlerByMessageChatID = HandlerByMessageChatID{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.PluginName == "" || handler.MessageHandler == nil {
			log.Error().
				Str(utils.GetCurrentFuncName()).
				Str("pluginName", handler.PluginName).
				Int64("forChatID", handler.ForChatID).
				Msgf("Not enough parameters, skip this handler")
			continue
		}
		if AllPlugins.HandlerByMessageChatID[handler.ForChatID] == nil { AllPlugins.HandlerByMessageChatID[handler.ForChatID] = map[string]ByMessageChatIDHandler{} }

		_, isExist := AllPlugins.HandlerByMessageChatID[handler.ForChatID][handler.PluginName]
		if isExist {
			log.Error().
				Str(utils.GetCurrentFuncName()).
				Int64("forChatID", handler.ForChatID).
				Str("name", handler.PluginName).
				Msg("Duplicate plugin exists, registration skipped")
		} else {
			AllPlugins.HandlerByMessageChatID[handler.ForChatID][handler.PluginName] = handler
			handlerCount++
		}
	}
	return handlerCount
}

func RemoveHandlerByMessageChatIDHandler(chatID int64, pluginName string) {
	if AllPlugins.HandlerByMessageChatID == nil { return }

	_, isExist := AllPlugins.HandlerByMessageChatID[chatID][pluginName]
	if isExist {
		delete(AllPlugins.HandlerByMessageChatID[chatID], pluginName)
	}
}

func RunByMessageChatIDHandlers(params *handler_params.Message) (int, error) {
	if AllPlugins.HandlerByMessageChatID == nil { return 0, nil }
	var handlerRunCount int
	var handlerErr      flaterr.MultErr

	logger := zerolog.Ctx(params.Ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Str("chatType", string(params.Message.Chat.Type)).
		Logger()

	for name, handler := range AllPlugins.HandlerByMessageChatID[params.Message.Chat.ID] {
		slogger := logger.With().
			Str("handlerName", name).
			Int64("forChatID", handler.ForChatID).
			Logger()

		if handler.MessageHandler != nil {
			slogger.Info().Msg("Hit by message chat ID handler")
			handlerRunCount++
			err := handler.MessageHandler(params)
			if err != nil {
				slogger.Error().
					Err(err).
					Msg("Error in by message chat ID handler")
				handlerErr.Addf("Error in by message chat ID handler [%s]: %w", name, err)
			}
		} else {
			slogger.Warn().Msg("Hit by message chat ID handler, but this handler function is nil, skip")
			handlerErr.Addf("Hit by message chat ID handler [%s], but this handler function is nil, skip", name)
		}
	}
	for name, handler := range AllPlugins.HandlerByMessageChatID[0] {
		slogger := logger.With().
			Str("handlerName", name).
			Int64("forChatID", handler.ForChatID).
			Logger()

		if handler.MessageHandler != nil {
			slogger.Info().Msg("Hit by message chat ID handler for any chat")
			handlerRunCount++
			err := handler.MessageHandler(params)
			if err != nil {
				slogger.Error().
					Err(err).
					Msg("Error in by message chat ID handler for any chat")
				handlerErr.Addf("Error in by message chat ID handler for any chat [%s]: %w", name, err)
			}
		} else {
			slogger.Warn().Msg("Hit by message chat ID handler, but this handler function is nil, skip")
			handlerErr.Addf("Hit by message chat ID handler [%s], but this handler function is nil, skip", name)
		}

	}

	return handlerRunCount, handlerErr.Flat()
}
