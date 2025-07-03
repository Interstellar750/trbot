package update_utils

import (
	"reflect"

	"github.com/go-telegram/bot/models"
)

// 更新类型
type UpdateType struct {
	Message                 bool `yaml:"Message,omitempty"`                 // *models.Message
	EditedMessage           bool `yaml:"EditedMessage,omitempty"`           // *models.Message
	ChannelPost             bool `yaml:"ChannelPost,omitempty"`             // *models.Message
	EditedChannelPost       bool `yaml:"EditedChannelPost,omitempty"`       // *models.Message
	BusinessConnection      bool `yaml:"BusinessConnection,omitempty"`      // *models.BusinessConnection
	BusinessMessage         bool `yaml:"BusinessMessage,omitempty"`         // *models.Message
	EditedBusinessMessage   bool `yaml:"EditedBusinessMessage,omitempty"`   // *models.Message
	DeletedBusinessMessages bool `yaml:"DeletedBusinessMessages,omitempty"` // *models.BusinessMessagesDeleted
	MessageReaction         bool `yaml:"MessageReaction,omitempty"`         // *models.MessageReactionUpdated
	MessageReactionCount    bool `yaml:"MessageReactionCount,omitempty"`    // *models.MessageReactionCountUpdated
	InlineQuery             bool `yaml:"InlineQuery,omitempty"`             // *models.InlineQuery
	ChosenInlineResult      bool `yaml:"ChosenInlineResult,omitempty"`      // *models.ChosenInlineResult
	CallbackQuery           bool `yaml:"CallbackQuery,omitempty"`           // *models.CallbackQuery
	ShippingQuery           bool `yaml:"ShippingQuery,omitempty"`           // *models.ShippingQuery
	PreCheckoutQuery        bool `yaml:"PreCheckoutQuery,omitempty"`        // *models.PreCheckoutQuery
	PurchasedPaidMedia      bool `yaml:"PurchasedPaidMedia,omitempty"`      // *models.PaidMediaPurchased
	Poll                    bool `yaml:"Poll,omitempty"`                    // *models.Poll
	PollAnswer              bool `yaml:"PollAnswer,omitempty"`              // *models.PollAnswer
	MyChatMember            bool `yaml:"MyChatMember,omitempty"`            // *models.ChatMemberUpdated
	ChatMember              bool `yaml:"ChatMember,omitempty"`              // *models.ChatMemberUpdated
	ChatJoinRequest         bool `yaml:"ChatJoinRequest,omitempty"`         // *models.ChatJoinRequest
	ChatBoost               bool `yaml:"ChatBoost,omitempty"`               // *models.ChatBoostUpdated
	RemovedChatBoost        bool `yaml:"RemovedChatBoost,omitempty"`        // *models.ChatBoostRemoved
}

// 将消息类型结构体转换为 UpdateTypeList(string) 类型
func (ut UpdateType)InString() UpdateTypeList {
	val := reflect.ValueOf(ut)
	typ := reflect.TypeOf(ut)

	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).Bool() {
			return UpdateTypeList(typ.Field(i).Name)
		}
	}

	return ""
}

type UpdateTypeList string

const (
	Message                 UpdateTypeList = "Message"
	EditedMessage          	UpdateTypeList = "EditedMessage"
	ChannelPost            	UpdateTypeList = "ChannelPost"
	EditedChannelPost      	UpdateTypeList = "EditedChannelPost"
	BusinessConnection     	UpdateTypeList = "BusinessConnection"
	BusinessMessage        	UpdateTypeList = "BusinessMessage"
	EditedBusinessMessage  	UpdateTypeList = "EditedBusinessMessage"
	DeletedBusinessMessages	UpdateTypeList = "DeletedBusinessMessages"
	MessageReaction        	UpdateTypeList = "MessageReaction"
	MessageReactionCount   	UpdateTypeList = "MessageReactionCount"
	InlineQuery            	UpdateTypeList = "InlineQuery"
	ChosenInlineResult     	UpdateTypeList = "ChosenInlineResult"
	CallbackQuery          	UpdateTypeList = "CallbackQuery"
	ShippingQuery          	UpdateTypeList = "ShippingQuery"
	PreCheckoutQuery       	UpdateTypeList = "PreCheckoutQuery"
	PurchasedPaidMedia     	UpdateTypeList = "PurchasedPaidMedia"
	Poll                   	UpdateTypeList = "Poll"
	PollAnswer             	UpdateTypeList = "PollAnswer"
	MyChatMember           	UpdateTypeList = "MyChatMember"
	ChatMember             	UpdateTypeList = "ChatMember"
	ChatJoinRequest        	UpdateTypeList = "ChatJoinRequest"
	ChatBoost              	UpdateTypeList = "ChatBoost"
	RemovedChatBoost       	UpdateTypeList = "RemovedChatBoost"
)

// 判断更新的类型
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
