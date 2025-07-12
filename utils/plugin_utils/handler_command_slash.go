package plugin_utils

import (
	"strings"
	"trbot/utils"
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

type SlashCommand struct {
	SlashCommand string       // the `command` in `/command`
	ForChatType  []models.ChatType // default for private, group, supergroup

	// only allowed access to `update.Message` field, If the handler can handle multiple update types, register it as an `UpdateHandler`.
	MessageHandler func(*handler_params.Message) error
	// with full access to `update`, If both `MessageHandler` and `UpdateHandler` are set, only `MessageHandler` will be called.
	UpdateHandler  func(*handler_params.Update)  error
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
func RunSlashCommandHandlers(params *handler_params.Update) (bool, error) {
	var isProcessed bool
	var err         error

	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "RunSlashCommandHandler").
		Logger()

	fields := strings.Fields(params.Update.Message.Text)

	for _, plugin := range AllPlugins.SlashCommand {
		if utils.CommandMaybeWithSuffixUsername(fields, "/" + plugin.SlashCommand) && utils.AnyContains(params.Update.Message.Chat.Type, plugin.ForChatType) {
			logger.Info().
				Str("slashCommand", plugin.SlashCommand).
				Msg("Hit slash command handler")

			switch {
			case plugin.MessageHandler != nil:
				err = plugin.MessageHandler(&handler_params.Message{
					Ctx:      params.Ctx,
					Thebot:   params.Thebot,
					Message:  params.Update.Message,
					ChatInfo: params.ChatInfo,
					Fields:   fields,
				})
			case plugin.UpdateHandler != nil:
				err = plugin.UpdateHandler(params)
			default:
				logger.Warn().
					Msg("Hit slash symbol command handler, but this handler all function is nil, skip")
				continue
			}
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Error in slash symbol command handler")
				return true, err
			}
			isProcessed = true
		}
	}
	return isProcessed, err


}
