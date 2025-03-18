package handler_utils

import (
	"context"
	"trbot/utils/database_yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// 调用子处理函数时的传递的参数，避免重复获取
type SubHandlerOpts struct {
	Ctx      context.Context
	Thebot   *bot.Bot
	Update   *models.Update
	ChatInfo *database_yaml.IDInfo
	Fields   []string // 根据请求的类型，可能是消息文本，也可能是 inline 的 query
}
