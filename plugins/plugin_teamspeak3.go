package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/flaterr"
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/plugin_utils"
	"trle5.xyz/trbot/utils/task"
	"trle5.xyz/trbot/utils/type/contain"
	"trle5.xyz/trbot/utils/type/message_utils"
	"trle5.xyz/trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/multiplay/go-ts3"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
	"github.com/rs/zerolog"
)

var tsConfig ServerConfig

var tsConfigPath string = filepath.Join(configs.YAMLDatabaseDir, "teamspeak/", configs.YAMLFileName)
var botNickName  string = "trbot_teamspeak_plugin"

var botInstance *bot.Bot

type ServerConfig struct {
	rw sync.RWMutex
	c  *ts3.Client
	s  Status

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

// Connect 检查配置并尝试连接到 TeamSpeak 服务器
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
		err = sc.c.Close()
		if err != nil {
			logger.Error().
				Err(err).
				Str("path", tsConfigPath).
				Msg("Failed to close client")
		}
		// 尽管它有时会返回错误，但可能还是会正常关闭，所以等一等，然后抛弃旧的客户端
		time.Sleep(10 * time.Second)
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

	if !sc.s.IsCheckClientTaskScheduled {
		err = sc.ScheduleCheckClientTask(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to schedule check client task")
			return fmt.Errorf("failed to schedule check client task: %w", err)
		}
	}

	if !sc.s.IsDeleteMessageTaskScheduled {
		err = sc.ScheduleDeleteMessageTask(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to schedule delete message task")
			return fmt.Errorf("failed to schedule delete message task: %w", err)
		}
	}

	return nil
}

// CheckClient 检查 TeamSpeak 服务器的状态和用户数量，根据配置通知用户或尝试重连
func (sc *ServerConfig) CheckClient(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	sc.rw.Lock()
	defer sc.rw.Unlock()

	onlineClients, err := sc.c.Server.ClientList()
	if err != nil {
		sc.s.CheckFailedCount++
		logger.Error().
			Err(err).
			Int("failedCount", sc.s.CheckFailedCount).
			Msg("Failed to get online client")
		if err.Error() == "not connected" {
			// 连不上服务器直接暂停人物为并尝试重连
			sc.PauseCheckClientTask(ctx, true)
		} else if sc.s.CheckFailedCount >= 5 {
			// 不是连不上服务器，则累积到五次后再重连
			sc.PauseCheckClientTask(ctx, true)
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
				sc.s.OldMessageID = append(tsConfig.s.OldMessageID, OldMessageID{
					Date: int(time.Now().Unix()),
					ID:   botMessage.ID,
				})
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
			sc.NotifyClientChange(ctx, added, removed)
		}
		if sc.PinMessageMode {
			sc.NewPinnedMessage(ctx)
			sc.ChangePinnedMessage(ctx, nowOnlineClient, added, removed)
		}
		sc.s.BeforeOnlineClient = nowOnlineClient
	}
}

// ScheduleOrResumeCheckClientTask 创建检查客户端任务
func (sc *ServerConfig) ScheduleCheckClientTask(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := task.ScheduleTask(ctx, task.Task{
		Name:  fmt.Sprintf("check_client_%d", sc.GroupID),
		Group: "teamspeak3",
		Job: job.NewFunctionJobWithDesc(
			func(ctx context.Context) (int, error) {
				sc.CheckClient(ctx)
				return 0, nil
			},
			"check teamspeak client changes",
		),
		Trigger: quartz.NewSimpleTrigger(time.Second * time.Duration(sc.PollingInterval)),
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule check client job")
		return err
	}

	sc.s.IsCheckClientTaskScheduled = true
	sc.s.IsCheckClientTaskRunning   = true

	return nil
}

// ResumeCheckClientTask 恢复检查客户端任务
func (sc *ServerConfig) ResumeCheckClientTask(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	if !sc.s.IsCheckClientTaskRunning {
		err := task.ResumeTask(ctx, fmt.Sprintf("check_client_%d", sc.GroupID), "teamspeak3")
		if err != nil {
			logger.Error().Err(err).Msg("Failed to resume check client task")
		} else {
			sc.s.IsCheckClientTaskRunning = true
		}
	}
}

// PauseCheckClientTask 暂停检查客户端任务，可选是否开始重试
func (sc *ServerConfig) PauseCheckClientTask(ctx context.Context, retry bool) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	if sc.s.IsCheckClientTaskRunning {
		err := task.PauseTask(ctx, fmt.Sprintf("check_client_%d", sc.GroupID), "teamspeak3")
		if err != nil {
			logger.Error().Err(err).Msg("Failed to pause check client task")
		} else {
			sc.s.IsCheckClientTaskRunning = false
		}
	}

	if retry && !sc.s.IsInRetryLoop {
		go sc.RetryLoop(ctx)
		logger.Warn().Msg("Start retry connect loop")
	}
}

