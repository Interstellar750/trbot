package plugin_utils

import (
	"trbot/utils/handler_utils"
)

type Plugin_All struct {
	Databases           []DatabaseHandler
	// Inline mode
	Inline              []Plugin_Inline            // 函数返回全部列表，由预设函数进行分页
	InlineManual        []Plugin_InlineManual      // 函数自行处理输出
	InlinePrefix        []Plugin_InlinePrefix      // 例如设定命令为 abc，则 abca, abcb 等后续包含任意字符和字段都会触发
	InlineCommandList   []Plugin_InlineCommandList // inline 命令列表

	SlashStart           *Plugin_SlashStart          // '/start' 命令和后面的 query
	SlashSymbolCommand  []Plugin_SlashSymbolCommand  // 以 '/' 符号开头的命令，例如 '/help' '/test'
	CustomSymbolCommand []Plugin_CustomSymbolCommand // 手动定义符号的命令，例如定义符号为 '!'，则命令为 '!help' 或 '!test', 也可以不用不符号，直接 help 或 test
	SuffixCommand       []Plugin_SuffixCommand       // 后缀命令，例如 'help' 'test'，需要以空格开头

	// InlineKeyboardMarkup
	CallbackQuery       []Plugin_CallbackQuery     // 处理 InlineKeyboardMarkup 的 callback 函数

	// 根据聊天类型设定的默认处理函数
	DefaultHandlerByMessageTypeForPrivate    *Plugin_HandlerByMessageType
	DefaultHandlerByMessageTypeForGroup      *Plugin_HandlerByMessageType
	DefaultHandlerByMessageTypeForSupergroup *Plugin_HandlerByMessageType
	DefaultHandlerByMessageTypeForChannel    *Plugin_HandlerByMessageType

	DefaultHandlerByChatID []Plugin_HandlerByChatID
}

var AllPugins = Plugin_All{}

type Plugin_HandlerByMessageType struct {
	Photo   func(*handler_utils.SubHandlerOpts)


	Message   func(*handler_utils.SubHandlerOpts)
	Sticker   func(*handler_utils.SubHandlerOpts)
	Document  func(*handler_utils.SubHandlerOpts)
	Audio     func(*handler_utils.SubHandlerOpts)
	Video     func(*handler_utils.SubHandlerOpts)
	VideoNote func(*handler_utils.SubHandlerOpts)
	Voice     func(*handler_utils.SubHandlerOpts)
	Contact   func(*handler_utils.SubHandlerOpts)
	Location  func(*handler_utils.SubHandlerOpts)

}
