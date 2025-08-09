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
	"trbot/utils/consts"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var KeywordDataList KeywordData = KeywordData{
	Chats: map[int64]KeywordChatList{},
	Users: map[int64]KeywordUserList{},
}
var KeywordDataErr  error
var KeywordDataPath string = filepath.Join(consts.YAMLDataBaseDir, "detectkeyword/", consts.YAMLFileName)

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "Detect Keyword",
		Func: func(ctx context.Context) error{
			err := readKeywordList(ctx)
			if err != nil {
				return err
			} else {
				plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
					Name:        "群组关键词检测",
					Description: "此功能可以检测群组中的每一条信息，当包含设定的关键词时，将会向用户发送提醒\n\n使用方法：\n首先将机器人添加至想要监听关键词的群组中，发送 /setkeyword 命令，等待机器人回应后点击下方的 “设定关键词” 按钮即可为自己添加要监听的群组\n\n设定关键词：您可以在对应的群组中直接发送 <code>/setkeyword 要设定的关键词</code> 来为该群组设定关键词\n或前往机器人聊天页面，发送 <code>/setkeyword</code> 命令后点击对应的群组或全局关键词按钮，根据提示来添加关键词",
					ParseMode:   models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
						{{
							Text: "将此机器人添加到群组或频道",
							URL:  fmt.Sprintf("https://t.me/%s?startgroup=detectkw", consts.BotMe.Username),
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
			}
			return nil
		},
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Detect Keyword",
		Loader: readKeywordList,
		Saver:  saveKeywordList,
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "setkeyword",
		ForChatType: []models.ChatType{
			models.ChatTypePrivate,
			models.ChatTypeGroup,
			models.ChatTypeSupergroup,
			models.ChatTypeChannel,
		},
		MessageHandler: addKeywordHandler,
	})
	plugin_utils.AddCallbackQueryHandlers([]plugin_utils.CallbackQuery{
		{
			CallbackDataPrefix:   "detectkw_g",
			CallbackQueryHandler: groupManageCallbackHandler,
		},
		{
			CallbackDataPrefix:   "detectkw_u",
			CallbackQueryHandler: userManageCallbackHandler,
		},
	}...)
	plugin_utils.AddSlashStartWithPrefixCommandHandlers(plugin_utils.SlashStartWithPrefixHandler{
		Name:           "detect keyword add chat to list",
		Prefix:         "detectkw",
		Argument:       "addgroup",
		ForChatType:    []models.ChatType{models.ChatTypePrivate},
		MessageHandler: startPrefixAddGroup,
	})
	plugin_utils.AddSlashStartCommandHandlers(plugin_utils.SlashStartHandler{
		Name:           "detect keyword add bot to chat",
		Argument:       "detectkw",
		ForChatType:    []models.ChatType{models.ChatTypeGroup, models.ChatTypeSupergroup},
		MessageHandler: startBotAddedToGroup,
	})
}

type KeywordData struct {
	Chats map[int64]KeywordChatList `yaml:"Chats"`
	Users map[int64]KeywordUserList `yaml:"Users"`
}

type KeywordChatList struct {
	ChatID       int64           `yaml:"ChatID"`
	ChatName     string          `yaml:"ChatName"`
	ChatUsername string          `yaml:"ChatUsername,omitempty"`
	ChatType     models.ChatType `yaml:"ChatType"`
	AddTime      string          `yaml:"AddTime"`
	InitByID     int64           `yaml:"InitByID"`
	IsDisable    bool            `yaml:"IsDisable,omitempty"`
	// 根据用户数量决定是否启用
	UsersID      []int64         `yaml:"UsersID"`
}

type KeywordUserList struct {
	UserID          int64         `yaml:"UserID"`
	AddTime         string        `yaml:"AddTime"`
	Limit           int           `yaml:"Limit"` // todo
	MentionCount    int           `yaml:"MentionCount,omitempty"`

	IsNotInit       bool          `yaml:"IsNotInit,omitempty"`
	IsDisable       bool          `yaml:"IsDisable,omitempty"`
	IsSilentNotice  bool          `yaml:"IsSilentNotice,omitempty"`
	IsIncludeSelf   bool          `yaml:"IsIncludeSelf,omitempty"`
	IsIncludeBot    bool          `yaml:"IsIncludeBot,omitempty"` // todo

	AddingChatID    int64         `yaml:"AddingChatID,omitempty"`
	GlobalKeyword   []string      `yaml:"GlobalKeyword"`
	WatchUser       []int64       `yaml:"WatchUser,omitempty"` // todo
	ChatsForUser    []ChatForUser `yaml:"ChatForUser"`
}

func (user KeywordUserList)enabledChatCount() (count int) {
	for _, v := range user.ChatsForUser {
		if !v.IsDisable { count++ }
	}
	return
}

func (user KeywordUserList)keywordCount() (count int) {
	for _, v := range user.ChatsForUser {
		count += len(v.Keyword)
	}
	return
}

func (user KeywordUserList)userStatus() string {
	var pendingMessage string
	if user.IsDisable {
		pendingMessage = "您已经全局禁用了此功能"
	} else {
		pendingMessage = fmt.Sprintf("您添加的群组中有 (%d/%d) 个处于启用状态\n您总共设定了 %d 个关键词",
			user.enabledChatCount(), len(user.ChatsForUser),
			user.keywordCount() + len(user.GlobalKeyword),
		)
	}

	return pendingMessage
}

func (user KeywordUserList)selectChat(keyword string) models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text:         "🌐 添加为全局关键词",
		CallbackData: fmt.Sprintf("detectkw_u_add_%d_%s", user.UserID, keyword),
	}})
	for _, chat := range user.ChatsForUser {
		targetChat := KeywordDataList.Chats[chat.ChatID]
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text:         "👥 " + targetChat.ChatName,
			CallbackData: fmt.Sprintf("detectkw_u_add_%d_%s", targetChat.ChatID, keyword),
		}})
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