// NotifyClientChange 通过在对话中发送信息的方式来通知用户变化
func (sc *ServerConfig) NotifyClientChange(ctx context.Context, add, remove []string) {
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
		ChatID: sc.GroupID,
		Text:   pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", sc.GroupID).
			Str("content", "teamspeak user change notify").
			Msg(flaterr.SendMessage.Str())
	}

	if tsConfig.s.IsDeleteMessageTaskScheduled {
		if tsConfig.AutoDeleteMessage {
			tsConfig.ResumeDeleteMessageTask(ctx)
			tsConfig.s.OldMessageID = append(tsConfig.s.OldMessageID, OldMessageID{
				Date: msg.Date,
				ID:   msg.ID,
			})
		} else if tsConfig.s.IsDeleteMessageTaskRunning {
			tsConfig.PauseDeleteMessageTask(ctx)
			tsConfig.s.OldMessageID = []OldMessageID{}
		}
	}
}

// ChangePinnedMessage 通过编辑置顶消息的方式来通知用户变化
func (sc *ServerConfig) ChangePinnedMessage(ctx context.Context, online []OnlineClient, add, remove []string) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	// 没有新加入和离开用户，等待一阵子后再更新用户在线时间
	if len(add) + len(remove) == 0 && sc.s.CheckCount < (60 / tsConfig.PollingInterval) {
		sc.s.CheckCount++
		return
	} else {
		sc.s.CheckCount = 0
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

	if !sc.s.IsMessagePinned {
		pendingMessage += "<blockquote expandable>无法置顶用户列表消息，请检查机器人是否拥有对应的权限，您也可以手动置顶此消息</blockquote>"
	}

	_, err := botInstance.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:    sc.GroupID,
		MessageID: sc.PinnedMessageID,
		Text:      pendingMessage,
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Int64("chatID", sc.GroupID).
			Str("content", "teamspeak user change notify").
			Msg(flaterr.EditMessageText.Str())
	}
}

// ScheduleDeleteMessageTask 创建定时删除旧通知信息的任务
func (sc *ServerConfig) ScheduleDeleteMessageTask(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)

	// 根据 sc.DeleteTimeoutInMinute 计划一个任务
	// 定时检查 sc.s.OldMessageID 中是否有过期的消息，如果有则删除
	err := task.ScheduleTask(ctx, task.Task{
		Name:  fmt.Sprintf("delete_old_message_%d", sc.GroupID),
		Group: "teamspeak3",
		Job: job.NewFunctionJobWithDesc(
			func(ctx context.Context) (int, error) {
				var needDelMsgIDs []int
				var keepMsgIDList []OldMessageID

				for _, msg := range sc.s.OldMessageID {
					if time.Now().Unix() > int64(msg.Date + sc.DeleteTimeoutInMinute * 60) {
						needDelMsgIDs = append(needDelMsgIDs, msg.ID)
					} else {
						keepMsgIDList = append(keepMsgIDList, msg)
					}
				}
				sc.s.OldMessageID = keepMsgIDList

				if len(needDelMsgIDs) > 0 {
					_, err := botInstance.DeleteMessages(ctx, &bot.DeleteMessagesParams{
						ChatID:     sc.GroupID,
						MessageIDs: needDelMsgIDs,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Ints("msgIDs", needDelMsgIDs).
							Int("deleteMessageMinute", sc.DeleteTimeoutInMinute).
							Str("content", "teamspeak user change notify").
							Msg(flaterr.DeleteMessage.Str())
						return 1, err
					}
				}
				return 0, nil
			},
			"delete teamspeak user change notify message",
		),
		Trigger: quartz.NewSimpleTrigger(time.Minute * time.Duration(sc.DeleteTimeoutInMinute)),
	})
	if err != nil {
		logger.Error().Err(err).Msg("Failed to schedule delete message job")
		return err
	}

	sc.s.IsDeleteMessageTaskScheduled = true
	sc.s.IsDeleteMessageTaskRunning   = true

	// 如果 sc.AutoDeleteMessage 不为真，则就是没有启用自动删除消息功能，所以暂停任务
	if !sc.AutoDeleteMessage {
		sc.PauseDeleteMessageTask(ctx)
	}

	return nil
}

