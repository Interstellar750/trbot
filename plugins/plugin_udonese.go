package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/errt"
	"trbot/utils/handler_structs"
	"trbot/utils/multe"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/message_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var UdoneseData Udonese
var UdoneseErr  error

var UdoneseDir  string = filepath.Join(consts.YAMLDataBasePath, "udonese/")
var UdonesePath string = filepath.Join(UdoneseDir, consts.YAMLFileName)
var UdonGroupID  int64 = -1002205667779
// var UdonGroupID  int64 = -1002499888124 // trbot
var UdoneseManagerIDs []int64 = []int64{
	872082796,  // akaudon
	1086395364, // trle5
}

type Udonese struct {
	Count int           `yaml:"count"`
	List  []UdoneseWord `yaml:"list"`
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
	UdoneseData.Count++
	return nil
}

func addUdoneseHandler(opts *handler_structs.SubHandlerParams) error {
	// 不响应来自转发的命令
	if opts.Update.Message.ForwardOrigin != nil { return nil }

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "addUdoneseHandler").
		Logger()

	var handlerErr multe.MultiError

	isManager := utils.AnyContains(opts.Update.Message.From.ID, UdoneseManagerIDs)

	if opts.Update.Message.Chat.ID != UdonGroupID {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Update.Message.Chat.ID,
			Text:      "抱歉，此命令仅在部分群组可用",
			ParseMode: models.ParseModeMarkdownV1,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			DisableNotification: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
				Str("content", "/udonese not allowed group").
				Msg(errt.SendMessage)
			handlerErr.Addf("failed to send `/udonese not allowed group` message: %w", err)
		}
	} else {
		if len(opts.Fields) < 3 {
			// 如果是管理员，则显示可以管理词的帮助
			if isManager {
				if len(opts.Fields) < 2 {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:    opts.Update.Message.Chat.ID,
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
						Text: "使用 `/udonese <词> <单个意思>` 来添加记录\n或使用 `/udonese <词>` 来管理记录",
						ParseMode: models.ParseModeMarkdownV1,
						DisableNotification: true,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", opts.Update.Message.Chat.ID).
							Str("content", "/udonese admin command help").
							Msg(errt.SendMessage)
						handlerErr.Addf("failed to send `/udonese admin command help` message: %w", err)
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
							ChatID:    opts.Update.Message.Chat.ID,
							Text:      "似乎没有这个词呢...",
							ParseMode: models.ParseModeMarkdownV1,
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
							DisableNotification: true,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Int64("chatID", opts.Update.Message.Chat.ID).
								Str("content", "/udonese admin command no this word").
								Msg(errt.SendMessage)
							handlerErr.Addf("failed to send `/udonese admin command no this word` message: %w", err)
						}
					} else {
						_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID:    opts.Update.Message.Chat.ID,
							Text:      fmt.Sprintf("词: [ %s ]\n有 %d 个意思，已使用 %d 次\n", targetWord.Word, len(targetWord.MeaningList), targetWord.Used),
							ParseMode: models.ParseModeHTML,
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
							ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
							DisableNotification: true,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Int64("chatID", opts.Update.Message.Chat.ID).
								Str("content", "/udonese manage keyboard").
								Msg(errt.SendMessage)
							handlerErr.Addf("failed to send `/udonese manage keyboard` message: %w", err)
						}
					}
				}
			} else /* 普通用户 */ {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					Text: "使用 `/udonese <词> <单个意思>` 来添加记录",
					ParseMode: models.ParseModeMarkdownV1,
					DisableNotification: true,
				})
				if err != nil {
					logger.Info().
						Err(err).
						Int64("chatID", opts.Update.Message.Chat.ID).
						Str("content", "/udonese command help").
						Msg(errt.SendMessage)
					handlerErr.Addf("failed to send `/udonese command help` message: %w", err)
				}
			}
		} else {
			meaning := strings.TrimSpace(opts.Update.Message.Text[len(opts.Fields[0]) + len(opts.Fields[1]) + 2:])

			var (
				fromID       int64
				fromUsername string
				fromName     string
				viaID        int64
				viaUsername  string
				viaName      string
			)

			msgAttr := message_utils.GetMessageAttribute(opts.Update.Message)

			if msgAttr.IsReplyToMessage {
				replyAttr := message_utils.GetMessageAttribute(opts.Update.Message.ReplyToMessage)

				if replyAttr.IsUserAsChannel || replyAttr.IsFromLinkedChannel || replyAttr.IsFromAnonymous {
					// 回复给一条频道身份的信息
					fromID = opts.Update.Message.ReplyToMessage.SenderChat.ID
					fromUsername = opts.Update.Message.ReplyToMessage.SenderChat.Username
					fromName = utils.ShowChatName(opts.Update.Message.ReplyToMessage.SenderChat)
				} else {
					// 回复给普通用户
					fromName = utils.ShowUserName(opts.Update.Message.ReplyToMessage.From)
					fromID = opts.Update.Message.ReplyToMessage.From.ID
				}
				if msgAttr.IsUserAsChannel || msgAttr.IsFromLinkedChannel || msgAttr.IsFromAnonymous {
					// 频道身份
					viaID = opts.Update.Message.SenderChat.ID
					viaUsername = opts.Update.Message.SenderChat.Username
					viaName = utils.ShowChatName(opts.Update.Message.SenderChat)
				} else {
					// 普通用户身份
					viaID = opts.Update.Message.From.ID
					viaName = utils.ShowUserName(opts.Update.Message.From)
				}
			} else {
				if msgAttr.IsUserAsChannel || msgAttr.IsFromAnonymous {
					// 频道身份
					fromID = opts.Update.Message.SenderChat.ID
					fromUsername = opts.Update.Message.SenderChat.Username
					fromName = utils.ShowChatName(opts.Update.Message.SenderChat)
				} else {
					// 普通用户身份
					fromID = opts.Update.Message.From.ID
					fromName = utils.ShowUserName(opts.Update.Message.From)
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
						Str("messageText", opts.Update.Message.Text).
						Msg("Failed to save udonese list after add word")
					handlerErr.Addf("failed to save udonese list after add word: %w", err)

					pendingMessage += fmt.Sprintln("保存语句时似乎发生了一些错误:\n", err)
				} else {
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
			}

			pendingMessage += fmt.Sprintln("<blockquote>发送的消息与此消息将在十秒后删除</blockquote>")
			botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text: pendingMessage,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ParseMode: models.ParseModeHTML,
				DisableNotification: true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", opts.Update.Message.Chat.ID).
					Str("content", "/udonese keyword added").
					Msg(errt.SendMessage)
				handlerErr.Addf("failed to send `/udonese keyword added` message: %w", err)
			} else {
				time.Sleep(time.Second * 10)
				_, err = opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
					ChatID: opts.Update.Message.Chat.ID,
					MessageIDs: []int{
						opts.Update.Message.ID,
						botMessage.ID,
					},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("chatID", opts.Update.Message.Chat.ID).
						Ints("messageIDs", []int{ opts.Update.Message.ID, botMessage.ID }).
						Str("content", "/udonese keyword added").
						Msg(errt.DeleteMessages)
					handlerErr.Addf("failed to delete `/udonese keyword added` messages: %w", err)
				}
			}
		}
	}
	return handlerErr.Flat()
}

