package personal

import (
	"context"
	"fmt"
	"trbot/plugins/saved_message/common"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/inline_utils"
	"trbot/utils/meilisearch_utils"
	"trbot/utils/origin_info"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
	"github.com/rs/zerolog"
)


func saveMessageHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	var handlerErr flaterr.MultErr

	if common.MeilisearchClient == nil {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.Chat.ID,
			Text:            "此功能不可用，因为 Meilisearch 服务尚未初始化",
			ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "meilisearch client uninitialized").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "meilisearch client uninitialized", err)
		}
	} else {
		user := common.SavedMessageList.GetUser(opts.Message.From.ID)
		if user == nil {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            "此功能需要保存一些信息才能正常工作，在使用这个功能前，请先阅读一下我们会保存哪些信息",
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击查看",
					URL:  fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy", configs.BotMe.Username),
				}}}},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "need agree privacy policy").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "need agree privacy policy", err)
			}
		} else {
			messageParams := &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				ParseMode:       models.ParseModeHTML,
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
					{
						Text:                         "点击浏览您的收藏",
						SwitchInlineQueryCurrentChat: configs.BotConfig.InlineSubCommandSymbol + "usersaved ",
					},
					{
						Text:         "关闭",
						CallbackData: "delete_this_message",
					},
				}}},
			}

			if opts.Message.ReplyToMessage == nil {
				messageParams.Text = "在回复一条消息的同时发送 <code>/save</code> 来收藏消息"
				if opts.Message.Chat.Type == models.ChatTypePrivate {
					messageParams.ReplyMarkup = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text:         "修改此功能偏好",
						CallbackData: "savedmsg_switch",
					}}}}
				}
			} else {
				if user.Limit == 0 && user.Count == 0 {
					// 每个用户初次添加时，默认限制 100 条
					user.Limit = 100
				}

				// 若不是初次添加，为 0 就是不限制
				if user.Limit != 0 && user.Count >= user.Limit {
					messageParams.Text = "已达到限制，无法保存更多内容"
				} else {
					msgData := meilisearch_utils.BuildMessageData(opts.Ctx, opts.Thebot, opts.Message.ReplyToMessage)

					var messageLength int

					if opts.Message.ReplyToMessage.Caption != "" {
						messageLength = common.UTF16Length(opts.Message.ReplyToMessage.Caption)
						msgData.Entities = opts.Message.ReplyToMessage.CaptionEntities
					} else if opts.Message.ReplyToMessage.Text != "" {
						messageLength = common.UTF16Length(opts.Message.ReplyToMessage.Text)
						msgData.Entities = opts.Message.ReplyToMessage.Entities
					}

					// 若字符长度大于设定的阈值，添加折叠样式引用再保存，如果是全文引用但不折叠，改成折叠样式
					if messageLength > common.TextExpandableLength && (len(msgData.Entities) == 0 || msgData.Entities[0].Type == models.MessageEntityTypeBlockquote && msgData.Entities[0].Length == messageLength) {
						msgData.Entities = []models.MessageEntity{{
							Type:   models.MessageEntityTypeExpandableBlockquote,
							Offset: 0,
							Length: messageLength,
						}}
					}

					msgData.ID = user.SavedTimes

					if !user.DropOriginInfo {
						msgData.OriginInfo = origin_info.GetOriginInfo(opts.Message.ReplyToMessage)
					}

					// 获取使用命令保存时设定的描述
					if len(opts.Message.Text) > len(opts.Fields[0])+1 {
						msgData.Desc = opts.Message.Text[len(opts.Fields[0])+1:]
					}

					_, err := common.MeilisearchClient.Index(user.IDStr()).AddDocuments(msgData)
					if err != nil {
						logger.Error().
							Err(err).
							Msg("Failed to send add documents task to meilisearch")
						handlerErr.Addf("failed to send add documents task to meilisearch: %w", err)
						messageParams.Text = "保存失败，可能是因为 Meilisearch 服务不可用或网络问题，请稍后再试"
						messageParams.ReplyMarkup = nil
					} else {
						user.Count++
						user.SavedTimes++
						err = common.SaveSavedMessageList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Msg("Failed to save savedmessage list after save a item")
							handlerErr.Addf("failed to save savedmessage list after save a item: %w", err)
							messageParams.Text = "消息已保存，但保存统计数据失败"
							messageParams.ReplyMarkup = nil
						} else {
							messageParams.Text = "已保存内容"
						}
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

func saveMessageFromCallbackQueryHandler(opts *handler_params.Message) error {
	if opts.Message == nil {
		return nil
	}

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if common.MeilisearchClient == nil {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.Chat.ID,
			Text:            "此功能不可用，因为 Meilisearch 服务尚未初始化",
			ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "meilisearch client uninitialized").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "meilisearch client uninitialized", err)
		}
	} else {
		user := common.SavedMessageList.GetUser(opts.Message.From.ID)
		if user == nil {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            "此功能需要保存一些信息才能正常工作，在使用这个功能前，请先阅读一下我们会保存哪些信息",
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击查看",
					URL:  fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy", configs.BotMe.Username),
				}}}},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "need agree privacy policy").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "need agree privacy policy", err)
			}
		} else {
			messageParams := &bot.SendMessageParams{
				ChatID:          opts.ChatInfo.ID,
				ParseMode:       models.ParseModeHTML,
				ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text:         "关闭",
					CallbackData: "delete_this_message",
				}}}},
			}

			if user.Limit == 0 && user.Count == 0 {
				// 每个用户初次添加时，默认限制 100 条
				user.Limit = 100
			}

			// 若不是初次添加，为 0 就是不限制
			if user.Limit != 0 && user.Count >= user.Limit {
				messageParams.Text = "已达到限制，无法保存更多内容"
			} else {
				msgData := meilisearch_utils.BuildMessageData(opts.Ctx, opts.Thebot, opts.Message)

				var messageLength int

				if opts.Message.Caption != "" {
					messageLength = common.UTF16Length(opts.Message.Caption)
					msgData.Entities = opts.Message.CaptionEntities
				} else if opts.Message.Text != "" {
					messageLength = common.UTF16Length(opts.Message.Text)
					msgData.Entities = opts.Message.Entities
				}

				// 若字符长度大于设定的阈值，添加折叠样式引用再保存，如果是全文引用但不折叠，改成折叠样式
				if messageLength > common.TextExpandableLength && (len(msgData.Entities) == 0 || msgData.Entities[0].Type == models.MessageEntityTypeBlockquote && msgData.Entities[0].Length == messageLength) {
					msgData.Entities = []models.MessageEntity{{
						Type:   models.MessageEntityTypeExpandableBlockquote,
						Offset: 0,
						Length: messageLength,
					}}
				}

				msgData.ID = user.SavedTimes

				if !user.DropOriginInfo {
					msgData.OriginInfo = origin_info.GetOriginInfo(opts.Message)
				}

				_, err := common.MeilisearchClient.Index(user.IDStr()).AddDocuments(msgData)
				if err != nil {
					logger.Error().
						Err(err).
						Msg("Failed to send add documents task to meilisearch")
					handlerErr.Addf("failed to send add documents task to meilisearch: %w", err)
					messageParams.Text = "保存失败，可能是因为 Meilisearch 服务不可用或网络问题，请稍后再试"
					messageParams.ReplyMarkup = nil
				} else {
					user.Count++
					user.SavedTimes++
					err = common.SaveSavedMessageList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Msg("Failed to save savedmessage list after save a item")
						handlerErr.Addf("failed to save savedmessage list after save a item: %w", err)
						messageParams.Text = "消息已保存，但保存统计数据失败"
						messageParams.ReplyMarkup = nil
					} else {
						messageParams.Text = "已保存内容"
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

func userInlineHandler(opts *handler_params.InlineQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.InlineQuery.From)).
		Str("query", opts.InlineQuery.Query).
		Logger()
	var handlerErr flaterr.MultErr

	var resultList []models.InlineQueryResult

	if common.MeilisearchClient == nil {
		resultList = append(resultList, &models.InlineQueryResultArticle{
			ID:                  "error",
			Title:               "此功能不可用",
			Description:         "Meilisearch 服务尚未初始化",
			InputMessageContent: &models.InputTextMessageContent{ MessageText: "Meilisearch 服务尚未初始化，无法使用收藏功能" },
		})
	} else {
		parsedQuery := inline_utils.ParseInlineFields(opts.Fields)

		// 单字符显示提示信息
		if parsedQuery.LastChar != "" {
			switch parsedQuery.LastChar {
			case configs.BotConfig.InlineCategorySymbol:
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.InlineQuery.ID,
					Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
						ID:          "keepInputCategory",
						Title:       "请继续输入分类名称",
						Description: fmt.Sprintf("当前列表有 %s 分类", common.ResultCategorys.StrList()),
						InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在尝试选择分类时点击了提示信息..." },
					}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "need a category name").
						Msg(flaterr.AnswerInlineQuery.Str())
					handlerErr.Addt(flaterr.AnswerInlineQuery, "need a category name", err)
				}
				return handlerErr.Flat()
			case configs.BotConfig.InlinePaginationSymbol:
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.InlineQuery.ID,
					Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
						ID:          "keepInputNumber",
						Title:       "请继续输入页码",
						Description: "请手动输入页码来尝试可用的页面",
						InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在尝试输入页码时点击了提示信息..." },
					}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "need a category name").
						Msg(flaterr.AnswerInlineQuery.Str())
					handlerErr.Addt(flaterr.AnswerInlineQuery, "need a category name", err)
				}

				return handlerErr.Flat()
			}
		}

		if parsedQuery.Page < 1 {
			_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.InlineQuery.ID,
				Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:          "wrongPageNumber",
					Title:       "错误的页码",
					Description: fmt.Sprintf("您输入的页码 %d 为负数", parsedQuery.Page),
					InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在浏览不存在的页面时点击了错误页码提示..." },
				}},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "invalid page number").
					Msg(flaterr.AnswerInlineQuery.Str())
				handlerErr.Addt(flaterr.AnswerInlineQuery, "invalid page number", err)
			}
			return handlerErr.Flat()
		}

		var filter string
		if parsedQuery.Category != "" {
			category, isExist := common.ResultCategorys.GetCategory(parsedQuery.Category)
			if !isExist {
				_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
					InlineQueryID: opts.InlineQuery.ID,
					Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
						ID:          "noThisCategory",
						Title:       fmt.Sprintf("无效的 [ %s ] 分类", parsedQuery.Category),
						Description: fmt.Sprintf("当前列表有 %s 分类", common.ResultCategorys.StrList()),
						InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在尝试访问不存在的分类时点击了提示信息..." },
					}},
					IsPersonal: true,
					CacheTime:  0,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "invalid category name").
						Msg(flaterr.AnswerInlineQuery.Str())
					handlerErr.Addt(flaterr.AnswerInlineQuery, "invalid category name", err)
				}
				return handlerErr.Flat()
			}
			filter = "type=" + category.Str()
		}

		user := common.SavedMessageList.GetUser(opts.InlineQuery.From.ID)
		if user == nil {
			_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
				InlineQueryID: opts.InlineQuery.ID,
				Results: []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:          "noInitYet",
					Title:       "您尚未初始化此功能",
					Description: "对一条信息回复 /save 来保存它",
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: fmt.Sprintf("您可以在任何聊天的输入栏中输入 <code>@%s +usersaved </code>来查看您的收藏\n若要添加，您需要确保机器人可以读取到您的指令，例如在群组中需要添加机器人，或点击 @%s 进入与机器人的聊天窗口，找到想要收藏的信息，然后对着那条信息回复 /save 即可\n若收藏成功，机器人会回复您并提示收藏成功，您也可以手动发送一条想要收藏的息，再使用 /save 命令回复它", configs.BotMe.Username, configs.BotMe.Username),
						ParseMode:   models.ParseModeHTML,
					},
				}},
				Button: &models.InlineQueryResultsButton{
					Text:           "点击此处快速跳转到机器人",
					StartParameter: "via-inline_noreply",
				},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "user not init yet").
					Msg(flaterr.AnswerInlineQuery.Str())
				handlerErr.Addt(flaterr.AnswerInlineQuery, "user not init yet", err)
			}
			return handlerErr.Flat()
		}

		datas, err := common.MeilisearchClient.Index(user.IDStr()).Search(parsedQuery.KeywordQuery(), &meilisearch.SearchRequest{
			HitsPerPage: int64(configs.BotConfig.InlineResultsPerPage - 1),
			Page:        int64(parsedQuery.Page),
			Filter:      filter,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to get user saved message")
			handlerErr.Addf("failed to get user saved message: %w", err)
			resultList = append(resultList, &models.InlineQueryResultArticle{
				ID:                  "error",
				Title:               "获取消息发生错误",
				Description:         "点击查看错误信息",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: fmt.Sprintf("获取消息时发生了错误：<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
					ParseMode:   models.ParseModeHTML,
				},
			})
		} else if int64(parsedQuery.Page) > datas.TotalPages {
			if len(parsedQuery.Keywords) > 0 {
				resultList = append(resultList, &models.InlineQueryResultArticle{
					ID:                  "noMatchItem",
					Title:               fmt.Sprintf("没有符合 %s 关键词的内容", parsedQuery.Keywords),
					Description:         "试试其他关键词？",
					InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在找不到想看的东西时无奈点击了提示信息..." },
				})
			} else if parsedQuery.Page > 1 {
				resultList = append(resultList, &models.InlineQueryResultArticle{
					ID:                  "noMorePage",
					Title:               "没有更多消息了",
					InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在找不到想看的东西时无奈点击了提示信息..." },
				})
			} else {
				resultList = append(resultList, &models.InlineQueryResultArticle{
					ID:                  "noItemInChannel",
					Title:               "什么都没有",
					Description:         "您还没有保存任何内容",
					InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在找不到想看的东西时无奈点击了提示信息..." },
				})
			}
		} else {
			var msgDatas []meilisearch_utils.MessageData
			err = meilisearch_utils.UnmarshalMessageData(&datas.Hits, &msgDatas)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to unmarshal user saved message data")
				handlerErr.Addf("Failed to unmarshal user saved message data: %w", err)
				resultList = append(resultList, &models.InlineQueryResultArticle{
					ID:                  "error",
					Title:               "解析消息发生错误",
					Description:         "点击查看错误信息",
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: fmt.Sprintf("解析消息时发生了错误：<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
						ParseMode:   models.ParseModeHTML,
					},
				})
			} else {
				for _, msgData := range msgDatas {
					switch msgData.Type {
					case message_utils.Text:
						resultList = append(resultList, &models.InlineQueryResultArticle{
							ID:                  msgData.MsgIDStr(),
							Title:               msgData.Text,
							Description:         msgData.Desc,
							// ReplyMarkup:         msgData.OriginInfo.BuildButton(),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText:        msgData.Text,
								Entities:           msgData.Entities,
								LinkPreviewOptions: msgData.LinkPreviewOptions,
							},
						})
					case message_utils.Audio:
						resultList = append(resultList, &models.InlineQueryResultCachedAudio{
							ID:              msgData.MsgIDStr(),
							AudioFileID:     msgData.FileID,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							// ReplyMarkup:     msgData.OriginInfo.BuildButton(),
						})
					case message_utils.Document:
						resultList = append(resultList, &models.InlineQueryResultCachedDocument{
							ID:              msgData.MsgIDStr(),
							DocumentFileID:  msgData.FileID,
							Title:           msgData.FileName,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							Description:     msgData.Desc,
							// ReplyMarkup:     msgData.OriginInfo.BuildButton(),
						})
					case message_utils.Animation:
						resultList = append(resultList, &models.InlineQueryResultCachedMpeg4Gif{
							ID:              msgData.MsgIDStr(),
							Mpeg4FileID:     msgData.FileID,
							Title:           msgData.FileName,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							// ReplyMarkup:     msgData.OriginInfo.BuildButton(),
						})
					case message_utils.Photo:
						resultList = append(resultList, &models.InlineQueryResultCachedPhoto{
							ID:                    msgData.MsgIDStr(),
							PhotoFileID:           msgData.FileID,
							Caption:               msgData.Text,
							CaptionEntities:       msgData.Entities,
							Description:           msgData.Desc,
							ShowCaptionAboveMedia: msgData.ShowCaptionAboveMedia,
							// ReplyMarkup:           msgData.OriginInfo.BuildButton(),
						})
					case message_utils.Sticker:
						resultList = append(resultList, &models.InlineQueryResultCachedSticker{
							ID:            msgData.MsgIDStr(),
							StickerFileID: msgData.FileID,
							// ReplyMarkup:   msgData.OriginInfo.BuildButton(),
						})
					case message_utils.Video:
						resultList = append(resultList, &models.InlineQueryResultCachedVideo{
							ID:              msgData.MsgIDStr(),
							VideoFileID:     msgData.FileID,
							Title:           msgData.FileName,
							Description:     msgData.Desc,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							// ReplyMarkup:     msgData.OriginInfo.BuildButton(),
						})
					case message_utils.VideoNote:
						resultList = append(resultList, &models.InlineQueryResultCachedDocument{
							ID:              msgData.MsgIDStr(),
							DocumentFileID:  msgData.FileID,
							Title:           msgData.FileName,
							Description:     msgData.Desc,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							// ReplyMarkup:     msgData.OriginInfo.BuildButton(),
						})
					case message_utils.Voice:
						resultList = append(resultList, &models.InlineQueryResultCachedVoice{
							ID:              msgData.MsgIDStr(),
							VoiceFileID:     msgData.FileID,
							Title:           msgData.FileTitle,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							// ReplyMarkup:     msgData.OriginInfo.BuildButton(),
						})
					}
				}
				if datas.TotalPages != int64(parsedQuery.Page) && datas.TotalHits > int64(configs.BotConfig.InlineResultsPerPage - 1) {
					resultList = append(resultList, &models.InlineQueryResultArticle{
						ID:          "paginationPage",
						Title:       fmt.Sprintf("当前您在第 %d 页", parsedQuery.Page),
						Description: fmt.Sprintf("后面还有 %d 页内容，输入 %s%d 查看下一页", datas.TotalPages - int64(parsedQuery.Page), configs.BotConfig.InlinePaginationSymbol, parsedQuery.Page + 1),
						InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在挑选内容时点击了分页提示..." },
					})
				}
			}
		}
	}

	_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: opts.InlineQuery.ID,
		Results:       resultList,
		// Button:        &models.InlineQueryResultsButton{
		// 	Text:           "当前为个人收藏内容",
		// 	StartParameter: "savedmessage_viewchannel",
		// },
		IsPersonal:    true,
		CacheTime:     0,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("content", "user saved message result").
			Msg(flaterr.AnswerInlineQuery.Str())
		handlerErr.Addt(flaterr.AnswerInlineQuery, "user saved message result", err)
	}

	return handlerErr.Flat()
}

