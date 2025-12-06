package message_index

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"trle5.xyz/gopkg/trbot/utils"
	"trle5.xyz/gopkg/trbot/utils/configs"
	"trle5.xyz/gopkg/trbot/utils/flaterr"
	"trle5.xyz/gopkg/trbot/utils/handler_params"
	"trle5.xyz/gopkg/trbot/utils/inline_utils"
	"trle5.xyz/gopkg/trbot/utils/meilisearch_utils"
	"trle5.xyz/gopkg/trbot/utils/plugin_utils"
	"trle5.xyz/gopkg/trbot/utils/type/contain"
	"trle5.xyz/gopkg/trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
	"github.com/rs/zerolog"
)

var meilisearchClient  meilisearch.ServiceManager
var cacheMessageChatID int = -1002614205550

func Init(client *meilisearch.ServiceManager) {
	meilisearchClient = *client

	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand: "getmessage",
		ForChatType: []models.ChatType{ models.ChatTypePrivate },
		MessageHandler: indexChat,
	})
// 	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
// 		SlashCommand: "searchmessage",
// 		ForChatType: []models.ChatType{
// 			models.ChatTypeGroup,
// 			models.ChatTypeSupergroup,
// 		},
// 		MessageHandler: messageSearchHandler,
// 	})
	plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
		CallbackDataPrefix: "msindex_",
		CallbackQueryHandler: indexChatCallbackHandler,
	})
// 	plugin_utils.AddInlineHandlers(plugin_utils.InlineHandler{
// 		Command: "meili",
// 		Description: "Meilisearch 消息搜索",
// 		InlineHandler: messageBrowserInlineHandler,
// 	})
}

func indexChat(opts *handler_params.Message) error {
	if opts.Message == nil { return nil }
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if opts.Message.From != nil && contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...) {
		if len(opts.Fields) != 4 {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            "参数不足，请使用 <code>/getmessage [chat ID] [start ID] [end ID]</code> 来请求索引消息",
				ParseMode:       models.ParseModeHTML,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "no enough parameters").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "no enough parameters", err)
			}
			return handlerErr.Flat()
		}

		var indexChatID  string = opts.Fields[1]
		var indexStartID int
		var indexEndID   int

		_, err := meilisearchClient.GetIndex(indexChatID)
		if err != nil {
			if err.(*meilisearch.Error).MeilisearchApiError.Code == "index_not_found" {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            fmt.Sprintf("索引 [ %s ] 不存在，是否要创建它？", indexChatID),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "➕ 创建索引",
							CallbackData: fmt.Sprintf("msindex_create_%s", indexChatID),
						},
						{
							Text: "🚫 取消",
							CallbackData: "delete_this_message",
						},
					}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "index not found notice").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "index not found notice", err)
				}
				return handlerErr.Flat()
			} else {
				logger.Error().
					Err(err).
					Str("content", "index not found").
					Msg("Failed to get chat index")
				handlerErr.Addf("failed to get chat index: %w", err)
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            fmt.Sprintf("获取 [%s] 索引失败：<blockquote expandable>%s</blockquote>", indexChatID, utils.IgnoreHTMLTags(err.Error())),
					ParseMode:       models.ParseModeHTML,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "get chat index failed notice").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "get chat index failed notice", err)
				}
			}
		}

		indexStartID, err = strconv.Atoi(opts.Fields[2])
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "invalid start number").
				Msg("Failed to parse start number")
			handlerErr.Addf("failed to parse start number: %w", err)
		} else {
			indexEndID, err = strconv.Atoi(opts.Fields[3])
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "invalid end number").
					Msg("Failed to parse end number")
				handlerErr.Addf("failed to parse end number: %w", err)
			}
		}

		if err != nil {
			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Message.Chat.ID,
				Text:            fmt.Sprintf("识别起始或结束 ID 时发生错误：<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
				ParseMode:       models.ParseModeHTML,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "failed to parse numbers").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "failed to parse numbers", err)
			}
			return handlerErr.Flat()
		}

		if indexStartID < 0 { indexStartID = 0 }

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.Chat.ID,
			Text:            fmt.Sprintf("在 <a href=\"https://t.me/c/%s/\">%s</a> 中索引 %d 条消息", strings.TrimPrefix(indexChatID, "-100"), indexChatID, indexEndID - indexStartID + 1),
			ParseMode:       models.ParseModeHTML,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text: "查看起点",
						URL:  fmt.Sprintf("https://t.me/c/%s/%d", strings.TrimPrefix(indexChatID, "-100"), indexStartID),
					},
					{
						Text: "查看末尾",
						URL:  fmt.Sprintf("https://t.me/c/%s/%d", strings.TrimPrefix(indexChatID, "-100"), indexEndID),
					},
				},
				{{
					Text:         "开始索引任务",
					CallbackData: fmt.Sprintf("msindex_task_%s_%d_%d", indexChatID, indexStartID, indexEndID),
				}},
			}},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "index message menu").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "index message menu", err)
		}
	} else {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.Chat.ID,
			Text:            "此功能不可用",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "no permission to index history message").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "no permission to index history message", err)
		}
	}
	return handlerErr.Flat()
}

