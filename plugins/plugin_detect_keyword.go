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
	"trbot/utils/flate"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"
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
var KeywordDataDir  string = filepath.Join(consts.YAMLDataBaseDir, "detectkeyword/")
var KeywordDataPath string = filepath.Join(KeywordDataDir, consts.YAMLFileName)

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "Detect Keyword",
		Func: ReadKeywordList,
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Detect Keyword",
		Loader: ReadKeywordList,
		Saver:  SaveKeywordList,
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "setkeyword",
		MessageHandler: addKeywordHandler,
	})
	plugin_utils.AddCallbackQueryHandlers([]plugin_utils.CallbackQuery{
		{
			CallbackDatePrefix:   "detectkw_g",
			CallbackQueryHandler: groupManageCallbackHandler,
		},
		{
			CallbackDatePrefix:   "detectkw_u",
			CallbackQueryHandler: userManageCallbackHandler,
		},
	}...)
	plugin_utils.AddSlashStartWithPrefixCommandHandlers(plugin_utils.SlashStartWithPrefixHandler{
		Prefix:   "detectkw",
		Argument: "addgroup",
		MessageHandler: startPrefixAddGroup,
	})
	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "群组关键词检测",
		Description: "此功能可以检测群组中的每一条信息，当包含设定的关键词时，将会向用户发送提醒\n\n使用方法：\n首先将机器人添加至想要监听关键词的群组中，发送 /setkeyword 命令，等待机器人回应后点击下方的 “设定关键词” 按钮即可为自己添加要监听的群组\n\n设定关键词：您可以在对应的群组中直接发送 <code>/setkeyword 要设定的关键词</code> 来为该群组设定关键词\n或前往机器人聊天页面，发送 <code>/setkeyword</code> 命令后点击对应的群组或全局关键词按钮，根据提示来添加关键词",
		ParseMode:   models.ParseModeHTML,
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
	Limit           int           `yaml:"Limit"`
	MentionCount    int           `yaml:"MentionCount,omitempty"`

	IsNotInit       bool          `yaml:"IsNotInit,omitempty"`
	IsDisable       bool          `yaml:"IsDisable,omitempty"`
	IsSilentNotice  bool          `yaml:"IsSilentNotice,omitempty"`
	IsIncludeSelf   bool          `yaml:"IsIncludeSelf,omitempty"`
	IsIncludeBot    bool          `yaml:"IsIncludeBot,omitempty"` // todo

	AddingChatID    int64         `yaml:"AddingChatID,omitempty"`
	GlobalKeyword   []string      `yaml:"GlobalKeyword"`
	WatchUser       []int64       `yaml:"WatchUser"` // todo
	ChatsForUser    []ChatForUser `yaml:"ChatForUser"`
}

func (user KeywordUserList)enabledChatCount() int {
	var count int
	for _, v := range user.ChatsForUser {
		if !v.IsDisable {
			count++
		}
	}
	return count
}

func (user KeywordUserList)keywordCount() int {
	var count int
	for _, v := range user.ChatsForUser {
		count += len(v.Keyword)
	}
	return count
}

func (user KeywordUserList)userStatus() string {
	var pendingMessage string
	if user.IsDisable {
		pendingMessage = "您已经全局禁用了此功能，要重新开启，请点击最下方的按钮"
	} else {
		pendingMessage = fmt.Sprintf("您添加的群组中有 (%d/%d) 个处于启用状态\n您总共设定了 %d 个关键词", user.enabledChatCount(), len(user.ChatsForUser), user.keywordCount() + len(user.GlobalKeyword))
	}

	return pendingMessage
}

func (user KeywordUserList)selectChat(keyword string) models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton
	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "🌐 添加为全局关键词",
		CallbackData: fmt.Sprintf("detectkw_u_add_%d_%s", user.UserID, keyword),
	}})
	for _, chat := range user.ChatsForUser {
		targetChat := KeywordDataList.Chats[chat.ChatID]
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: "👥 " + targetChat.ChatName,
			CallbackData: fmt.Sprintf("detectkw_u_add_%d_%s", targetChat.ChatID, keyword),
		}})
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

type ChatForUser struct {
	ChatID          int64    `yaml:"ChatID"`
	IsDisable       bool     `yaml:"IsDisable,omitempty"`
	IsConfirmDelete bool     `yaml:"IsConfirmDelete,omitempty"` // todo
	Keyword         []string `yaml:"Keyword"`
}

