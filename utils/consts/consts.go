package consts

import (
	"context"
	"runtime"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var WebhookListenPort string = "localhost:2847"

var YAMLDataBasePath string = "./db_yaml/"
var YAMLFileName     string = "metadata.yaml"

var CacheDirectory   string = "./cache/"
var LogFilePath      string = YAMLDataBasePath + "log.txt"

var BotMe *models.User // 用于存储 bot 信息

var Commit       string
var Branch       string
var Version      string
var BuildTime    string
var BuildMachine string
var Changes      string // uncommit files when build

func ShowConsts(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	if BuildTime == "" {
		logger.Warn().
			Str("runtime", runtime.Version()).
			Str("logLevel", zerolog.GlobalLevel().String()).
			Str("error", "Remind: You are using a version without build info").
			Msg("trbot")
	} else {
		logger.Info().
			Str("commit",    Commit).
			Str("branch",    Branch).
			Str("version",   Version).
			Str("buildTime", BuildTime).
			Str("changes",   Changes).
			Str("runtime",   runtime.Version()).
			Str("logLevel",  zerolog.GlobalLevel().String()).
			Msg("trbot")
	}
}
