package plugin_utils

import "trbot/utils/handler_utils"

type Plugin_SuffixCommand struct {
	SuffixCommand string
	Handler       func(*handler_utils.SubHandlerOpts)
}

func AddSuffixCommandPlugins(Plugins ...Plugin_SuffixCommand) int {
	if AllPugins.SuffixCommand == nil {
		AllPugins.SuffixCommand = []Plugin_SuffixCommand{}
	}

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.SuffixCommand = append(AllPugins.SuffixCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