func ReadKeywordList(ctx context.Context) error {
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

func SaveKeywordList(ctx context.Context) error {
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
		Logger()

	var handlerErr flate.MultErr

	if opts.Message.Chat.Type == models.ChatTypePrivate {
		// 与机器人的私聊对话
		user := KeywordDataList.Users[opts.Message.From.ID]
		if user.AddTime == "" {
			// 初始化用户
			user = KeywordUserList{
				UserID: opts.Message.From.ID,
				AddTime: time.Now().Format(time.RFC3339),
				Limit: 50,
				IsDisable: false,
				IsSilentNotice: false,
			}
			KeywordDataList.Users[opts.Message.From.ID] = user
			err := SaveKeywordList(opts.Ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Message.From)).
					Msg("Failed to init user and save keyword list")
				return handlerErr.Addf("failed to init user and save keyword list: %w", err).Flat()
			}
		}

		// 用户没有添加任何群组
		if len(user.ChatsForUser) == 0 {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:          opts.Message.Chat.ID,
				Text:            "您还没有添加任何群组，请在群组中使用 `/setkeyword` 命令来记录群组\n若发送信息后没有回应，请检查机器人是否在对应群组中",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				ParseMode:       models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&opts.Message.Chat)).
					Dict(utils.GetUserDict(opts.Message.From)).
					Str("content", "no group for user notice").
					Msg(flate.SendMessage.Str())
				handlerErr.Addf(flate.SendMessage.Fmt(), "no group for user notice", err)
			}
		} else {
			if len(opts.Fields) == 1 {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Message.Chat.ID,
					Text: user.userStatus(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: buildUserChatList(user),
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(opts.Message.From)).
						Str("content", "user group list keyboard").
						Msg(flate.SendMessage.Str())
					handlerErr.Addf(flate.SendMessage.Fmt(), "user group list keyboard", err)
				}
			} else {
				// 限制关键词长度
				if len(opts.Fields[1]) > 30 {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Message.Chat.ID,
						Text: "抱歉，单个关键词长度不能超过 30 个英文字符",
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetChatDict(&opts.Message.Chat)).
							Dict(utils.GetUserDict(opts.Message.From)).
							Int("length", len(opts.Fields[1])).
							Str("content", "keyword is too long").
							Msg(flate.SendMessage.Str())
						handlerErr.Addf(flate.SendMessage.Fmt(), "keyword is too long", err)
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
									Dict(utils.GetUserDict(opts.Message.From)).
									Str("globalKeyword", keyword).
									Msg("User add a global keyword")
								user.GlobalKeyword = append(user.GlobalKeyword, keyword)
								KeywordDataList.Users[user.UserID] = user
								err := SaveKeywordList(opts.Ctx)
								if err != nil {
									logger.Error().
										Err(err).
										Dict(utils.GetUserDict(opts.Message.From)).
										Str("globalKeyword", keyword).
										Msg("Failed to add global keyword and save keyword list")
									return handlerErr.Addf("failed to add global keyword and save keyword list: %w", err).Flat()
								}
								pendingMessage = fmt.Sprintf("已添加全局关键词: [ %s ], 您可以继续添加更多全局关键词", keyword)
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
									Dict(utils.GetUserDict(opts.Message.From)).
									Int64("chatID", chatForUser.ChatID).
									Str("keyword", keyword).
									Msg("User add a keyword to chat")
								chatForUser.Keyword = append(chatForUser.Keyword, keyword)
								user.ChatsForUser[chatForUserIndex] = chatForUser
								KeywordDataList.Users[user.UserID] = user
								err := SaveKeywordList(opts.Ctx)
								if err != nil {
									logger.Error().
										Err(err).
										Dict(utils.GetUserDict(opts.Message.From)).
										Int64("chatID", chatForUser.ChatID).
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
							button = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
								Text: "✅ 完成",
								CallbackData: "detectkw_u_finish",
							}}}}
						} else {
							button = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
								{
									Text: "↩️ 撤销操作",
									CallbackData: fmt.Sprintf("detectkw_u_undo_%d_%s", user.AddingChatID, opts.Fields[1]),
								},
								{
									Text: "✅ 完成",
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
								Dict(utils.GetUserDict(opts.Message.From)).
								Str("content", "keyword added notice").
								Msg(flate.SendMessage.Str())
							handlerErr.Addf(flate.SendMessage.Fmt(), "keyword added notice", err)
						}
					} else {
						_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
							ChatID: opts.Message.Chat.ID,
							Text: "您还没有选定要将关键词添加到哪个群组，请在下方挑选一个您已经添加的群组",
							ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
							ParseMode: models.ParseModeHTML,
							ReplyMarkup: user.selectChat(opts.Fields[1]),
						})
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(opts.Message.From)).
								Str("content", "keyword adding select keyboard").
								Msg(flate.SendMessage.Str())
							handlerErr.Addf(flate.SendMessage.Fmt(), "keyword adding select keyboard", err)
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
				ChatID: opts.Message.Chat.ID,
				Text: "群组管理员已禁用关键词功能，您可以询问管理员以获取更多信息",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "管理此功能",
					CallbackData: "detectkw_g",
				}}}},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&opts.Message.Chat)).
					Dict(utils.GetUserDict(opts.Message.From)).
					Str("content", "function is disabled by admins").
					Msg(flate.SendMessage.Str())
				handlerErr.Addf(flate.SendMessage.Fmt(), "function is disabled by admins", err)
			}
		} else {
			if chat.AddTime == "" {
				// 初始化群组
				chat = KeywordChatList{
					ChatID:       opts.Message.Chat.ID,
					ChatName:     opts.Message.Chat.Title,
					ChatUsername: opts.Message.Chat.Username,
					ChatType:     opts.Message.Chat.Type,
					AddTime:      time.Now().Format(time.RFC3339),
					InitByID:     opts.Message.From.ID,
					IsDisable:    false,
				}
				KeywordDataList.Chats[opts.Message.Chat.ID] = chat
				err := SaveKeywordList(opts.Ctx)
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
					ChatID: opts.Message.Chat.ID,
					Text: "已记录群组，点击下方左侧按钮来设定监听关键词\n若您是群组的管理员，您可以点击右侧的按钮来管理此功能",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "设定关键词",
							URL: fmt.Sprintf("https://t.me/%s?start=detectkw_addgroup_%d", consts.BotMe.Username, opts.Message.Chat.ID),
						},
						{
							Text: "管理此功能",
							CallbackData: "detectkw_g",
						},
					}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetChatDict(&opts.Message.Chat)).
						Str("content", "group record link button").
						Msg(flate.SendMessage.Str())
					handlerErr.Addf(flate.SendMessage.Fmt(), "group record link button", err)
				}
			} else {
				// 限制关键词长度
				if len(opts.Fields[1]) > 30 {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Message.Chat.ID,
						Text: "抱歉，单个关键词长度不能超过 30 个英文字符",
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(opts.Message.From)).
							Dict(utils.GetChatDict(&opts.Message.Chat)).
							Int("keywordLength", len(opts.Fields[1])).
							Str("content", "keyword is too long").
							Msg(flate.SendMessage.Str())
						handlerErr.Addf("failed to send `keyword is too long` message: %w", err)
					}
				} else {
					user := KeywordDataList.Users[opts.Message.From.ID]

					if user.AddTime == "" {
						// 初始化用户
						user = KeywordUserList{
							UserID: opts.Message.From.ID,
							AddTime: time.Now().Format(time.RFC3339),
							Limit: 50,
							IsNotInit: true,
						}
						KeywordDataList.Users[opts.Message.From.ID] = user
						err := SaveKeywordList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(opts.Message.From)).
								Msg("Failed to add a not init user and save keyword list")
							return handlerErr.Addf("failed to add a not init user and save keyword list: %w", err).Flat()
						}
					}

					var isChatAdded bool = false
					var chatForUser ChatForUser
					var chatForUserIndex int

					for index, keyword := range user.ChatsForUser {
						if keyword.ChatID == chat.ChatID {
							chatForUser = keyword
							chatForUserIndex = index
							isChatAdded = true
							break
						}
					}
					if !isChatAdded {
						logger.Debug().
							Dict(utils.GetUserDict(opts.Message.From)).
							Dict(utils.GetChatDict(&opts.Message.Chat)).
							Msg("User add a chat to listen list by set keyword in group")
						chatForUser = ChatForUser{
							ChatID: chat.ChatID,
						}
						user.ChatsForUser = append(user.ChatsForUser, chatForUser)
						KeywordDataList.Users[user.UserID] = user
						err := SaveKeywordList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetChatDict(&opts.Message.Chat)).
								Dict(utils.GetUserDict(opts.Message.From)).
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
					if !isKeywordExist {
						logger.Debug().
							Dict(utils.GetUserDict(opts.Message.From)).
							Int64("chatID", chatForUser.ChatID).
							Str("keyword", keyword).
							Msg("User add a keyword to chat")
						chatForUser.Keyword = append(chatForUser.Keyword, keyword)
						user.ChatsForUser[chatForUserIndex] = chatForUser
						KeywordDataList.Users[user.UserID] = user
						err := SaveKeywordList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(opts.Message.From)).
								Dict(utils.GetChatDict(&opts.Message.Chat)).
								Str("keyword", keyword).
								Msg("Failed to add keyword and save keyword list")
							return handlerErr.Addf("failed to add keyword and save keyword list: %w", err).Flat()
						}
						if user.IsNotInit {
							pendingMessage = fmt.Sprintf("已将 [ %s ] 添加到您的关键词列表\n<blockquote>若要在检测到关键词时收到提醒，请点击下方的按钮来初始化您的账号</blockquote>", strings.ToLower(opts.Fields[1]))
						} else {
							pendingMessage = fmt.Sprintf("已将 [ %s ] 添加到您的关键词列表，您可以继续向此群组添加更多关键词", strings.ToLower(opts.Fields[1]))
						}
					} else {
						if user.IsNotInit {
							pendingMessage = "您已经添加过这个关键词了。请点击下方的按钮来初始化您的账号以使用此功能"
						} else {
							pendingMessage = "您已经添加过这个关键词了，您可以继续向此群组添加其他关键词"
						}
					}

					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Message.Chat.ID,
						Text:   pendingMessage,
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
						ParseMode: models.ParseModeHTML,
						ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
							Text: "管理关键词",
							URL: fmt.Sprintf("https://t.me/%s?start=detectkw_addgroup_%d", consts.BotMe.Username, chat.ChatID),
						}}}},
					})
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(opts.Message.From)).
							Dict(utils.GetChatDict(&opts.Message.Chat)).
							Str("content", "keyword added notice").
							Msg(flate.SendMessage.Str())
						handlerErr.Addf(flate.SendMessage.Fmt(), "keyword added notice", err)
					}
				}
			}
		}
	}
	return handlerErr.Flat()
}

