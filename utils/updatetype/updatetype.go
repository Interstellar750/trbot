package updatetype

import "github.com/go-telegram/bot/models"

var All []MessageType

// 消息类型
type MessageType struct {
	// https://core.telegram.org/bots/api#message

	Animation bool // call gif, mpeg4 format, can save to GIFs, no caption
	Audio     bool // or call music, can have caption, some music may as a document
	Document  bool // can have caption
	PaidMedia bool // photo or video, unknow caption
	Photo     bool // a list, sort by resolution
	Sticker   bool // sticker, but some .webp file maybe will send as sticker, actual file format and resolution may not match the limitations. no caption
	Story     bool
	Video     bool
	VideoNote bool // A circular video shot in Telegram
	Voice     bool // can have caption
	OnlyText  bool // just text message
	Contact   bool
	Dice      bool
	Game      bool
	Poll      bool
	Venue     bool
	Location  bool
	Invoice   bool
	Giveaway   bool
}
// 判断消息的类型
func GetMessageType(msg *models.Message) MessageType {
	var msgType MessageType
	if msg.Document != nil {
		if msg.Animation != nil && msg.Animation.FileID == msg.Document.FileID && msg.Document.MimeType == "video/mp4" {
			msgType.Animation = true
		} else {
			msgType.Document = true
		}
	}
	if msg.Audio != nil {
		msgType.Audio = true
	}
	if msg.PaidMedia != nil {
		msgType.PaidMedia = true
	}
	if msg.Photo != nil {
		msgType.Photo = true
	}
	if msg.Sticker != nil {
		msgType.Sticker = true
	}
	if msg.Story != nil {
		msgType.Story = true
	}
	if msg.Video != nil {
		msgType.Video = true
	}
	if msg.VideoNote != nil {
		msgType.VideoNote = true
	}
	if msg.Voice != nil {
		msgType.Voice = true
	}
	if msg.Contact != nil {
		msgType.Contact = true
	}
	if msg.Dice != nil {
		msgType.Dice = true
	}
	if msg.Game != nil {
		msgType.Game = true
	}
	if msg.Poll != nil {
		msgType.Poll = true
	}
	if msg.Venue != nil {
		msgType.Venue = true
	}
	if msg.Location != nil {
		msgType.Location = true
	}
	if msg.Invoice != nil {
		msgType.Invoice = true
	}
	if msg.Giveaway != nil {
		msgType.Giveaway = true
	}
	if msg.Text != "" {
		msgType.OnlyText = true
	}
	return msgType
}
// 消息属性
type MessageAttribute struct {
	IsFromAnonymous      bool // anonymous admin or owner in group/supergroup
	IsFromLinkedChannel  bool // is automatic forward post from linked channel
	IsUserAsChannel      bool // user selected to send message as a channel
	IsHasSenderChat      bool // sender of the message when sent on behalf of a chat, eg current group/supergroup or linked channel
	IsChatEnableForum    bool // group or supergroup is enable topic
	IsForwardMessage     bool // not a origin message, forward from somewhere
	IsTopicMessage       bool // the message is sent to a forum topic
	IsAutomaticForward   bool // is post from linked channel, auto forward by server
	IsReplyToMessage     bool // message reply to a another message
	IsExternalReply      bool // message reply from another chat, or call 'quote'
	IsQuoteToMessage     bool // reply from another chat or manual quote from current chat, maybe only true for text message
	IsQuoteHasEntities   bool // is quote message has entities
	IsManualQuote        bool // user manually select text to quote a message. if false, just use 'reply to other chat'
	IsReplyToStory       bool // TODO
	IsViaBot             bool // message by inline mode
	IsEdited             bool // message aready edited
	IsFromOffline        bool // eg scheduled message
	IsGroupedMedia       bool // media group, like select more than one file or photo to send
	IsTextHasEntities    bool // message has text entities
	IsMessageHasEffect   bool // message has effect
	IsCaptionHasEntities bool // message has caption entities
	IsCaptionAboveMedia  bool // set the caption to appear before the content when sending videos or photos
	IsMediaHasSpoiler    bool // image or video has a spoiler
	IsHasInlineKeyboard  bool // message has inline keyboard
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
	if len(msg.ReplyMarkup.InlineKeyboard) > 0 {
		attribute.IsHasInlineKeyboard = true
	}
	return attribute
}
// 更新类型
type UpdateType struct {
	Message                 bool // *models.Message
	EditedMessage           bool // *models.Message
	ChannelPost             bool // *models.Message
	EditedChannelPost       bool // *models.Message
	BusinessConnection      bool // *models.BusinessConnection
	BusinessMessage         bool // *models.Message
	EditedBusinessMessage   bool // *models.Message
	DeletedBusinessMessages bool // *models.BusinessMessagesDeleted
	MessageReaction         bool // *models.MessageReactionUpdated
	MessageReactionCount    bool // *models.MessageReactionCountUpdated
	InlineQuery             bool // *models.InlineQuery
	ChosenInlineResult      bool // *models.ChosenInlineResult
	CallbackQuery           bool // *models.CallbackQuery
	ShippingQuery           bool // *models.ShippingQuery
	PreCheckoutQuery        bool // *models.PreCheckoutQuery
	PurchasedPaidMedia      bool // *models.PaidMediaPurchased
	Poll                    bool // *models.Poll
	PollAnswer              bool // *models.PollAnswer
	MyChatMember            bool // *models.ChatMemberUpdated
	ChatMember              bool // *models.ChatMemberUpdated
	ChatJoinRequest         bool // *models.ChatJoinRequest
	ChatBoost               bool // *models.ChatBoostUpdated
	RemovedChatBoost        bool // *models.ChatBoostRemoved
}

func GetUpdateType(update *models.Update) UpdateType {
	var updateType UpdateType
	if update.Message != nil {
		updateType.Message = true
	}
	if update.EditedMessage != nil {
		updateType.EditedMessage = true
	}
	if update.ChannelPost != nil {
		updateType.ChannelPost = true
	}
	if update.EditedChannelPost != nil {
		updateType.EditedChannelPost = true
	}
	if update.BusinessConnection != nil {
		updateType.BusinessConnection = true
	}
	if update.BusinessMessage != nil {
		updateType.BusinessMessage = true
	}
	if update.EditedBusinessMessage != nil {
		updateType.EditedBusinessMessage = true
	}
	if update.MessageReaction != nil {
		updateType.MessageReaction = true
	}
	if update.MessageReactionCount != nil {
		updateType.MessageReactionCount = true
	}
	if update.InlineQuery != nil {
		updateType.InlineQuery = true
	}
	if update.ChosenInlineResult != nil {
		updateType.ChosenInlineResult = true
	}
	if update.CallbackQuery != nil {
		updateType.CallbackQuery = true
	}
	if update.ShippingQuery != nil {
		updateType.ShippingQuery = true
	}
	if update.PreCheckoutQuery != nil {
		updateType.PreCheckoutQuery = true
	}
	if update.PurchasedPaidMedia != nil {
		updateType.PurchasedPaidMedia = true
	}
	if update.Poll != nil {
		updateType.Poll = true
	}
	if update.PollAnswer != nil {
		updateType.PollAnswer = true
	}
	if update.MyChatMember != nil {
		updateType.MyChatMember = true
	}
	if update.ChatMember != nil {
		updateType.ChatMember = true
	}
	if update.ChatJoinRequest != nil {
		updateType.ChatJoinRequest = true
	}
	if update.ChatBoost != nil {
		updateType.ChatBoost = true
	}
	if update.RemovedChatBoost != nil {
		updateType.RemovedChatBoost = true
	}
	return updateType
}
