package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/contain"
	"trbot/utils/type/message_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var LimitMessageList map[int64]AllowMessages
var LimitMessageErr  error

var LimitMessageDir string = filepath.Join(consts.YAMLDataBaseDir, "limitmessage/")
var LimitMessagePath string = filepath.Join(LimitMessageDir, consts.YAMLFileName)

type AllowMessages struct {
	IsEnable            bool                           `yaml:"IsEnable"`
	IsUnderTest         bool                           `yaml:"IsUnderTest"`
	AddTime             string                         `yaml:"AddTime"`
	IsLogicAnd          bool                           `yaml:"IsLogicAnd"` // true: `&&``, false: `||`
	IsWhiteForType      bool                           `yaml:"IsWhiteForType"`
	MessageType         message_utils.Message      `yaml:"MessageType"`
	IsWhiteForAttribute bool                           `yaml:"IsWhiteForAttribute"`
	MessageAttribute    message_utils.Attribute `yaml:"MessageAttribute"`
}

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "Limit Message",
		Func: ReadLimitMessageList,
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name: "Limit Message",
		Saver: SaveLimitMessageList,
		Loader: ReadLimitMessageList,
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "limitmessage",
		MessageHandler: SomeMessageOnlyHandler,
	})
	plugin_utils.AddCallbackQueryHandlers(plugin_utils.CallbackQuery{
		CallbackDataPrefix:   "limitmsg_",
		CallbackQueryHandler: LimitMessageCallback,
	})
	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "限制群组消息",
		Description: "此功能需要 bot 为群组管理员并拥有删除消息的权限\n可以按照消息类型和消息属性来自动删除不允许的消息，支持自定逻辑和黑白名单，作为管理员在群组中使用 /limitmessage 命令来查看菜单",
		ParseMode:   models.ParseModeHTML,
	})
}

func ReadLimitMessageList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "LimitMessage").
		Str("funcName", "ReadLimitMessageList").
		Logger()

	err := yaml.LoadYAML(LimitMessagePath, &LimitMessageList)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", LimitMessagePath).
				Msg("Not found limit message list file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(LimitMessagePath, &LimitMessageList)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", LimitMessagePath).
					Msg("Failed to create empty limit message list file")
				LimitMessageErr = fmt.Errorf("failed to create empty limit message list file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", LimitMessagePath).
				Msg("Failed to load limit message list file")
			LimitMessageErr = fmt.Errorf("failed to load limit message list file: %w", err)
		}
	} else {
		LimitMessageErr = nil
	}

	if LimitMessageList == nil {
		LimitMessageList = map[int64]AllowMessages{}
	}

	buildLimitGroupList()

	return LimitMessageErr
}

func SaveLimitMessageList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "LimitMessage").
		Str("funcName", "SaveLimitMessageList").
		Logger()
	err := yaml.SaveYAML(LimitMessagePath, &LimitMessageList)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", LimitMessagePath).
			Msg("Failed to save limit message list")
		LimitMessageErr = fmt.Errorf("failed to save limit message list: %w", err)
	} else {
		LimitMessageErr = nil
	}
	return LimitMessageErr
}

func SomeMessageOnlyHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "LimitMessage").
		Str("funcName", "SomeMessageOnlyHandler").
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	var handlerErr flaterr.MultErr

	if opts.Message.Chat.Type == models.ChatTypePrivate {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Message.Chat.ID,
			Text:            "此功能被设计为仅在群组中可用",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "limit message only allows in group").
				Msg(flaterr.SendMessage.Str())
			handlerErr.Addt(flaterr.SendMessage, "limit message only allows in group", err)
		}
	} else {
		if utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.Message.Chat.ID, opts.Message.From.ID) {
			thisChat := LimitMessageList[opts.Message.Chat.ID]

			var isNeedInit bool = false

			if thisChat.AddTime == "" {
				isNeedInit = true
				thisChat.AddTime = time.Now().Format(time.RFC3339)
			}

			if utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.Message.Chat.ID, consts.BotMe.ID) && utils.UserHavePermissionDeleteMessage(opts.Ctx, opts.Thebot, opts.Message.Chat.ID, consts.BotMe.ID) {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Message.Chat.ID,
					Text:   "Limit Message 菜单",
					ReplyMarkup: buildMessageAllKB(thisChat),
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "limit message main menu").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "limit message main menu", err)
				}
				_, err = opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
					ChatID: opts.Message.Chat.ID,
					MessageID: opts.Message.ID,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "limit message command").
						Msg(flaterr.DeleteMessage.Str())
					handlerErr.Addt(flaterr.DeleteMessage, "limit message command", err)
				}
				if isNeedInit {
				LimitMessageList[opts.Message.Chat.ID] = thisChat
					err = SaveLimitMessageList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Msg("Failed to save limit message list after adding new chat")
						handlerErr.Addf("failed to save limit message list after adding new chat: %w", err)
					}
				}
			} else {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Message.Chat.ID,
					Text:   "启用此功能前，请先将机器人设为管理员\n如果还是提示本消息，请检查机器人是否有删除消息的权限",
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "bot need be admin and have delete message permission").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "bot need be admin and have delete message permission", err)
				}
			}
		} else {
			botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Message.Chat.ID,
				Text:   "抱歉，您不是群组的管理员，无法为群组更改此功能",
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "non-admin can not change limit message config").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "non-admin can not change limit message config", err)
			}
			time.Sleep(time.Second * 5)
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
					Str("content", "non-admin can not change limit message config").
					Msg(flaterr.DeleteMessages.Str())
				handlerErr.Addt(flaterr.DeleteMessages, "non-admin can not change limit message config", err)
			}
		}
	}

	return handlerErr.Flat()
}

func DeleteNotAllowMessage(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "LimitMessage").
		Str("funcName", "SomeMessageOnlyHandler").
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Dict(utils.GetUserDict(opts.Message.From)).
		Int("messageID", opts.Message.ID).
		Logger()

	var handlerErr flaterr.MultErr

	var deleteAction bool
	var deleteHelp   string = "当前模式："
	if contain.AnyType(opts.Message.Chat.Type, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 处理消息删除逻辑，只有当群组启用该功能时才处理
		thisChat := LimitMessageList[opts.Message.Chat.ID]
		if thisChat.IsEnable || thisChat.IsUnderTest {
			thisMsgType := message_utils.GetMessageType(opts.Message)
			thisMsgAttr := message_utils.GetMessageAttribute(opts.Message)

			// 根据规则的黑白名单选择判断逻辑
			if thisChat.IsLogicAnd {
				deleteHelp += "同时触发两个规则才删除消息\n"
				msgType, typeHelp := CheckMessageType(thisMsgType, thisChat.MessageType, thisChat.IsWhiteForType)
				deleteHelp += "消息类型：" + typeHelp
				if msgType {
					msgAttr, attrHelp := CheckMessageAttribute(thisMsgAttr, thisChat.MessageAttribute, thisChat.IsWhiteForAttribute)
					deleteHelp += "消息属性：" + attrHelp
					if msgType && msgAttr {
						deleteAction = true
					}
				}
			} else {
				deleteHelp += "触发任一规则就删除消息\n"
				msgType, typeHelp := CheckMessageType(thisMsgType, thisChat.MessageType, thisChat.IsWhiteForType)
				deleteHelp += "消息类型：" + typeHelp
				if msgType {
					deleteAction = true
				} else {
					msgAttr, attrHelp := CheckMessageAttribute(thisMsgAttr, thisChat.MessageAttribute, thisChat.IsWhiteForAttribute)
					deleteHelp += "消息属性：" + attrHelp
					if msgAttr {
						deleteAction = true
					}
				}
			}

			if thisChat.IsUnderTest {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Message.Chat.ID,
					Text:   utils.TextForTrueOrFalse(deleteAction, "此消息会被设定的规则删除\n\n", "此消息不会被删除\n\n") +
							deleteHelp +
							utils.TextForTrueOrFalse(thisChat.IsEnable, "<blockquote>当前已启用，关闭测试模式将开始删除触发了规则的消息</blockquote>", "<blockquote>您可以继续进行测试，以便达到您想要的效果，之后请手动启用此功能\n</blockquote>"),
					DisableNotification: true,
					ParseMode: models.ParseModeHTML,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
					ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{
						{
							Text: "打开配置菜单",
							CallbackData: "limitmsg_back",
						},
						{
							Text: "关闭测试模式",
							CallbackData: "limitmsg_offtest",
						},
					}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "test mode delete message notification").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "test mode delete message notification", err)
				}
			} else if deleteAction {
				_, err := opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
					ChatID:    opts.Message.Chat.ID,
					MessageID: opts.Message.ID,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("messageType", thisMsgType.Str()).
						Str("content", "message trigger limit message rules").
						Bool("IsLogicAnd", thisChat.IsLogicAnd).
						Msg(flaterr.DeleteMessage.Str())
					handlerErr.Addt(flaterr.DeleteMessage, "message trigger limit message rules", err)
				} else {
					logger.Info().
						Str("messageType", thisMsgType.Str()).
						Bool("IsLogicAnd", thisChat.IsLogicAnd).
						Msg("Deleted message trigger limit message rules")
				}
			}
		}
	}
	return handlerErr.Flat()
}

