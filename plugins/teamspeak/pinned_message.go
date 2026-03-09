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
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/plugin_utils"
	"trle5.xyz/trbot/utils/type/message_utils"
	"trle5.xyz/trbot/utils/yaml"
)

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
				Str("path", tsConfigPath).
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
