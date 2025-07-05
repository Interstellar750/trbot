package plugin_utils

import "trbot/utils/handler_params"

type FullCommand struct {
	FullCommand    string
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
}

func AddFullCommandPlugins(Plugins ...FullCommand) int {
	if AllPlugins.FullCommand == nil { AllPlugins.FullCommand = []FullCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		if originPlugin.FullCommand == "" { continue }
		AllPlugins.FullCommand = append(AllPlugins.FullCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
