package plugins

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_utils"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

var KeywordDataList KeywordData = KeywordData{
	Chats: map[int64]KeywordChatList{},
	Users: map[int64]KeywordUserList{},
}
var KeywordDataErr error
var KeywordData_path string = consts.DB_path + "detectkeyword/"

func init() {
	ReadKeywordList()
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Detect Keyword",
		Loader: ReadKeywordList,
		Saver:  SaveKeywordList,
	})
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "setkeyword",
		Handler:      addKeywordHandler,
	})
	plugin_utils.AddCallbackQueryCommandPlugins([]plugin_utils.CallbackQuery{
		{
			CommandChar: "detectkw_groupmng",
			Handler:     groupManageCallbackHandler,
		},
		{
			CommandChar: "detectkw_mng",
			Handler:     userManageCallbackHandler,
		},
	}...)
	plugin_utils.AddSlashStartWithPrefixCommandPlugins(plugin_utils.SlashStartWithPrefixHandler{
		Prefix: "detectkw",
		Argument: "addgroup",
		Handler: startPrefixAddGroup,
	})
	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "群组关键词检测",
		Description: "此功能可以检测群组中的每一条信息，当包含设定的关键词时，将会向用户发送提醒\n\n使用方法：\n首先将机器人添加至想要监听关键词的群组中，发送 /setkeyword 命令，等待机器人回应后点击下方的 “设定关键词” 按钮即可为自己添加要监听的群组\n\n设定关键词：您可以在对应的群组中直接发送 <code>/setkeyword 要设定的关键词</code> 来为该群组设定关键词\n或前往机器人聊天页面，发送 <code>/setkeyword</code> 命令后点击对应的群组或全局关键词按钮，根据提示来添加关键词",
		ParseMode:   models.ParseModeHTML,
	})
	buildListenList()
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

func (user KeywordUserList)selectChat() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton
	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "添加为全局关键词",
		CallbackData: "detectkw_mng_globaladding",
	}})
	for _, chat := range user.ChatsForUser {
		targetChat := KeywordDataList.Chats[chat.ChatID]
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: targetChat.ChatName,
			CallbackData: fmt.Sprintf("detectkw_mng_adding_%d", targetChat.ChatID),
		}})
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

type ChatForUser struct {
	ChatID          int64    `yaml:"ChatID"`
	IsDisable       bool     `yaml:"IsDisable,omitempty"`
	IsConfirmDelete bool     `yaml:"IsConfirmDelete,omitempty"`
	Keyword         []string `yaml:"Keyword"`
}

func ReadKeywordList() {
	var lists KeywordData

	file, err := os.Open(KeywordData_path + consts.MetadataFileName)
	if err != nil {
		// 如果是找不到目录，新建一个
		log.Println("[DetectKeyword]: Not found database file. Created new one")
		SaveKeywordList()
		KeywordDataErr = err
		return
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&lists)
	if err != nil {
		if err == io.EOF {
			log.Println("[DetectKeyword]: keyword list looks empty. now format it")
			SaveKeywordList()
			KeywordDataErr = nil
			return
		}
		log.Println("(func)ReadKeywordList:", err)
		KeywordDataErr = err
		return
	}
	KeywordDataList, KeywordDataErr = lists, nil
}

