package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"trbot/utils/consts"
	"trbot/utils/handler_structs"
	"trbot/utils/logt"
	"trbot/utils/plugin_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/multiplay/go-ts3"
	"github.com/rs/zerolog"
)

// loginname serveradmin
// password 34lBKaih
// api key BACfzGIw3NCmmmqBXe1HAyi1p8BpCsUUeVb8JZQ
// sever admin privilege key Wwm5L2zNOHNiROqshy22fU9kconSORl+Pdt9aRqN

var tsClient *ts3.Client
var tsErr     error

var tsDataDir   string = filepath.Join(consts.YAMLDataBasePath, "teamspeak/")
var tsDataPath  string = filepath.Join(tsDataDir, consts.YAMLFileName)
var botNickName string = "trbot_teamspeak_plugin"

var isCanReInit    bool = true
var isSuccessInit  bool = false
var isListening    bool = false
var isCanListening bool = false
var isLoginFailed  bool = false

var hasHandlerByChatID bool

var resetListenTicker chan bool = make(chan bool)
var pollingInterval   time.Duration = time.Second * 5

var tsData      TSServerQuery
var privateOpts *handler_structs.SubHandlerParams

type TSServerQuery struct {
	// get Name And Password in TeamSpeak 3 Client -> `Tools`` -> `ServerQuery Login`
	URL      string `yaml:"URL"`
	Name     string `yaml:"Name"`
	Password string `yaml:"Password"`
	GroupID  int64  `yaml:"GroupID"`
}

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "teamspeak",
		Func: func(ctx context.Context) error{
			if initTeamSpeak(ctx) {
				isSuccessInit = true
				// 需要以群组 ID 来触发 handler 来获取 opts
				plugin_utils.AddHandlerByChatIDPlugins(plugin_utils.HandlerByChatID{
					ChatID:      tsData.GroupID,
					PluginName: "teamspeak_get_opts",
					Handler:     getOptsHandler,
				})
				hasHandlerByChatID = true
				return nil
			} else {
				return tsErr
			}
		},
	})

	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "TeamSpeak 检测用户变动",
		Description: "注意：使用此功能需要先在配置文件中手动填写配置文件\n\n使用 /ts3 命令随时查看服务器在线用户和监听状态，监听轮询时间为每 5 秒检测一次，若无法与服务器取得连接，将会自动尝试重新连接",
		ParseMode:   models.ParseModeHTML,
	})

	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "ts3",
		Handler: showStatus,
	})
}

func initTeamSpeak(ctx context.Context) bool {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str("funcName", "initTeamSpeak").
		Logger()
	
	// 判断配置文件是否存在
	_, err := os.Stat(tsDataDir)
	if err != nil {
		if os.IsNotExist(err) {
			// 不存在，创建一份空文件
			logger.Warn().
				Err(err).
				Str("path", tsDataPath).
				Msg("Not found config file. Created new one")
			err = yaml.SaveYAML(tsDataPath, &TSServerQuery{})
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", tsDataPath).
					Msg("Failed to create empty config")
				tsErr = fmt.Errorf("failed to create empty config: %w", err)
			}
			logger.Warn().
				Str("path", tsDataPath).
				Msg("Empty config file created, please fill in the config")
		} else {
			// 文件存在，但是遇到了其他错误
			logger.Error().
				Err(err).
				Str("path", tsDataPath).
				Msg("Failed to read config file")
			tsErr = fmt.Errorf("failed to read config file: %w", err)
		}

		// 无法获取到服务器地址和账号，无法初始化并设定不可重新启动
		isCanReInit = false
		return false
	}

	err = yaml.LoadYAML(tsDataPath, &tsData)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsDataPath).
			Msg("Failed to read config file")
		// 读取配置文件内容失败也不允许重新启动
		tsErr = fmt.Errorf("failed to read config file: %w", err)
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
		tsClient, tsErr = ts3.NewClient(tsData.URL)
		if tsErr != nil {
			logger.Error().
				Err(tsErr).
				Str("path", tsDataPath).
				Msg("Failed to connect to server")
			tsErr = fmt.Errorf("connect error: %w", tsErr)
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
			tsErr = fmt.Errorf("login error: %w", err)
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
		tsErr = fmt.Errorf("switch server error: %w", err)
		return false
	}

	// 改一下 bot 自己的 nickname，使得在检测用户列表时默认不显示自己
	m, err := tsClient.Whoami()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get bot info")
	} else if m != nil && m.ClientName != botNickName {
		// 当 bot 自己的 nickname 不等于配置文件中的 nickname 时，才进行修改
		err = tsClient.SetNick(botNickName)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to set bot nickname")
			tsErr = fmt.Errorf("set nickname error: %w", err)
		}
	}

	// 没遇到不可重新初始化的部分则返回初始化成功
	return true
}

