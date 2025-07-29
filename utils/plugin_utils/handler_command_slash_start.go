package plugin_utils

import (
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot/models"
)

type SlashStartCommand struct {
	Handler           []SlashStartHandler           // 例如 /start subcommand
	WithPrefixHandler []SlashStartWithPrefixHandler // 例如 /start subcommand_augument
}

type SlashStartHandler struct {
	Name           string
	Argument       string
	ForChatType    []models.ChatType // default for private, group, supergroup
	MessageHandler func(*handler_params.Message) error
}

func AddSlashStartCommandHandlers(handlers ...SlashStartHandler) int {
	if AllPlugins.SlashStartCommand.Handler == nil { AllPlugins.SlashStartCommand.Handler = []SlashStartHandler{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.Argument == "" { continue }
		if handler.ForChatType == nil {
			handler.ForChatType = []models.ChatType{models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup}
		}
		AllPlugins.SlashStartCommand.Handler = append(AllPlugins.SlashStartCommand.Handler, handler)
		handlerCount++
	}
	return handlerCount
}

// The Prefix is separated from the Argument by `_` (underline) symbol
type SlashStartWithPrefixHandler struct {
	Name           string
	Prefix         string
	Argument       string
	ForChatType    []models.ChatType // default for private, group, supergroup
	MessageHandler func(*handler_params.Message) error
}

func AddSlashStartWithPrefixCommandHandlers(handlers ...SlashStartWithPrefixHandler) int {
	if AllPlugins.SlashStartCommand.WithPrefixHandler == nil { AllPlugins.SlashStartCommand.WithPrefixHandler = []SlashStartWithPrefixHandler{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.Argument == "" { continue }
		if handler.ForChatType == nil {
			handler.ForChatType = []models.ChatType{models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup}
		}
		AllPlugins.SlashStartCommand.WithPrefixHandler = append(AllPlugins.SlashStartCommand.WithPrefixHandler, handler)
		handlerCount++
	}
	return handlerCount
}