func addKeywordStateHandler(opts *handler_params.Update) error {
	if opts.Update.Message == nil { return nil }
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "addKeywordStateHandler").
		Logger()

	var handlerErr flate.MultErr

	if opts.Update.Message.Text != "" {
		keyword := strings.ToLower(strings.Fields(opts.Update.Message.Text)[0])

		if len(keyword) > 30 {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text: "抱歉，单个关键词长度不能超过 30 个英文字符",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Int("length", len(keyword)).
					Str("content", "keyword is too long").
					Msg(flate.SendMessage.Str())
				handlerErr.Addf(flate.SendMessage.Fmt(), "keyword is too long", err)
			}
		} else {
			user := KeywordDataList.Users[opts.Update.Message.From.ID]
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
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Str("globalKeyword", keyword).
						Msg("User add a global keyword")
					user.GlobalKeyword = append(user.GlobalKeyword, keyword)
					KeywordDataList.Users[user.UserID] = user
					err := SaveKeywordList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(opts.Update.Message.From)).
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
				if !isKeywordExist {
					logger.Debug().
						Dict(utils.GetUserDict(opts.Update.Message.From)).
						Int64("chatID", chatForUser.ChatID).
						Str("keyword", keyword).
						Msg("User add a keyword to chat")
					chatForUser.Keyword = append(chatForUser.Keyword, keyword)
					user.ChatsForUser[chatForUserIndex] = chatForUser
					KeywordDataList.Users[user.UserID] = user
					err := SaveKeywordList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(opts.Update.Message.From)).
							Int64("chatID", chatForUser.ChatID).
							Str("keyword", keyword).
							Msg("Failed to add keyword and save keyword list")
						return handlerErr.Addf("failed to add keyword and save keyword list: %w", err).Flat()
					}
					pendingMessage = fmt.Sprintf("已为 <a href=\"https://t.me/c/%s/\">%s</a> 群组添加关键词 [ %s ]，您可以继续向此群组添加更多关键词\n或发送 /cancel 命令来取消操作", utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName, keyword)
				} else {
					pendingMessage = fmt.Sprintf("此关键词 [ %s ] 已存在于 <a href=\"https://t.me/c/%s/\">%s</a> 群组中，您可以继续向此群组添加其他关键词\n或发送 /cancel 命令来取消操作", keyword, utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName)
				}
			}

			if isKeywordExist {
				button = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "⬅️ 停止添加关键词",
					CallbackData: "detectkw_u_finish",
				}}}}
			} else {
				button = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
					{
						Text: "↩️ 撤销操作",
						CallbackData: fmt.Sprintf("detectkw_u_undo_%d_%s", user.AddingChatID, opts.Update.Message.Text),
					},
					{
						Text: "⬅️ 停止添加关键词",
						CallbackData: "detectkw_u_finish",
						// CallbackData: fmt.Sprintf("detectkw_u_chat_%d", user.AddingChatID),
					},
				}}}
			}

			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text: pendingMessage,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: button,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(opts.Update.Message.From)).
					Str("content", "keyword added notice").
					Msg(flate.SendMessage.Str())
				handlerErr.Addf(flate.SendMessage.Fmt(), "keyword added notice", err)
			}
		}
	} else {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			Text: "请输入有效的关键词",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "⬅️ 停止添加关键词",
				CallbackData: "detectkw_u_finish",
			}}}},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(opts.Update.Message.From)).
				Str("content", "keyword invalid notice").
				Msg(flate.SendMessage.Str())
			handlerErr.Addf(flate.SendMessage.Fmt(), "keyword invalid notice", err)
		}
		// plugin_utils.EditStateHandler(opts.Update.Message.Chat.ID, 1, nil)
	}

	return handlerErr.Flat()
}