func CheckMessageType(this, target message_utils.Message, IsWhiteList bool) (bool, string) {
	var delete bool = IsWhiteList
	var deleteHelp string

	v1 := reflect.ValueOf(this)
	v2 := reflect.ValueOf(target)
	t  := reflect.TypeOf(this)

	for i := 0; i < v1.NumField(); i++ {
		field := t.Field(i)
		val1 := v1.Field(i).Interface()
		val2 := v2.Field(i).Interface()

		if val1 == true && val1 == val2 {
			deleteHelp += fmt.Sprintf("%s 消息类型 %s %s\n",
				utils.TextForTrueOrFalse(IsWhiteList, "白名单", "黑名单"),
				field.Name,
				utils.TextForTrueOrFalse(IsWhiteList, "不删除", "删除"),
			)
			if IsWhiteList {
				delete = false
			} else {
				delete = true
			}
		} else if val1 == true && val1 != val2 {
			deleteHelp += fmt.Sprintf("%s 未命中 消息类型 %s 遵循默认规则 %s\n",
				utils.TextForTrueOrFalse(IsWhiteList, "白名单", "黑名单"),
				field.Name,
				utils.TextForTrueOrFalse(delete, "删除", "不删除"),
			)
		}
	}
	return delete, deleteHelp
}

func CheckMessageAttribute(this, target message_utils.Attribute, IsWhiteList bool) (bool, string) {
	var delete bool = IsWhiteList
	var noAttribute bool = true // 如果没有命中任何消息属性，提示内容，根据黑白名单判断是否删除
	var deleteHelp string

	v1 := reflect.ValueOf(this)
	v2 := reflect.ValueOf(target)
	t := reflect.TypeOf(this)

	for i := 0; i < v1.NumField(); i++ {
		field := t.Field(i)
		val1 := v1.Field(i).Interface()
		val2 := v2.Field(i).Interface()


		if val1 == true && val1 == val2 {
			noAttribute = false
			deleteHelp += fmt.Sprintf("%s 消息属性 %s %s\n",
				utils.TextForTrueOrFalse(IsWhiteList, "白名单", "黑名单"),
				field.Name,
				utils.TextForTrueOrFalse(IsWhiteList, "不删除", "删除"),
			)
			if IsWhiteList {
				delete = false
			} else {
				delete = true
			}
		} else if val1 == true && val1 != val2 {
			noAttribute = false
			deleteHelp += fmt.Sprintf("%s 未命中 消息属性 %s 遵循默认规则 %s\n",
				utils.TextForTrueOrFalse(IsWhiteList, "白名单", "黑名单"),
				field.Name,
				utils.TextForTrueOrFalse(delete, "删除", "不删除"),
			)
		}
	}
	if noAttribute {
		deleteHelp += fmt.Sprintf("%s 未命中 消息属性 无 遵循默认规则 %s\n",
			utils.TextForTrueOrFalse(IsWhiteList, "白名单", "黑名单"),
			utils.TextForTrueOrFalse(delete, "删除", "不删除"),
		)
	}

	return delete, deleteHelp
}

