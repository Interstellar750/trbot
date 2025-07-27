package saved_message

import (
	"fmt"
	"reflect"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/inline_utils"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/message_utils"
	"unicode/utf8"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

func saveMessageHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "saveMessageHandler").
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	var handlerErr flaterr.MultErr
	var needSave  bool = true
	UserSavedMessage := SavedMessageSet[opts.Message.From.ID]

	messageParams := &bot.SendMessageParams{
		ChatID:          opts.Message.Chat.ID,
		ParseMode:       models.ParseModeHTML,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
			{
				Text:                         "点击浏览您的收藏",
				SwitchInlineQueryCurrentChat: configs.BotConfig.InlineSubCommandSymbol + "saved ",
			},
			{
				Text:         "关闭",
				CallbackData: "delete_this_message",
			},
		}}},
	}

	if !UserSavedMessage.AgreePrivacyPolicy {
		messageParams.Text = "此功能需要保存一些信息才能正常工作，在使用这个功能前，请先阅读一下我们会保存哪些信息"
		messageParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击查看",
			URL:  fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy", consts.BotMe.Username),
		}}}}
		_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "need agree privacy policy").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "need agree privacy policy", err)
		}
	} else {
		if UserSavedMessage.Limit == 0 && UserSavedMessage.Count == 0 {
			// 每个用户初次添加时，默认限制 100 条
			UserSavedMessage.Limit = 100
		}

		// 若不是初次添加，为 0 就是不限制
		if UserSavedMessage.Limit != 0 && UserSavedMessage.Count >= UserSavedMessage.Limit {
			messageParams.Text = "已达到限制，无法保存更多内容"
			_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "reach saved limit").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "reach saved limit", err)
			}
		} else {
			// var pendingMessage string
			if opts.Message.ReplyToMessage == nil {
				needSave = false
				messageParams.Text = "在回复一条消息的同时发送 <code>/save</code> 来添加"
			} else {
				var DescriptionText string
				// 获取使用命令保存时设定的描述
				if len(opts.Message.Text) > len(opts.Fields[0])+1 {
					DescriptionText = opts.Message.Text[len(opts.Fields[0])+1:]
				}

				var originInfo *OriginInfo
				if opts.Message.ReplyToMessage.ForwardOrigin != nil && opts.Message.ReplyToMessage.ForwardOrigin.MessageOriginHiddenUser == nil {
					originInfo = getMessageOriginData(opts.Message.ReplyToMessage.ForwardOrigin)
				} else if opts.Message.Chat.Type != models.ChatTypePrivate {
					originInfo = getMessageLink(opts.Message)
				}

				var isSaved bool
				var messageLength int
				var pendingEntitites []models.MessageEntity
				var needChangeEntitites bool = true

				if opts.Message.ReplyToMessage.Caption != "" {
					messageLength = utf8.RuneCountInString(opts.Message.ReplyToMessage.Caption)
					pendingEntitites = opts.Message.ReplyToMessage.CaptionEntities
				} else if opts.Message.ReplyToMessage.Text != "" {
					messageLength = utf8.RuneCountInString(opts.Message.ReplyToMessage.Text)
					pendingEntitites = opts.Message.ReplyToMessage.Entities
				} else {
					needChangeEntitites = false
				}

				if needChangeEntitites {
					// 若字符长度大于设定的阈值，添加折叠样式引用再保存
					if messageLength > textExpandableLength {
						if len(pendingEntitites) == 1 && pendingEntitites[0].Type == models.MessageEntityTypeBlockquote && pendingEntitites[0].Offset == 0 && pendingEntitites[0].Length == messageLength {
							// 如果消息仅为一个消息格式实体，且是不折叠形式的引用，则将格式实体改为可折叠格式引用后再保存
							pendingEntitites = []models.MessageEntity{{
								Type:   models.MessageEntityTypeExpandableBlockquote,
								Offset: 0,
								Length: messageLength,
							}}
						} else {
							// 其他则仅在末尾加一个可折叠形式的引用
							pendingEntitites = append(pendingEntitites, models.MessageEntity{
								Type:   models.MessageEntityTypeExpandableBlockquote,
								Offset: 0,
								Length: messageLength,
							})
						}
					}
				}

				replyMsgType := message_utils.GetMessageType(opts.Message.ReplyToMessage)
				switch {
					case replyMsgType.OnlyText:
						for i, n := range UserSavedMessage.Item.OnlyText {
							if n.TitleAndMessageText == opts.Message.ReplyToMessage.Text && reflect.DeepEqual(n.Entities, pendingEntitites) {
								isSaved = true
								messageParams.Text = "已保存过该文本\n"
								if DescriptionText != "" {
									if n.Description == "" {
										messageParams.Text += fmt.Sprintf("已为此文本添加搜索关键词 [ %s ]", DescriptionText)
									} else if DescriptionText == n.Description {
										messageParams.Text += fmt.Sprintf("此文本的搜索关键词未修改 [ %s ]", DescriptionText)
										needSave = false
										break
									} else {
										messageParams.Text += fmt.Sprintf("已将此文本的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
									}
									n.Description = DescriptionText
									UserSavedMessage.Item.OnlyText[i] = n
								}
								break
							}
						}

						if !isSaved {
							UserSavedMessage.Item.OnlyText = append(UserSavedMessage.Item.OnlyText, SavedMessageTypeCachedOnlyText{
								ID:                  fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
								TitleAndMessageText: opts.Message.ReplyToMessage.Text,
								Description:         DescriptionText,
								Entities:            pendingEntitites,
								LinkPreviewOptions:  opts.Message.ReplyToMessage.LinkPreviewOptions,
								OriginInfo:          originInfo,
							})
							UserSavedMessage.Count++
							UserSavedMessage.SavedTimes++
							messageParams.Text = "已保存文本"
						}
					case replyMsgType.Audio:
						for i, n := range UserSavedMessage.Item.Audio {
							if n.FileID == opts.Message.ReplyToMessage.Audio.FileID {
								isSaved = true
								messageParams.Text = "已保存过该音乐\n"
								if DescriptionText != "" {
									if n.Description == "" {
										messageParams.Text += fmt.Sprintf("已为此音乐添加搜索关键词 [ %s ]", DescriptionText)
									} else if DescriptionText == n.Description {
										messageParams.Text += fmt.Sprintf("此音乐的搜索关键词未修改 [ %s ]", DescriptionText)
										needSave = false
										break
									} else {
										messageParams.Text += fmt.Sprintf("已将此音乐的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
									}
									n.Description = DescriptionText
									UserSavedMessage.Item.Audio[i] = n
								}
								break
							}
						}
						if !isSaved {
							UserSavedMessage.Item.Audio = append(UserSavedMessage.Item.Audio, SavedMessageTypeCachedAudio{
								ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
								FileID:          opts.Message.ReplyToMessage.Audio.FileID,
								Title:           opts.Message.ReplyToMessage.Audio.Title,
								FileName:        opts.Message.ReplyToMessage.Audio.FileName,
								Description:     DescriptionText,
								Caption:         opts.Message.ReplyToMessage.Caption,
								CaptionEntities: pendingEntitites,
								OriginInfo:      originInfo,
							})
							UserSavedMessage.Count++
							UserSavedMessage.SavedTimes++
							messageParams.Text = "已保存音乐"
						}
					case replyMsgType.Animation:
						for i, n := range UserSavedMessage.Item.Mpeg4gif {
							if n.FileID == opts.Message.ReplyToMessage.Animation.FileID {
								isSaved = true
								messageParams.Text = "已保存过该 GIF\n"
								if DescriptionText != "" {
									if n.Description == "" {
										messageParams.Text += fmt.Sprintf("已为此 GIF 添加搜索关键词 [ %s ]", DescriptionText)
									} else if DescriptionText == n.Description {
										messageParams.Text += fmt.Sprintf("此 GIF 搜索关键词未修改 [ %s ]", DescriptionText)
										needSave = false
										break
									} else {
										messageParams.Text += fmt.Sprintf("已将此 GIF 的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
									}
									n.Description = DescriptionText
									UserSavedMessage.Item.Mpeg4gif[i] = n
								}
								break
							}
						}
						if !isSaved {
							UserSavedMessage.Item.Mpeg4gif = append(UserSavedMessage.Item.Mpeg4gif, SavedMessageTypeCachedMpeg4Gif{
								ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
								FileID:          opts.Message.ReplyToMessage.Animation.FileID,
								Title:           opts.Message.ReplyToMessage.Caption,
								Description:     DescriptionText,
								Caption:         opts.Message.ReplyToMessage.Caption,
								CaptionEntities: pendingEntitites,
								OriginInfo:      originInfo,
							})
							UserSavedMessage.Count++
							UserSavedMessage.SavedTimes++
							messageParams.Text = "已保存 GIF"
						}
					case replyMsgType.Document:
						if opts.Message.ReplyToMessage.Document.MimeType == "image/gif" {
							for i, n := range UserSavedMessage.Item.Gif {
								if n.FileID == opts.Message.ReplyToMessage.Document.FileID {
									isSaved = true
									messageParams.Text = "已保存过该 GIF (文件)\n"
									if DescriptionText != "" {
										if n.Description == "" {
											messageParams.Text += fmt.Sprintf("已为此 GIF (文件) 添加搜索关键词 [ %s ]", DescriptionText)
										} else if DescriptionText == n.Description {
											messageParams.Text += fmt.Sprintf("此 GIF (文件) 搜索关键词未修改 [ %s ]", DescriptionText)
											needSave = false
											break
										} else {
											messageParams.Text += fmt.Sprintf("已将此 GIF (文件) 的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
										}
										n.Description = DescriptionText
										UserSavedMessage.Item.Gif[i] = n
									}
									break
								}
							}
							if !isSaved {
								UserSavedMessage.Item.Gif = append(UserSavedMessage.Item.Gif, SavedMessageTypeCachedGif{
									ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
									FileID:          opts.Message.ReplyToMessage.Document.FileID,
									Description:     DescriptionText,
									Caption:         opts.Message.ReplyToMessage.Caption,
									CaptionEntities: pendingEntitites,
									OriginInfo:      originInfo,
								})
								UserSavedMessage.Count++
								UserSavedMessage.SavedTimes++
								messageParams.Text = "已保存 GIF (文件)"
							}
						} else {
							for i, n := range UserSavedMessage.Item.Document {
								if n.FileID == opts.Message.ReplyToMessage.Document.FileID {
									isSaved = true
									messageParams.Text = "已保存过该文件\n"
									if DescriptionText != "" {
										if n.Description == "" {
											messageParams.Text += fmt.Sprintf("已为此文件添加搜索关键词 [ %s ]", DescriptionText)
										} else if DescriptionText == n.Description {
											messageParams.Text += fmt.Sprintf("此文件搜索关键词未修改 [ %s ]", DescriptionText)
											needSave = false
											break
										} else {
											messageParams.Text += fmt.Sprintf("已将此文件的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
										}
										n.Description = DescriptionText
										UserSavedMessage.Item.Document[i] = n
									}
									break
								}
							}
							if !isSaved {
								UserSavedMessage.Item.Document = append(UserSavedMessage.Item.Document, SavedMessageTypeCachedDocument{
									ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
									FileID:          opts.Message.ReplyToMessage.Document.FileID,
									Title:           opts.Message.ReplyToMessage.Document.FileName,
									Description:     DescriptionText,
									Caption:         opts.Message.ReplyToMessage.Caption,
									CaptionEntities: pendingEntitites,
									OriginInfo:      originInfo,
								})
								UserSavedMessage.Count++
								UserSavedMessage.SavedTimes++
								messageParams.Text = "已保存文件"
							}
						}
					case replyMsgType.Photo:
						for i, n := range UserSavedMessage.Item.Photo {
							if n.FileID == opts.Message.ReplyToMessage.Photo[len(opts.Message.ReplyToMessage.Photo)-1].FileID {
								isSaved = true
								messageParams.Text = "已保存过该图片\n"
								if DescriptionText != "" {
									if n.Description == "" {
										messageParams.Text += fmt.Sprintf("已为此图片添加搜索关键词 [ %s ]", DescriptionText)
									} else if DescriptionText == n.Description {
										messageParams.Text += fmt.Sprintf("此图片搜索关键词未修改 [ %s ]", DescriptionText)
										needSave = false
										break
									} else {
										messageParams.Text += fmt.Sprintf("已将此图片的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
									}
									n.Description = DescriptionText
									UserSavedMessage.Item.Photo[i] = n
								}
								break
							}
						}
						if !isSaved {
							UserSavedMessage.Item.Photo = append(UserSavedMessage.Item.Photo, SavedMessageTypeCachedPhoto{
								ID:                fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
								FileID:            opts.Message.ReplyToMessage.Photo[len(opts.Message.ReplyToMessage.Photo)-1].FileID,
								// Title:             opts.Message.ReplyToMessage.Caption,
								Description:       DescriptionText,
								Caption:           opts.Message.ReplyToMessage.Caption,
								CaptionEntities:   pendingEntitites,
								CaptionAboveMedia: opts.Message.ReplyToMessage.ShowCaptionAboveMedia,
								OriginInfo:        originInfo,
							})
							UserSavedMessage.Count++
							UserSavedMessage.SavedTimes++
							messageParams.Text = "已保存图片"
						}
					case replyMsgType.Sticker:
						for i, n := range UserSavedMessage.Item.Sticker {
							if n.FileID == opts.Message.ReplyToMessage.Sticker.FileID {
								isSaved = true
								messageParams.Text = "已保存过该贴纸\n"
								if DescriptionText != "" {
									if n.Description == "" {
										messageParams.Text += fmt.Sprintf("已为此贴纸添加搜索关键词 [ %s ]", DescriptionText)
									} else if DescriptionText == n.Description {
										messageParams.Text += fmt.Sprintf("此贴纸搜索关键词未修改 [ %s ]", DescriptionText)
										needSave = false
										break
									} else {
										messageParams.Text += fmt.Sprintf("已将此贴纸的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
									}
									n.Description = DescriptionText
									UserSavedMessage.Item.Sticker[i] = n
								}
								break
							}
						}

						if !isSaved {
							if opts.Message.ReplyToMessage.Sticker.SetName != "" {
								stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{Name: opts.Message.ReplyToMessage.Sticker.SetName})
								if err != nil {
									logger.Warn().
										Err(err).
										Str("setName", opts.Message.ReplyToMessage.Sticker.SetName).
										Msg("Failed to get sticker set info, save it as a custom sticker")
								}
								if stickerSet != nil {
									// 属于一个贴纸包中的贴纸
									UserSavedMessage.Item.Sticker = append(UserSavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
										ID:          fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
										FileID:      opts.Message.ReplyToMessage.Sticker.FileID,
										SetName:     stickerSet.Name,
										SetTitle:    stickerSet.Title,
										Description: DescriptionText,
										Emoji:       opts.Message.ReplyToMessage.Sticker.Emoji,
										OriginInfo:  originInfo,
									})
								} else {
									// 有贴纸信息，但是对应的贴纸包已经删掉了
									UserSavedMessage.Item.Sticker = append(UserSavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
										ID:          fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
										FileID:      opts.Message.ReplyToMessage.Sticker.FileID,
										Description: DescriptionText,
										Emoji:       opts.Message.ReplyToMessage.Sticker.Emoji,
										OriginInfo:  originInfo,
									})
								}
							} else {
								UserSavedMessage.Item.Sticker = append(UserSavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
									ID:          fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
									FileID:      opts.Message.ReplyToMessage.Sticker.FileID,
									Description: DescriptionText,
									Emoji:       opts.Message.ReplyToMessage.Sticker.Emoji,
									OriginInfo:  originInfo,
								})
							}
							UserSavedMessage.Count++
							UserSavedMessage.SavedTimes++
							messageParams.Text = "已保存贴纸"
						}
					case replyMsgType.Video:
						for i, n := range UserSavedMessage.Item.Video {
							if n.FileID == opts.Message.ReplyToMessage.Video.FileID {
								isSaved = true
								messageParams.Text = "已保存过该视频\n"
								if DescriptionText != "" {
									if n.Description == "" {
										messageParams.Text += fmt.Sprintf("已为此视频添加搜索关键词 [ %s ]", DescriptionText)
									} else if DescriptionText == n.Description {
										messageParams.Text += fmt.Sprintf("此视频搜索关键词未修改 [ %s ]", DescriptionText)
										needSave = false
										break
									} else {
										messageParams.Text += fmt.Sprintf("已将此视频的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
									}
									n.Description = DescriptionText
									UserSavedMessage.Item.Video[i] = n
								}
								break
							}
						}
						if !isSaved {
							videoTitle := opts.Message.ReplyToMessage.Video.FileName
							if videoTitle == "" {
								videoTitle = "video.mp4"
							}
							UserSavedMessage.Item.Video = append(UserSavedMessage.Item.Video, SavedMessageTypeCachedVideo{
								ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
								FileID:          opts.Message.ReplyToMessage.Video.FileID,
								Title:           videoTitle,
								Description:     DescriptionText,
								Caption:         opts.Message.ReplyToMessage.Caption,
								CaptionEntities: pendingEntitites,
								OriginInfo:      originInfo,
							})
							UserSavedMessage.Count++
							UserSavedMessage.SavedTimes++
							messageParams.Text = "已保存视频"
						}
					case replyMsgType.VideoNote:
						for i, n := range UserSavedMessage.Item.VideoNote {
							if n.FileID == opts.Message.ReplyToMessage.VideoNote.FileID {
								isSaved = true
								messageParams.Text = "已保存过该圆形视频\n"
								if DescriptionText != "" {
									if n.Description == "" {
										messageParams.Text += fmt.Sprintf("已为此圆形视频添加搜索关键词 [ %s ]", DescriptionText)
									} else if DescriptionText == n.Description {
										messageParams.Text += fmt.Sprintf("此圆形视频搜索关键词未修改 [ %s ]", DescriptionText)
										needSave = false
										break
									} else {
										messageParams.Text += fmt.Sprintf("已将此圆形视频的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
									}
									n.Description = DescriptionText
									UserSavedMessage.Item.VideoNote[i] = n
								}
								break
							}
						}
						if !isSaved {
							UserSavedMessage.Item.VideoNote = append(UserSavedMessage.Item.VideoNote, SavedMessageTypeCachedVideoNote{
								ID:          fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
								FileID:      opts.Message.ReplyToMessage.VideoNote.FileID,
								Title:       opts.Message.ReplyToMessage.VideoNote.FileUniqueID,
								Description: DescriptionText,
								OriginInfo:  originInfo,
							})
							UserSavedMessage.Count++
							UserSavedMessage.SavedTimes++
							messageParams.Text = "已保存圆形视频"
						}
					case replyMsgType.Voice:
						for i, n := range UserSavedMessage.Item.Voice {
							if n.FileID == opts.Message.ReplyToMessage.Voice.FileID {
								isSaved = true
								messageParams.Text = "已保存过该语音\n"
								if DescriptionText != "" {
									if n.Description == "" {
										messageParams.Text += fmt.Sprintf("已为此语音添加搜索关键词 [ %s ]", DescriptionText)
									} else if DescriptionText == n.Description {
										messageParams.Text += fmt.Sprintf("此语音搜索关键词未修改 [ %s ]", DescriptionText)
										needSave = false
										break
									} else {
										messageParams.Text += fmt.Sprintf("已将此语音的搜索关键词从 [ %s ] 改为 [ %s ]", n.Description, DescriptionText)
									}
									n.Description = DescriptionText
									UserSavedMessage.Item.Voice[i] = n
								}
								break
							}
						}
						if !isSaved {
							voiceTitle := DescriptionText
							if voiceTitle == "" {
								voiceTitle = opts.Message.ReplyToMessage.Voice.MimeType
							}
							UserSavedMessage.Item.Voice = append(UserSavedMessage.Item.Voice, SavedMessageTypeCachedVoice{
								ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
								FileID:          opts.Message.ReplyToMessage.Voice.FileID,
								Title:           voiceTitle,
								Description:     DescriptionText,
								Caption:         opts.Message.ReplyToMessage.Caption,
								CaptionEntities: pendingEntitites,
								OriginInfo:      originInfo,
							})
							UserSavedMessage.Count++
							UserSavedMessage.SavedTimes++
							messageParams.Text = "已保存语音"
						}
					default:
						messageParams.Text = "暂不支持的消息类型"
				}

				if needSave {
					SavedMessageSet[opts.Message.From.ID] = UserSavedMessage
					err := SaveSavedMessageList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Str("messageType", string(replyMsgType.AsValue())).
							Msg("Failed to save savedmessage list after save a item")
						handlerErr.Addf("failed to save savedmessage list after save a item: %w", err)

						_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID:          opts.Message.Chat.ID,
							Text:            fmt.Sprintf("保存内容时保存收藏列表数据库失败，请稍后再试或联系机器人管理员\n<blockquote expandable>%s<expandable>", err.Error()),
							ParseMode:       models.ParseModeHTML,
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						})
						if err != nil {
							logger.Error().
								Err(err).
								Str("messageType", string(replyMsgType.AsValue())).
								Str("content", "failed to save savedmessage list notice").
								Msg(flaterr.SendMessage.Str())
							handlerErr.Addt(flaterr.SendMessage, "failed to save savedmessage list notice", err)
						}

						return handlerErr.Flat()
					}
				}
			}

			_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Message.From)).
					Str("content", "saved message response").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "saved message response", err)
			}
		}
	}

	return handlerErr.Flat()
}

