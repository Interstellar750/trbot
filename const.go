package main

import "github.com/go-telegram/bot/models"

var botToken string // 全局 bot token

var webhookURL string // Webhook 运行模式下接受请求的 URL 地址
var webhookPort string = "localhost:2847" // Webhook 运行模式下监听的端口

var logChat_ID int64 = -1002499888124 // 用于接收日志的聊天 ID，可以是 用户 群聊 频道
var logMan_IDs []int64 = []int64{1086395364, 2074319561} // 拥有查看日志权限的用户，可设定多个

var metadataFileName string = "metadata.yaml"

var db_path      string = "./db_yaml/"
var voice_path   string = db_path + "voices/"
var logFile_path string = "./log.txt"
var udon_path    string = db_path + "udonese/"
var udonGroupID  int64  = -1002205667779

var IsDebugMode bool = false
var private_log bool = false

var botMe *models.User // 用于存储 bot 信息

var database DataBaseYaml
var AdditionalDatas AdditionalData

var DB_savenow = make(chan bool)
var ADR_reload = make(chan bool)

var InlineSubCommandSymbol string = ":"
var InlinePaginationSymbol string = "-"
var InlineResultsPerPage   int    = 50 // maxinum is 50, see https://core.telegram.org/bots/api#answerinlinequery

var cache_path string = "./cache/"
var stickerCache_path string = cache_path + "sticker/"
