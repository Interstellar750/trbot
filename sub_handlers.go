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

func startHandler(opts *subHandlerOpts) {
	if len(opts.fields) > 1 {
		if strings.HasPrefix(opts.fields[1], "via-inline") {
			inlineArgument := strings.Split(opts.fields[1], "_")
			if inlineArgument[1] == "test" {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text: "如果您愿意帮忙，请加入测试群组帮助我们完善机器人",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "点击加入测试群组",
						URL: "https://t.me/+BomkHuFsjqc3ZGE1",
					}}}},
				})
			} else if inlineArgument[1] == "noreply" {
				return
			}
			return
		} else if opts.fields[1] == "savedmessage_privacy_policy" {
			_, err := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				Text: "<blockquote>目前此机器人仍在开发阶段中，此信息可能会有更改</blockquote>\n" +
					"本机器人提供收藏信息功能，您可以在回复一条信息时输入 /save 来收藏它，之后在 inline 模式下随时浏览您的收藏内容并发送\n\n" +

					"我们会记录哪些数据？\n" +
					"<blockquote>" +
					"1. 您的用户信息，例如 用户昵称、用户 ID、聊天类型（当您将此机器人添加到群组或频道中时）\n" +
					"2. 您的使用情况，例如 消息计数、inline 调用计数、inline 条目计数、最后向机器人发送的消息、callback_query、inline_query 以及选择的 inline 结果\n" +
					"3. 收藏信息内容，您需要注意这个，因为您是为了这个而阅读此内容，例如 存储的收藏信息数量、其图片上传到 Telegram 时的文件 ID、图片下方的文本，还有您在使用添加命令时所自定义的搜索关键词" +
					"</blockquote>\n\n" +

					"我的数据安全吗？\n" +
					"<blockquote>" +
					"这是一个早期的项目，还有很多未发现的 bug 与漏洞，因此您不能也不应该将敏感的数据存储在此机器人中，若您觉得我们收集的信息不妥，您可以不点击底部的同意按钮，我们仅会收集一些基本的信息，防止对机器人造成滥用，基本信息为前一段的 1 至 2 条目" +
					"</blockquote>\n\n" +

					"我收藏的消息，有谁可以看到?" +
					"<blockquote>" +
					"此功能被设计为每个人有单独的存储空间，如果您不手动从 inline 模式下选择信息并发送，其他用户是没法查看您的收藏列表的。不过，与上一个条目一样，为了防止滥用，我们是可以也有权利查看您收藏的内容的，请不要在其中保存隐私数据" +
					"</blockquote>\n\n" +

					"内容待补充...",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击同意以上内容",
					URL: fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy_agree", botMe.Username),
				}}}},
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				log.Println("error when send savedmessage_privacy_policy:", err)
				return
			}
			return
		} else if opts.fields[1] == "savedmessage_privacy_policy_agree" {
			opts.chatInfo.SavedMessage.AgreePrivacyPolicy = true
			SignalsChannel.Database_save <- true
			_, err :=opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				Text: "您已成功开启收藏信息功能，回复一条信息的时候发送 /save 来使用收藏功能吧！\n由于服务器性能原因，每个人的收藏数量上限默认为 100 个，您也可以从机器人的个人信息中寻找管理员来申请更高的上限\n点击下方按钮来浏览您的收藏内容",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击浏览你的收藏",
					SwitchInlineQueryCurrentChat: InlineSubCommandSymbol + "photo ",
				}}}},
			})
			if err != nil {
				log.Println("error when send savedmessage_privacy_policy_agree:", err)
			}
			return
		}
	}

	opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
		ChatID:    opts.update.Message.Chat.ID,
		Text:      fmt.Sprintf("Hello, *%s %s*\n\n您可以向此处发送一个贴纸，您将会得到一张转换后的 png 图片\n\n您也可以使用 [inline](https://telegram.org/blog/inline-bots?setln=en) 模式进行交互，点击下方的按钮来使用它", opts.update.Message.From.FirstName, opts.update.Message.From.LastName),
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
		LinkPreviewOptions: &models.LinkPreviewOptions{ IsDisabled: bot.True() },
		ParseMode: models.ParseModeMarkdownV1,
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "尝试 Inline 模式",
			SwitchInlineQueryCurrentChat: " ",
		}}}},
	})
}