// func channelSaveMessageHandler(opts *handler_structs.SubHandlerParams) {
// 	ChannelSavedMessage := SavedMessageSet[opts.Update.ChannelPost.From.ID]

// 	messageParams := &bot.SendMessageParams{
// 		ReplyParameters: &models.ReplyParameters{MessageID: opts.Update.Message.ID},
// 		ParseMode:       models.ParseModeHTML,
// 	}

// 	if !ChannelSavedMessage.AgreePrivacyPolicy {
// 		messageParams.ChatID = opts.Update.ChannelPost.From.ID
// 		messageParams.Text = "此功能需要保存一些信息才能正常工作，在使用这个功能前，请先阅读一下我们会保存哪些信息"
// 		messageParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
// 			Text: "点击查看",
// 			URL:  fmt.Sprintf("https://t.me/%s?start=savedmessage_channel_privacy_policy", consts.BotMe.Username),
// 		}}}}
// 		_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
// 		if err != nil {
// 			log.Printf("Error response /save command initial info: %v", err)
// 		}
// 		return
// 	}

// 	if ChannelSavedMessage.DiscussionID == 0 {
// 		messageParams.Text = "您需要为此频道绑定一个讨论群组，用于接收收藏成功的确认信息与关键词更改"
// 		_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
// 		if err != nil {
// 			log.Printf("Error response /save command initial info: %v", err)
// 		}
// 	}

