package plugins

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/inline_utils"
	"trbot/utils/meilisearch_utils"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
	"github.com/rs/zerolog"
)

var meilisearchURL     string = "http://localhost:7700"
var meilisearchAPI     string
var meilisearchClient  meilisearch.ServiceManager
var cacheMessageChatID int = -1002614205550
var maxIndexHistory    int = 500

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "message_meilisearch",
		Func: func(ctx context.Context) error {
			meilisearchClient = meilisearch.New(meilisearchURL, meilisearch.WithAPIKey(meilisearchAPI))
			return nil
		},
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand: "getmessage",
		ForChatType: []models.ChatType{
			models.ChatTypeChannel,
			models.ChatTypeGroup,
			models.ChatTypeSupergroup,
		},
		MessageHandler: indexChat,
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand: "searchmessage",
		ForChatType: []models.ChatType{
			models.ChatTypeGroup,
			models.ChatTypeSupergroup,
		},
		MessageHandler: messageSearchHandeler,
	})
	plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
		CallbackDataPrefix: "msindex_task_",
		CallbackQueryHandler: indexChatCallbackHandler,
	})
	plugin_utils.AddInlineHandlers(plugin_utils.InlineHandler{
		Command: "meili",
		Description: "Meilisearch 消息搜索",
		InlineHandler: messageBrowserInlineHandler,
	})
}

func indexChat(opts *handler_params.Message) error {
	if opts.Message == nil { return nil }
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "message_meilisearch").
		Str("funcName", "indexChat").
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if opts.Message.Chat.Type == models.ChatTypePrivate {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Message.Chat.ID,
			Text: "索引消息功能在私聊中不可用",
		})
		logger.Error().
			Err(err).
			Str("content", "index feature is not available in private chat").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "index feature is not available in private chat", err)
	} else if opts.Message.From != nil && contain.Int64(opts.Message.From.ID, configs.BotConfig.AdminIDs...) {
		chatIDString := strconv.FormatInt(opts.Message.Chat.ID, 10)
		_, err := meilisearchClient.GetIndex(chatIDString)
		if err != nil {
			if err.(*meilisearch.Error).MeilisearchApiError.Code == "index_not_found" {
				meilisearch_utils.CreateChatIndex(&meilisearchClient, chatIDString)
			} else {
				logger.Error().
					Err(err).
					Str("content", "index not found").
					Msg("Failed to get chat index")
				return err
			}
		}

		var indexStartID int
		var indexEndID   int = opts.Message.ID

		if len(opts.Fields) == 1 {
			indexStartID = opts.Message.ID - maxIndexHistory
		} else if len(opts.Fields) == 2 {
			number, err := strconv.Atoi(opts.Fields[1])
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "invalid number").
					Msg("Failed to parse number")
				handlerErr.Addf("failed to parse number: %w", err)
				_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Message.Chat.ID,
					Text:   "无法识别数字",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "failed to parse number").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "failed to parse number", err)
				}
				return handlerErr.Flat()
			}
			indexStartID = opts.Message.ID - number
		} else if len(opts.Fields) == 3 {
			start, err := strconv.Atoi(opts.Fields[1])
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "invalid start number").
					Msg("Failed to parse start number")
				handlerErr.Addf("failed to parse start number: %w", err)
			} else {
				indexStartID = start
				end, err := strconv.Atoi(opts.Fields[2])
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "invalid end number").
						Msg("Failed to parse end number")
					handlerErr.Addf("failed to parse end number: %w", err)
				} else {
					indexEndID = end
				}
			}

			if len(handlerErr.Errors) > 0 {
				_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Message.Chat.ID,
					Text:   "无法识别数字",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "failed to parse start number").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "failed to parse start number", err)
				}
				return handlerErr.Flat()
			}
		}

		if indexStartID < 0 {
			indexStartID = 0
		}

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Message.Chat.ID,
			Text: fmt.Sprintf("索引 %d 条消息", indexEndID - indexStartID + 1),
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
				{
					{
						Text: "查看起点",
						URL: fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(opts.Message.Chat.ID), indexStartID),
					},
					{
						Text: "查看末尾",
						URL: fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(opts.Message.Chat.ID), indexEndID),
					},
				},
				{{
						Text: "开始索引任务",
						CallbackData: fmt.Sprintf("msindex_task_%d_%d", indexStartID, indexEndID),
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
			ChatID: opts.Message.Chat.ID,
			Text:   "索引历史消息的功能仅限管理员使用",
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

func indexMessage(ctx context.Context, thebot *bot.Bot, indexManager meilisearch.IndexManager, fromChatID int64, indexStartID, indexEndID int) (msgSuccess, msgSkip, msgError int) {
	defer utils.PanicCatcher(ctx, "indexMessage")
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "message_meilisearch").
		Str("funcName", "indexMessage").
		Int64("chatID", fromChatID).
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
			message.MsgID = msgID

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

func indexChatCallbackHandler(opts *handler_params.CallbackQuery) error {
	if opts.CallbackQuery == nil { return nil }
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "message_meilisearch").
		Str("funcName", "indexChatCallbackHandler").
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Logger()

	var handlerErr flaterr.MultErr

	var chatID int64 = opts.CallbackQuery.Message.Message.Chat.ID

	// switch opts.CallbackQuery.Data {
	// }
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
		if len(data) != 2 {
			logger.Error().
				Str("content", "invalid callback data format").
				Msg("Failed to parse callback data")
			return handlerErr.Addf("invalid callback data format: %s", opts.CallbackQuery.Data).Flat()
		} else {
			startID, err := strconv.Atoi(data[0])
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse callback data")
				return handlerErr.Addf("failed to parse callback data: %w", err).Flat()
			}
			endID, err := strconv.Atoi(data[1])
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse callback data")
				return handlerErr.Addf("failed to parse callback data: %w", err).Flat()
			}

			var msgSuccess, msgSkip, msgError int
			var single bool = false

			indexManager := meilisearchClient.Index(strconv.FormatInt(chatID, 10))

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
				ChatID: chatID,
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
		}
	}
	return handlerErr.Flat()
}

