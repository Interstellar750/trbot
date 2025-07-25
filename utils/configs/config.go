package configs

import (
	"fmt"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

// default "./config.yaml", can be changed by env
var ConfigPath string = "./config.yaml"
var BotConfig config

type config struct {
	// bot config
	BotToken   string `yaml:"BotToken"`
	WebhookURL string `yaml:"WebhookURL"`

	// log
	LogLevel     string `yaml:"LogLevel"` // `trace` `debug` `info` `warn` `error` `fatal` `panic`, default "info"
	LogFileLevel string `yaml:"LogFileLevel"`
	LogChatID    int64  `yaml:"LogChatID"`

	// admin
	AdminIDs []int64 `yaml:"AdminIDs"`

	// redis database
	RedisURL        string `yaml:"RedisURL"`
	RedisPassword   string `yaml:"RedisPassword"`
	RedisDatabaseID int    `yaml:"RedisDatabaseID"`

	// inline mode config
	InlineDefaultHandler   string `yaml:"InlineDefaultHandler"`   // Leave empty to show inline menu
	InlineSubCommandSymbol string `yaml:"InlineSubCommandSymbol"` // default is "+"
	InlinePaginationSymbol string `yaml:"InlinePaginationSymbol"` // default is "-"
	InlineResultsPerPage   int    `yaml:"InlineResultsPerPage"`   // default 50, maxinum 50, see https://core.telegram.org/bots/api#answerinlinequery

	AllowedUpdates bot.AllowedUpdates `yaml:"AllowedUpdates"`

	FFmpegPath string `yaml:"FFmpegPath"`
}

func (c config)LevelForZeroLog(forLogFile bool) zerolog.Level {
	var levelText string
	if forLogFile {
		levelText = c.LogFileLevel
	} else {
		levelText = c.LogLevel
	}

	switch strings.ToLower(levelText) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "info":
		return zerolog.InfoLevel
	case "warn":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		if forLogFile {
			fmt.Printf("Unknown log level [ %s ], using warn level for log file", c.LogLevel)
			return zerolog.WarnLevel
		} else {
			fmt.Printf("Unknown log level [ %s ], using info level for console", c.LogLevel)
			return zerolog.InfoLevel
		}
	}
}

func CreateDefaultConfig() config {
	return config{
		BotToken: "REPLACE_THIS_USE_YOUR_BOT_TOKEN",
		LogLevel: "info",
		LogFileLevel: "warn",

		InlineSubCommandSymbol: "+",
		InlinePaginationSymbol: "-",
		InlineResultsPerPage: 50,
		AllowedUpdates: bot.AllowedUpdates{
			models.AllowedUpdateMessage,
			models.AllowedUpdateEditedMessage,
			models.AllowedUpdateChannelPost,
			models.AllowedUpdateEditedChannelPost,
			models.AllowedUpdateInlineQuery,
			models.AllowedUpdateChosenInlineResult,
			models.AllowedUpdateCallbackQuery,
		},
	}
}
