package plugin_utils

import "trbot/utils/handler_utils"

type Plugin_HandlerByChatID struct {
	ChatID  int64
	Handler func(*handler_utils.SubHandlerOpts)
}

func AddHandlerByChatIDPlugins(Handlers ...Plugin_HandlerByChatID) int {
	if AllPugins.DefaultHandlerByChatID == nil {
		AllPugins.DefaultHandlerByChatID = []Plugin_HandlerByChatID{}
	}
	var pluginCount int
	for _, originPlugin := range Handlers {
		AllPugins.DefaultHandlerByChatID = append(AllPugins.DefaultHandlerByChatID, originPlugin)
		pluginCount++
	}
	return pluginCount
}