// }

func saveMessageFromCallbackQueryHandler(opts *handler_params.Message) error {
	if opts.Message == nil { return nil }

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "saveMessageFromCallBackQuery").
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var targetMessage *models.Message = opts.Message

	messageParams := &bot.SendMessageParams{
		ChatID:          opts.ChatInfo.ID,
		ParseMode:       models.ParseModeHTML,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text:         "关闭",
			CallbackData: "delete_this_message",
		}}}},
	}

	var handlerErr flaterr.MultErr

	UserSavedMessage := SavedMessageSet[opts.ChatInfo.ID]

	var originInfo *OriginInfo
	if targetMessage.ForwardOrigin != nil && targetMessage.ForwardOrigin.MessageOriginHiddenUser == nil {
		originInfo = getMessageOriginData(targetMessage.ForwardOrigin)
	} else if targetMessage.Chat.Type != models.ChatTypePrivate {
		originInfo = getMessageLink(targetMessage)
	}

	var isSaved             bool
	var messageLength       int
	var pendingEntitites    []models.MessageEntity
	var needChangeEntitites bool = true

	if targetMessage.Caption != "" {
		messageLength = utf8.RuneCountInString(targetMessage.Caption)
		pendingEntitites = targetMessage.CaptionEntities
	} else if targetMessage.Text != "" {
		messageLength = utf8.RuneCountInString(targetMessage.Text)
		pendingEntitites = targetMessage.Entities
	} else {
		needChangeEntitites = false
	}

	if needChangeEntitites {
		// 若字符长度大于设定的阈值，添加折叠样式引用再保存
		if messageLength > textExpandableLength {
			if len(pendingEntitites) == 1 && pendingEntitites[0].Type == models.MessageEntityTypeBlockquote && pendingEntitites[0].Offset == 0 && pendingEntitites[0].Length == messageLength {
				// 如果消息仅为一个消息格式实体，且是不折叠形式的引用，则将格式实体改为可折叠格式引用后再保存
				pendingEntitites = []models.MessageEntity{{
					Type:   models.MessageEntityTypeExpandableBlockquote,
					Offset: 0,
					Length: messageLength,
				}}
			} else {
				// 其他则仅在末尾加一个可折叠形式的引用
				pendingEntitites = append(pendingEntitites, models.MessageEntity{
					Type:   models.MessageEntityTypeExpandableBlockquote,
					Offset: 0,
					Length: messageLength,
				})
			}
		}
	}

	replyMsgType := message_utils.GetMessageType(targetMessage)
	switch {
		case replyMsgType.OnlyText:
			for _, n := range UserSavedMessage.Item.OnlyText {
				if n.TitleAndMessageText == targetMessage.Text && reflect.DeepEqual(n.Entities, pendingEntitites) {
					isSaved = true
					messageParams.Text = "已保存过该文本\n"
					break
				}
			}

			if !isSaved {
				UserSavedMessage.Item.OnlyText = append(UserSavedMessage.Item.OnlyText, SavedMessageTypeCachedOnlyText{
					ID:                  fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					TitleAndMessageText: targetMessage.Text,
					Entities:            pendingEntitites,
					LinkPreviewOptions:  targetMessage.LinkPreviewOptions,
					OriginInfo:          originInfo,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				messageParams.Text = "已保存文本"
			}
		case replyMsgType.Audio:
			for _, n := range UserSavedMessage.Item.Audio {
				if n.FileID == targetMessage.Audio.FileID {
					isSaved = true
					messageParams.Text = "已保存过该音乐\n"
					break
				}
			}
			if !isSaved {
				UserSavedMessage.Item.Audio = append(UserSavedMessage.Item.Audio, SavedMessageTypeCachedAudio{
					ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID:          targetMessage.Audio.FileID,
					Title:           targetMessage.Audio.Title,
					FileName:        targetMessage.Audio.FileName,
					Caption:         targetMessage.Caption,
					CaptionEntities: pendingEntitites,
					OriginInfo:      originInfo,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				messageParams.Text = "已保存音乐"
			}
		case replyMsgType.Animation:
			for _, n := range UserSavedMessage.Item.Mpeg4gif {
				if n.FileID == targetMessage.Animation.FileID {
					isSaved = true
					messageParams.Text = "已保存过该 GIF\n"
					break
				}
			}
			if !isSaved {
				UserSavedMessage.Item.Mpeg4gif = append(UserSavedMessage.Item.Mpeg4gif, SavedMessageTypeCachedMpeg4Gif{
					ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID:          targetMessage.Animation.FileID,
					Title:           targetMessage.Caption,
					Caption:         targetMessage.Caption,
					CaptionEntities: pendingEntitites,
					OriginInfo:      originInfo,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				messageParams.Text = "已保存 GIF"
			}
		case replyMsgType.Document:
			if targetMessage.Document.MimeType == "image/gif" {
				for _, n := range UserSavedMessage.Item.Gif {
					if n.FileID == targetMessage.Document.FileID {
						isSaved = true
						messageParams.Text = "已保存过该 GIF (文件)\n"
						break
					}
				}
				if !isSaved {
					UserSavedMessage.Item.Gif = append(UserSavedMessage.Item.Gif, SavedMessageTypeCachedGif{
						ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
						FileID:          targetMessage.Document.FileID,
						Caption:         targetMessage.Caption,
						CaptionEntities: pendingEntitites,
						OriginInfo:      originInfo,
					})
					UserSavedMessage.Count++
					UserSavedMessage.SavedTimes++
					messageParams.Text = "已保存 GIF (文件)"
				}
			} else {
				for _, n := range UserSavedMessage.Item.Document {
					if n.FileID == targetMessage.Document.FileID {
						isSaved = true
						messageParams.Text = "已保存过该文件\n"
						break
					}
				}
				if !isSaved {
					UserSavedMessage.Item.Document = append(UserSavedMessage.Item.Document, SavedMessageTypeCachedDocument{
						ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
						FileID:          targetMessage.Document.FileID,
						Title:           targetMessage.Document.FileName,
						Caption:         targetMessage.Caption,
						CaptionEntities: pendingEntitites,
						OriginInfo:      originInfo,
					})
					UserSavedMessage.Count++
					UserSavedMessage.SavedTimes++
					messageParams.Text = "已保存文件"
				}
			}
		case replyMsgType.Photo:
			for _, n := range UserSavedMessage.Item.Photo {
				if n.FileID == targetMessage.Photo[len(targetMessage.Photo)-1].FileID {
					isSaved = true
					messageParams.Text = "已保存过该图片\n"
					break
				}
			}
			if !isSaved {
				UserSavedMessage.Item.Photo = append(UserSavedMessage.Item.Photo, SavedMessageTypeCachedPhoto{
					ID:                fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID:            targetMessage.Photo[len(targetMessage.Photo)-1].FileID,
					Caption:           targetMessage.Caption,
					CaptionEntities:   pendingEntitites,
					CaptionAboveMedia: targetMessage.ShowCaptionAboveMedia,
					OriginInfo:        originInfo,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				messageParams.Text = "已保存图片"
			}
		case replyMsgType.Sticker:
			for _, n := range UserSavedMessage.Item.Sticker {
				if n.FileID == targetMessage.Sticker.FileID {
					isSaved = true
					messageParams.Text = "已保存过该贴纸\n"
					break
				}
			}

			if !isSaved {
				if targetMessage.Sticker.SetName != "" {
					stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{Name: targetMessage.Sticker.SetName})
					if err != nil {
						logger.Warn().
							Err(err).
							Str("setName", targetMessage.Sticker.SetName).
							Msg("Failed to get sticker set info, save it as a custom sticker")
					}
					if stickerSet != nil {
						// 属于一个贴纸包中的贴纸
						UserSavedMessage.Item.Sticker = append(UserSavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
							ID:          fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
							FileID:      targetMessage.Sticker.FileID,
							SetName:     stickerSet.Name,
							SetTitle:    stickerSet.Title,
							Emoji:       targetMessage.Sticker.Emoji,
							OriginInfo:  originInfo,
						})
					} else {
						// 有贴纸信息，但是对应的贴纸包已经删掉了
						UserSavedMessage.Item.Sticker = append(UserSavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
							ID:          fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
							FileID:      targetMessage.Sticker.FileID,
							Emoji:       targetMessage.Sticker.Emoji,
							OriginInfo:  originInfo,
						})
					}
				} else {
					UserSavedMessage.Item.Sticker = append(UserSavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
						ID:          fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
						FileID:      targetMessage.Sticker.FileID,
						Emoji:       targetMessage.Sticker.Emoji,
						OriginInfo:  originInfo,
					})
				}
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				messageParams.Text = "已保存贴纸"
			}
		case replyMsgType.Video:
			for _, n := range UserSavedMessage.Item.Video {
				if n.FileID == targetMessage.Video.FileID {
					isSaved = true
					messageParams.Text = "已保存过该视频\n"
					break
				}
			}
			if !isSaved {
				videoTitle := targetMessage.Video.FileName
				if videoTitle == "" {
					videoTitle = "video.mp4"
				}
				UserSavedMessage.Item.Video = append(UserSavedMessage.Item.Video, SavedMessageTypeCachedVideo{
					ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID:          targetMessage.Video.FileID,
					Title:           videoTitle,
					Caption:         targetMessage.Caption,
					CaptionEntities: pendingEntitites,
					OriginInfo:      originInfo,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				messageParams.Text = "已保存视频"
			}
		case replyMsgType.VideoNote:
			for _, n := range UserSavedMessage.Item.VideoNote {
				if n.FileID == targetMessage.VideoNote.FileID {
					isSaved = true
					messageParams.Text = "已保存过该圆形视频\n"
					break
				}
			}
			if !isSaved {
				UserSavedMessage.Item.VideoNote = append(UserSavedMessage.Item.VideoNote, SavedMessageTypeCachedVideoNote{
					ID:          fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID:      targetMessage.VideoNote.FileID,
					Title:       targetMessage.VideoNote.FileUniqueID,
					OriginInfo:  originInfo,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				messageParams.Text = "已保存圆形视频"
			}
		case replyMsgType.Voice:
			for _, n := range UserSavedMessage.Item.Voice {
				if n.FileID == targetMessage.Voice.FileID {
					isSaved = true
					messageParams.Text = "已保存过该语音\n"
					break
				}
			}
			if !isSaved {
				UserSavedMessage.Item.Voice = append(UserSavedMessage.Item.Voice, SavedMessageTypeCachedVoice{
					ID:              fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID:          targetMessage.Voice.FileID,
					Title:           targetMessage.Voice.MimeType,
					Caption:         targetMessage.Caption,
					CaptionEntities: pendingEntitites,
					OriginInfo:      originInfo,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				messageParams.Text = "已保存语音"
			}
		default:
			messageParams.Text = "暂不支持的消息类型"
	}

	if !isSaved {
		SavedMessageSet[opts.ChatInfo.ID] = UserSavedMessage
		err := SaveSavedMessageList(opts.Ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Str("messageType", replyMsgType.Str()).
				Msg("Failed to save savedmessage list after save a item")
			handlerErr.Addf("failed to save savedmessage list after save a item: %w", err)

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.ChatInfo.ID,
				ParseMode:       models.ParseModeHTML,
				Text:            fmt.Sprintf("保存内容时保存收藏列表数据库失败，请稍后再试或联系机器人管理员\n<blockquote expandable>%s<expandable>", err.Error()),
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("messageType", replyMsgType.Str()).
					Str("content", "failed to save savedmessage list notice").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "failed to save savedmessage list notice", err)
			}
			return handlerErr.Flat()
		}
	}

	_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
	if err != nil {
		logger.Error().
			Err(err).
			Str("messageType", replyMsgType.Str()).
			Str("content", "message saved notice").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "message saved notice", err)
	}
	return handlerErr.Flat()
}

func InlineShowSavedMessageHandler(opts *handler_params.InlineQuery) error {
	var handlerErr flaterr.MultErr
	var InlineSavedMessageResultList []models.InlineQueryResult
	var button *models.InlineQueryResultsButton

	SavedMessage := SavedMessageSet[opts.ChatInfo.ID]

	keywordFields := inline_utils.ExtractKeywords(opts.Fields)

	if len(keywordFields) == 0 {
		// 没有搜索关键词，返回所有消息
		var all []models.InlineQueryResult
		for _, n := range SavedMessage.Item.All() {
			if n.onlyText != nil {
				all = append(all, n.onlyText)
			} else if n.audio != nil {
				all = append(all, n.audio)
			} else if n.document != nil {
				all = append(all, n.document)
			} else if n.gif != nil {
				all = append(all, n.gif)
			} else if n.photo != nil {
				all = append(all, n.photo)
			} else if n.sticker != nil {
				all = append(all, n.sticker)
			} else if n.video != nil {
				all = append(all, n.video)
			} else if n.videoNote != nil {
				all = append(all, n.videoNote)
			} else if n.voice != nil {
				all = append(all, n.voice)
			} else if n.mpeg4gif != nil {
				all = append(all, n.mpeg4gif)
			}
		}
		InlineSavedMessageResultList = all
	} else {
		// 有搜索关键词，返回匹配的消息
		var all []models.InlineQueryResult
		for _, n := range SavedMessage.Item.All() {
			if n.onlyText != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.onlyText.Description, n.onlyText.Title}) {
				all = append(all, n.onlyText)
			} else if n.audio != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.audio.Caption, n.sharedData.Description, n.sharedData.Title, n.sharedData.FileName}) {
				all = append(all, n.audio)
			} else if n.document != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.document.Title, n.document.Caption, n.document.Description}) {
				all = append(all, n.document)
			} else if n.gif != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.gif.Title, n.gif.Caption, n.sharedData.Description}) {
				all = append(all, n.gif)
			} else if n.mpeg4gif != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.mpeg4gif.Title, n.mpeg4gif.Caption, n.sharedData.Description}) {
				all = append(all, n.mpeg4gif)
			} else if n.photo != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.photo.Title, n.photo.Caption, n.photo.Description}) {
				all = append(all, n.photo)
			} else if n.sticker != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.sharedData.Title, n.sharedData.Name, n.sharedData.Description, n.sharedData.FileName}) {
				all = append(all, n.sticker)
			} else if n.video != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.video.Title, n.video.Caption, n.video.Description}) {
				all = append(all, n.video)
			} else if n.videoNote != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.videoNote.Title, n.videoNote.Caption, n.videoNote.Description}) {
				all = append(all, n.videoNote)
			} else if n.voice != nil && inline_utils.MatchMultKeyword(keywordFields, []string{n.voice.Title, n.voice.Caption, n.sharedData.Description}) {
				all = append(all, n.voice)
			}
		}
		InlineSavedMessageResultList = all

		// 如果没有匹配的内容，则返回一个提示信息
		if len(InlineSavedMessageResultList) == 0 {
			InlineSavedMessageResultList = append(InlineSavedMessageResultList, &models.InlineQueryResultArticle{
				ID:                  "none",
				Title:               "没有符合关键词的内容",
				Description:         fmt.Sprintf("没有找到包含 %s 的内容", keywordFields),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
					ParseMode:   models.ParseModeMarkdownV1,
				},
			})
		}
	}

	if len(InlineSavedMessageResultList) == 0 {
		InlineSavedMessageResultList = append(InlineSavedMessageResultList, &models.InlineQueryResultArticle{
			ID:          "empty",
			Title:       "没有保存内容（点击查看详细教程）",
			Description: "对一条信息回复 /save 来保存它",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: fmt.Sprintf("您可以在任何聊天的输入栏中输入 <code>@%s +saved </code>来查看您的收藏\n若要添加，您需要确保机器人可以读取到您的指令，例如在群组中需要添加机器人，或点击 @%s 进入与机器人的聊天窗口，找到想要收藏的信息，然后对着那条信息回复 /save 即可\n若收藏成功，机器人会回复您并提示收藏成功，您也可以手动发送一条想要收藏的息，再使用 /save 命令回复它", consts.BotMe.Username, consts.BotMe.Username),
				ParseMode:   models.ParseModeHTML,
			},
		})
		button = &models.InlineQueryResultsButton{
			Text:           "点击此处快速跳转到机器人",
			StartParameter: "via-inline_noreply",
		}
	}

	_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: opts.InlineQuery.ID,
		Results:       inline_utils.ResultPagination(opts.Fields, InlineSavedMessageResultList),
		IsPersonal:    true,
		CacheTime:     0,
		Button:        button,
	})
	if err != nil {
		zerolog.Ctx(opts.Ctx).Error().
			Err(err).
			Str("pluginName", "Saved Message").
			Str("funcName", "InlineShowSavedMessageHandler").
			Dict(utils.GetUserDict(opts.InlineQuery.From)).
			Str("query", opts.InlineQuery.Query).
			Str("content", "saved message result").
			Msg(flaterr.AnswerInlineQuery.Str())
		handlerErr.Addt(flaterr.AnswerInlineQuery, "saved message result", err)
	}

	return handlerErr.Flat()
}

