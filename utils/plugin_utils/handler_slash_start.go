package plugin_utils

import "trbot/utils/handler_utils"

type Plugin_SlashStart struct {
	Handler           []SlashStartHandler           // 例如 /start subcommand
	WithPrefixHandler []SlashStartWithPrefixHandler // 例如 /start subcommand_augument
}

type SlashStartHandler struct {
	Argument string
	Handler  func(*handler_utils.SubHandlerOpts)
}

func AddSlashStartCommandPlugins(SlashStartCommandPlugins ...SlashStartHandler) int {
	if AllPugins.SlashStart == nil {
		AllPugins.SlashStart = &Plugin_SlashStart{}
	}
	if AllPugins.SlashStart.Handler == nil {
		AllPugins.SlashStart.Handler = []SlashStartHandler{}
	}

	var pluginCount int
	for _, originPlugin := range SlashStartCommandPlugins {
		AllPugins.SlashStart.Handler = append(AllPugins.SlashStart.Handler, originPlugin)
		pluginCount++
	}
	return pluginCount
}

type SlashStartWithPrefixHandler struct {
	Prefix   string
	Argument string
	Handler  func(*handler_utils.SubHandlerOpts)
}

func AddSlashStartWithPrefixCommandPlugins(SlashStartWithPrefixCommandPlugins ...SlashStartWithPrefixHandler) int {
	if AllPugins.SlashStart == nil {
		AllPugins.SlashStart = &Plugin_SlashStart{}
	}
	if AllPugins.SlashStart.WithPrefixHandler == nil {
		AllPugins.SlashStart.WithPrefixHandler = []SlashStartWithPrefixHandler{}
	}

	var pluginCount int
	for _, originPlugin := range SlashStartWithPrefixCommandPlugins {
		AllPugins.SlashStart.WithPrefixHandler = append(AllPugins.SlashStart.WithPrefixHandler, originPlugin)
		pluginCount++
	}
	return pluginCount
}