func sendPrivacyPolicy(opts *handler_params.Message) error {
	var handlerErr flaterr.MultErr

	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Message.Chat.ID,
		Text: "目前此机器人仍在开发阶段中，此信息可能会有更改\n" +
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
		ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击同意以上内容",
			URL:  fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy_agree", configs.BotMe.Username),
		}}}},
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		zerolog.Ctx(opts.Ctx).Error().
			Err(err).
			Str("pluginName", "Saved Message").
			Str(utils.GetCurrentFuncName()).
			Dict(utils.GetUserDict(opts.Message.From)).
			Str("content", "saved message privacy policy").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "saved message privacy policy", err)
	}

	return handlerErr.Flat()
}

func agreePrivacyPolicy(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if common.SavedMessageList.GetUser(opts.Message.From.ID) != nil {
		return nil
	}

	var user common.SavedMessageUser
	user.UserID = opts.Message.From.ID
	common.SavedMessageList.User = append(common.SavedMessageList.User, user)
	err := meilisearch_utils.CreateChatIndex(opts.Ctx, &common.MeilisearchClient, user.IDStr())
	if err != nil {
		logger.Error().
			Err(err).
			Msg("failed to create chat index for saved message user")
		handlerErr.Addf("failed to create chat index for saved message user: %w", err)
		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Message.Chat.ID,
			Text:      fmt.Sprintf("创建收藏信息索引失败，请稍后再试或联系机器人管理员:\n<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "failed to create chat index notice").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "failed to create chat index notice", err)
		}
		return handlerErr.Flat()
	}

	err = common.SaveSavedMessageList(opts.Ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("failed to save savemessage list after user agree privacy policy")
		handlerErr.Addf("failed to save savemessage list after user agree privacy policy: %w", err)

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Message.Chat.ID,
			Text:      fmt.Sprintf("保存收藏列表数据库失败，请稍后再试或联系机器人管理员\n<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "failed to save savedmessage list notice").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "failed to save savedmessage list notice", err)
		}
		return handlerErr.Flat()
	}

	buildSavedMessageByMessageHandlers()

	_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:          opts.Message.Chat.ID,
		Text:            "您已成功开启收藏信息功能，回复一条信息的时候发送 /save 来使用收藏功能吧！\n由于服务器性能原因，每个人的收藏数量上限默认为 100 个，您也可以从机器人的个人信息中寻找管理员来申请更高的上限\n点击下方按钮来浏览您的收藏内容",
		ReplyParameters: &models.ReplyParameters{MessageID: opts.Message.ID},
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text:                         "点击浏览您的收藏",
			SwitchInlineQueryCurrentChat: configs.BotConfig.InlineSubCommandSymbol + "usersaved ",
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

	return handlerErr.Flat()
}

func configKeyboardCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Str("callbackData", opts.CallbackQuery.Data).
		Logger()

	var handlerErr flaterr.MultErr
	var needSave bool = true

	user := common.SavedMessageList.GetUser(opts.CallbackQuery.From.ID)

	switch opts.CallbackQuery.Data {
	case "savedmsg_switch_use_quick_save":
		user.UseQuickSave = !user.UseQuickSave
		buildSavedMessageByMessageHandlers()
	case "savedmsg_switch_drop_origin_info":
		user.DropOriginInfo = !user.DropOriginInfo
	default:
		needSave = false
	}

	if needSave {
		err := common.SaveSavedMessageList(opts.Ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("failed to save savedmessage list after user config")
			handlerErr.Addf("failed to save savedmessage list after user config: %w", err)
			_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: opts.CallbackQuery.ID,
				Text:            "保存配置失败，请稍后再试或联系机器人管理员",
				ShowAlert:       true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "failed to save user config notice").
					Msg(flaterr.AnswerCallbackQuery.Str())
				handlerErr.Addt(flaterr.AnswerCallbackQuery, "failed to save user config notice", err)
			}
			return handlerErr.Flat()
		}
	}

	_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
		ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   opts.CallbackQuery.Message.Message.ID,
		Text:        "修改您的收藏偏好选项",
		ReplyMarkup: user.ConfigButtons(),
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("content", "config keyboard").
			Msg(flaterr.EditMessageText.Str())
		handlerErr.Addt(flaterr.EditMessageText, "config keyboard", err)
	}

	return handlerErr.Flat()

}

