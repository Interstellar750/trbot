package configs

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"trbot/utils/yaml"
	"unicode"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

func InitBot(ctx context.Context) error {
	var initFuncs = []func(ctx context.Context)error{
		readConfig,
		readBotToken,
		readEnvironment,
	}

	godotenv.Load()
	for _, initfunc := range initFuncs {
		err := initfunc(ctx)
		if err != nil { return err }
	}

	return nil
}

// 从 yaml 文件读取配置文件
func readConfig(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	// 先检查一下环境变量里有没有指定配置目录
	configPathToFile := os.Getenv("CONFIG_PATH_TO_FILE")
	configDirectory  := os.Getenv("CONFIG_DIRECTORY")

	if configPathToFile != "" {
		// 检查配置文件是否存在
		if _, err := os.Stat(configPathToFile); err != nil {
			if os.IsNotExist(err) {
				// 如果配置文件不存在，就以默认配置的方式创建一份
				logger.Warn().
					Str("configPathToFile", configPathToFile).
					Msg("The config file does not exist. creating...")
				err = yaml.SaveYAML(configPathToFile, CreateDefaultConfig())
				if err != nil {
					logger.Error().
						Err(err).
						Msg("Create default config failed")
					return err
				} else {
					logger.Warn().
						Str("configPathToFile", configPathToFile).
						Msg("The config file is created, please fill the bot token and restart")
					// 创建完成目录就跳到下方读取配置文件
					// 默认配置文件没 bot token 的错误就留后面处理
				}
			} else {
				// 读取配置文件时的其他错误
				logger.Error().
					Err(err).
					Str("configPathToFile", configPathToFile).
					Msg("Read config file failed")
				return err
			}
		}

		// 读取配置文件
		ConfigPath = configPathToFile
		logger.Info().
			Msg("Read config success from `CONFIG_PATH_TO_FILE` environment")
		return yaml.LoadYAML(configPathToFile, &BotConfig)
	} else if configDirectory != "" {
		// 检查目录是否存在
		if _, err := os.Stat(configDirectory); err != nil {
			if os.IsNotExist(err) {
				// 目录不存在则创建
				logger.Warn().
					Str("configDirectory", configDirectory).
					Msg("Config directory does not exist, creating...")
				err = os.MkdirAll(configDirectory, 0755)
				if err != nil {
					logger.Error().
						Err(err).
						Str("configDirectory", configDirectory).
						Msg("Create config directory failed")
					return err
				}
				// 如果不出错，到这里会跳到下方的读取配置文件部分
			} else {
				// 读取目录时的其他错误
				logger.Error().
					Err(err).
					Str("configDirectory", configDirectory).
					Msg("Read config directory failed")
				return err
			}
		}

		// 使用默认的配置文件名，把目标配置文件路径补全
		targetConfigPath := filepath.Join(configDirectory, "config.yaml")

		// 检查目录配置文件是否存在
		if _, err := os.Stat(targetConfigPath); err != nil {
			if os.IsNotExist(err) {
				// 用户指定目录的话，还是不创建配置文件了，提示用户想要自定义配置文件名的话，需要设定另外一个环境变量
				logger.Warn().
					Str("configDirectory", configDirectory).
					Msg("No configuration file named `config.yaml` was found in this directory, If you want to set a specific config file name, set the `CONFIG_PATH_TO_FILE` environment variable")
				return err
			} else {
				// 读取目标配置文件路径时的其他错误
				logger.Error().
					Err(err).
					Str("targetConfigPath", targetConfigPath).
					Msg("Read target config file path failed")
				return err
			}
		}

		// 读取配置文件
		ConfigPath = configPathToFile
		logger.Info().
			Msg("Read config path success from `CONFIG_DIRECTORY` environment")
		return yaml.LoadYAML(targetConfigPath, &BotConfig)
	} else {
		// 没有指定任何环境变量，就读取默认的路径
		if _, err := os.Stat(ConfigPath); err != nil {
			if os.IsNotExist(err) {
				// 如果配置文件不存在，就以默认配置的方式创建一份
				logger.Warn().
					Str("defaultConfigPath", ConfigPath).
					Msg("The default config file does not exist. creating...")
				err = yaml.SaveYAML(ConfigPath, CreateDefaultConfig())
				if err != nil {
					logger.Error().
						Err(err).
						Str("defaultConfigPath", ConfigPath).
						Msg("Create default config file failed")
					return err
				} else {
					logger.Warn().
						Str("defaultConfigPath", ConfigPath).
						Msg("Default config file is created, please fill the bot token and restart.")
					// 创建完成目录就跳到下方读取配置文件
					// 默认配置文件没 bot token 的错误就留后面处理
				}
			} else {
				// 读取配置文件时的其他错误
				logger.Error().
					Err(err).
					Str("defaultConfigPath", ConfigPath).
					Msg("Read default config file failed")
				return err
			}
		}

		logger.Info().
			Str("defaultConfigPath", ConfigPath).
			Msg("Read config file from default path")
		return yaml.LoadYAML(ConfigPath, &BotConfig)
	}
}

