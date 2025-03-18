package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"trbot/plugins"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/database_yaml"
	"trbot/utils/handler_utils"
	"trbot/utils/mess"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)


func defaultHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	var opts = handler_utils.SubHandlerOpts{
		Ctx:      ctx,
		Thebot:   thebot,
		Update:   update,
	}

	if update.Message != nil {
		// fmt.Println(getMessageType(update.Message))
		if update.Message.Chat.Type == "private" {
			// plugin_utils.AllPugins.DefaultHandlerByMessageTypeForPrivate
		}
	}

	// 根据 update 类型来设定
	if update.Message != nil {
		// 正常消息
		opts.Fields = strings.Fields(update.Message.Text)
		opts.ChatInfo = database_yaml.GetIDInfo(&update.Message.Chat.ID)

		if opts.ChatInfo == nil && database_yaml.AddChatInfo(&update.Message.Chat) {
			log.Printf("add (message)%s \"%s\"(%s)[%d] in database",
				update.Message.Chat.Type,
				utils.ShowChatName(&update.Message.Chat), update.Message.Chat.Username, update.Message.Chat.ID,
			)
			opts.ChatInfo = database_yaml.GetIDInfo(&update.Message.Chat.ID)
		}

		opts.ChatInfo.LatestMessage = update.Message.Text

		opts.ChatInfo.MessageCount++
		if consts.IsDebugMode {
			if update.Message.Photo != nil {
				log.Printf("photo message from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) caption: [%s]", 
					utils.ShowUserName(update.Message.From), update.Message.From.Username, update.Message.From.ID,
					utils.ShowChatName(&update.Message.Chat), update.Message.Chat.Username, update.Message.Chat.ID,
					update.Message.ID, update.Message.Caption,
				)
			} else if update.Message.Sticker != nil {
				log.Printf("sticker message from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) sticker: %s[%s:%s]", 
					utils.ShowUserName(update.Message.From), update.Message.From.Username, update.Message.From.ID,
					utils.ShowChatName(&update.Message.Chat), update.Message.Chat.Username, update.Message.Chat.ID,
					update.Message.ID, update.Message.Sticker.Emoji, update.Message.Sticker.SetName, update.Message.Sticker.FileID,
				)
			} else {
				log.Printf("message from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) message: [%s]", 
					utils.ShowUserName(update.Message.From), update.Message.From.Username, update.Message.From.ID,
					utils.ShowChatName(&update.Message.Chat), update.Message.Chat.Username, update.Message.Chat.ID,
					update.Message.ID, update.Message.Text,
				)
			}
		}

		messageHandler(&opts)
	} else if update.EditedMessage != nil {
		// 私聊或群组消息被编辑
		if consts.IsDebugMode {
			if update.EditedMessage.Photo != nil {
				log.Printf("edited from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) edited caption to [%s]", 
					utils.ShowUserName(update.EditedMessage.From), update.EditedMessage.From.Username, update.EditedMessage.From.ID,
					utils.ShowChatName(&update.EditedMessage.Chat), update.EditedMessage.Chat.Username, update.EditedMessage.Chat.ID,
					update.EditedMessage.ID, update.EditedMessage.Caption,
				)
			} else {
				log.Printf("edited from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) edited message to [%s]", 
					utils.ShowUserName(update.EditedMessage.From), update.EditedMessage.From.Username, update.EditedMessage.From.ID,
					utils.ShowChatName(&update.EditedMessage.Chat), update.EditedMessage.Chat.Username, update.EditedMessage.Chat.ID,
					update.EditedMessage.ID, update.EditedMessage.Text,
				)
			}
		}
	} else if update.InlineQuery != nil {
		// inline 查询
		opts.Fields = strings.Fields(update.InlineQuery.Query)

		opts.ChatInfo = database_yaml.GetIDInfo(&update.InlineQuery.From.ID)

		if opts.ChatInfo == nil && database_yaml.AddUserInfo(update.InlineQuery.From) {
			log.Printf("add (inline)private \"%s\"(%s)[%d] in database",
				utils.ShowUserName(update.InlineQuery.From), update.InlineQuery.From.Username, update.InlineQuery.From.ID,
			)
			opts.ChatInfo = database_yaml.GetIDInfo(&update.InlineQuery.From.ID)
		}

		opts.ChatInfo.LatestInlineQuery = update.InlineQuery.Query

		opts.ChatInfo.InlineCount++
		log.Printf("inline from: \"%s\"(%s)[%d], query: [%s]", 
			utils.ShowUserName(update.InlineQuery.From), update.InlineQuery.From.Username, update.InlineQuery.From.ID,
			update.InlineQuery.Query,
		)

		inlineHandler(&opts)
	} else if update.ChosenInlineResult != nil {
		// inline 查询结果被选择
		opts.ChatInfo = database_yaml.GetIDInfo(&update.ChosenInlineResult.From.ID)

		if opts.ChatInfo == nil && database_yaml.AddUserInfo(&update.ChosenInlineResult.From) {
			log.Printf("add (inlineResult)private \"%s\"(%s)[%d] in database",
				utils.ShowUserName(&update.ChosenInlineResult.From), update.ChosenInlineResult.From.Username, update.ChosenInlineResult.From.ID,
			)
			opts.ChatInfo = database_yaml.GetIDInfo(&update.ChosenInlineResult.From.ID)
		}

		opts.ChatInfo.LatestInlineResult = update.ChosenInlineResult.ResultID + "," + update.ChosenInlineResult.Query
		log.Printf("chosen inline from \"%s\"(%s)[%d], ID: [%s] query: [%s]",
			utils.ShowUserName(&update.ChosenInlineResult.From), update.ChosenInlineResult.From.Username, update.ChosenInlineResult.From.ID,
			update.ChosenInlineResult.ResultID, update.ChosenInlineResult.Query,
		)
	} else if update.CallbackQuery != nil {
		// replymarkup 回调
		log.Printf("callback from \"%s\"(%s)[%d] in \"%s\"(%s)[%d] query: [%s]",
			utils.ShowUserName(&update.CallbackQuery.From), update.CallbackQuery.From.Username, update.CallbackQuery.From.ID,
			utils.ShowChatName(&update.CallbackQuery.Message.Message.Chat), update.CallbackQuery.Message.Message.Chat.Username, update.CallbackQuery.Message.Message.Chat.ID,
			update.CallbackQuery.Data,
		)

		opts.ChatInfo = database_yaml.GetIDInfo(&update.CallbackQuery.From.ID)

		if opts.ChatInfo == nil && database_yaml.AddUserInfo(&update.CallbackQuery.From) {
			log.Printf("add (callback)private \"%s\"[%d] in database", utils.ShowUserName(&update.CallbackQuery.From), update.CallbackQuery.From.ID)
			opts.ChatInfo = database_yaml.GetIDInfo(&update.CallbackQuery.From.ID)
		}

		// 如果有一个正在处理的请求，且用户再次发送相同的请求，则提示用户等待
		if opts.ChatInfo.HasPendingCallbackQuery && update.CallbackQuery.Data == opts.ChatInfo.LatestCallbackQueryData {
			log.Println("same callback query, ignore")
			thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "当前的请求正在处理，请等待处理完成",
				ShowAlert:       true,
			})
			return
		} else if opts.ChatInfo.HasPendingCallbackQuery {
			// 如果有一个正在处理的请求，用户发送了不同的请求，则提示用户等待
			log.Println("a callback query is pending, ignore")
			thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "请等待上一个请求处理完成再尝试发送新的请求",
				ShowAlert:       true,
			})
			return
		} else {
			// 如果没有正在处理的请求，则接受新的请求
			log.Println("accept callback query")
			opts.ChatInfo.HasPendingCallbackQuery = true
			opts.ChatInfo.LatestCallbackQueryData = update.CallbackQuery.Data
			thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "已接受请求",
				ShowAlert:       false,
			})
		}

		for _, n := range plugin_utils.AllPugins.CallbackQuery {
			if strings.HasPrefix(update.CallbackQuery.Data, n.CommandChar) {
				n.Handler(&opts)
				break
			}
		}

		opts.ChatInfo.HasPendingCallbackQuery = false
		return
	} else if update.MessageReaction != nil {
		// 私聊或群组表情回应
		if consts.IsDebugMode {
			if len(update.MessageReaction.OldReaction) > 0 {
				for i, oldReaction := range update.MessageReaction.OldReaction {
					if oldReaction.ReactionTypeEmoji != nil {
						log.Printf("%d remove emoji reaction %s from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1, oldReaction.ReactionTypeEmoji.Emoji,
							utils.ShowUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							utils.ShowChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					} else if oldReaction.ReactionTypeCustomEmoji != nil {
						log.Printf("%d remove custom emoji reaction %s from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1, oldReaction.ReactionTypeCustomEmoji.CustomEmojiID,
							utils.ShowUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							utils.ShowChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					} else if oldReaction.ReactionTypePaid != nil {
						log.Printf("%d remove paid reaction from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1,
							utils.ShowUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							utils.ShowChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					}
				}
			}
			if len(update.MessageReaction.NewReaction) > 0 {
				for i, newReaction := range update.MessageReaction.NewReaction {
					if newReaction.ReactionTypeEmoji != nil {
						log.Printf("%d emoji reaction %s from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1, newReaction.ReactionTypeEmoji.Emoji,
							utils.ShowUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							utils.ShowChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					} else if newReaction.ReactionTypeCustomEmoji != nil {
						log.Printf("%d custom emoji reaction %s from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1, newReaction.ReactionTypeCustomEmoji.CustomEmojiID,
							utils.ShowUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							utils.ShowChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					} else if newReaction.ReactionTypePaid != nil {
						log.Printf("%d paid reaction from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1,
							utils.ShowUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							utils.ShowChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					}
				}
			}
		}
	} else if update.MessageReactionCount != nil {
		// 频道消息表情回应数量
		log.Printf("reaction count from in \"%s\"(%s)[%d], to message [%d], reactions: %v",
			utils.ShowChatName(&update.MessageReactionCount.Chat), update.MessageReactionCount.Chat.Username, update.MessageReactionCount.Chat.ID,
			update.MessageReactionCount.MessageID, update.MessageReactionCount.Reactions,
		)
	} else if update.ChannelPost != nil {
		// 频道信息
		if consts.IsDebugMode {
			if update.ChannelPost.From != nil { // 在频道中使用户身份发送
				log.Printf("channel post from user \"%s\"(%s)[%d], in \"%s\"(%s)[%d], (%d) message [%s]",
					utils.ShowUserName(update.ChannelPost.From), update.ChannelPost.From.Username, update.ChannelPost.From.ID,
					utils.ShowChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.ID, update.ChannelPost.Text,
				)
			} else if update.ChannelPost.SenderBusinessBot != nil { // 在频道中由商业 bot 发送
				log.Printf("channel post from businessbot \"%s\"(%s)[%d], in \"%s\"(%s)[%d], (%d) message [%s]",
					utils.ShowUserName(update.ChannelPost.SenderBusinessBot), update.ChannelPost.SenderBusinessBot.Username, update.ChannelPost.SenderBusinessBot.ID,
					utils.ShowChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.ID, update.ChannelPost.Text,
				)
			} else if update.ChannelPost.ViaBot != nil { // 在频道中由 bot 发送
				log.Printf("channel post from bot \"%s\"(%s)[%d], in \"%s\"(%s)[%d], (%d) message [%s]",
					utils.ShowUserName(update.ChannelPost.ViaBot), update.ChannelPost.ViaBot.Username, update.ChannelPost.ViaBot.ID,
					utils.ShowChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.ID, update.ChannelPost.Text,
				)
			} else if update.ChannelPost.SenderChat != nil { // 在频道中使用其他频道身份发送
				if update.ChannelPost.SenderChat.ID == update.ChannelPost.Chat.ID { // 在频道中由频道自己发送
					log.Printf("channel post in \"%s\"(%s)[%d], (%d) message [%s]",
						utils.ShowChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
						update.ChannelPost.ID, update.ChannelPost.Text,
					)
				} else {
					log.Printf("channel post from another channel \"%s\"(%s)[%d], in \"%s\"(%s)[%d], (%d) message [%s]",
						utils.ShowChatName(update.ChannelPost.SenderChat), update.ChannelPost.SenderChat.Username, update.ChannelPost.SenderChat.ID,
						utils.ShowChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
						update.ChannelPost.ID, update.ChannelPost.Text,
					)
				}
			} else { // 没有身份信息
				log.Printf("channel post from nobody in \"%s\"(%s)[%d], (%d) message [%s]",
					// utils.ShowUserName(update.ChannelPost.From), update.ChannelPost.From.Username, update.ChannelPost.From.ID,
					utils.ShowChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.ID, update.ChannelPost.Text,
				)
			}
			return
		}
	} else if update.EditedChannelPost != nil {
		// 频道中编辑过的消息
		if consts.IsDebugMode {
			log.Printf("edited channel post in \"%s\"(%s)[%d], message [%s]",
				utils.ShowChatName(&update.EditedChannelPost.Chat), update.EditedChannelPost.Chat.Username, update.EditedChannelPost.Chat.ID,
				update.EditedChannelPost.Text,
			)
		}
	} else {
		// 其他没有加入的更新类型
		if consts.IsDebugMode {
			log.Printf("unknown update type: %v", update)
			// thebot.CopyMessage(ctx, &bot.CopyMessageParams{
			// })
		}
	}
}

// 处理所有信息请求的处理函数，触发条件为任何消息
func messageHandler(opts *handler_utils.SubHandlerOpts) {
	var botMessage *models.Message // 存放 bot 发送的信息

	// 首先判断聊天类型，这里处理私聊、群组和超级群组的消息
	if utils.AnyContains(opts.Update.Message.Chat.Type, models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 检测如果消息开头是 / 符号，作为命令来处理
		if strings.HasPrefix(opts.Update.Message.Text, "/") {
			for _, plugin := range plugin_utils.AllPugins.SlashSymbolCommand {
				if utils.CommandMaybeWithSuffixUsername(opts.Fields, "/start") {
					if consts.IsDebugMode {
						log.Printf("hit startcommand: /%s", plugin.SlashCommand)
					}
					startHandler(opts)
					return
				} else if utils.CommandMaybeWithSuffixUsername(opts.Fields, "/" + plugin.SlashCommand) {
					if consts.IsDebugMode {
						log.Printf("hit slashcommand: /%s", plugin.SlashCommand)
					}
					plugin.Handler(opts)
					return
				}
			}
			// 当使用一个不存在的命令，但是命令末尾指定为此 bot 处理
			if strings.HasSuffix(opts.Fields[0], "@" + consts.BotMe.Username) {
				// 为防止与其他 bot 的命令冲突，默认不会处理不在命令列表中的命令
				// 如果消息以 /xxx@examplebot 的形式指定此 bot 回应，且 /xxx 不在预设的命令中时，才发送该命令不可用的提示
				botMessage, _ = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					Text:      "不存在的命令",
				})
				time.Sleep(time.Second * 10)
				opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
					ChatID:     opts.Update.Message.Chat.ID,
					MessageIDs: []int{
						opts.Update.Message.ID,
						botMessage.ID,
					},
				})
				return
			} 
		} else if len(opts.Update.Message.Text) > 0 {
			for _, plugin := range plugin_utils.AllPugins.CustomSymbolCommand {
				if utils.CommandMaybeWithSuffixUsername(opts.Fields, plugin.FullCommand) {
					if consts.IsDebugMode {
						log.Printf("hit fullcommand: %s", plugin.FullCommand)
					}
					plugin.Handler(opts)
					return
				}
			}
			for _, plugin := range plugin_utils.AllPugins.SuffixCommand {
				if strings.HasSuffix(opts.Update.Message.Text, plugin.SuffixCommand) {
					if consts.IsDebugMode {
						log.Printf("hit suffixcommand: %s", plugin.SuffixCommand)
					}
					plugin.Handler(opts)
					return
				}
			}
		}

		// 不符合上方条件，即消息开头不是 / 符号的信息
		if opts.Update.Message.Chat.Type == models.ChatTypePrivate {
			// 如果用户发送的是贴纸，下载并返回贴纸源文件给用户
			if opts.Update.Message.Sticker != nil {
				plugins.EchoStickerHandler(opts)
				return
			}

			// 不匹配上面项目的则提示不可用
			if strings.HasPrefix(opts.Update.Message.Text, "/") {
				// 非冗余条件，在私聊状态下应处理用户发送的所有开头为 / 的命令
				// 与群组中不同，群组中命令末尾不指定此 bot 回应的命令无须处理，以防与群组中的其他 bot 冲突
				// botMessage, _ = 
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					Text:      "不存在的命令",
				})
				if consts.Private_log { mess.PrivateLogToChat(opts.Ctx, opts.Thebot, opts.Update) }
			} else {
				// 非命令消息，提示无操作可用
				// botMessage, _ = 
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					Text:      "无操作可用",
				})
				if consts.Private_log { mess.PrivateLogToChat(opts.Ctx, opts.Thebot, opts.Update) }

				// opts.Thebot.ForwardMessages(opts.Ctx, &bot.ForwardMessagesParams{
				// 	ChatID:     logChat_ID,
				// 	FromChatID: opts.Update.Message.Chat.ID,
				// 	MessageIDs: []int{
				// 		opts.Update.Message.ID - 1,
				// 		opts.Update.Message.ID,
				// 	},
				// })
			}

			// 等待五秒删除请求信息和回复信息
			// time.Sleep(time.Second * 10)
			// opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
			// 	ChatID:     opts.Update.Message.Chat.ID,
			// 	MessageIDs: []int{
			// 		opts.Update.Message.ID,
			// 		botMessage.ID,
			// 	},
			// })
		} else {
			plugins.DeleteNotAllowMessage(opts)
		}
	}
}