func udoneseInlineHandler(opts *handler_structs.SubHandlerParams) []models.InlineQueryResult {
	var udoneseResultList []models.InlineQueryResult

	// 查语句需要不区分大小写
	for i := 0; i < len(opts.Fields); i++ {
		opts.Fields[i] = strings.ToLower(opts.Fields[i])
	}

	keywordFields := utils.InlineExtractKeywords(opts.Fields)

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
				InputMessageContent: models.InputTextMessageContent{
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
			if utils.InlineQueryMatchMultKeyword(keywordFields, []string{strings.ToLower(data.Word)}) {
				udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
					ID:    data.Word + "-word",
					Title: data.Word,
					Description: pendingMessage,
					InputMessageContent: models.InputTextMessageContent{
						MessageText: data.OutputMeanings(),
						ParseMode: models.ParseModeHTML,
					},
				})
			}
			// 通过意思查找词
			if utils.InlineQueryMatchMultKeyword(keywordFields, data.OnlyMeaning()) {
				for _, n := range data.MeaningList {
					if utils.InlineQueryMatchMultKeyword(keywordFields, []string{strings.ToLower(n.Meaning)}) {
						udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
							ID:    n.Meaning + "-meaning",
							Title: n.Meaning,
							Description: fmt.Sprintf("%s 对应的词是 %s", n.Meaning, data.Word),
							InputMessageContent: models.InputTextMessageContent{
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
				InputMessageContent: models.InputTextMessageContent{
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
			InputMessageContent: models.InputTextMessageContent{
				MessageText: "使用 `/udonese <词> <单个意思>` 来添加记录",
				ParseMode: models.ParseModeMarkdownV1,
			},
		})
	}
	return udoneseResultList
}

func udoneseGroupHandler(opts *handler_structs.SubHandlerParams) error {
	// 不响应来自转发的命令和空文本
	if opts.Update.Message.ForwardOrigin != nil || len(opts.Fields) < 1 { return nil }

	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "udoneseGroupHandler").
		Logger()

	var handlerErr multe.MultiError

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
		if n == opts.Update.Message.Text || strings.HasPrefix(opts.Update.Message.Text, n) {
			UdoneseData.List[i].Used++
			err := SaveUdonese(opts.Ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to save udonese list after add word usage count")
				handlerErr.Addf("failed to save udonese list after add word usage count: %w", err)
			}
		}
	}

	var needNotice bool

	if opts.Fields[0] == "sms" {
		// 参数过少，提示用法
		if len(opts.Fields) < 2 {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
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
					Int64("chatID", opts.Update.Message.Chat.ID).
					Str("content", "sms command usage").
					Msg(errt.SendMessage)
				handlerErr.Addf("failed to send `sms command usage` message: %w", err)
			}
		} else {
			// 在数据库循环查找这个词
			for _, word := range UdoneseData.List {
				if strings.EqualFold(word.Word, opts.Fields[1]) {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Update.Message.Chat.ID,
						Text:   word.OutputMeanings(),
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
						ParseMode: models.ParseModeHTML,
						DisableNotification: true,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", opts.Update.Message.Chat.ID).
							Str("content", "sms keyword meaning").
							Msg(errt.SendMessage)
						handlerErr.Addf("failed to send `sms keyword meaning` message: %w", err)
					}
				}
			}
			needNotice = true
		}
	} else if len(opts.Fields) == 2 && strings.HasSuffix(opts.Update.Message.Text, "ssm") {
		// 在数据库循环查找这个词
		for _, word := range UdoneseData.List {
			if strings.EqualFold(word.Word, opts.Fields[0]) {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					ParseMode: models.ParseModeHTML,
					DisableNotification: true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("chatID", opts.Update.Message.Chat.ID).
						Str("content", "sms keyword meaning").
						Msg(errt.SendMessage)
					handlerErr.Addf("failed to send `sms keyword meaning` message: %w", err)
				}
			}
		}
		needNotice = true
	}

	if needNotice {
		// 到这里就是没找到，提示没有
		botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
			DisableNotification: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Int64("chatID", opts.Update.Message.Chat.ID).
				Int("messageID", botMessage.ID).
				Str("content", "sms keyword no meaning").
				Msg(errt.SendMessage)
			handlerErr.Addf("failed to send `sms keyword no meaning` message: %w", err)
		} else {
			time.Sleep(time.Second * 10)
			_, err = opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
				ChatID: opts.Update.Message.Chat.ID,
				MessageIDs: []int{ botMessage.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", opts.Update.Message.Chat.ID).
					Int("messageID", botMessage.ID).
					Str("content", "sms keyword no meaning").
					Msg(errt.DeleteMessage)
				handlerErr.Addf("failed to delete `sms keyword no meaning` message: %w", err)
			}
		}
	}

	return handlerErr.Flat()
}

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "Udonese",
		Func: ReadUdonese,
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Udonese",
		Saver:  SaveUdonese,
		Loader: ReadUdonese,
	})
	plugin_utils.AddInlineHandlerPlugins(plugin_utils.InlineHandler{
		Command:     "sms",
		Handler:     udoneseInlineHandler,
		Description: "查询 Udonese 词典",
	})
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "udonese",
		Handler:      addUdoneseHandler,
	})
	plugin_utils.AddHandlerByChatIDPlugins(plugin_utils.HandlerByChatID{
		ChatID:     UdonGroupID,
		PluginName: "udoneseGroupHandler",
		Handler:    udoneseGroupHandler,
	})
	plugin_utils.AddCallbackQueryCommandPlugins(plugin_utils.CallbackQuery{
		CommandChar: "udonese",
		Handler:      udoneseCallbackHandler,
	})
}

