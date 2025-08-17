package plugin_utils

import (
	"trbot/utils"
	"trbot/utils/handler_params"
	"trbot/utils/type/contain"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

type SlashCommand struct {
	SlashCommand string            // the `command` in `/command`
	ForChatType  []models.ChatType // default for private, group, supergroup

	// only allowed access to `update.Message` field
	MessageHandler func(*handler_params.Message) error
}

func AddSlashCommandHandlers(handlers ...SlashCommand) int {
	if AllPlugins.SlashCommand == nil { AllPlugins.SlashCommand = []SlashCommand{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.SlashCommand == "" { continue }
		if handler.ForChatType == nil {
			handler.ForChatType = []models.ChatType{models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup}
		}
		AllPlugins.SlashCommand = append(AllPlugins.SlashCommand, handler)
		handlerCount++
	}
	return handlerCount
}

// is already run or not, error message
func RunSlashCommandHandlers(params *handler_params.Message) (bool, error) {
	for _, plugin := range AllPlugins.SlashCommand {
		if utils.CommandMaybeWithSuffixUsername(params.Fields, "/" + plugin.SlashCommand) && contain.AnyType(params.Message.Chat.Type, plugin.ForChatType...) {
			logger := zerolog.Ctx(params.Ctx).With().
				Str(utils.GetCurrentFuncName()).
				Str("slashCommand", plugin.SlashCommand).
				Logger()

			if plugin.MessageHandler != nil {
				logger.Info().Msg("Hit slash command handler")
				err := plugin.MessageHandler(params)
				if err != nil {
					logger.Error().
						Err(err).
						Msg("Error in slash command handler")
				}
				return true, err
			} else {
				logger.Warn().Msg("Hit slash command handler, but this handler function is nil, skip")
			}
		}
	}
	return false, nil
}