func indexChatCallbackHandler(opts *handler_params.CallbackQuery) error {
	if opts.CallbackQuery == nil { return nil }
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if !contain.Int64(opts.CallbackQuery.From.ID, configs.BotConfig.AdminIDs...) {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text: "索引消息功能仅限管理员使用",
			ShowAlert: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "no permission to start index task").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "no permission to start index task", err)
		}
	} else if strings.HasPrefix(opts.CallbackQuery.Data, "msindex_task_") {
		data := strings.Split(strings.TrimPrefix(opts.CallbackQuery.Data, "msindex_task_"), "_")
		if len(data) != 3 {
			logger.Error().
				Str("content", "invalid callback data format").
				Msg("Failed to parse callback data")
			return handlerErr.Addf("invalid callback data format: %s", opts.CallbackQuery.Data).Flat()
		}
		chatID := data[0]
		startID, err := strconv.Atoi(data[1])
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to parse callback data")
			return handlerErr.Addf("failed to parse callback data: %w", err).Flat()
		}
		endID, err := strconv.Atoi(data[2])
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to parse callback data")
			return handlerErr.Addf("failed to parse callback data: %w", err).Flat()
		}

		var msgSuccess, msgSkip, msgError int
		var single bool = false

		indexManager := meilisearchClient.Index(chatID)

		if single {
			msgSuccess, msgSkip, msgError = indexMessage(opts.Ctx, opts.Thebot, indexManager, chatID, startID, endID)
		} else {
			var wg sync.WaitGroup
			var batchSize = 200

			// 计算 batch 数
			for batchStart := startID; batchStart <= endID; batchStart += batchSize {
				batchEnd := batchStart + batchSize - 1
				if batchEnd > endID {
					batchEnd = endID
				}

				wg.Add(1)
				go func(start, end int) {
					defer wg.Done()
					msgS, msgk, msgE := indexMessage(opts.Ctx, opts.Thebot, indexManager, chatID, start, end)
					msgSuccess += msgS
					msgSkip += msgk
					msgError += msgE
				}(batchStart, batchEnd)
			}
			wg.Wait()
		}

		_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.CallbackQuery.Message.Message.ID,
			Text:   fmt.Sprintf("已索引 %d 条消息，跳过 %d 条，错误 %d 条", msgSuccess, msgSkip, msgError),
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "indexed message count").
				Msg(flaterr.EditMessageText.Str())
			handlerErr.Addt(flaterr.EditMessageText, "indexed message count", err)
		}
	} else if strings.HasPrefix(opts.CallbackQuery.Data, "msindex_create_") {
		chatIDstring := strings.TrimPrefix(opts.CallbackQuery.Data, "msindex_create_")
		err := meilisearch_utils.CreateChatIndex(opts.Ctx, &meilisearchClient, chatIDstring)
		if err != nil {
			logger.Error().
				Err(err).
				Str("chatID", chatIDstring).
				Msg("Failed to create chat index")
			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text:            fmt.Sprintf("创建索引失败：<blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "create chat index failed notice").
					Msg(flaterr.AnswerCallbackQuery.Str())
				handlerErr.Addt(flaterr.AnswerCallbackQuery, "create chat index failed notice", err)
			}
		} else {
			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("已创建索引 [ %s ]，请重新使用命令来索引消息", chatIDstring),
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "create chat index success").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "create chat index success", err)
			}
		}
	}
	return handlerErr.Flat()
}

