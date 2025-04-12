package consts

import (
	"github.com/go-telegram/bot/models"
)

var BotToken string // 全局 bot token

var WebhookURL  string // Webhook 运行模式下接受请求的 URL 地址
var WebhookPort string = "localhost:2847" // Webhook 运行模式下监听的端口

var LogChat_ID int64 = -1002499888124 // 用于接收日志的聊天 ID，可以是 用户 群聊 频道
var LogMan_IDs []int64 = []int64{ // 拥有查看日志权限的用户，可设定多个
	1086395364,
	2074319561,
}

var MetadataFileName string = "metadata.yaml"

var RedisURL string = "localhost:6379"
var RedisPassword string = ""
var RedisMainDB     int = 0
var RedisUserInfoDB int = 1
var RedisSubDB      int = 2

var DB_path      string = "./db_yaml/"
var LogFile_path string = DB_path + "log.txt"

var IsDebugMode bool
var Private_log bool = false

var BotMe *models.User // 用于存储 bot 信息

var InlineDefaultHandler   string = "voice" // 默认的 inline 命令，设为 "" 会显示打开菜单的提示
var InlineSubCommandSymbol string = "+"
var InlinePaginationSymbol string = "-"
var InlineResultsPerPage   int    = 50 // maxinum is 50, see https://core.telegram.org/bots/api#answerinlinequery

var Cache_path string = "./cache/"

var StickerCache_path    string = Cache_path + "sticker/"
var StickerCachePNG_path string = Cache_path + "sticker_png/"
var StickerCacheZip_path string = Cache_path + "sticker_zip/"

type SignalChannel struct {
	Database_save   chan bool
	PluginDB_save   chan bool
	PluginDB_reload chan bool
	WorkDone        chan bool
}

var SignalsChannel = SignalChannel{
	Database_save:   make(chan bool),
	PluginDB_save:   make(chan bool),
	PluginDB_reload: make(chan bool),
	WorkDone:        make(chan bool),
}