// ResumeDeleteMessageTask 恢复删除旧通知信息的任务
func (sc *ServerConfig) ResumeDeleteMessageTask(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	if !tsConfig.s.IsDeleteMessageTaskRunning {
		err := task.ResumeTask(ctx, fmt.Sprintf("delete_old_message_%d", tsConfig.GroupID), "teamspeak3")
		if err != nil {
			logger.Error().Err(err).Msg("Failed to resume delete old message task")
		} else {
			tsConfig.s.IsDeleteMessageTaskRunning = true
		}
	}
}

// PauseDeleteMessageTask 暂停删除旧通知信息的任务
func (sc *ServerConfig) PauseDeleteMessageTask(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	if tsConfig.s.IsDeleteMessageTaskRunning {
		err := task.PauseTask(ctx, fmt.Sprintf("delete_old_message_%d", tsConfig.GroupID), "teamspeak3")
		if err != nil {
			logger.Error().Err(err).Msg("Failed to pause delete old message task")
		} else {
			tsConfig.s.IsDeleteMessageTaskRunning = false
		}
	}
}

// RetryLoop 将循环尝试连接到服务器，直到成功为止
func (sc *ServerConfig) RetryLoop(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	sc.s.IsInRetryLoop = true
	defer func() {
		sc.s.IsInRetryLoop = false
		logger.Info().Msg("reconnect loop exited")
	}()
	sc.s.RetryCount = 0

	retryTicker := time.NewTicker(time.Second * 5)
	defer retryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sc.s.ResetTicker:
			sc.ResumeCheckClientTask(ctx)
			logger.Info().Msg("reconnect by command success")
			return
		case <-retryTicker.C:
			logger.Info().
				Int("retryCount", sc.s.RetryCount).
				Msg("try reconnect...")

			// 实际上并不生效...
			timeoutCtx, cancel := context.WithTimeout(ctx, time.Second * 30)
			defer cancel()

			err := sc.Connect(timeoutCtx)
			if err != nil {
				// 出现错误时，先降低 ticker 速度，然后尝试重新初始化
				// 无法成功则等待下一个周期继续尝试
				if sc.s.RetryCount < 15 {
					sc.s.RetryCount++
					retryTicker.Reset(time.Duration(sc.s.RetryCount * 20) * time.Second)
				}

				logger.Warn().
					Int("retryCount", sc.s.RetryCount).
					Time("nextRetry", time.Now().Add(time.Duration(sc.s.RetryCount * 20) * time.Second)).
					Msg("reconnect failed")
			} else {
				// 重新初始化成功则恢复查询任务
				sc.ResumeCheckClientTask(ctx)
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
					sc.s.OldMessageID = append(tsConfig.s.OldMessageID, OldMessageID{
						Date: int(time.Now().Unix()),
						ID:   botMessage.ID,
					})
				}
				return
			}
		}
	}
}

// BuildConfigKeyboard 构建一个 &models.InlineKeyboardMarkup 类型的配置键盘
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

