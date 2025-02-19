package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

// 调用子处理函数时的传递的参数，避免重复获取
type subHandlerOpts struct {
	ctx      context.Context
	thebot   *bot.Bot
	update   *models.Update
	chatInfo *IDInfo
	fields   []string // 根据请求的类型，可能是消息文本，也可能是 inline 的 query
}

func defaultHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	
	var opts = subHandlerOpts{
		ctx:      ctx,
		thebot:   thebot,
		update:   update,
	}

	// 根据 update 类型来设定
	if update.Message != nil {
		// 正常消息
		opts.fields = strings.Fields(update.Message.Text)
		opts.chatInfo = getIDInfo(&update.Message.Chat.ID)

		if opts.chatInfo == nil && AddChatInfo(&update.Message.Chat) {
			log.Printf("add (message)%s \"%s\"(%s)[%d] in database",
				update.Message.Chat.Type,
				showChatName(&update.Message.Chat), update.Message.Chat.Username, update.Message.Chat.ID,
			)
			opts.chatInfo = getIDInfo(&update.Message.Chat.ID)
		}

		opts.chatInfo.LatestMessage = update.Message.Text

		opts.chatInfo.MessageCount++
		if IsDebugMode {
			if update.Message.Photo != nil {
				log.Printf("photo message from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) caption: [%s]", 
					showUserName(update.Message.From), update.Message.From.Username, update.Message.From.ID,
					showChatName(&update.Message.Chat), update.Message.Chat.Username, update.Message.Chat.ID,
					update.Message.ID, update.Message.Caption,
				)
			} else if update.Message.Sticker != nil {
				log.Printf("sticker message from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) sticker: %s[%s:%s]", 
					showUserName(update.Message.From), update.Message.From.Username, update.Message.From.ID,
					showChatName(&update.Message.Chat), update.Message.Chat.Username, update.Message.Chat.ID,
					update.Message.ID, update.Message.Sticker.Emoji, update.Message.Sticker.SetName, update.Message.Sticker.FileID,
				)
			} else {
				log.Printf("message from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) message: [%s]", 
					showUserName(update.Message.From), update.Message.From.Username, update.Message.From.ID,
					showChatName(&update.Message.Chat), update.Message.Chat.Username, update.Message.Chat.ID,
					update.Message.ID, update.Message.Text,
				)
			}
		}

		messageHandler(&opts)
	} else if update.EditedMessage != nil {
		// 私聊或群组消息被编辑
		if IsDebugMode {
			if update.EditedMessage.Photo != nil {
				log.Printf("edited from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) edited caption to [%s]", 
					showUserName(update.EditedMessage.From), update.EditedMessage.From.Username, update.EditedMessage.From.ID,
					showChatName(&update.EditedMessage.Chat), update.EditedMessage.Chat.Username, update.EditedMessage.Chat.ID,
					update.EditedMessage.ID, update.EditedMessage.Caption,
				)
			} else {
				log.Printf("edited from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], (%d) edited message to [%s]", 
					showUserName(update.EditedMessage.From), update.EditedMessage.From.Username, update.EditedMessage.From.ID,
					showChatName(&update.EditedMessage.Chat), update.EditedMessage.Chat.Username, update.EditedMessage.Chat.ID,
					update.EditedMessage.ID, update.EditedMessage.Text,
				)
			}
		}
	} else if update.InlineQuery != nil {
		// inline 查询
		opts.fields = strings.Fields(update.InlineQuery.Query)

		// 由于 inline 请求实际上不区分大小写，全部转为小写，方便后续查询

		opts.chatInfo = getIDInfo(&update.InlineQuery.From.ID)

		if opts.chatInfo == nil && AddUserInfo(update.InlineQuery.From) {
			log.Printf("add (inline)private \"%s\"(%s)[%d] in database",
				showUserName(update.InlineQuery.From), update.InlineQuery.From.Username, update.InlineQuery.From.ID,
			)
			opts.chatInfo = getIDInfo(&update.InlineQuery.From.ID)
		}

		opts.chatInfo.LatestInlineQuery = update.InlineQuery.Query

		opts.chatInfo.InlineCount++
		log.Printf("inline from: \"%s\"(%s)[%d], query: [%s]", 
			showUserName(update.InlineQuery.From), update.InlineQuery.From.Username, update.InlineQuery.From.ID,
			update.InlineQuery.Query,
		)

		inlineHandler(&opts)
	} else if update.ChosenInlineResult != nil {
		// inline 查询结果被选择
		opts.chatInfo = getIDInfo(&update.ChosenInlineResult.From.ID)

		if opts.chatInfo == nil && AddUserInfo(&update.ChosenInlineResult.From) {
			log.Printf("add (inlineResult)private \"%s\"(%s)[%d] in database",
				showUserName(&update.ChosenInlineResult.From), update.ChosenInlineResult.From.Username, update.ChosenInlineResult.From.ID,
			)
			opts.chatInfo = getIDInfo(&update.ChosenInlineResult.From.ID)
		}

		opts.chatInfo.LatestInlineResult = update.ChosenInlineResult.ResultID + "," + update.ChosenInlineResult.Query
		log.Printf("chosen inline from \"%s\"(%s)[%d], ID: [%s] query: [%s]",
			showUserName(&update.ChosenInlineResult.From), update.ChosenInlineResult.From.Username, update.ChosenInlineResult.From.ID,
			update.ChosenInlineResult.ResultID, update.ChosenInlineResult.Query,
		)
	} else if update.CallbackQuery != nil {
		// replymarkup 回调
		log.Printf("callback from \"%s\"(%s)[%d] in \"%s\"(%s)[%d] query: [%s]",
			showUserName(&update.CallbackQuery.From), update.CallbackQuery.From.Username, update.CallbackQuery.From.ID,
			showChatName(&update.CallbackQuery.Message.Message.Chat), update.CallbackQuery.Message.Message.Chat.Username, update.CallbackQuery.Message.Message.Chat.ID,
			update.CallbackQuery.Data,
		)

		opts.chatInfo = getIDInfo(&update.CallbackQuery.From.ID)

		if opts.chatInfo == nil && AddUserInfo(&update.CallbackQuery.From) {
			log.Printf("add (callback)private \"%s\"[%d] in database", showUserName(&update.CallbackQuery.From), update.CallbackQuery.From.ID)
			opts.chatInfo = getIDInfo(&update.CallbackQuery.From.ID)
		}

		// 如果有一个正在处理的请求，且用户再次发送相同的请求，则提示用户等待
		if opts.chatInfo.HasPendingCallbackQuery && update.CallbackQuery.Data == opts.chatInfo.LatestCallbackQueryData {
			log.Println("same callback query, ignore")
			thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "当前的请求正在处理，请等待处理完成",
				ShowAlert:       true,
			})
			return
		} else if opts.chatInfo.HasPendingCallbackQuery { // 如果有一个正在处理的请求，用户发送了不同的请求，则提示用户等待
			log.Println("a callback query is pending, ignore")
			thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "请等待上一个请求处理完成再发送新的请求",
				ShowAlert:       true,
			})
			return
		} else { // 如果没有正在处理的请求，则接受新的请求
			log.Println("accept callback query")
			opts.chatInfo.HasPendingCallbackQuery = true
			opts.chatInfo.LatestCallbackQueryData = update.CallbackQuery.Data
			thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "已接受请求",
				ShowAlert:       false,
			})
		}

		if strings.HasPrefix(update.CallbackQuery.Data, "S_") || strings.HasPrefix(update.CallbackQuery.Data, "s_") {
			botMessage, _ := thebot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.CallbackQuery.Message.Message.Chat.ID,
				Text:   "已请求下载，请稍候",
				ParseMode: models.ParseModeMarkdownV1,
			})
			var packName string
			var isOnlyPNG bool
			if update.CallbackQuery.Data[0:2] == "S_" {
				packName = strings.TrimPrefix(update.CallbackQuery.Data, "S_")
				isOnlyPNG = true
			} else {
				packName = strings.TrimPrefix(update.CallbackQuery.Data, "s_")
				isOnlyPNG = false
			}

			// 通过贴纸的 packName 获取贴纸集
			stickerSet, err := opts.thebot.GetStickerSet(opts.ctx, &bot.GetStickerSetParams{ Name: packName })
			if err != nil {
				log.Printf("error getting sticker set: %v", err)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.CallbackQuery.From.ID,
					Text:   "获取贴纸包时发生了一些错误",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			}

			sitckerSetPack, count, err := getStickerPack(&opts, stickerSet, isOnlyPNG)
			if err != nil {
				log.Println("Error downloading sticker:", err)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.CallbackQuery.From.ID,
					Text:   "下载贴纸包时发生了一些错误",
					ParseMode: models.ParseModeMarkdownV1,
				})
			}
			if sitckerSetPack == nil {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.CallbackQuery.From.ID,
					Text:   "未能获取到压缩包",
					ParseMode: models.ParseModeMarkdownV1,
				})
				opts.chatInfo.HasPendingCallbackQuery = false
				return
			}

			documentParams := &bot.SendDocumentParams{
				ChatID: opts.update.CallbackQuery.From.ID,
				ParseMode: models.ParseModeMarkdownV1,
			}

			if isOnlyPNG {
				documentParams.Caption  = fmt.Sprintf("[%s](https://t.me/addstickers/%s) 已下载\n包含 %d 个贴纸（仅转换后的 PNG 格式）", stickerSet.Title, stickerSet.Name, count)
				documentParams.Document = &models.InputFileUpload{Filename: fmt.Sprintf("%s(%d)_png.zip", stickerSet.Name, count), Data: sitckerSetPack}
			} else {
				documentParams.Caption  = fmt.Sprintf("[%s](https://t.me/addstickers/%s) 已下载\n包含 %d 个贴纸", stickerSet.Title, stickerSet.Name, count)
				documentParams.Document = &models.InputFileUpload{Filename: fmt.Sprintf("%s(%d).zip", stickerSet.Name, count), Data: sitckerSetPack}
			}

			thebot.SendDocument(opts.ctx, documentParams)

			thebot.DeleteMessage(ctx, &bot.DeleteMessageParams{
				ChatID: opts.update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: botMessage.ID,
			})

			opts.chatInfo.HasPendingCallbackQuery = false
		}
	} else if update.MessageReaction != nil {
		// 私聊或群组表情回应
		if IsDebugMode {
			if len(update.MessageReaction.OldReaction) > 0 {
				for i, oldReaction := range update.MessageReaction.OldReaction {
					if oldReaction.ReactionTypeEmoji != nil {
						log.Printf("%d remove emoji reaction %s from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1, oldReaction.ReactionTypeEmoji.Emoji,
							showUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							showChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					} else if oldReaction.ReactionTypeCustomEmoji != nil {
						log.Printf("%d remove custom emoji reaction %s from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1, oldReaction.ReactionTypeCustomEmoji.CustomEmojiID,
							showUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							showChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					} else if oldReaction.ReactionTypePaid != nil {
						log.Printf("%d remove paid reaction from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1,
							showUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							showChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
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
							showUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							showChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					} else if newReaction.ReactionTypeCustomEmoji != nil {
						log.Printf("%d custom emoji reaction %s from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1, newReaction.ReactionTypeCustomEmoji.CustomEmojiID,
							showUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							showChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					} else if newReaction.ReactionTypePaid != nil {
						log.Printf("%d paid reaction from \"%s\"(%s)[%d] in \"%s\"(%s)[%d], to message [%d]",
							i + 1,
							showUserName(update.MessageReaction.User), update.MessageReaction.User.Username, update.MessageReaction.User.ID,
							showChatName(&update.MessageReaction.Chat), update.MessageReaction.Chat.Username, update.MessageReaction.Chat.ID,
							update.MessageReaction.MessageID,
						)
					}
				}
			}
		}
	} else if update.MessageReactionCount != nil {
		// 频道消息表情回应数量
		log.Printf("reaction count from in \"%s\"(%s)[%d], to message [%d], reactions: %v",
			showChatName(&update.MessageReactionCount.Chat), update.MessageReactionCount.Chat.Username, update.MessageReactionCount.Chat.ID,
			update.MessageReactionCount.MessageID, update.MessageReactionCount.Reactions,
		)
	} else if update.ChannelPost != nil {
		// 频道信息
		if IsDebugMode {
			if update.ChannelPost.From != nil { // 在频道中使用户身份发送
				log.Printf("channel post from user \"%s\"(%s)[%d], in \"%s\"(%s)[%d], (%d) message [%s]",
					showUserName(update.ChannelPost.From), update.ChannelPost.From.Username, update.ChannelPost.From.ID,
					showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.ID, update.ChannelPost.Text,
				)
			} else if update.ChannelPost.SenderBusinessBot != nil { // 在频道中由商业 bot 发送
				log.Printf("channel post from businessbot \"%s\"(%s)[%d], in \"%s\"(%s)[%d], (%d) message [%s]",
					showUserName(update.ChannelPost.SenderBusinessBot), update.ChannelPost.SenderBusinessBot.Username, update.ChannelPost.SenderBusinessBot.ID,
					showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.ID, update.ChannelPost.Text,
				)
			} else if update.ChannelPost.ViaBot != nil { // 在频道中由 bot 发送
				log.Printf("channel post from bot \"%s\"(%s)[%d], in \"%s\"(%s)[%d], (%d) message [%s]",
					showUserName(update.ChannelPost.ViaBot), update.ChannelPost.ViaBot.Username, update.ChannelPost.ViaBot.ID,
					showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.ID, update.ChannelPost.Text,
				)
			} else if update.ChannelPost.SenderChat != nil { // 在频道中使用其他频道身份发送
				if update.ChannelPost.SenderChat.ID == update.ChannelPost.Chat.ID { // 在频道中由频道自己发送
					log.Printf("channel post in \"%s\"(%s)[%d], (%d) message [%s]",
						showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
						update.ChannelPost.ID, update.ChannelPost.Text,
					)
				} else {
					log.Printf("channel post from another channel \"%s\"(%s)[%d], in \"%s\"(%s)[%d], (%d) message [%s]",
						showChatName(update.ChannelPost.SenderChat), update.ChannelPost.SenderChat.Username, update.ChannelPost.SenderChat.ID,
						showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
						update.ChannelPost.ID, update.ChannelPost.Text,
					)
				}
			} else { // 没有身份信息
				log.Printf("channel post from nobody in \"%s\"(%s)[%d], (%d) message [%s]",
					// showUserName(update.ChannelPost.From), update.ChannelPost.From.Username, update.ChannelPost.From.ID,
					showChatName(&update.ChannelPost.Chat), update.ChannelPost.Chat.Username, update.ChannelPost.Chat.ID,
					update.ChannelPost.ID, update.ChannelPost.Text,
				)
			}
			return
		}
	} else if update.EditedChannelPost != nil {
		// 频道中编辑过的消息
		if IsDebugMode {
			log.Printf("edited channel post in \"%s\"(%s)[%d], message [%s]",
				showChatName(&update.EditedChannelPost.Chat), update.EditedChannelPost.Chat.Username, update.EditedChannelPost.Chat.ID,
				update.EditedChannelPost.Text,
			)
		}
	} else {
		// 其他没有加入的更新类型
		if IsDebugMode {
			log.Printf("unknown update type: %v", update)
			// thebot.CopyMessage(ctx, &bot.CopyMessageParams{
			// })
		}
	}
}

// 处理所有信息请求的处理函数，触发条件为任何消息
func messageHandler(opts *subHandlerOpts) {
	var botMessage *models.Message // 存放 bot 发送的信息

	// 首先判断聊天类型，这里处理私聊、群组和超级群组的消息
	if AnyContains(opts.update.Message.Chat.Type, models.ChatTypePrivate, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 检测如果消息开头是 / 符号，作为命令来处理
		if strings.HasPrefix(opts.update.Message.Text, "/") {
			// 预设的多个命令
			if commandMaybeWithSuffixUsername(opts.fields, "/start") {
				startHandler(opts)
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/forwardonly") {
				forwardOnlyModeHandler(opts)
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/chatinfo") {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					Text: fmt.Sprintf("类型: [<code>%v</code>]\nID: [<code>%v</code>]\n用户名:[<code>%v</code>]", opts.update.Message.Chat.Type, opts.update.Message.Chat.ID, opts.update.Message.Chat.Username),
					ParseMode: models.ParseModeHTML,
				})
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/test") {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text: "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{ { {
						Text: "点击加入测试群组",
						URL: "https://t.me/+BomkHuFsjqc3ZGE1",
					}}}},
				})
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/fileid") {
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
			} else if commandMaybeWithSuffixUsername(opts.fields, "/save") {
				saveMessageHandler(opts)
				return
			} else if commandMaybeWithSuffixUsername(opts.fields, "/version") && AnyContains(opts.update.Message.From.ID, logMan_IDs) {
				// info, err := opts.thebot.GetWebhookInfo(ctx)
				// fmt.Println(info)
				// return
				botMessage, _ = opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
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
				return
			} else if strings.HasSuffix(opts.fields[0], "@" + botMe.Username) {
				// 注意，此段应该保持在此 if-else 语句的末尾，否则后续的命令将无法触发
				// 为防止与其他 bot 的命令冲突，默认不会处理不在命令列表中的命令
				// 如果消息以 /xxx@examplebot 的形式指定此 bot 回应，且 /xxx 不在预设的命令中时，才发送该命令不可用的提示
				botMessage, _ = opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					Text:      "不存在的命令",
				})
				time.Sleep(time.Second * 10)
				opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
					ChatID:     opts.update.Message.Chat.ID,
					MessageIDs: []int{
						opts.update.Message.ID,
						botMessage.ID,
					},
				})
				return
			}
		} else if opts.update.Message.Chat.ID == udonGroupID && len(opts.fields) > 0 {
			udoneseHandler(opts)
			return
		}

		// 不符合上方条件，即消息开头不是 / 符号的信息
		if opts.update.Message.Chat.Type == models.ChatTypePrivate {
			// 如果用户发送的是贴纸，下载并返回贴纸源文件给用户
			if opts.update.Message.Sticker != nil {
				echoStickerHandler(opts)
				return
			}

			// 不匹配上面项目的则提示不可用
			if strings.HasPrefix(opts.update.Message.Text, "/") {
				// 非冗余条件，在私聊状态下应处理用户发送的所有开头为 / 的命令
				// 与群组中不同，群组中命令末尾不指定此 bot 回应的命令无须处理，以防与群组中的其他 bot 冲突
				// botMessage, _ = 
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					Text:      "不存在的命令",
				})
				if private_log { privateLogToChat(opts.ctx, opts.thebot, opts.update) }
			} else {
				// 非命令消息，提示无操作可用
				// botMessage, _ = 
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID:    opts.update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					Text:      "无操作可用",
				})
				if private_log { privateLogToChat(opts.ctx, opts.thebot, opts.update) }

				// opts.thebot.ForwardMessages(opts.ctx, &bot.ForwardMessagesParams{
				// 	ChatID:     logChat_ID,
				// 	FromChatID: opts.update.Message.Chat.ID,
				// 	MessageIDs: []int{
				// 		opts.update.Message.ID - 1,
				// 		opts.update.Message.ID,
				// 	},
				// })
			}

			// 等待五秒删除请求信息和回复信息
			// time.Sleep(time.Second * 10)
			// opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			// 	ChatID:     opts.update.Message.Chat.ID,
			// 	MessageIDs: []int{
			// 		opts.update.Message.ID,
			// 		botMessage.ID,
			// 	},
			// })
		} else {

		}
	}
}