func SaveKeywordList() error {
	data, err := yaml.Marshal(KeywordDataList)
	if err != nil {
		return err
	}

	if _, err := os.Stat(KeywordData_path); os.IsNotExist(err) {
		if err := os.MkdirAll(KeywordData_path, 0755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(KeywordData_path + consts.MetadataFileName); os.IsNotExist(err) {
		_, err := os.Create(KeywordData_path + consts.MetadataFileName)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(KeywordData_path + consts.MetadataFileName, data, 0644)
}

func addKeywordHandler(opts *handler_utils.SubHandlerOpts) {
	if opts.Update.Message.Chat.Type != models.ChatTypePrivate {
		// 在群组中直接使用 /setkeyword 命令
		chat := KeywordDataList.Chats[opts.Update.Message.Chat.ID]
		if chat.IsDisable {
			// 此功能已被管理员手动禁用
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text: "群组管理员已禁用关键词功能，您可以询问管理员以获取更多信息",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "管理此功能",
					CallbackData: "detectkw_groupmng",
				}}}},
			})
			if err != nil {
				log.Printf("Error response /setkeyword command disabled : %v", err)
			}
			return
		} else {
			if chat.AddTime == "" {
				// 初始化群组
				chat = KeywordChatList{
					ChatID: opts.Update.Message.Chat.ID,
					ChatName: opts.Update.Message.Chat.Title,
					ChatUsername: opts.Update.Message.Chat.Username,
					ChatType: opts.Update.Message.Chat.Type,
					AddTime: time.Now().Format(time.RFC3339),
					InitByID: opts.Update.Message.From.ID,
					IsDisable: false,
				}
				KeywordDataList.Chats[opts.Update.Message.Chat.ID] = chat
				SaveKeywordList()
			}
			if len(opts.Fields) == 1 {
				// 只有一个 /setkeyword 命令
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					Text: "已记录群组，点击下方左侧按钮来设定监听关键词\n若您是群组的管理员，您可以点击右侧的按钮来管理此功能",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "设定关键词",
							URL: fmt.Sprintf("https://t.me/%s?start=detectkw_addgroup_%d", consts.BotMe.Username, opts.Update.Message.Chat.ID),
						},
						{
							Text: "管理此功能",
							CallbackData: "detectkw_groupmng",
						},
					}}},
				})
				if err != nil {
					log.Printf("Error response /setkeyword command: %v", err)
				}
			} else {
				user := KeywordDataList.Users[opts.Update.Message.From.ID]

				if user.AddTime == "" {
					// 初始化用户
					user = KeywordUserList{
						UserID: opts.Update.Message.From.ID,
						AddTime: time.Now().Format(time.RFC3339),
						Limit: 50,
						IsNotInit: true,
					}
					KeywordDataList.Users[opts.Update.Message.From.ID] = user
					SaveKeywordList()
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
					log.Println("init group", chat.ChatID, "to user", opts.Update.Message.From.ID)
					chatForUser = ChatForUser{
						ChatID: chat.ChatID,
					}
					user.ChatsForUser = append(user.ChatsForUser, chatForUser)
					KeywordDataList.Users[user.UserID] = user
					SaveKeywordList()
				}

				// 限制关键词长度
				if len(opts.Fields[1]) > 30 {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Update.Message.Chat.ID,
						Text: "抱歉，单个关键词长度不能超过 30 个字符",
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						log.Printf("Error response /setkeyword command keyword too long: %v", err)
					}
					return 
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
					log.Println("add keyword", keyword, "to user", opts.Update.Message.From.ID, "chat", chatForUser.ChatID)
					chatForUser.Keyword = append(chatForUser.Keyword, keyword)
					user.ChatsForUser[chatForUserIndex] = chatForUser
					KeywordDataList.Users[user.UserID] = user
					SaveKeywordList()
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
					ChatID: opts.Update.Message.Chat.ID,
					Text: pendingMessage,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "管理关键词",
						URL: fmt.Sprintf("https://t.me/%s?start=detectkw_addgroup_%d", consts.BotMe.Username, chat.ChatID),
					}}}},
				})
				if err != nil {
					log.Printf("Error response /setkeyword command: %v", err)
				}
			}
			
		}
	} else {
		// 与机器人的私聊对话
		user := KeywordDataList.Users[opts.Update.Message.From.ID]
		if user.AddTime == "" {
			// 初始化用户
			user = KeywordUserList{
				UserID: opts.Update.Message.From.ID,
				AddTime: time.Now().Format(time.RFC3339),
				Limit: 50,
				IsDisable: false,
				IsSilentNotice: false,
			}
			KeywordDataList.Users[opts.Update.Message.From.ID] = user
			SaveKeywordList()
		}
		if len(user.ChatsForUser) == 0 {
			// 没有添加群组
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text: "您还没有添加任何群组，请在群组中使用 `/setkeyword` 命令来记录群组\n若发送信息后没有回应，请检查机器人是否在对应群组中",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				log.Printf("Error response /setkeyword command: %v", err)
			}
			return
		}

		if len(opts.Fields) > 1 {
			if user.AddingChatID != 0 {
				var chatForUser ChatForUser
				var chatForUserIndex int
				for i, c := range user.ChatsForUser {
					if c.ChatID == user.AddingChatID {
						chatForUser = c
						chatForUserIndex = i
					}
				}

				// 限制关键词长度
				if len(opts.Fields[1]) > 30 {
					_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID: opts.Update.Message.Chat.ID,
						Text: "抱歉，单个关键词长度不能超过 30 个字符",
						ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						log.Printf("Error response /setkeyword command keyword too long: %v", err)
					}
					return 
				}

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
						log.Println("add global keyword", keyword, "to user", opts.Update.Message.From.ID)
						user.GlobalKeyword = append(user.GlobalKeyword, keyword)
						KeywordDataList.Users[user.UserID] = user
						SaveKeywordList()
						pendingMessage = fmt.Sprintf("已添加全局关键词: [ %s ]", opts.Fields[1])
					} else {
						pendingMessage = fmt.Sprintf("此全局关键词 [ %s ] 已存在", opts.Fields[1])
					}

				} else {
					targetChat := KeywordDataList.Chats[chatForUser.ChatID]

					// 群组关键词
					for _, k := range chatForUser.Keyword {
						if k == keyword {
							isKeywordExist = true
							break
						}
					}
					if !isKeywordExist {
						log.Println("add keyword", keyword, "to user", opts.Update.Message.From.ID, "chat", chatForUser.ChatID)
						chatForUser.Keyword = append(chatForUser.Keyword, keyword)
						user.ChatsForUser[chatForUserIndex] = chatForUser
						KeywordDataList.Users[user.UserID] = user
						SaveKeywordList()
						pendingMessage = fmt.Sprintf("已为 <a href=\"https://t.me/c/%s/\">%s</a> 群组添加关键词 [ %s ]，您可以继续向此群组添加更多关键词\n", utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName, strings.ToLower(opts.Fields[1]))
					} else {
						pendingMessage = fmt.Sprintf("此关键词 [ %s ] 已存在于 <a href=\"https://t.me/c/%s/\">%s</a> 群组中，您可以继续向此群组添加其他关键词", opts.Fields[1], utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName)
					}
				}
				if isKeywordExist {
					button = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "完成",
						CallbackData: "detectkw_mng_finish",
					}}}}
				} else {
					button = &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "撤销操作",
							CallbackData: fmt.Sprintf("detectkw_mng_undo_%d_%s", user.AddingChatID, opts.Fields[1]),
						},
						{
							Text: "完成",
							CallbackData: "detectkw_mng_finish",
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
					log.Printf("Error response /setkeyword command success: %v", err)
				}
			} else {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					Text: "您还没有选定要将关键词添加到哪个群组，请在下方挑选一个您已经添加的群组",
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: user.selectChat(),
				})
				if err != nil {
					log.Printf("Error response /setkeyword command: %v", err)
				}
			}
		} else {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text: user.userStatus(),
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: buildUserChatList(user),
			})
			if err != nil {
				log.Printf("Error response /setkeyword command: %v", err)
			}
		}
	}
}

