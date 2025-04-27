package utils

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"trbot/database/db_struct"
	"trbot/utils/consts"
	"trbot/utils/plugin_utils"
	"trbot/utils/updatetype"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

// 如果 target 是 candidates 的一部分, 返回 true
// 常规类型会判定值是否相等，字符串如果包含也符合条件，例如 "bc" 在 "abcd" 中
func AnyContains(target any, candidates ...any) bool {
	for _, candidate := range candidates {
		if candidate == nil { continue }
		// fmt.Println(reflect.ValueOf(target).Kind(), reflect.ValueOf(candidate).Kind(), reflect.Array, reflect.Slice)
		targetKind := reflect.ValueOf(target).Kind()
		candidateKind := reflect.ValueOf(candidate).Kind()
		if targetKind != candidateKind && !AnyContains(candidateKind, reflect.Slice, reflect.Array) {
			log.Printf("[Warn] (func)AnyContains: candidate(%v) not match target(%v)", candidateKind, targetKind)
		}
		switch c := candidate.(type) {
		case string:
			if targetKind == reflect.String && strings.Contains(strings.ToLower(c), strings.ToLower(target.(string))) {
				return true
			}
		default:
			if reflect.DeepEqual(target, c) {
				return true
			}
			if reflect.ValueOf(c).Kind() == reflect.Slice || reflect.ValueOf(c).Kind() == reflect.Array {
				if checkNested(target, reflect.ValueOf(c)) {
					return true
				}
			}
		}
	}
	return false
}

// 为 AnyContains 的递归函数
func checkNested(target any, value reflect.Value) bool {
	// fmt.Println(reflect.ValueOf(value.Index(0).Interface()).Kind())
	if reflect.TypeOf(target) != reflect.TypeOf(value.Index(0).Interface()) && !AnyContains(reflect.ValueOf(value.Index(0).Interface()).Kind(), reflect.Slice, reflect.Array) {
		log.Printf("[Error] (func)AnyContains: candidates's subitem(%v) not match target(%v), skip this compare", reflect.TypeOf(value.Index(0).Interface()), reflect.TypeOf(target))
		return false
	}
	for i := 0; i < value.Len(); i++ {
		element := value.Index(i).Interface()
		switch c := element.(type) {
		case string:
			if reflect.ValueOf(target).Kind() == reflect.String && strings.Contains(strings.ToLower(c), strings.ToLower(target.(string))) {
				return true
			}
		default:
			if reflect.DeepEqual(target, c) {
				return true
			}
			// Check nested slices or arrays
			elemValue := reflect.ValueOf(c)
			if elemValue.Kind() == reflect.Slice || elemValue.Kind() == reflect.Array {
				if checkNested(target, elemValue) {
					return true
				}
			}
		}
	}
	return false
}

