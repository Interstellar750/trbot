package plugin_utils

import "trbot/utils/handler_structs"

type SuffixCommand struct {
	SuffixCommand string
	Handler       func(*handler_structs.SubHandlerParams) error
}

func AddSuffixCommandPlugins(Plugins ...SuffixCommand) int {
	if AllPlugins.SuffixCommand == nil { AllPlugins.SuffixCommand = []SuffixCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		if originPlugin.SuffixCommand == "" { continue }
		AllPlugins.SuffixCommand = append(AllPlugins.SuffixCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