func (user KeywordUserList)buildUserChatList() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	if !user.IsDisable {
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: fmt.Sprintf("全局关键词 %d 个", len(user.GlobalKeyword)),
			CallbackData:  fmt.Sprintf("detectkw_u_chat_%d", user.UserID),
		}})

		for _, chat := range user.ChatsForUser {
			var subchats []models.InlineKeyboardButton
			var targetChat = KeywordDataList.Chats[chat.ChatID]

			subchats = append(subchats, models.InlineKeyboardButton{
				Text: fmt.Sprintf("(%d) %s", len(chat.Keyword), targetChat.ChatName),
				CallbackData: fmt.Sprintf("detectkw_u_chat_%d", targetChat.ChatID),
			})

			if targetChat.IsDisable {
				subchats = append(subchats, models.InlineKeyboardButton{
					Text: "🚫 查看帮助",
					CallbackData: "detectkw_u_chatdisablebyadmin",
				})
			} else {
				subchats = append(subchats, models.InlineKeyboardButton{
					Text: "🔄 " + utils.TextForTrueOrFalse(chat.IsDisable, "当前已禁用 ❌", "当前已启用 ✅"),
					CallbackData: fmt.Sprintf("detectkw_u_switch_chat_%d", targetChat.ChatID),
				})
			}

			buttons = append(buttons, subchats)
		}

		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: "🔄 通知偏好：" + utils.TextForTrueOrFalse(user.IsSilentNotice, "🔇 无声通知", "🔉 有声通知"),
			CallbackData: "detectkw_u_noticeswitch",
		}})
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: "🔄 检测偏好：" + utils.TextForTrueOrFalse(user.IsIncludeSelf, "不排除自己的消息", "排除自己的消息"),
			CallbackData: "detectkw_u_selfswitch",
		}})
	}

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "🔄 全局状态：" + utils.TextForTrueOrFalse(user.IsDisable, "已禁用 ❌", "已启用 ✅"),
		CallbackData: "detectkw_u_globalswitch",
	}})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

func (user KeywordUserList)globalKeywordButtons() [][]models.InlineKeyboardButton {
	var buttons    [][]models.InlineKeyboardButton
	var tempbutton []models.InlineKeyboardButton

	for index, keyword := range user.GlobalKeyword {
		if index % 2 == 0 && index != 0 {
			buttons = append(buttons, tempbutton)
			tempbutton = []models.InlineKeyboardButton{}
		}
		tempbutton = append(tempbutton, models.InlineKeyboardButton{
			Text:         keyword,
			CallbackData: fmt.Sprintf("detectkw_u_kw_%d_%s", user.UserID, keyword),
		})
	}
	if len(tempbutton) != 0 {
		buttons = append(buttons, tempbutton)
	}

	return buttons
}

type ChatForUser struct {
	ChatID          int64    `yaml:"ChatID"`
	IsDisable       bool     `yaml:"IsDisable,omitempty"`
	IsConfirmDelete bool     `yaml:"IsConfirmDelete,omitempty"` // todo
	Keyword         []string `yaml:"Keyword"`
}

func (chat ChatForUser)keywordButtons() [][]models.InlineKeyboardButton {
	var buttons    [][]models.InlineKeyboardButton
	var tempbutton []models.InlineKeyboardButton

	for index, keyword := range chat.Keyword {
		if index % 2 == 0 && index != 0 {
			buttons = append(buttons, tempbutton)
			tempbutton = []models.InlineKeyboardButton{}
		}
		tempbutton = append(tempbutton, models.InlineKeyboardButton{
			Text: keyword,
			CallbackData: fmt.Sprintf("detectkw_u_kw_%d_%s", chat.ChatID, keyword),
		})
	}
	if len(tempbutton) != 0 {
		buttons = append(buttons, tempbutton)
	}

	return buttons
}

func readKeywordList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "ReadKeywordList").
		Logger()

	err := yaml.LoadYAML(KeywordDataPath, &KeywordDataList)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", KeywordDataPath).
				Msg("Not found keyword list file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(KeywordDataPath, &KeywordDataList)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", KeywordDataPath).
					Msg("Failed to create empty keyword list file")
				KeywordDataErr = fmt.Errorf("failed to create empty keyword list file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", KeywordDataPath).
				Msg("Failed to load keyword list file")
			KeywordDataErr = fmt.Errorf("failed to load keyword list file: %w", err)
		}
	} else {
		KeywordDataErr = nil
	}

	buildListenList()
	return KeywordDataErr
}

func saveKeywordList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "SaveKeywordList").
		Logger()

	err := yaml.SaveYAML(KeywordDataPath, &KeywordDataList)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", KeywordDataPath).
			Msg("Failed to save udonese list")
		KeywordDataErr = fmt.Errorf("failed to save udonese list: %w", err)
	} else {
		KeywordDataErr = nil
	}

	return KeywordDataErr
}

func addKeywordHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "addKeywordHandler").
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if opts.Message.Chat.Type == models.ChatTypePrivate {
		// 与机器人的私聊对话
		user := KeywordDataList.Users[opts.Message.From.ID]
		if user.AddTime == "" {
			// 初始化用户
			user = KeywordUserList{
				UserID:  opts.Message.From.ID,
				AddTime: time.Now().Format(time.RFC3339),
				Limit:   50,
			}
			KeywordDataList.Users[opts.Message.From.ID] = user
			err := saveKeywordList(opts.Ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to init user and save keyword list")
				return handlerErr.Addf("failed to init user and save keyword list: %w", err).Flat()
			}
		}

		// 用户没有添加任何群组
		if len(user.ChatsForUser) == 0 {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            "您还没有添加任何群组，请在群组中使用 /setkeyword 命令来记录群组\n若发送信息后没有回应，请检查机器人是否在对应群组中",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "no group for user notice").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "no group for user notice", err)
			}
		} else {
			if len(opts.Fields) == 1 {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            user.userStatus(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					ParseMode:       models.ParseModeHTML,
					ReplyMarkup:     user.buildUserChatList(),
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "user group list keyboard").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "user group list keyboard", err)
				}
			} else {
				// 限制关键词长度
				if len(opts.Fields[1]) > 30 {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:          opts.Message.Chat.ID,
						Text:            "抱歉，单个关键词长度不能超过 30 个英文字符",
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int("length", len(opts.Fields[1])).
							Str("content", "keyword is too long").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "keyword is too long", err)
					}
				} else {
					if user.AddingChatID != 0 {
						keyword := strings.ToLower(opts.Fields[1])
						var pendingMessage string
						var button models.ReplyMarkup
						var isKeywordExist bool

						// 判断是全局关键词还是群组关键词
						if user.AddingChatID == user.UserID {
							// 全局关键词
							for _, k := range user.GlobalKeyword {
								if k == keyword {
									isKeywordExist = true
									break
								}
							}
							if !isKeywordExist {
								logger.Debug().
									Str("globalKeyword", keyword).
									Msg("User add a global keyword")
								user.GlobalKeyword = append(user.GlobalKeyword, keyword)
								KeywordDataList.Users[user.UserID] = user
								err := saveKeywordList(opts.Ctx)
								if err != nil {
									logger.Error().
										Err(err).
										Str("globalKeyword", keyword).
										Msg("Failed to add global keyword and save keyword list")
									return handlerErr.Addf("failed to add global keyword and save keyword list: %w", err).Flat()
								}
								pendingMessage = fmt.Sprintf("已添加全局关键词 [ %s ], 您可以继续添加更多全局关键词", keyword)
							} else {
								pendingMessage = fmt.Sprintf("此全局关键词 [ %s ] 已存在，您可以继续添加其他全局关键词", keyword)
							}

						} else {
							var chatForUser ChatForUser
							var chatForUserIndex int
							for i, c := range user.ChatsForUser {
								if c.ChatID == user.AddingChatID {
									chatForUser = c
									chatForUserIndex = i
								}
							}
							targetChat := KeywordDataList.Chats[chatForUser.ChatID]

							// 群组关键词
							for _, k := range chatForUser.Keyword {
								if k == keyword {
									isKeywordExist = true
									break
								}
							}
							if !isKeywordExist {
								logger.Debug().
									Str("keyword", keyword).
									Msg("User add a keyword to chat")
								chatForUser.Keyword = append(chatForUser.Keyword, keyword)
								user.ChatsForUser[chatForUserIndex] = chatForUser
								KeywordDataList.Users[user.UserID] = user
								err := saveKeywordList(opts.Ctx)
								if err != nil {
									logger.Error().
										Err(err).
										Str("keyword", keyword).
										Msg("Failed to add keyword and save keyword list")
									return handlerErr.Addf("failed to add keyword and save keyword list: %w", err).Flat()
								}

								pendingMessage = fmt.Sprintf("已为 <a href=\"https://t.me/c/%s/\">%s</a> 群组添加关键词 [ %s ]，您可以继续向此群组添加更多关键词\n", utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName, keyword)
							} else {
								pendingMessage = fmt.Sprintf("此关键词 [ %s ] 已存在于 <a href=\"https://t.me/c/%s/\">%s</a> 群组中，您可以继续向此群组添加其他关键词", keyword, utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName)
							}
						}
						if isKeywordExist {
							button = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
								Text:         "✅ 完成",
								CallbackData: "detectkw_u_finish",
							}}}}
						} else {
							button = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
								{
									Text:         "↩️ 撤销操作",
									CallbackData: fmt.Sprintf("detectkw_u_undo_%d_%s", user.AddingChatID, opts.Fields[1]),
								},
								{
									Text:         "✅ 完成",
									CallbackData: "detectkw_u_finish",
								},
							}}}
						}

						_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID: opts.Message.Chat.ID,
							Text: pendingMessage,
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
							ParseMode: models.ParseModeHTML,
							ReplyMarkup: button,
						})
						if err != nil {
							logger.Error().
								Err(err).
								Str("content", "keyword added notice").
								Msg(flaterr.SendMessage.Str())
							handlerErr.Addt(flaterr.SendMessage, "keyword added notice", err)
						}
					} else {
						_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID:          opts.Message.Chat.ID,
							Text:            "您还没有选定要将关键词添加到哪个群组，请在下方挑选一个您已经添加的群组",
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
							ParseMode:       models.ParseModeHTML,
							ReplyMarkup:     user.selectChat(opts.Fields[1]),
						})
						if err != nil {
							logger.Error().
								Err(err).
								Str("content", "keyword adding select keyboard").
								Msg(flaterr.SendMessage.Str())
							handlerErr.Addt(flaterr.SendMessage, "keyword adding select keyboard", err)
						}
					}
				}

			}
		}
	} else {
		// 在群组中直接使用 /setkeyword 命令
		chat := KeywordDataList.Chats[opts.Message.Chat.ID]
		if chat.IsDisable {
			// 此功能已被管理员手动禁用
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            "群组管理员已禁用关键词功能，您可以询问管理员以获取更多信息",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				ParseMode:       models.ParseModeHTML,
				ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text:         "管理此功能（管理员）",
					CallbackData: "detectkw_g",
				}}}},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "disabled by admins").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "disabled by admins", err)
			}
		} else {
			if chat.AddTime == "" {
				var initByID int64

				if opts.Message.From != nil {
					initByID = opts.Message.From.ID
				} else if opts.Message.SenderChat != nil {
					initByID = opts.Message.SenderChat.ID
				}

				// 初始化群组
				chat = KeywordChatList{
					ChatID:       opts.Message.Chat.ID,
					ChatName:     opts.Message.Chat.Title,
					ChatUsername: opts.Message.Chat.Username,
					ChatType:     opts.Message.Chat.Type,
					AddTime:      time.Now().Format(time.RFC3339),
					InitByID:     initByID,
				}
				KeywordDataList.Chats[opts.Message.Chat.ID] = chat
				err := saveKeywordList(opts.Ctx)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetChatDict(&opts.Message.Chat)).
						Msg("Failed to init chat and save keyword list")
					return handlerErr.Addf("failed to init chat and save keyword list: %w", err).Flat()
				}
			}
			if len(opts.Fields) == 1 {
				// 只有一个 /setkeyword 命令
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:          opts.Message.Chat.ID,
					Text:            "已记录群组，点击下方左侧按钮来设定监听关键词\n若您是群组的管理员，您可以点击右侧的按钮来管理此功能",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "设定关键词",
							URL:  fmt.Sprintf("https://t.me/%s?start=detectkw_addgroup_%d", consts.BotMe.Username, opts.Message.Chat.ID),
						},
						{
							Text:         "管理此功能",
							CallbackData: "detectkw_g",
						},
					}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "group record link button").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "group record link button", err)
				}
			} else if opts.Message.Chat.Type != models.ChatTypeChannel {
				// 限制关键词长度
				if len(opts.Fields[1]) > 30 {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:          opts.Message.Chat.ID,
						Text:            "抱歉，单个关键词长度不能超过 30 个英文字符",
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int("keywordLength", len(opts.Fields[1])).
							Str("content", "keyword is too long").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "keyword is too long", err)
					}
				} else {
					user := KeywordDataList.Users[opts.Message.From.ID]

					if user.AddTime == "" {
						// 初始化用户
						user = KeywordUserList{
							UserID:    opts.Message.From.ID,
							AddTime:   time.Now().Format(time.RFC3339),
							Limit:     50,
							IsNotInit: true,
						}
						KeywordDataList.Users[opts.Message.From.ID] = user
						err := saveKeywordList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Msg("Failed to add a not init user and save keyword list")
							return handlerErr.Addf("failed to add a not init user and save keyword list: %w", err).Flat()
						}
					}

					var chatForUser      ChatForUser
					var chatForUserIndex int
					var isChatAdded      bool

					for index, keyword := range user.ChatsForUser {
						if keyword.ChatID == chat.ChatID {
							chatForUser = keyword
							chatForUserIndex = index
							isChatAdded = true
							break
						}
					}
					if !isChatAdded {
						logger.Debug().Msg("User add a chat to listen list by set keyword in group")
						chatForUser = ChatForUser{
							ChatID: chat.ChatID,
						}
						user.ChatsForUser = append(user.ChatsForUser, chatForUser)
						KeywordDataList.Users[user.UserID] = user
						err := saveKeywordList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Msg("Failed to add chat to user listen list and save keyword list")
							return handlerErr.Addf("failed to add chat to user listen list and save keyword list: %w", err).Flat()
						}
					}

					keyword := strings.ToLower(opts.Fields[1])
					var pendingMessage string
					var isKeywordExist bool

					for _, k := range chatForUser.Keyword {
						if k == keyword {
							isKeywordExist = true
							break
						}
					}

					if isKeywordExist {
						if user.IsNotInit {
							pendingMessage = "您已经添加过这个关键词了。请点击下方的按钮来初始化您的账号以使用此功能"
						} else {
							pendingMessage = "您已经添加过这个关键词了，您可以继续向此群组添加其他关键词"
						}
					} else {
						logger.Debug().
							Str("keyword", keyword).
							Msg("User add a keyword to chat")
						chatForUser.Keyword = append(chatForUser.Keyword, keyword)
						user.ChatsForUser[chatForUserIndex] = chatForUser
						KeywordDataList.Users[user.UserID] = user
						err := saveKeywordList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Str("keyword", keyword).
								Msg("Failed to add keyword and save keyword list")
							return handlerErr.Addf("failed to add keyword and save keyword list: %w", err).Flat()
						}
						if user.IsNotInit {
							pendingMessage = fmt.Sprintf("已将 [ %s ] 添加到您的关键词列表\n<blockquote>若要在检测到关键词时收到提醒，请点击下方的按钮来初始化您的账号</blockquote>", keyword)
						} else {
							pendingMessage = fmt.Sprintf("已将 [ %s ] 添加到您的关键词列表，您可以继续向此群组添加更多关键词", keyword)
						}
					}

					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:          opts.Message.Chat.ID,
						Text:            pendingMessage,
						ParseMode:       models.ParseModeHTML,
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
							Text: "管理关键词",
							URL:  fmt.Sprintf("https://t.me/%s?start=detectkw_addgroup_%d", consts.BotMe.Username, chat.ChatID),
						}}}},
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "keyword added notice").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "keyword added notice", err)
					}
				}
			}
		}
	}
	return handlerErr.Flat()
}

