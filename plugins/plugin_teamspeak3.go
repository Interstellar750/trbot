package plugins

import (
	"fmt"
	"log"
	"os"
	"time"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_structs"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/multiplay/go-ts3"
)

// loginname serveradmin
// password 34lBKaih
// api key BACfzGIw3NCmmmqBXe1HAyi1p8BpCsUUeVb8JZQ
// sever admin privilege key Wwm5L2zNOHNiROqshy22fU9kconSORl+Pdt9aRqN

var tsClient *ts3.Client
var tsErr     error

var tsData_path string = consts.DB_path + "teamspeak/"
var botNickName string = "trbot_teamspeak_plugin"

var isCanReInit    bool = true
var isSuccessInit  bool = false
var isListening    bool = false
var isCanListening bool = false

var hasHandlerByChatID bool

var resetListenTicker chan bool = make(chan bool)
var pollingInterval   time.Duration = time.Second * 5

var tsServerQuery TSServerQuery
var privateOpts  *handler_structs.SubHandlerParams

type TSServerQuery struct {
	// get Name And Password in TeamSpeak 3 Client -> `Tools`` -> `ServerQuery Login`
	URL      string `yaml:"URL"`
	Name     string `yaml:"Name"`
	Password string `yaml:"Password"`
	GroupID  int64  `yaml:"GroupID"`
}