func SendPrivacyPolicy(opts *handler_params.Message) error {
	var handlerErr flaterr.MultErr

	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Message.Chat.ID,
		Text:   "目前此机器人仍在开发阶段中，此信息可能会有更改\n" +
			"<blockquote expandable>本机器人提供收藏信息功能，您可以在回复一条信息时输入 /save 来收藏它，之后在 inline 模式下随时浏览您的收藏内容并发送\n\n" +

			"我们会记录哪些数据？\n" +
			"1. 您的用户信息，例如 用户昵称、用户 ID、聊天类型（当您将此机器人添加到群组或频道中时）\n" +
			"2. 您的使用情况，例如 消息计数、inline 调用计数、inline 条目计数、最后向机器人发送的消息、callback_query、inline_query 以及选择的 inline 结果\n" +
			"3. 收藏信息内容，您需要注意这个，因为您是为了这个而阅读此内容，例如 存储的收藏信息数量、其图片上传到 Telegram 时的文件 ID、图片下方的文本，还有您在使用添加命令时所自定义的搜索关键词" +
			"\n\n" +

			"我的数据安全吗？\n" +
			"这是一个早期的项目，还有很多未发现的 bug 与漏洞，因此您不能也不应该将敏感的数据存储在此机器人中，若您觉得我们收集的信息不妥，您可以不点击底部的同意按钮，我们仅会收集一些基本的信息，防止对机器人造成滥用，基本信息为前一段的 1 至 2 条目" +
			"\n\n" +

			"我收藏的消息，有谁可以看到?\n" +
			"此功能被设计为每个人有单独的存储空间，如果您不手动从 inline 模式下选择信息并发送，其他用户是没法查看您的收藏列表的。不过，与上一个条目一样，为了防止滥用，我们是可以也有权利查看您收藏的内容的，请不要在其中保存隐私数据" +
			"</blockquote>" +

			"\n\n" +
			"内容待补充...",
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击同意以上内容",
			URL:  fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy_agree", consts.BotMe.Username),
		}}}},
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		zerolog.Ctx(opts.Ctx).Error().
			Err(err).
			Str("pluginName", "Saved Message").
			Str("funcName", "SendPrivacyPolicy").
			Dict(utils.GetUserDict(opts.Message.From)).
			Str("content", "saved message privacy policy").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "saved message privacy policy", err)
	}

	return handlerErr.Flat()
}

