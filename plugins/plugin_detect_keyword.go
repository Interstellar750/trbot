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
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.Plugin_SlashSymbolCommand{
		SlashCommand: "setkeyword",
		Handler:      addKeywordHandler,
	})
	plugin_utils.AddCallbackQueryCommandPlugins([]plugin_utils.Plugin_CallbackQuery{
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
	// IsAdding        bool          `yaml:"IsAdding,omitempty"`
	AddingChatID    int64         `yaml:"AddingChatID,omitempty"`
	IsDisable       bool          `yaml:"IsDisable,omitempty"`
	IsSilentNotice  bool          `yaml:"IsSilentNotice,omitempty"`
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
		pendingMessage = fmt.Sprintf("您添加的群组中有 (%d/%d) 个处于启用状态\n您总共设定了 %d 个关键词", user.enabledChatCount(), len(user.ChatsForUser), user.keywordCount())
	}

	return pendingMessage
}

func (user KeywordUserList)selectChat() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton
	for _, chat := range user.ChatsForUser {
		targetChat := KeywordDataList.Chats[chat.ChatID]
		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text: targetChat.ChatName,
				CallbackData: fmt.Sprintf("detectkw_mng_adding_%d", targetChat.ChatID),
			},
		})
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
			return
		}
	} else {
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
				var chat ChatForUser
				var chatindex int
				for i, c := range user.ChatsForUser {
					if c.ChatID == user.AddingChatID {
						chatindex = i
						chat = c
					}
				}
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
				chat.Keyword = append(chat.Keyword, strings.ToLower(opts.Fields[1]))
				user.ChatsForUser[chatindex] = chat
				KeywordDataList.Users[opts.Update.Message.From.ID] = user
				SaveKeywordList()
				targetChat := KeywordDataList.Chats[chat.ChatID]
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					Text: fmt.Sprintf("已为 <a href=\"https://t.me/c/%s/\">%s</a> 群组添加关键词 [%s]，您可以继续向此群组添加更多关键词\n", utils.RemoveIDPrefix(targetChat.ChatID), targetChat.ChatName, strings.ToLower(opts.Fields[1])),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					ParseMode: models.ParseModeHTML,
					ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "撤销操作",
							CallbackData: fmt.Sprintf("detectkw_mng_undo_%d_%s", chat.ChatID, opts.Fields[1]),
						},
						{
							Text: "完成",
							CallbackData: "detectkw_mng_finish",
						},
					}}},
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
		if !chat.IsDisable {
			chat.UsersID = []int64{}
			KeywordDataList.Chats[index] = chat
		}
	}
	for _, user := range KeywordDataList.Users {
		if !user.IsDisable {
			for _, key := range user.ChatsForUser {
				if !key.IsDisable {
					chat := KeywordDataList.Chats[key.ChatID]
					chat.UsersID = append(chat.UsersID, user.UserID)
					KeywordDataList.Chats[key.ChatID] = chat
				}
			}
		}
		
	}
}

func KeywordDetector(opts *handler_utils.SubHandlerOpts) {
	// 先循环一遍，找出该群组中启用此功能的用户 ID
	for _, userID := range KeywordDataList.Chats[opts.Update.Message.Chat.ID].UsersID {
		// 获取用户信息，开始匹配关键词
		user := KeywordDataList.Users[userID]
		if !user.IsDisable {
			for _, keywords := range user.ChatsForUser {
				// 判断是否是此群组
				if keywords.ChatID == opts.Update.Message.Chat.ID {
					if opts.Update.Message.Caption != "" {
						text := strings.ToLower(opts.Update.Message.Caption)
						for _, keyword := range keywords.Keyword {
							if strings.Contains(text, keyword) {
								notifyUser(opts, user, opts.Update.Message.Chat.Title, keyword, text)
								break
							}
						}
					} else if opts.Update.Message.Text != "" {
						text := strings.ToLower(opts.Update.Message.Text)
						for _, keyword := range keywords.Keyword {
							if strings.Contains(text, keyword) {
								notifyUser(opts, user, opts.Update.Message.Chat.Title, keyword, text)
								break
							}
						}
					}
				}
			}
		}
	}
}

