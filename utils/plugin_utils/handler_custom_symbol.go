package plugin_utils

import "trbot/utils/handler_utils"

type Plugin_CustomSymbolCommand struct {
	FullCommand string
	Handler func(*handler_utils.SubHandlerOpts)
}

func AddCustomSymbolCommandPlugins(Plugins ...Plugin_CustomSymbolCommand) int {
	if AllPugins.CustomSymbolCommand == nil { AllPugins.CustomSymbolCommand = []Plugin_CustomSymbolCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.CustomSymbolCommand = append(AllPugins.CustomSymbolCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
