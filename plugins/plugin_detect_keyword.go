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
var KeywordDataErr  error

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
	plugin_utils.AddCallbackQueryCommandPlugins(plugin_utils.Plugin_CallbackQuery{
		CommandChar: "detectkeyword_",
		Handler:     callbackHandler,
	})
	plugin_utils.AddSlashStartWithPrefixCommandPlugins(plugin_utils.SlashStartWithPrefixHandler{
		Prefix: "detectkeyword",
		Argument: "addgroup",
		Handler: startPrefixAddGroup,
	})
}

type KeywordData struct {
	Chats map[int64]KeywordChatList `yaml:"Chat"`
	Users map[int64]KeywordUserList `yaml:"User"`
}

type KeywordChatList struct {
	ChatID       int64           `yaml:"ChatID"`
	ChatName     string          `yaml:"ChatName"`
	ChatUsername string          `yaml:"ChatUsername"`
	ChatType     models.ChatType `yaml:"ChatType"`
	AddTime      string          `yaml:"AddTime"`
	InitByID     int64           `yaml:"InitByID"`
	IsDisable    bool            `yaml:"IsDisable"`
	// 根据用户数量决定是否启用
	UsersID      []int64         `yaml:"UsersID"`
}

type KeywordUserList struct {
	UserID         int64         `yaml:"UserID"`
	AddTime        string        `yaml:"AddTime"`
	Limit          int           `yaml:"Limit"`
	IsEnabled      bool          `yaml:"IsEnabled"`
	IsSilentNotice bool          `yaml:"IsSilentNotice"`
	Keywords       []KeywordItem `yaml:"Keywords"`
}

type KeywordItem struct {
	ChatID    int64    `yaml:"ChatID"`
	IsEnabled bool     `yaml:"IsEnabled"`
	Keyword   []string `yaml:"Keyword"`
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
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text: "群组管理员已禁用关键词功能，您可以询问管理员以获取更多信息",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ParseMode: models.ParseModeHTML,
				ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "管理此功能",
					CallbackData: "detectkeyword_groupmanage",
				}}}},
			})
			if err != nil {
				log.Printf("Error response /setkeyword command disabled : %v", err)
			}
			return
		} else {
			if chat.AddTime == "" {
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
						URL: fmt.Sprintf("https://t.me/%s?start=detectkeyword_addgroup_%d", consts.BotMe.Username, opts.Update.Message.Chat.ID),
					},
					{
						Text: "管理此功能",
						CallbackData: "detectkeyword_groupmanage",
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
			user = KeywordUserList{
				UserID: opts.Update.Message.From.ID,
				AddTime: time.Now().Format(time.RFC3339),
				Limit: 50,
				IsEnabled: false,
				IsSilentNotice: false,
			}
			KeywordDataList.Users[opts.Update.Message.From.ID] = user
			SaveKeywordList()
		}
		if len(user.Keywords) == 0 {
			_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text: "您还没有添加任何群组，请在群组中使用 /setkeyword 来记录群组\n若发送信息后没有回应，请检查机器人是否在对应群组中",
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				log.Printf("Error response /setkeyword command: %v", err)
			}
		}
	}
	
}

func buildListenList() {
	for _, user := range KeywordDataList.Users {
		if user.IsEnabled {
			for _, key := range user.Keywords {
				chat := KeywordDataList.Chats[key.ChatID]
				chat.UsersID = append(chat.UsersID, user.UserID)
				KeywordDataList.Chats[key.ChatID] = chat
			}
		}
		
	}
}

