package utils

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"trbot/utils/consts"
	"trbot/utils/type/contain"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
)

func GetChatAdminIDs(ctx context.Context, thebot *bot.Bot, chatID any) (ids []int64) {
	admins, err := thebot.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{ ChatID: chatID })
	if err != nil {
		log.Printf("Failed to get chat administrators: %w", err)
		return nil
	}
	for _, n := range admins {
		if n.Administrator != nil {
			ids = append(ids, n.Administrator.User.ID)
		} else if n.Owner != nil {
			ids = append(ids, n.Owner.User.ID)
		}
	}
	return
}

// 检查用户是否是管理员
// chat type: "private", "group", "supergroup", or "channel"
// not work for "private" chats
func UserIsAdmin(ctx context.Context, thebot *bot.Bot, chatID, userID any) bool {
	admins, err := thebot.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{ ChatID: chatID })
	if err != nil {
		log.Printf("Failed to get chat administrators: %w", err)
		return false
	}

	var admins_usernames []string
	var admins_userIDs []int64

	for _, admin := range admins {
		if admin.Owner != nil {
			admins_userIDs = append(admins_userIDs, admin.Owner.User.ID)
			if admin.Owner.User.Username != "" {
				admins_usernames = append(admins_usernames, admin.Owner.User.Username)
			}
		}
		if admin.Administrator != nil {
			admins_userIDs = append(admins_userIDs, admin.Administrator.User.ID)
			if admin.Administrator.User.Username != "" {
				admins_usernames = append(admins_usernames, admin.Administrator.User.Username)
			}
		}
	}

	switch value := userID.(type) {
	case int64:
		// fmt.Println(value)
		return contain.Int64(value, admins_userIDs...)
	case string:
		// fmt.Println(value)
		if strings.ContainsAny(value, "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ_") {
			return contain.SubStringCaseInsensitive(value, admins_usernames...)
		} else {
			int_userID, _ := strconv.Atoi(value)
			return contain.Int64(int64(int_userID), admins_userIDs...)
		}
	default:
		log.Println("userID type not supported")
		return false
	}
}
// 检查用户是否有权限删除消息
func UserHavePermissionDeleteMessage(ctx context.Context, thebot *bot.Bot, chatID, userID any) bool {
	admins, err := thebot.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{
		ChatID: chatID,
	})
	if err != nil {
		log.Printf("Failed to get chat administrators: %w", err)
		return false
	}

	var adminshavepermission_usernames []string
	var adminshavepermission_userIDs []int64

	for _, admin := range admins {
		// owner allways have all permission
		if admin.Administrator != nil && admin.Administrator.CanDeleteMessages {
			adminshavepermission_userIDs = append(adminshavepermission_userIDs, admin.Administrator.User.ID)
			if admin.Administrator.User.Username != "" {
				adminshavepermission_usernames = append(adminshavepermission_usernames, admin.Administrator.User.Username)
			}
		}
	}
	switch value := userID.(type) {
	case int64:
		// fmt.Println(value)
		return contain.Int64(value, adminshavepermission_userIDs...)
	case string:
		// fmt.Println(value)
		if strings.ContainsAny(value, "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ_") {
			return contain.SubStringCaseInsensitive(value, adminshavepermission_usernames...)
		} else {
			int_userID, _ := strconv.Atoi(value)
			return contain.Int64(int64(int_userID), adminshavepermission_userIDs...)
		}
	default:
		log.Println("userID type not supported")
		return false
	}
}

// 允许响应带有机器人用户名后缀的命令，例如 /help@examplebot
func CommandMaybeWithSuffixUsername(commandFields []string, command string) bool {
	if len(commandFields) == 0 { return false }
	atBotUsername := "@" + consts.BotMe.Username
	if commandFields[0] == command || commandFields[0] == command + atBotUsername {
		return true
	}
	return false
}

// return user fullname
func ShowUserName(user *models.User) string {
	if user == nil { return "" }
	if user.LastName != "" {
		return user.FirstName + " " + user.LastName
	} else {
		return user.FirstName
	}
}

// return chat fullname
func ShowChatName(chat *models.Chat) string {
	if chat == nil { return "" }
	if chat.Title != "" { // 群组
		return chat.Title
	} else if chat.LastName != "" { // 可能是用户正在与 bot 发送信息
		return chat.FirstName + " " + chat.LastName
	} else {
		return chat.FirstName
	}
}

// 如果一个 int64 类型的 ID 为 `-100`` 开头的负数，则去掉 `-100``
func RemoveIDPrefix(id int64) string {
	mayWithPrefix := fmt.Sprintf("%d", id)
	if strings.HasPrefix(mayWithPrefix, "-100") {
		return mayWithPrefix[4:]
	} else {
		return mayWithPrefix
	}
}

func TextForTrueOrFalse(condition bool, tureText, falseText string) string {
	if condition {
		return tureText
	} else {
		return falseText
	}
}