func buildListenList() {
	for index, chat := range KeywordDataList.Chats {
		chat.UsersID = []int64{}
		KeywordDataList.Chats[index] = chat
		plugin_utils.RemoveHandlerByChatIDHandler(chat.ChatID, "detect_keyword")
	}
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
	for _, chat := range KeywordDataList.Chats {
		if !chat.IsDisable && len(chat.UsersID) > 0 {
			plugin_utils.AddHandlerByChatIDHandlers(plugin_utils.ByChatIDHandler{
				ForChatID:         chat.ChatID,
				PluginName:     "detect_keyword",
				UpdateHandler:  KeywordDetector,
			})
		}
	}
}

func KeywordDetector(opts *handler_params.Update) error {
	var handlerErr flate.MultErr
	var text string
	if opts.Update.Message.Caption != "" {
		text = strings.ToLower(opts.Update.Message.Caption)
	} else if opts.Update.Message.Text != "" {
		text = strings.ToLower(opts.Update.Message.Text)
	}

	// 没有文字直接跳过
	if text == "" { return nil }

	// 先循环一遍，找出该群组中启用此功能的用户 ID
	for _, userID := range KeywordDataList.Chats[opts.Update.Message.Chat.ID].UsersID {
		// 获取用户信息，开始匹配关键词
		user := KeywordDataList.Users[userID]
		if !user.IsDisable && !user.IsNotInit {
			// 如果用户设定排除了自己发送的消息，则跳过
			if !user.IsIncludeSelf && opts.Update.Message.From.ID == userID { continue }

			// 用户为单独群组设定的关键词
			for _, userKeywordList := range user.ChatsForUser {
				// 判断是否是此群组
				if userKeywordList.ChatID == opts.Update.Message.Chat.ID {
					for _, keyword := range userKeywordList.Keyword {
						if strings.Contains(text, keyword) {
							handlerErr.Add(notifyUser(opts, user, opts.Update.Message.Chat.Title, keyword, text, false))
							break
						}
					}
				}
			}
			// 用户全局设定的关键词
			for _, userGlobalKeyword := range user.GlobalKeyword {
				if strings.Contains(text, userGlobalKeyword) {
					handlerErr.Add(notifyUser(opts, user, opts.Update.Message.Chat.Title, userGlobalKeyword, text, true))
					break
				}
			}
		}
	}
	return handlerErr.Flat()
}

func notifyUser(opts *handler_params.Update, user KeywordUserList, chatname, keyword, text string, isGlobalKeyword bool) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "notifyUser").
		Logger()

	var handlerErr flate.MultErr

	var messageLink string = fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(opts.Update.Message.Chat.ID), opts.Update.Message.ID)

	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: user.UserID,
		Text: fmt.Sprintf("在 <a href=\"https://t.me/c/%s/\">%s</a> 群组中\n来自 %s 的消息\n触发了%s关键词 [ %s ]\n<blockquote expandable>%s</blockquote>",
			utils.RemoveIDPrefix(opts.Update.Message.Chat.ID), chatname, utils.GetMessageFromHyperLink(opts.Update.Message, models.ParseModeHTML),
			utils.TextForTrueOrFalse(isGlobalKeyword, "全局", "群组"), keyword, text,
		),
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击前往查看",
			URL:  messageLink,
		}}}},
		DisableNotification: user.IsSilentNotice,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetChatDict(&opts.Update.Message.Chat)).
			Int64("userID", user.UserID).
			Str("keyword", keyword).
			Str("content", "keyword detected notice to user").
			Msg(flate.SendMessage.Str())
		handlerErr.Addf(flate.SendMessage.Fmt(), "keyword detected notice to user", err)
	}
	user.MentionCount++
	KeywordDataList.Users[user.UserID] = user

	return handlerErr.Flat()
}

func groupManageCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "groupManageCallbackHandler").
		Logger()

	var handlerErr flate.MultErr

	if !utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.CallbackQuery.Message.Message.Chat.ID, opts.CallbackQuery.From.ID) {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text: "您没有权限修改此配置",
			ShowAlert: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&opts.CallbackQuery.Message.Message.Chat)).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Str("content", "no permission to change group functions").
				Msg(flate.AnswerCallbackQuery.Str())
			handlerErr.Addf(flate.AnswerCallbackQuery.Fmt(), "no permission to change group functions", err)
		}
	} else {
		chat := KeywordDataList.Chats[opts.CallbackQuery.Message.Message.Chat.ID]

		if opts.CallbackQuery.Data == "detectkw_g_switch" {
			// 群组里的全局开关，是否允许群组内用户使用这个功能，优先级最高
			chat.IsDisable = !chat.IsDisable
			KeywordDataList.Chats[opts.CallbackQuery.Message.Message.Chat.ID] = chat
			buildListenList()
			err := SaveKeywordList(opts.Ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetChatDict(&opts.CallbackQuery.Message.Message.Chat)).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Msg("Failed to change group switch and save keyword list")
				handlerErr.Addf("failed to change group switch and save keyword list: %w", err)
			}
		}

		_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.CallbackQuery.Message.Message.ID,
			Text: fmt.Sprintf("消息关键词检测\n此功能允许用户设定一些关键词，当机器人检测到群组内的消息包含用户设定的关键词时，向用户发送提醒\n\n当前群组中有 %d 个用户启用了此功能\n\n%s", len(chat.UsersID),  utils.TextForTrueOrFalse(chat.IsDisable, "已为当前群组关闭关键词检测功能，已设定了关键词的用户将无法再收到此群组的提醒", "")),
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "🔄 当前状态: " + utils.TextForTrueOrFalse(chat.IsDisable, "已禁用 ❌", "已启用 ✅"),
				CallbackData: "detectkw_g_switch",
			}}}},
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetChatDict(&opts.CallbackQuery.Message.Message.Chat)).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Str("content", "group function manager keyboard").
				Msg(flate.EditMessageText.Str())
			handlerErr.Addf(flate.EditMessageText.Fmt(), "group function manager keyboard", err)
		}
	}

	return handlerErr.Flat()
}