func keywordDetector(opts *handler_utils.SubHandlerOpts) {
	// 先循环一遍，找出该群组中启用此功能的用户 ID
	for _, userID := range KeywordDataList.Chats[opts.Update.Message.Chat.ID].UsersID {
		// 获取用户信息，开始匹配关键词
		user := KeywordDataList.Users[userID]
		if user.IsEnabled {
			for _, keywords := range user.Keywords {
				// 判断是否是此群组
				if keywords.ChatID == opts.Update.Message.Chat.ID {
					if opts.Update.Message.Caption != "" {
						for _, keyword := range keywords.Keyword {
							if strings.Contains(opts.Update.Message.Caption, keyword) {
								notifyUser(opts, user, opts.Update.Message.Chat.Title, keyword)
								break
							}
						}
					} else if opts.Update.Message.Text != "" {
						for _, keyword := range keywords.Keyword {
							if strings.Contains(opts.Update.Message.Text, keyword) {
								notifyUser(opts, user, opts.Update.Message.Chat.Title, keyword)
								break
							}
						}
					}
				}
			}
		}
	}
}

func notifyUser(opts *handler_utils.SubHandlerOpts, user KeywordUserList, chatname, keyword string) {
	var messageLink string = fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(opts.Update.Message.Chat.ID), opts.Update.Message.ID)

	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: user.UserID,
		Text: fmt.Sprintf("在 <b>%s</b> 中有消息触发了设定的关键词 [%s]\n<blockquote>%s</blockquote>", chatname, keyword, opts.Update.Message.Text),
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

func callbackHandler(opts *handler_utils.SubHandlerOpts) {
	if !utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.Update.CallbackQuery.Message.Message.Chat.ID, opts.Update.CallbackQuery.From.ID) {
		opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.Update.CallbackQuery.ID,
			Text: "您没有权限修改此配置",
			ShowAlert: true,
		})
		return
	}

	chat := KeywordDataList.Chats[opts.Update.CallbackQuery.Message.Message.Chat.ID]

	switch opts.Update.CallbackQuery.Data {
	case "detectkeyword_groupmanage":
		opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text: "消息关键词检测\n",
			ReplyMarkup: buildGroupManageKB(chat),
		})
	case "detectkeyword_groupmanage_switch":
		chat.IsDisable = !chat.IsDisable
		_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text: "消息关键词检测\n",
			ReplyMarkup: buildGroupManageKB(chat),
		})
		if err != nil {
			fmt.Println(err)
		}
	default:
		
	}
	
	KeywordDataList.Chats[opts.Update.CallbackQuery.Message.Message.Chat.ID] = chat
	SaveKeywordList()
}

func showChatStatus(IsDisable bool) string {
	if IsDisable {
		return "此功能已禁用"
	} else {
		return "此功能已启用"
	}
}

func buildGroupManageKB(chat KeywordChatList) models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	buttons = append(buttons, []models.InlineKeyboardButton{
		{
			Text: showChatStatus(chat.IsDisable),
			CallbackData: "detectkeyword_groupmanage_switch",
		},
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

func startPrefixAddGroup(opts *handler_utils.SubHandlerOpts) {
	user := KeywordDataList.Users[opts.Update.Message.From.ID]
	if strings.HasPrefix(opts.Fields[1], "detectkeyword_addgroup_") {
		groupID := strings.TrimPrefix(opts.Fields[1], "detectkeyword_addgroup_")
		groupID_int64, err := strconv.ParseInt(groupID, 10, 64)
		if err != nil {
			fmt.Println("format groupID error:", err)
			return
		}
		var IsAdded bool = false
		for _, keyword := range user.Keywords {
			if keyword.ChatID == groupID_int64 {
				IsAdded = true
				break
			}
		}
		if !IsAdded {
			log.Println("add group", groupID_int64, "to user", opts.Update.Message.From.ID)
			user.Keywords = append(user.Keywords, KeywordItem{
				ChatID: groupID_int64,
				IsEnabled: false,
			})
		}
		KeywordDataList.Users[opts.Update.Message.From.ID] = user

		chat := KeywordDataList.Chats[groupID_int64]

		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			Text: fmt.Sprintf("已添加 <a href=\"https://t.me/c/%s/\">%s</a> 群组", utils.RemoveIDPrefix(chat.ChatID), chat.ChatName),
			ParseMode: models.ParseModeHTML,
		})
		if err != nil {
			fmt.Println(err)
		}
	}
	SaveKeywordList()

}
