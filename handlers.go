package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"trbot/database"
	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/handler_structs"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

func defaultHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	defer utils.PanicCatcher("defaultHandler")
	logger := zerolog.Ctx(ctx).
		With().
		Str("funcName", "defaultHandler").
		Logger()

	// var err error
	var opts = handler_structs.SubHandlerParams{
		Ctx:    ctx,
		Thebot: thebot,
		Update: update,
	}

	database.RecordData(&opts)

	// 需要重写来配合 handler by update type
	if update.Message != nil {
		// 正常消息
		if consts.IsDebugMode {
			if update.Message.Photo != nil {
				logger.Debug().
					Dict(utils.GetUserDict(update.Message.From)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Str("caption", update.Message.Caption).
					Msg("photoMessage")
			} else if update.Message.Sticker != nil {
				logger.Debug().
					Dict(utils.GetUserDict(update.Message.From)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Str("stickerEmoji", update.Message.Sticker.Emoji).
					Str("stickerSetname", update.Message.Sticker.SetName).
					Str("stickerFileID", update.Message.Sticker.FileID).
					Msg("stickerMessage")
			} else {
				logger.Debug().
					Dict(utils.GetUserDict(update.Message.From)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Str("text", update.Message.Text).
					Msg("textMessage")
			}
		}

		messageHandler(&opts)
	} else if update.EditedMessage != nil {
		// 私聊或群组消息被编辑
		if consts.IsDebugMode {
			if update.EditedMessage.Caption != "" {
				logger.Debug().
					Dict(utils.GetUserDict(update.EditedMessage.From)).
					Dict(utils.GetChatDict(&update.EditedMessage.Chat)).
					Int("messageID", update.EditedMessage.ID).
					Str("editedCaption", update.EditedMessage.Caption).
					Msg("editedMessage")
			} else {
				logger.Debug().
					Dict(utils.GetUserDict(update.EditedMessage.From)).
					Dict(utils.GetChatDict(&update.EditedMessage.Chat)).
					Int("messageID", update.EditedMessage.ID).
					Str("editedText", update.EditedMessage.Text).
					Msg("editedMessage")
			}
		}
	} else if update.InlineQuery != nil {
		// inline 查询

		logger.Debug().
			Dict(utils.GetUserDict(update.InlineQuery.From)).
			Str("query", update.InlineQuery.Query).
			Msg("inline request")

		inlineHandler(&opts)
	} else if update.ChosenInlineResult != nil {
		// inline 查询结果被选择
		logger.Debug().
			Dict(utils.GetUserDict(&update.ChosenInlineResult.From)).
			Str("query", update.ChosenInlineResult.Query).
			Str("resultID", update.ChosenInlineResult.ResultID).
			Msg("chosen inline result")

		
	} else if update.CallbackQuery != nil {
		// replymarkup 回调
		logger.Debug().
			Dict(utils.GetUserDict(&update.CallbackQuery.From)).
			Dict(utils.GetChatDict(&update.CallbackQuery.Message.Message.Chat)).
			Str("query", update.CallbackQuery.Data).
			Msg("callback query")

		callbackQueryHandler(&opts)

		

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
func messageHandler(opts *handler_structs.SubHandlerParams) {
	defer utils.PanicCatcher("messageHandler")
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("funcName", "messageHandler").
		Logger()

	// 检测如果消息开头是 / 符号，作为命令来处理
	if strings.HasPrefix(opts.Update.Message.Text, "/") {
		// 匹配默认的 `/xxx` 命令
		for _, plugin := range plugin_utils.AllPlugins.SlashSymbolCommand {
			if utils.CommandMaybeWithSuffixUsername(opts.Fields, "/" + plugin.SlashCommand) {
				logger.Debug().
					Str("slashCommand", plugin.SlashCommand).
					Str("message", opts.Update.Message.Text).
					Msg("Hit slash command handler")
				if plugin.Handler == nil {
					logger.Debug().
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("slashCommand", plugin.SlashCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Hit slash symbol command handler, but this handler function is nil, skip")
					continue
				}
				err := database.IncrementalUsageCount(opts.Ctx, opts.Update.Message.Chat.ID, db_struct.MessageCommand)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("slashCommand", plugin.SlashCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Incremental message command count error")
				}
				err = plugin.Handler(opts)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("slashCommand", plugin.SlashCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Error in slash symbol command handler")
				}
				return
			}
		}
		// 不存在以 `/` 作为前缀的命令
		if opts.Update.Message.Chat.Type == models.ChatTypePrivate {
			// 非冗余条件，在私聊状态下应处理用户发送的所有开头为 / 的命令
			// 与群组中不同，群组中命令末尾不指定此 bot 回应的命令无须处理，以防与群组中的其他 bot 冲突
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:    opts.Update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				Text:      "不存在的命令",
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
					Str("message", opts.Update.Message.Text).
					Msg("Send `no this command` message failed")
			}
			err = database.IncrementalUsageCount(opts.Ctx, opts.Update.Message.Chat.ID, db_struct.MessageCommand)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
					Str("message", opts.Update.Message.Text).
					Msg("Incremental message command count error")
			}
			
			// if configs.BotConfig.LogChatID != 0 { mess.PrivateLogToChat(opts.Ctx, opts.Thebot, opts.Update) }
		} else if strings.HasSuffix(opts.Fields[0], "@" + consts.BotMe.Username) {
			// 当使用一个不存在的命令，但是命令末尾指定为此 bot 处理
			// 为防止与其他 bot 的命令冲突，默认不会处理不在命令列表中的命令
			// 如果消息以 /xxx@examplebot 的形式指定此 bot 回应，且 /xxx 不在预设的命令中时，才发送该命令不可用的提示
			botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:    opts.Update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				Text:      "不存在的命令",
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
					Str("message", opts.Update.Message.Text).
					Msg("Send `no this command` message failed")
			}
			err = database.IncrementalUsageCount(opts.Ctx, opts.Update.Message.Chat.ID, db_struct.MessageCommand)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
					Str("message", opts.Update.Message.Text).
					Msg("Incremental message command count error")
			}
			time.Sleep(time.Second * 10)
			_, err = opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
				ChatID:     opts.Update.Message.Chat.ID,
				MessageIDs: []int{
					opts.Update.Message.ID,
					botMessage.ID,
				},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
					Str("message", opts.Update.Message.Text).
					Msg("Delete `no this command` message failed")
			}
			return
		}
	} else if len(opts.Update.Message.Text) > 0 {
		// 没有 `/` 号作为前缀，检查是不是自定义命令
		for _, plugin := range plugin_utils.AllPlugins.CustomSymbolCommand {
			if utils.CommandMaybeWithSuffixUsername(opts.Fields, plugin.FullCommand) {
				logger.Debug().
					Str("fullCommand", plugin.FullCommand).
					Str("message", opts.Update.Message.Text).
					Msg("Hit full command handler")
				if plugin.Handler == nil {
					logger.Debug().
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("fullCommand", plugin.FullCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Hit full command handler, but this handler function is nil, skip")
					continue
				}
				err := database.IncrementalUsageCount(opts.Ctx, opts.Update.Message.Chat.ID, db_struct.MessageCommand)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("fullCommand", plugin.FullCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Incremental message command count error")
				}
				err = plugin.Handler(opts)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("fullCommand", plugin.FullCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Error in full command handler")
				}
				return
			}
		}
		// 以后缀来触发的命令
		for _, plugin := range plugin_utils.AllPlugins.SuffixCommand {
			if strings.HasSuffix(opts.Update.Message.Text, plugin.SuffixCommand) {
				logger.Debug().
					Str("suffixCommand", plugin.SuffixCommand).
					Str("message", opts.Update.Message.Text).
					Msg("Hit suffix command handler")
				if plugin.Handler == nil {
					logger.Debug().
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("suffixCommand", plugin.SuffixCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Hit suffix command handler, but this handler function is nil, skip")
					continue
				}
				err := database.IncrementalUsageCount(opts.Ctx, opts.Update.Message.Chat.ID, db_struct.MessageCommand)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("suffixCommand", plugin.SuffixCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Incremental message command count error")
				}
				err = plugin.Handler(opts)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("suffixCommand", plugin.SuffixCommand).
						Str("message", opts.Update.Message.Text).
						Msg("Error in suffix command handler")
				}
				return
			}
		}
	}

	// 按消息类型来触发的 handler
	// handler by message type
	if plugin_utils.AllPlugins.HandlerByMessageType[opts.Update.Message.Chat.Type] != nil {
		msgTypeInString := message_utils.GetMessageType(opts.Update.Message).InString()

		if plugin_utils.AllPlugins.HandlerByMessageType[opts.Update.Message.Chat.Type][msgTypeInString] != nil {
			handlerInThisTypeCount := len(plugin_utils.AllPlugins.HandlerByMessageType[opts.Update.Message.Chat.Type][msgTypeInString])
			if handlerInThisTypeCount == 1 {
				// 虽然是遍历，但实际上只能遍历一次
				for name, handler := range plugin_utils.AllPlugins.HandlerByMessageType[opts.Update.Message.Chat.Type][msgTypeInString] {
					if handler.AllowAutoTrigger {
						// 允许自动触发的 handler
						logger.Debug().
							Dict(utils.GetUserDict(opts.Update.Message.From)).
							Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
							Str("messageType", string(msgTypeInString)).
							Str("handlerName", name).
							Str("chatType", string(opts.Update.Message.Chat.Type)).
							Msg("trigger handler by message type")
						err := handler.Handler(opts)
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(opts.Update.Message.From)).
								Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
								Str("messageType", string(msgTypeInString)).
								Str("handlerName", name).
								Str("chatType", string(opts.Update.Message.Chat.Type)).
								Msg("Error in handler by message type")
						}
					} else {
						// 此 handler 不允许自动触发，回复一条带按钮的消息让用户手动操作
						_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID:    opts.Update.Message.Chat.ID,
							Text:      fmt.Sprintf("请选择一个 [ %s ] 类型消息的功能", msgTypeInString),
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
							ReplyMarkup: plugin_utils.AllPlugins.HandlerByMessageType[opts.Update.Message.Chat.Type][msgTypeInString].BuildSelectKeyboard(),
						})
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(opts.Update.Message.From)).
								Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
								Str("messageType", string(msgTypeInString)).
								Str("chatType", string(opts.Update.Message.Chat.Type)).
								Int("handlerInThisTypeCount", handlerInThisTypeCount).
								Msg("Send `select a handler by message type keyboard` message failed")
						}
					}
				}
			} else {
				// 多个 handler 自动回复一条带按钮的消息让用户手动操作
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					Text:      fmt.Sprintf("请选择一个 [ %s ] 类型消息的功能", msgTypeInString),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					ReplyMarkup: plugin_utils.AllPlugins.HandlerByMessageType[opts.Update.Message.Chat.Type][msgTypeInString].BuildSelectKeyboard(),
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
						Str("messageType", string(msgTypeInString)).
						Str("chatType", string(opts.Update.Message.Chat.Type)).
						Int("handlerInThisTypeCount", handlerInThisTypeCount).
						Msg("Send `select a handler by message type keyboard` message failed")
				}
			}
		} else if opts.Update.Message.Chat.Type == models.ChatTypePrivate {
			// 仅在 private 对话中显示无默认处理插件的消息
			// 如果没有设定任何对于 private 对话按消息来触发的 handler，则代码不会运行到这里
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:    opts.Update.Message.Chat.ID,
				Text:      fmt.Sprintf("对于 [ %s ] 类型的消息没有默认处理插件", msgTypeInString),
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
					Str("messageType", string(msgTypeInString)).
					Str("chatType", string(opts.Update.Message.Chat.Type)).
					Msg("Send `no handler by message type plugin for this message type` message failed")
			}
		}
	}

	// 最后才运行针对群组 ID 的 handler
	// handler by chat ID
	if plugin_utils.AllPlugins.HandlerByChatID[opts.Update.Message.Chat.ID] != nil {
		for name, handler := range plugin_utils.AllPlugins.HandlerByChatID[opts.Update.Message.Chat.ID] {
			logger.Debug().
				Dict(utils.GetUserDict(opts.Update.Message.From)).
				Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
				Str("handlerName", name).
				Int64("chatID", handler.ChatID).
				Str("chatType", string(opts.Update.Message.Chat.Type)).
				Msg("trigger handler by chat ID")
			err := handler.Handler(opts)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
					Str("handlerName", name).
					Int64("chatID", handler.ChatID).
					Str("chatType", string(opts.Update.Message.Chat.Type)).
					Msg("Error in handler by chat ID")
			}
		}
	}
}