// CheckPinnedMessage 检查是否存在置顶的消息，并检查它是否可以被编辑。
//
// 如果可以编辑，则不进行其他操作，否则将删除或取消固定消息，并重新发送一条新的消息用于编辑。
//
// 若不存在置顶消息，则发送一条新的消息用于编辑。
func (sc *ServerConfig) CheckPinnedMessage(ctx context.Context) {

	// 尝试编辑旧的消息
	if sc.IsPinnedMessageCanEdit(ctx) {
		// 因为没有简单的方法得知旧消息有没有被置顶，就假设已经成功置顶了
		sc.s.IsMessagePinned = true
	} else {
		// 无法编辑，就取消置顶或删除消息，再由后续逻辑发送新消息
		sc.RemovePinnedMessage(ctx, false)
	}

	// 发送一条新的信息
	sc.NewPinnedMessage(ctx)
}

// IsPinnedMessageCanEdit 当 sc.PinnedMessageID 不为 0 时检查是否可以编辑被固定的消息
func (sc *ServerConfig) IsPinnedMessageCanEdit(ctx context.Context) bool {
	if sc.PinnedMessageID != 0 {
		_, err := botInstance.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    sc.GroupID,
			MessageID: sc.PinnedMessageID,
			Text:      fmt.Sprintf("%s | 开始监听 Teamspeak 3 用户状态", time.Now().Format("15:04")),
		})
		if err != nil {
			if err.Error() == "bad request, Bad Request: message is not modified: specified new message content and reply markup are exactly the same as a current content and reply markup of the message" {
				// 机器人重启的太快，导致消息文本相同，但实际上还是能编辑的
				return true
			}
			zerolog.Ctx(ctx).Error().
				Err(err).
				Str("pluginName", "teamspeak3").
				Str(utils.GetCurrentFuncName()).
				Int64("chatID", sc.GroupID).
				Str("content", "start listen teamspeak user changes").
				Msg(flaterr.EditMessageText.Str())
			return false
		}
		return true
	}

	return false
}

// NewPinnedMessage 当 sc.PinnedMessageID 为 0 时发送一条新的消息用于编辑
func (sc *ServerConfig) NewPinnedMessage(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	if sc.PinnedMessageID == 0 {
		message, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:              sc.GroupID,
			Text:                fmt.Sprintf("%s | 开始监听 Teamspeak 3 用户状态", time.Now().Format("15:04")),
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
}

// RemovePinnedMessage 当 sc.PinnedMessageID 不为 0 时取消或删除置顶消息
//
// keepID 参数仅在 sc.DeleteOldPinnedMessage 不为 true 时才生效
func (sc *ServerConfig) RemovePinnedMessage(ctx context.Context, keepID bool) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	// 取消置顶或删除上一次的置顶消息
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

		if sc.DeleteOldPinnedMessage || !keepID {
			// 如果设置是删除旧消息，或不需要保留 ID，则清空消息 ID
			sc.PinnedMessageID = 0
		}

		err := saveTeamspeakData(ctx)
		if err != nil {
			logger.Error().Err(err).Msg("Failed to save teamspeak data after delete or unpin message")
		}
	}
}

type Status struct {
	IsMessagePinned bool

	ResetTicker (chan bool)

	IsInRetryLoop bool

	RetryCount       int
	CheckCount       int
	CheckFailedCount int

	BeforeOnlineClient []OnlineClient

	IsCheckClientTaskScheduled bool
	IsCheckClientTaskRunning   bool

	IsDeleteMessageTaskScheduled bool
	IsDeleteMessageTaskRunning   bool

	OldMessageID []OldMessageID
}

type OldMessageID struct {
	Date int
	ID  int
}

type OnlineClient struct {
	Username string
	JoinTime time.Time
}