func buttonText(text string, opt, IsWhiteList bool) string {
	if opt {
		return utils.TextForTrueOrFalse(IsWhiteList, "✅ ", "❌ ") + text
	}

	return text
}

func buildMessageTypeKB(chat AllowMessages) models.ReplyMarkup {

	var msgTypeItems [][]models.InlineKeyboardButton
	var msgTypeItemsTemp []models.InlineKeyboardButton

	v := reflect.ValueOf(chat.MessageType) // 解除指针获取值
	t := reflect.TypeOf(chat.MessageType)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if i % 3 == 0 && i != 0 {
			msgTypeItems = append(msgTypeItems, msgTypeItemsTemp)
			msgTypeItemsTemp = []models.InlineKeyboardButton{}
		}
		msgTypeItemsTemp = append(msgTypeItemsTemp, models.InlineKeyboardButton{
			Text:         buttonText(field.Name, value.Bool(), chat.IsWhiteForType),
			CallbackData: "limitmsg_type_" + field.Name,
		})
	}
	if len(msgTypeItemsTemp) != 0 {
		msgTypeItems = append(msgTypeItems, msgTypeItemsTemp)
	}


	msgTypeItems = append(msgTypeItems, []models.InlineKeyboardButton{{
		Text: "⬅️ 返回上一级",
		CallbackData: "limitmsg_back",
	}})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: msgTypeItems,
	}

	return kb
}

func buildMessageAttributeKB(chat AllowMessages) models.ReplyMarkup {

	var msgAttributeItems [][]models.InlineKeyboardButton
	var msgAttributeItemsTemp []models.InlineKeyboardButton

	v := reflect.ValueOf(chat.MessageAttribute) // 解除指针获取值
	t := reflect.TypeOf(chat.MessageAttribute)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)
		if i % 2 == 0 && i != 0 {
			msgAttributeItems = append(msgAttributeItems, msgAttributeItemsTemp)
			msgAttributeItemsTemp = []models.InlineKeyboardButton{}
		}
		msgAttributeItemsTemp = append(msgAttributeItemsTemp, models.InlineKeyboardButton{
			Text:         buttonText(field.Name, value.Bool(), chat.IsWhiteForAttribute),
			CallbackData: "limitmsg_attr_" + field.Name,
		})
	}
	if len(msgAttributeItemsTemp) != 0 {
		msgAttributeItems = append(msgAttributeItems, msgAttributeItemsTemp)
	}


	msgAttributeItems = append(msgAttributeItems, []models.InlineKeyboardButton{{
		Text:         "⬅️ 返回上一级",
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
			Text:         "选择消息类型",
			CallbackData: "limitmsg_typekb",
		},
		{
			Text:         "🔄 " + utils.TextForTrueOrFalse(chat.IsWhiteForType, "白名单模式", "黑名单模式"),
			CallbackData: "limitmsg_typekb_switchrule",
		},
	})

	chatAllow = append(chatAllow, []models.InlineKeyboardButton{
		{
			Text:         "选择消息属性",
			CallbackData: "limitmsg_attrkb",
		},
		{
			Text:         "🔄 " + utils.TextForTrueOrFalse(chat.IsWhiteForAttribute, "白名单模式", "黑名单模式"),
			CallbackData: "limitmsg_attrkb_switchrule",
		},
	})

	chatAllow = append(chatAllow, []models.InlineKeyboardButton{
		{
			Text:         "🔄 " + utils.TextForTrueOrFalse(chat.IsLogicAnd, "满足上方所有条件才删除消息", "满足其中一个条件就删除消息"),
			CallbackData: "limitmsg_switchlogic",
		},
	})

	chatAllow = append(chatAllow, []models.InlineKeyboardButton{
		{
			Text:         "🔄 " + utils.TextForTrueOrFalse(chat.IsUnderTest, "测试模式已开启 ✅", "测试模式已关闭 ❌"),
			CallbackData: "limitmsg_switchtest",
		},
	})

	chatAllow = append(chatAllow, []models.InlineKeyboardButton{
		{
			Text:         "🚫 关闭菜单",
			CallbackData: "delete_this_message",
		},
		{
			Text:         "🔄 " + utils.TextForTrueOrFalse(chat.IsEnable, "当前已启用 ✅", "当前已关闭 ❌"),
			CallbackData: "limitmsg_switchenable",
		},
	})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: chatAllow,
	}

	return kb
}

