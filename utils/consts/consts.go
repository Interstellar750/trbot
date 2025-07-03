package consts

import (
	"github.com/go-telegram/bot/models"
)

var WebhookListenPort string = "localhost:2847"

var YAMLDataBaseDir string = "./db_yaml/"
var YAMLFileName     string = "metadata.yaml"

var CacheDirectory   string = "./cache/"
var LogFilePath      string = YAMLDataBaseDir + "log.txt"

var BotMe *models.User // 用于存储 bot 信息

var Commit  string
var Branch  string
var Version string
var BuildAt string
var BuildOn string
var Changes string // uncommit files when build
