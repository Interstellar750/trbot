package plugins

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"strings"
	"time"

	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_utils"
	"trbot/utils/plugin_utils"
	"trbot/utils/updatetype"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

var LimitMessageList map[int64]AllowMessages
var LimitMessageErr  error

var LimitMessage_path string = consts.DB_path + "limitmessage/"

type AllowMessages struct {
	IsEnable            bool                        `yaml:"IsEnable"`
	AddTime             string                      `yaml:"AddTime"`
	MessageType         updatetype.MessageType      `yaml:"MessageType"`
	IsWhiteForType      bool                        `yaml:"IsWhiteForType"`
	MessageAttribute    updatetype.MessageAttribute `yaml:"MessageAttribute"`
	IsWhiteForAttribute bool                        `yaml:"IsWhiteForAttribute"`
}

func SaveLimitMessageList() error {
	data, err := yaml.Marshal(LimitMessageList)
	if err != nil { return err }

	if _, err := os.Stat(LimitMessage_path); os.IsNotExist(err) {
		if err := os.MkdirAll(LimitMessage_path, 0755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(LimitMessage_path + consts.MetadataFileName); os.IsNotExist(err) {
		_, err := os.Create(LimitMessage_path + consts.MetadataFileName)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(LimitMessage_path + consts.MetadataFileName, data, 0644)
}

func ReadLimitMessageList() {
	var limitMessageList map[int64]AllowMessages

	file, err := os.Open(LimitMessage_path + consts.MetadataFileName)
	if err != nil {
		// 如果是找不到目录，新建一个
		log.Println("[LimitMessage]: Not found database file. Created new one")
		SaveLimitMessageList()
		LimitMessageList, LimitMessageErr = map[int64]AllowMessages{}, err
		return
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&limitMessageList)
	if err != nil {
		if err == io.EOF {
			log.Println("[LimitMessage]: Udonese list looks empty. now format it")
			SaveLimitMessageList()
			LimitMessageList, LimitMessageErr = map[int64]AllowMessages{}, nil
			return
		}
		log.Println("(func)ReadLimitMessageList:", err)
		LimitMessageList, LimitMessageErr = map[int64]AllowMessages{}, err
		return
	}
	LimitMessageList, LimitMessageErr = limitMessageList, nil
}

func SomeMessageOnlyHandler(opts *handler_utils.SubHandlerOpts) {
	if opts.Update.Message.Chat.Type == "private" {
		opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Update.Message.Chat.ID,
			Text:            "此功能被设计为仅在群组中可用",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
		})
	} else if utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.Update.Message.Chat.ID, opts.Update.Message.From.ID) {
		thisChat := LimitMessageList[opts.Update.Message.Chat.ID]

		if utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.Update.Message.Chat.ID, consts.BotMe.ID) && utils.UserHavePermissionDeleteMessage(opts.Ctx, opts.Thebot, opts.Update.Message.Chat.ID, consts.BotMe.ID) {
			var pendingMessage string = "限制消息功能，当前"
			if thisChat.IsEnable {
				pendingMessage += "已启用"
			} else {
				pendingMessage += "已禁用"
			}

			opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text:   pendingMessage,
				ReplyMarkup: buildMessageAllKB(thisChat),
			})
			opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				MessageID: opts.Update.Message.ID,
			})
			LimitMessageList[opts.Update.Message.Chat.ID] = thisChat
			SaveLimitMessageList()
		} else {
			opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				Text:   "启用此功能前，请先将机器人设为管理员\n如果还是提示本消息，请检查机器人是否有删除消息的权限",
			})
		}
	} else {
		botMessage, _ := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			Text:   "抱歉，您不是群组的管理员，无法为群组更改此功能",
		})
		time.Sleep(time.Second * 5)
		opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
			ChatID: opts.Update.Message.Chat.ID,
			MessageIDs: []int{
				opts.Update.Message.ID,
				botMessage.ID,
			},
		})
	}
}

