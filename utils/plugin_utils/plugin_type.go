package plugin_utils

type Plugin_All struct {
	Databases           []DatabaseHandler

	HandlerHelp         []HandlerHelp
	// Inline mode
	InlineHandler       []InlineHandler       // 函数返回全部列表，由预设函数进行分页
	InlineManualHandler []InlineManualHandler // 函数自行处理输出
	InlinePrefixHandler []InlinePrefixHandler // 例如设定命令为 abc，则 abca, abcb 等后续包含任意字符和字段都会触发
	InlineCommandList   []InlineCommandList   // inline 命令列表

	SlashStart           *SlashStartCommand   // '/start' 命令和后面的 query
	SlashSymbolCommand  []SlashSymbolCommand  // 以 '/' 符号开头的命令，例如 '/help' '/test'
	CustomSymbolCommand []CustomSymbolCommand // 手动定义符号的命令，例如定义符号为 '!'，则命令为 '!help' 或 '!test', 也可以不用不符号，直接 help 或 test
	SuffixCommand       []SuffixCommand       // 后缀命令，例如 'help' 'test'，需要以空格开头

	// InlineKeyboardMarkup
	CallbackQuery []CallbackQuery // 处理 InlineKeyboardMarkup 的 callback 函数

	// 根据聊天类型设定的默认处理函数
	HandlerByMessageTypeFor HandlerByMessageTypeFor

	// 以聊天 ID 设定的默认处理函数，第一个 map 为 ID，第二个为 handler 名称
	HandlerByChatID map[int64]map[string]HandlerByChatID
}

var AllPlugins = Plugin_All{}
