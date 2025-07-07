package plugin_utils

import (
	"fmt"
	"trbot/utils/handler_params"
)

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

// is already run or not, error message
func RunFullCommandPlugin(params *handler_params.Update, messageParams *handler_params.Message, plugin FullCommand) (bool, error) {
	var err error
	switch {
	case plugin.MessageHandler != nil:
		err = plugin.MessageHandler(messageParams)
	case plugin.UpdateHandler != nil:
		err = plugin.UpdateHandler(params)
	default:
		return false, fmt.Errorf("hit full command handler, but this handler all function is nil, skip")
	}
	if err != nil {
		return true, err
	}
	return true, nil
}