func addKeywordStateHandler(opts *handler_params.Message) error {
	if opts.Message == nil { return nil }
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "addKeywordStateHandler").
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if opts.Message.Text != "" {
		keyword := strings.ToLower(strings.Fields(opts.Message.Text)[0])

		if len(keyword) > 30 {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            "抱歉，单个关键词长度不能超过 30 个英文字符",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Int("length", len(keyword)).
					Str("content", "keyword is too long").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "keyword is too long", err)
			}
		} else {
			user := KeywordDataList.Users[opts.Message.From.ID]
			var pendingMessage string
			var button models.ReplyMarkup
			var isKeywordExist bool

			if user.AddingChatID == user.UserID {
				for _, k := range user.GlobalKeyword {
					if k == keyword {
						pendingMessage = "关键词已存在"
						isKeywordExist = true
						break
					}
				}
				if !isKeywordExist {
					logger.Debug().
						Str("globalKeyword", keyword).
						Msg("User add a global keyword")
					user.GlobalKeyword = append(user.GlobalKeyword, keyword)
					KeywordDataList.Users[user.UserID] = user
					err := saveKeywordList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Str("globalKeyword", keyword).
							Msg("Failed to add global keyword and save keyword list")
						handlerErr.Addf("failed to add global keyword and save keyword list: %w", err)
					}
					pendingMessage = fmt.Sprintf("已添加全局关键词: [ %s ]，您可以继续添加更多全局关键词", keyword)
				} else {
					pendingMessage = fmt.Sprintf("此全局关键词 [ %s ] 已存在，您可以继续添加其他全局关键词", keyword)
				}
			} else {
				var chatForUser ChatForUser
				var chatForUserIndex int
				for i, c := range user.ChatsForUser {
					if c.ChatID == user.AddingChatID {
						chatForUser = c
						chatForUserIndex = i
					}
				}
				targetChat := KeywordDataList.Chats[chatForUser.ChatID]

				// 群组关键词
				for _, k := range chatForUser.Keyword {
					if k == keyword {
						isKeywordExist = true
						break
					}
				}

				if isKeywordExist {
					pendingMessage = fmt.Sprintf("此关键词 [ %s ] 已存在于 <a href=\"https://t.me/c/%s/\">%s</a> 群组中，您可以继续向此群组添加其他关键词\n或发送 /cancel 命令来取消操作", keyword, utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName)
				} else {
					logger.Debug().
						Str("keyword", keyword).
						Msg("User add a keyword to chat")
					chatForUser.Keyword = append(chatForUser.Keyword, keyword)
					user.ChatsForUser[chatForUserIndex] = chatForUser
					KeywordDataList.Users[user.UserID] = user
					err := saveKeywordList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Str("keyword", keyword).
							Msg("Failed to add keyword and save keyword list")
						return handlerErr.Addf("failed to add keyword and save keyword list: %w", err).Flat()
					}
					pendingMessage = fmt.Sprintf("已为 <a href=\"https://t.me/c/%s/\">%s</a> 群组添加关键词 [ %s ]，您可以继续向此群组添加更多关键词\n或发送 /cancel 命令来取消操作", utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName, keyword)
				}
			}

			if isKeywordExist {
				button = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text:         "⬅️ 停止添加关键词",
					CallbackData: "detectkw_u_finish",
				}}}}
			} else {
				button = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
					{
						Text:         "↩️ 撤销操作",
						CallbackData: fmt.Sprintf("detectkw_u_undo_%d_%s", user.AddingChatID, opts.Message.Text),
					},
					{
						Text:         "⬅️ 停止添加关键词",
						CallbackData: "detectkw_u_finish",
						// CallbackData: fmt.Sprintf("detectkw_u_chat_%d", user.AddingChatID),
					},
				}}}
			}

			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            pendingMessage,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				ParseMode:       models.ParseModeHTML,
				ReplyMarkup:     button,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "keyword added notice").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "keyword added notice", err)
			}
		}
	} else {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Message.Chat.ID,
			Text: "请输入有效的关键词",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text:         "⬅️ 停止添加关键词",
				CallbackData: "detectkw_u_finish",
			}}}},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "keyword invalid notice").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "keyword invalid notice", err)
		}
		// plugin_utils.EditStateHandler(opts.Message.Chat.ID, 1, nil)
	}

	return handlerErr.Flat()
}

