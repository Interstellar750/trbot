package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Plugin_All struct {
	Inline              []Plugin_Inline              // 函数返回全部列表，由预设函数进行分页
	InlineManual        []Plugin_InlineManual        // 函数自行处理输出
	SlashStart           *Plugin_SlashStart          // '/start' 命令和后面的 query
	SlashSymbolCommand  []Plugin_SlashSymbolCommand  // 以 '/' 符号开头的命令，例如 '/help' '/test'
	CustomSymbolCommand []Plugin_CustomSymbolCommand // 手动定义符号的命令，例如定义符号为 '!'，则命令为 '!help' 或 '!test', 也可以不用不符号，直接 help 或 test
	CallbackQuery       []Plugin_CallbackQuery       // 处理 inline query 的 callback 函数

	// 根据聊天类型设定的默认处理函数
	DefaultHandlerByMessageTypeForPrivate    *Plugin_HandlerByMessageType
	DefaultHandlerByMessageTypeForGroup      *Plugin_HandlerByMessageType
	DefaultHandlerByMessageTypeForSupergroup *Plugin_HandlerByMessageType
	DefaultHandlerByMessageTypeForChannel    *Plugin_HandlerByMessageType
}

var AllPugins = Plugin_All{}

type Plugin_HandlerByMessageType struct {
	Photo   func(*subHandlerOpts)


	Message func(*subHandlerOpts)
	Sticker func(*subHandlerOpts)
	Document func(*subHandlerOpts)
	Audio   func(*subHandlerOpts)
	Video   func(*subHandlerOpts)
	VideoNote func(*subHandlerOpts)
	Voice   func(*subHandlerOpts)
	Contact func(*subHandlerOpts)
	Location func(*subHandlerOpts)

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
	// 触发：'/start via-inline_test'
	AddSlashStartWithPrefixCommandPlugins(SlashStartWithPrefixHandler{
		prefix: "via-inline",
		argument: "test",
		handler: func(opts *subHandlerOpts) {
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


	AddSlashSymbolCommandPlugins(SavedMessage_SlashSymbolCommandHandler)
	AddSlashSymbolCommandPlugins(Plugin_SlashSymbolCommand{
		slashCommand: "chatinfo",
		handler: func(opts *subHandlerOpts) {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				Text: fmt.Sprintf("类型: [<code>%v</code>]\nID: [<code>%v</code>]\n用户名:[<code>%v</code>]", opts.update.Message.Chat.Type, opts.update.Message.Chat.ID, opts.update.Message.Chat.Username),
				ParseMode: models.ParseModeHTML,
			})
		},
	})
	AddSlashSymbolCommandPlugins(Plugin_SlashSymbolCommand{
		slashCommand: "test",
		handler: func(opts *subHandlerOpts) {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				Text: "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{ { {
					Text: "点击加入测试群组",
					URL: "https://t.me/+BomkHuFsjqc3ZGE1",
				}}}},
			})
		},
	})
	AddSlashSymbolCommandPlugins(Plugin_SlashSymbolCommand{
		slashCommand: "fileid",
		handler: func(opts *subHandlerOpts) {
			var pendingMessage string
			if opts.update.Message.ReplyToMessage != nil {
				if opts.update.Message.ReplyToMessage.Sticker != nil {
					pendingMessage = fmt.Sprintf("Type: [Sticker] \nFileID: [<code>%v</code>]", opts.update.Message.ReplyToMessage.Sticker.FileID)
				} else if opts.update.Message.ReplyToMessage.Document != nil {
					pendingMessage = fmt.Sprintf("Type: [Document] \nFileID: [<code>%v</code>]", opts.update.Message.ReplyToMessage.Document.FileID)
				} else if opts.update.Message.ReplyToMessage.Photo != nil {
					pendingMessage = "Type: [Photo]\n"
					if len(opts.fields) > 1 && opts.fields[1] == "all" { // 如果有 all 指示，显示图片所有分辨率的 File ID
						for i, n := range opts.update.Message.ReplyToMessage.Photo {
							pendingMessage += fmt.Sprintf("\nPhotoID_%d: W:%d H:%d Size:%d \n[<code>%s</code>]\n", i, n.Width, n.Height, n.FileSize, n.FileID)
						}
					} else { // 否则显示最后一个的 File ID (应该是最高分辨率的)
						pendingMessage += fmt.Sprintf("PhotoID: [<code>%s</code>]\n", opts.update.Message.ReplyToMessage.Photo[len(opts.update.Message.ReplyToMessage.Photo)-1].FileID)
					}
				} else if opts.update.Message.ReplyToMessage.Video != nil {
					pendingMessage = fmt.Sprintf("Type: [Video] \nFileID: [<code>%v</code>]", opts.update.Message.ReplyToMessage.Video.FileID)
				} else if opts.update.Message.ReplyToMessage.Voice != nil {
					pendingMessage = fmt.Sprintf("Type: [Voice] \nFileID: [<code>%v</code>]", opts.update.Message.ReplyToMessage.Voice.FileID)
				} else if opts.update.Message.ReplyToMessage.Audio != nil {
					pendingMessage = fmt.Sprintf("Type: [Audio] \nFileID: [<code>%v</code>]", opts.update.Message.ReplyToMessage.Audio.FileID)
				} else {
					pendingMessage = "Unknown message type"
				}
			} else {
				pendingMessage = "Reply to a Sticker, Document or Photo to get its FileID"
			}
			_, err := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				Text: pendingMessage,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				log.Printf("Error response /fileid command: %v", err)
			}
		},
	})
	AddSlashSymbolCommandPlugins(Plugin_SlashSymbolCommand{
		slashCommand: "version",
		handler: func(opts *subHandlerOpts) {
			// info, err := opts.thebot.GetWebhookInfo(ctx)
			// fmt.Println(info)
			// return
			botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				Text: outputVersionInfo(),
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				ParseMode: models.ParseModeMarkdownV1,
			})
			time.Sleep(time.Second * 20)
			opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
				ChatID: opts.update.Message.Chat.ID,
				MessageIDs: []int{
					opts.update.Message.ID,
					botMessage.ID,
				},
			})
		},
	})
	AddSlashSymbolCommandPlugins(ForwardOnly_SlashSymbolCommandHandler)

	AddCustomSymbolCommandPlugins(Udonese_SlashCommandHandler...)

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