// 处理 inline 模式下的请求
func inlineHandler(opts *handler_structs.SubHandlerParams) {
	defer utils.PanicCatcher("inlineHandler")
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("funcName", "inlineHandler").
		Logger()

	var IsAdmin bool = utils.AnyContains(opts.Update.InlineQuery.From.ID, configs.BotConfig.AdminIDs)

	if opts.Update.InlineQuery.Query == configs.BotConfig.InlineSubCommandSymbol {
		// 仅输入了命令符号，展示命令列表
		var inlineButton = &models.InlineQueryResultsButton{
			Text: "点击此处修改默认命令",
			StartParameter: "via-inline_change-inline-command",
		}
		// 展示全部命令
		var results []models.InlineQueryResult
		results = append(results, &models.InlineQueryResultArticle{
			ID:    "keepInput",
			Title: "请不要点击列表中的命令",
			Description: "由于限制，您需要手动输入完整的命令",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "请不要点击选单中的命令...",
			},
		})
		for _, plugin := range plugin_utils.AllPlugins.InlineCommandList {
			if !IsAdmin && plugin.Attr.IsHideInCommandList {
				continue
			}
			var command = &models.InlineQueryResultArticle{
				ID:    "inlinemenu" + plugin.Command,
				Title: plugin.Command,
				Description: plugin.Description,
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "请不要点击选单中的命令...",
				},
			}
			if plugin.Attr.IsHideInCommandList {
				command.Description = "隐藏 | " + command.Description
			}
			if plugin.Attr.IsOnlyAllowAdmin {
				command.Description = "管理员 | " + command.Description
			}
			results = append(results, command)
		}
		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.Update.InlineQuery.ID,
			Results:       results,
			Button:        inlineButton,
			IsPersonal:    true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
				Msg("Send /setkeyword command answer failed")
			log.Printf("Error sending inline query response: %v", err)
			return
		}
	} else if strings.HasPrefix(opts.Update.InlineQuery.Query, configs.BotConfig.InlineSubCommandSymbol) {
		// 用户输入了分页符号和一些字符，判断接着的命令是否正确，正确则交给对应的插件处理，否则提示错误

		// 插件处理完后返回全部列表，由设定好的函数进行分页输出
		for _, plugin := range plugin_utils.AllPlugins.InlineHandler {
			if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
				continue
			}
			if opts.Fields[0][1:] == plugin.Command {
				if plugin.Handler == nil { continue }
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
		for _, plugin := range plugin_utils.AllPlugins.InlineManualHandler {
			if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
				continue
			}
			if opts.Fields[0][1:] == plugin.Command {
				if plugin.Handler == nil { continue }
				plugin.Handler(opts)
				return
			}
		}
		// 符合命令前缀，完全由插件自行控制输出
		for _, plugin := range plugin_utils.AllPlugins.InlinePrefixHandler {
			if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
				continue
			}
			if strings.HasPrefix(opts.Update.InlineQuery.Query, configs.BotConfig.InlineSubCommandSymbol + plugin.PrefixCommand) {
				if plugin.Handler == nil { continue }
				plugin.Handler(opts)
				return
			}
		}

		// 没有匹配到任何命令
		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.Update.InlineQuery.ID,
			Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:    "noinlinecommand",
				Title: fmt.Sprintf("不存在的命令 [%s]", opts.Fields[0]),
				Description: "请检查命令是否正确",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "您在使用 inline 模式时没有选择正确的命令...",
					ParseMode: models.ParseModeMarkdownV1,
				},
			}},
		})
		if err != nil {
			log.Println("Error when answering inline no command", err)
		}
		return
	} else {
		if opts.ChatInfo.DefaultInlinePlugin != "" {
			// 来自用户设定的默认命令

			// 插件处理完后返回全部列表，由设定好的函数进行分页输出
			for _, plugin := range plugin_utils.AllPlugins.InlineHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
					continue
				}
				if opts.ChatInfo.DefaultInlinePlugin == plugin.Command {
					if plugin.Handler == nil { continue }
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
			for _, plugin := range plugin_utils.AllPlugins.InlineManualHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
					continue
				}
				if opts.ChatInfo.DefaultInlinePlugin == plugin.Command {
					if plugin.Handler == nil { continue }
					plugin.Handler(opts)
					return
				}
			}

			// 符合命令前缀，完全由插件自行控制输出
			for _, plugin := range plugin_utils.AllPlugins.InlinePrefixHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
					continue
				}
				if opts.ChatInfo.DefaultInlinePlugin == plugin.PrefixCommand {
					if plugin.Handler == nil { continue }
					plugin.Handler(opts)
					return
				}
			}

			// 没有匹配到命令，提示不存在的命令
			_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.Update.InlineQuery.ID,
				Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:    "noinlineplugin",
					Title: fmt.Sprintf("不存在的默认命令 [%s]", opts.ChatInfo.DefaultInlinePlugin),
					Description: "或许是因为管理员已经移除了这个插件，请重新选择一个默认命令",
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
						ParseMode: models.ParseModeMarkdownV1,
					},
				}},
				Button: &models.InlineQueryResultsButton{
					Text: "点击此处修改默认命令",
					StartParameter: "via-inline_change-inline-command",
				},
			})
			if err != nil {
				log.Println("Error when answering inline default command invailid:", err)
			}
			return
		} else if configs.BotConfig.InlineDefaultHandler != "" {
			// 全局设定里设定的默认命令

			// 插件处理完后返回全部列表，由设定好的函数进行分页输出
			for _, plugin := range plugin_utils.AllPlugins.InlineHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
					continue
				}
				if configs.BotConfig.InlineDefaultHandler == plugin.Command {
					if plugin.Handler == nil { continue }
					ResultList := plugin.Handler(opts)
					_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.Update.InlineQuery.ID,
						Results:       utils.InlineResultPagination(opts.Fields, ResultList),
						IsPersonal:    true,
						CacheTime:     30,
						Button: &models.InlineQueryResultsButton{
							Text: "输入 + 号显示菜单，或点击此处修改默认命令",
							StartParameter: "via-inline_change-inline-command",
						},
					})
					if err != nil {
						log.Printf("Error when answering inline [%s] command: %v", plugin.Command, err)
						// 本来想写一个发生错误后再给用户回答一个错误信息，让用户可以点击发送，结果同一个 ID 的 inlineQuery 只能回答一次
					}
					return
				}
			}
			// 完全由插件控制输出，若回答请求时列表数量超过 50 项会出错，无法回应用户请求
			for _, plugin := range plugin_utils.AllPlugins.InlineManualHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
					continue
				}
				if configs.BotConfig.InlineDefaultHandler == plugin.Command {
					if plugin.Handler == nil { continue }
					plugin.Handler(opts)
					return
				}
			}
			// 符合命令前缀，完全由插件自行控制输出
			for _, plugin := range plugin_utils.AllPlugins.InlinePrefixHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin {
					continue
				}
				if opts.ChatInfo.DefaultInlinePlugin == plugin.PrefixCommand {
					if plugin.Handler == nil { continue }
					plugin.Handler(opts)
					return
				}
			}

			// 判断是否有足够的插件，以及默认插件是否存在
			var pendingMessage string
			if len(plugin_utils.AllPlugins.InlineCommandList) == 0 {
				pendingMessage = "此 bot 似乎并没有使用任何 inline 模式插件，请联系管理员"
			} else {
				pendingMessage = fmt.Sprintf("您可以继续输入 %s 号来查看其他可用的命令", configs.BotConfig.InlineSubCommandSymbol)
			}
			 _, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.Update.InlineQuery.ID,
				Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:    "invaliddefaulthandler",
					Title: "管理员设定了无效的默认命令",
					Description: pendingMessage,
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "机器人管理员设定了一个无效的默认 inline 命令",
						ParseMode: models.ParseModeMarkdownV1,
					},
				}},
				Button: &models.InlineQueryResultsButton{
					Text: "您可以点击此处设定一个默认命令",
					StartParameter: "via-inline_change-inline-command",
				},
			})
			if err != nil {
				log.Printf("Error sending inline query response: %v", err)
				return
			}
			return
		}

		// 用户没设定默认命令，配置里也没有填写默认命令 consts.InlineDefaultHandler，
		var inlineButton = &models.InlineQueryResultsButton{
			Text: "您可以点击此处设定一个默认命令",
			StartParameter: "via-inline_change-inline-command",
		}
		var message string = "可用的 Inline 模式命令:\n\n"
		for _, command := range plugin_utils.AllPlugins.InlineCommandList {
			if command.Attr.IsHideInCommandList {
				continue
			}
			message += fmt.Sprintf("命令: <code>%s%s</code>\n", configs.BotConfig.InlineSubCommandSymbol, command.Command)
			if command.Description != "" {
				message += fmt.Sprintf("描述: %s\n", command.Description)
			}
			message += "\n"
		}

		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.Update.InlineQuery.ID,
			Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:    "nodefaulthandler",
				Title: fmt.Sprintf("请继续输入 %s 来查看可用的命令", configs.BotConfig.InlineSubCommandSymbol),
				Description: "由于管理员没有设定默认命令，您需要手动选择一个命令，点击此处查看命令列表",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: message,
					ParseMode: models.ParseModeHTML,
				},
			}},
			Button: inlineButton,
		})
		if err != nil {
			log.Printf("Error sending inline query no default handler: %v", err)
			return
		}
	}
}