func AgreePrivacyPolicy(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "AgreePrivacyPolicy").
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	var UserSavedMessage SavedMessage
	UserSavedMessage.AgreePrivacyPolicy = true
	SavedMessageSet[opts.ChatInfo.ID] = UserSavedMessage

	err := SaveSavedMessageList(opts.Ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("failed to save savemessage list after user agree privacy policy")
		handlerErr.Addf("failed to save savemessage list after user agree privacy policy: %w", err)

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Message.Chat.ID,
			Text:      fmt.Sprintf("保存收藏列表数据库失败，请稍后再试或联系机器人管理员\n<blockquote expandable>%s<expandable>", err.Error()),
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "failed to save savedmessage list notice").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "failed to save savedmessage list notice", err)
		}
	} else {
		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.Chat.ID,
			Text:            "您已成功开启收藏信息功能，回复一条信息的时候发送 /save 来使用收藏功能吧！\n由于服务器性能原因，每个人的收藏数量上限默认为 100 个，您也可以从机器人的个人信息中寻找管理员来申请更高的上限\n点击下方按钮来浏览您的收藏内容",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text:                         "点击浏览您的收藏",
				SwitchInlineQueryCurrentChat: configs.BotConfig.InlineSubCommandSymbol + "saved ",
			}}}},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "saved message function enabled").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "saved message function enabled", err)
		} else {
			buildSavedMessageByMessageHandlers()
		}
	}

	return handlerErr.Flat()
}

