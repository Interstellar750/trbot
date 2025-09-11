package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
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

var tsClient *ts3.Client
var tsErr     error

var tsDataPath  string = filepath.Join(configs.YAMLDatabaseDir, "teamspeak/", configs.YAMLFileName)
var botNickName string = "trbot_teamspeak_plugin"

var tsData      TSConfig
var botInstance *bot.Bot

var resetListenTicker = make(chan bool)

var isSuccessInit  bool
var isListening    bool
var isCanListening bool

var isMessagePinned    bool
var reconnectMessageID int

type TSConfig struct {
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
				go listenUserStatus(ctx)
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

	// 读取配置文件
	err := readTeamspeakData(ctx)
	if err != nil {
		tsErr = fmt.Errorf("failed to read teamspeak config data: %w", err)
		return false
	}

	// 如果服务器地址为空不允许重新启动
	if tsData.URL == "" {
		logger.Error().
			Str("path", tsDataPath).
			Msg("No URL in config")
		tsErr = fmt.Errorf("no URL in config")
		return false
	} else {
		if tsClient != nil { tsClient.Close() }
		tsClient, err = ts3.NewClient(tsData.URL)
		if err != nil {
			logger.Error().
				Err(err).
				Str("path", tsDataPath).
				Msg("Failed to connect to server")
			tsErr = fmt.Errorf("failed to connnect to server: %w", err)
			return false
		}
	}

	// ServerQuery 账号名或密码为空也不允许重新启动
	if tsData.Name == "" || tsData.Password == "" {
		logger.Error().
			Str("path", tsDataPath).
			Msg("No Name/Password in config")
		tsErr = fmt.Errorf("no Name/Password in config")
		return false
	} else {
		err = tsClient.Login(tsData.Name, tsData.Password)
		if err != nil {
			logger.Error().
				Err(err).
				Str("path", tsDataPath).
				Msg("Failed to login to server")
			tsErr = fmt.Errorf("failed to login to server: %w", err)
			return false
		}
	}

	// 检查要设定通知的群组 ID 是否存在
	if tsData.GroupID == 0 {
		logger.Error().
			Str("path", tsDataPath).
			Msg("No GroupID in config")
		tsErr = fmt.Errorf("no GroupID in config")
		return false
	}

	// 显示服务端版本测试一下连接
	v, err := tsClient.Version()
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsDataPath).
			Msg("Failed to get server version")
		tsErr = fmt.Errorf("failed to get server version: %w", err)
		return false
	} else {
		logger.Info().
			Str("version", v.Version).
			Str("platform", v.Platform).
			Int("build", v.Build).
			Msg("TeamSpeak server connected")
	}

	// 切换默认虚拟服务器
	err = tsClient.Use(1)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to switch server")
		tsErr = fmt.Errorf("failed to switch server: %w", err)
		return false
	}

	// 改一下 bot 自己的 nickname，使得在检测用户列表时默认不显示自己
	m, err := tsClient.Whoami()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get bot info")
		tsErr = fmt.Errorf("failed to get bot info: %w", err)
		return false
	} else if m != nil && m.ClientName != botNickName {
		// 当 bot 自己的 nickname 不等于配置文件中的 nickname 时，才进行修改
		err = tsClient.SetNick(botNickName)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to set bot nickname")
			tsErr = fmt.Errorf("failed to set nickname: %w", err)
			return false
		}
	}

	// 没遇到不可重新初始化的部分则返回初始化成功
	isSuccessInit = true
	return true
}

