package consts

import (
	"github.com/go-telegram/bot/models"
)

var BotToken string // 全局 bot token

var WebhookURL string // Webhook 运行模式下接受请求的 URL 地址
var WebhookPort string = "localhost:2847" // Webhook 运行模式下监听的端口

var LogChat_ID int64 = -1002499888124 // 用于接收日志的聊天 ID，可以是 用户 群聊 频道
var LogMan_IDs []int64 = []int64{1086395364, 2074319561} // 拥有查看日志权限的用户，可设定多个

var MetadataFileName string = "metadata.yaml"

var RedisURL string = "localhost:6379"
var RedisPassword string = ""
var RedisMainDB  int = 0
var RedisCountDB int = 1
var RedisSubDB   int = 2

var DB_path      string = "./db_yaml/"
var Voice_path   string = DB_path + "voices/"
var LogFile_path string = DB_path + "log.txt"
var Udon_path    string = DB_path + "udonese/"
var UdonGroupID  int64  = -1002205667779

var IsDebugMode bool = false
var Private_log bool = false

// 禁用 Inline 默认函数，启用后会提示 Inline 用法
var Inline_NoDefaultHandler bool = false

var BotMe *models.User // 用于存储 bot 信息


var InlineSubCommandSymbol string = "+"
var InlinePaginationSymbol string = "-"
var InlineResultsPerPage   int    = 50 // maxinum is 50, see https://core.telegram.org/bots/api#answerinlinequery

var Cache_path string = "./cache/"

var StickerCache_path    string = Cache_path + "sticker/"
var StickerCachePNG_path string = Cache_path + "sticker_png/"
var StickerCacheZip_path string = Cache_path + "sticker_zip/"

// type AdditionalDataPath struct {
// 	Voice   string
// 	Udonese string
// }

// var AdditionalDatas_paths = &AdditionalDataPath{
// 	Voice: Voice_path,
// 	Udonese: Udon_path,
// }


type SignalChannel struct {
	Database_save          chan bool
	AdditionalDatas_reload chan bool
	WorkDone               chan bool
}

var SignalsChannel = SignalChannel{
	Database_save:          make(chan bool),
	AdditionalDatas_reload: make(chan bool),
	WorkDone:               make(chan bool),
}