// 处理 inline 模式下的请求
func inlineHandler(opts *handler_utils.SubHandlerOpts) {
	if !strings.HasPrefix(opts.Update.InlineQuery.Query, consts.InlineSubCommandSymbol) {
		if consts.Inline_NoDefaultHandler {
			var inlineButton *models.InlineQueryResultsButton
			// SavedMessage := database_yaml.Database.Data.SavedMessage[opts.ChatInfo.ID]

			// if SavedMessage.Count == 0 && !SavedMessage.AgreePrivacyPolicy {
			// 	inlineButton = &models.InlineQueryResultsButton{
			// 		Text: "点击此处来尝试保存内容",
			// 		StartParameter: "via-inline_savedmessage-help",
			// 	}
			// }

			var commandList []plugin_utils.Plugin_InlineCommandList

			for _, plugin := range plugin_utils.AllPugins.Inline {
				var command plugin_utils.Plugin_InlineCommandList
				command.Command = plugin.Command
				if plugin.Description != "" {
					command.Description = plugin.Description
				} else {
					command.Description = "此插件没有设定描述..."
				}
				commandList = append(commandList, command)
			}

			for _, plugin := range plugin_utils.AllPugins.InlineManual {
				var command plugin_utils.Plugin_InlineCommandList
				command.Command = plugin.Command
				if plugin.Description != "" {
					command.Description = plugin.Description
				} else {
					command.Description = "此插件没有设定描述..."
				}
				commandList = append(commandList, command)
			}

			for _, plugin := range plugin_utils.AllPugins.InlinePrefix {
				var command plugin_utils.Plugin_InlineCommandList
				command.Command = plugin.PrefixCommand
				if plugin.Description != "" {
					command.Description = "仅限管理员 " + plugin.Description
				} else {
					command.Description = "仅限管理员 " + "此插件没有设定描述..."
				}
				commandList = append(commandList, command)
			}

			var message string = "可用的 Inline 模式命令:\n\n"

			for _, command := range commandList {
				message += fmt.Sprintf("命令: <code>%s%s</code>\n描述: %s\n\n", consts.InlineSubCommandSymbol, command.Command, command.Description)
			}

			_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.Update.InlineQuery.ID,
				Results:       []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:    "nodefaulthandler",
					Title: fmt.Sprintf("请继续输入 %s 来查看可用的命令", consts.InlineSubCommandSymbol),
					Description: "由于管理员没有设定默认命令，您需要手动选择一个 Inline 模式下的命令，点击此处查看命令列表",
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: message,
						ParseMode: models.ParseModeHTML,
					},
				}},
				Button: inlineButton,
			})
			if err != nil {
				log.Printf("Error sending inline query response: %v", err)
				return
			}
		} else {
			plugin_utils.AllPugins.InlineManual[0].Handler(opts)
		}
	} else if opts.Update.InlineQuery.Query == consts.InlineSubCommandSymbol {
		// 展示全部命令
	} else if strings.HasPrefix(opts.Update.InlineQuery.Query, consts.InlineSubCommandSymbol) {
		// 插件处理完后返回全部列表，由设定好的函数进行分页
		for _, plugin := range plugin_utils.AllPugins.Inline {
			if opts.Fields[0][1:] == plugin.Command {
				ResultList := plugin.Handler(opts)
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.Update.InlineQuery.ID,
					Results:       utils.InlineResultPagination(opts.Fields, ResultList),
					IsPersonal:    true,
					CacheTime:     30,
				})
				if err != nil {
					log.Printf("Error when answering inline [%s] command: %v", plugin.Command, err)
					// 本来想写一个发生错误后再给用户回答一个错误信息，让用户可以点击发送，结果同一个 ID 的 inlineQuery 只能回答一次
				}
				return
			}
		}
		// 完全由插件控制输出，若回答请求时列表数量超过 50 项会出错，无法回应用户请求
		for _, plugins := range plugin_utils.AllPugins.InlineManual {
			if opts.Fields[0][1:] == plugins.Command {
				plugins.Handler(opts)
				return
			}
		}
		// 仅限管理员使用的命令
		if utils.AnyContains(opts.Update.InlineQuery.From.ID, consts.LogMan_IDs) {
			for _, plugin := range plugin_utils.AllPugins.InlinePrefix {
				if strings.HasPrefix(opts.Update.InlineQuery.Query, consts.InlineSubCommandSymbol + plugin.PrefixCommand) {
					plugin.Handler(opts)
					return
				}
			}
			
		}
		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.Update.InlineQuery.ID,
			Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:    "noinlinecommand",
				Title: fmt.Sprintf("不存在的命令 [%s]", opts.Fields[0]),
				Description: "请检查命令是否正确",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
					ParseMode: models.ParseModeMarkdownV1,
				},
			}},
		})
		if err != nil {
			log.Println("Error when answering inline no command", err)
		}
		return
	} else {
		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.Update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:    "empty",
				Title: "没有保存内容（点击查看详细教程）",
				Description: "对一条信息回复 /save 来保存它",
				InputMessageContent: models.InputTextMessageContent{
					MessageText: fmt.Sprintf("您可以在任何聊天的输入栏中输入 <code>@%s +saved </code>来查看您的收藏\n若要添加，您需要确保机器人可以读取到您的指令，例如在群组中需要添加机器人，或点击 @%s 进入与机器人的聊天窗口，找到想要收藏的信息，然后对着那条信息回复 /save 即可\n若收藏成功，机器人会回复您并提示收藏成功，您也可以手动发送一条想要收藏的息，再使用 /save 命令回复它", consts.BotMe.Username, consts.BotMe.Username),
					ParseMode: models.ParseModeHTML,
				},
			}},
			Button: &models.InlineQueryResultsButton{
				Text: "点击此处快速跳转到机器人",
				StartParameter: "via-inline_noreply",
			},

		})
		if err != nil {
			log.Println("Error when answering inline [saved] command", err)
		}
	}


	

	// fmt.Println(opts.Fields, len(results))

	// _, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
	// 	InlineQueryID: opts.Update.InlineQuery.ID,
	// 	Results:       InlineResultPagination(opts.Fields, results),
	// 	// Button: inlineButton,
	// })
	// if err != nil {
	// 	log.Printf("Error sending inline query response: %v", err)
	// 	return
	// }
}
