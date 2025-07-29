package main

import (
	"context"
	"fmt"
	"strings"

	"trbot/database"
	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/inline_utils"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"trbot/utils/type/message_utils"
	"trbot/utils/type/update_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

func defaultHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	defer utils.PanicCatcher(ctx, "defaultHandler")
	logger := zerolog.Ctx(ctx)

	var opts = handler_params.Update{
		Ctx:    ctx,
		Thebot: thebot,
		Update: update,
		// ChatInfo will fill in `database.RecordData()` function
	}

	// 判断更新类型
	updateType := update_utils.GetUpdateType(update)

	// 记录数据和读取信息
	database.RecordData(&opts, updateType)

	// Debug or Trace Level
	// 消息日志，因为比较占用资源，先判断日志等级
	if zerolog.GlobalLevel() <= zerolog.InfoLevel {
		switch {
		case updateType.Message:
			// 正常消息
			if update.Message.Photo != nil {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.Message)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Str("caption", update.Message.Caption).
					Str("photoID", update.Message.Photo[len(update.Message.Photo)-1].FileID).
					Msg("photoMessage")
			} else if update.Message.Sticker != nil {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.Message)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Dict("sticker", zerolog.Dict().
						Str("emoji", update.Message.Sticker.Emoji).
						Str("setname", update.Message.Sticker.SetName).
						Str("fileID", update.Message.Sticker.FileID),
					).
					Msg("stickerMessage")
			} else if update.Message.Video != nil {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.Message)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Str("caption", update.Message.Caption).
					Dict("video", zerolog.Dict().
						Str("type", update.Message.Video.MimeType).
						Int("duration", update.Message.Video.Duration).
						Str("fileID", update.Message.Video.FileID),
					).
					Msg("videoMessage")
			} else if update.Message.Animation != nil {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.Message)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Dict("animation", zerolog.Dict().
						Int("duration", update.Message.Animation.Duration).
						Str("fileID", update.Message.Animation.FileID),
					).
					Msg("gifMessage")
			} else if update.Message.Document != nil {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.Message)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Str("caption", update.Message.Caption).
					Dict("document", zerolog.Dict().
						Str("fileName", update.Message.Document.FileName).
						Str("fileID", update.Message.Document.FileID),
					).
					Msg("documentMessage")
			} else if update.Message.PinnedMessage != nil {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.Message)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Msg("pinMessage")
			} else if update.Message.NewChatMembers != nil {
				for _, chatMember := range update.Message.NewChatMembers {
					logger.Info().
						Dict(utils.GetUserDict(&chatMember)).
						Dict(utils.GetChatDict(&update.Message.Chat)).
						Int("messageID", update.Message.ID).
						Msg("newChatMemberMessage")
				}
			} else if update.Message.LeftChatMember != nil {
				logger.Info().
					Dict(utils.GetUserDict(update.Message.LeftChatMember)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Msg("leftChatMemberMessage")
			} else {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.Message)).
					Dict(utils.GetChatDict(&update.Message.Chat)).
					Int("messageID", update.Message.ID).
					Str("text", update.Message.Text).
					Str("type", message_utils.GetMessageType(update.Message).Str()).
					Msg("normalMessage")
			}
		case updateType.EditedMessage:
			// 私聊或群组消息被编辑
			if update.EditedMessage.Caption != "" {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.EditedMessage)).
					Dict(utils.GetChatDict(&update.EditedMessage.Chat)).
					Int("messageID", update.EditedMessage.ID).
					Str("editedCaption", update.EditedMessage.Caption).
					Msg("editedMessage")
			} else {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.EditedMessage)).
					Dict(utils.GetChatDict(&update.EditedMessage.Chat)).
					Int("messageID", update.EditedMessage.ID).
					Str("editedText", update.EditedMessage.Text).
					Msg("editedMessage")
			}
		case updateType.InlineQuery:
			// inline 查询
			logger.Info().
				Dict(utils.GetUserDict(update.InlineQuery.From)).
				Str("query", update.InlineQuery.Query).
				Msg("inline request")
		case updateType.ChosenInlineResult:
			// inline 查询结果被选择
			logger.Info().
				Dict(utils.GetUserDict(&update.ChosenInlineResult.From)).
				Str("query", update.ChosenInlineResult.Query).
				Str("resultID", update.ChosenInlineResult.ResultID).
				Msg("chosen inline result")
		case updateType.CallbackQuery:
			// replymarkup 回调
			var chat = zerolog.Dict()
			if update.CallbackQuery.Message.Message != nil {
				// some time `update.CallbackQuery.Message` will be nil
				_, chat = utils.GetChatDict(&update.CallbackQuery.Message.Message.Chat)
			}
			logger.Info().
				Dict(utils.GetUserDict(&update.CallbackQuery.From)).
				Dict("chat", chat).
				Str("callbackQueryData", update.CallbackQuery.Data).
				Msg("callback query")
		case updateType.MessageReaction:
			// 私聊或群组表情回应
			if len(update.MessageReaction.OldReaction) > 0 {
				for i, oldReaction := range update.MessageReaction.OldReaction {
					if oldReaction.ReactionTypeEmoji != nil {
						logger.Info().
							Dict(utils.GetUserDict(update.MessageReaction.User)).
							Dict(utils.GetChatDict(&update.MessageReaction.Chat)).
							Int("messageID", update.MessageReaction.MessageID).
							Str("removedEmoji", oldReaction.ReactionTypeEmoji.Emoji).
							Str("emojiType", string(oldReaction.ReactionTypeEmoji.Type)).
							Int("count", i + 1).
							Msg("removed emoji reaction")
					} else if oldReaction.ReactionTypeCustomEmoji != nil {
						logger.Info().
							Dict(utils.GetUserDict(update.MessageReaction.User)).
							Dict(utils.GetChatDict(&update.MessageReaction.Chat)).
							Int("messageID", update.MessageReaction.MessageID).
							Str("removedEmojiID", oldReaction.ReactionTypeCustomEmoji.CustomEmojiID).
							Str("emojiType", string(oldReaction.ReactionTypeCustomEmoji.Type)).
							Int("count", i + 1).
							Msg("removed custom emoji reaction")
					} else if oldReaction.ReactionTypePaid != nil {
						logger.Info().
							Dict(utils.GetUserDict(update.MessageReaction.User)).
							Dict(utils.GetChatDict(&update.MessageReaction.Chat)).
							Int("messageID", update.MessageReaction.MessageID).
							Str("emojiType", string(oldReaction.ReactionTypePaid.Type)).
							Int("count", i + 1).
							Msg("removed paid emoji reaction")
					}
				}
			}
			if len(update.MessageReaction.NewReaction) > 0 {
				for i, newReaction := range update.MessageReaction.NewReaction {
					if newReaction.ReactionTypeEmoji != nil {
						logger.Info().
							Dict(utils.GetUserDict(update.MessageReaction.User)).
							Dict(utils.GetChatDict(&update.MessageReaction.Chat)).
							Int("messageID", update.MessageReaction.MessageID).
							Str("addEmoji", newReaction.ReactionTypeEmoji.Emoji).
							Str("emojiType", string(newReaction.ReactionTypeEmoji.Type)).
							Int("count", i + 1).
							Msg("add emoji reaction")
					} else if newReaction.ReactionTypeCustomEmoji != nil {
						logger.Info().
							Dict(utils.GetUserDict(update.MessageReaction.User)).
							Dict(utils.GetChatDict(&update.MessageReaction.Chat)).
							Int("messageID", update.MessageReaction.MessageID).
							Str("addEmojiID", newReaction.ReactionTypeCustomEmoji.CustomEmojiID).
							Str("emojiType", string(newReaction.ReactionTypeCustomEmoji.Type)).
							Int("count", i + 1).
							Msg("add custom emoji reaction")
					} else if newReaction.ReactionTypePaid != nil {
						logger.Info().
							Dict(utils.GetUserDict(update.MessageReaction.User)).
							Dict(utils.GetChatDict(&update.MessageReaction.Chat)).
							Int("messageID", update.MessageReaction.MessageID).
							Str("emojiType", string(newReaction.ReactionTypePaid.Type)).
							Int("count", i + 1).
							Msg("add paid emoji reaction")
					}
				}
			}
		case updateType.MessageReactionCount:
			// 频道消息表情回应数量
			var emoji        = zerolog.Dict()
			var customEmoji  = zerolog.Dict()
			var paid         = zerolog.Dict()
			for _, n := range update.MessageReactionCount.Reactions {
				switch n.Type.Type {
				case models.ReactionTypeTypeEmoji:
					emoji.Dict(n.Type.ReactionTypeEmoji.Emoji, zerolog.Dict().
						// Str("type", string(n.Type.ReactionTypeEmoji.Type)).
						// Str("emoji", n.Type.ReactionTypeEmoji.Emoji).
						Int("count", n.TotalCount),
					)
				case models.ReactionTypeTypeCustomEmoji:
					customEmoji.Dict(n.Type.ReactionTypeCustomEmoji.CustomEmojiID, zerolog.Dict().
						// Str("type", string(n.Type.ReactionTypeCustomEmoji.Type)).
						// Str("customEmojiID", n.Type.ReactionTypeCustomEmoji.CustomEmojiID).
						Int("count", n.TotalCount),
					)
				case models.ReactionTypeTypePaid:
					paid.Dict(n.Type.ReactionTypePaid.Type, zerolog.Dict().
						// Str("type", n.Type.ReactionTypePaid.Type).
						Int("count", n.TotalCount),
					)
				}

			}

			logger.Info().
				Dict(utils.GetChatDict(&update.MessageReactionCount.Chat)).
				Dict("reactions", zerolog.Dict().
					Dict("emoji", emoji).
					Dict("customEmoji", customEmoji).
					Dict("paid", paid),
				).
				Int("messageID", update.MessageReactionCount.MessageID).
				Msg("emoji reaction count updated")
		case updateType.ChannelPost:
			// 频道信息
			logger.Info().
				Dict(utils.GetUserOrSenderChatDict(update.ChannelPost)).
				Dict(utils.GetChatDict(&update.ChannelPost.Chat)).
				Str("text", update.ChannelPost.Text).
				Int("messageID", update.ChannelPost.ID).
				Msg("channel post")
			if update.ChannelPost.ViaBot != nil {
				// 在频道中由 bot 发送
				_, viaBot := utils.GetUserDict(update.ChannelPost.ViaBot)
				logger.Info().
					Dict("viaBot", viaBot).
					Dict(utils.GetChatDict(&update.ChannelPost.Chat)).
					Str("text", update.ChannelPost.Text).
					Int("messageID", update.ChannelPost.ID).
					Msg("channel post send via bot")
			}
		case updateType.EditedChannelPost:
			// 频道中编辑过的消息
			if update.EditedChannelPost.Caption != "" {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.EditedChannelPost)).
					Dict(utils.GetChatDict(&update.EditedChannelPost.Chat)).
					Int("messageID", update.EditedChannelPost.ID).
					Str("editedCaption", update.EditedChannelPost.Caption).
					Msg("edited channel post caption")
			} else {
				logger.Info().
					Dict(utils.GetUserOrSenderChatDict(update.EditedChannelPost)).
					Dict(utils.GetChatDict(&update.EditedChannelPost.Chat)).
					Int("messageID", update.EditedChannelPost.ID).
					Str("editedText", update.EditedChannelPost.Text).
					Msg("edited channel post")
			}
		case updateType.ChatMember:
			logger.Info().
				Dict(utils.GetUserDict(&update.ChatMember.From)).
				Dict(utils.GetChatDict(&update.ChatMember.Chat)).
				Str("oldType", string(update.ChatMember.OldChatMember.Type)).
				Str("newType", string(update.ChatMember.NewChatMember.Type)).
				Str("notice", "the user field may not be the actual user whose status was changed").
				Msg("chatMemberUpdated")
		default:
			// 其他没有加入的更新类型
			logger.Warn().
				Str("updateType", updateType.Str()).
				Msg("Receive a no tagged update type")
		}
	}

	// 根据更新类型调用相应的处理函数
	switch {
	case updateType.Message:
		messageHandler(&handler_params.Message{
			Ctx:      opts.Ctx,
			Thebot:   opts.Thebot,
			Message:  opts.Update.Message,
			ChatInfo: opts.ChatInfo,
			Fields:   strings.Fields(opts.Update.Message.Text),
		})
	case updateType.ChannelPost:
		messageHandler(&handler_params.Message{
			Ctx:      opts.Ctx,
			Thebot:   opts.Thebot,
			Message:  opts.Update.ChannelPost,
			ChatInfo: opts.ChatInfo,
			Fields:   strings.Fields(opts.Update.ChannelPost.Text),
		})
	case updateType.InlineQuery:
		inlineHandler(&handler_params.InlineQuery{
		Ctx:         opts.Ctx,
		Thebot:      opts.Thebot,
		InlineQuery: opts.Update.InlineQuery,
		ChatInfo:    opts.ChatInfo,
		Fields:      strings.Fields(opts.Update.InlineQuery.Query),
	})
	case updateType.CallbackQuery:
		callbackQueryHandler(&handler_params.CallbackQuery{
		Ctx:           opts.Ctx,
		Thebot:        opts.Thebot,
		CallbackQuery: opts.Update.CallbackQuery,
		ChatInfo:      opts.ChatInfo,
	})
	}
}

