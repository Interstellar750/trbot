package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"trbot/utils/type/message_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/multiplay/go-ts3"
	"github.com/rs/zerolog"
)

var tsConfig ServerConfig
var tsErr    error

var tsConfigPath string = filepath.Join(configs.YAMLDatabaseDir, "teamspeak/", configs.YAMLFileName)
var botNickName  string = "trbot_teamspeak_plugin"

var botInstance *bot.Bot

type ServerConfig struct {
	rw sync.RWMutex
	c *ts3.Client
	s Status

	// get `Name` And `Password` in `TeamSpeak 3 Client` -> `Tools` -> `ServerQuery Login`
	URL                    string `yaml:"URL"`
	Name                   string `yaml:"Name"`
	Password               string `yaml:"Password"`
	GroupID                int64  `yaml:"GroupID"`
	PollingInterval        int    `yaml:"PollingInterval"`
	SendMessageMode        bool   `yaml:"SendMessageMode"`
	AutoDeleteMessage      bool   `yaml:"AutoDeleteMessage"`
	DeleteTimeoutInMinute  int    `yaml:"DeleteTimeoutInMinute"`
	PinMessageMode         bool   `yaml:"PinMessageMode"`
	DeleteOldPinnedMessage bool   `yaml:"DeleteOldPinnedMessage"`
	PinnedMessageID        int    `yaml:"PinnedMessageID"`
}

func (sc *ServerConfig) Connect(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Int64("GroupID", sc.GroupID).
		Logger()

	var err error
	sc.rw.Lock()
	defer sc.rw.Unlock()

	// 如果服务器地址为空不允许重新启动
	if sc.URL == "" {
		logger.Error().
			Str("path", tsConfigPath).
			Msg("No URL in config")
		return fmt.Errorf("no URL in config")
	}
	if sc.c != nil {
		logger.Info().Msg("Closing client...")
		sc.c.Close()
		sc.c = nil
	}

	logger.Info().Msg("Starting client...")
	sc.c, err = ts3.NewClient(sc.URL)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsConfigPath).
			Msg("Failed to connect to server")
		return fmt.Errorf("failed to connnect to server: %w", err)
	}

	logger.Info().Msg("Checking credentials...")
	// ServerQuery 账号名或密码为空也不允许重新启动
	if sc.Name == "" || sc.Password == "" {
		logger.Error().
			Str("path", tsConfigPath).
			Msg("No Name/Password in config")
		return fmt.Errorf("no Name/Password in config")
	}

	logger.Info().Msg("Logining...")
	err = sc.c.Login(sc.Name, sc.Password)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsConfigPath).
			Msg("Failed to login to server")
		return fmt.Errorf("failed to login to server: %w", err)
	}


	logger.Info().Msg("Checking Group ID...")
	// 检查要设定通知的群组 ID 是否存在
	if sc.GroupID == 0 {
		logger.Error().
			Str("path", tsConfigPath).
			Msg("No GroupID in config")
		return fmt.Errorf("no GroupID in config")
	}

	logger.Info().Msg("Testing connection...")
	// 显示服务端版本测试一下连接
	v, err := sc.c.Version()
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsConfigPath).
			Msg("Failed to get server version")
		return fmt.Errorf("failed to get server version: %w", err)
	}

	logger.Info().
		Str("version", v.Version).
		Str("platform", v.Platform).
		Int("build", v.Build).
		Msg("TeamSpeak server connected")

	logger.Info().Msg("Switching virtual servers...")

	// 切换默认虚拟服务器
	err = sc.c.Use(1)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to switch server")
		return fmt.Errorf("failed to switch server: %w", err)
	}

	logger.Info().Msg("Checking nickname...")

	// 改一下 bot 自己的 nickname，使得在检测用户列表时默认不显示自己
	m, err := sc.c.Whoami()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get bot info")
		return fmt.Errorf("failed to get bot info: %w", err)
	} else if m != nil && m.ClientName != botNickName {
		logger.Info().Msg("Setting nickname...")
		// 当 bot 自己的 nickname 不等于配置文件中的 nickname 时，才进行修改
		err = sc.c.SetNick(botNickName)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to set bot nickname")
			return fmt.Errorf("failed to set nickname: %w", err)
		}
	}

	logger.Info().Msg("Successfully connected!")

	// 没遇到不可重新初始化的部分则返回初始化成功
	sc.s.IsSuccessInit = true
	return nil
}

