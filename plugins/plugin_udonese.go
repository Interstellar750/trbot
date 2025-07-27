package plugins

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/inline_utils"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"trbot/utils/type/message_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var UdoneseData Udonese
var UdoneseErr  error
var UdonesePath string = filepath.Join(consts.YAMLDataBaseDir, "udonese/", consts.YAMLFileName)

type Udonese struct {
	GroupID    int64         `yaml:"GroupID"`
	ManagerIDs []int64       `yaml:"ManagerIDs"`
	Count      int           `yaml:"Count"`
	List       []UdoneseWord `yaml:"List"`
}

// 获取全部的词
func (udonese Udonese) OnlyWord() []string {
	var Words []string
	for _, n := range udonese.List {
		Words = append(Words, n.Word)
	}
	return Words
}

type UdoneseWord struct {
	Word        string           `yaml:"Word,omitempty"`
	Used        int              `yaml:"Used"`
	MeaningList []UdoneseMeaning `yaml:"MeaningList,omitempty"`
}

// 从 UdoneseWord 列表中提取 Meaning 切片, 转换为小写
func (list UdoneseWord) OnlyMeaning() []string {
	var meanings []string
	for _, singleMeaning := range list.MeaningList {
		meanings = append(meanings, strings.ToLower(singleMeaning.Meaning))
	}
	return meanings
}

// 以 models.ParseModeHTML 的格式输出一个词和其对应的全部意思
func (list UdoneseWord) OutputMeanings() string {
	var pendingMessage = fmt.Sprintf("[ <code>%s</code> ] 已使用 %d 次，", list.Word, list.Used)
	if len(list.MeaningList) != 0 {
		pendingMessage += "它的意思有\n"
	} else {
		pendingMessage += "它还没有添加任何意思\n"
	}
	for i, s := range list.MeaningList {
		// 先加意思
		pendingMessage += fmt.Sprintf("<code>%d</code>. [ %s ] ", i+1, s.Meaning)

		// 来源的用户或频道
		if s.FromUsername != "" {
			pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a> ", s.FromUsername, s.FromName)
		} else if s.FromID != 0 {
			if s.FromID < 0 {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(s.FromID), s.FromName)
			} else {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", s.FromID, s.FromName)
			}
		}

		// 由其他用户添加时的信息
		if s.ViaUsername != "" {
			pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", s.ViaUsername, s.ViaName)
		} else if s.ViaID != 0 {
			if s.ViaID < 0 {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(s.ViaID), s.ViaName)
			} else {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", s.ViaID, s.ViaName)
			}
		}

		// 末尾换行
		pendingMessage += "\n"
	}
	return pendingMessage
}

// 管理一个词和其所有意思的键盘
func (list UdoneseWord) buildUdoneseWordKeyboard() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton
	for index, singleMeaning := range list.MeaningList {
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: singleMeaning.Meaning,
			CallbackData: fmt.Sprintf("udonese_meaning_%s_%d", list.Word, index),
		}})
	}

	buttons = append(buttons, []models.InlineKeyboardButton{
		{
			Text: "删除整个词",
			CallbackData: "udonese_delword_" + list.Word,
		},
		{
			Text: "关闭菜单",
			CallbackData: "udonese_done",
		},
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

type UdoneseMeaning struct {
	Meaning      string `yaml:"Meaning"`

	FromID       int64  `yaml:"FromID,omitempty"`
	FromUsername string `yaml:"FromUsername,omitempty"`
	FromName     string `yaml:"FromName,omitempty"`

	ViaID        int64  `yaml:"ViaID,omitempty"`
	ViaUsername  string `yaml:"ViaUsername,omitempty"`
	ViaName      string `yaml:"ViaName,omitempty"`
}

func ReadUdonese(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "ReadUdonese").
		Logger()

	err := yaml.LoadYAML(UdonesePath, &UdoneseData)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", UdonesePath).
				Msg("Not found udonese list file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(UdonesePath, &UdoneseData)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", UdonesePath).
					Msg("Failed to create empty udonese list file")
				UdoneseErr = fmt.Errorf("failed to create empty udonese list file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", UdonesePath).
				Msg("Failed to load udonese list file")
			UdoneseErr = fmt.Errorf("failed to load udonese list file: %w", err)
		}
	} else {
		UdoneseErr = nil
	}

	return UdoneseErr
}