// 处理所有信息请求的处理函数，触发条件为任何消息
func messageHandler(opts *handler_params.Message) {
	defer utils.PanicCatcher(opts.Ctx, "messageHandler")

	messageLogger := zerolog.Ctx(opts.Ctx).
		With().
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Str("text",    opts.Message.Text).
		Str("caption", opts.Message.Caption).
		Logger()

	if plugin_utils.RunStateHandler(opts) {
		// 如果状态处理函数返回 true，则表示已经处理了该消息，直接返回
		return
	}

	// 判断是否为命令
	isProcessed, err := plugin_utils.RunCommandHandlers(opts)
	if isProcessed {
		if err != nil {
			messageLogger.
				Err(err).
				Msg("Error when running command handler")
		}
		return
	}

	// 按消息类型来触发的 handler
	// handler by message type
	isProcessed, msgType, err := plugin_utils.RunByMessageTypeHandlers(opts)
	if !isProcessed && opts.Message.Chat.Type == models.ChatTypePrivate {
		// 仅在 private 对话中显示无默认处理插件的消息
		// 如果没有设定任何对于 private 对话按消息来触发的 handler，则代码不会运行到这里
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Message.Chat.ID,
			Text:      fmt.Sprintf("对于 [ %s ] 类型的消息没有默认处理插件", msgType),
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		})
		if err != nil {
			messageLogger.Error().
				Err(err).
				Str("messageType", msgType).
				Str("content", "no handler by message type plugin for this private chat").
				Msg(flaterr.SendMessage.Str())
		}
	}
	if err != nil {
		messageLogger.Error().
			Err(err).
			Bool("isProcessed", isProcessed).
			Msg("Error when running by message type handler")
	}

	// 最后才运行针对群组 ID 的 handler
	// handler by chat ID
	count, err := plugin_utils.RunByChatIDHandlers(opts)
	if err != nil {
		messageLogger.Error().
			Err(err).
			Int("handlerRunCount", count).
			Msg("Error when running by chat ID handlers")
	}
}