func buildListenList() {
	for index, chat := range KeywordDataList.Chats {
		chat.UsersID = []int64{}
		KeywordDataList.Chats[index] = chat
		plugin_utils.RemoveHandlerByChatIDPlugin(chat.ChatID, "detect_keyword")
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
			plugin_utils.AddHandlerByChatIDPlugins(plugin_utils.HandlerByChatID{
				ChatID: chat.ChatID,
				PluginName: "detect_keyword",
				Handler: KeywordDetector,
			})
		}
	}
}

func KeywordDetector(opts *handler_utils.SubHandlerOpts) {
	var text string
	if opts.Update.Message.Caption != "" {
		text = strings.ToLower(opts.Update.Message.Caption)
	} else if opts.Update.Message.Text != "" {
		text = strings.ToLower(opts.Update.Message.Text)
	}

	if text == "" {
		return
	}

	// 先循环一遍，找出该群组中启用此功能的用户 ID
	for _, userID := range KeywordDataList.Chats[opts.Update.Message.Chat.ID].UsersID {
		// 获取用户信息，开始匹配关键词
		user := KeywordDataList.Users[userID]
		if !user.IsDisable && !user.IsNotInit {
			if !user.IsIncludeSelf && opts.Update.Message.From.ID == userID {
				// 如果用户设定排除了自己发送的消息，则跳过
				continue
			}

			// 用户为单独群组设定的关键词
			for _, userKeywordList := range user.ChatsForUser {
				// 判断是否是此群组
				if userKeywordList.ChatID == opts.Update.Message.Chat.ID {
					for _, keyword := range userKeywordList.Keyword {
						if strings.Contains(text, keyword) {
							notifyUser(opts, user, opts.Update.Message.Chat.Title, keyword, text, false)
							break
						}
					}
				}
			}
			// 用户全局设定的关键词
			for _, userGlobalKeyword := range user.GlobalKeyword {
				if strings.Contains(text, userGlobalKeyword) {
					notifyUser(opts, user, opts.Update.Message.Chat.Title, userGlobalKeyword, text, true)
					break
				}
			}
		}
	}
}