// 重新构建 by chat ID handler 列表
func buildListenList() {
	// 遍历群组列表，清理群组的用户 ID 列表，再删除 by chat ID handler
	for index, chat := range KeywordDataList.Chats {
		chat.UsersID = []int64{}
		KeywordDataList.Chats[index] = chat
		plugin_utils.RemoveHandlerByChatIDHandler(chat.ChatID, "detect_keyword")
	}

	// 遍历用户列表，若用户启用了该群组，则将用户 ID 添加到群组的用户 ID 列表
	for _, user := range KeywordDataList.Users {
		if !user.IsDisable {
			for _, keywordForChat := range user.ChatsForUser {
				if !keywordForChat.IsDisable {
					chat := KeywordDataList.Chats[keywordForChat.ChatID]
					chat.UsersID = append(chat.UsersID, user.UserID)
					KeywordDataList.Chats[keywordForChat.ChatID] = chat
				}
			}
		}
	}

	// 遍历群组列表，判断用户 ID 列表数量不为 0 时，添加该群组的 by chat ID handler
	for _, chat := range KeywordDataList.Chats {
		if !chat.IsDisable && len(chat.UsersID) > 0 {
			plugin_utils.AddHandlerByMessageChatIDHandlers(plugin_utils.ByMessageChatIDHandler{
				ForChatID:      chat.ChatID,
				PluginName:     "detect_keyword",
				MessageHandler: keywordDetector,
			})
		}
	}
}

func keywordDetector(opts *handler_params.Message) error {
	var handlerErr flaterr.MultErr
	var text string

	if opts.Message.Caption != "" {
		text = opts.Message.Caption
	} else if opts.Message.Text != "" {
		text = opts.Message.Text
	} else {
		// 没有文字和 caption 直接返回
		return nil
	}

	// 遍历该群组中启用此功能的用户 ID
	for _, userID := range KeywordDataList.Chats[opts.Message.Chat.ID].UsersID {
		// 获取用户信息，开始匹配关键词
		user := KeywordDataList.Users[userID]
		if !user.IsDisable && !user.IsNotInit {
			// 如果用户设定排除了自己发送的消息，则跳过
			if !user.IsIncludeSelf && opts.Message.From != nil && opts.Message.From.ID == userID { continue }

			// 用户为单独群组设定的关键词
			for _, userChatKeywordList := range user.ChatsForUser {
				// 判断是否是此群组
				if userChatKeywordList.ChatID == opts.Message.Chat.ID {
					for _, keyword := range userChatKeywordList.Keyword {
						if contain.SubStringCaseInsensitive(keyword, text) {
							return handlerErr.Add(notifyUser(opts.Ctx, opts.Thebot, opts.Message, &user, keyword, text, false)).Flat()
						}
					}
				}
			}
			// 用户全局设定的关键词
			for _, userGlobalKeyword := range user.GlobalKeyword {
				if contain.SubStringCaseInsensitive(userGlobalKeyword, text) {
					return handlerErr.Add(notifyUser(opts.Ctx, opts.Thebot, opts.Message, &user, userGlobalKeyword, text, true)).Flat()
				}
			}
		}
	}
	return handlerErr.Flat()
}

func notifyUser(ctx context.Context, thebot *bot.Bot, message *models.Message, user *KeywordUserList, keyword, text string, isGlobalKeyword bool) error {
	var handlerErr flaterr.MultErr

	_, err := thebot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: user.UserID,
		Text:   fmt.Sprintf("在 <a href=\"https://t.me/c/%s/\">%s</a> 中\n来自 %s 的消息\n触发了%s关键词 [ %s ]\n<blockquote expandable>%s</blockquote>",
			utils.RemoveIDPrefix(message.Chat.ID), message.Chat.Title, utils.GetMessageFromHyperLink(message, models.ParseModeHTML),
			utils.TextForTrueOrFalse(isGlobalKeyword, "全局", "群组"), keyword, utils.IgnoreHTMLTags(text),
		),
		ParseMode:           models.ParseModeHTML,
		DisableNotification: user.IsSilentNotice,
		ReplyMarkup:         &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "前往查看",
			URL:  fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(message.Chat.ID), message.ID),
		}}}},
	})
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Err(err).
			Str("pluginName", "DetectKeyword").
			Str("funcName", "notifyUser").
			Dict(utils.GetChatDict(&message.Chat)).
			Int64("userID", user.UserID).
			Str("keyword", keyword).
			Str("content", "keyword detected notice to user").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "keyword detected notice to user", err)
	}
	user.MentionCount++
	KeywordDataList.Users[user.UserID] = *user

	return handlerErr.Flat()
}

func groupManageCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "groupManageCallbackHandler").
		Dict(utils.GetChatDict(&opts.CallbackQuery.Message.Message.Chat)).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Logger()

	var handlerErr flaterr.MultErr

	if !utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.CallbackQuery.Message.Message.Chat.ID, opts.CallbackQuery.From.ID) {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "您没有权限修改此配置",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "no permission to change group functions").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "no permission to change group functions", err)
		}
	} else if opts.CallbackQuery.Data == "detectkw_g_switch" {
		chat := KeywordDataList.Chats[opts.CallbackQuery.Message.Message.Chat.ID]
		// 群组里的全局开关，是否允许群组内用户使用这个功能，优先级最高
		chat.IsDisable = !chat.IsDisable
		KeywordDataList.Chats[opts.CallbackQuery.Message.Message.Chat.ID] = chat
		buildListenList()
		err := saveKeywordList(opts.Ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to change group switch and save keyword list")
			handlerErr.Addf("failed to change group switch and save keyword list: %w", err)
			_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
				CallbackQueryID: opts.CallbackQuery.ID,
				Text:            "更改此选项时发生错误",
				ShowAlert:       true,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "failed to change group switch notice").
					Msg(flaterr.AnswerCallbackQuery.Str())
				handlerErr.Addt(flaterr.AnswerCallbackQuery, "failed to change group switch notice", err)
			}
		} else {
			var buttons [][]models.InlineKeyboardButton

			if chat.IsDisable {
				buttons = [][]models.InlineKeyboardButton{
					{{
						Text:         "🔄 当前状态: 已禁用 ❌",
						CallbackData: "detectkw_g_switch",
					}},
				}
			} else {
				buttons = [][]models.InlineKeyboardButton{
					{{
						Text: "设定关键词",
						URL:  fmt.Sprintf("https://t.me/%s?start=detectkw_addgroup_%d", consts.BotMe.Username, opts.CallbackQuery.Message.Message.Chat.ID),
					}},
					{{
						Text:         "🔄 当前状态: 已启用 ✅",
						CallbackData: "detectkw_g_switch",
					}},
				}
			}

			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID:   opts.CallbackQuery.Message.Message.ID,
				Text:        fmt.Sprintf("消息关键词检测\n%s\n\n当前群组中有 %d 个用户启用了此功能",utils.TextForTrueOrFalse(chat.IsDisable, "已为当前群组关闭关键词检测功能，已设定了关键词的用户将无法再收到此群组的提醒", "此功能允许用户设定一些关键词，当机器人检测到群组内的消息包含用户设定的关键词时，向用户发送提醒"), len(chat.UsersID)),
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: buttons},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "group function manager keyboard").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "group function manager keyboard", err)
			}
		}
	}

	return handlerErr.Flat()
}

func userManageCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "userManageCallbackHandler").
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Str("callbackQueryData", opts.CallbackQuery.Data).
		Logger()

	var handlerErr flaterr.MultErr

	var needEdit bool = true
	var needSave bool = true

	user := KeywordDataList.Users[opts.CallbackQuery.From.ID]

	switch opts.CallbackQuery.Data {
	case "detectkw_u_globalswitch":       // 功能全局开关
		user.IsDisable = !user.IsDisable
	case "detectkw_u_noticeswitch":       // 是否静默通知
		user.IsSilentNotice = !user.IsSilentNotice
	case "detectkw_u_selfswitch":         // 是否检测自己发送的消息
		user.IsIncludeSelf = !user.IsIncludeSelf
	case "detectkw_u_finish":             // 停止添加群组关键词
		user.AddingChatID = 0
		plugin_utils.RemoveStateHandler(user.UserID)
	case "detectkw_u_chatdisablebyadmin": // 目标群组的管理员为群组关闭了此功能
		needEdit = false
		needSave = false
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "此群组中的管理员禁用了此功能，因此，您无法再收到来自该群组的关键词提醒，您可以询问该群组的管理员是否可以重新开启这个功能",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "this group is disable by admins").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "this group is disable by admins", err)
		}
	default:
		if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_undo_") || strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_delkw_") {
			needEdit = false
			// 撤销添加或删除关键词
			var chatIDAndKeyword string
			if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_undo_") {
				chatIDAndKeyword = strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_undo_")
			} else {
				chatIDAndKeyword = strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_delkw_")
			}
			chatIDAndKeywordList := strings.Split(chatIDAndKeyword, "_")
			chatID, err := strconv.ParseInt(chatIDAndKeywordList[0], 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse chat ID when user undo add or delete a keyword")
				return handlerErr.Addf("failed to parse chat ID when user undo add or delete a keyword: %w", err).Flat()
			} else {
				// 删除关键词过程
				if chatID == user.UserID {
					// 全局关键词
					var tempKeyword []string
					for _, keyword := range user.GlobalKeyword {
						if keyword != chatIDAndKeywordList[1] {
							tempKeyword = append(tempKeyword, keyword)
						}
					}
					user.GlobalKeyword = tempKeyword
				} else {
					// 群组关键词
					for index, chatForUser := range user.ChatsForUser {
						if chatForUser.ChatID == chatID {
							var tempKeyword []string
							for _, keyword := range chatForUser.Keyword {
								if keyword != chatIDAndKeywordList[1] {
									tempKeyword = append(tempKeyword, keyword)
								}
							}
							chatForUser.Keyword = tempKeyword
							user.ChatsForUser[index] = chatForUser
						}
					}
				}

				if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_undo_") {
					_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
						ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
						MessageID:   opts.CallbackQuery.Message.Message.ID,
						Text:        fmt.Sprintf("已取消添加 [ %s ] 关键词，您可以继续添加其他关键词", chatIDAndKeywordList[1]),
						ParseMode:   models.ParseModeHTML,
						ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{ {
							Text:         "✅ 完成",
							CallbackData: "detectkw_u_finish",
						}}}},
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "add keyword has been canceled notice").
							Msg(flaterr.EditMessageText.Str())
						handlerErr.Addt(flaterr.EditMessageText, "add keyword has been canceled notice", err)
					}
				} else {
					var buttons [][]models.InlineKeyboardButton
					var pendingMessage string

					if chatID == user.UserID {
						buttons = user.globalKeywordButtons()
						pendingMessage = fmt.Sprintf("已删除 [ %s ] 关键词\n\n您当前设定了 %d 个全局关键词", chatIDAndKeywordList[1], len(buttons))
					} else {
						for _, chat := range user.ChatsForUser {
							if chat.ChatID == chatID {
								buttons = chat.keywordButtons()
								pendingMessage = fmt.Sprintf("已删除 [ %s ] 关键词\n\n您当前为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定了 %d 个关键词", chatIDAndKeywordList[1], utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName, len(buttons))
								break
							}
						}
					}

					buttons = append(buttons, []models.InlineKeyboardButton{
						{
							Text: "⬅️ 返回主菜单",
							CallbackData: "detectkw_u",
						},
						{
							Text: "➕ 添加关键词",
							CallbackData: fmt.Sprintf("detectkw_u_adding_%d", chatID),
						},
					})

					_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
						ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
						MessageID:   opts.CallbackQuery.Message.Message.ID,
						Text:        pendingMessage,
						ParseMode:   models.ParseModeHTML,
						ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: buttons },
					})
					if err != nil {
						logger.Error().
							Err(err).
							Str("content", "keyword list keyboard with deleted keyword notice").
							Msg(flaterr.EditMessageText.Str())
					}
				}
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_adding_") {
			needEdit = false
			// 设定要往哪个群组里添加关键词
			chatID := strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_adding_")
			chatID_int64, err := strconv.ParseInt(chatID, 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse chat ID when user selecting chat to add keyword")
				return handlerErr.Addf("failed to parse chat ID when user selecting chat to add keyword: %w", err).Flat()
			} else {
				user.AddingChatID = chatID_int64
				var pendingMessage string
				if chatID_int64 == user.UserID {
					pendingMessage = "已将全局关键词设为添加关键词的目标，请继续发送您要添加单个的全局关键词\n或发送 /cancel 命令来取消操作"
				} else {
					pendingMessage = fmt.Sprintf("已将 <a href=\"https://t.me/c/%s/\">%s</a> 群组设为添加关键词的目标群组，请继续发送您要添加单个的群组关键词\n或发送 /cancel 命令来取消操作", utils.RemoveIDPrefix(chatID_int64), KeywordDataList.Chats[chatID_int64].ChatName)
				}

				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					Text:      pendingMessage,
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text:         "⬅️ 停止添加关键词",
						CallbackData: "detectkw_u_finish",
					}}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "ready to add keyword notice").
						Msg(flaterr.EditMessageText.Str())
					handlerErr.Addt(flaterr.EditMessageText, "ready to add keyword notice", err)
				} else {
					plugin_utils.AddStateHandler(plugin_utils.StateHandler{
						ForChatID: opts.CallbackQuery.Message.Message.Chat.ID,
						PluginName: "addKeywordState",
						Remaining: -1,
						MessageHandler: addKeywordStateHandler,
					})
				}
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_switch_chat_") {
			// 用户启用或禁用某个群组的关键词检测开关
			chatID, err := strconv.ParseInt(strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_switch_chat_"), 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse chat ID when user change the group switch")
				return handlerErr.Addf("failed to parse chat ID when user change the group switch: %w", err).Flat()
			} else {
				for index, chat := range user.ChatsForUser {
					if chat.ChatID == chatID {
						chat.IsDisable = !chat.IsDisable
					}
					user.ChatsForUser[index] = chat
				}
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_chat_") {
			needEdit = false
			needSave = false
			// 显示某个群组的关键词列表
			chatID, err := strconv.ParseInt(strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_chat_"), 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse chat ID when user wanna manage keyword for group")
				return handlerErr.Addf("failed to parse chat ID when user wanna manage keyword for group: %w", err).Flat()
			} else {
				var buttons [][]models.InlineKeyboardButton
				var pendingMessage string

				if chatID == user.UserID {
					// 全局关键词
					buttons = user.globalKeywordButtons()
					if len(buttons) == 0 {
						pendingMessage = "您没有设定任何全局关键词\n点击下方按钮来添加全局关键词"
					} else {
						pendingMessage = fmt.Sprintf("您当前设定了 %d 个全局关键词\n<blockquote expandable>全局关键词将对您添加的全部群组生效，但在部分情况下，全局关键词不会生效：\n- 您手动将群组设定为禁用状态\n- 对应群组的管理员为该群组关闭了此功能</blockquote>", len(buttons))
					}
				} else {
					// 为群组设定的关键词
					for _, chat := range KeywordDataList.Users[opts.CallbackQuery.From.ID].ChatsForUser {
						if chat.ChatID == chatID {
							buttons = chat.keywordButtons()
							if len(buttons) == 0 {
								pendingMessage = fmt.Sprintf("当前群组 <a href=\"https://t.me/c/%s/\">%s</a> 没有关键词\n点击下方按钮来为此群组添加关键词", utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName)
							} else {
								pendingMessage = fmt.Sprintf("您当前为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定了 %d 个关键词", utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName, len(buttons))
							}
							break
						}
					}

				}

				buttons = append(buttons, []models.InlineKeyboardButton{
					{
						Text:         "⬅️ 返回主菜单",
						CallbackData: "detectkw_u",
					},
					{
						Text:         "➕ 添加关键词",
						CallbackData: fmt.Sprintf("detectkw_u_adding_%d", chatID),
					},
				})

				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID:   opts.CallbackQuery.Message.Message.ID,
					Text:        pendingMessage,
					ParseMode:   models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: buttons },
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "group keyword list keyboard").
						Msg(flaterr.EditMessageText.Str())
					handlerErr.Addt(flaterr.EditMessageText, "group keyword list keyboard", err)
				}
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_kw_") {
			needEdit = false
			needSave = false
			// 管理一个关键词
			chatIDAndKeyword := strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_kw_")
			chatIDAndKeywordList := strings.Split(chatIDAndKeyword, "_")
			chatID, err := strconv.ParseInt(chatIDAndKeywordList[0], 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse chat ID when user wanna manage a keyword")
				return handlerErr.Addf("failed to parse chat ID when user wanna manage a keyword: %w", err).Flat()
			} else {
				var pendingMessage string

				if chatID == user.UserID {
					pendingMessage = fmt.Sprintf("[ %s ] 是您设定的全局关键词", chatIDAndKeywordList[1])
				} else {
					pendingMessage = fmt.Sprintf("[ %s ] 是为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定的关键词", chatIDAndKeywordList[1], utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName)
				}

				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID:   opts.CallbackQuery.Message.Message.ID,
					Text:        pendingMessage,
					ParseMode:   models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text:         "⬅️ 返回",
							CallbackData: "detectkw_u_chat_" + chatIDAndKeywordList[0],
						},
						{
							Text:         "❌ 删除此关键词",
							CallbackData: "detectkw_u_delkw_" + chatIDAndKeyword,
						},
					}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "keyword manager keyboard").
						Msg(flaterr.EditMessageText.Str())
					handlerErr.Addt(flaterr.EditMessageText, "keyword manager keyboard", err)
				}
			}
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_add_") {
			needEdit = false
			// 在未设定添加群组时直接发送 `/setkeyword 关键词` 显示的键盘中的按钮，见 KeywordUserList.selectChat 方法
			chatIDAndKeyword := strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_add_")
			chatIDAndKeywordList := strings.Split(chatIDAndKeyword, "_")
			chatID, err := strconv.ParseInt(chatIDAndKeywordList[0], 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parse chat ID when user wanna add a keyword")
				return handlerErr.Addf("failed to parse chat ID when user wanna add a keyword: %w", err).Flat()
			} else {
				var pendingMessage string
				var button models.ReplyMarkup
				var isKeywordExist bool

				if chatID == user.UserID {
					// 全局关键词
					for _, k := range user.GlobalKeyword {
						if k == chatIDAndKeywordList[1] {
							isKeywordExist = true
							break
						}
					}
					if !isKeywordExist {
						logger.Debug().
							Str("globalKeyword", chatIDAndKeywordList[1]).
							Msg("User add a global keyword")
						user.GlobalKeyword = append(user.GlobalKeyword, chatIDAndKeywordList[1])
						pendingMessage = fmt.Sprintf("已添加全局关键词 [ %s ]", chatIDAndKeywordList[1])
					} else {
						pendingMessage = fmt.Sprintf("此全局关键词 [ %s ] 已存在", chatIDAndKeywordList[1])
					}
				} else {
					// 群组关键词
					var chatForUser ChatForUser
					var chatForUserIndex int
					for i, c := range user.ChatsForUser {
						if c.ChatID == chatID {
							chatForUser = c
							chatForUserIndex = i
						}
					}
					targetChat := KeywordDataList.Chats[chatID]
					for _, k := range chatForUser.Keyword {
						if k == chatIDAndKeywordList[1] {
							isKeywordExist = true
							break
						}
					}
					if !isKeywordExist {
						logger.Debug().
							Str("keyword", chatIDAndKeywordList[1]).
							Msg("User add a keyword to chat")
						chatForUser.Keyword = append(chatForUser.Keyword, chatIDAndKeywordList[1])
						user.ChatsForUser[chatForUserIndex] = chatForUser

						pendingMessage = fmt.Sprintf("已为 <a href=\"https://t.me/c/%s/\">%s</a> 群组添加关键词 [ %s ]", utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName, strings.ToLower(chatIDAndKeywordList[1]))
					} else {
						pendingMessage = fmt.Sprintf("此关键词 [ %s ] 已存在于 <a href=\"https://t.me/c/%s/\">%s</a> 群组中，您可以继续向此群组添加其他关键词", chatIDAndKeywordList[1], utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName)
					}
				}

				if isKeywordExist {
					button = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text:         "✅ 完成",
						CallbackData: "detectkw_u",
					}}}}
				} else {
					button = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text:         "↩️ 撤销操作",
							CallbackData: "detectkw_u_undo_" + chatIDAndKeyword,
						},
						{
							Text:         "✅ 完成",
							CallbackData: "detectkw_u",
						},
					}}}
				}
				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID:   opts.CallbackQuery.Message.Message.ID,
					Text:        pendingMessage,
					ReplyMarkup: button,
					ParseMode:   models.ParseModeHTML,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "keyword added notice").
						Msg(flaterr.EditMessageText.Str())
					handlerErr.Addt(flaterr.EditMessageText, "keyword added notice", err)
				}
			}
		}
	}

	if needEdit {
		_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
			MessageID:   opts.CallbackQuery.Message.Message.ID,
			Text:        user.userStatus(),
			ReplyMarkup: user.buildUserChatList(),
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "main manager keyboard").
				Msg(flaterr.EditMessageText.Str())
			handlerErr.Addt(flaterr.EditMessageText, "main manager keyboard", err)
		}
	}

	if needSave {
		KeywordDataList.Users[opts.CallbackQuery.From.ID] = user
		buildListenList()
		err := saveKeywordList(opts.Ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Msg("Failed to save keyword list")
			handlerErr.Addf("failed to save keyword list: %w", err)
		}
	}

	return handlerErr.Flat()
}

