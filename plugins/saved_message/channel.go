package saved_message

import (
	"fmt"
	"strconv"
	"strings"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/meilisearch_utils"
	"trbot/utils/origin_info"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"unicode/utf8"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
	"github.com/rs/zerolog"
)

var channelPubliclink string
var channelPrivateLink string

func channelSaveMessageHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "channelSaveMessageHandler").
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	if meilisearchClient == nil {
		logger.Warn().
			Int("messageID", opts.Message.ID).
			Msg("Meilisearch client is not initialized, skipping indexing")
	} else {
		msgData := meilisearch_utils.BuildMessageData(opts.Ctx, opts.Thebot, opts.Message)

		var messageLength int

		if opts.Message.Caption != "" {
			messageLength = utf8.RuneCountInString(opts.Message.Caption)
			msgData.Entities = opts.Message.CaptionEntities
		} else if opts.Message.Text != "" {
			messageLength = utf8.RuneCountInString(opts.Message.Text)
			msgData.Entities = opts.Message.Entities
		}

		// 若字符长度大于设定的阈值，添加折叠样式引用再保存，如果是全文引用但不折叠，改成折叠样式
		if messageLength > textExpandableLength && (len(msgData.Entities) == 0 || msgData.Entities[0].Type == models.MessageEntityTypeBlockquote && msgData.Entities[0].Length == messageLength) {
			msgData.Entities = []models.MessageEntity{{
				Type:   models.MessageEntityTypeExpandableBlockquote,
				Offset: 0,
				Length: messageLength,
			}}
		}

		msgData.OriginInfo = origin_info.GetOriginInfo(opts.Message)

		_, err := meilisearchClient.Index(SavedMessageList.ChannelIDStr()).AddDocuments(msgData)
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.Message.ID).
				Str("content", "add message to index failed").
				Msg("failed to add message to index")
			return fmt.Errorf("failed to add message to index: %w", err)
		}
	}

	return nil
}

func channelMetadataHandler(opts *handler_params.Message) error {
	var handlerErr flaterr.MultErr

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "channelMetadataHandler").
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	var isAdmin bool = contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...)

	var msgIDString string
	var msgData meilisearch_utils.MessageData
	indexManager := meilisearchClient.Index(SavedMessageList.ChannelIDStr())

	if !isAdmin {
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
				pendingMessage += fmt.Sprintf("ID: %d\n", msgData.MsgID)
				pendingMessage += fmt.Sprintf("类型: %s\n", msgData.MsgType)
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
					ReplyMarkup:     buildMessageDataKeyboard(msgData),
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

func buildMessageDataKeyboard(msgData meilisearch_utils.MessageData) models.ReplyMarkup {
	var button [][]models.InlineKeyboardButton
	button = append(button, []models.InlineKeyboardButton{
		{
			Text:         "修改描述",
			CallbackData: fmt.Sprintf("savedmsg_channel_add_desc_%d", msgData.MsgID),
		},
		{
			Text: "查看消息",
			URL:  fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(SavedMessageList.ChannelID), msgData.MsgID),
		},
		// {
		// 	Text:         "移除来源信息",
		// 	CallbackData: fmt.Sprintf("savedmsg_remove_origin_%d", msgData.MsgID),
		// },
		// {
		// 	Text:         "删除消息",
		// 	CallbackData: fmt.Sprintf("delete_message:%d", msgData.MsgID),
		// },
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: button,
	}
}

func channelCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "channelCallbackHandler").
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Str("callbackData", opts.CallbackQuery.Data).
		Logger()

	if strings.HasPrefix(opts.CallbackQuery.Data, "savedmsg_channel_add_desc_") {
		msgIDString := strings.TrimPrefix(opts.CallbackQuery.Data, "savedmsg_channel_add_desc_")
		msgID, err := strconv.Atoi(msgIDString)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to parse message ID from callback data")
			return fmt.Errorf("failed to parse message ID: %w", err)
		}
		editingMessageID = msgID
		plugin_utils.AddMessageStateHandler(plugin_utils.MessageStateHandler{
			ForChatID: opts.CallbackQuery.Message.Message.Chat.ID,
			PluginName: "edit_description_state",
			Remaining: 1,
			MessageHandler: editDescriptionHandler,
		})
		_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.CallbackQuery.Message.Message.ID,
			Text:   "请输入新的描述信息，或发送 /cancel 取消编辑。",
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "查看消息",
				URL:  fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(SavedMessageList.ChannelID), editingMessageID),
			}}}},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.CallbackQuery.Message.Message.ID).
				Str("content", "send text to edit description").
				Msg(flaterr.EditMessageText.Str())
			return fmt.Errorf(flaterr.EditMessageText.Fmt(), "send text to edit description", err)
		}
	}
	return nil
}

func editDescriptionHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "editDescriptionHandler").
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	_, err := meilisearchClient.Index(SavedMessageList.ChannelIDStr()).UpdateDocuments(
		&meilisearch_utils.MessageData{
			MsgID: editingMessageID,
			Desc:  opts.Message.Text,
		},
		// map[string]any{
		// 	"msg_id": editingMessageID,
		// 	"desc":  opts.Message.Text,
		// },
	)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to send update document description request to Meilisearch")
		return fmt.Errorf("failed to send update document description request to Meilisearch: %w", err)
	} else {
		editingMessageID = 0 // Reset the editing message ID after successful update
		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Message.Chat.ID,
			Text:   fmt.Sprintf("描述已更新为：[ %s ]", opts.Message.Text),
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "查看消息",
				URL:  fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(SavedMessageList.ChannelID), editingMessageID),
			}}}},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.Message.ID).
				Str("content", "description updated notice").
				Msg(flaterr.SendMessage.Str())
			return fmt.Errorf(flaterr.SendMessage.Fmt(), "description updated notice", err)
		}
	}
	return nil
}
