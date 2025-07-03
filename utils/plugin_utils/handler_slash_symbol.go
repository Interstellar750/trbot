package plugin_utils

import "trbot/utils/handler_structs"

type SlashSymbolCommand struct {
	SlashCommand string // 'command' in '/command'
	Handler      func(*handler_structs.SubHandlerParams) error
}

func AddSlashSymbolCommandPlugins(Plugins ...SlashSymbolCommand) int {
	if AllPlugins.SlashSymbolCommand == nil { AllPlugins.SlashSymbolCommand = []SlashSymbolCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		if originPlugin.SlashCommand == "" { continue }
		AllPlugins.SlashSymbolCommand = append(AllPlugins.SlashSymbolCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
