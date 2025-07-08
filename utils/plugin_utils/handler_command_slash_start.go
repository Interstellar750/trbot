package plugin_utils

import "trbot/utils/handler_params"

type SlashStartCommand struct {
	Handler           []SlashStartHandler           // 例如 /start subcommand
	WithPrefixHandler []SlashStartWithPrefixHandler // 例如 /start subcommand_augument
}

type SlashStartHandler struct {
	Name           string
	Argument       string
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
}

func AddSlashStartCommandPlugins(SlashStartCommandPlugins ...SlashStartHandler) int {
	if AllPlugins.SlashStartCommand.Handler == nil { AllPlugins.SlashStartCommand.Handler = []SlashStartHandler{} }

	var pluginCount int
	for _, originPlugin := range SlashStartCommandPlugins {
		if originPlugin.Argument == "" { continue }
		AllPlugins.SlashStartCommand.Handler = append(AllPlugins.SlashStartCommand.Handler, originPlugin)
		pluginCount++
	}
	return pluginCount
}

type SlashStartWithPrefixHandler struct {
	Name           string
	Prefix         string
	Argument       string
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
}

func AddSlashStartWithPrefixCommandPlugins(SlashStartWithPrefixCommandPlugins ...SlashStartWithPrefixHandler) int {
	if AllPlugins.SlashStartCommand.WithPrefixHandler == nil { AllPlugins.SlashStartCommand.WithPrefixHandler = []SlashStartWithPrefixHandler{} }

	var pluginCount int
	for _, originPlugin := range SlashStartWithPrefixCommandPlugins {
		if originPlugin.Argument == "" { continue }
		AllPlugins.SlashStartCommand.WithPrefixHandler = append(AllPlugins.SlashStartCommand.WithPrefixHandler, originPlugin)
		pluginCount++
	}
	return pluginCount
}