func udoneseCallbackHandler(opts *handler_structs.SubHandlerParams) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Udonese").
		Str("funcName", "udoneseCallbackHandler").
		Logger()

	var handlerErr multe.MultiError

	if !utils.AnyContains(opts.Update.CallbackQuery.From.ID, UdoneseManagerIDs) {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.Update.CallbackQuery.ID,
			Text: "不可以！",
			ShowAlert: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("callbackQueryID", opts.Update.CallbackQuery.ID).
				Str("content", "udonese no edit permissions").
				Msg(errt.AnswerCallbackQuery)
			handlerErr.Addf("failed to send `udonese no edit permissions` inline result: %w", err)
		}
	} else {
		if opts.Update.CallbackQuery.Data == "udonese_done" {
			_, err := opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
				ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", opts.Update.CallbackQuery.Message.Message.Chat.ID).
					Int("messageID", opts.Update.CallbackQuery.Message.Message.ID).
					Str("content", "udonese keyword manage keyboard").
					Msg(errt.DeleteMessage)
				handlerErr.Addf("failed to delete `udonese keyword manage keyboard` inline result: %w", err)
			}
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "udonese_word_") {
			word := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "udonese_word_")
			var targetWord UdoneseWord
			for _, wordlist := range UdoneseData.List {
				if wordlist.Word == word {
					targetWord = wordlist
				}
			}

			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
				Text:      fmt.Sprintf("词: [ %s ]\n有 %d 个意思，已使用 %d 次\n", targetWord.Word, len(targetWord.MeaningList), targetWord.Used),
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int64("chatID", opts.Update.CallbackQuery.Message.Message.Chat.ID).
					Int("messageID", opts.Update.CallbackQuery.Message.Message.ID).
					Str("content", "udonese word meaning list").
					Msg(errt.EditMessageText)
				handlerErr.Addf("failed to edit message to `udonese keyword manage keyboard`: %w", err)
			}
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "udonese_meaning_") {
			wordAndIndex := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "udonese_meaning_")
			wordAndIndexList := strings.Split(wordAndIndex, "_")
			meanningIndex, err := strconv.Atoi(wordAndIndexList[1])
			if err != nil {
				logger.Error().
					Err(err).
					Str("callbackQueryData", opts.Update.CallbackQuery.Data).
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
					ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.Update.CallbackQuery.Message.Message.ID,
					Text: pendingMessage,
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
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
						Int64("chatID", opts.Update.CallbackQuery.Message.Message.Chat.ID).
						Int("messageID", opts.Update.CallbackQuery.Message.Message.ID).
						Str("content", "udonese meaning manage keyboard").
						Msg(errt.EditMessageText)
					handlerErr.Addf("failed to edit message to `udonese meaning manage keyboard`: %w", err)
				}
			}
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "udonese_delmeaning_") {
			wordAndIndex := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "udonese_delmeaning_")
			wordAndIndexList := strings.Split(wordAndIndex, "_")
			meanningIndex, err := strconv.Atoi(wordAndIndexList[1])
			if err != nil {
				logger.Error().
					Err(err).
					Str("callbackQueryData", opts.Update.CallbackQuery.Data).
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
						CallbackQueryID: opts.Update.CallbackQuery.ID,
						Text:            "删除意思时保存数据库失败，请重试或联系机器人管理员\n" + err.Error(),
						ShowAlert:       true,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "failed to save udonese data after delete meaning").
							Msg(errt.AnswerCallbackQuery)
						handlerErr.Addf("failed to send `failed to save udonese data after delete meaning` callback answer: %w", err)
					}
				} else {
					_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
						ChatID:    opts.Update.CallbackQuery.Message.Message.Chat.ID,
						MessageID: opts.Update.CallbackQuery.Message.Message.ID,
						Text:      fmt.Sprintf("词: [ %s ]\n有 %d 个意思，已使用 %d 次\n<blockquote>已删除 [ %s ] 词中的 [ %s ] 意思</blockquote>", targetWord.Word, len(targetWord.MeaningList), targetWord.Used, targetWord.Word, deletedMeaning),
						ParseMode: models.ParseModeHTML,
						ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("chatID", opts.Update.CallbackQuery.Message.Message.Chat.ID).
							Int("messageID", opts.Update.CallbackQuery.Message.Message.ID).
							Str("content", "udonese meaning manage keyboard after delete meaning").
							Msg(errt.EditMessageText)
						handlerErr.Addf("failed to edit message to `udonese meaning manage keyboard after delete meaning`: %w", err)
					}
				}
			}
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "udonese_delword_") {
			word := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "udonese_delword_")
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
					CallbackQueryID: opts.Update.CallbackQuery.ID,
					Text:            "删除词时保存数据库失败，请重试\n" + err.Error(),
					ShowAlert:       true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Msg(errt.AnswerCallbackQuery)
					handlerErr.Addf("failed to send `failed to save udonese data after delete word` callback answer: %w", err)
				}
			} else {
				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID:    opts.Update.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.Update.CallbackQuery.Message.Message.ID,
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
						Int64("chatID", opts.Update.CallbackQuery.Message.Message.Chat.ID).
						Int("messageID", opts.Update.CallbackQuery.Message.Message.ID).
						Str("content", "udonese word deleted notice").
						Msg(errt.EditMessageText)
					handlerErr.Addf("failed to edit message to `udonese word deleted notice`: %w", err)
				}
			}
		}
	}
	return handlerErr.Flat()
}