// 用于首次初始化成功时只要对应群组有任何消息，都能自动获取 privateOpts 用来定时发送消息，并开启监听协程
func getOptsHandler(opts *handler_structs.SubHandlerParams) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str("funcName", "getOptsHandler").
		Logger()

	if !isListening && isCanReInit && opts.Update.Message.Chat.ID == tsData.GroupID {
		privateOpts = opts
		isCanListening = true
		logger.Debug().
			Msg("success get opts by handler")
		if !isLoginFailed {
			go listenUserStatus(opts.Ctx)
			logger.Debug().
				Msg("success start listen user status")
		}
	}
	return nil
}

func showStatus(opts *handler_structs.SubHandlerParams) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str("funcName", "showStatus").
		Logger()

	var pendingMessage string

	// 如果首次初始化没成功，没有添加根据群组 ID 来触发的 handler，用户发送 /ts3 后可以通过这个来自动获取 opts 并启动监听
	// if isSuccessInit && !isCanListening && opts.Update != nil && opts.Update.Message != nil && opts.Update.Message.Chat.ID == tsServerQuery.GroupID {
	if !isListening && isCanReInit && opts.Update.Message.Chat.ID == tsData.GroupID {
		privateOpts = opts
		isCanListening = true
		logger.Debug().
			Msg("success get opts by showStatus")
		if !isLoginFailed {
			go listenUserStatus(opts.Ctx)
			logger.Debug().
				Msg("success start listen user status")
		}
		// pendingMessage += fmt.Sprintln("已准备好发送用户状态")
	}

	if isSuccessInit && isCanListening {
		olClient, err := tsClient.Server.ClientList()
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to get online client")
			pendingMessage = fmt.Sprintf("连接到 teamspeak 服务器发生错误:\n<blockquote expandable>%s</blockquote>", err)
		} else {
			pendingMessage += fmt.Sprintln("在线客户端:")
			var userCount int
			for _, n := range olClient {
				if n.Nickname == botNickName {
					// 统计用户数量时跳过此机器人
					continue
				}
				pendingMessage += fmt.Sprintf("用户 [ %s ] ", n.Nickname)
				userCount++
				if n.OnlineClientExt != nil {
					pendingMessage += fmt.Sprintf("在线时长 %d", *n.OnlineClientTimes.LastConnected)
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
			if initTeamSpeak(opts.Ctx) {
				isSuccessInit = true
				if !isListening && !isLoginFailed {
					go listenUserStatus(opts.Ctx)
					logger.Debug().
						Msg("Start listening user status")
				}
				resetListenTicker <- true
				pendingMessage = "尝试重新初始化成功，现可正常运行"
			} else if isListening {
				pendingMessage += "尝试重新初始化失败，您可以使用 /ts3 命令来尝试手动初始化，或等待自动重连"
			} else {
				pendingMessage += "尝试重新初始化失败，您需要在服务器在线时手动使用 /ts3 命令来尝试初始化"
			}
		} else {
			pendingMessage += "这是一个无法恢复的错误，您可能需要联系机器人管理员"
		}
	}

	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:    opts.Update.Message.Chat.ID,
		Text:      pendingMessage,
		ParseMode: models.ParseModeHTML,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", opts.Update.Message.Chat.ID).
			Str("content", "teamspeak status").
			Msg(logt.SendMessage)
	}
	return nil
}

