package channel

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"trle5.xyz/trbot/plugins/saved_message/common"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/flaterr"
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/inline_utils"
	"trle5.xyz/trbot/utils/meilisearch_utils"
	"trle5.xyz/trbot/utils/origin_info"
	"trle5.xyz/trbot/utils/plugin_utils"
	"trle5.xyz/trbot/utils/type/contain"
	"trle5.xyz/trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
	"github.com/rs/zerolog"
)

var channelPubliclink  string
var channelPrivateLink string
var stateMessageID     int
var editingMessageID   int // 用于编辑消息时的消息ID

// by chat ID handler
func channelSaveMessageHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	if common.MeilisearchClient == nil {
		logger.Warn().
			Int("messageID", opts.Message.ID).
			Msg("Meilisearch client is not initialized, skipping indexing")
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

		msgData.OriginInfo = origin_info.GetOriginInfo(opts.Message)

		taskinfo, err := common.MeilisearchClient.Index(common.SavedMessageList.ChannelIDStr()).AddDocuments(msgData)
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.Message.ID).
				Msg("failed to send add document request to Meilisearch")
			return fmt.Errorf("failed to send add document request to Meilisearch: %w", err)
		} else {
			task, err := meilisearch_utils.WaitForTask(opts.Ctx, &common.MeilisearchClient, taskinfo.TaskUID, time.Second * 1)
			if err != nil {
				return fmt.Errorf("wait for add document task failed: %w", err)
			} else if task.Status != meilisearch.TaskStatusSucceeded {
				return fmt.Errorf("failed to add document: %s", task.Error.Message)
			}
		}

		if common.SavedMessageList.NoticeChatID != 0 {
			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:              common.SavedMessageList.NoticeChatID,
				Text:                "已经在频道中保存了一个新消息，要为它添加关键词吗？",
				DisableNotification: true,
				ReplyMarkup:          buildMetadataKeyboard(msgData),
				ReplyParameters:     &models.ReplyParameters{
					MessageID: opts.Message.ID,
					ChatID:    common.SavedMessageList.ChannelID,
				},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int("messageID", opts.Message.ID).
					Str("content", "channel message saved").
					Msg(flaterr.SendMessage.Str())
				return fmt.Errorf(flaterr.SendMessage.Fmt(), "channel message saved", err)
			}
		}
	}

	return nil
}

// send link to bot to edit metadata
func channelMetadataHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	var handlerErr flaterr.MultErr

	var msgIDString string
	var msgData meilisearch_utils.MessageData

	indexManager := common.MeilisearchClient.Index(common.SavedMessageList.ChannelIDStr())

	if !contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...) {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Message.Chat.ID,
			Text:   "??",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.Message.ID).
				Str("content", "no permission to edit metadata").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "no permission to edit metadata", err)
		}
	} else {
		if len(opts.Fields) > 1 {
			msgIDString = opts.Fields[1]
		} else if strings.HasPrefix(opts.Message.Text, channelPubliclink) {
			msgIDString = strings.TrimPrefix(opts.Message.Text, channelPubliclink + "/")
		} else if strings.HasPrefix(opts.Message.Text, channelPrivateLink) {
			msgIDString = strings.TrimPrefix(opts.Message.Text, channelPrivateLink + "/")
		}

		if msgIDString != "" {
			err := indexManager.GetDocument(msgIDString, nil, &msgData)
			if err != nil {
				if err.(*meilisearch.Error).MeilisearchApiError.Code == "document_not_found" {
					_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Message.Chat.ID,
						Text:   "未找到该消息，请检查消息 ID 是否正确",
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						LinkPreviewOptions: &models.LinkPreviewOptions{
						},
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int("messageID", opts.Message.ID).
							Str("content", "document not found in index").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "document not found in index", err)
					}
				} else {
					logger.Error().
						Err(err).
						Int("messageID", opts.Message.ID).
						Msg("Failed to get message from index")
					handlerErr.Addf("failed to get message from index: %w", err)
					_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Message.Chat.ID,
						Text:   "获取消息失败，请稍后再试",
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						LinkPreviewOptions: &models.LinkPreviewOptions{
						},
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int("messageID", opts.Message.ID).
							Str("content", "failed to get message notice").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "failed to get message notice", err)
					}
				}
			} else {
				var pendingMessage string
				pendingMessage += fmt.Sprintf("ID: %d\n", msgData.ID)
				pendingMessage += fmt.Sprintf("类型: %s\n", msgData.Type)
				if msgData.FileID != "" {
					pendingMessage += fmt.Sprintf("文件 ID: %s\n", msgData.FileID)
				}
				if msgData.FileName != "" {
					pendingMessage += fmt.Sprintf("文件名: %s\n", msgData.FileName)
				}
				if msgData.FileTitle != "" {
					pendingMessage += fmt.Sprintf("文件标题: %s\n", msgData.FileTitle)
				}
				if msgData.Text != "" {
					pendingMessage += fmt.Sprintf("文本: <blockquote expandable>%s</blockquote>\n", msgData.Text)
				}
				if msgData.Desc != "" {
					pendingMessage += fmt.Sprintf("描述: <blockquote expandable>%s</blockquote>\n", msgData.Desc)
				}


				_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            pendingMessage,
					ParseMode:       models.ParseModeHTML,
					ReplyMarkup:     buildMetadataKeyboard(msgData),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int("messageID", opts.Message.ID).
						Str("content", "channel document info").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "channel document info", err)
				}
				logger.Info().
					Interface("msgData", msgData).
					Msg("Retrieved message data from index")
			}
		}
	}
	return handlerErr.Flat()
}

