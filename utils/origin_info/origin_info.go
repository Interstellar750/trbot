package origin_info

import (
	"fmt"

	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/type/message_utils"

	"github.com/go-telegram/bot/models"
)

type OriginInfo struct {
	FromName string `yaml:"FromName,omitempty" json:"from_name,omitempty"`
	FromID   int64  `yaml:"FromID,omitempty" json:"from_id,omitempty"`

	// 用于查看消息来源
	ChatID    int64 `yaml:"ChatID,omitempty" json:"chat_id,omitempty"`
	MessageID int   `yaml:"MessageID,omitempty" json:"message_id,omitempty"`
}

func GetOriginInfo(msg *models.Message) *OriginInfo {
	if msg == nil { return nil }

	if msg.ForwardOrigin != nil && msg.ForwardOrigin.MessageOriginHiddenUser == nil {
		switch msg.ForwardOrigin.Type {
		case models.MessageOriginTypeUser:
			return &OriginInfo{
				FromName: utils.ShowUserName(&msg.ForwardOrigin.MessageOriginUser.SenderUser),
				FromID:   msg.ForwardOrigin.MessageOriginUser.SenderUser.ID,
			}
		// case models.MessageOriginTypeHiddenUser:
		// 	return &OriginInfo{
		// 		FromName: msg.ForwardOrigin.MessageOriginHiddenUser.SenderUserName,
		// 	}
		case models.MessageOriginTypeChat:
			return &OriginInfo{
				FromName: utils.ShowChatName(&msg.ForwardOrigin.MessageOriginChat.SenderChat),
				FromID:   msg.ForwardOrigin.MessageOriginChat.SenderChat.ID,
			}
		case models.MessageOriginTypeChannel:
			return &OriginInfo{
				FromName:  utils.ShowChatName(&msg.ForwardOrigin.MessageOriginChannel.Chat),
				FromID:    msg.ForwardOrigin.MessageOriginChannel.Chat.ID,
				MessageID: msg.ForwardOrigin.MessageOriginChannel.MessageID,
			}
		}
	} else if msg.Chat.Type != models.ChatTypePrivate {
		attr := message_utils.GetMessageAttribute(msg)
		if attr.IsFromLinkedChannel || attr.IsFromAnonymous || attr.IsUserAsChannel {
			return &OriginInfo{
				FromName:  utils.ShowChatName(msg.SenderChat),
				FromID:    msg.SenderChat.ID,
				ChatID:    msg.Chat.ID,
				MessageID: msg.ID,
			}
		} else if msg.From != nil {
			return &OriginInfo{
				FromName:  utils.ShowUserName(msg.From),
				FromID:    msg.From.ID,
				ChatID:    msg.Chat.ID,
				MessageID: msg.ID,
			}
		}
	}

	return nil
}

// build a InlineKeyboardButton for origin info
func (o *OriginInfo) BuildButton() models.ReplyMarkup {
	if o == nil {
		return nil
	}
	var buttons []models.InlineKeyboardButton

	if o.FromID != 0 {
		if o.FromID < 0 {
			// -100 开头的 ID，为群组或频道
			buttons = append(buttons, models.InlineKeyboardButton{
				Text: "来自 " + o.FromName,
				URL:  fmt.Sprintf("https://t.me/c/%s/0", utils.RemoveIDPrefix(o.FromID)),
			})
		} else {
			buttons = append(buttons, models.InlineKeyboardButton{
				Text: "来自用户 " + o.FromName,
				URL:  fmt.Sprintf("tg://user?id=%d", o.FromID),
			})
		}
	}
	if o.MessageID != 0 {
		if o.ChatID == 0 {
			// 保存来源是频道
			buttons = append(buttons, models.InlineKeyboardButton{
				Text: "查看消息",
				URL: utils.MsgLinkPrivate(o.FromID, o.MessageID),
			})
		} else {
			// 从群组中保存的消息
			buttons = append(buttons, models.InlineKeyboardButton{
				Text: "查看消息",
				URL: utils.MsgLinkPrivate(o.ChatID, o.MessageID),
			})
		}
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{buttons}}
}