func readTeamspeakData(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(tsDataPath, &tsData)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", tsDataPath).
				Msg("Not found teamspeak config file. Created new one")
			err = yaml.SaveYAML(tsDataPath, &TSConfig{
				PollingInterval:     5,
				SendMessageMode:     true,
				DeleteTimeoutInMinute: 10,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", tsDataPath).
					Msg("Failed to create empty config")
				return fmt.Errorf("failed to create empty config: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", tsDataPath).
				Msg("Failed to read config file")

			// 读取配置文件内容失败也不允许重新启动
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return err
}

func saveTeamspeakData(ctx context.Context) error {
	err := yaml.SaveYAML(tsDataPath, &tsData)
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Str("pluginName", "teamspeak3").
			Str(utils.GetCurrentFuncName()).
			Err(err).
			Str("path", tsDataPath).
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

	var handlerErr     flaterr.MultErr
	var pendingMessage string

	// 正常运行就输出用户列表，否则发送错误信息
	if isSuccessInit && isCanListening {
		onlineClients, err := tsClient.Server.ClientList()
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
		pendingMessage = fmt.Sprintf("初始化 teamspeak 插件时发生了一些错误:\n<blockquote expandable>%s</blockquote>\n\n", tsErr)
		timeoutCtx, cancel := context.WithTimeout(opts.Ctx, time.Second * 10)
		if initTeamSpeak(timeoutCtx) {
			if isListening {
				resetListenTicker <- true
			} else {
				go listenUserStatus(opts.Ctx)
			}
			pendingMessage = "尝试重新初始化成功，现可正常运行"
		} else {
			handlerErr.Addf("failed to reinit teamspeak plugin: %w", tsErr)
			if isListening {
				pendingMessage += "尝试重新初始化失败，您可以使用 /ts3 命令来尝试重新连接或等待自动重连"
			} else {
				pendingMessage += "尝试重新初始化失败，由于监听服务未在运行，您需要手动使用 /ts3 命令来尝试重新连接"
			}
		}
		cancel()
	}

	var buttons models.ReplyMarkup
	// 显示管理按钮
	if opts.Message.Chat.ID == tsData.GroupID && contain.Int64(opts.Message.From.ID, utils.GetChatAdminIDs(opts.Ctx, opts.Thebot, tsData.GroupID)...) || contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...) {
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

func listenUserStatus(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	isListening = true
	isCanListening = true

	defer func() {
		isListening = false
		isCanListening = false
		logger.Warn().Msg("listenUserStatus goroutine stopped")
	}()

	listenTicker := time.NewTicker(time.Second * time.Duration(tsData.PollingInterval))
	defer listenTicker.Stop()

	if tsData.PollingInterval == 0 { tsData.PollingInterval = 5 }
	if tsData.DeleteTimeoutInMinute == 0 { tsData.DeleteTimeoutInMinute = 10 }
	// 取消置顶上一次的置顶消息
	if tsData.PinnedMessageID != 0 {
		if tsData.DeleteOldPinnedMessage {
			_, err := botInstance.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID:    tsData.GroupID,
				MessageID: tsData.PinnedMessageID,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", tsData.GroupID).
					Int("messageID", tsData.PinnedMessageID).
					Str("content", "latest pinned online client status").
					Msg(flaterr.DeleteMessage.Str())
			}
		} else {
			_, err := botInstance.UnpinChatMessage(ctx, &bot.UnpinChatMessageParams{
				ChatID:    tsData.GroupID,
				MessageID: tsData.PinnedMessageID,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", tsData.GroupID).
					Int("messageID", tsData.PinnedMessageID).
					Str("content", "latest pinned online client status").
					Msg(flaterr.UnpinChatMessage.Str())
			}
		}

		tsData.PinnedMessageID = 0
		err := saveTeamspeakData(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to save teamspeak data after delete or unpin message")
		}
	}

	var retryCount       int
	var checkCount       int
	var checkFailedCount int

	var beforeOnlineClient []OnlineClient

	for {
		select {
		case <-resetListenTicker:
			listenTicker.Reset(time.Second * time.Duration(tsData.PollingInterval))
			isCanListening = true
			retryCount = 0
		case <-listenTicker.C:
			if isSuccessInit && isCanListening {
				beforeOnlineClient = checkOnlineClientChange(ctx, &checkCount, &checkFailedCount, beforeOnlineClient)
			} else {
				logger.Info().
					Int("retryCount", retryCount).
					Msg("try reconnect...")
				// 出现错误时，先降低 ticker 速度，然后尝试重新初始化
				if retryCount < 15 {
					retryCount++
					listenTicker.Reset(time.Duration(retryCount) * 20 * time.Second)
				}
				timeoutCtx, cancel := context.WithTimeout(ctx, time.Second * 10)
				if initTeamSpeak(timeoutCtx) {
					// 重新初始化成功则恢复 ticker 速度
					listenTicker.Reset(time.Second * time.Duration(tsData.PollingInterval))
					isCanListening = true
					retryCount = 0
					logger.Info().Msg("reconnect success")
					botMessage, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: tsData.GroupID,
						Text:   "已成功与服务器重新建立连接",
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", tsData.GroupID).
							Str("content", "success reconnect to server notice").
							Msg(flaterr.SendMessage.Str())
					} else {
						time.Sleep(time.Second * 5)
						var deleteMessageIDs []int = []int{botMessage.ID}
						if reconnectMessageID != 0 {
							deleteMessageIDs = []int{botMessage.ID, reconnectMessageID}
							reconnectMessageID = 0
						}
						_, err = botInstance.DeleteMessages(ctx, &bot.DeleteMessagesParams{
							ChatID:     tsData.GroupID,
							MessageIDs: deleteMessageIDs,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Int64("chatID", tsData.GroupID).
								Ints("messageIDs", deleteMessageIDs).
								Str("content", "success reconnect to server notice").
								Msg(flaterr.DeleteMessages.Str())
						}
					}
				} else {
					// 无法成功则等待下一个周期继续尝试
					logger.Warn().
						Err(tsErr).
						Int("retryCount", retryCount).
						Int("nextRetry", (retryCount) * 20).
						Msg("reconnect failed")
				}
				cancel()
			}
		}
	}
}

func checkOnlineClientChange(ctx context.Context, checkCount, errCount *int, before []OnlineClient) []OnlineClient {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	onlineClients, err := tsClient.Server.ClientList()
	if err != nil {
		*errCount++
		logger.Error().
			Err(err).
			Int("failedCount", *errCount).
			Msg("Failed to get online client")
		// 连不上服务器直接尝试重连
		if err.Error() == "not connected" {
			*errCount = 0
			isCanListening = false
		}
		// 不是连不上服务器，则累积到五次后再重连
		if *errCount >= 5 {
			*errCount = 0
			isCanListening = false
			botMessage, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: tsData.GroupID,
				Text:   "已连续五次检查在线客户端失败，开始尝试自动重连",
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", tsData.GroupID).
					Str("content", "failed to check online client 5 times, start auto reconnect").
					Msg(flaterr.SendMessage.Str())
			} else {
				reconnectMessageID = botMessage.ID
			}
		}
		// 获取失败时暂时返回之前的在线用户
		return before
	} else {
		var nowOnlineClient []OnlineClient
		*errCount = 0 // 重置失败计数
		for _, client := range onlineClients {
			if client.Nickname == botNickName { continue }
			var isExist bool
			for _, user := range before {
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
		added, removed := diffSlices(before, nowOnlineClient)
		if tsData.SendMessageMode && len(added) + len(removed) > 0 {
			logger.Debug().
				Int("clientJoin", len(added)).
				Int("clientLeave", len(removed)).
				Msg("online client change detected")
			notifyClientChange(ctx, added, removed)
		}
		if tsData.PinMessageMode {
			if tsData.PinnedMessageID == 0 {
				message, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
					ChatID:              tsData.GroupID,
					Text:                "开始监听 Teamspeak 3 用户状态",
					DisableNotification: true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("chatID", tsData.GroupID).
						Str("content", "start listen teamspeak user changes").
						Msg(flaterr.SendMessage.Str())
					return nowOnlineClient
				}
				tsData.PinnedMessageID = message.ID // 虽然后面可能会因为权限问题没法成功置顶，不过为了防止重复发送，所以假设它已经被置顶了
				err = yaml.SaveYAML(tsDataPath, &tsData)
				if err != nil {
					logger.Error().
						Err(err).
						Str("path", KeywordDataPath).
						Msg("Failed to save teamspeak data after pin message")
				} else {
					// 置顶消息提醒
					ok, err := botInstance.PinChatMessage(ctx, &bot.PinChatMessageParams{
						ChatID:              tsData.GroupID,
						MessageID:           message.ID,
						DisableNotification: true,
					})
					if ok {
						isMessagePinned = true
						// 删除置顶消息提示
						plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
							PluginName:       "remove pin message notice",
							ChatType:         message.Chat.Type,
							MessageType:      message_utils.PinnedMessage,
							ForChatID:        tsData.GroupID,
							AllowAutoTrigger: true,
							MessageHandler:   func(opts *handler_params.Message) error {
								if opts.Message.PinnedMessage != nil && opts.Message.PinnedMessage.Message.ID == tsData.PinnedMessageID {
									_, err := opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
										ChatID:    tsData.GroupID,
										MessageID: opts.Message.ID,
									})
									// 不管成功与否，都注销这个 handler
									plugin_utils.RemoveHandlerByMessageTypeHandler(models.ChatTypeSupergroup, message_utils.PinnedMessage, tsData.GroupID, "remove pin message notice")
									return err
								}
								return nil
							},
						})
					} else {
						logger.Error().
							Err(err).
							Int64("chatID", tsData.GroupID).
							Str("content", "listen teamspeak user changes").
							Msg(flaterr.PinChatMessage.Str())
					}
				}

			}
			changePinnedMessage(ctx, checkCount, nowOnlineClient, added, removed)
		}
		return nowOnlineClient
	}
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

