package type_utils

import "github.com/go-telegram/bot/models"

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

// 判断更新属性
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
