package plugin_utils

import (
	"errors"
	"fmt"
	"trbot/utils/handler_params"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

type StateHandler struct {
	ForChatID   int64
	PluginName  string
	// StopTimeout time.Time

	// handler will auto remove when Remaining == 0
	//
	// if Remaining == -1, it will never remove
	Remaining   int
	Handler     func(*handler_params.Update) error
}

func AddStateHandler(handler StateHandler) bool {
	if AllPlugins.StateHandler == nil { AllPlugins.StateHandler = map[int64]StateHandler{} }
	if handler.ForChatID == 0 {
		log.Error().
			Str("funcName", "AddStateHandler").
			Int64("forChatID", handler.ForChatID).
			Str("name", handler.PluginName).
			Int("Remaining", handler.Remaining).
			Msg("Duplicate plugin exists, registration skipped")
		return false
	}
	if handler.Remaining == 0 {
		log.Error().
			Str("funcName", "AddStateHandler").
			Int64("forChatID", handler.ForChatID).
			Str("name", handler.PluginName).
			Int("Remaining", handler.Remaining).
			Msg("No remaining times set, registration skipped")
		return false
	}
	if handler.Handler == nil {
		log.Error().
			Str("funcName", "AddStateHandler").
			Int64("forChatID", handler.ForChatID).
			Str("name", handler.PluginName).
			Int("Remaining", handler.Remaining).
			Msg("No handler set, registration skipped")
		return false
	}
	AllPlugins.StateHandler[handler.ForChatID] = handler
	return true
}

func RemoveStateHandler(chatID int64) {
	if chatID == 0 { return }
	delete(AllPlugins.StateHandler, chatID)
}

// this can't edit `remainingTime` to `0` or `stateFunc` to `nil`, if you need remove state handler, use `plugin_utils.RemoveStateHandler()`
func EditStateHandler(chatID int64, remainingTime int, stateFunc func(*handler_params.Update) error) (bool, error) {
	if chatID == 0 { return false, errors.New("chatID is required") }

	targetHandler, isExist := AllPlugins.StateHandler[chatID]
	if !isExist {
		log.Warn().
			Str("funcName", "EditStateHandler").
			Int64("forChatID", chatID).
			Msg("No state handler exists, edit stopped")
		return false, fmt.Errorf("no state handler for %d chatID", chatID)
	}
	if remainingTime != 0 {
		targetHandler.Remaining = remainingTime
	}
	if stateFunc != nil {
		targetHandler.Handler = stateFunc
	}
	AllPlugins.StateHandler[chatID] = targetHandler
	return true, nil
}

func RunStateHandler(opts *handler_params.Update) bool {
	if opts.ChatInfo.ID == 0 { return false }
	handler, isExist := AllPlugins.StateHandler[opts.ChatInfo.ID]
	if isExist {
		logger := zerolog.Ctx(opts.Ctx).
			With().
			Str("funcName", "RunStateHandler").
			Int64("forChatID", opts.ChatInfo.ID).
			Str("pluginName", handler.PluginName).
			Logger()

		logger.Info().Msg("Hit state handler")
		if handler.Handler == nil {
			logger.Error().Msg("Handler is nil")
			return false
		}
		if handler.Remaining > 0 { handler.Remaining-- }
		err := handler.Handler(opts)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Error in state handler")
		}
		if handler.Remaining == 0 {
			delete(AllPlugins.StateHandler, opts.ChatInfo.ID)
		} else {
			AllPlugins.StateHandler[opts.ChatInfo.ID] = handler
		}
	}
	return isExist
}