func (sc *ServerConfig) CheckClient(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	onlineClients, err := sc.c.Server.ClientList()
	if err != nil {
		sc.s.CheckFailedCount++
		logger.Error().
			Err(err).
			Int("failedCount", sc.s.CheckFailedCount).
			Msg("Failed to get online client")
		// 连不上服务器直接尝试重连
		if err.Error() == "not connected" {
			sc.s.CheckFailedCount = 0
			sc.s.IsCanListening = false
		}
		// 不是连不上服务器，则累积到五次后再重连
		if sc.s.CheckFailedCount >= 5 {
			sc.s.CheckFailedCount = 0
			sc.s.IsCanListening = false
			botMessage, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: sc.GroupID,
				Text:   "已连续五次检查在线客户端失败，开始尝试自动重连",
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", sc.GroupID).
					Str("content", "failed to check online client 5 times, start auto reconnect").
					Msg(flaterr.SendMessage.Str())
			} else {
				sc.s.ReconnectMessageID = botMessage.ID
			}
		}
		return
	} else {
		var nowOnlineClient []OnlineClient
		sc.s.CheckFailedCount = 0 // 重置失败计数
		for _, client := range onlineClients {
			if client.Nickname == botNickName { continue }
			var isExist bool
			for _, user := range sc.s.BeforeOnlineClient {
				if user.Username == client.Nickname {
					nowOnlineClient = append(nowOnlineClient, user)
					isExist = true
				}
			}
			if !isExist {
				nowOnlineClient = append(nowOnlineClient, OnlineClient{
					Username: client.Nickname,
					JoinTime: time.Now(),
				})
			}
		}
		added, removed := diffSlices(sc.s.BeforeOnlineClient, nowOnlineClient)
		if sc.SendMessageMode && len(added) + len(removed) > 0 {
			logger.Debug().
				Int("clientJoin", len(added)).
				Int("clientLeave", len(removed)).
				Msg("online client change detected")
			notifyClientChange(ctx, sc, added, removed)
		}
		if sc.PinMessageMode {
			if sc.PinnedMessageID == 0 {
				message, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:              sc.GroupID,
					Text:                "开始监听 Teamspeak 3 用户状态",
					DisableNotification: true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("chatID", sc.GroupID).
						Str("content", "start listen teamspeak user changes").
						Msg(flaterr.SendMessage.Str())
					return
				}
				sc.PinnedMessageID = message.ID // 虽然后面可能会因为权限问题没法成功置顶，不过为了防止重复发送，所以假设它已经被置顶了
				err = yaml.SaveYAML(tsConfigPath, &tsConfig)
				if err != nil {
					logger.Error().
						Err(err).
						Str("path", KeywordDataPath).
						Msg("Failed to save teamspeak data after pin message")
				} else {
					// 置顶消息提醒
					ok, err := botInstance.PinChatMessage(ctx, &bot.PinChatMessageParams{
						ChatID:              sc.GroupID,
						MessageID:           message.ID,
						DisableNotification: true,
					})
					if ok {
						sc.s.IsMessagePinned = true
						// 删除置顶消息提示
						plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
							PluginName:       "remove pin message notice",
							ChatType:         message.Chat.Type,
							MessageType:      message_utils.PinnedMessage,
							ForChatID:        sc.GroupID,
							AllowAutoTrigger: true,
							MessageHandler:   func(opts *handler_params.Message) error {
								if opts.Message.PinnedMessage != nil && opts.Message.PinnedMessage.Message.ID == sc.PinnedMessageID {
									_, err := opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
										ChatID:    sc.GroupID,
										MessageID: opts.Message.ID,
									})
									// 不管成功与否，都注销这个 handler
									plugin_utils.RemoveHandlerByMessageTypeHandler(models.ChatTypeSupergroup, message_utils.PinnedMessage, sc.GroupID, "remove pin message notice")
									return err
								}
								return nil
							},
						})
					} else {
						logger.Error().
							Err(err).
							Int64("chatID", sc.GroupID).
							Str("content", "listen teamspeak user changes").
							Msg(flaterr.PinChatMessage.Str())
					}
				}

			}
			changePinnedMessage(ctx, sc, &sc.s.CheckCount, nowOnlineClient, added, removed)
		}
		sc.s.BeforeOnlineClient = nowOnlineClient
	}
}