func addToWriteListHandler(opts *subHandlerOpts) {
	if opts.update.Message.Chat.Type == "private" {
		botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text:   "仅限转发模式被设计为仅在群组中可用",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
		})
		time.Sleep(time.Second * 10)
		opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			ChatID:     opts.update.Message.Chat.ID,
			MessageIDs: []int{
				opts.update.Message.ID,
				botMessage.ID,
			},
		})
	} else if userIsAdmin(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, opts.update.Message.From.ID) {
		if !opts.chatInfo.IsEnableForwardonly && strings.HasSuffix(opts.update.Message.Text, fmt.Sprint(opts.update.Message.Chat.ID)) {
			if opts.chatInfo.ID != opts.update.Message.Chat.ID {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "发送的群组 ID 与当前群组的 ID 不符，请先发送 `/forwardonly`",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.chatInfo.IsEnableForwardonly = true
				log.Println("Turn forwardonly on for group", opts.update.Message.Chat.ID)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "仅限转发模式已启用",
					ParseMode: models.ParseModeMarkdownV1,
				})
				SignalsChannel.Database_save <- true
			}
		} else if opts.update.Message.Text == "/forwardonly disable" {
			if !opts.chatInfo.IsEnableForwardonly {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "此群组并没有开启仅限转发模式哦",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.chatInfo.IsEnableForwardonly = false
				log.Println("Turn forwardonly off for group", opts.update.Message.Chat.ID)
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   fmt.Sprintf("仅限转发模式已关闭，重新启用请发送 `/forwardonly %d`", opts.update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
				SignalsChannel.Database_save <- true
			}
		} else if strings.HasPrefix(opts.update.Message.Text, "/forwardonly") {
			if userIsAdmin(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, botMe.ID) && userHavePermissionDeleteMessage(opts.ctx, opts.thebot, opts.update.Message.Chat.ID, botMe.ID) {
				if opts.chatInfo.IsEnableForwardonly {
					botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
						ChatID: opts.update.Message.Chat.ID,
						Text:   "仅限转发模式已启用，无须重复开启，若要关闭，请发送 `/forwardonly disable` 来关闭它",
						ParseMode: models.ParseModeMarkdownV1,
					})
					time.Sleep(time.Second * 5)
					opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
						ChatID:     opts.update.Message.Chat.ID,
						MessageIDs: []int{
							opts.update.Message.ID,
							botMessage.ID,
						},
					})
					return
				}
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   fmt.Sprintf("请求已确定，继续发送 `/forwardonly %d` 以启用仅限转发模式", opts.update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
			} else {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   "启用此功能前，请先将机器人设为管理员\n如果还是提示本消息，请检查机器人是否有删除消息的权限",
				})
			}
		}
	} else {
		botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text:   "抱歉，您不是群组的管理员，无法为群组更改此功能",
		})
		time.Sleep(time.Second * 5)
		opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			ChatID:     opts.update.Message.Chat.ID,
			MessageIDs: []int{
				opts.update.Message.ID,
				botMessage.ID,
			},
		})
	}
}

