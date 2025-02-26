package main

import (
	"fmt"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Plugin_All struct {
	Inline              []Plugin_Inline
	InlineManual        []Plugin_InlineManual
	SlashStart           *Plugin_SlashStart
	SlashCommand        []Plugin_SlashCommand
	NoSymbolCommand     []Plugin_NoSymbolCommand
	CustomSymbolCommand []Plugin_CustomSymbolCommand
	CallbackQuery       []Plugin_CallbackQuery

	// MessageTypeForPrivateChat *Plugin_
}

var AllPugins = Plugin_All{}

// 需要返回一个列表，将由程序的分页函数来控制分页和输出
type Plugin_Inline struct {
    command string
    handler func(*subHandlerOpts) []models.InlineQueryResult
}

func AddInlineHandlerPlugins(InlineHandlerPlugins ...Plugin_Inline) int {
	if AllPugins.Inline == nil { AllPugins.Inline = []Plugin_Inline{} }
	var pluginCount int
	for _, originPlugin := range AllPugins.Inline {
		AllPugins.Inline = append(AllPugins.Inline, originPlugin)
		pluginCount++
	}
	return pluginCount
}

// 完全由插件自行控制输出
type Plugin_InlineManual struct {
    command string
    handler func(*subHandlerOpts)
}

func AddInlineManualHandlerPlugins(InlineManualHandlerPlugins ...Plugin_InlineManual) int {
	if AllPugins.InlineManual == nil { AllPugins.InlineManual = []Plugin_InlineManual{} }
	var pluginCount int
	for _, originPlugin := range AllPugins.InlineManual {
		AllPugins.InlineManual = append(AllPugins.InlineManual, originPlugin)
		pluginCount++
	}
	return pluginCount
}


type Plugin_SlashStart struct {
	handler           []SlashStartHandler
	withPrefixHandler []SlashStartWithPrefixHandler
}

type SlashStartHandler struct {
	argument string
	handler  func(*subHandlerOpts)
}

type SlashStartWithPrefixHandler struct {
	prefix   string
	argument string
	handler  func(*subHandlerOpts)
}

func AddSlashStartCommandPlugins(SlashStartCommandPlugins ...SlashStartHandler) int {
	if AllPugins.SlashStart == nil { AllPugins.SlashStart = &Plugin_SlashStart{} }
	if AllPugins.SlashStart.handler == nil { AllPugins.SlashStart.handler = []SlashStartHandler{} }

	var pluginCount int
	for _, originPlugin := range SlashStartCommandPlugins {
		AllPugins.SlashStart.handler = append(AllPugins.SlashStart.handler, originPlugin)
		pluginCount++
	}
	return pluginCount
}

func AddSlashStartWithPrefixCommandPlugins(SlashStartWithPrefixCommandPlugins ...SlashStartWithPrefixHandler) int {
	if AllPugins.SlashStart == nil { AllPugins.SlashStart = &Plugin_SlashStart{} }
	if AllPugins.SlashStart.withPrefixHandler == nil { AllPugins.SlashStart.withPrefixHandler = []SlashStartWithPrefixHandler{} }

	var pluginCount int
	for _, originPlugin := range SlashStartWithPrefixCommandPlugins {
		AllPugins.SlashStart.withPrefixHandler = append(AllPugins.SlashStart.withPrefixHandler, originPlugin)
		pluginCount++
	}
	return pluginCount
}

type Plugin_SlashCommand struct {
	command string
	handler func(*subHandlerOpts)
}

type Plugin_NoSymbolCommand struct {
	command string
	handler func(*subHandlerOpts)
}

type Plugin_CustomSymbolCommand struct {
	symbol  string
	command string
	handler func(*subHandlerOpts)
}

// 为了兼容性考虑，建议仅将 commandChar 设置为单个字符（区分大小写），
// 因为 CallbackQuery 有长度限制，为 64 个字符，而贴纸包名的长度最大为 62。
// 再使用一个符号来隔开内容时，实际上能使用的识别字符长度只有一个字符。
// 你也可以忽略这个提醒，但在发送消息时使用 ReplyMarkup 参数添加按钮的时候，
// 需要评断并控制一下 CallbackData 的长度是否超过了 64 个字符，否则消息会无法发出。
// 或许用户发送的 Callback 请求，其 Query  可能会出现大小写不同，但服务器认为是同一个请求的情况，
// 建议为一个 handler 设定一个字符，同时捕获大小写
type Plugin_CallbackQuery struct {
	commandChar string
	handler func(*subHandlerOpts)
}

func AddCallbackQueryCommandPlugins(Plugins ...Plugin_CallbackQuery) int {
	if AllPugins.CallbackQuery == nil { AllPugins.CallbackQuery = []Plugin_CallbackQuery{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.CallbackQuery = append(AllPugins.CallbackQuery, originPlugin)
		pluginCount++
	}
	return pluginCount
}


func AddPlugins(Plugins ...interface{}) int {
	var pluginCount int
	for _, originPlugin := range Plugins {
		switch plugin := originPlugin.(type) {
		case SlashStartHandler:
			AllPugins.SlashStart.handler = append(AllPugins.SlashStart.handler, plugin)
			pluginCount++
		case []SlashStartHandler:
			AllPugins.SlashStart.handler = append(AllPugins.SlashStart.handler, plugin...)
			pluginCount++
		case SlashStartWithPrefixHandler:
			AllPugins.SlashStart.withPrefixHandler = append(AllPugins.SlashStart.withPrefixHandler, plugin)
			pluginCount++
		case []SlashStartWithPrefixHandler:
			AllPugins.SlashStart.withPrefixHandler = append(AllPugins.SlashStart.withPrefixHandler, plugin...)
			pluginCount++
		default:
			log.Printf("Unknown plugin type: %T, skipped", plugin)
		}
	}
	return pluginCount
}

func InitPlugins() {
	AddSlashStartWithPrefixCommandPlugins(SlashStartWithPrefixHandler{
		"via-inline",
		"test",
		func(opts *subHandlerOpts) {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID:          opts.update.Message.Chat.ID,
				Text:            "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
				ReplyParameters: &models.ReplyParameters{MessageID: opts.update.Message.ID},
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击加入测试群组",
					URL:  "https://t.me/+BomkHuFsjqc3ZGE1",
				}}}},
			})
		},
	})

	AddSlashStartCommandPlugins(SavedMessage_StartCommandHandlers...)
	AddSlashStartWithPrefixCommandPlugins(SavedMessage_StartCommandWithPrefixHandlers...)

	AddInlineManualHandlerPlugins(Plugin_InlineManual{
		command: "uaav",
		handler: func(opts *subHandlerOpts) {
			if len(opts.fields) < 2 {
				_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:    "custom voices",
							Title: "URL as a voice",
							Description: "接着输入一个音频 URL 来其作为语音样式发送（不会转换格式）",
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
								ParseMode: models.ParseModeMarkdownV1,
							},
						},
					},
				})
				if err != nil {
					printLogAndSave(fmt.Sprintln("some error when answer custom voice tips,", err))
				}
			} else if len(opts.fields) == 2 {
				if strings.HasPrefix(opts.fields[1], "https://") {
					_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultVoice{
								ID: "custom",
								Title: "Custom voice",
								VoiceURL: opts.fields[1],
							},
						},
						IsPersonal: true,
					})
					if err != nil {
						log.Println("Error when answering inline query: ", err)
					}
				} else {
					_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:    "error",
								Title: "音频 URL 格式错误",
								Description: "请确保音频链接以 https:// 作为开头，若填写完整 URL 后此消息依然存在，请检查 URL 是否有效",
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
									ParseMode: models.ParseModeMarkdownV1,
								},
							},
						},
					})
					if err != nil {
						log.Println("Error when answering inline query", err)
					}
				}
			} else {
				_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.update.InlineQuery.ID,
					Results: []models.InlineQueryResult{
						&models.InlineQueryResultArticle{
							ID:    "error",
							Title: "参数过多，请注意空格",
							Description: fmt.Sprintf("使用方法：@%s %suaav <单个音频链接>", botMe.Username, InlineSubCommandSymbol),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
								ParseMode: models.ParseModeMarkdownV1,
							},
						},
					},
				})
				if err != nil {
					log.Println("Error when answering inline query", err)
				}
			}
			return
		},
	})

	AddInlineHandlerPlugins(SavedMessage_InlineCommandHandler)
	AddInlineHandlerPlugins(Udonese_InlineCommandHandler)

	AddCallbackQueryCommandPlugins(Sticker_CallBackQueryHandler...)
}
