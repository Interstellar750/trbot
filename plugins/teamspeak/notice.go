package teamspeak

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/flaterr"
)

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
