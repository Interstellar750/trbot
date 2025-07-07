package plugin_utils

import (
	"fmt"
	"trbot/utils/handler_params"
)

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

// is already run or not, error message
func RunSuffixCommandPlugin(params *handler_params.Update, messageParams *handler_params.Message, plugin SuffixCommand) (bool, error) {
	var err error
	switch {
	case plugin.MessageHandler != nil:
		err = plugin.MessageHandler(messageParams)
	case plugin.UpdateHandler != nil:
		err = plugin.UpdateHandler(params)
	default:
		return false, fmt.Errorf("hit suffix command handler, but this handler all function is nil, skip")
	}
	if err != nil {
		return true, err
	}
	return true, nil
}
