package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

func echoSticker(filePath string) *io.PipeReader {
	log.Printf("https://api.telegram.org/file/bot%s/%s\n", botToken, filePath)
	resp, err := http.Get(fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, filePath))
	if err != nil { log.Printf("error downloading file: %v", err) }
	defer resp.Body.Close()
	reader, writer := io.Pipe()
	go func() {
		defer writer.Close()
		_, err := io.Copy(writer, resp.Body)
		if err != nil {
			log.Println("Error copying to pipe:", err)
		}
	}()

	return reader
}

// 定义消息类型枚举
type MessageType int
const (
	MessageTypeText MessageType = iota
	MessageTypePhoto
	MessageTypeVideo
	MessageTypeVoice
	MessageTypeDocument
	MessageTypeAudio
	MessageTypeForwarded
	MessageTypeSticker
	MessageTypeUnknown
)

// 判断消息的类型 需要重写
func getMessageType(message *models.Message) MessageType {
	switch {
	case message.ForwardOrigin != nil:
		return MessageTypeForwarded
	case message.Photo != nil:
		return MessageTypePhoto
	case message.Video != nil:
		return MessageTypeVideo
	case message.Voice != nil:
		return MessageTypeVoice
	case message.Document != nil:
		return MessageTypeDocument
	case message.Audio != nil:
		return MessageTypeAudio
	case message.Sticker != nil:
		return MessageTypeSticker
	case message.Text != "":
		return MessageTypeText
	default:
		return MessageTypeUnknown
	}
}

// 检查用户是否是管理员
// chat type: "private", "group", "supergroup", or "channel"
// not work for "private" chats
func userIsAdmin(ctx context.Context, thebot *bot.Bot, chatID, userID any) bool {
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
func userHavePermissionDeleteMessage(ctx context.Context, thebot *bot.Bot, chatID, userID any) bool {
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
// 查找 bot token，优先级为 环境变量 > .env 文件
func whereIsBotToken() string {
	botToken = os.Getenv("BOT_TOKEN")
	if botToken == "" {
		// log.Printf("No bot token in environment, trying to read it from the .env file")
		godotenv.Load()
		botToken = os.Getenv("BOT_TOKEN")
		if botToken == "" {
			log.Fatalln("No bot token in environment and .env file, try create a bot from @botfather https://core.telegram.org/bots/tutorial#obtain-your-bot-token")
		}
		log.Printf("Get token from .env file: %s", showBotID())
	} else {
		log.Printf("Get token from environment: %s", showBotID())
	}
	return botToken
}

// 输出 bot 的 ID
func showBotID() string {
	var botID string
	for _, char := range botToken {
		if unicode.IsDigit(char) {
			botID += string(char)
		} else {
			break // 遇到非数字字符停止
		}
	}
	return botID
}

func usingWebhook() bool {
	webhookURL = os.Getenv("WEBHOOK_URL")
	if webhookURL == "" {
		// 到这里可能变量没在环境里，试着读一下 .env 文件
		godotenv.Load()
		webhookURL = os.Getenv("WEBHOOK_URL")
		if webhookURL == "" {
			// 到这里就是 .env 文件里也没有，不启用
			log.Printf("No Webhook URL in environment and .env file, using getUpdate")
			return false
		}
		// 从 .env 文件中获取到了 URL，启用 Webhook
		log.Printf("Get Webhook URL from .env file: %s", webhookURL)
		return true
	} else {
		// 从环境变量中获取到了 URL，启用 Webhook
		log.Printf("Get Webhook URL from environment: %s", webhookURL)
		return true
	}
}

func setUpWebhook(ctx context.Context, thebot *bot.Bot, url string) {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != url {
		if webHookInfo.URL == "" {
			log.Println("Webhook not set, setting it now...")
		} else {
			log.Printf("unsame Webhook URL [%s], save it and setting up new URL...", webHookInfo.URL)
			printLogAndSave(time.Now().Format(time.RFC3339) + " (unsame) old Webhook URL: " + webHookInfo.URL)
		}
		success, err := thebot.SetWebhook(ctx, &bot.SetWebhookParams{ URL: url })
		if err != nil { log.Panicln("Set Webhook URL err:", err) }
		if success { log.Println("Webhook setup successfully:", url) }

	} else {
		log.Println("Webhook is already set:", webHookInfo.URL)
	}
}

func saveAndCleanRemoteWebhookURL(ctx context.Context, thebot *bot.Bot) *models.WebhookInfo {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != "" {
		log.Printf("found Webhook URL [%s] set at api server, save and clean it...", webHookInfo.URL)
		printLogAndSave(time.Now().Format(time.RFC3339) + " (remote) old Webhook URL: " + webHookInfo.URL)
		thebot.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
			DropPendingUpdates: false,
		})
		return webHookInfo
	}
	return nil
}

