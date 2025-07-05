package plugin_utils

import (
	"trbot/utils/handler_params"
)

/*
	It is allowed to set multiple handlers,
	and each handler will be triggered.

	However, due to the nature of the map,
	the execution order cannot be guaranteed.
*/
type HandlerByChatID struct {
	ChatID         int64
	PluginName     string
	MessageHandler func(*handler_params.Message) error
	UpdateHandler  func(*handler_params.Update)  error
}

func AddHandlerByChatIDPlugins(handlers ...HandlerByChatID) int {
	if AllPlugins.HandlerByChatID == nil { AllPlugins.HandlerByChatID = map[int64]map[string]HandlerByChatID{} }

	var pluginCount int
	for _, handler := range handlers {
		if handler.ChatID     == 0 || handler.PluginName == "" { continue }
		if AllPlugins.HandlerByChatID[handler.ChatID] == nil { AllPlugins.HandlerByChatID[handler.ChatID] = map[string]HandlerByChatID{} }

		chatIDMap := AllPlugins.HandlerByChatID[handler.ChatID]
		chatIDMap[handler.PluginName] = handler
		AllPlugins.HandlerByChatID[handler.ChatID] = chatIDMap
		pluginCount++
	}
	// fmt.Println(AllPlugins.HandlerByChatID)
	return pluginCount
}

func RemoveHandlerByChatIDPlugin(chatID int64, pluginName string) {
	if AllPlugins.HandlerByChatID == nil { return }

	_, isExist := AllPlugins.HandlerByChatID[chatID][pluginName]
	if isExist {
		delete(AllPlugins.HandlerByChatID[chatID], pluginName)
	}
}
