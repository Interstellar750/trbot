package plugin_utils

import (
	"strings"
	"trbot/utils"
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

type FullCommand struct {
	FullCommand    string
	ForChatType    []models.ChatType // default for private, group, supergroup
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
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
func RunFullCommandHandlers(params *handler_params.Update) (bool, error) {
	var isProcessed bool
	var err         error

	logger := zerolog.Ctx(params.Ctx).
		With().
		Str("funcName", "RunFullCommandPlugin").
		Logger()

	for _, plugin := range AllPlugins.FullCommand {
		if strings.HasPrefix(params.Update.Message.Text, plugin.FullCommand) && utils.AnyContains(params.Update.Message.Chat.Type, plugin.ForChatType) {

			logger.Info().
				Str("fullCommand", plugin.FullCommand).
				Msg("Hit full command handler")

			switch {
			case plugin.MessageHandler != nil:
				err = plugin.MessageHandler(&handler_params.Message{
					Ctx:      params.Ctx,
					Thebot:   params.Thebot,
					Message:  params.Update.Message,
					ChatInfo: params.ChatInfo,
					Fields:   strings.Fields(params.Update.Message.Text),
				})
			case plugin.UpdateHandler != nil:
				err = plugin.UpdateHandler(params)
			default:
				logger.Warn().
					Msg("Hit full command handler, but this handler all function is nil, skip")
				continue
			}
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Error in full command handler")
				return true, err
			}
			isProcessed = true
		}
	}
	return isProcessed, err
}