// 处理 inline 模式下的请求
func inlineHandler(opts *subHandlerOpts) {
	if strings.HasPrefix(opts.update.InlineQuery.Query, InlineSubCommandSymbol) {
		switch opts.fields[0][1:] { // 添加 [1:] 抛弃第一个字符是因为子命令需要一个触发符号
		// 普通命令添加到 switch 的 case 语句中，任何人都能用
		case "uaav":
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
		case savedMessageInlineCommand:
			ResultList := showSavedMessageInlineHandler(opts)
			_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.update.InlineQuery.ID,
				Results:       InlineResultPagination(opts.fields, ResultList),
			})
			if err != nil {
				log.Printf("Error when answering inline [%s] command: %v", savedMessageInlineCommand, err)
				// 本来想写一个发生错误后再给用户回答一个错误信息，让用户可以点击发送，结果同一个 ID 的 inlineQuery 只能回答一次
			}
			return
		case udoneseInlineCommand:
			ResultList := udoneseInlineHandler(opts)
			// if len(ResultList) == 0 {
			// 	_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
			// 		InlineQueryID: opts.update.InlineQuery.ID,
			// 		Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
			// 			ID:       "none",
			// 			Title:    "没有内容",
			// 			Description: fmt.Sprintf("可能是没有符合关键词的内容，也可能是处理 %s 命令的组件出现了错误", opts.fields[0][1:]),
			// 			InputMessageContent: models.InputTextMessageContent{
			// 				MessageText: fmt.Sprintf("组件没有返回内容，可能是没有符合关键词的内容，也可能是处理 %s 命令的组件出现了错误", opts.fields[0][1:]),
			// 				ParseMode: models.ParseModeMarkdownV1,
			// 			},
			// 		}},
			// 	})
			// 	if err != nil {
			// 		log.Printf("Error when answering inline [%s] command plugin no result: %v", udoneseInlineSubCommand(), err)
			// 	}
			// }

			_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.update.InlineQuery.ID,
				Results:       InlineResultPagination(opts.fields, ResultList),
			})
			if err != nil {
				log.Printf("Error when answering inline [%s] command: %v", udoneseInlineCommand, err)
				// 本来想写一个发生错误后再给用户回答一个错误信息，让用户可以点击发送，结果同一个 ID 的 inlineQuery 只能回答一次
			}
			return
		default: // default 中设定一些管理员命令和无命令提示
			if AnyContains(opts.update.InlineQuery.From.ID, logMan_IDs) {
				if strings.HasPrefix(opts.update.InlineQuery.Query, InlineSubCommandSymbol + "log") {
					logs := readLog()
					if logs != nil {
						log_count := len(logs)
						var log_all string
						for index, log := range logs {
							log_all = fmt.Sprintf("%s\n%02d %s", log_all, index, log)
						}
						_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
							InlineQueryID: opts.update.InlineQuery.ID,
							Results: []models.InlineQueryResult{
								&models.InlineQueryResultArticle{
									ID:    "log",
									Title: fmt.Sprintf("%d logs update at %s", log_count, time.Now().Format(time.RFC3339)),
									InputMessageContent: &models.InputTextMessageContent{
										MessageText: fmt.Sprintf("last update at %s\n%s", time.Now().Format(time.RFC3339), log_all),
										ParseMode: models.ParseModeMarkdownV1,
									},
								},
							},
							IsPersonal: true,
							CacheTime: 0,
						})
						if err != nil {
							log.Println("Error when answering inline query :log", err)
						}
					} else {
						log.Println("Error when reading log file")
					}
					return
				} else if strings.HasPrefix(opts.update.InlineQuery.Query, InlineSubCommandSymbol + "reload") {
					SignalsChannel.AdditionalDatas_reload <- true
					_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:    "reload",
								Title: "已请求更新",
								Description: fmt.Sprintf("last update at %s", time.Now().Format(time.RFC3339)),
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: "???",
									ParseMode: models.ParseModeMarkdownV1,
								},
							},
						},
						IsPersonal: true,
						CacheTime: 0,
					})
					if err != nil {
						log.Println("Error when answering inline query :reload", err)
					}
					return
				} else if strings.HasPrefix(opts.update.InlineQuery.Query, InlineSubCommandSymbol + "savedb") {
					SignalsChannel.Database_save <- true
					_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
						InlineQueryID: opts.update.InlineQuery.ID,
						Results: []models.InlineQueryResult{
							&models.InlineQueryResultArticle{
								ID:    "savedb",
								Title: "已请求保存",
								Description: fmt.Sprintf("last update at %s", time.Now().Format(time.RFC3339)),
								InputMessageContent: &models.InputTextMessageContent{
									MessageText: "???",
									ParseMode: models.ParseModeMarkdownV1,
								},
							},
						},
						IsPersonal: true,
						CacheTime: 0,
					})
					if err != nil {
						log.Println("Error when answering inline query :savedb", err)
					}
					return
				}
			}
			_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.update.InlineQuery.ID,
				Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:    "noinlinecommand",
					Title: fmt.Sprintf("不存在的命令 [%s]", opts.fields[0]),
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
		}
	}

	if AdditionalDatas.VoiceErr != nil {
		log.Printf("Error when reading metadata files: %v", AdditionalDatas.VoiceErr)
		opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{
				&models.InlineQueryResultVoice{
					ID:       "none",
					Title:    "读取语音文件时发生错误，请联系机器人管理员",
					Caption:  "由于无法读取到语音文件，此处被替换为预设的 `♿otto: 我是说的道理~` ",
					VoiceURL: "https://otto-hzys.huazhiwan.xyz/static/ysddTokens/wssddl.mp3",
					ParseMode: models.ParseModeMarkdownV1,
				},
			},
			CacheTime: 0,
		})
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: logChat_ID,
			Text:   fmt.Sprintf("%s\nInline Mode: some user get error, %v", time.Now().Format(time.RFC3339), AdditionalDatas.VoiceErr),
		})
		return
	} else if AdditionalDatas.Voices == nil {
		log.Printf("No voices file in voices_path: %s", voice_path)
		opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{
				&models.InlineQueryResultVoice{
					ID:       "none",
					Title:    "无法读取到语音文件，请联系机器人管理员",
					Caption:  "由于无法读取到语音文件，此处被替换为预设的 `♿otto: 我是说的道理~` ",
					VoiceURL: "https://otto-hzys.huazhiwan.xyz/static/ysddTokens/wssddl.mp3",
					ParseMode: models.ParseModeMarkdownV1,
				},
			},
			CacheTime: 0,
		})
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID:    logChat_ID,
			Text:      fmt.Sprintf("%s\nInline Mode: some user can't load voices", time.Now().Format(time.RFC3339)),
		})
		return
	}

	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult

	// 没有查询字符串或使用分页搜索符号，返回所有结果
	if opts.update.InlineQuery.Query == "" || len(opts.fields) == 1 && strings.HasPrefix(opts.fields[len(opts.fields)-1], InlinePaginationSymbol) {
		for _, voicePack := range AdditionalDatas.Voices {
			for _, voice := range voicePack.Voices {
				results = append(results, &models.InlineQueryResultVoice{
					ID:       voice.ID,
					Title:    voicePack.Name + ": " + voice.Title,
					Caption:  voice.Caption,
					VoiceURL: voice.VoiceURL,
				})
			}
		}
	} else {
		for _, voicePack := range AdditionalDatas.Voices {
			for _, voice := range voicePack.Voices {
				if InlineQueryMatchMultKeyword(opts.fields, []string{voicePack.Name, voice.Title, voice.Caption}, false) {
					results = append(results, &models.InlineQueryResultVoice{
						ID:       voice.ID,
						Title:    voicePack.Name + ": " + voice.Title,
						Caption:  voice.Caption,
						VoiceURL: voice.VoiceURL,
					})
				}
			}
		}
		if len(results) == 0 {
			results = append(results, &models.InlineQueryResultArticle{
				ID:    "none",
				Title: "没有符合关键词的内容",
				Description: fmt.Sprintf("没有找到包含 %s 的内容", opts.update.InlineQuery.Query),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}

	var inlineButton *models.InlineQueryResultsButton

	SavedMessage := database.Data.SavedMessage[opts.chatInfo.ID]

	if SavedMessage.Count == 0 && !SavedMessage.AgreePrivacyPolicy {
		inlineButton = &models.InlineQueryResultsButton{
			Text: "点击此处来尝试保存内容",
			StartParameter: "via-inline_savedmessage-help",
		}
	}

	// fmt.Println(opts.fields, len(results))

	_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: opts.update.InlineQuery.ID,
		Results:       InlineResultPagination(opts.fields, results),
		Button: inlineButton,
	})
	if err != nil {
		log.Printf("Error sending inline query response: %v", err)
		return
	}
}
