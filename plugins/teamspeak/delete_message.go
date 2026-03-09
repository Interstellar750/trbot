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

func (sc *ServerConfig) DeleteMessage(ctx context.Context) (int, error) {
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
			zerolog.Ctx(ctx).Error().
				Err(err).
				Ints("msgIDs", needDelMsgIDs).
				Int("deleteMessageMinute", sc.DeleteTimeoutInMinute).
				Str("content", "teamspeak user change notify").
				Msg(flaterr.DeleteMessage.Str())
			return 1, err
		}
	}
	return 0, nil
}

// ScheduleDeleteMessageTask 创建定时删除旧通知信息的任务
func (sc *ServerConfig) ScheduleDeleteMessageTask(ctx context.Context) error {
	// 根据 sc.DeleteTimeoutInMinute 计划一个任务
	// 定时检查 sc.s.OldMessageID 中是否有过期的消息，如果有则删除
	err := task.ScheduleTask(ctx, task.Task{
		Name:  fmt.Sprintf("delete_old_message_%d", sc.GroupID),
		Group: "teamspeak3",
		Job: job.NewFunctionJobWithDesc(
			sc.DeleteMessage,
			"delete teamspeak user change notify message",
		),
		Trigger: quartz.NewSimpleTrigger(time.Minute * time.Duration(sc.DeleteTimeoutInMinute)),
	})
	if err != nil {
		zerolog.Ctx(ctx).Error().Err(err).Msg("Failed to schedule delete message job")
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
