package plugin_utils

import (
	"strings"

	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/type/contain"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

type SuffixCommand struct {
	SuffixCommand  string
	ForChatType    []models.ChatType // default for private, group, supergroup
	MessageHandler func(*handler_params.Message) error
}

func AddSuffixCommandHandlers(handlers ...SuffixCommand) int {
	if AllPlugins.SuffixCommand == nil { AllPlugins.SuffixCommand = []SuffixCommand{} }

	var pluginCount int
	for _, handler := range handlers {
		if handler.SuffixCommand == "" { continue }
		if handler.ForChatType == nil {
			handler.ForChatType = []models.ChatType{models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup}
		}
		AllPlugins.SuffixCommand = append(AllPlugins.SuffixCommand, handler)
		pluginCount++
	}
	return pluginCount
}

// is already run or not, error message
func RunSuffixCommandHandlers(params *handler_params.Message) (bool, error) {
	for _, plugin := range AllPlugins.SuffixCommand {
		if strings.HasSuffix(params.Fields[len(params.Fields)-1], plugin.SuffixCommand) && contain.AnyType(params.Message.Chat.Type, plugin.ForChatType...) {
			logger := zerolog.Ctx(params.Ctx).With().
				Str(utils.GetCurrentFuncName()).
				Str("suffixCommand", plugin.SuffixCommand).
				Logger()

			if plugin.MessageHandler != nil {
				logger.Info().Msg("Hit suffix command handler")
				err := plugin.MessageHandler(params)
				if err != nil {
					logger.Error().
						Err(err).
						Msg("Error in suffix command handler")
				}
				return true, err
			} else {
				logger.Warn().Msg("Hit suffix command handler, but this handler function is nil, skip")
			}
		}
	}
	return false, nil
}
