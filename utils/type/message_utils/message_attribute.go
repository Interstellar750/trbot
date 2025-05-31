package message_utils

import "github.com/go-telegram/bot/models"

// 消息属性
type MessageAttribute struct {
	IsFromAnonymous      bool `yaml:"IsFromAnonymous,omitempty"`      // anonymous admin or owner in group/supergroup
	IsFromLinkedChannel  bool `yaml:"IsFromLinkedChannel,omitempty"`  // is automatic forward post from linked channel
	IsUserAsChannel      bool `yaml:"IsUserAsChannel,omitempty"`      // user selected to send message as a channel
	IsHasSenderChat      bool `yaml:"IsHasSenderChat,omitempty"`      // sender of the message when sent on behalf of a chat, eg current group/supergroup or linked channel
	IsChatEnableForum    bool `yaml:"IsChatEnableForum,omitempty"`    // group or supergroup is enable topic
	IsForwardMessage     bool `yaml:"IsForwardMessage,omitempty"`     // not a origin message, forward from somewhere
	IsTopicMessage       bool `yaml:"IsTopicMessage,omitempty"`       // the message is sent to a forum topic
	IsAutomaticForward   bool `yaml:"IsAutomaticForward,omitempty"`   // is post from linked channel, auto forward by server
	IsReplyToMessage     bool `yaml:"IsReplyToMessage,omitempty"`     // message reply to a another message
	IsExternalReply      bool `yaml:"IsExternalReply,omitempty"`      // message reply from another chat, or call 'quote'
	IsQuoteToMessage     bool `yaml:"IsQuoteToMessage,omitempty"`     // reply from another chat or manual quote from current chat, maybe only true for text message
	IsQuoteHasEntities   bool `yaml:"IsQuoteHasEntities,omitempty"`   // is quote message has entities
	IsManualQuote        bool `yaml:"IsManualQuote,omitempty"`        // user manually select text to quote a message. if false, just use 'reply to other chat'
	IsReplyToStory       bool `yaml:"IsReplyToStory,omitempty"`       // TODO
	IsViaBot             bool `yaml:"IsViaBot,omitempty"`             // message by inline mode
	IsEdited             bool `yaml:"IsEdited,omitempty"`             // message aready edited
	IsFromOffline        bool `yaml:"IsFromOffline,omitempty"`        // eg scheduled message
	IsGroupedMedia       bool `yaml:"IsGroupedMedia,omitempty"`       // media group, like select more than one file or photo to send
	IsTextHasEntities    bool `yaml:"IsTextHasEntities,omitempty"`    // message has text entities
	IsMessageHasEffect   bool `yaml:"IsMessageHasEffect,omitempty"`   // message has effect
	IsCaptionHasEntities bool `yaml:"IsCaptionHasEntities,omitempty"` // message has caption entities
	IsCaptionAboveMedia  bool `yaml:"IsCaptionAboveMedia,omitempty"`  // set the caption to appear before the content when sending videos or photos
	IsMediaHasSpoiler    bool `yaml:"IsMediaHasSpoiler,omitempty"`    // image or video has a spoiler
	IsHasInlineKeyboard  bool `yaml:"IsHasInlineKeyboard,omitempty"`  // message has inline keyboard
}

// 判断消息属性
func GetMessageAttribute(msg *models.Message) MessageAttribute {
	var attribute MessageAttribute
	if msg.SenderChat != nil {
		attribute.IsHasSenderChat = true
		if msg.From != nil {
			if msg.From.IsBot {
				if msg.From.ID == 1087968824 && msg.SenderChat.ID == msg.Chat.ID {
					attribute.IsFromAnonymous = true
				}
				if msg.From.ID == 136817688 {
					attribute.IsUserAsChannel = true
				}
			}
			if msg.From.ID == 777000 && msg.ForwardOrigin != nil && msg.ForwardOrigin.MessageOriginChannel != nil && msg.SenderChat.ID == msg.ForwardOrigin.MessageOriginChannel.Chat.ID {
				attribute.IsFromLinkedChannel = true
			}
		}
	}
	if msg.Chat.IsForum {
		attribute.IsChatEnableForum = true
	}
	if msg.ForwardOrigin != nil {
		attribute.IsForwardMessage = true
	}
	if msg.IsTopicMessage {
		attribute.IsTopicMessage = true
	}
	if msg.IsAutomaticForward {
		attribute.IsAutomaticForward = true
	}
	if msg.ReplyToMessage != nil {
		attribute.IsReplyToMessage = true
	}
	if msg.ExternalReply != nil {
		attribute.IsExternalReply = true
	}
	if msg.Quote != nil {
		attribute.IsQuoteToMessage = true
		if msg.Quote.Entities != nil {
			attribute.IsQuoteHasEntities = true
		}
		if msg.Quote.IsManual {
			attribute.IsManualQuote = true
		}
	}
	if msg.ReplyToStore != nil {
		attribute.IsReplyToStory = true
	}
	if msg.ViaBot != nil {
		attribute.IsViaBot = true
	}
	if msg.EditDate != 0 {
		attribute.IsEdited = true
	}
	if msg.IsFromOffline {
		attribute.IsFromOffline = true
	}
	if msg.MediaGroupID != "" {
		attribute.IsGroupedMedia = true
	}
	if msg.Entities != nil {
		attribute.IsTextHasEntities = true
	}
	if msg.EffectID != "" {
		attribute.IsMessageHasEffect = true
	}
	if msg.CaptionEntities != nil {
		attribute.IsCaptionHasEntities = true
	}
	if msg.ShowCaptionAboveMedia {
		attribute.IsCaptionAboveMedia = true
	}
	if msg.HasMediaSpoiler {
		attribute.IsMediaHasSpoiler = true
	}
	if msg.ReplyMarkup != nil && len(msg.ReplyMarkup.InlineKeyboard) > 0 {
		attribute.IsHasInlineKeyboard = true
	}
	return attribute
}
