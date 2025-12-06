package plugin_utils

import (
	"strings"

	"trle5.xyz/gopkg/trbot/utils"
	"trle5.xyz/gopkg/trbot/utils/handler_params"
	"trle5.xyz/gopkg/trbot/utils/type/contain"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

type FullCommand struct {
	FullCommand    string
	ForChatType    []models.ChatType // default for private, group, supergroup
	MessageHandler func(*handler_params.Message) error
}

func AddFullCommandHandlers(handlers ...FullCommand) int {
	if AllPlugins.FullCommand == nil { AllPlugins.FullCommand = []FullCommand{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.FullCommand == "" { continue }
		if handler.ForChatType == nil {
			handler.ForChatType = []models.ChatType{models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup}
		}
		AllPlugins.FullCommand = append(AllPlugins.FullCommand, handler)
		handlerCount++
	}
	return handlerCount
}

// is already run or not, error message
func RunFullCommandHandlers(params *handler_params.Message) (bool, error) {
	for _, plugin := range AllPlugins.FullCommand {
		if strings.HasPrefix(params.Message.Text, plugin.FullCommand) && contain.AnyType(params.Message.Chat.Type, plugin.ForChatType...) {
			logger := zerolog.Ctx(params.Ctx).With().
				Str(utils.GetCurrentFuncName()).
				Str("FullCommand", plugin.FullCommand).
				Logger()

			if plugin.MessageHandler != nil {
				logger.Info().Msg("Hit full command handler")
				err := plugin.MessageHandler(params)
				if err != nil {
					logger.Error().
						Err(err).
						Msg("Error in full command handler")
				}
				return true, err
			} else {
				logger.Warn().Msg("Hit full command handler, but this handler function is nil, skip")
			}
		}
	}
	return false, nil
}
