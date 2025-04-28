package plugins

import (
	"fmt"
	"log"
	"os"
	"time"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_utils"
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
var tsErr error

var TSData_path    string = consts.DB_path + "teamspeak/"
var botNickName    string = "trbot_teamspeak_plugin"
var notifyGroupID  int64  = -1002499888124
var isCanRunSignal bool   = false
// var isListening    bool   = true
// var needReInit     bool   = false
var tsServerQuery *TSServerQuery

var privateOpts *handler_utils.SubHandlerOpts

type TSServerQuery struct {
	// get Name And Password in TeamSpeak 3 Client -> `Tools`` -> `ServerQuery Login`
	URL      string `yaml:"URL"`
	Name     string `yaml:"Name"`
	Password string `yaml:"Password"`
}

func init() {
	// todo 初始化不成功时依然注册 `/ts3` 命令，使用命令式输出初始化时的错误
	if initTeamSpeak() {
		log.Println("TeamSpeak plugin loaded")
		plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
			SlashCommand: "ts3",
			Handler: showStatus,
		})

		// 需要以群组 ID 来触发 handler 来获取 opts
		plugin_utils.AddHandlerByChatIDPlugins(plugin_utils.HandlerByChatID{
			ChatID: notifyGroupID,
			PluginName: "teamspeak_get_opts",
			Handler: getOptsHandler,
		})
	}
}

func initTeamSpeak() bool {
	_, err := os.Stat(TSData_path)
	if err != nil {
		if os.IsNotExist(err) {
			err = utils.SaveYAML(TSData_path + consts.MetadataFileName, &TSServerQuery{})
			if err != nil {
				log.Println("[teamspeak] empty config create faild:", err)
			} else {
				log.Printf("[teamspeak] empty config created at [ %s ]", TSData_path)
			}
		} else {
			log.Println("[teamspeak] some error when read config file:", err)
			return false
		}
	}

	err = utils.LoadYAML(TSData_path + consts.MetadataFileName, &tsServerQuery)
	if err != nil {
		log.Println("[teamspeak] read config error:", err)
		return false
	}

	if tsServerQuery.URL == "" {
		log.Println("[teamspeak] no URL in config")
		return false
	} else {
		tsClient, tsErr = ts3.NewClient(tsServerQuery.URL)
		if tsErr != nil {
			log.Println("[teamspeak] connect error:", tsErr)
			return false
		}
	}

	if tsServerQuery.Name == "" || tsServerQuery.Password == "" {
		log.Println("[teamspeak] no Name/Password in config")
		return false
	} else {
		err = tsClient.Login(tsServerQuery.Name, tsServerQuery.Password)
		if err != nil {
			log.Println("[teamspeak] login error:", err)
			return false
		}
	}

	v, err := tsClient.Version()
	if err != nil {
		log.Println("[teamspeak] show version error:", err)
		return false
	} else {
		log.Printf("[teamspeak] running: %v", v)
	}

	err = tsClient.Use(1)
	if err != nil {
		log.Println("[teamspeak] switch server error:", err)
		return false
	}

	m, err := tsClient.Whoami()
	if err != nil {
		log.Println("[teamspeak] get my error:", err)
	}

	if m.ClientName != botNickName {
		err = tsClient.SetNick(botNickName)
		if err != nil {
			log.Println("[teamspeak] set nickname error:", err)
		}
	}

	return true
}

func getOptsHandler(opts *handler_utils.SubHandlerOpts) {
	if !isCanRunSignal && opts.Update != nil && opts.Update.Message != nil && opts.Update.Message.Chat.ID == notifyGroupID {
		privateOpts = opts
		isCanRunSignal = true

		// 获取到 opts 后删掉 handler by chatID
		plugin_utils.RemoveHandlerByChatIDPlugin(notifyGroupID, "teamspeak_get_opts")
		go listenUserStatus()
	}
}

func showStatus(opts *handler_utils.SubHandlerOpts) {
	var pendingMessage string

	if !isCanRunSignal && opts.Update != nil && opts.Update.Message != nil && opts.Update.Message.Chat.ID == notifyGroupID {
		privateOpts = opts
		isCanRunSignal = true
		// 把启动线程的 goroutine 挪到这里？
		// pendingMessage += fmt.Sprintln("已准备好发送用户状态")
	}

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
	
	
	opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:    opts.Update.Message.Chat.ID,
		Text:      pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
}

func listenUserStatus() {
	// isListening = true
	listenTicker := time.NewTicker(5 * time.Second)
	defer listenTicker.Stop()

	var beforeOnlineClient []string

	for {
		select {
		case <-listenTicker.C:
			if isCanRunSignal {
				beforeOnlineClient = checkOnlineClientChange(beforeOnlineClient)
			} else {
				log.Println("[teamspeak] try reconnect...")
				listenTicker.Reset(10 * time.Second)
				if initTeamSpeak() {
					isCanRunSignal = true
					log.Println("[teamspeak] reconnect success")
				} else {
					log.Println("[teamspeak] reconnect failed, retry in 10s")
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
	} else {
		for _, n := range olClient {
			nowOnlineClient = append(nowOnlineClient, n.Nickname)
		}
		added, removed := DiffSlices(before, nowOnlineClient)
		if len(added) + len(removed) > 0 {
			log.Printf("[teamspeak] online client change: added %v, removed %v", added, removed)
			notifyClientChange(privateOpts, notifyGroupID, added, removed)
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

func notifyClientChange(opts *handler_utils.SubHandlerOpts, chatID int64, add, remove []string) {
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
