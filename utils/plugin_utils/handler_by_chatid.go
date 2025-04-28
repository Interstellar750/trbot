package plugin_utils

import (
	"fmt"
	"trbot/utils/handler_utils"
)

type HandlerByChatID struct {
	ChatID     int64
	PluginName string
	Handler    func(*handler_utils.SubHandlerOpts)
}

func AddHandlerByChatIDPlugins(Handlers ...HandlerByChatID) int {
	if AllPlugins.HandlerByChatID == nil {
		AllPlugins.HandlerByChatID = map[int64]map[string]HandlerByChatID{}
	}
	var pluginCount int
	for _, originPlugin := range Handlers {
		if originPlugin.ChatID     == 0  { continue }
		if originPlugin.PluginName == "" { continue }
		// var chatIDMap = map[string]HandlerByChatID{}
		chatIDMap := AllPlugins.HandlerByChatID[originPlugin.ChatID]
		if chatIDMap == nil {
			chatIDMap = map[string]HandlerByChatID{}
		}
		chatIDMap[originPlugin.PluginName] = originPlugin
		AllPlugins.HandlerByChatID[originPlugin.ChatID] = chatIDMap
		pluginCount++
	}
	fmt.Println(AllPlugins.HandlerByChatID)
	return pluginCount
}

func RemoveHandlerByChatIDPlugin(chatID int64, pluginName string) bool {
	if AllPlugins.HandlerByChatID == nil { return false }

	nameMap, isExist := AllPlugins.HandlerByChatID[chatID]
	if !isExist{ return false }

	_, isExist = nameMap[pluginName]
	if isExist{
		delete(AllPlugins.HandlerByChatID[chatID], pluginName)
		return true
	}

	return false
}
