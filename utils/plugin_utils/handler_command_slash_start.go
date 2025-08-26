package plugin_utils

import (
	"strings"
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog/log"
)

/*
	If your plugin needs to obtain parameters from the /start command string, register
	it as `SlashStartWithPrefixHandler`. It will use the prefix to determine whether to trigger the command

	For example, with `/start command-name_argument1_argument2`, the handler will check if the `command-name` portion matches the registered plugin

	`SlashStartCommand` simply checks whether the `Argument` string is equal
*/

type SlashStartCommand struct {
	Handler           []SlashStartHandler           // 例如 /start subcommand
	WithPrefixHandler []SlashStartPrefixHandler // 例如 /start subcommand_augument
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
		if strings.Contains(handler.PrefixArgument, "_") {
			log.Warn().
				Str("HandlerName", handler.Name).
				Str("PrefixArgument", handler.PrefixArgument).
				Msg("Ignore this handler, `PrefixArgument` should not contain the underscore character `_`, it is used by the user to separate subsequent parameters in the program")
			continue
		}
		if handler.ForChatType == nil {
			handler.ForChatType = []models.ChatType{models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup}
		}
		AllPlugins.SlashStartCommand.WithPrefixHandler = append(AllPlugins.SlashStartCommand.WithPrefixHandler, handler)
		handlerCount++
	}
	return handlerCount
}