func buildMetadataKeyboard(msgData meilisearch_utils.MessageData) models.ReplyMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
		{
			{
				Text:         "修改描述",
				CallbackData: fmt.Sprintf("savedmsg_channel_add_desc_%d", msgData.ID),
			},
			{
				Text:         "折叠消息",
				CallbackData: fmt.Sprintf("savedmsg_channel_expandable_%d", msgData.ID),
			},
		},
		{
			{
				Text: "查看消息",
				URL:  utils.MsgLinkPrivate(common.SavedMessageList.ChannelID, msgData.ID),
			},
		},
	},
	}
}

func channelCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Str("callbackData", opts.CallbackQuery.Data).
		Logger()

	if strings.HasPrefix(opts.CallbackQuery.Data, "savedmsg_channel_add_desc_") {
		msgIDString := strings.TrimPrefix(opts.CallbackQuery.Data, "savedmsg_channel_add_desc_")
		msgID, err  := strconv.Atoi(msgIDString)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to parse message ID from callback data")
			return fmt.Errorf("failed to parse message ID: %w", err)
		}

		msg, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.CallbackQuery.Message.Message.ID,
			Text:   "请输入新的描述信息，或发送 /cancel 取消编辑。",
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "查看消息",
				URL:  utils.MsgLinkPrivate(common.SavedMessageList.ChannelID, editingMessageID),
			}}}},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.CallbackQuery.Message.Message.ID).
				Str("content", "send text to edit description").
				Msg(flaterr.EditMessageText.Str())
			return fmt.Errorf(flaterr.EditMessageText.Fmt(), "send text to edit description", err)
		} else {
			stateMessageID   = msg.ID
			editingMessageID = msgID
			plugin_utils.AddMessageStateHandler(plugin_utils.MessageStateHandler{
				ForChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				PluginName: "edit_description_state",
				Remaining: 1,
				MessageHandler: editDescriptionHandler,
				CancelHandler: func(opts *handler_params.Message) error {
					_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
						ChatID:      opts.Message.Chat.ID,
						MessageID:   stateMessageID,
						Text:        "已取消编辑",
						ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
							Text: "查看消息",
							URL:  utils.MsgLinkPrivate(common.SavedMessageList.ChannelID, editingMessageID),
						}}}},
					})
					stateMessageID   = 0
					editingMessageID = 0
					if err != nil {
						return fmt.Errorf(flaterr.EditMessageText.Fmt(), "cancel edit description notice", err)
					}
					_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
						ChatID:    opts.Message.Chat.ID,
						MessageID: opts.Message.ID,
					})
					if err != nil {
						return fmt.Errorf(flaterr.DeleteMessage.Fmt(), "cancel edit description command", err)
					}
					return nil
				},
			})
		}
	} else if strings.HasPrefix(opts.CallbackQuery.Data, "savedmsg_channel_expandable_") {
		msgIDString := strings.TrimPrefix(opts.CallbackQuery.Data, "savedmsg_channel_expandable_")

		var msgData meilisearch_utils.MessageData

		err := common.MeilisearchClient.Index(common.SavedMessageList.ChannelIDStr()).GetDocument(msgIDString, nil, &msgData)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to get message data that need edit to expandable blockquote")
			return fmt.Errorf("failed to get message data that need edit to expandable blockquote: %w", err)
		}

		taskinfo, err := common.MeilisearchClient.Index(common.SavedMessageList.ChannelIDStr()).UpdateDocuments(&meilisearch_utils.MessageData{
			ID:   msgData.ID,
			Entities: append(msgData.Entities, models.MessageEntity{
				Type:   models.MessageEntityTypeExpandableBlockquote,
				Offset: 0,
				Length: common.UTF16Length(msgData.Text),
			}),
		})
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to send update document entities request to Meilisearch")
			return fmt.Errorf("failed to send update document entities request to Meilisearch: %w", err)
		} else {
			task, err := meilisearch_utils.WaitForTask(opts.Ctx, &common.MeilisearchClient, taskinfo.TaskUID, time.Second * 1)
			if err != nil {
				return fmt.Errorf("wait for update document entities failed: %w", err)
			} else if task.Status != meilisearch.TaskStatusSucceeded {
				return fmt.Errorf("failed to update document entities: %s", task.Error.Message)
			}

			_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID:   opts.CallbackQuery.Message.Message.ID,
				Text:        "已为消息启用折叠样式",
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "查看消息",
					URL:  utils.MsgLinkPrivate(common.SavedMessageList.ChannelID, editingMessageID),
				}}}},
			})
			editingMessageID = 0 // Reset the editing message ID after successful update
			if err != nil {
				logger.Error().
					Err(err).
					Int("messageID", opts.CallbackQuery.Message.Message.ID).
					Str("content", "entities updated notice").
					Msg(flaterr.EditMessageText.Str())
				return fmt.Errorf(flaterr.EditMessageText.Fmt(), "entities updated notice", err)
			}
		}


	}
	return nil
}

func editDescriptionHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Int("editingMessageID", editingMessageID).
		Logger()

	taskinfo, err := common.MeilisearchClient.Index(common.SavedMessageList.ChannelIDStr()).UpdateDocuments(&meilisearch_utils.MessageData{
		ID:   editingMessageID,
		Desc: opts.Message.Text,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to send update document description request to Meilisearch")
		return fmt.Errorf("failed to send update document description request to Meilisearch: %w", err)
	} else {
		task, err := meilisearch_utils.WaitForTask(opts.Ctx, &common.MeilisearchClient, taskinfo.TaskUID, time.Second * 1)
		if err != nil {
			return fmt.Errorf("wait for update document description failed: %w", err)
		} else if task.Status != meilisearch.TaskStatusSucceeded {
			return fmt.Errorf("failed to update document description: %s", task.Error.Message)
		}

		_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID:      opts.Message.Chat.ID,
			MessageID:   stateMessageID,
			Text:        fmt.Sprintf("消息描述已更新为：[ %s ]", opts.Message.Text),
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "查看消息",
				URL:  utils.MsgLinkPrivate(common.SavedMessageList.ChannelID, editingMessageID),
			}}}},
		})
		editingMessageID = 0 // Reset the editing message ID after successful update
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.Message.ID).
				Str("content", "description updated notice").
				Msg(flaterr.EditMessageText.Str())
			return fmt.Errorf(flaterr.EditMessageText.Fmt(), "description updated notice", err)
		}

		_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
			ChatID:    opts.Message.Chat.ID,
			MessageID: opts.Message.ID,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.Message.ID).
				Str("content", "update description text message").
				Msg(flaterr.DeleteMessage.Str())
			return fmt.Errorf(flaterr.DeleteMessage.Fmt(), "update description text message", err)
		}
	}
	return nil
}