func SaveUdonese(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "SaveUdonese").
		Logger()

	UdoneseData.Count = len(UdoneseData.List)
	err := yaml.SaveYAML(UdonesePath, &UdoneseData)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", UdonesePath).
			Msg("Failed to save udonese list")
		UdoneseErr = fmt.Errorf("failed to save udonese list: %w", err)
	} else {
		UdoneseErr = nil
	}
	return UdoneseErr
}

// 如果要添加的意思重复，返回对应意思的单个词结构体指针，否则返回空指针
// 设计之初可以添加多个意思，但现在不推荐这样
func addUdonese(ctx context.Context, params *UdoneseWord) *UdoneseWord {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "SaveUdonese").
		Logger()

	for wordIndex, savedList := range UdoneseData.List {
		if strings.EqualFold(savedList.Word, params.Word){
			logger.Info().
				Str("word", params.Word).
				Msg("Found existing word")
			for _, newMeaning := range params.MeaningList {
				var isreallynew bool = true
				for _, oldmeanlist := range savedList.MeaningList {
					if newMeaning.Meaning == oldmeanlist.Meaning {
						isreallynew = false
					}
				}
				if isreallynew {
					UdoneseData.List[wordIndex].MeaningList = append(UdoneseData.List[wordIndex].MeaningList, newMeaning)
					logger.Info().
						Str("word", params.Word).
						Str("meaning", newMeaning.Meaning).
						Msg("Add new meaning")
				} else {
					logger.Info().
						Str("word", params.Word).
						Str("meaning", newMeaning.Meaning).
						Msg("Skip existing meaning")
					return &savedList
				}
			}
			return nil
		}
	}
	logger.Info().
		Str("word", params.Word).
		Interface("meaningList", params.MeaningList).
		Msg("Add new word")
	UdoneseData.List = append(UdoneseData.List, *params)
	return nil
}

