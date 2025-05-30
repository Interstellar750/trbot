package consts

import (
	"github.com/go-telegram/bot/models"
)

var IsDebugMode bool

var WebhookListenPort string = "localhost:2847"

var YAMLDataBasePath string = "./db_yaml/"
var YAMLFileName     string = "metadata.yaml"

var CacheDirectory   string = "./cache/"
var LogFilePath      string = YAMLDataBasePath + "log.txt"

var BotMe *models.User // 用于存储 bot 信息