func channelInlineHandler(opts *handler_params.InlineQuery) error {
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

		datas, err := common.MeilisearchClient.Index(common.SavedMessageList.ChannelIDStr()).Search(parsedQuery.KeywordQuery(), &meilisearch.SearchRequest{
			HitsPerPage: int64(configs.BotConfig.InlineResultsPerPage - 1),
			Page:        int64(parsedQuery.Page),
			Filter:      filter,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to get channel saved message")
			handlerErr.Addf("failed to get channel saved message: %w", err)
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
					Description:         "公共频道中没有保存任何内容",
					InputMessageContent: &models.InputTextMessageContent{ MessageText: "用户在找不到想看的东西时无奈点击了提示信息..." },
				})
			}
		} else {
			var msgDatas []meilisearch_utils.MessageData
			err = meilisearch_utils.UnmarshalMessageData(&datas.Hits, &msgDatas)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to unmarshal channel saved message data")
				handlerErr.Addf("Failed to unmarshal channel saved message data: %w", err)
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
							ReplyMarkup:         msgData.BuildButton(common.SavedMessageList.ChannelUsername),
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
							ReplyMarkup:     msgData.BuildButton(common.SavedMessageList.ChannelUsername),
						})
					case message_utils.Document:
						resultList = append(resultList, &models.InlineQueryResultCachedDocument{
							ID:              msgData.MsgIDStr(),
							DocumentFileID:  msgData.FileID,
							Title:           msgData.FileName,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							Description:     msgData.Desc,
							ReplyMarkup:     msgData.BuildButton(common.SavedMessageList.ChannelUsername),
						})
					case message_utils.Animation:
						resultList = append(resultList, &models.InlineQueryResultCachedMpeg4Gif{
							ID:              msgData.MsgIDStr(),
							Mpeg4FileID:     msgData.FileID,
							Title:           msgData.FileName,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							ReplyMarkup:     msgData.BuildButton(common.SavedMessageList.ChannelUsername),
						})
					case message_utils.Photo:
						resultList = append(resultList, &models.InlineQueryResultCachedPhoto{
							ID:                    msgData.MsgIDStr(),
							PhotoFileID:           msgData.FileID,
							Caption:               msgData.Text,
							CaptionEntities:       msgData.Entities,
							Description:           msgData.Desc,
							ShowCaptionAboveMedia: msgData.ShowCaptionAboveMedia,
							ReplyMarkup:           msgData.BuildButton(common.SavedMessageList.ChannelUsername),
						})
					case message_utils.Sticker:
						resultList = append(resultList, &models.InlineQueryResultCachedSticker{
							ID:            msgData.MsgIDStr(),
							StickerFileID: msgData.FileID,
							ReplyMarkup:   msgData.BuildButton(common.SavedMessageList.ChannelUsername),
						})
					case message_utils.Video:
						resultList = append(resultList, &models.InlineQueryResultCachedVideo{
							ID:              msgData.MsgIDStr(),
							VideoFileID:     msgData.FileID,
							Title:           msgData.FileName,
							Description:     msgData.Desc,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							ReplyMarkup:     msgData.BuildButton(common.SavedMessageList.ChannelUsername),
						})
					case message_utils.VideoNote:
						resultList = append(resultList, &models.InlineQueryResultCachedDocument{
							ID:              msgData.MsgIDStr(),
							DocumentFileID:  msgData.FileID,
							Title:           msgData.FileName,
							Description:     msgData.Desc,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							ReplyMarkup:     msgData.BuildButton(common.SavedMessageList.ChannelUsername),
						})
					case message_utils.Voice:
						resultList = append(resultList, &models.InlineQueryResultCachedVoice{
							ID:              msgData.MsgIDStr(),
							VoiceFileID:     msgData.FileID,
							Title:           msgData.FileTitle,
							Caption:         msgData.Text,
							CaptionEntities: msgData.Entities,
							ReplyMarkup:     msgData.BuildButton(common.SavedMessageList.ChannelUsername),
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
		Button:        &models.InlineQueryResultsButton{
			Text:           "当前为公共收藏内容",
			StartParameter: "savedmessage_viewchannel",
		},
		IsPersonal:    true,
		CacheTime:     0,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("content", "saved message result").
			Msg(flaterr.AnswerInlineQuery.Str())
		handlerErr.Addt(flaterr.AnswerInlineQuery, "saved message result", err)
	}

	return handlerErr.Flat()
}

