package configs

import (
	"context"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"trbot/utils/yaml"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

func InitBot() (context.Context, context.CancelFunc, zerolog.Logger) {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack // set stack trace func
	logger := zerolog.New(zerolog.ConsoleWriter{ Out: os.Stdout, TimeFormat: "15:04:05"}).With().Timestamp().Logger()

	godotenv.Load()

	var cfg config

	/* read config file */ {
		configPath := os.Getenv("CONFIG_PATH")
		configDir  := os.Getenv("CONFIG_DIR")
		if configPath != "" {
			// 检查配置文件是否存在
			if _, err := os.Stat(configPath); err != nil {
				if os.IsNotExist(err) {
					// 如果配置文件不存在，就以默认配置的方式创建一份
					logger.Warn().
						Str("configPath", configPath).
						Msg("The config file does not exist. creating a default config file...")
					err = yaml.SaveYAML(configPath, BotConfig)
					if err != nil {
						logger.Fatal().
							Err(err).
							Msg("Failed to create default config file")
					} else {
						logger.Warn().
							Str("configPath", configPath).
							Msg("The default config file is created, please fill the bot token and restart")
					}
				} else {
					logger.Fatal().
						Err(err).
						Str("configPath", configPath).
						Msg("Failed to read config file info")
				}
			}

			// 读取配置文件
			ConfigPath = configPath
			err := yaml.LoadYAML(configPath, &cfg)
			if err != nil {
				logger.Fatal().
					Err(err).
					Str("configPath", configPath).
					Msg("Failed to read config file")
			} else {
				logger.Info().Msg("Read config success from `CONFIG_PATH` environment")
			}
		} else if configDir != "" {
			if _, err := os.Stat(configDir); err != nil {
				if os.IsNotExist(err) {
					logger.Warn().
						Str("configDir", configDir).
						Msg("The config directory does not exist, creating...")
					err = os.MkdirAll(configDir, 0755)
					if err != nil {
						logger.Fatal().
							Err(err).
							Str("configDir", configDir).
							Msg("Failed to create config directory")
					}
				} else {
					logger.Fatal().
						Err(err).
						Str("configDir", configDir).
						Msg("Failed to read config directory info")
				}
			}

			// 使用默认的配置文件名，把目标配置文件路径补全
			targetConfigPath := filepath.Join(configDir, "config.yaml")

			// 检查目录配置文件是否存在
			if _, err := os.Stat(targetConfigPath); err != nil {
				if os.IsNotExist(err) {
					// 用户指定目录的话，还是不创建配置文件了，提示用户想要自定义配置文件名的话，需要设定另外一个环境变量
					logger.Fatal().
						Str("configDir", configDir).
						Msg("No config file named `config.yaml` was found in this directory, If you want to set a specific config file name, set the `CONFIG_PATH` environment variable")
				} else {
					// 读取目标配置文件路径时的其他错误
					logger.Fatal().
						Err(err).
						Str("targetConfigPath", targetConfigPath).
						Msg("Failed to read target config file info")
				}
			}

			// 读取配置文件
			ConfigPath = configPath
			err := yaml.LoadYAML(targetConfigPath, &cfg)
			if err != nil {
				logger.Fatal().
					Err(err).
					Str("targetConfigPath", targetConfigPath).
					Msg("Failed to read config file")
			} else {
				logger.Info().Msg("Read config file success from `CONFIG_DIR` environment")
			}
		} else {
			// 没有指定任何环境变量，就读取默认的路径
			if _, err := os.Stat(ConfigPath); err != nil {
				if os.IsNotExist(err) {
					// 如果配置文件不存在，就以默认配置的方式创建一份
					logger.Warn().
						Str("defaultConfigPath", ConfigPath).
						Msg("The default config file does not exist. creating a default config file...")
					err = yaml.SaveYAML(ConfigPath, BotConfig)
					if err != nil {
						logger.Fatal().
							Err(err).
							Str("defaultConfigPath", ConfigPath).
							Msg("Failed to create default config file")
					} else {
						logger.Warn().
							Str("defaultConfigPath", ConfigPath).
							Msg("The default config file is created, please fill the bot token and restart")
					}
				} else {
					logger.Fatal().
						Err(err).
						Str("defaultConfigPath", ConfigPath).
						Msg("Failed to read default config file")
				}
			}

			err := yaml.LoadYAML(ConfigPath, &cfg)
			if err != nil {
				logger.Fatal().
					Err(err).
					Str("defaultConfigPath", ConfigPath).
					Msg("Failed to read default config file")
			} else {
				logger.Info().Msg("Read config file success")
			}
		}
	}

	/* logger */ {
		if os.Getenv("DEBUG") != "" {
			BotConfig.LogLevel = "debug"
			logger.Warn().
				Msg("Get `DEBUG` flag from environment, set log level to `debug`")
		} else {
			logLevel := os.Getenv("LOG_LEVEL")
			if logLevel != "" {
				BotConfig.LogLevel = logLevel
				logger.Info().
					Str("LogLevel", BotConfig.LogLevel).
					Msg("Get log level from environment")
			} else if cfg.LogLevel != "" {
				BotConfig.LogLevel = cfg.LogLevel
				logger.Info().
					Str("LogLevel", BotConfig.LogLevel).
					Msg("Get log level from config file")
			} else {
				logger.Info().
					Str("LogLevel", BotConfig.LogLevel).
					Msg("No log level config, use default level")
			}
		}

		logFilePath := os.Getenv("LOG_FILE_PATH")
		if logFilePath != "" {
			BotConfig.LogFilePath = logFilePath
			logger.Warn().
				Str("LogFilePath", BotConfig.LogFilePath).
				Msg("Get log file path from environment")
		} else if cfg.LogFilePath != "" {
			BotConfig.LogFilePath = cfg.LogFilePath
			logger.Info().
				Str("LogFilePath", BotConfig.LogFilePath).
				Msg("Get log file path from config file")
		} else {
			logger.Info().
				Str("LogFilePath", BotConfig.LogFilePath).
				Msg("No log file path config, use default path")
		}

		if BotConfig.LogFilePath != "disable" {
			logFileLevel := os.Getenv("LOG_FILE_LEVEL")
			if logFileLevel != "" {
				BotConfig.LogFileLevel = logFileLevel
				logger.Info().
					Str("LogFileLevel", BotConfig.LogFileLevel).
					Msg("Get log file level from environment")
			} else if cfg.LogFileLevel != "" {
				BotConfig.LogFileLevel = cfg.LogFileLevel
				logger.Info().
					Str("LogFileLevel", BotConfig.LogFileLevel).
					Msg("Get log file level from config file")
			} else {
				logger.Info().
					Str("LogFileLevel", BotConfig.LogFileLevel).
					Msg("No log file level config, use default level")
			}

			file, err := os.OpenFile(BotConfig.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				logger.Error().
					Err(err).
					Str("LogFilePath", BotConfig.LogFilePath).
					Msg("Failed to open log file, use console log writer only")
			} else {
				logger = zerolog.New(zerolog.MultiLevelWriter(
					&zerolog.ConsoleWriter{
						Out:        os.Stdout,
						TimeFormat: "15:04:05",
					},
					&zerolog.FilteredLevelWriter{
						Writer: zerolog.MultiLevelWriter(file),
						Level:  BotConfig.LevelForZeroLog(true),
					},
				)).With().Timestamp().Logger()

				logger.Info().
					Str("LogFilePath", BotConfig.LogFilePath).
					Str("LogFileLevel", BotConfig.LogFileLevel).
					Msg("Use mult log writer")
			}
		} else {
			logger.Warn().Msg("LogFilePath is set to `disable`, use console log writer only")
		}

		logChatID := os.Getenv("LOG_CHAT_ID")
		if logChatID != "" {
			logChatID_int64, err := strconv.ParseInt(logChatID, 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Str("LogChatID", logChatID).
					Msg("Failed to parse log chat ID from environment")
			} else {
				BotConfig.LogChatID = logChatID_int64
			}
			logger.Warn().
				Int64("LogChatID", BotConfig.LogChatID).
				Msg("Get log chat ID from environment")
		} else if cfg.LogChatID != 0 {
			BotConfig.LogChatID = cfg.LogChatID
			logger.Info().
				Int64("LogChatID", BotConfig.LogChatID).
				Msg("Get log chat id from config")
		}
	}

	/* bot token and mode */ {
		botToken := os.Getenv("BOT_TOKEN")
		if botToken != "" {
			BotConfig.BotToken = botToken
			logger.Info().
				Str("botTokenID", strings.Split(BotConfig.BotToken, ":")[0]).
				Msg("Get token from environment")
		} else if cfg.BotToken != "" {
			BotConfig.BotToken = cfg.BotToken
			logger.Info().
				Str("botTokenID", strings.Split(BotConfig.BotToken, ":")[0]).
				Msg("Get token from config file")
		} else {
			logger.Fatal().Msg("No bot token in environment, `.env` file and YAML config file, try create a bot from https://t.me/@botfather https://core.telegram.org/bots/tutorial#obtain-your-bot-token and fill it")
		}

		webhookURL := os.Getenv("WEBHOOK_URL")
		if webhookURL != "" {
			BotConfig.WebhookURL = webhookURL
			logger.Warn().
				Str("WebhookURL", BotConfig.WebhookURL).
				Msg("Get Webhook URL from environment")
		} else if cfg.WebhookURL != "" {
			BotConfig.WebhookURL = cfg.WebhookURL
			logger.Info().
				Str("WebhookURL", BotConfig.WebhookURL).
				Msg("Get Webhook URL from config file")
		} else {
			logger.Info().Msg("No Webhook URL in environment `.env` file and YAML config file, using getUpdate mode")
		}

		if BotConfig.WebhookURL != "" {
			webhookAddress := os.Getenv("WEBHOOK_ADDR")
			if webhookAddress != "" {
				BotConfig.WebhookListenAddress = webhookAddress
				logger.Warn().
					Str("WebhookListenAddress", BotConfig.WebhookListenAddress).
					Msg("Get Webhook Listen Address from environment")
			} else if cfg.WebhookListenAddress != "" {
				BotConfig.WebhookListenAddress = cfg.WebhookListenAddress
				logger.Warn().
					Str("WebhookListenAddress", BotConfig.WebhookListenAddress).
					Msg("Get Webhook Listen Address from config file")
			} else {
				logger.Info().
					Str("WebhookListenAddress", BotConfig.WebhookListenAddress).
					Msg("Webhook listen address is not set, use default address")
			}
		}

		if len(cfg.AllowedUpdates) != 0 {
			BotConfig.AllowedUpdates = cfg.AllowedUpdates
			logger.Warn().
				Strs("AllowedUpdates", BotConfig.AllowedUpdates).
				Msg("Allowed updates list is set")
		} else {
			logger.Info().Msg("Allowed updates list is empty, you will receive all updates except `chat_member`, `message_reaction` and `message_reaction_count`. See `allowed_updates` in https://core.telegram.org/bots/api#getting-updates for details")
		}

		if len(cfg.AdminIDs) != 0 {
			BotConfig.AdminIDs = cfg.AdminIDs
			logger.Info().
				Ints64("AdminIDs", BotConfig.AdminIDs).
				Msg("Admin ID list is set")
		} else {
			logger.Warn().
				Msg("Admin ID list is not set, fill it in config file to use admin only features")
		}

		if cfg.InlineDefaultHandler != "" {
			BotConfig.InlineDefaultHandler = cfg.InlineDefaultHandler
		} else {
			logger.Info().Msg("Inline default handler is not set, default show all commands")
		}

		if cfg.InlineSubCommandSymbol != "" {
			BotConfig.InlineSubCommandSymbol = cfg.InlineSubCommandSymbol
		} else {
			logger.Info().
				Str("SubCommandSymbol", BotConfig.InlineSubCommandSymbol).
				Msg("Inline sub command symbol is not set, use default symbol")
		}

		if cfg.InlinePaginationSymbol != "" {
			BotConfig.InlinePaginationSymbol = cfg.InlinePaginationSymbol
		} else {
			logger.Info().
				Str("PaginationSymbol", BotConfig.InlinePaginationSymbol).
				Msg("Inline pagination symbol is not set, use default symbol")
		}

		if cfg.InlineCategorySymbol != "" {
			BotConfig.InlineCategorySymbol = cfg.InlineCategorySymbol
		} else {
			logger.Info().
				Str("CategorySymbol", BotConfig.InlineCategorySymbol).
				Msg("Inline category symbol is not set, use default symbol")
		}

		if cfg.InlineResultsPerPage != 0 {
			if cfg.InlineResultsPerPage < 1 || cfg.InlineResultsPerPage > 50 {
				logger.Warn().
					Int("invalidNumber", BotConfig.InlineResultsPerPage).
					Int("ResultsPerPage", cfg.InlineResultsPerPage).
					Msg("Inline results per page number is invalid, use default number")
			} else {
				BotConfig.InlineResultsPerPage = cfg.InlineResultsPerPage
			}
		} else {
			logger.Info().
				Int("ResultsPerPage", BotConfig.InlineResultsPerPage).
				Msg("Inline results per page number is not set, use default number")
		}

		logger.Info().
			Str("DefaultHandler",   BotConfig.InlineDefaultHandler).
			Str("SubCommandSymbol", BotConfig.InlineSubCommandSymbol).
			Str("CategorySymbol",   BotConfig.InlineCategorySymbol).
			Str("PaginationSymbol", BotConfig.InlinePaginationSymbol).
			Int("ResultsPerPage",   BotConfig.InlineResultsPerPage).
			Msg("Inline mode config")
	}

	/* build info */ {
		if BuildAt == "" {
			logger.Warn().
				Str("error", "Remind: You are using a version without build info").
				Str("runtime", runtime.Version()).
				Str("logLevel", BotConfig.LogLevel).
				Msg("trbot")
		} else {
			logger.Info().
				Str("commit",   Commit).
				Str("branch",   Branch).
				Str("version",  Version).
				Str("buildAt",  BuildAt).
				Str("buildOn",  BuildOn).
				Str("changes",  Changes).
				Str("runtime",  runtime.Version()).
				Str("logLevel", BotConfig.LogLevel).
				Msg("trbot")
		}
	}

	/* other */ {
		if cfg.RedisURL != "" {
			BotConfig.RedisURL = cfg.RedisURL
			BotConfig.RedisPassword = cfg.RedisPassword
			BotConfig.RedisDatabaseID = cfg.RedisDatabaseID
			logger.Info().
				Str("RedisURL", BotConfig.RedisURL).
				Int("RedisDatabaseID", BotConfig.RedisDatabaseID).
				Msg("Get Redis URL and Database ID from config")
		}

		FFmpegPath := os.Getenv("FFMPEG_PATH")
		if FFmpegPath != "" {
			BotConfig.FFmpegPath = FFmpegPath
			logger.Info().
				Str("FFmpegPath", BotConfig.FFmpegPath).
				Msg("Get FFmpeg path from environment")
		} else if cfg.FFmpegPath != "" {
			BotConfig.FFmpegPath = cfg.FFmpegPath
			logger.Info().
				Str("FFmpegPath", BotConfig.FFmpegPath).
				Msg("Get FFmpeg path from config")
		} else {
			logger.Warn().
				Msg("No FFmpeg path in environment `.env` file and YAML config file, you will not be able to use some features that depend on it")
		}
	}

	// attach logger into ctx
	ctx = logger.WithContext(ctx)

	return ctx, cancel, logger
}
