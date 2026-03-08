package plugin_utils

import "trle5.xyz/trbot/utils/handler_params"

// 为了兼容性考虑，建议请处理好 CallbackDatePrefix 的长度，因为 CallbackQuery 有长度限制，为 64 个字符。
// 例如贴纸包名的长度最大为 62 个字符，再使用一个符号来隔开内容时，实际上能使用的识别字符长度只有一个字符。
// 你也可以忽略这个提醒，但在发送消息时使用 ReplyMarkup 参数添加按钮的时候，需要评断并控制一下 CallbackData 的长度是否超过了 64 个字符，否则消息会无法发出。
type CallbackQuery struct {
	CallbackDataPrefix string

	CallbackQueryHandler func(*handler_params.CallbackQuery) error
}

func AddCallbackQueryHandlers(handlers ...CallbackQuery) int {
	if AllPlugins.CallbackQuery == nil { AllPlugins.CallbackQuery = []CallbackQuery{} }

	var handlerCount int
	for _, handler := range handlers {
		AllPlugins.CallbackQuery = append(AllPlugins.CallbackQuery, handler)
		handlerCount++
	}
	return handlerCount
}