// 查找 bot token，优先级为 环境变量 > .env 文件 > 配置文件
func readBotToken(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	botToken := os.Getenv("BOT_TOKEN")
	if botToken != "" {
		BotConfig.BotToken = botToken
		logger.Info().
			Str("botTokenID", showBotID()).
			Msg("Get token from environment or .env file")
		return nil
	}

	// 从 yaml 配置文件中读取
	if BotConfig.BotToken != "" {
		logger.Info().
			Str("botTokenID", showBotID()).
			Msg("Get token from config file")
		return nil
	}

	// 都不存在，提示错误
	logger.Warn().
		Msg("No bot token in environment, .env file and yaml config file, try create a bot from https://t.me/@botfather https://core.telegram.org/bots/tutorial#obtain-your-bot-token and fill it")
	return fmt.Errorf("no bot token")

}

func readEnvironment(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	if os.Getenv("DEBUG") != "" {
		BotConfig.LogLevel = "debug"
		logger.Warn().
			Msg("The DEBUG environment variable is set")
	}

	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel != "" {
		BotConfig.LogLevel = logLevel
		logger.Warn().
			Str("logLevel", logLevel).
			Msg("Get log level from environment")
	}

	return nil
}

func showBotID() string {
	var botID string
	for _, char := range BotConfig.BotToken {
		if unicode.IsDigit(char) {
			botID += string(char)
		} else {
			break // 遇到非数字字符停止
		}
	}
	return botID
}

func ShowConfigs(ctx context.Context) {
	logger := zerolog.Ctx(ctx)

	if len(BotConfig.AllowedUpdates) != 0 {
		logger.Info().
			Strs("allowedUpdates", BotConfig.AllowedUpdates).
			Msg("Allowed updates list is set")
	}

	if len(BotConfig.AdminIDs) != 0 {
		logger.Info().
			Ints64("AdminIDs", BotConfig.AdminIDs).
			Msg("Admin list is set")
	}

	if BotConfig.LogChatID != 0 {
		logger.Info().
			Int64("LogChatID", BotConfig.LogChatID).
			Msg("Enabled log to chat")
	}

	if BotConfig.InlineDefaultHandler == "" {
		logger.Info().
			Msg("Inline default handler is not set, default show all commands")
	}

	if BotConfig.InlineSubCommandSymbol == "" {
		BotConfig.InlineSubCommandSymbol = "+"
		logger.Info().
			Msg("Inline sub command symbol is not set, set it to `+` (plus sign)")
	}

	if BotConfig.InlinePaginationSymbol == "" {
		BotConfig.InlinePaginationSymbol = "-"
		logger.Info().
			Msg("Inline pagination symbol is not set, set it to `-` (minus sign)")
	}

	if BotConfig.InlineResultsPerPage == 0 {
		BotConfig.InlineResultsPerPage = 50
		logger.Info().
			Msg("Inline results per page number is not set, set it to 50")
	} else if BotConfig.InlineResultsPerPage < 1 || BotConfig.InlineResultsPerPage > 50 {
		logger.Warn().
			Int("invalidNumber", BotConfig.InlineResultsPerPage).
			Msg("Inline results per page number is invalid, set it to 50")
		BotConfig.InlineResultsPerPage = 50
	}

	logger.Info().
		Str("DefaultHandler",   BotConfig.InlineDefaultHandler).
		Str("SubCommandSymbol", BotConfig.InlineSubCommandSymbol).
		Str("PaginationSymbol", BotConfig.InlinePaginationSymbol).
		Int("ResultsPerPage",   BotConfig.InlineResultsPerPage).
		Msg("Inline mode config has been read")
}