func notifyClientChange(ctx context.Context, add, remove []string) {
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
		ChatID: tsData.GroupID,
		Text:   pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", tsData.GroupID).
			Str("content", "teamspeak user change notify").
			Msg(flaterr.SendMessage.Str())
	} else {
		// oldMessageIDs = append(oldMessageIDs, msg.ID)
		go deleteOldMessage(ctx, msg.ID)
	}
}

func changePinnedMessage(ctx context.Context, checkCount *int, online []OnlineClient, add, remove []string) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	// var checkcount int static

	// 没有新加入和离开用户，等待一阵子后再更新用户在线时间
	if len(add) + len(remove) == 0 && *checkCount < 12 {
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

	if !isMessagePinned {
		pendingMessage += "<blockquote expandable>无法置顶用户列表消息，请检查机器人是否拥有对应的权限，您也可以手动置顶此消息</blockquote>"
	}

	_, err := botInstance.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID: tsData.GroupID,
		MessageID: tsData.PinnedMessageID,
		Text:   pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", tsData.GroupID).
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

	if contain.Int64(opts.CallbackQuery.From.ID, utils.GetChatAdminIDs(opts.Ctx, opts.Thebot, tsData.GroupID)...) || contain.Int64(opts.CallbackQuery.From.ID, configs.BotConfig.AdminIDs...) {
		var needEdit bool = true
		var needSave bool = false

		switch opts.CallbackQuery.Data {
		case "teamspeak_pinmessage":
			if tsData.PinMessageMode && !tsData.SendMessageMode {
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
				tsData.PinMessageMode = !tsData.PinMessageMode
				needSave = true
			}
		case "teamspeak_pinmessage_deletepinedmessage":
			tsData.DeleteOldPinnedMessage = !tsData.DeleteOldPinnedMessage
			needSave = true
		case "teamspeak_sendmessage":
			if tsData.SendMessageMode && !tsData.PinMessageMode {
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
				tsData.SendMessageMode = !tsData.SendMessageMode
				needSave = true
			}
		case "teamspeak_sendmessage_autodelete":
			tsData.AutoDeleteMessage = !tsData.AutoDeleteMessage
			needSave = true
		}

		if needEdit {
			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text:      "选择通知用户变动的方式",
				ReplyMarkup: buildTeamspeakConfigKeyboard(),
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

func buildTeamspeakConfigKeyboard() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	if tsData.SendMessageMode {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text: utils.TextForTrueOrFalse(tsData.SendMessageMode, "✅ ", "") + "发送消息通知",
				CallbackData: "teamspeak_sendmessage",
			},
			{
				Text: utils.TextForTrueOrFalse(tsData.AutoDeleteMessage, "✅ ", "") + "自动删除旧消息",
				CallbackData: "teamspeak_sendmessage_autodelete",
			},
		})
	} else {
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: utils.TextForTrueOrFalse(tsData.SendMessageMode, "✅ ", "") + "发送消息通知",
			CallbackData: "teamspeak_sendmessage",
		}})
	}

	if tsData.PinMessageMode {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text: utils.TextForTrueOrFalse(tsData.PinMessageMode, "✅ ", "") + "显示在置顶消息",
				CallbackData: "teamspeak_pinmessage",
			},
			{
				Text: utils.TextForTrueOrFalse(tsData.DeleteOldPinnedMessage, "✅ ", "") + "删除旧的置顶消息",
				CallbackData: "teamspeak_pinmessage_deletepinedmessage",
			},
		})
	} else {
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: utils.TextForTrueOrFalse(tsData.PinMessageMode, "✅ ", "") + "显示在置顶消息",
			CallbackData: "teamspeak_pinmessage",
		}})
	}
	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "关闭菜单",
		CallbackData: "delete_this_message",
	}},)

	return &models.InlineKeyboardMarkup{ InlineKeyboard: buttons }
}

// This is not a good solution
func deleteOldMessage(ctx context.Context, msgID int) {
	if tsData.DeleteTimeoutInMinute == 0 { tsData.DeleteTimeoutInMinute = 10 }
	time.Sleep(time.Minute * time.Duration(tsData.DeleteTimeoutInMinute))
	_, err := botInstance.DeleteMessage(ctx, &bot.DeleteMessageParams{
		ChatID: tsData.GroupID,
		MessageID: msgID,
	})
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Err(err).
			Str("pluginName", "teamspeak3").
			Str(utils.GetCurrentFuncName()).
			Int("messageID", msgID).
			Int("deleteMessageMinute", tsData.DeleteTimeoutInMinute).
			Str("content", "teamspeak user change notify").
			Msg(flaterr.DeleteMessage.Str())
	}
}