func addUdoneseHandler(opts *handler_params.Message) error {
	// 不响应来自转发的命令
	if opts.Message.ForwardOrigin != nil { return nil }

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "addUdoneseHandler").
		Logger()

	var handlerErr flaterr.MultErr

	isManager := contain.Int64(opts.Message.From.ID, UdoneseData.ManagerIDs...)

	if opts.Message.Chat.ID != UdoneseData.GroupID {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Message.Chat.ID,
			Text:      "抱歉，此命令仅在部分群组可用",
			ParseMode: models.ParseModeMarkdownV1,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			DisableNotification: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&opts.Message.Chat)).
				Str("content", "not in allowed group").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "not in allowed group", err)
		}
	} else {
		if len(opts.Fields) < 3 {
			// 如果是管理员，则显示可以管理词的帮助
			if isManager {
				if len(opts.Fields) < 2 {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:    opts.Message.Chat.ID,
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						Text: "使用 `/udonese <词> <单个意思>` 来添加记录\n或使用 `/udonese <词>` 来管理记录",
						ParseMode: models.ParseModeMarkdownV1,
						DisableNotification: true,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", opts.Message.Chat.ID).
							Str("content", "admin command help").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "admin command help", err)
					}
				} else /* 词信息 */ {
					checkWord := opts.Fields[1]
					var targetWord UdoneseWord
					for _, wordlist := range UdoneseData.List {
						if wordlist.Word == checkWord {
							targetWord = wordlist
						}
					}

					// 如果词存在，则显示词的信息
					if targetWord.Word == "" {
						_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID:    opts.Message.Chat.ID,
							Text:      "似乎没有这个词呢...",
							ParseMode: models.ParseModeMarkdownV1,
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
							DisableNotification: true,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Int64("chatID", opts.Message.Chat.ID).
								Str("content", "admin command no this word").
								Msg(flaterr.SendMessage.Str())
							handlerErr.Addt(flaterr.SendMessage, "admin command no this word", err)
						}
					} else {
						_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID:    opts.Message.Chat.ID,
							Text:      fmt.Sprintf("词: [ %s ]\n有 %d 个意思，已使用 %d 次\n", targetWord.Word, len(targetWord.MeaningList), targetWord.Used),
							ParseMode: models.ParseModeHTML,
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
							ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
							DisableNotification: true,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Int64("chatID", opts.Message.Chat.ID).
								Str("content", "udonese manage keyboard").
								Msg(flaterr.SendMessage.Str())
							handlerErr.Addt(flaterr.SendMessage, "udonese manage keyboard", err)
						}
					}
				}
			} else /* 普通用户 */ {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					Text: "使用 `/udonese <词> <单个意思>` 来添加记录",
					ParseMode: models.ParseModeMarkdownV1,
					DisableNotification: true,
				})
				if err != nil {
					logger.Info().
						Err(err).
						Int64("chatID", opts.Message.Chat.ID).
						Str("content", "udonese command help").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "udonese command help", err)
				}
			}
		} else {
			meaning := strings.TrimSpace(opts.Message.Text[len(opts.Fields[0]) + len(opts.Fields[1]) + 2:])

			var (
				fromID       int64
				fromUsername string
				fromName     string
				viaID        int64
				viaUsername  string
				viaName      string
			)

			msgAttr := message_utils.GetMessageAttribute(opts.Message)

			if msgAttr.IsReplyToMessage {
				replyAttr := message_utils.GetMessageAttribute(opts.Message.ReplyToMessage)

				if replyAttr.IsUserAsChannel || replyAttr.IsFromLinkedChannel || replyAttr.IsFromAnonymous {
					// 回复给一条频道身份的信息
					fromID = opts.Message.ReplyToMessage.SenderChat.ID
					fromUsername = opts.Message.ReplyToMessage.SenderChat.Username
					fromName = utils.ShowChatName(opts.Message.ReplyToMessage.SenderChat)
				} else {
					// 回复给普通用户
					fromName = utils.ShowUserName(opts.Message.ReplyToMessage.From)
					fromID = opts.Message.ReplyToMessage.From.ID
				}
				if msgAttr.IsUserAsChannel || msgAttr.IsFromLinkedChannel || msgAttr.IsFromAnonymous {
					// 频道身份
					viaID = opts.Message.SenderChat.ID
					viaUsername = opts.Message.SenderChat.Username
					viaName = utils.ShowChatName(opts.Message.SenderChat)
				} else {
					// 普通用户身份
					viaID = opts.Message.From.ID
					viaName = utils.ShowUserName(opts.Message.From)
				}
			} else {
				if msgAttr.IsUserAsChannel || msgAttr.IsFromAnonymous {
					// 频道身份
					fromID = opts.Message.SenderChat.ID
					fromUsername = opts.Message.SenderChat.Username
					fromName = utils.ShowChatName(opts.Message.SenderChat)
				} else {
					// 普通用户身份
					fromID = opts.Message.From.ID
					fromName = utils.ShowUserName(opts.Message.From)
				}
			}

			// 来源和经过都是同一位用户，删除 via 信息
			if fromID == viaID {
				viaID = 0
				viaUsername = ""
				viaName = ""
			}

			var pendingMessage string
			var err error

			oldMeaning := addUdonese(opts.Ctx, &UdoneseWord{
				Word: opts.Fields[1],
				MeaningList: []UdoneseMeaning{{
					Meaning:      meaning,
					FromID:       fromID,
					FromUsername: fromUsername,
					FromName:     fromName,
					ViaID:        viaID,
					ViaUsername:  viaUsername,
					ViaName:      viaName,
				}},
			})
			if oldMeaning != nil {
				pendingMessage += fmt.Sprintf("[%s] 意思已存在于 [%s] 中:\n", meaning, oldMeaning.Word)
				for i, s := range oldMeaning.MeaningList {
					if meaning == s.Meaning {
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] ", i + 1, s.Meaning)

						// 来源的用户或频道
						if s.FromUsername != "" {
							pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a> ", s.FromUsername, s.FromName)
						} else if s.FromID != 0 {
							if s.FromID < 0 {
								pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(s.FromID), s.FromName)
							} else {
								pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", s.FromID, s.FromName)
							}
						}

						// 由其他用户添加时的信息
						if s.ViaUsername != "" {
							pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", s.ViaUsername, s.ViaName)
						} else if s.ViaID != 0 {
							if s.ViaID < 0 {
								pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(s.ViaID), s.ViaName)
							} else {
								pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", s.ViaID, s.ViaName)
							}
						}

						// 末尾换行
						pendingMessage += "\n"
					}
				}
			} else {
				err = SaveUdonese(opts.Ctx)
				if err != nil {
					logger.Error().
						Err(err).
						Str("messageText", opts.Message.Text).
						Msg("Failed to save udonese list after add word")
					handlerErr.Addf("failed to save udonese list after add word: %w", err)

					pendingMessage += fmt.Sprintf("保存语句时似乎发生了一些错误: <blockquote expandable>%s</blockquote>", err.Error())
					_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Message.Chat.ID,
						Text: pendingMessage,
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("messageText", opts.Message.Text).
							Str("content", "failed save udonese list notice").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "failed to save udonese list notice", err)
					}
					return handlerErr.Flat()
				}

				pendingMessage += fmt.Sprintf("已添加 [<code>%s</code>]\n", opts.Fields[1])
				pendingMessage += fmt.Sprintf("[%s] ", meaning)

				// 来源的用户或频道
				if fromUsername != "" {
					pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a> ", fromUsername, fromName)
				} else if fromID != 0 {
					if fromID < 0 {
						pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(fromID), fromName)
					} else {
						pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", fromID, fromName)
					}
				}

				// 由其他用户添加时的信息
				if viaUsername != "" {
					pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", viaUsername, viaName)
				} else if viaID != 0 {
					if viaID < 0 {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(viaID), viaName)
					} else {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", viaID, viaName)
					}
				}
			}

			pendingMessage += fmt.Sprintln("<blockquote>发送的消息与此消息将在十秒后删除</blockquote>")
			botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Message.Chat.ID,
				Text: pendingMessage,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				ParseMode: models.ParseModeHTML,
				DisableNotification: true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&opts.Message.Chat)).
					Str("content", "udonese keyword added").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "udonese keyword added", err)
			} else {
				time.Sleep(time.Second * 10)
				_, err = opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
					ChatID: opts.Message.Chat.ID,
					MessageIDs: []int{
						opts.Message.ID,
						botMessage.ID,
					},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetChatDict(&opts.Message.Chat)).
						Ints("messageIDs", []int{ opts.Message.ID, botMessage.ID }).
						Str("content", "udonese keyword added").
						Msg(flaterr.DeleteMessages.Str())
					handlerErr.Addt(flaterr.DeleteMessages, "udonese keyword added", err)
				}
			}
		}
	}
	return handlerErr.Flat()
}