func echoStickerHandler(opts *subHandlerOpts) {
	// 下载 webp 格式的贴纸
	fmt.Println(opts.update.Message.Sticker)

	stickerdata, isCustomSticker, err := echoSticker(opts)
	if err != nil {
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text:   fmt.Sprintf("下载贴纸时发生了一些错误\n<blockquote>Error downloading sticker: %s</blockquote>", err),
			ParseMode: models.ParseModeHTML,
		})
	}

	if stickerdata == nil {
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text:   "未能获取到贴纸",
			ParseMode: models.ParseModeMarkdownV1,
		})
		return
	}

	documentParams := &bot.SendDocumentParams{
		ChatID:    opts.update.Message.Chat.ID,
		ParseMode: models.ParseModeMarkdownV1,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
	}

	// 仅在不为自定义贴纸时显示下载整个贴纸包按钮
	if !isCustomSticker {
		documentParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "下载贴纸包中的静态贴纸", CallbackData: fmt.Sprintf("S_%s", opts.update.Message.Sticker.SetName)},
			},
			{
				{Text: "下载整个贴纸包（不转换格式）", CallbackData: fmt.Sprintf("s_%s", opts.update.Message.Sticker.SetName)},
			},
		}}
		// if opts.update.Message.Sticker.IsVideo {
		// 	documentParams.Caption  = fmt.Sprintf("[%s](https://t.me/addstickers/%s)\nsee [wikipedia/WebM](https://wikipedia.org/wiki/WebM)", opts.update.Message.Sticker.Title, opts.update.Message.Sticker.SetName)
		// 	documentParams.Document = &models.InputFileUpload{Filename: "sticker.webm", Data: stickerdata}
		// } else if opts.update.Message.Sticker.IsAnimated {
		// 	documentParams.Caption  = "see [stickers/animated-stickers](https://core.telegram.org/stickers#animated-stickers)"
		// 	documentParams.Document = &models.InputFileUpload{Filename: "sticker.tgs.file", Data: stickerdata}
		// } else {
		// 	documentParams.Document = &models.InputFileUpload{Filename: "sticker.png", Data: stickerdata}
		// }
	}

	if opts.update.Message.Sticker.IsVideo {
		documentParams.Caption  = "see [wikipedia/WebM](https://wikipedia.org/wiki/WebM)"
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.webm", Data: stickerdata}
	} else if opts.update.Message.Sticker.IsAnimated {
		documentParams.Caption  = "see [stickers/animated-stickers](https://core.telegram.org/stickers#animated-stickers)"
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.tgs.file", Data: stickerdata}
	} else {
		documentParams.Document = &models.InputFileUpload{Filename: "sticker.png", Data: stickerdata}
	}

	opts.thebot.SendDocument(opts.ctx, documentParams)
}

var currentOptions = []bool{false, false, false}

func callbackHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	// answering callback query first to let Telegram know that we received the callback query,
	// and we're handling it. Otherwise, Telegram might retry sending the update repetitively
	// as it thinks the callback query doesn't reach to our application. learn more by
	// reading the footnote of the https://core.telegram.org/bots/api#callbackquery type.
	thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	switch update.CallbackQuery.Data {
	case "btn_opt1":
		currentOptions[0] = !currentOptions[0]
	case "btn_opt2":
		currentOptions[1] = !currentOptions[1]
	case "btn_opt3":
		currentOptions[2] = !currentOptions[2]
	case "btn_select":
		thebot.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:   fmt.Sprintf("Selected options: %v", currentOptions),
		})
		// b.SendMessage(ctx, &bot.SendMessageParams{
		// 	ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		// })
		return
	}

	thebot.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ReplyMarkup: buildKeyboard(),
	})
}

func buildKeyboard() models.ReplyMarkup {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: buttonText("Option 1", currentOptions[0]), CallbackData: "btn_opt1"},
				{Text: buttonText("Option 2", currentOptions[1]), CallbackData: "btn_opt2"},
				{Text: buttonText("Option 3", currentOptions[2]), CallbackData: "btn_opt3"},
			},
			{
				{Text: "Select", CallbackData: "btn_select"},
			},
		},
	}

	return kb
}

func buttonText(text string, opt bool) string {
	if opt { return "✅ " + text }
	return "❌ " + text
}

func commandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Select multiple options",
		ReplyMarkup: buildKeyboard(),
	})
}

