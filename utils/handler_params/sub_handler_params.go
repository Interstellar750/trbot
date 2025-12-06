package handler_params

import (
	"context"

	"trle5.xyz/gopkg/trbot/database/db_struct"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// 调用子处理函数时的传递的参数，避免重复获取
type Update struct {
	Ctx      context.Context
	Thebot   *bot.Bot
	Update   *models.Update
	ChatInfo *db_struct.ChatInfo
}

type Message struct {
	Ctx      context.Context
	Thebot   *bot.Bot
	Message  *models.Message
	ChatInfo *db_struct.ChatInfo
	Fields   []string
}

type EditMessage struct {
	Ctx         context.Context
	Thebot      *bot.Bot
	EditMessage *models.Message
	ChatInfo    *db_struct.ChatInfo
}

type ChannelPost struct {
	Ctx         context.Context
	Thebot      *bot.Bot
	ChannelPost *models.Message
	ChatInfo    *db_struct.ChatInfo
}

type EditedChannelPost struct {
	Ctx                context.Context
	Thebot             *bot.Bot
	EditedChannelPost  *models.Message
	ChatInfo           *db_struct.ChatInfo
}

type BusinessConnection struct {
	Ctx                context.Context
	Thebot             *bot.Bot
	BusinessConnection *models.BusinessConnection
	ChatInfo           *db_struct.ChatInfo
}

type BusinessMessage struct {
	Ctx             context.Context
	Thebot          *bot.Bot
	BusinessMessage *models.Message
	ChatInfo        *db_struct.ChatInfo
}

type EditedBusinessMessage struct {
	Ctx                   context.Context
	Thebot                *bot.Bot
	EditedBusinessMessage *models.Message
	ChatInfo              *db_struct.ChatInfo
}

type DeletedBusinessMessages struct {
	Ctx                     context.Context
	Thebot                  *bot.Bot
	BusinessMessagesDeleted *models.BusinessMessagesDeleted
	ChatInfo                *db_struct.ChatInfo
}

type MessageReaction struct {
	Ctx             context.Context
	Thebot          *bot.Bot
	MessageReaction *models.MessageReactionUpdated
	ChatInfo        *db_struct.ChatInfo
}

type MessageReactionCount struct {
	Ctx                  context.Context
	Thebot               *bot.Bot
	MessageReactionCount *models.MessageReactionCountUpdated
	ChatInfo             *db_struct.ChatInfo
}

type InlineQuery struct {
	Ctx         context.Context
	Thebot      *bot.Bot
	InlineQuery *models.InlineQuery
	ChatInfo    *db_struct.ChatInfo
	Fields      []string
}

type ChosenInlineResult struct {
	Ctx                context.Context
	Thebot             *bot.Bot
	ChosenInlineResult *models.ChosenInlineResult
	ChatInfo           *db_struct.ChatInfo
	Fields             []string
}

type CallbackQuery struct {
	Ctx           context.Context
	Thebot        *bot.Bot
	CallbackQuery *models.CallbackQuery
	ChatInfo      *db_struct.ChatInfo
}

type ShippingQuery struct {
	Ctx           context.Context
	Thebot        *bot.Bot
	ShippingQuery *models.ShippingQuery
	ChatInfo      *db_struct.ChatInfo
}

type PreCheckoutQuery struct {
	Ctx              context.Context
	Thebot           *bot.Bot
	PreCheckoutQuery *models.PreCheckoutQuery
	ChatInfo         *db_struct.ChatInfo
}

type PurchasedPaidMedia struct {
	Ctx                context.Context
	Thebot             *bot.Bot
	PurchasedPaidMedia *models.PaidMediaPurchased
	ChatInfo           *db_struct.ChatInfo
}

type Poll struct {
	Ctx      context.Context
	Thebot   *bot.Bot
	Poll     *models.Poll
	ChatInfo *db_struct.ChatInfo
}

type PollAnswer struct {
	Ctx        context.Context
	Thebot     *bot.Bot
	PollAnswer *models.PollAnswer
	ChatInfo   *db_struct.ChatInfo
}

type MyChatMember struct {
	Ctx          context.Context
	Thebot       *bot.Bot
	MyChatMember *models.ChatMemberUpdated
	ChatInfo     *db_struct.ChatInfo
}

type ChatMember struct {
	Ctx        context.Context
	Thebot     *bot.Bot
	ChatMember *models.ChatMemberUpdated
	ChatInfo   *db_struct.ChatInfo
}

type ChatJoinRequest struct {
	Ctx             context.Context
	Thebot          *bot.Bot
	ChatJoinRequest *models.ChatJoinRequest
	ChatInfo        *db_struct.ChatInfo
}

type ChatBoost struct {
	Ctx       context.Context
	Thebot    *bot.Bot
	ChatBoost *models.ChatBoostUpdated
	ChatInfo  *db_struct.ChatInfo
}

type RemovedChatBoost struct {
	Ctx              context.Context
	Thebot           *bot.Bot
	RemovedChatBoost *models.ChatBoostRemoved
	ChatInfo         *db_struct.ChatInfo
}
