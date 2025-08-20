package configs

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog"
)

var YAMLDatabaseDir string = "data/"
var YAMLFileName    string = "metadata.yaml"
var CacheDir        string = "cache/"

var ConfigPath string = "./config.yaml" // can be changed by env

var BotConfig = config{
	WebhookListenAddress: "localhost:2847",

	LogLevel:     "info",
	LogFileLevel: "info",
	LogFilePath:  "log.txt",

	InlineSubCommandSymbol: "+",
	InlinePaginationSymbol: "-",
	InlineCategorySymbol:   "=",
	InlineResultsPerPage:   50,
}

type config struct {
	// bot config
	BotToken             string `yaml:"BotToken"`
	WebhookURL           string `yaml:"WebhookURL"`
	WebhookListenAddress string `yaml:"WebhookListenAddress"`

	// log
	LogLevel     string `yaml:"LogLevel"`     // `trace` `debug` `info` `warn` `error` `fatal` `panic`, default "info"
	LogFileLevel string `yaml:"LogFileLevel"`
	LogFilePath  string `yaml:"LogFilePath"`
	LogChatID    int64  `yaml:"LogChatID"`

	// admin
	AdminIDs []int64 `yaml:"AdminIDs"`

	// redis database
	RedisURL        string `yaml:"RedisURL"`
	RedisPassword   string `yaml:"RedisPassword"`
	RedisDatabaseID int    `yaml:"RedisDatabaseID"`

	// inline mode config
	InlineDefaultHandler   string `yaml:"InlineDefaultHandler"`   // leave empty to show inline menu
	InlineSubCommandSymbol string `yaml:"InlineSubCommandSymbol"` // default is "+"
	InlinePaginationSymbol string `yaml:"InlinePaginationSymbol"` // default is "-"
	InlineCategorySymbol   string `yaml:"InlineCategorySymbol"`   // default is "="
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
			fmt.Printf("Unknown log level [ %s ], using info level for log file", c.LogLevel)
		} else {
			fmt.Printf("Unknown log level [ %s ], using info level for console", c.LogLevel)
		}
		return zerolog.InfoLevel
	}
}

func GetPluginDir(pluginName string) string {
	return filepath.Join(YAMLDatabaseDir, pluginName)
}