func DeleteNotAllowMessage(opts *handler_utils.SubHandlerOpts) {
	
	var deleteAction bool
	if utils.AnyContains(opts.Update.Message.Chat.Type, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 处理消息删除逻辑，只有当群组启用该功能时才处理
		thisChat := LimitMessageList[opts.Update.Message.Chat.ID]
		if thisChat.IsEnable {
			this := updatetype.GetMessageType(opts.Update.Message)
			thisattribute  := updatetype.GetMessageAttribute(opts.Update.Message)

			// 根据规则的黑白名单选择判断逻辑
			if thisChat.IsWhiteForType == thisChat.IsWhiteForAttribute {
				deleteAction = CheckMessageType(this, thisChat.MessageType, thisChat.IsWhiteForType) || CheckMessageAttribute(thisattribute, thisChat.MessageAttribute, thisChat.IsWhiteForAttribute)
			} else {
				deleteAction = CheckMessageType(this, thisChat.MessageType, thisChat.IsWhiteForType) && CheckMessageAttribute(thisattribute, thisChat.MessageAttribute, thisChat.IsWhiteForAttribute)
			}

			if deleteAction {
				_, err := opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					MessageID: opts.Update.Message.ID,
				})
				if err != nil {
					log.Printf("Failed to delete message: %v", err)
				} else {
					log.Printf("Deleted message from %d in %d: %s\n", opts.Update.Message.From.ID, opts.Update.Message.Chat.ID, opts.Update.Message.Text)
				}
			}
		}
	}
}

func CheckMessageType(this, target updatetype.MessageType, IsWhiteList bool) bool {
	var delete bool = IsWhiteList

	v1 := reflect.ValueOf(this)
	v2 := reflect.ValueOf(target)
	t  := reflect.TypeOf(this)

	for i := 0; i < v1.NumField(); i++ {
		field := t.Field(i)
		val1 := v1.Field(i).Interface()
		val2 := v2.Field(i).Interface()

		if val1 == true && val1 == val2 {
			if IsWhiteList {
				fmt.Printf("白名单 消息类型 %s 不删除\n", field.Name)
				delete = false
			} else {
				fmt.Printf("黑名单 消息类型 %s 删除\n", field.Name)
				delete = true
			}
		} else if val1 == true && val1 != val2 {
			if IsWhiteList {
				fmt.Printf("白名单 ")
			} else {
				fmt.Printf("黑名单 ")
			}
			fmt.Printf("未命中 消息类型 %s 遵循默认规则 ", field.Name)
			if delete {
				fmt.Println("删除")
			} else {
				fmt.Println("不删除")
			}
		}
	}
	return delete
}

func CheckMessageAttribute(this, target updatetype.MessageAttribute, IsWhiteList bool) bool {
	var delete bool = IsWhiteList
	var noAttribute bool = true // 如果没有命中任何消息属性，提示内容，根据黑白名单判断是否删除

	v1 := reflect.ValueOf(this)
	v2 := reflect.ValueOf(target)
	t := reflect.TypeOf(this)

	for i := 0; i < v1.NumField(); i++ {
		field := t.Field(i)
		val1 := v1.Field(i).Interface()
		val2 := v2.Field(i).Interface()


		if val1 == true && val1 == val2 {
			noAttribute = false
			if IsWhiteList {
				fmt.Printf("白名单 消息属性 %s 不删除\n", field.Name)
				delete = false
			} else {
				fmt.Printf("黑名单 消息属性 %s 删除\n", field.Name)
				delete = true
			}
		} else if val1 == true && val1 != val2 {
			noAttribute = false
			if IsWhiteList {
				fmt.Printf("白名单 ")
			} else {
				fmt.Printf("黑名单 ")
			}
			fmt.Printf("未命中 消息属性 %s 遵循默认规则 ", field.Name)
			if delete {
				fmt.Println("删除")
			} else {
				fmt.Println("不删除")
			}
		}
	}
	if noAttribute {
		if IsWhiteList {
			fmt.Printf("白名单 ")
		} else {
			fmt.Printf("黑名单 ")
		}
		fmt.Printf("未命中 消息属性 无 遵循默认规则 ")
		if delete {
			fmt.Println("删除")
		} else {
			fmt.Println("不删除")
		}
	}
	return delete
}