func startPrefixAddGroup(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "startPrefixAddGroup").
		Dict(utils.GetUserDict(opts.Message.From)).
		Str("messageText", opts.Message.Text).
		Logger()

	var handlerErr flaterr.MultErr
	var needSave bool = true

	user := KeywordDataList.Users[opts.Message.From.ID]
	if user.AddTime == "" {
		// 初始化用户
		user = KeywordUserList{
			UserID:  opts.Message.From.ID,
			AddTime: time.Now().Format(time.RFC3339),
			Limit:   50,
		}
	} else if user.IsNotInit {
		// 用户之前仅在群组内发送命令添加了关键词，但并没有点击机器人来初始化
		user.IsNotInit = false
		needSave = true
	}

	if strings.HasPrefix(opts.Fields[1], "detectkw_addgroup_") {
		groupID_int64, err := strconv.ParseInt(strings.TrimPrefix(opts.Fields[1], "detectkw_addgroup_"), 10, 64)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to parse chat ID when user add a group by /start command")
			handlerErr.Addf("failed to parse chat ID when user add a group by /start command: %w", err)
		}

		chat, isExist := KeywordDataList.Chats[groupID_int64]
		if !isExist {
			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            "无法添加群组，请尝试在对应群组中发送 /setkeyword 命令后再尝试点击 [ 设定关键词 ] 按钮",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "将此机器人添加到群组或频道",
					URL:  fmt.Sprintf("https://t.me/%s?startgroup=detectkw", consts.BotMe.Username),
				}}}},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "invalid chat ID in /start message").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "invalid chat ID in /start message", err)
			}
		} else {
			var IsAdded bool = false
			var pendingMessage string
			for _, keyword := range user.ChatsForUser {
				if keyword.ChatID == groupID_int64 {
					IsAdded = true
					break
				}
			}
			if !IsAdded {
				logger.Debug().
					Msg("User add a chat to listen list by /start command")
				user.ChatsForUser = append(user.ChatsForUser, ChatForUser{
					ChatID: groupID_int64,
				})
				needSave = true
				pendingMessage = fmt.Sprintf("已添加 <a href=\"https://t.me/c/%s/\">%s</a> 群组\n%s", utils.RemoveIDPrefix(chat.ChatID), chat.ChatName, user.userStatus())
			} else {
				pendingMessage = user.userStatus()
			}

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:      opts.Message.Chat.ID,
				Text:        pendingMessage,
				ParseMode:   models.ParseModeHTML,
				ReplyMarkup: user.buildUserChatList(),
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "added group in user list").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "added group in user list", err)
			}
		}
	}

	if needSave {
		KeywordDataList.Users[opts.Message.From.ID] = user
		buildListenList()
		err := saveKeywordList(opts.Ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(opts.Message.From)).
				Msg("Failed to save keyword list")
			handlerErr.Addf("failed to save keyword list: %w", err)
		}
	}

	return handlerErr.Flat()
}

