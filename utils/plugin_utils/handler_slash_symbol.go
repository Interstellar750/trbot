package plugin_utils

import "trbot/utils/handler_utils"

type SlashSymbolCommand struct {
	SlashCommand string // 'command' in '/command'
	Handler      func(*handler_utils.SubHandlerOpts)
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