func userManageCallbackHandler(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "userManageCallbackHandler").
		Logger()

	var handlerErr flate.MultErr
	user := KeywordDataList.Users[opts.CallbackQuery.From.ID]

	switch opts.CallbackQuery.Data {
	case "detectkw_u_globalswitch":
		// 功能全局开关
		user.IsDisable = !user.IsDisable
	case "detectkw_u_noticeswitch":
		// 是否静默通知
		user.IsSilentNotice = !user.IsSilentNotice
	case "detectkw_u_selfswitch":
		// 是否检测自己发送的消息
		user.IsIncludeSelf = !user.IsIncludeSelf
	case "detectkw_u_finish":
		// 停止添加群组关键词
		user.AddingChatID = 0
		plugin_utils.RemoveStateHandler(user.UserID)
	case "detectkw_u_chatdisablebyadmin":
		// 目标群组的管理员为群组关闭了此功能
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "此群组中的管理员禁用了此功能，因此，您无法再收到来自该群组的关键词提醒，您可以询问该群组的管理员是否可以重新开启这个功能",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Str("content", "this group is disable by admins").
				Msg(flate.AnswerCallbackQuery.Str())
			handlerErr.Addf(flate.AnswerCallbackQuery.Fmt(), "this group is disable by admins", err)
		}
		return handlerErr.Flat()
	default:
		if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_undo_") || strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_delkw_") {
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
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("callbackQueryData", opts.CallbackQuery.Data).
					Msg("Failed to parse chat ID when user undo add or delete a keyword")
				handlerErr.Addf("failed to parse chat ID when user undo add or delete a keyword: %w", err)
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
					KeywordDataList.Users[user.UserID] = user
				} else {
					// 群组关键词
					for index, chatForUser := range KeywordDataList.Users[user.UserID].ChatsForUser {
						if chatForUser.ChatID == chatID {
							var tempKeyword []string
							for _, keyword := range chatForUser.Keyword {
								if keyword != chatIDAndKeywordList[1] {
									tempKeyword = append(tempKeyword, keyword)
								}
							}
							chatForUser.Keyword = tempKeyword
						}
						KeywordDataList.Users[user.UserID].ChatsForUser[index] = chatForUser
					}
				}
				err = SaveKeywordList(opts.Ctx)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
						Str("callbackQueryData", opts.CallbackQuery.Data).
						Int64("chatID", chatID).
						Str("keyword", chatIDAndKeywordList[1]).
						Msg("Failed to undo add or remove keyword and save keyword list")
					handlerErr.Addf("failed to undo add or remove keyword and save keyword list: %w", err)
				} else {
					if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_undo_") {
						_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
							ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
							MessageID: opts.CallbackQuery.Message.Message.ID,
							Text: fmt.Sprintf("已取消添加 [ %s ] 关键词，您可以继续添加其他关键词", chatIDAndKeywordList[1]),
							ParseMode: models.ParseModeHTML,
							ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{ {
								Text: "✅ 完成",
								CallbackData: "detectkw_u_finish",
							}}}},
						})
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
								Str("callbackQueryData", opts.CallbackQuery.Data).
								Str("content", "add keyword has been canceled notice").
								Msg(flate.EditMessageText.Str())
							handlerErr.Addf(flate.EditMessageText.Fmt(), "add keyword has been canceled notice", err)
						}
					} else {
						var buttons [][]models.InlineKeyboardButton
						var tempbutton []models.InlineKeyboardButton
						var keywordCount int
						var pendingMessage string

						if chatID == user.UserID {
							for index, keyword := range user.GlobalKeyword {
								if index % 2 == 0 && index != 0 {
									buttons = append(buttons, tempbutton)
									tempbutton = []models.InlineKeyboardButton{}
								}
								tempbutton = append(tempbutton, models.InlineKeyboardButton{
									Text: keyword,
									CallbackData: fmt.Sprintf("detectkw_u_kw_%d_%s", user.UserID, keyword),
								})
								keywordCount++
								// buttons = append(buttons, tempbutton)
							}
							if len(tempbutton) != 0 {
								buttons = append(buttons, tempbutton)
							}
							pendingMessage = fmt.Sprintf("已删除 [ %s ] 关键词\n\n您当前设定了 %d 个全局关键词", chatIDAndKeywordList[1], keywordCount)

						} else {
							for _, chat := range user.ChatsForUser {
								if chat.ChatID == chatID {
									for index, keyword := range chat.Keyword {
										if index % 2 == 0 && index != 0 {
											buttons = append(buttons, tempbutton)
											tempbutton = []models.InlineKeyboardButton{}
										}
										tempbutton = append(tempbutton, models.InlineKeyboardButton{
											Text: keyword,
											CallbackData: fmt.Sprintf("detectkw_u_kw_%d_%s", chat.ChatID, keyword),
										})
										keywordCount++
										// buttons = append(buttons, tempbutton)
									}
									if len(tempbutton) != 0 {
										buttons = append(buttons, tempbutton)
									}
								}
							}
							pendingMessage = fmt.Sprintf("已删除 [ %s ] 关键词\n\n您当前为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定了 %d 个关键词", chatIDAndKeywordList[1], utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName, keywordCount)
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
							ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
							MessageID: opts.CallbackQuery.Message.Message.ID,
							Text: pendingMessage,
							ParseMode: models.ParseModeHTML,
							ReplyMarkup: &models.InlineKeyboardMarkup{
								InlineKeyboard: buttons,
							},
						})
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
								Str("callbackQueryData", opts.CallbackQuery.Data).
								Str("content", "keyword list keyboard with deleted keyword notice").
								Msg(flate.EditMessageText.Str())
						}
					}
				}
			}

			return handlerErr.Flat()
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_adding_") {
			// 设定要往哪个群组里添加关键词
			chatID := strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_adding_")
			chatID_int64, err := strconv.ParseInt(chatID, 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("callbackQueryData", opts.CallbackQuery.Data).
					Msg("Failed to parse chat ID when user selecting chat to add keyword")
				handlerErr.Addf("failed to parse chat ID when user selecting chat to add keyword: %w", err)
			} else {
				user := KeywordDataList.Users[user.UserID]
				user.AddingChatID = chatID_int64
				KeywordDataList.Users[user.UserID] = user
				buildListenList()
				err = SaveKeywordList(opts.Ctx)
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
						Str("callbackQueryData", opts.CallbackQuery.Data).
						Int64("chatID", chatID_int64).
						Msg("Failed to set a chat ID for user add keyword and save keyword list")
					handlerErr.Addf("failed to set a chat ID for user add keyword and save keyword list: %w", err)
				} else {
					var pendingMessage string
					if chatID_int64 == user.UserID {
						pendingMessage = "已将全局关键词设为添加关键词的目标，请继续发送您要添加单个的全局关键词\n或发送 /cancel 命令来取消操作"
					} else {
						pendingMessage = fmt.Sprintf("已将 <a href=\"https://t.me/c/%s/\">%s</a> 群组设为添加关键词的目标群组，请继续发送您要添加单个的群组关键词\n或发送 /cancel 命令来取消操作", utils.RemoveIDPrefix(chatID_int64), KeywordDataList.Chats[chatID_int64].ChatName)
					}

					_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
						ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
						MessageID: opts.CallbackQuery.Message.Message.ID,
						Text: pendingMessage,
						ParseMode: models.ParseModeHTML,
						ReplyMarkup: models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
							Text: "⬅️ 停止添加关键词",
							CallbackData: "detectkw_u_finish",
						}}}},
					})
					if err != nil {
						logger.Error().
							Err(err).
							Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
							Str("callbackQueryData", opts.CallbackQuery.Data).
							Str("content", "ready to add keyword notice").
							Msg(flate.EditMessageText.Str())
						handlerErr.Addf(flate.EditMessageText.Fmt(), "ready to add keyword notice", err)
					} else {
						plugin_utils.AddStateHandler(plugin_utils.StateHandler{
							ForChatID: opts.CallbackQuery.Message.Message.Chat.ID,
							PluginName: "addKeywordState",
							Remaining: -1,
							Handler: addKeywordStateHandler,
						})
					}
				}
			}

			return handlerErr.Flat()
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_switch_chat_") {
		// 启用或禁用某个群组的关键词检测开关
			id := strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_switch_chat_")
			id_int64, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("callbackQueryData", opts.CallbackQuery.Data).
					Msg("Failed to parse chat ID when user change the group switch")
				return handlerErr.Addf("failed to parse chat ID when user change the group switch: %w", err).Flat()
			} else {
				for index, chat := range KeywordDataList.Users[opts.CallbackQuery.From.ID].ChatsForUser {
					if chat.ChatID == id_int64 {
						chat.IsDisable = !chat.IsDisable
					}
					KeywordDataList.Users[opts.CallbackQuery.From.ID].ChatsForUser[index] = chat
				}
			}
			// edit by the end
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_chat_") {
		// 显示某个群组的关键词列表
			chatID := strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_chat_")
			chatID_int64, err := strconv.ParseInt(chatID, 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("callbackQueryData", opts.CallbackQuery.Data).
					Msg("Failed to parse chat ID when user wanna manage keyword for group")
				handlerErr.Addf("failed to parse chat ID when user wanna manage keyword for group: %w", err)
			} else {
				var buttons [][]models.InlineKeyboardButton
				var tempbutton []models.InlineKeyboardButton
				var pendingMessage string
				var keywordCount int

				if chatID_int64 == user.UserID {
					// 全局关键词
					for index, keyword := range user.GlobalKeyword {
						if index % 2 == 0 && index != 0 {
							buttons = append(buttons, tempbutton)
							tempbutton = []models.InlineKeyboardButton{}
						}
						tempbutton = append(tempbutton, models.InlineKeyboardButton{
							Text: keyword,
							CallbackData: fmt.Sprintf("detectkw_u_kw_%d_%s", user.UserID, keyword),
						})
						keywordCount++
					}
					if len(tempbutton) != 0 {
						buttons = append(buttons, tempbutton)
					}
					if len(buttons) == 0 {
						pendingMessage = "您没有设定任何全局关键词\n点击下方按钮来添加全局关键词"
					} else {
						pendingMessage = fmt.Sprintf("您当前设定了 %d 个全局关键词\n<blockquote expandable>全局关键词将对您添加的全部群组生效，但在部分情况下，全局关键词不会生效：\n- 您手动将群组设定为禁用状态\n- 对应群组的管理员为该群组关闭了此功能</blockquote>", keywordCount)
					}
				} else {
					// 为群组设定的关键词
					for _, chat := range KeywordDataList.Users[opts.CallbackQuery.From.ID].ChatsForUser {
						if chat.ChatID == chatID_int64 {
							for index, keyword := range chat.Keyword {
								if index % 2 == 0 && index != 0 {
									buttons = append(buttons, tempbutton)
									tempbutton = []models.InlineKeyboardButton{}
								}
								tempbutton = append(tempbutton, models.InlineKeyboardButton{
									Text: keyword,
									CallbackData: fmt.Sprintf("detectkw_u_kw_%d_%s", chat.ChatID, keyword),
								})
								keywordCount++
								// buttons = append(buttons, tempbutton)
							}
							if len(tempbutton) != 0 {
								buttons = append(buttons, tempbutton)
							}
							break
						}
					}
					if len(buttons) == 0 {
						pendingMessage = fmt.Sprintf("当前群组 <a href=\"https://t.me/c/%s/\">%s</a> 没有关键词\n点击下方按钮来为此群组添加关键词", utils.RemoveIDPrefix(chatID_int64), KeywordDataList.Chats[chatID_int64].ChatName)
					} else {
						pendingMessage = fmt.Sprintf("您当前为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定了 %d 个关键词", utils.RemoveIDPrefix(chatID_int64), KeywordDataList.Chats[chatID_int64].ChatName, keywordCount)
					}
				}

				buttons = append(buttons, []models.InlineKeyboardButton{
					{
						Text: "⬅️ 返回主菜单",
						CallbackData: "detectkw_u",
					},
					{
						Text: "➕ 添加关键词",
						CallbackData: fmt.Sprintf("detectkw_u_adding_%d", chatID_int64),
					},
				})

				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					Text: pendingMessage,
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{
						InlineKeyboard: buttons,
					},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
						Str("callbackQueryData", opts.CallbackQuery.Data).
						Str("content", "group keyword list keyboard").
						Msg(flate.EditMessageText.Str())
					handlerErr.Addf(flate.EditMessageText.Fmt(), "group keyword list keyboard", err)
				}
			}

			return handlerErr.Flat()
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_kw_") {
			// 管理一个关键词
			chatIDAndKeyword := strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_kw_")
			chatIDAndKeywordList := strings.Split(chatIDAndKeyword, "_")
			chatID, err := strconv.ParseInt(chatIDAndKeywordList[0], 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("callbackQueryData", opts.CallbackQuery.Data).
					Msg("Failed to parse chat ID when user wanna manage a keyword")
				handlerErr.Addf("failed to parse chat ID when user wanna manage a keyword: %w", err)
			} else {
				var pendingMessage string

				if chatID == user.UserID {
					pendingMessage = fmt.Sprintf("[ %s ] 是您设定的全局关键词", chatIDAndKeywordList[1])
				} else {
					pendingMessage = fmt.Sprintf("[ %s ] 是为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定的关键词", chatIDAndKeywordList[1], utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName)
				}

				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					Text: pendingMessage,
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "⬅️ 返回",
							CallbackData: "detectkw_u_chat_" + chatIDAndKeywordList[0],
						},
						{
							Text: "❌ 删除此关键词",
							CallbackData: "detectkw_u_delkw_" + chatIDAndKeyword,
						},
					}}},
					ParseMode: models.ParseModeHTML,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
						Str("callbackQueryData", opts.CallbackQuery.Data).
						Str("content", "keyword manager keyboard").
						Msg(flate.EditMessageText.Str())
					handlerErr.Addf(flate.EditMessageText.Fmt(), "keyword manager keyboard", err)
				}
			}

			return handlerErr.Flat()
		} else if strings.HasPrefix(opts.CallbackQuery.Data, "detectkw_u_add_") {
			chatIDAndKeyword := strings.TrimPrefix(opts.CallbackQuery.Data, "detectkw_u_add_")
			chatIDAndKeywordList := strings.Split(chatIDAndKeyword, "_")
			chatID, err := strconv.ParseInt(chatIDAndKeywordList[0], 10, 64)
			if err != nil {
				logger.Error().
					Err(err).
					Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
					Str("callbackQueryData", opts.CallbackQuery.Data).
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
							Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
							Str("callbackQueryData", opts.CallbackQuery.Data).
							Str("globalKeyword", chatIDAndKeywordList[1]).
							Msg("User add a global keyword")
						user.GlobalKeyword = append(user.GlobalKeyword, chatIDAndKeywordList[1])
						KeywordDataList.Users[user.UserID] = user
						err := SaveKeywordList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
								Str("callbackQueryData", opts.CallbackQuery.Data).
								Str("globalKeyword", chatIDAndKeywordList[1]).
								Msg("Failed to add global keyword and save keyword list")
							return handlerErr.Addf("failed to add global keyword and save keyword list: %w", err).Flat()
						}
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
							Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
							Str("callbackQueryData", opts.CallbackQuery.Data).
							Msg("User add a keyword to chat")
						chatForUser.Keyword = append(chatForUser.Keyword, chatIDAndKeywordList[1])
						user.ChatsForUser[chatForUserIndex] = chatForUser
						KeywordDataList.Users[user.UserID] = user
						err := SaveKeywordList(opts.Ctx)
						if err != nil {
							logger.Error().
								Err(err).
								Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
								Str("callbackQueryData", opts.CallbackQuery.Data).
								Msg("Failed to add keyword and save keyword list")
							return handlerErr.Addf("failed to add keyword and save keyword list: %w", err).Flat()
						}
						pendingMessage = fmt.Sprintf("已为 <a href=\"https://t.me/c/%s/\">%s</a> 群组添加关键词 [ %s ]", utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName, strings.ToLower(chatIDAndKeywordList[1]))
					} else {
						pendingMessage = fmt.Sprintf("此关键词 [ %s ] 已存在于 <a href=\"https://t.me/c/%s/\">%s</a> 群组中，您可以继续向此群组添加其他关键词", chatIDAndKeywordList[1], utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName)
					}
				}

				if isKeywordExist {
					button = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "✅ 完成",
						CallbackData: "detectkw_u",
					}}}}
				} else {
					button = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "↩️ 撤销操作",
							CallbackData: fmt.Sprintf("detectkw_u_undo_%d_%s", chatID, chatIDAndKeywordList[1]),
						},
						{
							Text: "✅ 完成",
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
						Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
						Str("callbackQueryData", opts.CallbackQuery.Data).
						Str("content", "keyword added notice").
						Msg(flate.EditMessageText.Str())
					handlerErr.Addf(flate.EditMessageText.Fmt(),"keyword added notice", err)
				}
			}

			return handlerErr.Flat()
		}
	}

	_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
		ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
		MessageID: opts.CallbackQuery.Message.Message.ID,
		Text: user.userStatus(),
		ReplyMarkup: buildUserChatList(user),
	})
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
			Str("callbackQueryData", opts.CallbackQuery.Data).
			Str("content", "main manager keyboard").
			Msg(flate.EditMessageText.Str())
		handlerErr.Addf(flate.EditMessageText.Fmt(), "main manager keyboard", err)
	}

	KeywordDataList.Users[opts.CallbackQuery.From.ID] = user
	buildListenList()
	err = SaveKeywordList(opts.Ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
			Str("callbackQueryData", opts.CallbackQuery.Data).
			Msg("Failed to save keyword list")
		handlerErr.Addf("failed to save keyword list: %w", err)
	}

	return handlerErr.Flat()
}