// 处理 inline 模式下的请求
func inlineHandler(opts *handler_params.InlineQuery) {
	defer utils.PanicCatcher(opts.Ctx, "inlineHandler")
	inlineLogger := zerolog.Ctx(opts.Ctx).
		With().
		Dict(utils.GetUserDict(opts.InlineQuery.From)).
		Str("query", opts.InlineQuery.Query).
		Logger()

	IsAdmin     := contain.Int64(opts.InlineQuery.From.ID, configs.BotConfig.AdminIDs...)
	parsedQuery := inline_utils.ParseInlineFields(opts.Fields)

	if strings.HasPrefix(opts.InlineQuery.Query, configs.BotConfig.InlineSubCommandSymbol) {
		// 用户输入了分页符号和一些字符，判断接着的命令是否正确，正确则交给对应的插件处理，否则显示命令菜单

		// 插件处理完后返回全部列表，由设定好的函数进行分页输出
		for _, plugin := range plugin_utils.AllPlugins.InlineHandler {
			if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
			if opts.Fields[0][1:] == plugin.Command {
				slogger := inlineLogger.With().
					Str("handlerCommand", plugin.Command).
					Str("handlerType", "returnResult").
					Logger()

				if plugin.InlineHandler != nil {
					slogger.Info().Msg("Hit inline handler")
					ResultList := plugin.InlineHandler(opts)
					_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.InlineQuery.ID,
						Results:       inline_utils.ResultPagination(parsedQuery, ResultList),
						IsPersonal:    true,
						CacheTime:     0,
					})
					if err != nil {
						slogger.Error().
							Err(err).
							Str("content", "sub inline handler").
							Msg(flaterr.AnswerInlineQuery.Str())
						// 本来想写一个发生错误后再给用户回答一个错误信息，让用户可以点击发送，结果同一个 ID 的 inlineQuery 只能回答一次
					}
					return
				} else {
					slogger.Warn().Msg("Hit inline handler, but this handler function is nil, skip")
				}
			}
		}
		// 完全由插件控制输出，若回答请求时列表数量超过 50 项会出错，无法回应用户请求
		for _, plugin := range plugin_utils.AllPlugins.InlineManualHandler {
			if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
			if opts.Fields[0][1:] == plugin.Command {
				slogger := inlineLogger.With().
					Str("handlerCommand", plugin.Command).
					Str("handlerType", "manuallyAnswerResult").
					Logger()

				if plugin.InlineHandler != nil {
					slogger.Info().Msg("Hit inline manual answer handler")
					err := plugin.InlineHandler(opts)
					if err != nil {
						slogger.Error().
							Err(err).
							Msg("Error in inline manual answer handler")
					}
					return
				} else {
					slogger.Warn().Msg("Hit inline manual answer handler, but this handler function is nil, skip")
				}
			}
		}
		// 符合命令前缀，完全由插件自行控制输出
		for _, plugin := range plugin_utils.AllPlugins.InlinePrefixHandler {
			if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
			if strings.HasPrefix(opts.InlineQuery.Query, configs.BotConfig.InlineSubCommandSymbol + plugin.PrefixCommand) {
				slogger := inlineLogger.With().
					Str("handlerPrefixCommand", plugin.PrefixCommand).
					Str("handlerType", "manuallyAnswerResult_PrefixCommand").
					Logger()

				if plugin.InlineHandler != nil {
					slogger.Info().Msg("Hit inline prefix manual answer handler")
					err := plugin.InlineHandler(opts)
					if err != nil {
						slogger.Error().
							Err(err).
							Msg("Error in inline prefix manual answer handler")
					}
					return
				} else {
					slogger.Warn().Msg("Hit inline prefix manual answer handler, but this handler function is nil, skip")
				}
			}
		}

		// 没有触发任何 handler
		inlineLogger.Debug().Msg("No any handler is hit")

		// 创建变量存放提示和命令菜单
		var results []models.InlineQueryResult = []models.InlineQueryResult{ &models.InlineQueryResultArticle{
			ID:                  "keepInput",
			Title:               "请不要点击列表中的命令",
			Description:         "由于限制，您需要手动输入完整的命令",
			InputMessageContent: &models.InputTextMessageContent{ MessageText: "请不要点击选单中的命令..." },
		}}

		// 添加匹配输入的命令列表
		for _, plugin := range plugin_utils.AllPlugins.InlineCommandList {
			if !IsAdmin && plugin.Attr.IsHideInCommandList { continue }
			if strings.HasPrefix(plugin.Command, parsedQuery.SubCommand) {
				var description string
				if plugin.Attr.IsOnlyAllowAdmin    { description += "管理员 | " }
				if plugin.Attr.IsHideInCommandList { description += "隐藏 | "   }

				results = append(results, &models.InlineQueryResultArticle{
					ID:          "inlineMenu_" + plugin.Command,
					Title:       plugin.Command,
					Description: description + plugin.Description,
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "请不要点击选单中的命令...",
					},
				})
			}
		}

		// 没有匹配的命令
		if len(results) == 1 {
			results = []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:                  "noinlinecommand",
				Title:               fmt.Sprintf("不存在的命令 [%s]", opts.Fields[0]),
				Description:         "请检查命令是否正确",
				InputMessageContent: &models.InputTextMessageContent{ MessageText: "您在使用 inline 模式时没有输入正确的命令..." },
			}}
		}

		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.InlineQuery.ID,
			Results:       results,
			IsPersonal:    true,
			CacheTime:     0,
			Button:        &models.InlineQueryResultsButton{
				Text:           "点击此处修改默认命令",
				StartParameter: "via-inline_change-inline-command",
			},
		})
		if err != nil {
			inlineLogger.Error().
				Err(err).
				Str("content", "bot inline handler list").
				Msg(flaterr.AnswerInlineQuery.Str())
		}
	} else {
		// inline query 不以命令符号开头，检查是否有默认 handler
		if opts.ChatInfo.DefaultInlinePlugin != "" {
			// 来自用户设定的默认命令
			defaultHandlerLogger := inlineLogger.With().
				Str("userDefaultHandlerCommand", opts.ChatInfo.DefaultInlinePlugin).
				Logger()

			// 插件处理完后返回全部列表，由设定好的函数进行分页输出
			for _, plugin := range plugin_utils.AllPlugins.InlineHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
				if opts.ChatInfo.DefaultInlinePlugin == plugin.Command {
					slogger := defaultHandlerLogger.With().
						Str("handlerType", "returnResult").
						Logger()

					if plugin.InlineHandler != nil {
						slogger.Info().Msg("Hit user default inline handler")
						resultList := plugin.InlineHandler(opts)
						_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
							InlineQueryID: opts.InlineQuery.ID,
							Results:       inline_utils.ResultPagination(parsedQuery, resultList),
							IsPersonal:    true,
							CacheTime:     0,
						})
						if err != nil {
							slogger.Error().
								Err(err).
								Str("content", "user default inline handler result").
								Msg(flaterr.AnswerInlineQuery.Str())
							// 本来想写一个发生错误后再给用户回答一个错误信息，让用户可以点击发送，结果同一个 ID 的 inlineQuery 只能回答一次
						}
						return
					} else {
						slogger.Warn().Msg("Hit user default inline handler, but this handler function is nil, skip")
					}
				}
			}
			// 完全由插件控制输出，若回答请求时列表数量超过 50 项会出错，无法回应用户请求
			for _, plugin := range plugin_utils.AllPlugins.InlineManualHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
				if opts.ChatInfo.DefaultInlinePlugin == plugin.Command {
					slogger := defaultHandlerLogger.With().
						Str("handlerType", "manuallyAnswerResult").
						Logger()

					if plugin.InlineHandler != nil {
						slogger.Info().Msg("Hit user default inline manual answer handler")
						err := plugin.InlineHandler(opts)
						if err != nil {
							slogger.Error().
								Err(err).
								Msg("Error in user default inline manual answer handler")
						}
						return
					} else {
						slogger.Warn().Msg("Hit user default inline manual answer handler, but this handler function is nil, skip")
					}
				}
			}
			// 符合命令前缀，完全由插件自行控制输出
			for _, plugin := range plugin_utils.AllPlugins.InlinePrefixHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
				if opts.ChatInfo.DefaultInlinePlugin == plugin.PrefixCommand {
					slogger := defaultHandlerLogger.With().
						Str("handlerType", "manuallyAnswerResult_PrefixCommand").
						Logger()

					if plugin.InlineHandler != nil {
						slogger.Info().Msg("Hit user default inline prefix manual answer handler")
						err := plugin.InlineHandler(opts)
						if err != nil {
							slogger.Error().
								Err(err).
								Msg("Error in user inline prefix manual answer handler")
						}
						return
					} else {
						slogger.Warn().Msg("Hit user inline prefix manual answer handler, but this handler function is nil, skip")
					}
				}
			}

			// 没有匹配到命令，提示不存在的命令
			_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.InlineQuery.ID,
				Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:                  "noInlinePlugin",
					Title:               fmt.Sprintf("不存在的默认命令 [%s]", opts.ChatInfo.DefaultInlinePlugin),
					Description:         "或许是因为管理员已经移除了这个插件，请重新选择一个默认命令",
					InputMessageContent: &models.InputTextMessageContent{ MessageText: "此默认命令无效，或许是因为管理员已经移除了这个插件，请重新选择一个默认命令" },
				}},
				Button: &models.InlineQueryResultsButton{
					Text:           "点击此处修改默认命令",
					StartParameter: "via-inline_change-inline-command",
				},
			})
			if err != nil {
				defaultHandlerLogger.Error().
					Err(err).
					Str("content", "invalid user default inline handler").
					Msg(flaterr.AnswerInlineQuery.Str())
			}
			return
		} else if configs.BotConfig.InlineDefaultHandler != "" {
			// 全局设定里设定的默认命令
			defaultHandlerLogger := inlineLogger.With().
				Str("botDefaultHandlerCommand", configs.BotConfig.InlineDefaultHandler).
				Logger()

			// 插件处理完后返回全部列表，由设定好的函数进行分页输出
			for _, plugin := range plugin_utils.AllPlugins.InlineHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
				if configs.BotConfig.InlineDefaultHandler == plugin.Command {
					slogger := defaultHandlerLogger.With().
						Str("handlerType", "returnResult").
						Logger()

					if plugin.InlineHandler != nil {
						slogger.Info().Msg("Hit bot default inline handler")
						resultList := plugin.InlineHandler(opts)
						_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
							InlineQueryID: opts.InlineQuery.ID,
							Results:       inline_utils.ResultPagination(parsedQuery, resultList),
							IsPersonal:    true,
							CacheTime:     0,
							Button: &models.InlineQueryResultsButton{
								Text:           fmt.Sprintf("输入 %s 号显示菜单，或点击此处修改默认命令", configs.BotConfig.InlineSubCommandSymbol),
								StartParameter: "via-inline_change-inline-command",
							},
						})
						if err != nil {
							slogger.Error().
								Err(err).
								Str("content", "bot default inline handler result").
								Msg(flaterr.AnswerInlineQuery.Str())
							// 本来想写一个发生错误后再给用户回答一个错误信息，让用户可以点击发送，结果同一个 ID 的 inlineQuery 只能回答一次
						}
						return
					} else {
						slogger.Warn().Msg("Hit bot default inline handler, but this handler function is nil, skip")
					}
				}
			}
			// 完全由插件控制输出，若回答请求时列表数量超过 50 项会出错，无法回应用户请求
			for _, plugin := range plugin_utils.AllPlugins.InlineManualHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
				if configs.BotConfig.InlineDefaultHandler == plugin.Command {
					slogger := defaultHandlerLogger.With().
						Str("handlerType", "manuallyAnswerResult").
						Logger()

					if plugin.InlineHandler != nil {
						slogger.Info().Msg("Hit bot default inline manual answer handler")
						err := plugin.InlineHandler(opts)
						if err != nil {
							slogger.Error().
								Err(err).
								Msg("Error in bot default inline manual answer handler")
						}
						return
					} else {
						slogger.Warn().Msg("Hit bot default inline manual answer handler, but this handler function is nil, skip")
					}
				}
			}
			// 符合命令前缀，完全由插件自行控制输出
			for _, plugin := range plugin_utils.AllPlugins.InlinePrefixHandler {
				if plugin.Attr.IsOnlyAllowAdmin && !IsAdmin { continue }
				if configs.BotConfig.InlineDefaultHandler == plugin.PrefixCommand {
					slogger := defaultHandlerLogger.With().
						Str("handlerType", "manuallyAnswerResult_PrefixCommand").
						Logger()

					if plugin.InlineHandler != nil {
						slogger.Info().Msg("Hit bot default inline prefix manual answer handler")
						err := plugin.InlineHandler(opts)
						if err != nil {
							slogger.Error().
								Err(err).
								Msg("Error in bot default inline prefix manual answer handler")
						}
						return
					} else {
						slogger.Warn().Msg("Hit bot default inline prefix manual answer handler, but this handler function is nil, skip")
					}
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
				InlineQueryID: opts.InlineQuery.ID,
				Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:    "invalidDefaultHandler",
					Title: "管理员设定了无效的默认命令",
					Description: pendingMessage,
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "机器人管理员设定了一个无效的默认 inline 命令",
					},
				}},
				Button: &models.InlineQueryResultsButton{
					Text:           "您可以点击此处设定一个默认命令",
					StartParameter: "via-inline_change-inline-command",
				},
			})
			if err != nil {
				defaultHandlerLogger.Error().
					Err(err).
					Str("content", "invalid bot default inline handler").
					Msg(flaterr.AnswerInlineQuery.Str())
			}
			return
		}

		// 用户没设定默认命令，配置里也没有填写默认命令 consts.InlineDefaultHandler，提示如何打开命令菜单
		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.InlineQuery.ID,
			Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:    "nodefaulthandler",
				Title: fmt.Sprintf("请继续输入 %s 来查看可用的命令", configs.BotConfig.InlineSubCommandSymbol),
				Description: "由于管理员没有设定默认命令，您需要手动选择一个命令，点击此处查看命令列表",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: func() string {
						var message string = "可用的 Inline 模式命令:\n\n"
						for _, command := range plugin_utils.AllPlugins.InlineCommandList {
							if command.Attr.IsHideInCommandList { continue }
							message += fmt.Sprintf("命令: <code>%s%s</code>\n", configs.BotConfig.InlineSubCommandSymbol, command.Command)
							if command.Description != "" {
								message += fmt.Sprintf("描述: %s\n", command.Description)
							}
							message += "\n"
						}
						return message
					}(),
					ParseMode:   models.ParseModeHTML,
				},
			}},
			Button: &models.InlineQueryResultsButton{
				Text:           "您可以点击此处设定一个默认命令",
				StartParameter: "via-inline_change-inline-command",
			},
		})
		if err != nil {
			inlineLogger.Error().
				Err(err).
				Str("content", "bot no default inline handler").
				Msg(flaterr.AnswerInlineQuery.Str())
		}
	}
}

