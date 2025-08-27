package plugin_utils

import (
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot/models"
)

type SlashStartCommand struct {
	Handler           []SlashStartHandler           // 例如 /start subcommand
	WithPrefixHandler []SlashStartPrefixHandler // 例如 /start subcommand_augument
}

// SlashStartHandler needs to match `argument_sometext` exactly in `/start argument_sometext` string
//
// If you need to match by prefix, register it as `SlashStartPrefixHandler` to get some parameters
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

// SlashStartPrefixHandler uses the prefix to match the `argument` in the `/start argument_sometext` string
//
// If you need to match the `argument_sometext` in the string exactly, register it as `SlashStartHandler`
type SlashStartPrefixHandler struct {
	Name           string
	PrefixArgument string
	ForChatType    []models.ChatType // default for private, group, supergroup
	MessageHandler func(*handler_params.Message) error
}

func AddSlashStartPrefixCommandHandlers(handlers ...SlashStartPrefixHandler) int {
	if AllPlugins.SlashStartCommand.WithPrefixHandler == nil { AllPlugins.SlashStartCommand.WithPrefixHandler = []SlashStartPrefixHandler{} }

	var handlerCount int
	for _, handler := range handlers {
		if handler.PrefixArgument == "" { continue }
		if handler.ForChatType == nil {
			handler.ForChatType = []models.ChatType{models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup}
		}
		AllPlugins.SlashStartCommand.WithPrefixHandler = append(AllPlugins.SlashStartCommand.WithPrefixHandler, handler)
		handlerCount++
	}
	return handlerCount
}
