package plugin_utils

import "trbot/utils/handler_utils"

type Plugin_SlashSymbolCommand struct {
	SlashCommand string // 'command' in '/command'
	Handler      func(*handler_utils.SubHandlerOpts)
}

func AddSlashSymbolCommandPlugins(Plugins ...Plugin_SlashSymbolCommand) int {
	if AllPugins.SlashSymbolCommand == nil {
		AllPugins.SlashSymbolCommand = []Plugin_SlashSymbolCommand{}
	}

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.SlashSymbolCommand = append(AllPugins.SlashSymbolCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