func listenUserStatus(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str("funcName", "listenUserStatus").
		Logger()

	isListening = true
	listenTicker := time.NewTicker(pollingInterval)
	defer listenTicker.Stop()

	if hasHandlerByChatID {
		hasHandlerByChatID = false
		// 获取到 privateOpts 后删掉 handler by chatID
		plugin_utils.RemoveHandlerByChatIDPlugin(tsData.GroupID, "teamspeak_get_opts")
	}

	var retryCount int = 1
	var beforeOnlineClient []string

	for {
		select {
		case <-resetListenTicker:
			listenTicker.Reset(pollingInterval)
			isCanListening = true
			retryCount = 1
		case <-listenTicker.C:
			if isSuccessInit && isCanListening {
				beforeOnlineClient = checkOnlineClientChange(ctx, beforeOnlineClient)
			} else {
				logger.Info().
					Msg("try reconnect...")
				// 出现错误时，先降低 ticker 速度，然后尝试重新初始化
				listenTicker.Reset(time.Duration(retryCount) * 20 * time.Second)
				if retryCount < 15 { retryCount++ }
				if initTeamSpeak(ctx) {
					isSuccessInit  = true
					isCanListening = true
					// 重新初始化成功则恢复 ticker 速度
					retryCount = 1
					listenTicker.Reset(pollingInterval)
					logger.Info().
						Msg("reconnect success")
					_, err := privateOpts.Thebot.SendMessage(privateOpts.Ctx, &bot.SendMessageParams{
						ChatID:    tsData.GroupID,
						Text:      "已成功与服务器重新建立连接",
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", tsData.GroupID).
							Str("content", "success reconnect to server").
							Msg(logt.SendMessage)
					}
				} else {
					// 无法成功则等待下一个周期继续尝试
					logger.Warn().
						Err(tsErr).
						Int("retryCount", retryCount).
						Int("nextRetry", (retryCount - 1) * 20).
						Msg("connect failed")
				}
			}
		}
	}
}

func checkOnlineClientChange(ctx context.Context, before []string) []string {
	var nowOnlineClient []string
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str("funcName", "checkOnlineClientChange").
		Logger()

	olClient, err := tsClient.Server.ClientList()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get online client")
		isCanListening = false
		_, err := privateOpts.Thebot.SendMessage(privateOpts.Ctx, &bot.SendMessageParams{
			ChatID:    tsData.GroupID,
			Text:      "已断开与服务器的连接，开始尝试自动重连",
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int64("chatID", tsData.GroupID).
				Str("content", "disconnect to server").
				Msg(logt.SendMessage)
		}
	} else {
		for _, n := range olClient {
			nowOnlineClient = append(nowOnlineClient, n.Nickname)
		}
		added, removed := DiffSlices(before, nowOnlineClient)
		if len(added) + len(removed) > 0 {
			logger.Debug().
				Strs("added", added).
				Strs("removed", removed).
				Msg("online client change detected")
			notifyClientChange(privateOpts, added, removed)
		}
	}

	return nowOnlineClient
}

func DiffSlices(before, now []string) (added, removed []string) {
	beforeMap := make(map[string]bool)
	nowMap    := make(map[string]bool)

	// 把 A 和 B 转成 map
	for _, item := range before { beforeMap[item] = true }
	for _, item := range now    { nowMap[item]    = true }

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

func notifyClientChange(opts *handler_structs.SubHandlerParams, add, remove []string) {
	var pendingMessage string
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str("funcName", "notifyClientChange").
		Logger()

	if len(add) > 0 {
		pendingMessage += fmt.Sprintln("以下用户进入了服务器:")
		for _, n := range add {
			pendingMessage += fmt.Sprintf("用户 [ %s ]\n", n)
		}
	}
	if len(remove) > 0 {
		pendingMessage += fmt.Sprintln("以下用户离开了服务器:")
		for _, n := range remove {
			pendingMessage += fmt.Sprintf("用户 [ %s ]\n", n)
		}
	}

	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: tsData.GroupID,
		Text:   pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", tsData.GroupID).
			Str("content", "teamspeak user change notify").
			Msg(logt.SendMessage)
	}
}