func init() {
	ReadLimitMessageList()
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.Plugin_SlashSymbolCommand{
		SlashCommand: "limitmessage",
		Handler: SomeMessageOnlyHandler,
	})
	plugin_utils.AddCallbackQueryCommandPlugins(plugin_utils.Plugin_CallbackQuery{
		CommandChar: "limitmsg_",
		Handler: LimitMessageCallback,
	})
}

func buttonText(text string, opt bool) string {
	if opt {
		return "✅ " + text
	}

	return "❌ " + text
}

func buttonWhiteBlackRule(opt bool) string {
	if opt {
		return "白名单模式"
	}

	return "黑名单模式"
}

func buttonIsEnable(opt bool) string {
	if opt {
		return "已启用功能，点击关闭"
	}

	return "已关闭功能，点击启用"
}

func buildMessageTypeKB(msgType updatetype.MessageType) models.ReplyMarkup {

	var msgTypeItems [][]models.InlineKeyboardButton
	var msgTypeItemsTemp []models.InlineKeyboardButton

	v := reflect.ValueOf(msgType) // 解除指针获取值
	t := reflect.TypeOf(msgType)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if i % 2 == 0 && i != 0 {
			msgTypeItems = append(msgTypeItems, msgTypeItemsTemp)
			msgTypeItemsTemp = []models.InlineKeyboardButton{}
		}
		msgTypeItemsTemp = append(msgTypeItemsTemp, models.InlineKeyboardButton{
			Text:         buttonText(field.Name, value.Bool()),
			CallbackData: "limitmsg_type_" + field.Name,
		})
	}


	msgTypeItems = append(msgTypeItems, []models.InlineKeyboardButton{{
		Text: "返回上一级",
		CallbackData: "limitmsg_back",
	}})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: msgTypeItems,
	}

	return kb
}

func buildMessageAttributeKB(msgAttribute updatetype.MessageAttribute) models.ReplyMarkup {

	var msgAttributeItems [][]models.InlineKeyboardButton
	var msgAttributeItemsTemp []models.InlineKeyboardButton

	v := reflect.ValueOf(msgAttribute) // 解除指针获取值
	t := reflect.TypeOf(msgAttribute)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if i % 2 == 0 && i != 0 {
			msgAttributeItems = append(msgAttributeItems, msgAttributeItemsTemp)
			msgAttributeItemsTemp = []models.InlineKeyboardButton{}
		}
		msgAttributeItemsTemp = append(msgAttributeItemsTemp, models.InlineKeyboardButton{
			Text:         buttonText(field.Name, value.Bool()),
			CallbackData: "limitmsg_attr_" + field.Name,
		})
	}


	msgAttributeItems = append(msgAttributeItems, []models.InlineKeyboardButton{{
		Text: "返回上一级",
		CallbackData: "limitmsg_back",
	}})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: msgAttributeItems,
	}

	return kb
}

func buildMessageAllKB(chat AllowMessages) models.ReplyMarkup {
	var chatAllow [][]models.InlineKeyboardButton

	chatAllow = append(chatAllow, []models.InlineKeyboardButton{
		{
			Text: "选择消息类型",
			CallbackData: "limitmsg_typekb",
		},
		{
			Text: "<- " + buttonWhiteBlackRule(chat.IsWhiteForType),
			CallbackData: "limitmsg_typekb_switchrule",
		},
	})

	chatAllow = append(chatAllow, []models.InlineKeyboardButton{
		{
			Text: "选择消息属性",
			CallbackData: "limitmsg_attrkb",
		},
		{
			Text: buttonWhiteBlackRule(chat.IsWhiteForAttribute),
			CallbackData: "limitmsg_attrkb_switchrule",
		},
	})

	chatAllow = append(chatAllow, []models.InlineKeyboardButton{
		{
			Text: "关闭菜单",
			CallbackData: "limitmsg_done",
		},
		{
			Text: buttonIsEnable(chat.IsEnable),
			CallbackData: "limitmsg_switchenable",
		},
	})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: chatAllow,
	}

	return kb
}

