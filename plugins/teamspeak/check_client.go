package teamspeak

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
	"github.com/rs/zerolog"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/flaterr"
	"trle5.xyz/trbot/utils/task"
)

// CheckClient 检查 TeamSpeak 服务器的状态和用户数量，根据配置通知用户或尝试重连
func (sc *ServerConfig) CheckClient(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	// 仅在能立即锁定时继续操作，否则直接退出
	if sc.rw.TryLock() {
		defer sc.rw.Unlock()
	} else {
		return
	}

	onlineClients, err := sc.c.ClientList()
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
				sc.s.RetryMsgID = botMessage.ID
			}
		}
		return
	} else if onlineClients != nil {
		var nowOnlineClient []OnlineClient
		sc.s.CheckFailedCount = 0 // 重置失败计数
		for _, client := range *onlineClients {
			if client.ClientNickname == botNickName { continue }

			var isExist bool
			for _, user := range sc.s.BeforeOnlineClient {
				if user.DatabaseID == client.ClientDatabaseId {
					nowOnlineClient = append(nowOnlineClient, user)
					isExist = true
				}
			}
			if !isExist {
				nowOnlineClient = append(nowOnlineClient, OnlineClient{
					Username:   client.ClientNickname,
					DatabaseID: client.ClientDatabaseId,
					JoinTime:   time.Now(),
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
