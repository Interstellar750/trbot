package plugins

import (
	"fmt"
	"log"
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
)

func SomeMessageOnlyHandler(opts *handler_utils.SubHandlerOpts) {
	if opts.Update.Message.Chat.Type == "private" {
		botMessage, _ := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:          opts.Update.Message.Chat.ID,
			Text:            "此功能被设计为仅在群组中可用",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
		})
		time.Sleep(time.Second * 10)
		opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
			ChatID: opts.Update.Message.Chat.ID,
			MessageIDs: []int{
				opts.Update.Message.ID,
				botMessage.ID,
			},
		})
	} else if utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.Update.Message.Chat.ID, opts.Update.Message.From.ID) {
		if !opts.ChatInfo.IsEnableForwardonly && strings.HasSuffix(opts.Update.Message.Text, fmt.Sprint(opts.Update.Message.Chat.ID)) {
			if opts.ChatInfo.ID != opts.Update.Message.Chat.ID {
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					Text:      "发送的群组 ID 与当前群组的 ID 不符，请先发送 `/forwardonly`",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.ChatInfo.IsEnableForwardonly = true
				log.Println("Turn forwardonly on for group", opts.Update.Message.Chat.ID)
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					Text:      "仅限转发模式已启用",
					ParseMode: models.ParseModeMarkdownV1,
				})
				consts.SignalsChannel.Database_save <- true
			}
		} else if opts.Update.Message.Text == "/forwardonly disable" {
			if !opts.ChatInfo.IsEnableForwardonly {
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					Text:      "此群组并没有开启仅限转发模式哦",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			} else {
				opts.ChatInfo.IsEnableForwardonly = false
				log.Println("Turn forwardonly off for group", opts.Update.Message.Chat.ID)
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					Text:      fmt.Sprintf("仅限转发模式已关闭，重新启用请发送 `/forwardonly %d`", opts.Update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
				consts.SignalsChannel.Database_save <- true
			}
		} else if strings.HasPrefix(opts.Update.Message.Text, "/forwardonly") {
			if utils.UserIsAdmin(opts.Ctx, opts.Thebot, opts.Update.Message.Chat.ID, consts.BotMe.ID) && utils.UserHavePermissionDeleteMessage(opts.Ctx, opts.Thebot, opts.Update.Message.Chat.ID, consts.BotMe.ID) {
				if opts.ChatInfo.IsEnableForwardonly {
					botMessage, _ := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:    opts.Update.Message.Chat.ID,
						Text:      "仅限转发模式已启用，无须重复开启，若要关闭，请发送 `/forwardonly disable` 来关闭它",
						ParseMode: models.ParseModeMarkdownV1,
					})
					time.Sleep(time.Second * 5)
					opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
						ChatID: opts.Update.Message.Chat.ID,
						MessageIDs: []int{
							opts.Update.Message.ID,
							botMessage.ID,
						},
					})
					return
				}
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					Text:      fmt.Sprintf("请求已确定，继续发送 `/forwardonly %d` 以启用仅限转发模式", opts.Update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
			} else {
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					Text:   "启用此功能前，请先将机器人设为管理员\n如果还是提示本消息，请检查机器人是否有删除消息的权限",
				})
			}
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
	var deleteMessageWhiteList   bool = true
	var deleteAttributeWhiteList bool = true
	
	var deleteAction bool
	if utils.AnyContains(opts.Update.Message.Chat.Type, models.ChatTypeGroup, models.ChatTypeSupergroup) {
		// 处理消息删除逻辑，只有当群组启用该功能时才处理
		if opts.ChatInfo.IsEnableForwardonly {
			this := updatetype.GetMessageType(opts.Update.Message)
			thisattribute  := updatetype.GetMessageAttribute(opts.Update.Message)

			// 根据规则的黑白名单选择判断逻辑
			if deleteMessageWhiteList == deleteAttributeWhiteList {
				deleteAction = CheckMessageType(this, AllowedMessageTypeList, deleteMessageWhiteList) || CheckMessageAttribute(thisattribute, AllowedMessageAttributeList, deleteAttributeWhiteList)
			} else {
				deleteAction = CheckMessageType(this, AllowedMessageTypeList, deleteMessageWhiteList) && CheckMessageAttribute(thisattribute, AllowedMessageAttributeList, deleteAttributeWhiteList)
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

var ForwardOnly_SlashSymbolCommandHandler = plugin_utils.Plugin_SlashSymbolCommand{
	SlashCommand: "forwardonly",
	Handler: SomeMessageOnlyHandler,
}

var AllowedMessageTypeList = updatetype.MessageType{
	// default blacklist
	OnlyText: true,
	Sticker:  true,
	Voice:    true,
}

var AllowedMessageAttributeList = updatetype.MessageAttribute{
	IsForwardMessage: true,
}