func messageBrowserInlineHandler(opts *handler_params.InlineQuery) (result []models.InlineQueryResult) {
	// var chatIDstring string = strconv.FormatInt(opts.InlineQuery.From.ID, 10)
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
		for i, data := range datas.Hits {
			result = append(result, &models.InlineQueryResultArticle{
				ID:    strconv.Itoa(i),
				Title: data.(map[string]interface{})["msg_type"].(string) + ": " + data.(map[string]interface{})["text"].(string),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: data.(map[string]interface{})["msg_type"].(string) + ": " + data.(map[string]interface{})["text"].(string),
				},
				ReplyMarkup: &models.InlineKeyboardMarkup{
					InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "Open",
						URL:  fmt.Sprintf("https://t.me/c/%s/%d", chatIDstring[4:], int(data.(map[string]interface{})["msg_id"].(float64))),
					}}},
				},
			})
		}
		if len(result) == 0 {
			result = append(result, &models.InlineQueryResultArticle{
				ID:    "0",
				Title: "No results found",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "No results found",
				},
			})
		}
	}
	return
}

func messageSearchHandeler(opts *handler_params.Message) error {
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
		var buttons [][]models.InlineKeyboardButton
		for i, data := range datas.Hits {
			if i == 10 { break }
			buttons = append(buttons, []models.InlineKeyboardButton{
				{
					Text: data.(map[string]interface{})["text"].(string),
					URL:  fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(opts.Message.Chat.ID), int(data.(map[string]interface{})["msg_id"].(float64))),
				},
			})
		}
		var pendingMessage string
		if len(buttons) != 0 {
			pendingMessage = fmt.Sprintf("Found %d messages", len(datas.Hits))
		} else {
			pendingMessage = "No results found"
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