func LimitMessageCallback(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "LimitMessage").
		Str("funcName", "LimitMessageCallback").
		Dict(utils.GetChatDict(&opts.CallbackQuery.Message.Message.Chat)).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Str("callbackQueryData", opts.CallbackQuery.Data).
		Logger()

	var handlerErr flaterr.MultErr

	if !utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.CallbackQuery.Message.Message.Chat.ID, opts.CallbackQuery.From.ID) {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text: "您没有权限修改此配置",
			ShowAlert: true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "no permission to change limit message config").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "no permission to change limit message config", err)
		}
	} else {
		thisChat := LimitMessageList[opts.CallbackQuery.Message.Message.Chat.ID]

		var needRebuildGroupList     bool
		var needSavelimitMessageList bool
		var needEditMainMenuMessage  bool

		switch opts.CallbackQuery.Data {
		case "limitmsg_typekb":
			// opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			// 	CallbackQueryID: opts.CallbackQuery.ID,
			// 	Text: "已选择消息类型",
			// })
			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text: utils.TextForTrueOrFalse(thisChat.IsWhiteForType, "白名单模式", "黑名单模式") + ": " + utils.TextForTrueOrFalse(thisChat.IsWhiteForType, "仅允许发送选中的项目，其他消息将被删除", "将删除选中的项目"),
				ReplyMarkup: buildMessageTypeKB(thisChat),
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "limit message type keyboard").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "limit message type keyboard", err)
			}
		case "limitmsg_typekb_switchrule":
			thisChat.IsWhiteForType = !thisChat.IsWhiteForType
			needSavelimitMessageList = true
			needEditMainMenuMessage = true
		case "limitmsg_attrkb":
			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text: utils.TextForTrueOrFalse(thisChat.IsWhiteForAttribute, "白名单模式", "黑名单模式") + ": " + utils.TextForTrueOrFalse(thisChat.IsWhiteForAttribute, "仅允许发送选中的项目，其他消息将被删除", "将删除选中的项目") + "\n有一些项目可能无法使用",
				ReplyMarkup: buildMessageAttributeKB(thisChat),
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "limit message attribute keyboard").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "limit message attribute keyboard", err)
			}
		case "limitmsg_attrkb_switchrule":
			thisChat.IsWhiteForAttribute = !thisChat.IsWhiteForAttribute
			needSavelimitMessageList = true
			needEditMainMenuMessage = true
		case "limitmsg_back":
			needEditMainMenuMessage = true
		case "limitmsg_switchenable":
			thisChat.IsEnable = !thisChat.IsEnable
			if thisChat.IsEnable { thisChat.IsUnderTest = false }
			needRebuildGroupList = true
			needSavelimitMessageList = true
			needEditMainMenuMessage = true
		case "limitmsg_switchlogic":
			thisChat.IsLogicAnd = !thisChat.IsLogicAnd
			needSavelimitMessageList = true
			needEditMainMenuMessage = true
		case "limitmsg_switchtest":
			thisChat.IsUnderTest = !thisChat.IsUnderTest
			needEditMainMenuMessage = true
			needRebuildGroupList = true
			needSavelimitMessageList = true
		case "limitmsg_offtest":
			thisChat.IsUnderTest = false
			needSavelimitMessageList = true
			needRebuildGroupList = true
			_, err := opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
				ChatID:      opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID:   opts.CallbackQuery.Message.Message.ID,
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text:         "删除此提醒",
					CallbackData: "delete_this_message",
				}}}},
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "test mode turned off notice").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "test mode turned off notice", err)
			}
		default:
			if strings.HasPrefix(opts.CallbackQuery.Data, "limitmsg_type_") {
				needSavelimitMessageList = true
				callbackField := strings.TrimPrefix(opts.CallbackQuery.Data, "limitmsg_type_")

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
				thisChat.MessageType = newStruct.Interface().(message_utils.Message)

				_, err := opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
					ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					ReplyMarkup: buildMessageTypeKB(thisChat),
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "limit message type keyboard").
						Msg(flaterr.EditMessageReplyMarkup.Str())
					handlerErr.Addt(flaterr.EditMessageReplyMarkup, "limit message type keyboard", err)
				}
			} else if strings.HasPrefix(opts.CallbackQuery.Data, "limitmsg_attr_") {
				needSavelimitMessageList = true
				callbackField := strings.TrimPrefix(opts.CallbackQuery.Data, "limitmsg_attr_")
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

				thisChat.MessageAttribute = newStruct.Interface().(message_utils.Attribute)

				_, err := opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
					ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					ReplyMarkup: buildMessageAttributeKB(thisChat),
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "limit message attribute keyboard").
						Msg(flaterr.EditMessageReplyMarkup.Str())
					handlerErr.Addt(flaterr.EditMessageReplyMarkup, "limit message attribute keyboard", err)
				}
			}
		}

		if needSavelimitMessageList {
			LimitMessageList[opts.CallbackQuery.Message.Message.Chat.ID] = thisChat
			err := SaveLimitMessageList(opts.Ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to save limit message list")
				handlerErr.Addf("failed to save limit message list: %w", err)
				_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "保存修改失败，请重试或联系机器人管理员\n" + err.Error(),
					ShowAlert:       true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "failed to save limit message list notice").
						Msg(flaterr.AnswerCallbackQuery.Str())
					handlerErr.Addt(flaterr.AnswerCallbackQuery, "failed to save limit message list notice", err)
				}
			}
		}

		if needRebuildGroupList {
			buildLimitGroupList()
		}

		if needEditMainMenuMessage {
			_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
				ChatID: opts.CallbackQuery.Message.Message.Chat.ID,
				MessageID: opts.CallbackQuery.Message.Message.ID,
				Text:   "Limit Message 菜单",
				ReplyMarkup: buildMessageAllKB(thisChat),
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "limit message main menu").
					Msg(flaterr.EditMessageText.Str())
				handlerErr.Addt(flaterr.EditMessageText, "limit message main menu", err)
			}
		}
	}

	return handlerErr.Flat()
}

func buildLimitGroupList() {
	for chatID, n := range LimitMessageList {
		if n.IsEnable || n.IsUnderTest {
			plugin_utils.AddHandlerByMessageChatIDHandlers(plugin_utils.ByMessageChatIDHandler{
				ForChatID:      chatID,
				PluginName:     "limit_message",
				MessageHandler: DeleteNotAllowMessage,
			})
		} else {
			plugin_utils.RemoveHandlerByChatIDHandler(chatID, "limit_message")
		}
	}
}
