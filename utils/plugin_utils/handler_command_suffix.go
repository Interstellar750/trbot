package plugin_utils

import (
	"strings"
	"trbot/utils"
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

type SuffixCommand struct {
	SuffixCommand  string
	ForChatType    []models.ChatType // default for private, group, supergroup
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
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
func RunSuffixCommandHandlers(params *handler_params.Update) (bool, error) {
	var isProcessed bool
	var err         error

	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "RunSuffixCommandPlugin").
		Logger()

	fields := strings.Fields(params.Update.Message.Text)

	for _, plugin := range AllPlugins.SuffixCommand {
		if strings.HasSuffix(fields[len(fields)-1], plugin.SuffixCommand) && utils.AnyContains(params.Update.Message.Chat.Type, plugin.ForChatType) {
			logger.Info().
				Str("suffixCommand", plugin.SuffixCommand).
				Msg("Hit suffix command handler")

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
					Msg("Hit suffix command handler, but this handler all function is nil, skip")
				continue
			}
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Error in suffix command handler")
				return true, err
			}
			isProcessed = true
		}
	}
	return isProcessed, err
}