// 检查用户是否是管理员
// chat type: "private", "group", "supergroup", or "channel"
// not work for "private" chats
func UserIsAdmin(ctx context.Context, thebot *bot.Bot, chatID, userID any) bool {
	admins, err := thebot.GetChatAdministrators(ctx, &bot.GetChatAdministratorsParams{ ChatID: chatID })
	if err != nil {
		log.Printf("Failed to get chat administrators: %v", err)
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
	case int:
		return AnyContains(value, admins_userIDs)
	case int64:
		// fmt.Println(value)
		return AnyContains(value, admins_userIDs)
	case string:
		// fmt.Println(value)
		if strings.ContainsAny(value, "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ_") {
			return AnyContains(value, admins_usernames)
		} else {
			int_userID, _ := strconv.Atoi(value)
			return AnyContains(int64(int_userID), admins_userIDs)
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
		log.Printf("Failed to get chat administrators: %v", err)
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
	case int:
		return AnyContains(value, adminshavepermission_userIDs)
	case int64:
		// fmt.Println(value)
		return AnyContains(value, adminshavepermission_userIDs)
	case string:
		// fmt.Println(value)
		if strings.ContainsAny(value, "aAbBcCdDeEfFgGhHiIjJkKlLmMnNoOpPqQrRsStTuUvVwWxXyYzZ_") {
			return AnyContains(value, adminshavepermission_usernames)
		} else {
			int_userID, _ := strconv.Atoi(value)
			return AnyContains(int64(int_userID), adminshavepermission_userIDs)
		}
	default:
		log.Println("userID type not supported")
		return false
	}
}

// 将 InlineQueryResult 列表进行分页处理
func InlineResultPagination(queryFields []string, results []models.InlineQueryResult) []models.InlineQueryResult {
	// 当 result 的数量超过 InlineResultsPerPage 时，进行分页
	// fmt.Println(len(results), InlineResultsPerPage)
	if len(results) > consts.InlineResultsPerPage {
		// 获取 update.InlineQuery.Query 末尾的 `<分页符号><数字>` 来选择输出第几页
		var pageNow int = 1
		var pageSize = (consts.InlineResultsPerPage - 1)

		pageNow, err := InlineExtractPageNumber(queryFields)
		// 读取页码发生错误
		if err != nil {
			// 输入了分页符号没有输入数字
			if queryFields[len(queryFields)-1][1:] == "" {
				return []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:          "keepInputNumber",
					Title:       "请继续输入数字",
					Description: fmt.Sprintf("继续输入一个数字来查看对应的页面，当前列表有 %d 页", (len(results)+pageSize-1)/pageSize),
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "用户在尝试进行分页时点击了分页提示...",
						ParseMode:   models.ParseModeMarkdownV1,
					},
				}}
			} else {
				// 在分页符号后输入了非数字字符
				return []models.InlineQueryResult{&models.InlineQueryResultArticle{
					ID:          "noThisOperation",
					Title:       "无效的操作",
					Description: fmt.Sprintf("若您想翻页查看，请尝试输入 `%s2` 来查看第二页", consts.InlinePaginationSymbol),
					InputMessageContent: &models.InputTextMessageContent{
						MessageText: "用户在尝试进行分页时输入了错误的页码并点击了分页提示...",
						ParseMode:   models.ParseModeMarkdownV1,
					},
				}}
			}
		}

		start := (pageNow - 1) * pageSize
		end := start + pageSize

		if start < 0 || start >= len(results) {
			return []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:          "wrongPageNumber",
				Title:       "错误的页码",
				Description: fmt.Sprintf("您输入的页码 %d 超出范围，当前列表有 %d 页", pageNow, (len(results)+pageSize-1)/pageSize),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "用户在浏览不存在的页面时点击了错误页码提示...",
					ParseMode:   models.ParseModeMarkdownV1,
				},
			}}
		}

		if end > len(results) {
			end = len(results)
		}
		pageResults := results[start:end]

		// 添加翻页提示
		if end < len(results) {
			totalPages := (len(results) + pageSize - 1) / pageSize
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID:          "paginationPage",
				Title:       fmt.Sprintf("当前您在第 %d 页", pageNow),
				Description: fmt.Sprintf("后面还有 %d 页内容，输入 %s%d 查看下一页", totalPages-pageNow, consts.InlinePaginationSymbol, pageNow+1),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "用户在挑选内容时点击了分页提示...",
					ParseMode:   models.ParseModeMarkdownV1,
				},
			})
		} else {
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID:          "paginationPage",
				Title:       fmt.Sprintf("当前您在第 %d 页", pageNow),
				Description: "后面已经没有东西了",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "用户在挑选内容时点击了分页提示...",
					ParseMode:   models.ParseModeMarkdownV1,
				},
			})
		}

		return pageResults
	} else if len(queryFields) > 0 && strings.HasPrefix(queryFields[len(queryFields)-1], consts.InlinePaginationSymbol) {
		return []models.InlineQueryResult{&models.InlineQueryResultArticle{
			ID:          "noNeedPagination",
			Title:       "没有多余的内容",
			Description: fmt.Sprintf("只有 %d 个条目，你想翻页也没有多的了", len(results)),
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
				ParseMode:   models.ParseModeMarkdownV1,
			},
		}}
	} else {
		return results
	}
}

// 从 inline 字段中提取子命令字符串
func InlineExtractSubCommand(fields []string) string {
	if len(fields) == 0 {
		return ""
	}

	// 判断是不是子命令
	if strings.HasPrefix(fields[0], consts.InlineSubCommandSymbol) {
		return strings.TrimPrefix(fields[0], consts.InlineSubCommandSymbol)
	}
	return ""
}

// 从 Inline 字段中提取查询关键词，去除子命令的前缀或后缀的分页符号
func InlineExtractKeywords(fields []string) []string {
	if len(fields) == 0 {
		return []string{}
	}

	// 判断是不是子命令
	if strings.HasPrefix(fields[0], consts.InlineSubCommandSymbol) {
		fields = fields[1:]
	}
	// 判断有没有分页符号
	if len(fields) > 0 && strings.HasPrefix(fields[len(fields)-1], consts.InlinePaginationSymbol) {
		fields = fields[:len(fields)-1]
	}

	return fields
}

// 从 inline 字段中提取页码
func InlineExtractPageNumber(fields []string) (int, error) {
	if len(fields) == 0 {
		return 1, nil
	}

	// 判断有没有分页符号
	if strings.HasPrefix(fields[len(fields)-1], consts.InlinePaginationSymbol) {
		return strconv.Atoi(fields[len(fields)-1][1:])
	}
	return 1, nil
}