func init() {
	// 初始化不成功时依然注册 `/ts3` 命令，使用命令式输出初始化时的错误
	if initTeamSpeak() {
		isSuccessInit = true
		log.Println("TeamSpeak plugin loaded")

		// 需要以群组 ID 来触发 handler 来获取 opts
		plugin_utils.AddHandlerByChatIDPlugins(plugin_utils.HandlerByChatID{
			ChatID:      tsServerQuery.GroupID,
			PluginName: "teamspeak_get_opts",
			Handler:     getOptsHandler,
		})
		hasHandlerByChatID = true
	} else {
		log.Println("TeamSpeak plugin loaded failed:", tsErr)
	}

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

func initTeamSpeak() bool {
	// 判断配置文件是否存在
	_, err := os.Stat(tsData_path)
	if err != nil {
		if os.IsNotExist(err) {
			// 不存在，创建一份空文件
			err = utils.SaveYAML(tsData_path + consts.MetadataFileName, &TSServerQuery{})
			if err != nil {
				log.Println("[teamspeak] empty config create faild:", err)
			} else {
				log.Printf("[teamspeak] empty config created at [ %s ]", tsData_path)
			}
		} else {
			// 文件存在，但是遇到了其他错误
			tsErr = fmt.Errorf("[teamspeak] some error when read config file: %w", err)
		}

		// 无法获取到服务器地址和账号，无法初始化并设定不可重新启动
		isCanReInit = false
		return false
	}

	err = utils.LoadYAML(tsData_path + consts.MetadataFileName, &tsServerQuery)
	if err != nil {
	// if err != nil || tsServerQuery == nil {
		// 读取配置文件内容失败也不允许重新启动
		tsErr = fmt.Errorf("[teamspeak] read config error: %w", err)
		isCanReInit = false
		return false
	}

	// 如果服务器地址为空不允许重新启动
	if tsServerQuery.URL == "" {
		tsErr = fmt.Errorf("[teamspeak] no URL in config")
		isCanReInit = false
		return false
	} else {
		if tsClient != nil {
			// 如果指针不为空，那就先注销一下之前的登录
			tsClient.Logout()
		}
		tsClient, tsErr = ts3.NewClient(tsServerQuery.URL)
		if tsErr != nil {
			tsErr = fmt.Errorf("[teamspeak] connect error: %w", tsErr)
			return false
		}
	}

	// ServerQuery 账号名或密码为空也不允许重新启动
	if tsServerQuery.Name == "" || tsServerQuery.Password == "" {
		tsErr = fmt.Errorf("[teamspeak] no Name/Password in config")
		isCanReInit = false
		return false
	} else {
		err = tsClient.Login(tsServerQuery.Name, tsServerQuery.Password)
		if err != nil {
			tsErr = fmt.Errorf("[teamspeak] login error: %w", err)
			return false
		}
	}

	// 检查要设定通知的群组 ID 是否存在
	if tsServerQuery.GroupID == 0 {
		tsErr = fmt.Errorf("[teamspeak] no GroupID in config")
		isCanReInit = false
		return false
	}

	// 显示服务端版本测试一下连接
	v, err := tsClient.Version()
	if err != nil {
		tsErr = fmt.Errorf("[teamspeak] show version error: %w", err)
		return false
	} else {
		log.Printf("[teamspeak] running: %v", v)
	}

	// 切换默认虚拟服务器
	err = tsClient.Use(1)
	if err != nil {
		tsErr = fmt.Errorf("[teamspeak] switch server error: %w", err)
		return false
	}

	// 改一下 bot 自己的 nickname，使得在检测用户列表时默认不显示自己
	m, err := tsClient.Whoami()
	if err != nil {
		tsErr = fmt.Errorf("[teamspeak] get my info error: %w", err)
	} else if m != nil && m.ClientName != botNickName {
		// 当 bot 自己的 nickname 不等于配置文件中的 nickname 时，才进行修改
		err = tsClient.SetNick(botNickName)
		if err != nil {
			tsErr = fmt.Errorf("[teamspeak] set nickname error: %w", err)
		}
	}

	// 没遇到不可重新初始化的部分则返回初始化成功
	return true
}

// 用于首次初始化成功时只要对应群组有任何消息，都能自动获取 privateOpts 用来定时发送消息，并开启监听协程
func getOptsHandler(opts *handler_structs.SubHandlerParams) {
	if !isListening && isCanReInit && opts.Update.Message.Chat.ID == tsServerQuery.GroupID {
		privateOpts = opts
		isCanListening = true
		go listenUserStatus()
		if consts.IsDebugMode { log.Println("success get opts and start listening") }
	}
}

func showStatus(opts *handler_structs.SubHandlerParams) {
	var pendingMessage string

	// 如果首次初始化没成功，没有添加根据群组 ID 来触发的 handler，用户发送 /ts3 后可以通过这个来自动获取 opts 并启动监听
	// if isSuccessInit && !isCanListening && opts.Update != nil && opts.Update.Message != nil && opts.Update.Message.Chat.ID == tsServerQuery.GroupID {
	if !isListening && isCanReInit && opts.Update.Message.Chat.ID == tsServerQuery.GroupID {
		privateOpts = opts
		isCanListening = true
		if consts.IsDebugMode { log.Println("success get opts") }
		if !isListening {
			go listenUserStatus()
			if consts.IsDebugMode { log.Println("success start listening") }
		}
		// pendingMessage += fmt.Sprintln("已准备好发送用户状态")
	}

	if isSuccessInit && isCanListening {
		olClient, err := tsClient.Server.ClientList()
		if err != nil {
			log.Println("[teamspeak] get online client error:", err)
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
				pendingMessage += "当前无用户在线"
			}
		}
	} else {
		pendingMessage = fmt.Sprintf("初始化 teamspeak 插件时发生了一些错误:\n<blockquote expandable>%s</blockquote>\n\n", tsErr)
		if isCanReInit {
			if initTeamSpeak() {
				isSuccessInit = true
				tsErr = fmt.Errorf("")
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
		log.Println("[teamspeak] can't answer `/ts3` command:",err)
	}
}

func listenUserStatus() {
	isListening = true
	listenTicker := time.NewTicker(pollingInterval)
	defer listenTicker.Stop()

	if hasHandlerByChatID {
		hasHandlerByChatID = false
		// 获取到 privateOpts 后删掉 handler by chatID
		plugin_utils.RemoveHandlerByChatIDPlugin(tsServerQuery.GroupID, "teamspeak_get_opts")
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
				beforeOnlineClient = checkOnlineClientChange(beforeOnlineClient)
			} else {
				if consts.IsDebugMode {
					log.Println("[teamspeak] try reconnect...")
				}
				// 出现错误时，先降低 ticker 速度，然后尝试重新初始化
				listenTicker.Reset(time.Duration(retryCount) * 20 * time.Second)
				if retryCount < 15 {
					retryCount++
				}
				if tsClient != nil {
					// 重试前尝试注销一次
					tsClient.Logout()
				}
				if initTeamSpeak() {
					isSuccessInit  = true
					isCanListening = true
					// 重新初始化成功则恢复 ticker 速度
					retryCount = 1
					listenTicker.Reset(pollingInterval)
					if consts.IsDebugMode {
						log.Println("[teamspeak] reconnect success")
					}
					privateOpts.Thebot.SendMessage(privateOpts.Ctx, &bot.SendMessageParams{
						ChatID:    privateOpts.Update.Message.Chat.ID,
						Text:      "已成功与服务器重新建立连接",
						ParseMode: models.ParseModeHTML,
					})
				} else {
					// 无法成功则等待下一个周期继续尝试
					if consts.IsDebugMode {
						log.Printf("[teamspeak] reconnect failed, retry in %ds", (retryCount -1) * 20)
					}
				}
			}
		}
	}
}

func checkOnlineClientChange(before []string) []string {
	var nowOnlineClient []string

	olClient, err := tsClient.Server.ClientList()
	if err != nil {
		log.Println("[teamspeak] get online client error:", err)
		isCanListening = false
		privateOpts.Thebot.SendMessage(privateOpts.Ctx, &bot.SendMessageParams{
			ChatID:    privateOpts.Update.Message.Chat.ID,
			Text:      "已断开与服务器的连接，开始尝试自动重连",
			ParseMode: models.ParseModeHTML,
		})
	} else {
		for _, n := range olClient {
			nowOnlineClient = append(nowOnlineClient, n.Nickname)
		}
		added, removed := DiffSlices(before, nowOnlineClient)
		if len(added) + len(removed) > 0 {
			if consts.IsDebugMode {
				log.Printf("[teamspeak] online client change: added %v, removed %v", added, removed)
			}
			notifyClientChange(privateOpts, tsServerQuery.GroupID, added, removed)
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

func notifyClientChange(opts *handler_structs.SubHandlerParams, chatID int64, add, remove []string) {
	var pendingMessage string

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

	opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
}