func buildSavedMessageByMessageHandlers() {
	msgTypeList := []message_utils.Type{
		message_utils.Text,
		message_utils.Audio,
		message_utils.Animation,
		message_utils.Document,
		message_utils.Photo,
		message_utils.Sticker,
		message_utils.Video,
		message_utils.VideoNote,
		message_utils.Voice,
	}

	for _, user := range common.SavedMessageList.User {
		for _, msgType := range msgTypeList {
			plugin_utils.RemoveHandlerByMessageTypeHandler(
				models.ChatTypePrivate,
				msgType,
				user.UserID,
				"保存消息到收藏夹",
			)
		}
	}
	for _, user := range common.SavedMessageList.User {
		if user.UseQuickSave {
			for _, msgType := range msgTypeList {
				plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
					PluginName:     "保存消息到收藏夹",
					ChatType:       models.ChatTypePrivate,
					ForChatID:      user.UserID,
					MessageType:    msgType,
					MessageHandler: saveMessageFromCallbackQueryHandler,
				})
			}
		}
	}
}

func InitUserPart(ctx context.Context) error {
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "save",
		MessageHandler: saveMessageHandler,
	})
	plugin_utils.AddInlineManualHandlers(plugin_utils.InlineManualHandler{
		Command:       "usersaved",
		InlineHandler: userInlineHandler,
		Description:   "显示自己保存的内容",
	})
	plugin_utils.AddSlashStartCommandHandlers([]plugin_utils.SlashStartHandler{
		{
			Argument:       "savedmessage_privacy_policy",
			MessageHandler: sendPrivacyPolicy,
		},
		{
			Argument:       "savedmessage_privacy_policy_agree",
			MessageHandler: agreePrivacyPolicy,
		},
	}...)
	plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
		CallbackDataPrefix:   "savedmsg_switch",
		CallbackQueryHandler: configKeyboardCallbackHandler,
	})

	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "收藏消息-个人",
		Description: "此功能可以收藏用户指定的消息，之后使用 inline 模式查看并发送保存的内容\n\n保存消息：\n向机器人发送要保存的消息，然后使用 <code>/save 关键词</code> 命令回复要保存的消息，关键词可以忽略。若机器人在群组中，也可以直接使用 <code>/save 关键词</code> 命令回复要保存的消息。\n\n发送保存的消息：\n点击下方的按钮来使用 inline 模式，当您多次在 inline 模式下使用此 bot 时，在输入框中输入 <code>@</code> 即可看到 bot 会出现在列表中",
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
			{{
				Text:                         "点击浏览您的收藏",
				SwitchInlineQueryCurrentChat: configs.BotConfig.InlineSubCommandSymbol + "usersaved ",
			}},
			{{
				Text:         "将此功能设定为您的默认 inline 命令",
				CallbackData: "inline_default_noedit_usersaved",
			}},
			{
				{
					Text:         "返回",
					CallbackData: "help",
				},
				{
					Text:         "关闭",
					CallbackData: "delete_this_message",
				},
			},
		}},
	})

	buildSavedMessageByMessageHandlers()
	return nil
}
