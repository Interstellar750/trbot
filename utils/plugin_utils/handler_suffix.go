package plugin_utils

import "trbot/utils/handler_params"

type SuffixCommand struct {
	SuffixCommand  string
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
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