// 从 inline 查询字段中匹配多个关键词
func InlineQueryMatchMultKeyword(fields []string, keywords []string) bool {
	var allkeywords int

	fields = InlineExtractKeywords(fields)
	if len(fields) != 0 {
		allkeywords = len(fields)
	}
	// fmt.Println(allkeywords)
	if allkeywords == 1 {
		if len(keywords) == 0 {
			return false
		}
		if AnyContains(fields[0], keywords) {
			return true
		}
	} else {
		var allMatch bool = true

		for _, n := range fields {
			if AnyContains(n, keywords) {
				// 保持 current 内容，继续过滤
				// continue
			} else {
				// 只要有一个关键词未匹配，返回 false
				allMatch = false
			}
		}
		if allMatch {
			return true
		}
	}
	return false
}

// 允许响应带有机器人用户名后缀的命令，例如 /help@examplebot
func CommandMaybeWithSuffixUsername(commandFields []string, command string) bool {
	atBotUsername := "@" + consts.BotMe.Username
	if commandFields[0] == command || commandFields[0] == command + atBotUsername {
		return true
	}
	return false
}

func ShowUserName(user *models.User) string {
	if user.LastName != "" {
		return user.FirstName + " " + user.LastName
	} else {
		return user.FirstName
	}
}

func ShowChatName(chat *models.Chat) string {
	if chat.Title != "" { // 群组
		return chat.Title
	} else if chat.LastName != "" { // 可能是用户正在与 bot 发送信息
		return chat.FirstName + " " + chat.LastName
	} else {
		return chat.FirstName
	}
}


func BuildDefaultInlineCommandSelectKeyboard(chatInfo *db_struct.ChatInfo) models.ReplyMarkup {
	var inlinePlugins [][]models.InlineKeyboardButton
	for _, v := range plugin_utils.AllPlugins.InlineCommandList {
		if v.Attr.IsCantBeDefault {
			continue
		}
		if chatInfo.DefaultInlinePlugin == v.Command {
			inlinePlugins = append(inlinePlugins, []models.InlineKeyboardButton{{
				Text: fmt.Sprintf("✅ [ %s%s ] - %s", consts.InlineSubCommandSymbol, v.Command, v.Description),
				CallbackData: "inline_default_" + v.Command,
			}})
		} else {
			inlinePlugins = append(inlinePlugins, []models.InlineKeyboardButton{{
				Text: fmt.Sprintf("[ %s%s ] - %s", consts.InlineSubCommandSymbol, v.Command, v.Description),
				CallbackData: "inline_default_" + v.Command,
			}})
		}
	}
	
	inlinePlugins = append(inlinePlugins, []models.InlineKeyboardButton{
		{
			Text: "取消默认命令",
			CallbackData: "inline_default_none",
		},
		{
			Text: "浏览 inline 命令菜单",
			SwitchInlineQueryCurrentChat: "+",
		},
	})

	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: inlinePlugins,
	}

	return kb
}

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
	attr := updatetype.GetMessageAttribute(msg)

	switch ParseMode {
	case models.ParseModeHTML:
		if attr.IsFromLinkedChannel || attr.IsFromAnonymous {
			senderLink += fmt.Sprintf("<a href=\"https://t.me/c/%s\">%s</a>", RemoveIDPrefix(msg.SenderChat.ID), ShowChatName(msg.SenderChat))
		} else if attr.IsUserAsChannel {
			senderLink += fmt.Sprintf("<a href=\"https://t.me/%s\">%s</a>", msg.SenderChat.Username, ShowChatName(msg.SenderChat))
		} else {
			senderLink += fmt.Sprintf("<a href=\"https://t.me/@id%d\">%s</a>", msg.From.ID, ShowUserName(msg.From))
		}
	default:
		if attr.IsFromLinkedChannel || attr.IsFromAnonymous {
			senderLink += fmt.Sprintf("[%s][https://t.me/c/%s]", ShowChatName(msg.SenderChat), RemoveIDPrefix(msg.SenderChat.ID))
		} else if attr.IsUserAsChannel {
			senderLink += fmt.Sprintf("[%s][https://t.me/%s]", ShowChatName(msg.SenderChat), msg.SenderChat.Username)
		} else {
			senderLink += fmt.Sprintf("[%s][https://t.me/@id%d]", ShowUserName(msg.From), msg.From.ID)
		}
	}
	return senderLink
}

// 一个通用的 yaml 结构体读取函数
func LoadYAML(pathToFile string, out interface{}) error {
	file, err := os.ReadFile(pathToFile)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	if err := yaml.Unmarshal(file, out); err != nil {
		return fmt.Errorf("解析 YAML 失败: %w", err)
	}

	return nil
}

// 一个通用的 yaml 结构体保存函数，目录和文件不存在则创建，并以结构体类型保存
func SaveYAML(pathToFile string, data interface{}) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("编码 YAML 失败: %w", err)
	}

	dir := filepath.Dir(pathToFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	if err := os.WriteFile(pathToFile, out, 0644); err != nil {
		return fmt.Errorf("写入文件失败: %w", err)
	}

	return nil
}
