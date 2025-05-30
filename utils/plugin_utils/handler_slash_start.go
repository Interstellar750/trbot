package plugin_utils

import "trbot/utils/handler_structs"

type SlashStartCommand struct {
	Handler           []SlashStartHandler           // 例如 /start subcommand
	WithPrefixHandler []SlashStartWithPrefixHandler // 例如 /start subcommand_augument
}

type SlashStartHandler struct {
	Name     string
	Argument string
	Handler  func(*handler_structs.SubHandlerParams) error
}

func AddSlashStartCommandPlugins(SlashStartCommandPlugins ...SlashStartHandler) int {
	if AllPlugins.SlashStart.Handler == nil { AllPlugins.SlashStart.Handler = []SlashStartHandler{} }

	var pluginCount int
	for _, originPlugin := range SlashStartCommandPlugins {
		if originPlugin.Argument == "" { continue }
		AllPlugins.SlashStart.Handler = append(AllPlugins.SlashStart.Handler, originPlugin)
		pluginCount++
	}
	return pluginCount
}

type SlashStartWithPrefixHandler struct {
	Name     string
	Prefix   string
	Argument string
	Handler  func(*handler_structs.SubHandlerParams) error
}

func AddSlashStartWithPrefixCommandPlugins(SlashStartWithPrefixCommandPlugins ...SlashStartWithPrefixHandler) int {
	if AllPlugins.SlashStart.WithPrefixHandler == nil { AllPlugins.SlashStart.WithPrefixHandler = []SlashStartWithPrefixHandler{} }

	var pluginCount int
	for _, originPlugin := range SlashStartWithPrefixCommandPlugins {
		if originPlugin.Argument == "" { continue }
		AllPlugins.SlashStart.WithPrefixHandler = append(AllPlugins.SlashStart.WithPrefixHandler, originPlugin)
		pluginCount++
	}
	return pluginCount
}
