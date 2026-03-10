package utils

import (
	"context"
	"fmt"
	"html"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"

	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/type/contain"
	"trle5.xyz/trbot/utils/type/message_utils"

	"github.com/dustin/go-humanize"
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
	atBotUsername := "@" + configs.BotMe.Username
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
		} else if msg.From != nil {
			senderLink = fmt.Sprintf("<a href=\"https://t.me/@id%d\">%s</a>", msg.From.ID, ShowUserName(msg.From))
		}
	default:
		if attr.IsUserAsChannel {
			senderLink = fmt.Sprintf("[%s](https://t.me/%s)", ShowChatName(msg.SenderChat), msg.SenderChat.Username)
		} else if attr.IsFromLinkedChannel || attr.IsFromAnonymous || attr.IsHasSenderChat {
			senderLink = fmt.Sprintf("[%s](https://t.me/c/%s)", ShowChatName(msg.SenderChat), RemoveIDPrefix(msg.SenderChat.ID))
		} else if msg.From != nil {
			senderLink = fmt.Sprintf("[%s](https://t.me/@id%d)", ShowUserName(msg.From), msg.From.ID)
		}
	}
	return senderLink
}

func PanicCatcher(ctx context.Context, funcName string) {
	panic := recover()
	if panic != nil {
		zerolog.Ctx(ctx).Error().
			Stack().
			Str("commit", configs.Commit).
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

// 输出版本信息
func OutputVersionInfo() string {
	var (
		gitTagURL    = "https://gitea.trle5.xyz/trle5/trbot/src/tag/"
		gitBranchURL = "https://gitea.trle5.xyz/trle5/trbot/src/branch/"
		gitCommitURL = "https://gitea.trle5.xyz/trle5/trbot/commit/"
	)
	var m runtime.MemStats;
	runtime.ReadMemStats(&m)
	hostname, _ := os.Hostname()

	if configs.BuildAt != "" {
		info := "<blockquote expandable>"
		info += fmt.Sprintf("<code>MemUsage:  </code>%s\n", humanize.IBytes(m.Sys))
		info += fmt.Sprintf("<code>Goroutine: </code>%d\n", runtime.NumGoroutine())
		info += fmt.Sprintf("<code>Uptime:    </code>%s\n", humanize.Time(configs.StartAt))
		info += fmt.Sprintf("<code>Version:   </code><a href=\"%s%s\">%s</a>\n", gitTagURL, configs.Version, configs.Version)
		info += fmt.Sprintf("<code>Branch:    </code><a href=\"%s%s\">%s</a>\n", gitBranchURL, configs.Branch, configs.Branch)
		info += fmt.Sprintf("<code>Commit:    </code><a href=\"%s%s\">%s</a> (%s)\n", gitCommitURL, configs.Commit, configs.Commit[:10], configs.Changes)
		info += fmt.Sprintf("<code>BuildOn:   </code>%s\n", configs.BuildOn)
		info += fmt.Sprintf("<code>BuildAt:   </code><a href=\"%s%s\">%s</a>\n", "https://timestamp.online/timestamp/", configs.BuildAt, configs.BuildAt)
		info += fmt.Sprintf("<code>Platform:  </code>%s / %s\n", runtime.GOOS, runtime.GOARCH)
		info += fmt.Sprintf("<code>Runtime:   </code>%s\n", runtime.Version())
		info += fmt.Sprintf("<code>Hostname:  </code>%s\n", hostname)
		info += "</blockquote>"
		return info
	}
	return fmt.Sprintln(
		"<b>Warning: No build info</b>\n",
		"\n<code>MemUsage:  </code>", humanize.IBytes(m.Sys),
		"\n<code>Goroutine: </code>", runtime.NumGoroutine(),
		"\n<code>Uptime:    </code>", humanize.Time(configs.StartAt),
		"\n<code>Platform:  </code>", runtime.GOOS, "/", runtime.GOARCH,
		"\n<code>Runtime:   </code>", runtime.Version(),
		"\n<code>Hostname:  </code>", hostname,
	)
}

func MemStats() string {
	m := runtime.MemStats{}
	runtime.ReadMemStats(&m)
	return fmt.Sprintln(
		"<code>Goroutine: </code>", runtime.NumGoroutine(),

	  "\n\n<code>Alloc:      </code>", humanize.IBytes(m.Alloc),
		"\n<code>TotalAlloc: </code>", humanize.IBytes(m.TotalAlloc),
		"\n<code>Sys:        </code>", humanize.IBytes(m.Sys),
		"\n<code>Lookups:    </code>", m.Lookups,
		"\n<code>Frees:      </code>", m.Frees,

	  "\n\n<code>HeapAlloc:    </code>", humanize.IBytes(m.HeapAlloc),
		"\n<code>HeapSys:      </code>", humanize.IBytes(m.HeapSys),
		"\n<code>HeapIdle:     </code>", humanize.IBytes(m.HeapIdle),
		"\n<code>HeapInuse:    </code>", humanize.IBytes(m.HeapInuse),
		"\n<code>HeapReleased: </code>", humanize.IBytes(m.HeapReleased),
		"\n<code>HeapObjects:  </code>", m.HeapObjects,

	  "\n\n<code>StackInuse:  </code>", humanize.IBytes(m.StackInuse),
		"\n<code>StackSys:    </code>", humanize.IBytes(m.StackSys),
		"\n<code>MSpanInuse:  </code>", humanize.IBytes(m.MSpanInuse),
		"\n<code>MSpanSys:    </code>", humanize.IBytes(m.MSpanSys),
		"\n<code>MCacheInuse: </code>", humanize.IBytes(m.MCacheInuse),
		"\n<code>MCacheSys:   </code>", humanize.IBytes(m.MCacheSys),
	)
}

func TextBlockquoteMarkdown(text string, expandable bool) (out string) {
	strs := strings.Split(text, "\n")
	for i, str := range strs {
		if expandable && i == len(strs)-1 {
			out += fmt.Sprintf(">%s||\n", bot.EscapeMarkdown(str))
		} else {
			out += fmt.Sprintf(">%s\n", bot.EscapeMarkdown(str))
		}
	}
	return
}

// not work for user
func GetChatIDLink(chatID int64) string {
	return fmt.Sprintf("https://t.me/c/%s/", RemoveIDPrefix(chatID))
}

func IgnoreHTMLTags(text string) string {
	return html.EscapeString(text)
}

func MsgLink(username string, msgID int) string {
	return fmt.Sprintf("https://t.me/%s/%d", username, msgID)
}

func MsgLinkPrivate(chatID int64, msgID int) string {
	return fmt.Sprintf("https://t.me/c/%s/%d", RemoveIDPrefix(chatID), msgID)
}

func GetCurrentFuncName() (string, string) {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "func", "failed to get function name"
	}
	return "func", runtime.FuncForPC(pc).Name()
}