func callbackQueryHandler(params *handler_structs.SubHandlerParams) {
	defer utils.PanicCatcher("callbackQueryHandler")
		logger := zerolog.Ctx(params.Ctx).
			With().
			Str("funcName", "callbackQueryHandler").
			Logger()

	// 如果有一个正在处理的请求，且用户再次发送相同的请求，则提示用户等待
		if params.ChatInfo.HasPendingCallbackQuery && params.Update.CallbackQuery.Data == params.ChatInfo.LatestCallbackQueryData {
			logger.Info().
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Str("query", params.Update.CallbackQuery.Data).
				Msg("this callback request is processing, ignore")
			_, err := params.Thebot.AnswerCallbackQuery(params.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: params.Update.CallbackQuery.ID,
				Text:            "当前请求正在处理中，请等待处理完成",
				ShowAlert:       true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
					Msg("Send `this callback request is processing` callback answer failed")
			}
			return
		} else if params.ChatInfo.HasPendingCallbackQuery {
			// 如果有一个正在处理的请求，用户发送了不同的请求，则提示用户等待
			logger.Info().
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Str("pendingQuery", params.ChatInfo.LatestCallbackQueryData).
				Str("query", params.Update.CallbackQuery.Data).
				Msg("another callback request is processing, ignore")
			_, err := params.Thebot.AnswerCallbackQuery(params.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: params.Update.CallbackQuery.ID,
				Text:            "请等待上一个请求处理完成后再尝试发送新的请求",
				ShowAlert:       true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
					Msg("Send `a callback request is processing, send new request later` callback answer failed")
			}
			return
		} else {
			// 如果没有正在处理的请求，则接受新的请求
			logger.Debug().
				Dict(utils.GetUserDict(&params.Update.CallbackQuery.From)).
				Str("query", params.Update.CallbackQuery.Data).
				Msg("accept callback query")

			params.ChatInfo.HasPendingCallbackQuery = true
			params.ChatInfo.LatestCallbackQueryData = params.Update.CallbackQuery.Data
			// params.Thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			// 	CallbackQueryID: params.Update.CallbackQuery.ID,
			// 	Text:            "已接受请求",
			// 	ShowAlert:       false,
			// })
		}

		for _, n := range plugin_utils.AllPlugins.CallbackQuery {
			if strings.HasPrefix(params.Update.CallbackQuery.Data, n.CommandChar) {
				if n.Handler == nil {
					logger.Debug().
						Dict(utils.GetUserDict(params.Update.Message.From)).
						Str("handlerPrefix", n.CommandChar).
						Str("query", params.Update.CallbackQuery.Data).
						Msg("tigger a callback query handler, but this handler function is nil, skip")
					continue
				}
				err := n.Handler(params)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(params.Update.Message.From)).
						Str("handlerPrefix", n.CommandChar).
						Str("query", params.Update.CallbackQuery.Data).
						Msg("Error in callback query handler")
				}
				break
			}
		}
}