// 获取消息来源的链接
func GetMessageFromHyperLink(msg *models.Message, ParseMode models.ParseMode) string {
	var senderLink string
	attr := message_utils.GetMessageAttribute(msg)

	switch ParseMode {
	case models.ParseModeHTML:
		if attr.IsUserAsChannel {
			senderLink = fmt.Sprintf("<a href=\"https://t.me/%s\">%s</a>", msg.SenderChat.Username, ShowChatName(msg.SenderChat))
		} else if attr.IsFromLinkedChannel || attr.IsFromAnonymous || attr.IsHasSenderChat {
			senderLink = fmt.Sprintf("<a href=\"https://t.me/c/%s\">%s</a>", RemoveIDPrefix(msg.SenderChat.ID), ShowChatName(msg.SenderChat))
		} else  if msg.From != nil {
			senderLink = fmt.Sprintf("<a href=\"https://t.me/@id%d\">%s</a>", msg.From.ID, ShowUserName(msg.From))
		}
	default:
		if attr.IsUserAsChannel {
			senderLink = fmt.Sprintf("[%s][https://t.me/%s]", ShowChatName(msg.SenderChat), msg.SenderChat.Username)
		} else if attr.IsFromLinkedChannel || attr.IsFromAnonymous || attr.IsHasSenderChat {
			senderLink = fmt.Sprintf("[%s][https://t.me/c/%s]", ShowChatName(msg.SenderChat), RemoveIDPrefix(msg.SenderChat.ID))
		} else if msg.From != nil {
			senderLink = fmt.Sprintf("[%s][https://t.me/@id%d]", ShowUserName(msg.From), msg.From.ID)
		}
	}
	return senderLink
}

func PanicCatcher(ctx context.Context, funcName string) {
	panic := recover()
	if panic != nil {
		zerolog.Ctx(ctx).Error().
			Stack().
			Str("commit", consts.Commit).
			Err(errors.WithStack(fmt.Errorf("%v", panic))).
			Str("catchFunc", funcName).
			Msg("Panic recovered")
	}
}

// return a "user" string and a `zerolog.Dict()` with `name`(string), `username`(string), `ID`(int64) *zerolog.Event
func GetUserDict(user *models.User) (string, *zerolog.Event) {
	if user == nil { return "user", zerolog.Dict() }
	return "user", zerolog.Dict().
		Str("name", ShowUserName(user)).
		Str("username", user.Username).
		Int64("ID", user.ID)
}

// return a "chat" string and a `zerolog.Dict()` with `name`(string), `username`(string), `ID`(int64), `type`(string) *zerolog.Event
func GetChatDict(chat *models.Chat) (string, *zerolog.Event) {
	if chat == nil { return "chat", zerolog.Dict() }
	return "chat", zerolog.Dict().
		Str("name", ShowChatName(chat)).
		Str("username", chat.Username).
		Str("type", string(chat.Type)).
		Int64("ID", chat.ID)
}

// Can replace GetUserDict(), not for GetChatDict(), and not available for some update type.
// return a sender type string and a `zerolog.Dict()` to show sender info
func GetUserOrSenderChatDict(msg *models.Message) (string, *zerolog.Event) {
	if msg == nil { return "noMessage", zerolog.Dict().Str("error", "no message to check") }

	attr := message_utils.GetMessageAttribute(msg)

	switch {
	case attr.IsFromAnonymous:
		return "groupAnonymous", zerolog.Dict().
			Str("chat", ShowChatName(msg.SenderChat)).
			Str("username", msg.SenderChat.Username).
			Int64("ID", msg.SenderChat.ID)
	case attr.IsUserAsChannel:
		return "userAsChannel", zerolog.Dict().
			Str("chat", ShowChatName(msg.SenderChat)).
			Str("username", msg.SenderChat.Username).
			Int64("ID", msg.SenderChat.ID)
	case attr.IsFromLinkedChannel:
		return "linkedChannel", zerolog.Dict().
			Str("chat", ShowChatName(msg.SenderChat)).
			Str("username", msg.SenderChat.Username).
			Int64("ID", msg.SenderChat.ID)
	case attr.IsFromBusinessBot:
		return "businessBot", zerolog.Dict().
			Str("name", ShowUserName(msg.SenderBusinessBot)).
			Str("username", msg.SenderBusinessBot.Username).
			Int64("ID", msg.SenderBusinessBot.ID)
	case attr.IsHasSenderChat && msg.SenderChat.ID != msg.Chat.ID:
		// use other channel send message in this channel
		return "senderChat", zerolog.Dict().
			Str("chat", ShowChatName(msg.SenderChat)).
			Str("username", msg.SenderChat.Username).
			Int64("ID", msg.SenderChat.ID)
	default:
		if msg.From != nil {
			return GetUserDict(msg.From)
		}
	}

	return "noUserOrSender", zerolog.Dict()
}

// 从 log.txt 读取文件
func ReadLog() ([]string, error) {
	// 打开日志文件
	file, err := os.Open(consts.LogFilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	// 读取文件内容
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return lines, nil
}

// 输出版本信息
func OutputVersionInfo() string {
	hostname, _ := os.Hostname()
	var gitURL string = "https://gitea.trle5.xyz/trle5/trbot/commit/"
	var info   string
	if consts.BuildAt != "" {
		info += fmt.Sprintf("`Version:   `%s\n", consts.Version)
		info += fmt.Sprintf("`Branch:    `%s\n", consts.Branch)
		info += fmt.Sprintf("`Commit:    `[%s](%s%s) (%s)\n", consts.Commit[:10], gitURL, consts.Commit, consts.Changes)
		info += fmt.Sprintf("`BuildAt:   `[%s](%s%s)\n", consts.BuildAt, "https://timestamp.online/timestamp/", consts.BuildAt)
		info += fmt.Sprintf("`BuildOn:   `%s\n", consts.BuildOn)
		info += fmt.Sprintf("`Runtime:   `%s\n", runtime.Version())
		info += fmt.Sprintf("`Goroutine: `%d\n", runtime.NumGoroutine())
		info += fmt.Sprintf("`Hostname:  `%s\n", hostname)
		return info
	}
	return fmt.Sprintln(
		"Warning: No build info\n",
		"\n`Runtime:   `", runtime.Version(),
		"\n`Goroutine: `", runtime.NumGoroutine(),
		"\n`Hostname:  `", hostname,
	)
}