func printLogAndSave(message string) {
	log.Println(message)
	// 打开日志文件，如果不存在则创建
	file, err := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println(err)
		return
	}
	defer file.Close()

	// 将文本写入日志文件
	_, err = file.WriteString(message + "\n")
	if err != nil {
		log.Println(err)
		return
	}
}

// 从 log.txt 读取文件
func readLog() []string {
	// 打开日志文件
	file, err := os.Open(logFile_path)
	if err != nil {
		log.Println(err)
		return nil
	}
	defer file.Close()

	// 读取文件内容
	scanner := bufio.NewScanner(file)
	var lines []string
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		log.Println(err)
		return nil
	}
	return lines
}

func privateLogToChat(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	thebot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    logChat_ID,
		Text:      fmt.Sprintf("[%s %s](t.me/@id%d) say: \n%s", update.Message.From.FirstName, update.Message.From.LastName, update.Message.Chat.ID, update.Message.Text),
		ParseMode: models.ParseModeMarkdownV1,
	})
}


// 如果 target 是 candidates 的一部分, 返回 true
// 常规类型会判定值是否相等，字符串如果包含也符合条件，例如 "bc" 在 "abcd" 中
func AnyContains(target any, candidates ...any) bool {
	for _, candidate := range candidates {
		if candidates == nil { continue }
		// fmt.Println(reflect.ValueOf(target).Kind(), reflect.ValueOf(candidate).Kind(), reflect.Array, reflect.Slice)
		targetKind := reflect.ValueOf(target).Kind()
		candidateKind := reflect.ValueOf(candidate).Kind()
		if targetKind != candidateKind && !AnyContains(candidateKind, reflect.Slice, reflect.Array) {
			log.Printf("[Warn] (func)AnyContains: candidate(%v) not match target(%v)", candidateKind, targetKind)
		}
		switch c := candidate.(type) {
		case string:
			if targetKind == reflect.String && strings.Contains(c, target.(string)) {
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
			if reflect.ValueOf(target).Kind() == reflect.String && strings.Contains(c, target.(string)) {
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

// 允许响应带有机器人用户名后缀的命令，例如 /help@examplebot
func commandMaybeWithSuffixUsername(commandFields []string, command string) bool {
	atBotUsername := "@" + botMe.Username
	if commandFields[0] == command || commandFields[0] == command + atBotUsername {
		return true
	}
	return false
}

func outputVersionInfo() string {
	// 获取 git sha 和 commit 时间
	c, _ := exec.Command("git", "rev-parse", "HEAD").Output()
	// 获取 git 分支
	b, _ := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	r := runtime.Version()
	h, _ := os.Hostname()
	info := fmt.Sprintf("Branch: %sCommit: [%s](https://gitea.trle5.xyz/trle5/trbot/commit/%s)\nRuntime: %s\nHostname: %s", b, c[:10], c, r, h)
	return info
}

func showUserNickName(update *models.Update) string {
	if update.Message.From.LastName != "" {
		return update.Message.From.FirstName + " " + update.Message.From.LastName
	} else {
		return update.Message.From.FirstName
	}
}

func InlineResultPagination(queryFields []string, results []models.InlineQueryResult) []models.InlineQueryResult {
	// 当 result 的数量超过 InlineResultsPerPage 时，进行分页
	// fmt.Println(len(results), InlineResultsPerPage)
	if len(results) > InlineResultsPerPage {
		// 获取 update.InlineQuery.Query 末尾的 `-<数字>` 来选择输出第几页
		var pageNow int = 1
		var pageSize = (InlineResultsPerPage -1)

		if len(queryFields) > 0 && strings.HasPrefix(queryFields[len(queryFields)-1], InlinePaginationSymbol) {
			var err error
			pageNow, err = strconv.Atoi(queryFields[len(queryFields)-1][1:])
			if err != nil {
				if queryFields[len(queryFields)-1][1:] != "" {
					return []models.InlineQueryResult{&models.InlineQueryResultArticle{
						ID: "noThisOperation",
						Title: "无效的操作",
						Description: fmt.Sprintf("若您想翻页查看，请尝试输入 `%s2` 来查看第二页", InlinePaginationSymbol),
						InputMessageContent: &models.InputTextMessageContent{
							MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
							ParseMode: models.ParseModeMarkdownV1,
						},
					}}
				} else {
					return []models.InlineQueryResult{&models.InlineQueryResultArticle{
						ID: "keepInputNumber",
						Title: "请继续输入数字",
						Description: fmt.Sprintf("继续输入一个数字来查看对应的页面，当前列表有 %d 页", (len(results) + pageSize - 1) / pageSize),
						InputMessageContent: &models.InputTextMessageContent{
							MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
							ParseMode: models.ParseModeMarkdownV1,
						},
					}}
				}
			}
		}

		start := (pageNow - 1) * pageSize
		end := start + pageSize

		if start >= len(results) {
			return []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID: "wrongPageNumber",
				Title: "错误的页码",
				Description: fmt.Sprintf("您输入的页码 %d 超出范围，当前列表有 %d 页", pageNow, (len(results) + pageSize - 1) / pageSize),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
					ParseMode: models.ParseModeMarkdownV1,
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
				ID: "paginationPage",
				Title: fmt.Sprintf("当前您在第 %d 页", pageNow),
				Description: fmt.Sprintf("后面还有 %d 页内容，输入 %s%d 查看下一页", totalPages - pageNow, InlinePaginationSymbol, pageNow + 1),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		} else {
			pageResults = append(pageResults, &models.InlineQueryResultArticle{
				ID: "paginationPage",
				Title: fmt.Sprintf("当前您在第 %d 页", pageNow),
				Description: "后面已经没有东西了",
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}

		return pageResults
	} else if len(queryFields) > 0 && strings.HasPrefix(queryFields[len(queryFields)-1], InlinePaginationSymbol) {
		return []models.InlineQueryResult{&models.InlineQueryResultArticle{
			ID: "noNeedPagination",
			Title: "没有多余的内容",
			Description: fmt.Sprintf("只有 %d 个条目，你想翻页也没有多的了", len(results)),
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: "由于在使用 inline 模式时没有正确填写参数，无法完成消息",
				ParseMode: models.ParseModeMarkdownV1,
			},
		}}
	} else {
		return results
	}
}

func InlineQueryMatchMultKeyword(queryFields []string, Keyword []string, inSubCommand bool) bool {
	var allkeywords int
	if strings.HasPrefix(queryFields[len(queryFields)-1], InlinePaginationSymbol) {
		queryFields = queryFields[:len(queryFields) -1]
	} else {
		allkeywords = len(queryFields)
	}
	if inSubCommand && len(queryFields) > 0 {
		queryFields = queryFields[1:]
	}
	if allkeywords == 1 {
		if AnyContains(queryFields[0], Keyword) {
			return true
		}
	} else {
		var allMatch bool = true

		for _, n := range queryFields {
			if AnyContains(n, Keyword) {
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
