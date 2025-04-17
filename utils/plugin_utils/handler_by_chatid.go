package plugin_utils

import "trbot/utils/handler_utils"

type HandlerByChatID struct {
	ChatID  int64
	Handler func(*handler_utils.SubHandlerOpts)
}

func AddHandlerByChatIDPlugins(Handlers ...HandlerByChatID) int {
	if AllPlugins.HandlerByChatID == nil {
		AllPlugins.HandlerByChatID = []HandlerByChatID{}
	}
	var pluginCount int
	for _, originPlugin := range Handlers {
		if originPlugin.ChatID == 0 { continue }
		AllPlugins.HandlerByChatID = append(AllPlugins.HandlerByChatID, originPlugin)
		pluginCount++
	}
	return pluginCount
}