func udoneseInlineHandler(opts *handler_params.InlineQuery) []models.InlineQueryResult {
	var udoneseResultList []models.InlineQueryResult

	// 查语句需要不区分大小写
	for i := 0; i < len(opts.Fields); i++ {
		opts.Fields[i] = strings.ToLower(opts.Fields[i])
	}

	keywordFields := inline_utils.ExtractKeywords(opts.Fields)

	// 仅 :sms 参数，或带有分页符号，输出全部词
	if len(keywordFields) == 0 {
		for _, data := range UdoneseData.List {
			var pendingMessage string
			if len(data.MeaningList) > 0 {
				pendingMessage = fmt.Sprintf("已使用 %d 次，有 %d 个意思: %s...", data.Used, len(data.MeaningList), data.MeaningList[0].Meaning)
			} else {
				pendingMessage = fmt.Sprintf("已使用 %d 次，暂无意思", data.Used)
			}
			udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
				ID:    data.Word + "-word",
				Title: data.Word,
				Description: pendingMessage,
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: data.OutputMeanings(),
					ParseMode: models.ParseModeHTML,
				},
			})
		}
	} else {
		for _, data := range UdoneseData.List {
			// 通过词查找意思
			var pendingMessage string
			if len(data.MeaningList) > 0 {
				pendingMessage = fmt.Sprintf("已使用 %d 次，有 %d 个意思: %s...", data.Used, len(data.MeaningList), data.MeaningList[0].Meaning)
			} else {
				pendingMessage = fmt.Sprintf("已使用 %d 次，暂无意思", data.Used)
			}
			if inline_utils.MatchMultKeyword(keywordFields, []string{strings.ToLower(data.Word)}) {
				udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
					ID:    data.Word + "-word",
					Title: data.Word,
					Description: pendingMessage,
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: data.OutputMeanings(),
						ParseMode: models.ParseModeHTML,
					},
				})
			}
			// 通过意思查找词
			if inline_utils.MatchMultKeyword(keywordFields, data.OnlyMeaning()) {
				for i, n := range data.MeaningList {
					if inline_utils.MatchMultKeyword(keywordFields, []string{strings.ToLower(n.Meaning)}) {
						udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
							ID:    fmt.Sprintf("%s-meaning-%d", data.Word, i),
							Title: n.Meaning,
							Description: fmt.Sprintf("%s 对应的词是 %s", n.Meaning, data.Word),
							InputMessageContent: &models.InputTextMessageContent{
								MessageText: fmt.Sprintf("%s 对应的词是 <code>%s</code>", n.Meaning, data.Word),
								ParseMode: models.ParseModeHTML,
							},
						})
					}
				}
			}
		}
		if len(udoneseResultList) == 0 {
			udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
				ID:       "none",
				Title:    "没有符合关键词的内容",
				Description: fmt.Sprintf("没有找到包含 %s 的词或意思，若想查看添加方法，请点击这条内容", keywordFields),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "没有这个词，使用 `udonese <词> <意思>` 来添加吧",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}
	if len(udoneseResultList) == 0 {
		udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
			ID:    "none",
			Title: "没有记录任何内容",
			Description: "什么都没有，使用 `/udonese <词> <意思>` 来添加吧",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "使用 `/udonese <词> <单个意思>` 来添加记录",
				ParseMode: models.ParseModeMarkdownV1,
			},
		})
	}
	return udoneseResultList
}