func notifyUser(opts *handler_utils.SubHandlerOpts, user KeywordUserList, chatname, keyword, text string, isGlobalKeyword bool) {
	var messageLink string = fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(opts.Update.Message.Chat.ID), opts.Update.Message.ID)

	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: user.UserID,
		Text: fmt.Sprintf("在 <a href=\"https://t.me/c/%s/\">%s</a> 群组中\n来自 %s 的消息\n触发了设定的%s关键词 [ %s ]\n<blockquote expandable>%s</blockquote>",
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
		log.Printf("Error response /setkeyword command: %v", err)
	}
	user.MentionCount++
	KeywordDataList.Users[user.UserID] = user
}

func groupManageCallbackHandler(opts *handler_utils.SubHandlerOpts) {
	if !utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.Update.CallbackQuery.Message.Message.Chat.ID, opts.Update.CallbackQuery.From.ID) {
		opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.Update.CallbackQuery.ID,
			Text: "您没有权限修改此配置",
			ShowAlert: true,
		})
		return
	}

	chat := KeywordDataList.Chats[opts.Update.CallbackQuery.Message.Message.Chat.ID]

	if opts.Update.CallbackQuery.Data == "detectkw_groupmng_switch" {
		// 群组里的全局开关，是否允许群组内用户使用这个功能，优先级最高
		chat.IsDisable = !chat.IsDisable
		KeywordDataList.Chats[opts.Update.CallbackQuery.Message.Message.Chat.ID] = chat
		buildListenList()
		SaveKeywordList()
	}

	_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
		ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: opts.Update.CallbackQuery.Message.Message.ID,
		Text: fmt.Sprintf("消息关键词检测\n此功能允许用户设定一些关键词，当机器人检测到群组内的消息包含用户设定的关键词时，向用户发送提醒\n\n当前群组中有 %d 个用户启用了此功能\n\n%s", len(chat.UsersID),  utils.TextForTrueOrFalse(chat.IsDisable, "已为当前群组关闭关键词检测功能，已设定了关键词的用户将无法再收到此群组的提醒", "")),
		ReplyMarkup: buildGroupManageKB(chat),
	})
	if err != nil {
		fmt.Println(err)
	}
}

