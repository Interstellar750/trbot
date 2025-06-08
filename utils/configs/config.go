package configs

import (
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

// default "./config.yaml", can be changed by env
var ConfigPath string = "./config.yaml"

type config struct {
	// bot config
	BotToken   string `yaml:"BotToken"`
	WebhookURL string `yaml:"WebhookURL"`

	// log
	LogLevel  string `yaml:"LogLevel"` // `trace` `debug` `info` `warn` `error` `fatal` `panic`, default "info"
	LogChatID int64  `yaml:"LogChatID"`

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
}

func (c config)LevelForZeroLog() zerolog.Level {
	switch strings.ToLower(c.LogLevel) {
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
		log.Printf("Unknown log level [ %s ], using info level", c.LogLevel)
		return zerolog.InfoLevel
	}
}

func CreateDefaultConfig() config {
	return config{
		BotToken: "REPLACE_THIS_USE_YOUR_BOT_TOKEN",
		LogLevel: "info",

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

var BotConfig config
