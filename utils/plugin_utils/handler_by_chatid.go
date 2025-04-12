package plugin_utils

import "trbot/utils/handler_utils"

type HandlerByChatID struct {
	ChatID  int64
	Handler func(*handler_utils.SubHandlerOpts)
}

func AddHandlerByChatIDPlugins(Handlers ...HandlerByChatID) int {
	if AllPlugins.DefaultHandlerByChatID == nil {
		AllPlugins.DefaultHandlerByChatID = []HandlerByChatID{}
	}
	var pluginCount int
	for _, originPlugin := range Handlers {
		if originPlugin.ChatID == 0 { continue }
		AllPlugins.DefaultHandlerByChatID = append(AllPlugins.DefaultHandlerByChatID, originPlugin)
		pluginCount++
	}
	return pluginCount
}