func startBotAddedToGroup(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "addKeywordHandler").
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	var handlerErr flaterr.MultErr

	chat := KeywordDataList.Chats[opts.Message.Chat.ID]

	if chat.AddTime == "" {
		// 初始化群组
		chat = KeywordChatList{
			ChatID:       opts.Message.Chat.ID,
			ChatName:     opts.Message.Chat.Title,
			ChatUsername: opts.Message.Chat.Username,
			ChatType:     opts.Message.Chat.Type,
			AddTime:      time.Now().Format(time.RFC3339),
			InitByID:     opts.Message.From.ID,
		}
		KeywordDataList.Chats[opts.Message.Chat.ID] = chat
		err := saveKeywordList(opts.Ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&opts.Message.Chat)).
				Msg("Failed to init chat and save keyword list")
			return handlerErr.Addf("failed to init chat and save keyword list: %w", err).Flat()
		}
	}
	// 只有一个 /setkeyword 命令
	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:          opts.Message.Chat.ID,
		Text:            "已记录群组，点击下方左侧按钮来设定监听关键词\n若您是群组的管理员，您可以点击右侧的按钮来管理此功能",
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
			{
				Text: "设定关键词",
				URL:  fmt.Sprintf("https://t.me/%s?start=detectkw_addgroup_%d", consts.BotMe.Username, opts.Message.Chat.ID),
			},
			{
				Text:         "管理此功能",
				CallbackData: "detectkw_g",
			},
		}}},
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("content", "group record link button").
			Msg(flaterr.SendMessage.Str())
		handlerErr.Addt(flaterr.SendMessage, "group record link button", err)
	}

	return handlerErr.Flat()
}