func userManageCallbackHandler(opts *handler_utils.SubHandlerOpts) {
	user := KeywordDataList.Users[opts.Update.CallbackQuery.From.ID]

	switch opts.Update.CallbackQuery.Data {
	case "detectkw_mng_globalswitch":
		// 功能全局开关
		user.IsDisable = !user.IsDisable
	case "detectkw_mng_noticeswitch":
		// 是否静默通知
		user.IsSilentNotice = !user.IsSilentNotice
	case "detectkw_mng_selfswitch":
		// 是否检测自己发送的消息
		user.IsIncludeSelf = !user.IsIncludeSelf
	case "detectkw_mng_finish":
		// 停止添加群组关键词
		user.AddingChatID = 0
	case "detectkw_mng_chatdisablebyadmin":
		// 目标群组的管理员为群组关闭了此功能
		opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.Update.CallbackQuery.ID,
			Text:            "此群组的的管理员禁用了此功能，因此，您无法再收到来自该群组的关键词提醒，您可以询问该群组的管理员是否可以重新开启这个功能",
			ShowAlert:       true,
		})
	default:
		if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_undo_") || strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_delkw_") {
		// 撤销添加或删除关键词
			var chatIDAndKeyword string
			if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_undo_") {
				chatIDAndKeyword = strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_undo_")
			} else {
				chatIDAndKeyword = strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_delkw_")
			}
			chatIDAndKeywordList := strings.Split(chatIDAndKeyword, "_")
			chatID, err := strconv.ParseInt(chatIDAndKeywordList[0], 10, 64)
			if err != nil {
				fmt.Println(err)
			}

			if chatID == user.UserID {
				var tempKeyword []string
				for _, keyword := range user.GlobalKeyword {
					if keyword != chatIDAndKeywordList[1] {
						tempKeyword = append(tempKeyword, keyword)
					}
				}
				user.GlobalKeyword = tempKeyword
				KeywordDataList.Users[user.UserID] = user
			} else {
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
			SaveKeywordList()

			if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_undo_") {
				_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.Update.CallbackQuery.Message.Message.ID,
					Text: "已撤销操作，您可以继续使用 <code>/setkeyword 关键词</code> 来添加其他关键词",
					ParseMode: models.ParseModeHTML,
				})
				if err != nil {
					fmt.Println(err)
				}
			} else {
				var buttons [][]models.InlineKeyboardButton
				var tempbutton []models.InlineKeyboardButton
				for _, chat := range user.ChatsForUser {
					if chat.ChatID == chatID {
						for index, keyword := range chat.Keyword {
							if index % 2 == 0 && index != 0 {
								buttons = append(buttons, tempbutton)
								tempbutton = []models.InlineKeyboardButton{}
							}
							tempbutton = append(tempbutton, models.InlineKeyboardButton{
								Text: keyword,
								CallbackData: fmt.Sprintf("detectkw_mng_kw_%d_%s", chat.ChatID, keyword),
							})
							// buttons = append(buttons, tempbutton)
						}
						if len(tempbutton) != 0 {
							buttons = append(buttons, tempbutton)
						}
					}
				}

				var pendingMessage string
				if chatID == user.UserID {
					pendingMessage = fmt.Sprintf("已删除 [ %s ] 关键词\n\n您当前设定了 %d 个全局关键词", chatIDAndKeywordList[1], len(buttons))
				} else {
					pendingMessage = fmt.Sprintf("已删除 [ %s ] 关键词\n\n您当前为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定了 %d 个关键词", chatIDAndKeywordList[1], utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName, len(buttons))
				}

				buttons = append(buttons, []models.InlineKeyboardButton{
					{
						Text: "返回主菜单",
						CallbackData: "detectkw_mng",
					},
					{
						Text: "添加关键词",
						CallbackData: fmt.Sprintf("detectkw_mng_adding_%d", chatID),
					},
				})

				_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.Update.CallbackQuery.Message.Message.ID,
					Text: pendingMessage,
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{
						InlineKeyboard: buttons,
					},
				})
				if err != nil {
					fmt.Println(err)
				}
			}
			
			return
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_adding_") {
		// 设定要往哪个群组里添加关键词
			chatID := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_adding_")
			chatID_int64, err := strconv.ParseInt(chatID, 10, 64)
			if err != nil {
				fmt.Println(err)
			}
			user := KeywordDataList.Users[user.UserID]
			user.AddingChatID = chatID_int64
			KeywordDataList.Users[user.UserID] = user
			buildListenList()
			SaveKeywordList()

			var pendingMessage string
			if chatID_int64 == user.UserID {
				pendingMessage = "已将全局关键词设为添加关键词的目标，请继续使用 <code>/setkeyword 关键词</code> 来添加全局关键词"
			} else {
				pendingMessage = fmt.Sprintf("已将 <a href=\"https://t.me/c/%s/\">%s</a> 群组设为添加关键词的目标群组，请继续使用 <code>/setkeyword 关键词</code> 来为该群组添加关键词", utils.RemoveIDPrefix(chatID_int64), KeywordDataList.Chats[chatID_int64].ChatName)
			}

			_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
				Text: pendingMessage,
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				fmt.Println(err)
			}
			return
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_switch_chat_") {
		// 启用或禁用某个群组的关键词检测开关
			id := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_switch_chat_")
			id_int64, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				fmt.Println(err)
			}
			for index, chat := range KeywordDataList.Users[opts.Update.CallbackQuery.From.ID].ChatsForUser {
				if chat.ChatID == id_int64 {
					chat.IsDisable = !chat.IsDisable
				}
				KeywordDataList.Users[opts.Update.CallbackQuery.From.ID].ChatsForUser[index] = chat
			}
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_chat_") {
		// 显示某个群组的关键词列表
			chatID := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_chat_")
			chatID_int64, err := strconv.ParseInt(chatID, 10, 64)
			if err != nil {
				fmt.Println(err)
			}
			var buttons [][]models.InlineKeyboardButton
			var tempbutton []models.InlineKeyboardButton
			var pendingMessage string

			if chatID_int64 == user.UserID {
				// 全局关键词
				for index, keyword := range user.GlobalKeyword {
					if index % 2 == 0 && index != 0 {
						buttons = append(buttons, tempbutton)
						tempbutton = []models.InlineKeyboardButton{}
					}
					tempbutton = append(tempbutton, models.InlineKeyboardButton{
						Text: keyword,
						CallbackData: fmt.Sprintf("detectkw_mng_kw_%d_%s", user.UserID, keyword),
					})
				}
				if len(tempbutton) != 0 {
					buttons = append(buttons, tempbutton)
				}
				if len(buttons) == 0 {
					pendingMessage = "您没有设定任何全局关键词\n点击下方按钮来添加全局关键词"
				} else {
					pendingMessage = fmt.Sprintf("您当前设定了 %d 个全局关键词\n<blockquote expandable>全局关键词将对您添加的全部群组生效\n但在部分情况下，全局关键词不会生效：\n- 您手动将群组设定为禁用状态\n- 对应群组的管理员为该群组关闭了此功能</blockquote>", len(buttons))
				}
			} else {
				// 为群组设定的关键词
				for _, chat := range KeywordDataList.Users[opts.Update.CallbackQuery.From.ID].ChatsForUser {
					if chat.ChatID == chatID_int64 {
						for index, keyword := range chat.Keyword {
							if index % 2 == 0 && index != 0 {
								buttons = append(buttons, tempbutton)
								tempbutton = []models.InlineKeyboardButton{}
							}
							tempbutton = append(tempbutton, models.InlineKeyboardButton{
								Text: keyword,
								CallbackData: fmt.Sprintf("detectkw_mng_kw_%d_%s", chat.ChatID, keyword),
							})
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
					pendingMessage = fmt.Sprintf("您当前为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定了 %d 个关键词", utils.RemoveIDPrefix(chatID_int64), KeywordDataList.Chats[chatID_int64].ChatName, len(buttons))
				}
			}

			buttons = append(buttons, []models.InlineKeyboardButton{
				{
					Text: "返回主菜单",
					CallbackData: "detectkw_mng",
				},
				{
					Text: "添加关键词",
					CallbackData: fmt.Sprintf("detectkw_mng_adding_%d", chatID_int64),
				},
			})

			_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
				Text: pendingMessage,
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: &models.InlineKeyboardMarkup{
					InlineKeyboard: buttons,
				},
			})
			if err != nil {
				fmt.Println(err)
			}
			return
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_kw_") {
		// 删除某个关键词
			chatIDAndKeyword := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_kw_")
			chatIDAndKeywordList := strings.Split(chatIDAndKeyword, "_")
			chatID, _ := strconv.ParseInt(chatIDAndKeywordList[0], 10, 64)

			var pendingMessage string

			if chatID == user.UserID {
				pendingMessage = fmt.Sprintf("[ %s ] 是您设定的全局关键词", chatIDAndKeywordList[1])
			} else {
				pendingMessage = fmt.Sprintf("[ %s ] 是为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定的关键词", chatIDAndKeywordList[1], utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName)
			}

			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
				Text: pendingMessage,
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
					{
						Text: "返回",
						CallbackData: "detectkw_mng_chat_" + chatIDAndKeywordList[0],
					},
					{
						Text: "删除此关键词",
						CallbackData: "detectkw_mng_delkw_" + chatIDAndKeyword,
					},
				}}},
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				fmt.Println(err)
			}
			return
		}
	}

	_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
		ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
		MessageID: opts.Update.CallbackQuery.Message.Message.ID,
		Text: user.userStatus(),
		ReplyMarkup: buildUserChatList(user),
	})
	if err != nil {
		fmt.Println(err)
	}

	KeywordDataList.Users[opts.Update.CallbackQuery.From.ID] = user
	buildListenList()
	SaveKeywordList()
}

