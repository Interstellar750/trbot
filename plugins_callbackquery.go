package main

// 为了兼容性考虑，建议仅将 commandChar 设置为单个字符（区分大小写），
// 因为 CallbackQuery 有长度限制，为 64 个字符，而贴纸包名的长度最大为 62。
// 再使用一个符号来隔开内容时，实际上能使用的识别字符长度只有一个字符。
// 你也可以忽略这个提醒，但在发送消息时使用 ReplyMarkup 参数添加按钮的时候，
// 需要评断并控制一下 CallbackData 的长度是否超过了 64 个字符，否则消息会无法发出。
// 或许用户发送的 Callback 请求，其 Query  可能会出现大小写不同，但服务器认为是同一个请求的情况，
// 建议为一个 handler 设定一个字符，同时捕获大小写
type Plugin_CallbackQuery struct {
	commandChar string
	handler func(*subHandlerOpts)
}

func AddCallbackQueryCommandPlugins(Plugins ...Plugin_CallbackQuery) int {
	if AllPugins.CallbackQuery == nil { AllPugins.CallbackQuery = []Plugin_CallbackQuery{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.CallbackQuery = append(AllPugins.CallbackQuery, originPlugin)
		pluginCount++
	}
	return pluginCount
}