func LimitMessageCallback(opts *handler_utils.SubHandlerOpts) {
	thisChat := LimitMessageList[opts.Update.CallbackQuery.Message.Message.Chat.ID]

	switch opts.Update.CallbackQuery.Data {
	case "limitmsg_typekb":
		// opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
		// 	CallbackQueryID: opts.Update.CallbackQuery.ID,
		// 	Text: "已选择消息类型",
		// })
		opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text: "当前消息类型：" + buttonWhiteBlackRule(thisChat.IsWhiteForType),
			ReplyMarkup: buildMessageTypeKB(thisChat.MessageType),
		})
		
	case "limitmsg_typekb_switchrule":
		thisChat.IsWhiteForType = !thisChat.IsWhiteForType
		opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			ReplyMarkup: buildMessageAllKB(thisChat),
		})
	case "limitmsg_attrkb":
		opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text: "当前消息属性：" + buttonWhiteBlackRule(thisChat.IsWhiteForAttribute),
			ReplyMarkup: buildMessageAttributeKB(thisChat.MessageAttribute),
		})
	case "limitmsg_attrkb_switchrule":
		thisChat.IsWhiteForAttribute = !thisChat.IsWhiteForAttribute
		opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			ReplyMarkup: buildMessageAllKB(thisChat),
		})
	case "limitmsg_back":
		var pendingMessage string = "限制消息功能，当前"
		if thisChat.IsEnable {
			pendingMessage += "已启用"
		} else {
			pendingMessage += "已禁用"
		}

		opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text: pendingMessage,
			ReplyMarkup: buildMessageAllKB(thisChat),
		})
	case "limitmsg_done":
		opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
		})
	case "limitmsg_switchenable":
		thisChat.IsEnable = !thisChat.IsEnable
		opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			ReplyMarkup: buildMessageAllKB(thisChat),
		})
	// 	opts.CallbackQuery.AnswerCallbackQuery()
	// 	opts.CallbackQuery.EditMessageReplyMarkup(buildMessageAttrKB(opts.Chat))
	// case "limitmsg_switchrule":
	// 	opts.CallbackQuery.AnswerCallbackQuery()
	// 	opts.CallbackQuery.EditMessageReplyMarkup(buildMessageAllKB(opts.Chat))
	// case "limitmsg_done":
	// 	opts.CallbackQuery.AnswerCallbackQuery()
	// 	opts.CallbackQuery.EditMessageReplyMarkup(buildMessageAllKB(opts.Chat))
	default:
		if strings.HasPrefix(opts.Update.CallbackQuery.Data, "limitmsg_type_") {
			callbackField := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "limitmsg_type_")
	
			data := thisChat.MessageType
			v := reflect.ValueOf(data) // 解除指针获取值
			t := reflect.TypeOf(data)
			newStruct := reflect.New(v.Type()).Elem()
			newStruct.Set(v) // 复制原始值
			for i := 0; i < newStruct.NumField(); i++ {
				if t.Field(i).Name == callbackField {
					newStruct.Field(i).SetBool(!newStruct.Field(i).Bool())
				}
			}
			thisChat.MessageType = newStruct.Interface().(updatetype.MessageType)

			opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
				ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
				ReplyMarkup: buildMessageTypeKB(thisChat.MessageType),
			})
		} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "limitmsg_attr_") {
			callbackField := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "limitmsg_attr_")
			data := thisChat.MessageAttribute
			v := reflect.ValueOf(data) // 解除指针获取值
			t := reflect.TypeOf(data)
			newStruct := reflect.New(v.Type()).Elem()
			newStruct.Set(v) // 复制原始值
			for i := 0; i < newStruct.NumField(); i++ {
				if t.Field(i).Name == callbackField {
					newStruct.Field(i).SetBool(!newStruct.Field(i).Bool())
				}
			}

			thisChat.MessageAttribute = newStruct.Interface().(updatetype.MessageAttribute)

			opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
				ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.Update.CallbackQuery.Message.Message.ID,
				ReplyMarkup: buildMessageAttributeKB(thisChat.MessageAttribute),
			})
		}
	}

	LimitMessageList[opts.Update.CallbackQuery.Message.Message.Chat.ID] = thisChat
	SaveLimitMessageList()
}