func buildGroupManageKB(chat KeywordChatList) models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "🔄 当前状态: " + utils.TextForTrueOrFalse(chat.IsDisable, "已禁用", "已启用"),
		CallbackData: "detectkw_groupmng_switch",
	}})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

func startPrefixAddGroup(opts *handler_utils.SubHandlerOpts) {
	user := KeywordDataList.Users[opts.Update.Message.From.ID]
	if user.AddTime == "" {
		// 初始化用户
		user = KeywordUserList{
			UserID: opts.Update.Message.From.ID,
			AddTime: time.Now().Format(time.RFC3339),
			Limit: 50,
			IsDisable: false,
			IsSilentNotice: false,
		}
		KeywordDataList.Users[user.UserID] = user
		SaveKeywordList()
	}
	if user.IsNotInit {
		// 用户之前仅在群组内发送了命令，但并没有点击机器人来初始化
		user.IsNotInit = false
		KeywordDataList.Users[user.UserID] = user
		buildListenList()
		SaveKeywordList()
	}
	if strings.HasPrefix(opts.Fields[1], "detectkw_addgroup_") {
		groupID := strings.TrimPrefix(opts.Fields[1], "detectkw_addgroup_")
		groupID_int64, err := strconv.ParseInt(groupID, 10, 64)
		if err != nil {
			fmt.Println("format groupID error:", err)
			return
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
			log.Println("add group", groupID_int64, "to user", opts.Update.Message.From.ID)
			user.ChatsForUser = append(user.ChatsForUser, ChatForUser{
				ChatID: groupID_int64,
			})
			KeywordDataList.Users[opts.Update.Message.From.ID] = user
			pendingMessage = fmt.Sprintf("已添加 <a href=\"https://t.me/c/%s/\">%s</a> 群组\n%s", utils.RemoveIDPrefix(chat.ChatID), chat.ChatName, user.userStatus())
		} else {
			pendingMessage = user.userStatus()
		}

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			Text: pendingMessage,
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: buildUserChatList(user),
		})
		if err != nil {
			fmt.Println(err)
		}
	}
	SaveKeywordList()

}

