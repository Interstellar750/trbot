package plugin_utils

import "trbot/utils/handler_utils"

type SuffixCommand struct {
	SuffixCommand string
	Handler       func(*handler_utils.SubHandlerOpts)
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