func indexMessage(ctx context.Context, thebot *bot.Bot, indexManager meilisearch.IndexManager, fromChatID string, indexStartID, indexEndID int) (msgSuccess, msgSkip, msgError int) {
	defer utils.PanicCatcher(ctx, "indexMessage")
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Str("chatID", fromChatID).
		Logger()

	for msgID := indexStartID; msgID <= indexEndID; msgID++ {
		msg, err := thebot.ForwardMessage(ctx, &bot.ForwardMessageParams{
			ChatID:     cacheMessageChatID,
			FromChatID: fromChatID,
			MessageID:  msgID,
		})
		if err != nil {
			switch {
			case err.Error() == "bad request, Bad Request: message to forward not found":
				logger.Info().
					Err(err).
					Int("messageID", msgID).
					Msg("skiped")
				msgSkip++
				continue
			case err.Error() == "bad request, Bad Request: the message can't be forwarded":
				logger.Info().
					Err(err).
					Int("messageID", msgID).
					Msg("skiped")
				msgSkip++
				continue
			default:
				logger.Error().
					Err(err).
					Int("messageID", msgID).
					Str("content", "forward message failed").
					Msg(flaterr.ForwardMessage.Str())
				msgError++
				time.Sleep(time.Second * 1)
			}
		} else {
			message := meilisearch_utils.BuildMessageData(ctx, thebot, msg)
			message.ID = msgID

			_, err := indexManager.AddDocuments(message)
			if err != nil {
				logger.Error().
					Err(err).
					Int("messageID", msgID).
					Str("content", "add message to index failed").
					Msg("failed to add message to index")
				msgError++
			} else {
				msgSuccess++
				logger.Info().
					Str("messageType", message_utils.GetMessageType(msg).Str()).
					Int("messageID", msgID).
					Str("text", message.Text).
					Msg("ForwardMessage")
			}
		}
	}

	return
}

func messageBrowserInlineHandler(opts *handler_params.InlineQuery) (result []models.InlineQueryResult) {
	var chatIDstring string = "-1001247493529"
	indexManager := meilisearchClient.Index(chatIDstring)
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("chat_id", chatIDstring).
		Logger()

	datas, err := indexManager.Search(inline_utils.ParseInlineFields(opts.Fields).KeywordQuery(), &meilisearch.SearchRequest{})
	if err != nil {
		logger.Error().
			Err(err).
			Msg("failed to get message")
	} else {
		var msgDatas []meilisearch_utils.MessageData
		err = meilisearch_utils.UnmarshalMessageData(&datas.Hits, &msgDatas)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("failed to unmarshal message data")
			return []models.InlineQueryResult{ &models.InlineQueryResultArticle{
				ID:    "0",
				Title: "Failed to unmarshal message data",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "Failed to unmarshal message data: " + err.Error(),
				},
			}}
		} else {
			var result []models.InlineQueryResult
			for i, data := range msgDatas {
				result = append(result, &models.InlineQueryResultArticle{
					ID:    strconv.Itoa(i),
					Title: data.Type.Str() + ": " + data.Text,
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: data.Type.Str() + ": " + data.Text,
					},
					ReplyMarkup: &models.InlineKeyboardMarkup{
						InlineKeyboard: [][]models.InlineKeyboardButton{{{
							Text: "Open",
							URL:  fmt.Sprintf("https://t.me/c/%s/%d", chatIDstring[4:], data.ID),
						}}},
					},
				})
			}

			if len(result) == 0 {
				result = append(result, &models.InlineQueryResultArticle{
					ID:    "no_results",
					Title: "No results found",
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "No results found",
					},
				})
			}

			return result
		}
	}
	return nil
}

func messageSearchHandler(opts *handler_params.Message) error {
	var chatIDstring string = strconv.FormatInt(opts.Message.Chat.ID, 10)
	indexManager := meilisearchClient.Index(chatIDstring)
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("chat_id", chatIDstring).
		Logger()

	query := strings.TrimPrefix(opts.Message.Text, opts.Fields[0])

	datas, err := indexManager.Search(query, &meilisearch.SearchRequest{})
	if err != nil {
		logger.Error().
			Err(err).
			Msg("failed to get message")
		return fmt.Errorf("failed to get message: %w", err)
	} else {
		var pendingMessage string
		var buttons [][]models.InlineKeyboardButton
		var msgDatas []meilisearch_utils.MessageData
		err = meilisearch_utils.UnmarshalMessageData(&datas.Hits, &msgDatas)
		if err != nil {
			pendingMessage = fmt.Sprintf("Failed to unmarshal message data: %s", err.Error())
		} else {
			for i, data := range msgDatas {
				if i == 10 { break }
				buttons = append(buttons, []models.InlineKeyboardButton{
					{
						Text: data.Text,
						URL: utils.MsgLinkPrivate(opts.Message.Chat.ID, data.ID),
					},
				})
			}

			if len(buttons) != 0 {
				pendingMessage = fmt.Sprintf("Found %d messages", len(datas.Hits))
			} else {
				pendingMessage = "No results found"
			}
		}

		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.Chat.ID,
			Text:            pendingMessage,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: buttons },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "search message result").
				Msg(flaterr.SendMessage.Str())
			return fmt.Errorf(flaterr.SendMessage.Fmt(), "search message result", err)
		}
	}

	return nil
}
