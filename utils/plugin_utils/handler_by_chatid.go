package plugin_utils

import (
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type HandlerByChatID map[int64]map[string]ByChatIDHandler

/*
	It is allowed to set multiple handlers,
	and each handler will be triggered.

	However, due to the nature of the map,
	the execution order cannot be guaranteed.
*/
type ByChatIDHandler struct {
	ForChatID  int64 // 0 for all chats
	PluginName string

	// with full access to `update`
	UpdateHandler func(*handler_params.Update) error
}

func AddHandlerByChatIDHandlers(handlers ...ByChatIDHandler) int {
	if AllPlugins.HandlerByChatID == nil { AllPlugins.HandlerByChatID = HandlerByChatID{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.PluginName == "" || handler.UpdateHandler == nil {
			log.Error().
				Str("funcName", "AddHandlerByChatIDHandlers").
				Str("pluginName", handler.PluginName).
				Int64("forChatID", handler.ForChatID).
				Msgf("Not enough parameters, skip this handler")
			continue
		}
		if AllPlugins.HandlerByChatID[handler.ForChatID] == nil { AllPlugins.HandlerByChatID[handler.ForChatID] = map[string]ByChatIDHandler{} }

		_, isExist := AllPlugins.HandlerByChatID[handler.ForChatID][handler.PluginName]
		if isExist {
			log.Error().
				Str("funcName", "AddHandlerByChatIDPlugins").
				Int64("forChatID", handler.ForChatID).
				Str("name", handler.PluginName).
				Msg("Duplicate plugin exists, registration skipped")
		} else {
			AllPlugins.HandlerByChatID[handler.ForChatID][handler.PluginName] = handler
			handlerCount++
		}
	}
	return handlerCount
}

func RemoveHandlerByChatIDHandler(chatID int64, pluginName string) {
	if AllPlugins.HandlerByChatID == nil { return }

	_, isExist := AllPlugins.HandlerByChatID[chatID][pluginName]
	if isExist {
		delete(AllPlugins.HandlerByChatID[chatID], pluginName)
	}
}

func RunByChatIDHandlers(params *handler_params.Update) (int, error) {
	var handlerRunCount int
	var handlerErr      flaterr.MultErr

	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "RunByChatIDHandlers").
		Str("chatType", string(params.Update.Message.Chat.Type)).
		Logger()

	if AllPlugins.HandlerByChatID[params.Update.Message.Chat.ID] != nil {
		for name, handler := range AllPlugins.HandlerByChatID[params.Update.Message.Chat.ID] {
			slogger := logger.With().
				Str("handlerName", name).
				Int64("forChatID", handler.ForChatID).
				Logger()

			if handler.UpdateHandler != nil {
				slogger.Info().Msg("Hit by chat ID handler")
				handlerRunCount++
				err := handler.UpdateHandler(params)
				if err != nil {
					slogger.Error().
						Err(err).
						Msg("Error in by chat ID handler")
					handlerErr.Addf("Error in by chat ID handler [%s]: %w", name, err)
				}
			} else {
				slogger.Warn().Msg("Hit by chat ID handler, but this handler function is nil, skip")
				handlerErr.Addf("hit by chat ID handler [%s], but this handler function is nil, skip", name)
			}
		}
	}
	if AllPlugins.HandlerByChatID[0] != nil {
		for name, handler := range AllPlugins.HandlerByChatID[0] {
			slogger := logger.With().
				Str("handlerName", name).
				Int64("forChatID", handler.ForChatID).
				Logger()

			slogger.Info().Msg("Hit by chat ID handler for any chat")

			if handler.UpdateHandler == nil {
				slogger.Warn().Msg("Hit by chat ID handler, but this handler function is nil, skip")
				handlerErr.Addf("hit by chat ID handler [%s], but this handler function is nil, skip", name)
				continue
			}
			handlerRunCount++
			err := handler.UpdateHandler(params)
			if err != nil {
				slogger.Error().
					Err(err).
					Msg("Error in by chat ID handler for any chat")
				handlerErr.Addf("Error in by chat ID handler for any chat [%s]: %w", name, err)
			}
		}
	}

	return handlerRunCount, handlerErr.Flat()
}