func callbackQueryHandler(params *handler_params.CallbackQuery) {
	defer utils.PanicCatcher(params.Ctx, "callbackQueryHandler")
	var isProcessing bool
	defer func() {
		if isProcessing { database.UpdateOperationStatus(params.Ctx, params.ChatInfo.ID, db_struct.HasPendingCallbackQuery, false) }
	}()

	callbackQueryLogger := zerolog.Ctx(params.Ctx).
		With().
		Dict(utils.GetUserDict(&params.CallbackQuery.From)).
		Str("callbackQueryData", params.CallbackQuery.Data).
		Logger()

	// 如果有一个正在处理的请求，且用户再次发送相同的请求，则提示用户等待
	if params.ChatInfo.HasPendingCallbackQuery && params.CallbackQuery.Data == params.ChatInfo.LatestCallbackQueryData {
		callbackQueryLogger.Info().Msg("this callback request is processing, ignore")

		_, err := params.Thebot.AnswerCallbackQuery(params.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: params.CallbackQuery.ID,
			Text:            "当前请求正在处理中，请等待处理完成",
			ShowAlert:       true,
		})
		if err != nil {
			callbackQueryLogger.Error().
				Err(err).
				Str("content", "this callback request is processing").
				Msg(flaterr.AnswerCallbackQuery.Str())
		}
		return
	} else if params.ChatInfo.HasPendingCallbackQuery {
		// 如果有一个正在处理的请求，用户发送了不同的请求，则提示用户等待
		callbackQueryLogger.Info().
			Str("pendingQueryData", params.ChatInfo.LatestCallbackQueryData).
			Msg("another callback request is processing, ignore")
		_, err := params.Thebot.AnswerCallbackQuery(params.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: params.CallbackQuery.ID,
			Text:            "请等待上一个请求处理完成后再尝试发送新的请求",
			ShowAlert:       true,
		})
		if err != nil {
			callbackQueryLogger.Error().
				Err(err).
				Str("content", "a callback request is processing, send new request later").
				Msg(flaterr.AnswerCallbackQuery.Str())
		}
		return
	} else {
		// 如果没有正在处理的请求，则接受新的请求
		callbackQueryLogger.Debug().Msg("accept callback query")

		isProcessing = true
		database.UpdateOperationStatus(params.Ctx, params.ChatInfo.ID, db_struct.HasPendingCallbackQuery, true)
		params.ChatInfo.LatestCallbackQueryData = params.CallbackQuery.Data
		// params.Thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		// 	CallbackQueryID: params.CallbackQuery.ID,
		// 	Text:            "已接受请求",
		// 	ShowAlert:       false,
		// })
	}

	for _, plugin := range plugin_utils.AllPlugins.CallbackQuery {
		if strings.HasPrefix(params.CallbackQuery.Data, plugin.CallbackDataPrefix) {
			slogger := callbackQueryLogger.With().
				Str("handlerPrefix", plugin.CallbackDataPrefix).
				Logger()

			if plugin.CallbackQueryHandler != nil {
				slogger.Info().Msg("Hit callback query handler")
				err := plugin.CallbackQueryHandler(params)
				if err != nil {
					callbackQueryLogger.Error().
						Err(err).
						Msg("Error in callback query handler")
				}
				return
			} else {
				slogger.Warn().Msg("Hit callback query handler, but this handler all function is nil, skip")
			}
		}
	}
}