func udoneseHandler(opts *subHandlerOpts) {
	// 不响应来自转发的命令
	if opts.update.Message.ForwardOrigin != nil {
		return
	}

	udon, err := AdditionalDatas.Udonese, AdditionalDatas.UdoneseErr
	if err != nil {
		log.Println("some error in while read udonese list: ", err)
	}

	// 统计词使用次数
	for i, n := range udon.OnlyWord() {
		if n == opts.update.Message.Text || strings.HasPrefix(opts.update.Message.Text, n) {
			udon.List[i].Used++
			err = SaveYamlDB(udon_path, metadataFileName, *udon)
			if err != nil {
				log.Println("get some error when add used count:", err)
			}
			// fmt.Println(udon.List[i].Word, "+1", udon.List[i].Used)
		} 
	}

	if opts.fields[0] == "sms" {
		// 参数过少，提示用法
		if len(opts.fields) < 2 {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				Text:   "使用方法：发送 `sms <词>` 来查看对应的意思",
				ParseMode: models.ParseModeMarkdownV1,
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击浏览全部词与意思",
					SwitchInlineQueryCurrentChat: InlineSubCommandSymbol + "sms ",
				}}}},
			})
			return
		}

		// 在数据库循环查找这个词
		for _, word := range udon.List {
			if strings.EqualFold(word.Word, opts.fields[1]) && len(word.MeaningList) > 0 {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ParseMode: models.ParseModeHTML,
				})
				return
			}
		}

		// 到这里就是没找到，提示没有
		botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
		})

		time.Sleep(time.Second * 10)
		opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			ChatID: opts.update.Message.Chat.ID,
			MessageIDs: []int{
				botMessage.ID,
			},
		})

		return
	} else if opts.fields[0] == "udonese" {
		if len(opts.fields) < 3 {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID:    opts.update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				Text: "使用 `udonese <词> <单个意思>` 来添加记录",
				ParseMode: models.ParseModeMarkdownV1,
			})
			return
		}

		meaning := strings.TrimSpace(opts.update.Message.Text[len(opts.fields[0])+len(opts.fields[1])+2:])

		var fromName string
		var fromID   int64
		var viaName  string
		var viaID    int64

		if opts.update.Message.ReplyToMessage == nil {
			fromName = showUserName(opts.update.Message.From)
			fromID = opts.update.Message.From.ID
		} else {
			fromName = showUserName(opts.update.Message.ReplyToMessage.From)
			fromID = opts.update.Message.ReplyToMessage.From.ID
			viaName = showUserName(opts.update.Message.From)
			viaID = opts.update.Message.From.ID
		}

		var pendingMessage string
		var botMessage *models.Message

		oldMeaning := addUdonese(udon, &UdoneseWord{
			Word: opts.fields[1],
			MeaningList: []UdoneseMeaning{ {
				Meaning: meaning,
				FromID: fromID,
				FromName: fromName,
				ViaID: viaID,
				ViaName: viaName,
			}},
		})
		if oldMeaning != nil {
			pendingMessage += fmt.Sprintf("[%s] 意思已存在于 [%s] 中:\n", meaning, oldMeaning.Word)
			for i, s := range oldMeaning.MeaningList {
				if meaning == s.Meaning {
					if s.ViaID != 0 { // 通过回复添加
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] From <a href=\"https://t.me/@id%d\">%s</a> Via <a href=\"https://t.me/@id%d\">%s</a>\n",
							i + 1, s.Meaning, s.FromID, s.FromName, s.ViaID, s.ViaName,
						)
					} else if s.FromID == 0 { // 有添加人信息
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] From <a href=\"https://t.me/@id%d\">%s</a>\n",
							i + 1, s.Meaning, s.FromID, s.FromName,
						)
					} else { // 只有意思
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s]\n", i + 1, s.Meaning)
					}
				}
			}
		} else {
			err = SaveYamlDB(udon_path, metadataFileName, *udon)
			if err != nil {
				pendingMessage += fmt.Sprintln("保存语句时似乎发生了一些错误:\n", err)
			} else {
				pendingMessage += fmt.Sprintf("已添加 [<code>%s</code>]\n", opts.fields[1])
				if viaID != 0 { // 通过回复添加
					pendingMessage += fmt.Sprintf("[%s] From <a href=\"https://t.me/@id%d\">%s</a> Via <a href=\"https://t.me/@id%d\">%s</a>\n",
						meaning, fromID, fromName, viaID, viaName,
					)
				} else if fromID != 0 { // 普通命令添加
					pendingMessage += fmt.Sprintf("[%s] From <a href=\"https://t.me/@id%d\">%s</a>\n",
						meaning, fromID, fromName,
					)
				}
			}
		}

		pendingMessage += fmt.Sprintln("<blockquote>发送的消息与此消息将在十秒后删除</blockquote>")
		botMessage, _= opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text: pendingMessage,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			ParseMode: models.ParseModeHTML,
		})
		if err == nil {
			time.Sleep(time.Second * 10)
			opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
				ChatID: opts.update.Message.Chat.ID,
				MessageIDs: []int{
					opts.update.Message.ID,
					botMessage.ID,
				},
			})
		}
		return
	} else if len(opts.fields) > 1 && strings.HasSuffix(opts.update.Message.Text, "ssm") {
		// 在数据库循环查找这个词
		for _, word := range udon.List {
			if strings.EqualFold(word.Word, opts.fields[1]) && len(word.MeaningList) > 0 {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ParseMode: models.ParseModeHTML,
				})
				return
			}
		}

		// 到这里就是没找到，提示没有
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
		})
	}
}