func buildUserChatList(user KeywordUserList) models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	if !user.IsDisable {
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: fmt.Sprintf("全局关键词 %d 个", len(user.GlobalKeyword)),
			CallbackData:  fmt.Sprintf("detectkw_mng_chat_%d", user.UserID),
		}})

		for _, chat := range user.ChatsForUser {
			var subchats []models.InlineKeyboardButton
			var targetChat = KeywordDataList.Chats[chat.ChatID]
			
			subchats = append(subchats, models.InlineKeyboardButton{
				Text: targetChat.ChatName,
				CallbackData: fmt.Sprintf("detectkw_mng_chat_%d", targetChat.ChatID),
			})

			if targetChat.IsDisable {
				subchats = append(subchats, models.InlineKeyboardButton{
					Text: "🚫 查看帮助",
					CallbackData: "detectkw_mng_chatdisablebyadmin",
				})
			} else {
				subchats = append(subchats, models.InlineKeyboardButton{
					Text: "🔄 " + utils.TextForTrueOrFalse(chat.IsDisable, "当前已禁用 ❌", "当前已启用 ✅"),
					CallbackData: fmt.Sprintf("detectkw_mng_switch_chat_%d", targetChat.ChatID),
				})
			}
			
			buttons = append(buttons, subchats)
		}

		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: "🔄 通知偏好：" + utils.TextForTrueOrFalse(user.IsSilentNotice, "🔇 无声通知", "🔉 有声通知"),
			CallbackData: "detectkw_mng_noticeswitch",
		}})
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: "🔄 检测偏好：" + utils.TextForTrueOrFalse(user.IsIncludeSelf, "不排除自己的消息", "排除自己的消息"),
			CallbackData: "detectkw_mng_selfswitch",
		}})
	}

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "🔄 全局状态：" + utils.TextForTrueOrFalse(user.IsDisable, "已禁用 ❌", "已启用 ✅"),
		CallbackData: "detectkw_mng_globalswitch",
	}})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}
