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

var isCanReInit    bool = true
var isSuccessInit  bool = false
var isListening    bool = false
var isCanListening bool = false
var isLoginFailed  bool = false

var hasHandlerByChatID bool

var resetListenTicker = make(chan bool)

var tsData      TSServerQuery
var privateOpts *handler_params.Message

var checkcount         int
var isMessagePinned    bool
var reconnectMessageID int
// var oldMessageIDs      []int

type TSServerQuery struct {
	// get Name And Password in TeamSpeak 3 Client -> `Tools` -> `ServerQuery Login`
	URL                   string `yaml:"URL"`
	Name                  string `yaml:"Name"`
	Password              string `yaml:"Password"`
	GroupID               int64  `yaml:"GroupID"`
	PollingInterval       int    `yaml:"PollingInterval"`
	SendMessageMode       bool   `yaml:"SendMessageMode"`
	AutoDeleteMessage     bool   `yaml:"AutoDeleteMessage"`
	DeleteTimeoutInMinute int    `yaml:"DeleteTimeoutInMinute"`
	PinMessageMode        bool   `yaml:"PinMessageMode"`
	PinnedMessageID       int    `yaml:"PinnedMessageID"`
}

type OnlineClient struct {
	Username string
	JoinTime time.Time
}


func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "teamspeak",
		Func: func(ctx context.Context, thebot *bot.Bot) error{
			if initTeamSpeak(ctx) {
				isSuccessInit = true
				// 需要以群组 ID 来触发 handler 来获取 opts
				plugin_utils.AddHandlerByMessageChatIDHandlers(plugin_utils.ByMessageChatIDHandler{
					ForChatID:      tsData.GroupID,
					PluginName:     "teamspeak_get_opts",
					MessageHandler: getOptsHandler,
				})
				hasHandlerByChatID = true
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
		Name:        "TeamSpeak 检测用户变动",
		Description: "注意：使用此功能需要先在配置文件中手动填写配置文件\n\n使用 /ts3 命令随时查看服务器在线用户和监听状态，监听轮询时间为每 5 秒检测一次，若无法与服务器取得连接，将会自动尝试重新连接",
	})

	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:  "ts3",
		MessageHandler: showStatus,
	})

	plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
		CallbackDataPrefix: "teamspeak",
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
		isCanReInit = false
		return false
	}

	// 如果服务器地址为空不允许重新启动
	if tsData.URL == "" {
		logger.Error().
			Str("path", tsDataPath).
			Msg("No URL in config")
		tsErr = fmt.Errorf("no URL in config")
		isCanReInit = false
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
		isCanReInit = false
		return false
	} else {
		err = tsClient.Login(tsData.Name, tsData.Password)
		if err != nil {
			logger.Error().
				Err(err).
				Str("path", tsDataPath).
				Msg("Failed to login to server")
			tsErr = fmt.Errorf("failed to login to server: %w", err)
			isLoginFailed = true
			return false
		} else {
			isLoginFailed = false
		}
	}

	// 检查要设定通知的群组 ID 是否存在
	if tsData.GroupID == 0 {
		logger.Error().
			Str("path", tsDataPath).
			Msg("No GroupID in config")
		tsErr = fmt.Errorf("no GroupID in config")
		isCanReInit = false
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
			err = yaml.SaveYAML(tsDataPath, &TSServerQuery{
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
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.SaveYAML(tsDataPath, &tsData)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsDataPath).
			Msg("Failed to save teamspeak data")
		return fmt.Errorf("failed to save teamspeak data: %w", err)
	}

	return nil
}

