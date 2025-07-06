package plugin_utils

import "trbot/utils/handler_params"

type SlashCommand struct {
	SlashCommand   string // 'command' in '/command'
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
}

func AddSlashCommandPlugins(Plugins ...SlashCommand) int {
	if AllPlugins.SlashCommand == nil { AllPlugins.SlashCommand = []SlashCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		if originPlugin.SlashCommand == "" { continue }
		AllPlugins.SlashCommand = append(AllPlugins.SlashCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