func (sc *ServerConfig) Retry(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	logger.Info().
		Int("retryCount", sc.s.RetryCount).
		Msg("try reconnect...")

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Second * 30)
	defer cancel()
	return sc.Connect(timeoutCtx)
}

func (sc *ServerConfig) BuildConfigKeyboard() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	if sc.SendMessageMode {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text: utils.TextForTrueOrFalse(sc.SendMessageMode, "✅ ", "") + "发送消息通知",
				CallbackData: "teamspeak_sendmessage",
			},
			{
				Text: utils.TextForTrueOrFalse(sc.AutoDeleteMessage, "✅ ", "") + "自动删除旧消息",
				CallbackData: "teamspeak_sendmessage_autodelete",
			},
		})
	} else {
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: utils.TextForTrueOrFalse(sc.SendMessageMode, "✅ ", "") + "发送消息通知",
			CallbackData: "teamspeak_sendmessage",
		}})
	}

	if sc.PinMessageMode {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text: utils.TextForTrueOrFalse(sc.PinMessageMode, "✅ ", "") + "显示在置顶消息",
				CallbackData: "teamspeak_pinmessage",
			},
			{
				Text: utils.TextForTrueOrFalse(sc.DeleteOldPinnedMessage, "✅ ", "") + "删除旧的置顶消息",
				CallbackData: "teamspeak_pinmessage_deletepinedmessage",
			},
		})
	} else {
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: utils.TextForTrueOrFalse(sc.PinMessageMode, "✅ ", "") + "显示在置顶消息",
			CallbackData: "teamspeak_pinmessage",
		}})
	}
	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "关闭菜单",
		CallbackData: "delete_this_message",
	}},)

	return &models.InlineKeyboardMarkup{ InlineKeyboard: buttons }
}

func (sc *ServerConfig) StartListening(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	sc.s.IsListening = true
	sc.s.IsCanListening = true

	defer func() {
		sc.s.IsListening = false
		logger.Warn().Msg("listenUserStatus goroutine stopped")
	}()

	if sc.s.ResetTicker == nil {
		sc.s.ResetTicker = make(chan bool)
	}

	if sc.PollingInterval == 0 {
		sc.PollingInterval = 5
	}
	if sc.DeleteTimeoutInMinute == 0 {
		sc.DeleteTimeoutInMinute = 10
	}
	// 取消置顶上一次的置顶消息
	if sc.PinnedMessageID != 0 {
		if sc.DeleteOldPinnedMessage {
			_, err := botInstance.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    sc.GroupID,
				MessageID: sc.PinnedMessageID,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", sc.GroupID).
					Int("messageID", sc.PinnedMessageID).
					Str("content", "latest pinned online client status").
					Msg(flaterr.DeleteMessage.Str())
			}
		} else {
			_, err := botInstance.UnpinChatMessage(ctx, &bot.UnpinChatMessageParams{
				ChatID:    sc.GroupID,
				MessageID: sc.PinnedMessageID,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", sc.GroupID).
					Int("messageID", sc.PinnedMessageID).
					Str("content", "latest pinned online client status").
					Msg(flaterr.UnpinChatMessage.Str())
			}
		}

		sc.PinnedMessageID = 0
		err := saveTeamspeakData(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to save teamspeak data after delete or unpin message")
		}
	}

	listenTicker := time.NewTicker(time.Second * time.Duration(sc.PollingInterval))
	defer listenTicker.Stop()

	for {
		select {
		case <-sc.s.ResetTicker:
			listenTicker.Reset(time.Second * time.Duration(sc.PollingInterval))
			sc.s.IsCanListening = true
			sc.s.RetryCount = 0
		case <-listenTicker.C:
			if sc.s.IsCanListening {
				sc.CheckClient(ctx)
			} else {
				err := sc.Retry(ctx)
				if err != nil {
					// 出现错误时，先降低 ticker 速度，然后尝试重新初始化
					// 无法成功则等待下一个周期继续尝试
					if sc.s.RetryCount < 15 {
						sc.s.RetryCount++
						listenTicker.Reset(time.Duration(sc.s.RetryCount) * 20 * time.Second)
					}

					logger.Warn().
						Int("retryCount", sc.s.RetryCount).
						Time("nextRetry", time.Now().Add(time.Duration(sc.s.RetryCount) * 20 * time.Second)).
						Msg("reconnect failed")
				} else {
					// 重新初始化成功则恢复 ticker 速度
					listenTicker.Reset(time.Second * time.Duration(sc.PollingInterval))
					sc.s.IsCanListening = true
					sc.s.RetryCount = 0
					logger.Info().Msg("reconnect success")
					botMessage, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: sc.GroupID,
						Text:   "已成功与服务器重新建立连接",
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", sc.GroupID).
							Str("content", "success reconnect to server notice").
							Msg(flaterr.SendMessage.Str())
					} else {
						time.Sleep(time.Second * 5)
						var deleteMessageIDs []int = []int{botMessage.ID}
						if sc.s.ReconnectMessageID != 0 {
							deleteMessageIDs = []int{botMessage.ID, sc.s.ReconnectMessageID}
							sc.s.ReconnectMessageID = 0
						}
						_, err = botInstance.DeleteMessages(ctx, &bot.DeleteMessagesParams{
							ChatID:     sc.GroupID,
							MessageIDs: deleteMessageIDs,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Int64("chatID", sc.GroupID).
								Ints("messageIDs", deleteMessageIDs).
								Str("content", "success reconnect to server notice").
								Msg(flaterr.DeleteMessages.Str())
						}
					}
				}
			}
		}
	}
}