func startPrefixAddGroup(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "DetectKeyword").
		Str("funcName", "startPrefixAddGroup").
		Logger()

	var handlerErr flate.MultErr

	user := KeywordDataList.Users[opts.Message.From.ID]
	if user.AddTime == "" {
		// 初始化用户
		user = KeywordUserList{
			UserID: opts.Message.From.ID,
			AddTime: time.Now().Format(time.RFC3339),
			Limit: 50,
			IsDisable: false,
			IsSilentNotice: false,
		}
		KeywordDataList.Users[user.UserID] = user
		err := SaveKeywordList(opts.Ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(opts.Message.From)).
				Str("messageText", opts.Message.Text).
				Msg("Failed to add user and save keyword list")
			return handlerErr.Addf("failed to add user and save keyword list: %w", err).Flat()
		}
	}
	if user.IsNotInit {
		// 用户之前仅在群组内发送命令添加了关键词，但并没有点击机器人来初始化
		user.IsNotInit = false
		KeywordDataList.Users[user.UserID] = user
		buildListenList()
		err := SaveKeywordList(opts.Ctx)
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(opts.Message.From)).
			Str("messageText", opts.Message.Text).
			Msg("Failed to init user and save keyword list")
		return handlerErr.Addf("failed to init user and save keyword list: %w", err).Flat()
	}
	if strings.HasPrefix(opts.Fields[1], "detectkw_addgroup_") {
		groupID := strings.TrimPrefix(opts.Fields[1], "detectkw_addgroup_")
		groupID_int64, err := strconv.ParseInt(groupID, 10, 64)
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(opts.Message.From)).
				Str("messageText", opts.Message.Text).
				Msg("Failed to parse chat ID when user add a group by /start command")
			return handlerErr.Addf("failed to parse chat ID when user add a group by /start command: %w", err).Flat()
		}

		chat := KeywordDataList.Chats[groupID_int64]

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
				Dict(utils.GetChatDict(&opts.Message.Chat)).
				Dict(utils.GetUserDict(opts.Message.From)).
				Msg("User add a chat to listen list by /start command")
			user.ChatsForUser = append(user.ChatsForUser, ChatForUser{
				ChatID: groupID_int64,
			})
			KeywordDataList.Users[opts.Message.From.ID] = user
			pendingMessage = fmt.Sprintf("已添加 <a href=\"https://t.me/c/%s/\">%s</a> 群组\n%s", utils.RemoveIDPrefix(chat.ChatID), chat.ChatName, user.userStatus())
		} else {
			pendingMessage = user.userStatus()
		}

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Message.Chat.ID,
			Text: pendingMessage,
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: buildUserChatList(user),
		})
		if err != nil {
			logger.Error().
				Err(err).
				Dict(utils.GetUserDict(opts.Message.From)).
				Str("content", "added group in user list").
				Msg(flate.SendMessage.Str())
			handlerErr.Addf(flate.SendMessage.Fmt(), "added group in user list", err)
		}
	}
	err := SaveKeywordList(opts.Ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Dict(utils.GetUserDict(opts.Message.From)).
			Str("messageText", opts.Message.Text).
			Msg("Failed to add group for user and save keyword list")
		return handlerErr.Addf("failed to add group for user and save keyword list: %w", err).Flat()
	}

	return handlerErr.Flat()
}

func buildUserChatList(user KeywordUserList) models.ReplyMarkup {
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