func init() {
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

// showStatus 响应 `/ts3` 命令，显示用户列表或触发手动连接，可显示连接错误信息
func showStatus(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var handlerErr flaterr.MultErr
	var pendingMessage string

	// 正常运行就输出用户列表，否则发送错误信息
	if tsConfig.s.IsCheckClientTaskScheduled && tsConfig.s.IsCheckClientTaskRunning {
		onlineClients, err := tsConfig.c.Server.ClientList()
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to get online client")
			handlerErr.Addf("failed to get online client: %w", err)
			pendingMessage = fmt.Sprintf("获取服务器用户列表时发生了一些错误:\n<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error()))
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
			pendingMessage = fmt.Sprintf("teamspeak 插件发生了一些错误:\n<blockquote expandable>%s</blockquote>\n\n", err)
			if tsConfig.s.IsCheckClientTaskScheduled{
				pendingMessage += "您可以使用 /ts3 命令来尝试重新连接或等待自动重连"
			} else {
				pendingMessage += "尝试重新初始化失败，由于检查任务未在运行，您需要手动使用 /ts3 命令来尝试重新连接"
			}
			handlerErr.Addf("failed to reinit teamspeak plugin: %w", err)
		} else {
			if tsConfig.s.IsInRetryLoop {
				tsConfig.s.ResetTicker <- true
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

// diffSlices 比较两个 OnlineClient 类型切片，返回新增和删除的字符串类型切片
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

// teamspeakCallbackHandler 响应前缀为 "teamspeak" 的 callbackQuery 请求
func teamspeakCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Str("callbackQueryData", opts.CallbackQuery.Data).
		Logger()

	wrap := flaterr.NewWrapper(logger.Error())

	if contain.Int64(opts.CallbackQuery.From.ID, utils.GetChatAdminIDs(opts.Ctx, opts.Thebot, tsConfig.GroupID)...) || contain.Int64(opts.CallbackQuery.From.ID, configs.BotConfig.AdminIDs...) {
		var needEdit     bool = true
		var needSave     bool = false
		var needEditTask bool = false

		switch opts.CallbackQuery.Data {
		case "teamspeak_pinmessage":
			if tsConfig.PinMessageMode && !tsConfig.SendMessageMode {
				needEdit = false
				_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "您至少要保留一个检测用户变动的方式",
					ShowAlert:       true,
				})
				wrap.ErrIf(err).MsgT(flaterr.AnswerCallbackQuery, "at least need one notice method")
			} else {
				tsConfig.PinMessageMode = !tsConfig.PinMessageMode
				needSave = true
				if tsConfig.PinMessageMode {
					if tsConfig.PinnedMessageID != 0 {
						// 机器人启动时没有使用置顶模式，但后续开了，而且消息 ID 不为零，就假设已经成功置顶了
						tsConfig.s.IsMessagePinned = true
					}
				} else {
					// 关闭时解除当前消息的置顶
					tsConfig.RemovePinnedMessage(opts.Ctx, true)
				}
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
				wrap.ErrIf(err).MsgT(flaterr.AnswerCallbackQuery, "at least need one notice method")
			} else {
				tsConfig.SendMessageMode = !tsConfig.SendMessageMode
				needSave = true
				needEditTask = true
			}
		case "teamspeak_sendmessage_autodelete":
			tsConfig.AutoDeleteMessage = !tsConfig.AutoDeleteMessage
			needSave = true
			needEditTask = true
		}

		if needEditTask {
			if tsConfig.s.IsDeleteMessageTaskScheduled {
				if tsConfig.SendMessageMode && tsConfig.AutoDeleteMessage {
					tsConfig.ResumeDeleteMessageTask(opts.Ctx)
				} else {
					tsConfig.PauseDeleteMessageTask(opts.Ctx)
					tsConfig.s.OldMessageID = []OldMessageID{}
				}
			} else {
				_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "您的操作已保存，但因为删除消息的任务没有添加成功，无法自动删除消息，请尝试重启机器人",
					ShowAlert:       true,
				})
				wrap.ErrIf(err).MsgT(flaterr.AnswerCallbackQuery, "delete message task not scheduled")
			}
		}

		if needEdit {
			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text:      "选择通知用户变动的方式",
				ReplyMarkup: tsConfig.BuildConfigKeyboard(),
			})
			wrap.ErrIf(err).MsgT(flaterr.EditMessageText, "teamspeak manage keyboard")
		}

		if needSave {
			err := saveTeamspeakData(opts.Ctx)
			wrap.ErrIf(err).Msg("failed to save teamspeak data")
		}
	} else {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "您没有权限修改此内容",
			ShowAlert:       true,
		})
		wrap.ErrIf(err).MsgT(flaterr.AnswerCallbackQuery, "no permission to change teamspeak config")
	}

	return wrap.Flat()
}