type Status struct {
	IsSuccessInit   bool
	IsListening     bool
	IsCanListening  bool
	IsMessagePinned bool

	ResetTicker (chan bool)

	ReconnectMessageID int

	RetryCount       int
	CheckCount       int
	CheckFailedCount int

	BeforeOnlineClient []OnlineClient
}

type OnlineClient struct {
	Username string
	JoinTime time.Time
}

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "teamspeak",
		Func: func(ctx context.Context, thebot *bot.Bot) error{
			botInstance = thebot
			if initTeamSpeak(ctx) {
				if tsConfig.s.IsSuccessInit {
					go tsConfig.StartListening(ctx)
				}
			}
			return tsErr
		},
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

func initTeamSpeak(ctx context.Context) bool {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	logger.Info().Msg("Reading config file...")

	// 读取配置文件
	err := readTeamspeakData(ctx)
	if err != nil {
		tsErr = fmt.Errorf("failed to read teamspeak config data: %w", err)
		return false
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
		return false
	}

	logger.Info().
		Int64("ChatID", tsConfig.GroupID).
		Msg("Successfully initialized TeamSpeak")

	return true
}

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
				PollingInterval: 10,
				SendMessageMode: true,
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

func showStatus(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var handlerErr flaterr.MultErr
	var pendingMessage string

	// 正常运行就输出用户列表，否则发送错误信息
	if tsConfig.s.IsSuccessInit && tsConfig.s.IsCanListening {
		onlineClients, err := tsConfig.c.Server.ClientList()
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to get online client")
			handlerErr.Addf("failed to get online client: %w", err)
			pendingMessage = fmt.Sprintf("获取服务器用户列表时发生了一些错误:\n<blockquote expandable>%s</blockquote>", err)
		} else {
			pendingMessage += fmt.Sprintln("在线客户端:")
			var userCount int
			for _, client := range onlineClients {
				if client.Nickname == botNickName {
					// 统计用户数量时跳过此机器人
					continue
				}
				pendingMessage += fmt.Sprintf("用户 [ %s ] ", client.Nickname)
				userCount++
				if client.OnlineClientExt != nil {
					pendingMessage += fmt.Sprintf("在线时长 %d", *client.OnlineClientTimes.LastConnected)
				}
				pendingMessage += "\n"
			}
			if userCount == 0 {
				pendingMessage = "当前无用户在线"
			}
		}
	} else {
		timeoutCtx, cancel := context.WithTimeout(opts.Ctx, time.Second * 30)
		err := tsConfig.Connect(timeoutCtx)
		if err != nil {
			pendingMessage = fmt.Sprintf("初始化 teamspeak 插件时发生了一些错误:\n<blockquote expandable>%s</blockquote>\n\n", err)
			handlerErr.Addf("failed to reinit teamspeak plugin: %w", err)
			if tsConfig.s.IsListening {
				pendingMessage += "尝试重新初始化失败，您可以使用 /ts3 命令来尝试重新连接或等待自动重连"
			} else {
				pendingMessage += "尝试重新初始化失败，由于监听服务未在运行，您需要手动使用 /ts3 命令来尝试重新连接"
			}
		} else {
			if tsConfig.s.IsListening  {
				tsConfig.s.ResetTicker <- true
			} else {
				go tsConfig.StartListening(opts.Ctx)
			}
			pendingMessage = "尝试重新初始化成功，现可正常运行"
		}
		cancel()
	}

	var buttons models.ReplyMarkup
	// 显示管理按钮
	if opts.Message.Chat.ID == tsConfig.GroupID && contain.Int64(opts.Message.From.ID, utils.GetChatAdminIDs(opts.Ctx, opts.Thebot, tsConfig.GroupID)...) || contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...) {
		buttons = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text:         "管理此功能",
			CallbackData: "teamspeak",
		}}}}
	}

	// 发送消息
	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:          opts.Message.Chat.ID,
		Text:            pendingMessage,
		ParseMode:       models.ParseModeHTML,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		ReplyMarkup:     buttons,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", opts.Message.Chat.ID).
			Str("content", "teamspeak online client status").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "teamspeak online client status", err)
	}

	return handlerErr.Flat()
}

