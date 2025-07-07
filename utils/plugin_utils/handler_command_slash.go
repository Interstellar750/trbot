package plugin_utils

import (
	"fmt"
	"trbot/utils/handler_params"
)

type SlashCommand struct {
	SlashCommand   string // 'command' in '/command'
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
}

func AddSlashCommandPlugins(Plugins ...SlashCommand) int {
	if AllPlugins.SlashCommand == nil { AllPlugins.SlashCommand = []SlashCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		if originPlugin.SlashCommand == "" { continue }
		AllPlugins.SlashCommand = append(AllPlugins.SlashCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}

// is already run or not, error message
func RunSlashCommandPlugin(params *handler_params.Update, messageParams *handler_params.Message, plugin SlashCommand) (bool, error) {
	var err error
	switch {
	case plugin.MessageHandler != nil:
		err = plugin.MessageHandler(messageParams)
	case plugin.UpdateHandler != nil:
		err = plugin.UpdateHandler(params)
	default:
		return false, fmt.Errorf("hit slash symbol command handler, but this handler all function is nil, skip")
	}
	if err != nil {
		return true, err
	}
	return true, nil
}