func udoneseGroupHandler(opts *handler_params.Message) error {
	// 不响应来自转发的命令和空文本
	if opts.Message.ForwardOrigin != nil || len(opts.Fields) < 1 { return nil }

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "udoneseGroupHandler").
		Logger()

	var handlerErr flaterr.MultErr

	if UdoneseErr != nil {
		logger.Warn().
			Err(UdoneseErr).
			Msg("Some error in while read udonese list, try to read again")

		err := ReadUdonese(opts.Ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to read udonese list")
			return handlerErr.Addf("failed to read udonese list: %w", err).Flat()
		}
	}

	// 统计词使用次数
	for i, n := range UdoneseData.OnlyWord() {
		if n == opts.Message.Text || strings.HasPrefix(opts.Message.Text, n) {
			UdoneseData.List[i].Used++
			err := SaveUdonese(opts.Ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to save udonese list after add word usage count")
				handlerErr.Addf("failed to save udonese list after add word usage count: %w", err)
			}
			break
		}
	}

	var needNotice bool

	if opts.Fields[0] == "sms" {
		// 参数过少，提示用法
		if len(opts.Fields) < 2 {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				Text:   "使用方法：发送 `sms <词>` 来查看对应的意思",
				ParseMode: models.ParseModeMarkdownV1,
				DisableNotification: true,
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击浏览全部词与意思",
					SwitchInlineQueryCurrentChat: configs.BotConfig.InlineSubCommandSymbol + "sms ",
				}}}},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", opts.Message.Chat.ID).
					Str("content", "sms command usage").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "sms command usage", err)
			}
		} else {
			// 在数据库循环查找这个词
			for _, word := range UdoneseData.List {
				if strings.EqualFold(word.Word, opts.Fields[1]) {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Message.Chat.ID,
						Text:   word.OutputMeanings(),
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						ParseMode: models.ParseModeHTML,
						DisableNotification: true,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", opts.Message.Chat.ID).
							Str("content", "sms keyword meaning").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "sms keyword meaning", err)
					}
					return handlerErr.Flat()
				}
			}
			needNotice = true
		}
	} else if len(opts.Fields) == 2 && strings.HasSuffix(opts.Message.Text, "ssm") {
		// 在数据库循环查找这个词
		for _, word := range UdoneseData.List {
			if strings.EqualFold(word.Word, opts.Fields[0]) {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					ParseMode: models.ParseModeHTML,
					DisableNotification: true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("chatID", opts.Message.Chat.ID).
						Str("content", "sms keyword meaning").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "sms keyword meaning", err)
				}
				return handlerErr.Flat()
			}
		}
		needNotice = true
	}

	if needNotice {
		// 到这里就是没找到，提示没有
		botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
			DisableNotification: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int64("chatID", opts.Message.Chat.ID).
				Int("messageID", botMessage.ID).
				Str("content", "sms keyword no meaning").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "sms keyword no meaning", err)
		} else {
			time.Sleep(time.Second * 10)
			_, err = opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
				ChatID: opts.Message.Chat.ID,
				MessageIDs: []int{ botMessage.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", opts.Message.Chat.ID).
					Int("messageID", botMessage.ID).
					Str("content", "sms keyword no meaning").
					Msg(flaterr.DeleteMessage.Str())
				handlerErr.Addt(flaterr.DeleteMessage, "sms keyword no meaning", err)
			}
		}
	}

	return handlerErr.Flat()
}

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "Udonese",
		Func: func(ctx context.Context) error {
			err := ReadUdonese(ctx)
			if err != nil {
				return err
			} else {
				if UdoneseData.GroupID != 0 {
					plugin_utils.AddHandlerByChatIDHandlers(plugin_utils.ByChatIDHandler{
						ForChatID:      UdoneseData.GroupID,
						PluginName:     "udoneseGroupHandler",
						MessageHandler: udoneseGroupHandler,
					})
				} else {
					return errors.New("Udonese group ID is not set")
				}
			}
			return nil
		},
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Udonese",
		Saver:  SaveUdonese,
		Loader: ReadUdonese,
	})
	plugin_utils.AddInlineHandlerHandlers(plugin_utils.InlineHandler{
		Command:       "sms",
		InlineHandler: udoneseInlineHandler,
		Description:   "查询 Udonese 词典",
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "udonese",
		MessageHandler: addUdoneseHandler,
	})
	plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
		CallbackDataPrefix:   "udonese",
		CallbackQueryHandler: udoneseCallbackHandler,
	})
}

func udoneseCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "udoneseCallbackHandler").
		Str("callbackQueryData", opts.CallbackQuery.Data).
		Logger()

	var handlerErr flaterr.MultErr


	if !contain.Int64(opts.CallbackQuery.From.ID, UdoneseData.ManagerIDs...) {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text: "不可以！",
			ShowAlert: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "udonese no edit permissions").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "udonese no edit permissions", err)
		}
	} else {
		if opts.CallbackQuery.Data == "udonese_done" {
			_, err := opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
				ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", opts.CallbackQuery.Message.Message.Chat.ID).
					Int("messageID", opts.CallbackQuery.Message.Message.ID).
					Str("content", "udonese keyword manage keyboard").
					Msg(flaterr.DeleteMessage.Str())
				handlerErr.Addt(flaterr.DeleteMessage, "udonese keyword manage keyboard", err)
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "udonese_word_") {
			word := strings.TrimPrefix(opts.CallbackQuery.Data, "udonese_word_")
			var targetWord UdoneseWord
			for _, wordlist := range UdoneseData.List {
				if wordlist.Word == word {
					targetWord = wordlist
				}
			}

			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text:      fmt.Sprintf("词: [ %s ]\n有 %d 个意思，已使用 %d 次\n", targetWord.Word, len(targetWord.MeaningList), targetWord.Used),
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", opts.CallbackQuery.Message.Message.Chat.ID).
					Int("messageID", opts.CallbackQuery.Message.Message.ID).
					Str("content", "udonese word meaning list").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "udonese keyword manage keyboard", err)
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "udonese_meaning_") {
			wordAndIndex := strings.TrimPrefix(opts.CallbackQuery.Data, "udonese_meaning_")
			wordAndIndexList := strings.Split(wordAndIndex, "_")
			meanningIndex, err := strconv.Atoi(wordAndIndexList[1])
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse meanning index")
				handlerErr.Addf("failed to parse meanning index: %w", err)
			} else {
				var targetMeaning UdoneseMeaning

				for _, udonese := range UdoneseData.List {
					if udonese.Word == wordAndIndexList[0] {
						for i, meaning := range udonese.MeaningList {
							if i == meanningIndex {
								targetMeaning = meaning
							}
						}
					}
				}

				var pendingMessage string = fmt.Sprintf("意思: [ %s ]\n", targetMeaning.Meaning)

				// 来源的用户或频道
				if targetMeaning.FromUsername != "" {
					pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a>\n", targetMeaning.FromUsername, targetMeaning.FromName)
				} else if targetMeaning.FromID != 0 {
					if targetMeaning.FromID < 0 {
						pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a>\n", utils.RemoveIDPrefix(targetMeaning.FromID), targetMeaning.FromName)
					} else {
						pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a>\n", targetMeaning.FromID, targetMeaning.FromName)
					}
				}

				// 由其他用户添加时的信息
				if targetMeaning.ViaUsername != "" {
					pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a>\n", targetMeaning.ViaUsername, targetMeaning.ViaName)
				} else if targetMeaning.ViaID != 0 {
					if targetMeaning.ViaID < 0 {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a>\n", utils.RemoveIDPrefix(targetMeaning.ViaID), targetMeaning.ViaName)
					} else {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a>\n", targetMeaning.ViaID, targetMeaning.ViaName)
					}
				}

				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					Text: pendingMessage,
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "删除此意思",
							CallbackData: fmt.Sprintf("udonese_delmeaning_%s_%d", wordAndIndexList[0], meanningIndex),
						},
						{
							Text: "返回",
							CallbackData: "udonese_word_" + wordAndIndexList[0],
						},
					}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("chatID", opts.CallbackQuery.Message.Message.Chat.ID).
						Int("messageID", opts.CallbackQuery.Message.Message.ID).
						Str("content", "udonese meaning manage keyboard").
						Msg(flaterr.EditMessageText.Str())
					handlerErr.Addt(flaterr.EditMessageText, "udonese meaning manage keyboard", err)
				}
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "udonese_delmeaning_") {
			wordAndIndex := strings.TrimPrefix(opts.CallbackQuery.Data, "udonese_delmeaning_")
			wordAndIndexList := strings.Split(wordAndIndex, "_")
			meanningIndex, err := strconv.Atoi(wordAndIndexList[1])
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse meanning index")
				handlerErr.Addf("failed to parse meanning index: %w", err)
			} else {
				var newMeaningList []UdoneseMeaning
				var targetWord UdoneseWord
				var deletedMeaning string

				for index, udonese := range UdoneseData.List {
					if udonese.Word == wordAndIndexList[0] {
						for i, meaning := range udonese.MeaningList {
							if i == meanningIndex {
								deletedMeaning = meaning.Meaning
							} else {
								newMeaningList = append(newMeaningList, meaning)
							}
						}
						UdoneseData.List[index].MeaningList = newMeaningList
						targetWord = UdoneseData.List[index]
					}
				}

				err = SaveUdonese(opts.Ctx)
				if err != nil {
					logger.Error().
						Err(err).
						Msg("Failed to save udonese data after deleting meaning")
					handlerErr.Addf("failed to save udonese data after deleting meaning: %w", err)

					_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
						CallbackQueryID: opts.CallbackQuery.ID,
						Text:            "删除意思时保存数据库失败，请重试或联系机器人管理员\n" + err.Error(),
						ShowAlert:       true,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "failed to save udonese data after delete meaning").
							Msg(flaterr.AnswerCallbackQuery.Str())
						handlerErr.Addt(flaterr.AnswerCallbackQuery, "failed to save udonese data after delete meaning", err)
					}
				} else {
					_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
						ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
						MessageID: opts.CallbackQuery.Message.Message.ID,
						Text:      fmt.Sprintf("词: [ %s ]\n有 %d 个意思，已使用 %d 次\n<blockquote>已删除 [ %s ] 词中的 [ %s ] 意思</blockquote>", targetWord.Word, len(targetWord.MeaningList), targetWord.Used, targetWord.Word, deletedMeaning),
						ParseMode: models.ParseModeHTML,
						ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", opts.CallbackQuery.Message.Message.Chat.ID).
							Int("messageID", opts.CallbackQuery.Message.Message.ID).
							Str("content", "udonese meaning manage keyboard after delete meaning").
							Msg(flaterr.EditMessageText.Str())
						handlerErr.Addt(flaterr.EditMessageText, "udonese meaning manage keyboard after delete meaning", err)
					}
				}
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "udonese_delword_") {
			word := strings.TrimPrefix(opts.CallbackQuery.Data, "udonese_delword_")
			var newWordList []UdoneseWord
			for _, udonese := range UdoneseData.List {
				if udonese.Word != word {
					newWordList = append(newWordList, udonese)
				}
			}
			UdoneseData.List = newWordList

			err := SaveUdonese(opts.Ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to save udonese data after delete word")
				handlerErr.Addf("failed to save udonese data after delete word: %w", err)

				_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "删除词时保存数据库失败，请重试\n" + err.Error(),
					ShowAlert:       true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Msg(flaterr.AnswerCallbackQuery.Str())
					handlerErr.Addt(flaterr.AnswerCallbackQuery, "failed to save udonese data after delete word", err)
				}
			} else {
				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					Text:      fmt.Sprintf("<blockquote>已删除 [ %s ] 词</blockquote>", word),
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "关闭菜单",
						CallbackData: "udonese_done",
					}}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("chatID", opts.CallbackQuery.Message.Message.Chat.ID).
						Int("messageID", opts.CallbackQuery.Message.Message.ID).
						Str("content", "udonese word deleted notice").
						Msg(flaterr.EditMessageText.Str())
					handlerErr.Addt(flaterr.EditMessageText, "udonese word deleted notice", err)
				}
			}
		}
	}
	return handlerErr.Flat()
}