func saveMessageHandler(opts *subHandlerOpts) {
	userData, _ := getIDInfoAndIndex(&opts.update.Message.From.ID)

	messageParams := &bot.SendMessageParams{
		ChatID: opts.update.Message.Chat.ID,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
		ParseMode: models.ParseModeHTML,
	}

	if !userData.SavedMessage.AgreePrivacyPolicy {
		messageParams.Text = "此功能需要保存一些信息才能正常工作，在使用这个功能前，请先阅读一下我们会保存哪些信息"
		messageParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击查看",
			URL: fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy", botMe.Username),
		}}}}
		_, err := opts.thebot.SendMessage(opts.ctx, messageParams)
		if err != nil {
			log.Printf("Error response /save command initial info: %v", err)
		}
		return
	}

	if userData.SavedMessage.Limit == 0 && userData.SavedMessage.Count == 0 {
		userData.SavedMessage.Limit = 100
	}
	if userData.SavedMessage.Limit == 0 {
		// 若不是初次添加，为 0 就是不限制
	} else if userData.SavedMessage.Count >= userData.SavedMessage.Limit {
		messageParams.Text = "已达到限制，无法保存更多图片"
		_, err := opts.thebot.SendMessage(opts.ctx, messageParams)
		if err != nil {
			log.Printf("Error response /save command: %v", err)
		}
		return
	}

	// var pendingMessage string
	if opts.update.Message.ReplyToMessage != nil {
		if opts.update.Message.ReplyToMessage.Photo != nil {
			
			userData.SavedMessage.Item.Photo = append(userData.SavedMessage.Item.Photo, SavedMessageTypeCachedPhoto{
				ID: fmt.Sprintf("%d-%d", userData.ID, userData.SavedMessage.Count),
				FileID: opts.update.Message.ReplyToMessage.Photo[len(opts.update.Message.ReplyToMessage.Photo)-1].FileID,
				Caption: opts.update.Message.ReplyToMessage.Caption,
				// Title: opts.update.Message.Text[len(opts.fields[0]):],
			})
			userData.SavedMessage.Count++
			SignalsChannel.Database_save <- true
			messageParams.Text = "已保存图片"
			messageParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "点击浏览你的收藏",
				SwitchInlineQueryCurrentChat: InlineSubCommandSymbol + "photo ",
			}}}}
		} else {
			messageParams.Text = "Reply to a photo to save it"
		}
	} else {
		messageParams.Text = "Reply to a photo to save it"
	}

	// fmt.Println(opts.chatInfo)

	_, err := opts.thebot.SendMessage(opts.ctx, messageParams)
	if err != nil {
		log.Printf("Error response /save command: %v", err)
	}
}