func showChannelLink(opts *handler_params.Message) error {
	var handlerErr flaterr.MultErr

	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Message.Chat.ID,
		Text:   fmt.Sprintf("当前频道链接: https://t.me/%s", common.SavedMessageList.ChannelUsername),
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "查看频道",
			URL:  "https://t.me/" + common.SavedMessageList.ChannelUsername,
		}}}},
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		zerolog.Ctx(opts.Ctx).Error().
			Err(err).
			Str("pluginName", "Saved Message").
			Str(utils.GetCurrentFuncName()).
			Dict(utils.GetUserDict(opts.Message.From)).
			Str("content", "saved message channel link").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "saved message channel link", err)
	}

	return handlerErr.Flat()
}

func InitChannelPart(ctx context.Context) error {
	chatIDString := common.SavedMessageList.ChannelIDStr()

	_, err := common.MeilisearchClient.GetIndex(chatIDString)
	if err != nil {
		if err.(*meilisearch.Error).MeilisearchApiError.Code == "index_not_found" {
			err := meilisearch_utils.CreateChatIndex(ctx, &common.MeilisearchClient, chatIDString)
			if err != nil {
				return fmt.Errorf("failed to create channel index: %w", err)
			}
		} else {
			return fmt.Errorf("failed to get channel index: %w", err)
		}
	}

	channelPubliclink  = "https://t.me/"   + common.SavedMessageList.ChannelUsername
	channelPrivateLink = "https://t.me/c/" + utils.RemoveIDPrefix(common.SavedMessageList.ChannelID)

	plugin_utils.AddHandlerByMessageChatIDHandlers(plugin_utils.ByMessageChatIDHandler{
		ForChatID:      common.SavedMessageList.ChannelID,
		PluginName:     "savedmessage_channel",
		MessageHandler: channelSaveMessageHandler,
	})

	plugin_utils.AddInlineManualHandlers(plugin_utils.InlineManualHandler{
		Command:       "saved",
		InlineHandler: channelInlineHandler,
		Description:   "公共频道中保存的消息",
	})
	plugin_utils.AddSlashStartCommandHandlers(plugin_utils.SlashStartHandler{
		Argument:       "savedmessage_viewchannel",
		MessageHandler: showChannelLink,
	})

	plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
		CallbackDataPrefix:   "savedmsg_channel",
		CallbackQueryHandler: channelCallbackHandler,
	})

	plugin_utils.AddFullCommandHandlers([]plugin_utils.FullCommand{
		{
			FullCommand:    channelPubliclink,
			ForChatType:    []models.ChatType{models.ChatTypePrivate},
			MessageHandler: channelMetadataHandler,
		},
		{
			FullCommand:    channelPrivateLink,
			ForChatType:    []models.ChatType{models.ChatTypePrivate},
			MessageHandler: channelMetadataHandler,
		},
	}...)

	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "收藏消息-公共频道",
		Description: "此功能会实时记录一个频道中的消息并保存它，在之后可以使用 inline 模式查看并发送保存的内容\n\n使用方法：\n点击下方的按钮来使用 inline 模式，当您多次在 inline 模式下使用此 bot 时，在输入框中输入 <code>@</code> 即可看到 bot 会出现在列表中",
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{
			{{
				Text:                         "点击浏览频道收藏",
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
					CallbackData: "delete_this_message",
				},
			},
		}},
	})

	return nil
}
