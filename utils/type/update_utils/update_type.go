package update_utils

import (
	"reflect"

	"github.com/go-telegram/bot/models"
)

// 更新类型
type Update struct {
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

// 将消息类型结构体转换为对应的 type
func (u Update)AsType() Type {
	val := reflect.ValueOf(u)
	typ := reflect.TypeOf(u)

	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).Bool() {
			return Type(typ.Field(i).Name)
		}
	}

	return ""
}

// 将消息类型结构体转换为对应的 model
func (u Update)AsModel() Model {
	switch u.AsType() {
	case Message:                 return ModelMessage
	case EditedMessage:           return ModelMessage
	case ChannelPost:             return ModelMessage
	case EditedChannelPost:       return ModelMessage
	case BusinessConnection:      return ModelBusinessConnection
	case BusinessMessage:         return ModelMessage
	case EditedBusinessMessage:   return ModelMessage
	case DeletedBusinessMessages: return ModelBusinessMessagesDeleted
	case MessageReaction:         return ModelMessageReactionUpdated
	case MessageReactionCount:    return ModelMessageReactionCountUpdated
	case InlineQuery:             return ModelInlineQuery
	case ChosenInlineResult:      return ModelChosenInlineResult
	case CallbackQuery:           return ModelCallbackQuery
	case ShippingQuery:           return ModelShippingQuery
	case PreCheckoutQuery:        return ModelPreCheckoutQuery
	case PurchasedPaidMedia:      return ModelPaidMediaPurchased
	case Poll:                    return ModelPoll
	case PollAnswer:              return ModelPollAnswer
	case MyChatMember:            return ModelChatMemberUpdated
	case ChatMember:              return ModelChatMemberUpdated
	case ChatJoinRequest:         return ModelChatJoinRequest
	case ChatBoost:               return ModelChatBoostUpdated
	case RemovedChatBoost:        return ModelChatBoostRemoved
	default:
		return ""
	}
}

func (u Update)Str() string {
	val := reflect.ValueOf(u)
	typ := reflect.TypeOf(u)

	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).Bool() {
			return string(Type(typ.Field(i).Name))
		}
	}

	return ""
}

type Type string

const (
	Message                 Type = "Message"
	EditedMessage          	Type = "EditedMessage"
	ChannelPost            	Type = "ChannelPost"
	EditedChannelPost      	Type = "EditedChannelPost"
	BusinessConnection     	Type = "BusinessConnection"
	BusinessMessage        	Type = "BusinessMessage"
	EditedBusinessMessage  	Type = "EditedBusinessMessage"
	DeletedBusinessMessages	Type = "DeletedBusinessMessages"
	MessageReaction        	Type = "MessageReaction"
	MessageReactionCount   	Type = "MessageReactionCount"
	InlineQuery            	Type = "InlineQuery"
	ChosenInlineResult     	Type = "ChosenInlineResult"
	CallbackQuery          	Type = "CallbackQuery"
	ShippingQuery          	Type = "ShippingQuery"
	PreCheckoutQuery       	Type = "PreCheckoutQuery"
	PurchasedPaidMedia     	Type = "PurchasedPaidMedia"
	Poll                   	Type = "Poll"
	PollAnswer             	Type = "PollAnswer"
	MyChatMember           	Type = "MyChatMember"
	ChatMember             	Type = "ChatMember"
	ChatJoinRequest        	Type = "ChatJoinRequest"
	ChatBoost              	Type = "ChatBoost"
	RemovedChatBoost       	Type = "RemovedChatBoost"
)

func (t Type)Str() string {
	return string(t)
}

type Model string

const (
	ModelMessage                     Model = "Message"
	ModelBusinessConnection          Model = "BusinessConnection"
	ModelBusinessMessagesDeleted     Model = "BusinessMessagesDeleted"
	ModelMessageReactionUpdated      Model = "MessageReactionUpdated"
	ModelMessageReactionCountUpdated Model = "MessageReactionCountUpdated"
	ModelInlineQuery                 Model = "InlineQuery"
	ModelChosenInlineResult          Model = "ChosenInlineResult"
	ModelCallbackQuery               Model = "CallbackQuery"
	ModelShippingQuery               Model = "ShippingQuery"
	ModelPreCheckoutQuery            Model = "PreCheckoutQuery"
	ModelPaidMediaPurchased          Model = "PaidMediaPurchased"
	ModelPoll                        Model = "Poll"
	ModelPollAnswer                  Model = "PollAnswer"
	ModelChatMemberUpdated           Model = "ChatMemberUpdated"
	ModelChatJoinRequest             Model = "ChatJoinRequest"
	ModelChatBoostUpdated            Model = "ChatBoostUpdated"
	ModelChatBoostRemoved            Model = "ChatBoostRemoved"
)

func (m Model)Str() string {
	return string(m)
}

func (m Model)Types() []Type {
	switch m {
	case ModelMessage:
		return []Type{
			Message,
			ChannelPost,
			BusinessMessage,
		}
	}
	return []Type{}
}

// 判断更新的类型
func GetUpdateType(update *models.Update) Update {
	return Update{
		Message:                 update.Message != nil,
		EditedMessage:           update.EditedMessage != nil,
		ChannelPost:             update.ChannelPost != nil,
		EditedChannelPost:       update.EditedChannelPost != nil,
		BusinessConnection:      update.BusinessConnection != nil,
		BusinessMessage:         update.BusinessMessage != nil,
		EditedBusinessMessage:   update.EditedBusinessMessage != nil,
		DeletedBusinessMessages: update.DeletedBusinessMessages != nil,
		MessageReaction:         update.MessageReaction != nil,
		MessageReactionCount:    update.MessageReactionCount != nil,
		InlineQuery:             update.InlineQuery != nil,
		ChosenInlineResult:      update.ChosenInlineResult != nil,
		CallbackQuery:           update.CallbackQuery != nil,
		ShippingQuery:           update.ShippingQuery != nil,
		PreCheckoutQuery:        update.PreCheckoutQuery != nil,
		PurchasedPaidMedia:      update.PurchasedPaidMedia != nil,
		Poll:                    update.Poll != nil,
		PollAnswer:              update.PollAnswer != nil,
		MyChatMember:            update.MyChatMember != nil,
		ChatMember:              update.ChatMember != nil,
		ChatJoinRequest:         update.ChatJoinRequest != nil,
		ChatBoost:               update.ChatBoost != nil,
		RemovedChatBoost:        update.RemovedChatBoost != nil,
	}
}
