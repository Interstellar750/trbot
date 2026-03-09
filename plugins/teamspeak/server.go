package teamspeak

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/go-telegram/bot"
	ts3 "github.com/jkoenig134/go-ts3"
	"github.com/rs/zerolog"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/plugin_utils"
	"trle5.xyz/trbot/utils/yaml"
)

var tsConfig ServerConfig

var tsConfigPath string = filepath.Join(configs.YAMLDatabaseDir, "teamspeak/", configs.YAMLFileName)
var botNickName  string

var botInstance *bot.Bot

type ServerConfig struct {
	rw sync.RWMutex
	c  ts3.TeamspeakHttpClient
	s  Status

	URL                    string `yaml:"URL"`
	API                    string `yaml:"API"`
	GroupID                int64  `yaml:"GroupID"`
	PollingInterval        int    `yaml:"PollingInterval"`
	SendMessageMode        bool   `yaml:"SendMessageMode"`
	AutoDeleteMessage      bool   `yaml:"AutoDeleteMessage"`
	DeleteTimeoutInMinute  int    `yaml:"DeleteTimeoutInMinute"`
	PinMessageMode         bool   `yaml:"PinMessageMode"`
	DeleteOldPinnedMessage bool   `yaml:"DeleteOldPinnedMessage"`
	PinnedMessageID        int    `yaml:"PinnedMessageID"`
}


func Init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "teamspeak",
		Func: initTeamSpeak,
	})

	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "teamspeak",
		Loader: readTeamspeakData,
		Saver:  saveTeamspeakData,
	})

	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "TeamSpeak",
		Description: "注意：使用此功能需要先在配置文件中手动填写配置文件\n\n此功能可以按照设定好的轮询时间来检查 TeamSpeak 服务器中的用户列表，并可以在用户列表发送变动时在群组中发送提醒\n\n使用 /ts3 命令来随时查看服务器在线用户和监听状态\n支持设定多种提醒方式（更新置顶消息、发送消息）\n自定义配置轮询间隔（单位 秒）\n自定义删除旧通知消息超时（单位 分钟）\n服务器掉线自动重连（若 bot 首次启动未能连接成功，则需要手动发送 /ts3 命令后才可自动重连）",
	})

	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:  "ts3",
		MessageHandler: showStatus,
	})

	plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
		CallbackDataPrefix:   "teamspeak",
		CallbackQueryHandler: teamspeakCallbackHandler,
	})
}

// initTeamSpeak 从 tsConfigPath 读取服务器配置后调用 tsConfig.Connect 尝试连接服务器
func initTeamSpeak(ctx context.Context, thebot *bot.Bot) error {
	// 保存 bot 实例
	botInstance = thebot

	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	logger.Info().Msg("Reading config file...")

	// 读取配置文件
	err := readTeamspeakData(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsConfigPath).
			Msg("Failed to read teamspeak config data")
		return fmt.Errorf("failed to read teamspeak config data: %w", err)
	}

	if tsConfig.s.ResetTicker == nil {
		tsConfig.s.ResetTicker = make(chan bool)
	}

	if tsConfig.PollingInterval == 0 {
		tsConfig.PollingInterval = 5
	}

	if tsConfig.DeleteTimeoutInMinute == 0 {
		tsConfig.DeleteTimeoutInMinute = 10
	}

	if tsConfig.PinMessageMode {
		// 启用功能时检查消息是否存在或是否可编辑
		tsConfig.CheckPinnedMessage(ctx)
	} else if tsConfig.PinnedMessageID != 0 {
		// 禁用功能时且消息 ID 不为 0 时优先解除置顶
		tsConfig.RemovePinnedMessage(ctx, true)
	}

	logger.Info().
		Int64("ChatID", tsConfig.GroupID).
		Msg("Initializing TeamSpeak client...")

	err = tsConfig.Connect(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Int64("ChatID", tsConfig.GroupID).
			Msg("Failed to initialize TeamSpeak client")
		return fmt.Errorf("failed to initialize TeamSpeak client: %w", err)
	}

	logger.Info().
		Int64("ChatID", tsConfig.GroupID).
		Msg("Successfully initialized TeamSpeak")

	return nil
}

// readTeamspeakData 从 tsConfigPath 读取服务器配置并加载到 tsConfig 中
func readTeamspeakData(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(tsConfigPath, &tsConfig)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", tsConfigPath).
				Msg("Not found teamspeak config file. Created new one")
			err = yaml.SaveYAML(tsConfigPath, &ServerConfig{
				PollingInterval:       10,
				SendMessageMode:       true,
				DeleteTimeoutInMinute: 10,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", tsConfigPath).
					Msg("Failed to create empty config")
				return fmt.Errorf("failed to create empty config: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", tsConfigPath).
				Msg("Failed to read config file")

			// 读取配置文件内容失败也不允许重新启动
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return err
}

// saveTeamspeakData 保存 tsConfig 配置到 tsConfigPath 文件中
func saveTeamspeakData(ctx context.Context) error {
	err := yaml.SaveYAML(tsConfigPath, &tsConfig)
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Str("pluginName", "teamspeak3").
			Str(utils.GetCurrentFuncName()).
			Err(err).
			Str("path", tsConfigPath).
			Msg("Failed to save teamspeak data")
		return fmt.Errorf("failed to save teamspeak data: %w", err)
	}

	return nil
}