func Init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "Saved Message",
		Func: ReadSavedMessageList,
	})
	// ReadSavedMessageList()
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Saved Message",
		Saver:  SaveSavedMessageList,
		Loader: ReadSavedMessageList,
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "save",
		MessageHandler: saveMessageHandler,
	})
	plugin_utils.AddInlineManualHandlerHandlers(plugin_utils.InlineManualHandler{
		Command:       "saved",
		InlineHandler: InlineShowSavedMessageHandler,
		Description:   "显示自己保存的消息",
	})
	plugin_utils.AddSlashStartCommandHandlers([]plugin_utils.SlashStartHandler{
		{
			Argument: "savedmessage_privacy_policy",
			MessageHandler:  SendPrivacyPolicy,
		},
		{
			Argument: "savedmessage_privacy_policy_agree",
			MessageHandler:  AgreePrivacyPolicy,
		},
		// {
		// 	Argument: "savedmessage_channel_privacy_policy",
		// 	Handler:  SendPrivacyPolicy,
		// },
		// {
		// 	Argument: "savedmessage_channel_privacy_policy_agree",
		// 	Handler:  AgreePrivacyPolicy,
		// },
	}...)
	plugin_utils.AddSlashStartWithPrefixCommandHandlers(plugin_utils.SlashStartWithPrefixHandler{
		Prefix:         "via-inline",
		Argument:       "savedmessage-help",
		MessageHandler: saveMessageHandler,
	})
	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "收藏消息",
		Description: "此功能可以收藏用户指定的消息，之后使用 inline 模式查看并发送保存的内容\n\n保存消息：\n向机器人发送要保存的消息，然后使用 <code>/save 关键词</code> 命令回复要保存的消息，关键词可以忽略。若机器人在群组中，也可以直接使用 <code>/save 关键词</code> 命令回复要保存的消息。\n\n发送保存的消息：点击下方的按钮来使用 inline 模式，当您多次在 inline 模式下使用此 bot 时，在输入框中输入 <code>@</code> 即可看到 bot 会出现在列表中",
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
			{{
				Text:                         "点击浏览您的收藏",
				SwitchInlineQueryCurrentChat: configs.BotConfig.InlineSubCommandSymbol + "saved ",
			}},
			{{
				Text:         "将此功能设定为您的默认 inline 命令",
				CallbackData: "inline_default_noedit_saved",
			}},
			{
				{
					Text:         "返回",
					CallbackData: "help",
				},
				{
					Text:         "关闭",
					CallbackData: "help-close",
				},
			},
		}},
	})
}