func diffSlices(before, now []OnlineClient) (added, removed []string) {
	beforeMap := make(map[string]bool)
	nowMap    := make(map[string]bool)

	// 把 A 和 B 转成 map
	for _, item := range before { beforeMap[item.Username] = true }
	for _, item := range now    { nowMap[item.Username]    = true }

	// 删除
	for item := range nowMap {
		if !beforeMap[item] { added = append(added, item)}
	}

	// 新增
	for item := range beforeMap {
		if !nowMap[item] { removed = append(removed, item) }
	}

	return
}

func notifyClientChange(ctx context.Context, server *ServerConfig, add, remove []string) {
	var pendingMessage string
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	if len(add) > 0 {
		pendingMessage += fmt.Sprintln("以下用户进入了服务器:")
		for _, nickname := range add {
			pendingMessage += fmt.Sprintf("用户 [ %s ]\n", nickname)
		}
	}
	if len(remove) > 0 {
		pendingMessage += fmt.Sprintln("以下用户离开了服务器:")
		for _, nickname := range remove {
			pendingMessage += fmt.Sprintf("用户 [ %s ]\n", nickname)
		}
	}

	msg, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: server.GroupID,
		Text:   pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", server.GroupID).
			Str("content", "teamspeak user change notify").
			Msg(flaterr.SendMessage.Str())
	} else {
		// oldMessageIDs = append(oldMessageIDs, msg.ID)
		go deleteOldMessage(ctx, server, msg.ID)
	}
}

func changePinnedMessage(ctx context.Context, server *ServerConfig, checkCount *int, online []OnlineClient, add, remove []string) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	// 没有新加入和离开用户，等待一阵子后再更新用户在线时间
	if len(add) + len(remove) == 0 && *checkCount < (60 / tsConfig.PollingInterval) {
		*checkCount++
		return
	} else {
		*checkCount = 0
	}

	var pendingMessage string = fmt.Sprintf("%s | ", time.Now().Format("15:04"))

	if len(online) > 0 {
		pendingMessage += fmt.Sprintf("有 %d 位用户在线:\n<blockquote>", len(online))
		for _, client := range online {
			pendingMessage += fmt.Sprintf("[ %s ] 已在线 %.1f 分钟\n", client.Username, time.Since(client.JoinTime).Minutes())
		}
		pendingMessage += "</blockquote>\n"
	} else {
		pendingMessage += "没有用户在线\n\n"
	}

	if len(add) + len(remove) > 0 {
		if len(add) > 0 {
			pendingMessage += fmt.Sprintln("以下用户进入了服务器:")
			for _, nickname := range add {
				pendingMessage += fmt.Sprintf("用户 [ %s ]\n", nickname)
			}
		}
		if len(remove) > 0 {
			pendingMessage += fmt.Sprintln("以下用户离开了服务器:")
			for _, nickname := range remove {
				pendingMessage += fmt.Sprintf("用户 [ %s ]\n", nickname)
			}
		}
	}

	if !server.s.IsMessagePinned {
		pendingMessage += "<blockquote expandable>无法置顶用户列表消息，请检查机器人是否拥有对应的权限，您也可以手动置顶此消息</blockquote>"
	}

	_, err := botInstance.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: server.GroupID,
		MessageID: server.PinnedMessageID,
		Text:   pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", server.GroupID).
			Str("content", "teamspeak user change notify").
			Msg(flaterr.EditMessageText.Str())
	}
}

func teamspeakCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Str("callbackQueryData", opts.CallbackQuery.Data).
		Logger()

	var handlerErr flaterr.MultErr

	if contain.Int64(opts.CallbackQuery.From.ID, utils.GetChatAdminIDs(opts.Ctx, opts.Thebot, tsConfig.GroupID)...) || contain.Int64(opts.CallbackQuery.From.ID, configs.BotConfig.AdminIDs...) {
		var needEdit bool = true
		var needSave bool = false

		switch opts.CallbackQuery.Data {
		case "teamspeak_pinmessage":
			if tsConfig.PinMessageMode && !tsConfig.SendMessageMode {
				needEdit = false
				_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "您至少要保留一个检测用户变动的方式",
					ShowAlert:       true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "at least need one notice method").
						Msg(flaterr.AnswerCallbackQuery.Str())
					handlerErr.Addt(flaterr.AnswerCallbackQuery, "at least need one notice method", err)
				}
			} else {
				tsConfig.PinMessageMode = !tsConfig.PinMessageMode
				needSave = true
			}
		case "teamspeak_pinmessage_deletepinedmessage":
			tsConfig.DeleteOldPinnedMessage = !tsConfig.DeleteOldPinnedMessage
			needSave = true
		case "teamspeak_sendmessage":
			if tsConfig.SendMessageMode && !tsConfig.PinMessageMode {
				needEdit = false
				_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "您至少要保留一个检测用户变动的方式",
					ShowAlert:       true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "at least need one notice method").
						Msg(flaterr.AnswerCallbackQuery.Str())
					handlerErr.Addt(flaterr.AnswerCallbackQuery, "at least need one notice method", err)
				}
			} else {
				tsConfig.SendMessageMode = !tsConfig.SendMessageMode
				needSave = true
			}
		case "teamspeak_sendmessage_autodelete":
			tsConfig.AutoDeleteMessage = !tsConfig.AutoDeleteMessage
			needSave = true
		}

		if needEdit {
			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text:      "选择通知用户变动的方式",
				ReplyMarkup: tsConfig.BuildConfigKeyboard(),
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "teamspeak manage keyboard").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "teamspeak manage keyboard", err)
			}
		}

		if needSave {
			err := saveTeamspeakData(opts.Ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to save teamspeak data")
				handlerErr.Addf("failed to save teamspeak data: %w", err)
			}
		}
	} else {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "您没有权限修改此内容",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "no permission to change teamspeak config").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "no permission to change teamspeak config", err)
		}
	}

	return handlerErr.Flat()
}

// This is not a good solution
func deleteOldMessage(ctx context.Context, server *ServerConfig, msgID int) {
	if server.DeleteTimeoutInMinute == 0 { server.DeleteTimeoutInMinute = 10 }
	time.Sleep(time.Minute * time.Duration(server.DeleteTimeoutInMinute))
	_, err := botInstance.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID: server.GroupID,
		MessageID: msgID,
	})
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Err(err).
			Str("pluginName", "teamspeak3").
			Str(utils.GetCurrentFuncName()).
			Int("messageID", msgID).
			Int("deleteMessageMinute", server.DeleteTimeoutInMinute).
			Str("content", "teamspeak user change notify").
			Msg(flaterr.DeleteMessage.Str())
	}
}
