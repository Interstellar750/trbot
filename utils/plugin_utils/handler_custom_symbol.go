package plugin_utils

import "trbot/utils/handler_structs"

type CustomSymbolCommand struct {
	FullCommand string
	Handler func(*handler_structs.SubHandlerParams) error
}

func AddCustomSymbolCommandPlugins(Plugins ...CustomSymbolCommand) int {
	if AllPlugins.CustomSymbolCommand == nil { AllPlugins.CustomSymbolCommand = []CustomSymbolCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		if originPlugin.FullCommand == "" { continue }
		AllPlugins.CustomSymbolCommand = append(AllPlugins.CustomSymbolCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