func notifyUser(opts *handler_utils.SubHandlerOpts, user KeywordUserList, chatname, keyword, text string) {
	var messageLink string = fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(opts.Update.Message.Chat.ID), opts.Update.Message.ID)
	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: user.UserID,
		Text: fmt.Sprintf("在 <a href=\"https://t.me/c/%s/\">%s</a> 群组中有消息触发了设定的关键词 [%s]\n<blockquote>%s</blockquote>", utils.RemoveIDPrefix(opts.Update.Message.Chat.ID), chatname, keyword, text),
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
		chat.IsDisable = !chat.IsDisable
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
	
	KeywordDataList.Chats[opts.Update.CallbackQuery.Message.Message.Chat.ID] = chat
	SaveKeywordList()
}

func userManageCallbackHandler(opts *handler_utils.SubHandlerOpts) {
	user := KeywordDataList.Users[opts.Update.CallbackQuery.From.ID]

	switch opts.Update.CallbackQuery.Data {
	case "detectkw_mng_globalswitch":
		user.IsDisable = !user.IsDisable
	case "detectkw_mng_noticeswitch":
		user.IsSilentNotice = !user.IsSilentNotice
	case "detectkw_mng_finish":
		user.AddingChatID = 0
	default:
		if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_undo_") || strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_delkw_") {
			var idAndKeyword string
			if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_undo_") {
				idAndKeyword = strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_undo_")
			} else {
				idAndKeyword = strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_delkw_")
			}
			idAndKeywordList := strings.Split(idAndKeyword, "_")
			chatID, err := strconv.ParseInt(idAndKeywordList[0], 10, 64)
			if err != nil {
				fmt.Println(err)
			}
			for index, chat := range KeywordDataList.Users[opts.Update.CallbackQuery.From.ID].ChatsForUser {
				
				if chat.ChatID == chatID {
					var tempKeyword []string
					for _, keyword := range chat.Keyword {
						if keyword != idAndKeywordList[1] {
							tempKeyword = append(tempKeyword, keyword)
						}
					}
					chat.Keyword = tempKeyword
				}
				KeywordDataList.Users[opts.Update.CallbackQuery.From.ID].ChatsForUser[index] = chat
				SaveKeywordList()
			}
			if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_undo_") {
				_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID: opts.Update.CallbackQuery.From.ID,
					MessageID: opts.Update.CallbackQuery.Message.Message.ID,
					Text: "已撤销操作",
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
								CallbackData: fmt.Sprintf("detectkw_mng_keyword_%d_%s", chat.ChatID, keyword),
							})
							// buttons = append(buttons, tempbutton)
						}
						if len(tempbutton) != 0 {
							buttons = append(buttons, tempbutton)
						}
					}
				}

				buttons = append(buttons, []models.InlineKeyboardButton{{
					Text: "返回上一级",
					CallbackData: "detectkw_mng_chat_" + idAndKeywordList[0],
				}})

				_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID: opts.Update.CallbackQuery.From.ID,
					MessageID: opts.Update.CallbackQuery.Message.Message.ID,
					Text: fmt.Sprintf("已删除 [%s] 关键词\n\n您当前为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定了 %d 个关键词", idAndKeywordList[1], utils.RemoveIDPrefix(chatID), KeywordDataList.Chats[chatID].ChatName, len(buttons) - 1),
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
			id := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_adding_")
			id_int64, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				fmt.Println(err)
			}
			user := KeywordDataList.Users[opts.Update.CallbackQuery.From.ID]
			user.AddingChatID = id_int64
			KeywordDataList.Users[opts.Update.CallbackQuery.From.ID] = user
			buildListenList()
			SaveKeywordList()

			_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.Update.CallbackQuery.From.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("已将 <a href=\"https://t.me/c/%s/\">%s</a> 群组设为添加关键词的目标群组，请使用 <code>/setkeyword 关键词</code> 来为该群组添加关键词", utils.RemoveIDPrefix(id_int64), KeywordDataList.Chats[id_int64].ChatName),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				fmt.Println(err)
			}
			return
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_switch_chat_") {
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
			id := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_chat_")
			id_int64, err := strconv.ParseInt(id, 10, 64)
			if err != nil {
				fmt.Println(err)
			}
			var buttons [][]models.InlineKeyboardButton
			var tempbutton []models.InlineKeyboardButton
			for _, chat := range KeywordDataList.Users[opts.Update.CallbackQuery.From.ID].ChatsForUser {
				if chat.ChatID == id_int64 {
					for index, keyword := range chat.Keyword {
						if index % 2 == 0 && index != 0 {
							buttons = append(buttons, tempbutton)
							tempbutton = []models.InlineKeyboardButton{}
						}
						tempbutton = append(tempbutton, models.InlineKeyboardButton{
							Text: keyword,
							CallbackData: fmt.Sprintf("detectkw_mng_keyword_%d_%s", chat.ChatID, keyword),
						})
						// buttons = append(buttons, tempbutton)
					}
					if len(tempbutton) != 0 {
						buttons = append(buttons, tempbutton)
					}
				}
			}

			var pendingMessage string
			if len(buttons) == 0 {
				pendingMessage = fmt.Sprintf("当前群组 <a href=\"https://t.me/c/%s/\">%s</a> 没有关键词\n点击下方按钮来为此群组添加关键词", utils.RemoveIDPrefix(id_int64), KeywordDataList.Chats[id_int64].ChatName)
				buttons = append(buttons, []models.InlineKeyboardButton{{
					Text: "添加关键词",
					CallbackData: fmt.Sprintf("detectkw_mng_adding_%d", id_int64),
				}})
			} else {
				pendingMessage = fmt.Sprintf("您当前为 <a href=\"https://t.me/c/%s/\">%s</a> 群组设定了 %d 个关键词", utils.RemoveIDPrefix(id_int64), KeywordDataList.Chats[id_int64].ChatName, len(buttons))
			}

			buttons = append(buttons, []models.InlineKeyboardButton{{
				Text: "返回上一级",
				CallbackData: "detectkw_mng",
			}})


			_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.Update.CallbackQuery.From.ID,
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
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_keyword_") {
			idAndKeyword := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "detectkw_mng_keyword_")
			idAndKeywordList := strings.Split(idAndKeyword, "_")
			id_int64, _ := strconv.ParseInt(idAndKeywordList[0], 10, 64)

			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.Update.CallbackQuery.From.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
				Text: fmt.Sprintf("[%s] 是为群组 <a href=\"https://t.me/c/%s/\">%s</a> 设定的关键词", idAndKeywordList[1], utils.RemoveIDPrefix(id_int64), KeywordDataList.Chats[id_int64].ChatName),
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
					{
						Text: "返回",
						CallbackData: "detectkw_mng_chat_" + idAndKeywordList[0],
					},
					{
						Text: "删除此关键词",
						CallbackData: "detectkw_mng_delkw_" + idAndKeyword,
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
		ChatID: opts.Update.CallbackQuery.From.ID,
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
		KeywordDataList.Users[opts.Update.Message.From.ID] = user
		SaveKeywordList()
	}
	if strings.HasPrefix(opts.Fields[1], "detectkw_addgroup_") {
		groupID := strings.TrimPrefix(opts.Fields[1], "detectkw_addgroup_")
		groupID_int64, err := strconv.ParseInt(groupID, 10, 64)
		if err != nil {
			fmt.Println("format groupID error:", err)
			return
		}
		var IsAdded bool = false
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
		}
		KeywordDataList.Users[opts.Update.Message.From.ID] = user

		chat := KeywordDataList.Chats[groupID_int64]

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			Text: fmt.Sprintf("已添加 <a href=\"https://t.me/c/%s/\">%s</a> 群组\n%s", utils.RemoveIDPrefix(chat.ChatID), chat.ChatName, user.userStatus()),
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
				Text: "🔄 " + utils.TextForTrueOrFalse(chat.IsDisable, "已禁用 ❌", "已启用 ✅"),
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
		Text: "🔄 全局状态：" + utils.TextForTrueOrFalse(user.IsDisable, "已禁用", "已启用"),
		CallbackData: "detectkw_mng_globalswitch",
	}})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}