// 用于首次初始化成功时只要对应群组有任何消息，都能自动获取 privateOpts 用来定时发送消息，并开启监听协程
func getOptsHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	if !isListening && isCanReInit && opts.Message.Chat.ID == tsData.GroupID {
		privateOpts = opts
		isCanListening = true
		logger.Debug().Msg("success get opts by handler")
		if !isLoginFailed {
			go listenUserStatus(opts.Ctx)
			logger.Debug().Msg("success start listen user status")
		}
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

	// 如果首次初始化没成功，没有添加根据群组 ID 来触发的 handler，用户发送 /ts3 后可以通过这个来自动获取 opts 并启动监听
	if !isListening && isCanReInit && opts.Message.Chat.ID == tsData.GroupID {
		privateOpts = opts
		isCanListening = true
		logger.Debug().Msg("success get opts by showStatus")
		if !isLoginFailed {
			go listenUserStatus(opts.Ctx)
			logger.Debug().Msg("success start listen user status")
		}
	}

	var pendingMessage string

	if isSuccessInit && isCanListening {
		onlineClient, err := tsClient.Server.ClientList()
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to get online client")
			handlerErr.Addf("failed to get online client: %w", err)
			pendingMessage = fmt.Sprintf("连接到 teamspeak 服务器发生错误:\n<blockquote expandable>%s</blockquote>", err)
		} else {
			pendingMessage += fmt.Sprintln("在线客户端:")
			var userCount int
			for _, client := range onlineClient {
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
		if isCanReInit {
			initCtx, cancel := context.WithTimeout(opts.Ctx, time.Second * 10)
			defer cancel()
			if initTeamSpeak(initCtx) {
				isSuccessInit = true
				if !isListening && !isLoginFailed && opts.Message.Chat.ID == tsData.GroupID {
					go listenUserStatus(opts.Ctx)
					logger.Debug().
						Msg("Start listening user status")
				}
				if isListening {
					resetListenTicker <- true
					pendingMessage = "尝试重新初始化成功，现可正常运行"
				} else {
					pendingMessage = "当前用户列表不可用，正在等待对应群组中的消息..."
				}
			} else {
				handlerErr.Addf("failed to reinit teamspeak plugin: %w", tsErr)
				if isListening {
					pendingMessage += "尝试重新初始化失败，您可以使用 /ts3 命令来尝试手动初始化，或等待自动重连"
				} else {
					pendingMessage += "尝试重新初始化失败，您需要在服务器在线时手动使用 /ts3 命令来尝试初始化"
				}
			}
		} else {
			pendingMessage += "这是一个无法恢复的错误，您可能需要联系机器人管理员"
		}
	}

	var buttons models.ReplyMarkup
	if opts.Message.Chat.ID == tsData.GroupID && contain.Int64(opts.Message.From.ID, utils.GetChatAdminIDs(opts.Ctx, opts.Thebot, tsData.GroupID)...) || contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...) {
		buttons = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text:         "管理此功能",
			CallbackData: "teamspeak",
		}}}}
	}

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
	if tsData.PollingInterval == 0 { tsData.PollingInterval = 5 }
	listenTicker := time.NewTicker(time.Second * time.Duration(tsData.PollingInterval))
	defer listenTicker.Stop()

	// if tsData.DeleteTimeoutInMinute == 0 { tsData.DeleteTimeoutInMinute = 10 }
	// deleteMessageTicker := time.NewTicker(time.Second * time.Duration(tsData.DeleteTimeoutInMinute))
	// defer deleteMessageTicker.Stop()

	if hasHandlerByChatID {
		hasHandlerByChatID = false
		// 获取到 privateOpts 后删掉 handler by chatID
		plugin_utils.RemoveHandlerByMessageChatIDHandler(tsData.GroupID, "teamspeak_get_opts")
	}

	// 取消置顶上一次的置顶消息
	if tsData.PinnedMessageID != 0 {
		_, err := privateOpts.Thebot.UnpinChatMessage(ctx, &bot.UnpinChatMessageParams{
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
		tsData.PinnedMessageID = 0
		saveTeamspeakData(ctx)
	}

	var retryCount int
	var checkFailedCount int
	var beforeOnlineClient []OnlineClient

	for {
		select {
		case <-resetListenTicker:
			listenTicker.Reset(time.Second * time.Duration(tsData.PollingInterval))
			isCanListening = true
			retryCount = 0
		// case <-deleteMessageTicker.C:
		case <-listenTicker.C:
			if isSuccessInit && isCanListening {
				beforeOnlineClient = checkOnlineClientChange(ctx, &checkFailedCount, beforeOnlineClient)
			} else {
				logger.Info().
					Int("retryCount", retryCount).
					Msg("try reconnect...")
				// 出现错误时，先降低 ticker 速度，然后尝试重新初始化
				if retryCount < 15 { retryCount++ }
				listenTicker.Reset(time.Duration(retryCount) * 20 * time.Second)
				initCtx, cancel := context.WithTimeout(ctx, time.Second * 10)
				if initTeamSpeak(initCtx) {
					isSuccessInit  = true
					isCanListening = true
					// 重新初始化成功则恢复 ticker 速度
					retryCount = 1
					listenTicker.Reset(time.Second * time.Duration(tsData.PollingInterval))
					logger.Info().Msg("reconnect success")
					botMessage, err := privateOpts.Thebot.SendMessage(privateOpts.Ctx, &bot.SendMessageParams{
						ChatID:    tsData.GroupID,
						Text:      "已成功与服务器重新建立连接",
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", tsData.GroupID).
							Str("content", "success reconnect to server notice").
							Msg(flaterr.SendMessage.Str())
					} else {
						time.Sleep(time.Second * 3)
						var deleteMessageIDs []int = []int{botMessage.ID}
						if reconnectMessageID != 0 {
							deleteMessageIDs = []int{botMessage.ID, reconnectMessageID}
							reconnectMessageID = 0
						}
						_, err = privateOpts.Thebot.DeleteMessages(privateOpts.Ctx, &bot.DeleteMessagesParams{
							ChatID:     tsData.GroupID,
							MessageIDs: deleteMessageIDs,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Int64("chatID", tsData.GroupID).
								Int("messageID", botMessage.ID).
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
						Msg("connect failed")
				}
				cancel()
			}
		}
	}
}

func checkOnlineClientChange(ctx context.Context, count *int, before []OnlineClient) []OnlineClient {
	var nowOnlineClient []OnlineClient
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	onlineClient, err := tsClient.Server.ClientList()
	if err != nil {
		*count++
		logger.Error().
			Err(err).
			Int("failedCount", *count).
			Msg("Failed to get online client")
		if err.Error() == "not connected" {
			*count = 0
			isCanListening = false
		}
		if *count >= 5 {
			*count = 0
			isCanListening = false
			botMessage, err := privateOpts.Thebot.SendMessage(privateOpts.Ctx, &bot.SendMessageParams{
				ChatID:    tsData.GroupID,
				Text:      "已连续五次检查在线客户端失败，开始尝试自动重连",
				ParseMode: models.ParseModeHTML,
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
		return before
	} else {
		*count = 0
		for _, client := range onlineClient {
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
			notifyClientChange(added, removed)
		}
		if tsData.PinMessageMode {
			if tsData.PinnedMessageID == 0 {
				message, err := privateOpts.Thebot.SendMessage(privateOpts.Ctx, &bot.SendMessageParams{
					ChatID:    tsData.GroupID,
					Text:      "开始监听 Teamspeak 3 用户状态",
					ParseMode: models.ParseModeHTML,
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
				tsData.PinnedMessageID = message.ID
				err = yaml.SaveYAML(tsDataPath, &tsData)
				if err != nil {
					logger.Error().
						Err(err).
						Str("path", KeywordDataPath).
						Msg("Failed to save teamspeak data")
				} else {
					// 置顶消息提醒
					ok, err := privateOpts.Thebot.PinChatMessage(privateOpts.Ctx, &bot.PinChatMessageParams{
						ChatID:              tsData.GroupID,
						MessageID:           tsData.PinnedMessageID,
						DisableNotification: true,
					})
					if ok {
						isMessagePinned = true
						// 删除置顶消息提示
						plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
							PluginName:       "remove pin message notice",
							ChatType:         privateOpts.Message.Chat.Type,
							MessageType:      message_utils.PinnedMessage,
							ForChatID:        tsData.GroupID,
							AllowAutoTrigger: true,
							MessageHandler:   func(opts *handler_params.Message) error {
								if opts.Message.PinnedMessage != nil {
									if opts.Message.PinnedMessage.Message.ID == tsData.PinnedMessageID {
										_, err := opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
											ChatID:    tsData.GroupID,
											MessageID: opts.Message.ID,
										})
										// 不管成功与否，都注销这个 handler
										plugin_utils.RemoveHandlerByMessageTypeHandler(models.ChatTypeSupergroup, message_utils.PinnedMessage, tsData.GroupID, "remove pin message notice")
										return err
									}
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
			changePinnedMessage(nowOnlineClient, added, removed)
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

func notifyClientChange(add, remove []string) {
	var pendingMessage string
	logger := zerolog.Ctx(privateOpts.Ctx).
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

	msg, err := privateOpts.Thebot.SendMessage(privateOpts.Ctx, &bot.SendMessageParams{
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
		go deleteOldMessage(msg.ID)
	}
}

func changePinnedMessage(online []OnlineClient, add, remove []string) {
	logger := zerolog.Ctx(privateOpts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	// var checkcount int static

	// 没有新加入和离开用户，等待一阵子后再更新用户在线时间
	if len(add) + len(remove) == 0 && checkcount < 12 {
		checkcount++
		return
	} else {
		checkcount = 0
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

	_, err := privateOpts.Thebot.EditMessageText(privateOpts.Ctx, &bot.EditMessageTextParams{
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

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: utils.TextForTrueOrFalse(tsData.PinMessageMode, "✅", "") + "显示在置顶消息",
		CallbackData: "teamspeak_pinmessage",
	}})
	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "关闭菜单",
		CallbackData: "delete_this_message",
	}},)

	return &models.InlineKeyboardMarkup{ InlineKeyboard: buttons }
}

// This is not a good solution
func deleteOldMessage(msgID int) {
	if tsData.DeleteTimeoutInMinute == 0 { tsData.DeleteTimeoutInMinute = 10 }
	time.Sleep(time.Minute * time.Duration(tsData.DeleteTimeoutInMinute))
	_, err := privateOpts.Thebot.DeleteMessage(privateOpts.Ctx, &bot.DeleteMessageParams{
		ChatID: tsData.GroupID,
		MessageID: msgID,
	})
	if err != nil {
		zerolog.Ctx(privateOpts.Ctx).Error().
			Err(err).
			Str("pluginName", "teamspeak3").
			Str(utils.GetCurrentFuncName()).
			Int("messageID", msgID).
			Int("deleteMessageMinute", tsData.DeleteTimeoutInMinute).
			Str("content", "teamspeak user change notify").
			Msg(flaterr.DeleteMessage.Str())
	}
}
