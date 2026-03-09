package teamspeak

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/flaterr"
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/type/contain"
)

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
		// 这里可能要加锁？
		onlineClients, err := tsConfig.c.ClientList()
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to get online client")
			handlerErr.Addf("failed to get online client: %w", err)
			pendingMessage = fmt.Sprintf("获取服务器用户列表时发生了一些错误:\n<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error()))
		} else if onlineClients != nil {
			pendingMessage += fmt.Sprintln("在线客户端:")
			var userCount int
			for _, client := range *onlineClients {
				if client.ClientNickname == botNickName {
					// 统计用户数量时跳过此机器人
					continue
				}
				pendingMessage += fmt.Sprintf("用户 [ %s ] ", client.ClientNickname)
				userCount++
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
		buttons = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
			{{
				Text:         "管理此功能",
				CallbackData: "teamspeak",
			}},
			{{
				Text:         "清理通知消息",
				CallbackData: "teamspeak_clear",
			}},
		}}
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
		case "teamspeak_clear":
			needEdit = false
			var needDelMsgIDs []int

			for _, msg := range tsConfig.s.OldMessageID {
				needDelMsgIDs = append(needDelMsgIDs, msg.ID)
			}

			if len(needDelMsgIDs) > 0 {
				_, err := botInstance.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
					ChatID:     tsConfig.GroupID,
					MessageIDs: needDelMsgIDs,
				})
				if err != nil {
					_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
						CallbackQueryID: opts.CallbackQuery.ID,
						Text:            fmt.Sprintln("删除旧的通知消息发生错误", err.Error()),
						ShowAlert:       true,
					})
					wrap.ErrIf(err).MsgT(flaterr.AnswerCallbackQuery, "delete old message failed notice")
				} else {
					_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
						CallbackQueryID: opts.CallbackQuery.ID,
						Text:            fmt.Sprintf("已删除 %d 条旧的通知消息", len(needDelMsgIDs)),
					})
					wrap.ErrIf(err).MsgT(flaterr.AnswerCallbackQuery, "delete old message success notice")
				}
			} else {
				_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "没有旧的通知消息",
					ShowAlert:       true,
				})
				wrap.ErrIf(err).MsgT(flaterr.AnswerCallbackQuery, "no old messages notice")
			}
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
			var buttons [][]models.InlineKeyboardButton

			if tsConfig.SendMessageMode {
				buttons = append(buttons, []models.InlineKeyboardButton{
					{
						Text: utils.TextForTrueOrFalse(tsConfig.SendMessageMode, "✅ ", "") + "发送消息通知",
						CallbackData: "teamspeak_sendmessage",
					},
					{
						Text: utils.TextForTrueOrFalse(tsConfig.AutoDeleteMessage, "✅ ", "") + "自动删除旧消息",
						CallbackData: "teamspeak_sendmessage_autodelete",
					},
				})
			} else {
				buttons = append(buttons, []models.InlineKeyboardButton{{
					Text: utils.TextForTrueOrFalse(tsConfig.SendMessageMode, "✅ ", "") + "发送消息通知",
					CallbackData: "teamspeak_sendmessage",
				}})
			}

			if tsConfig.PinMessageMode {
				buttons = append(buttons, []models.InlineKeyboardButton{
					{
						Text: utils.TextForTrueOrFalse(tsConfig.PinMessageMode, "✅ ", "") + "显示在置顶消息",
						CallbackData: "teamspeak_pinmessage",
					},
					{
						Text: utils.TextForTrueOrFalse(tsConfig.DeleteOldPinnedMessage, "✅ ", "") + "删除旧的置顶消息",
						CallbackData: "teamspeak_pinmessage_deletepinedmessage",
					},
				})
			} else {
				buttons = append(buttons, []models.InlineKeyboardButton{{
					Text: utils.TextForTrueOrFalse(tsConfig.PinMessageMode, "✅ ", "") + "显示在置顶消息",
					CallbackData: "teamspeak_pinmessage",
				}})
			}
			buttons = append(buttons, []models.InlineKeyboardButton{{
				Text: "关闭菜单",
				CallbackData: "delete_this_message",
			}},)

			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID:   opts.CallbackQuery.Message.Message.ID,
				Text:        "选择通知用户变动的方式",
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: buttons },
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
