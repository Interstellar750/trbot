package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
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

// 定义消息类型枚举
type MessageType struct {
	Attribute MessageAttribute

	// https://core.telegram.org/bots/api#message

	Animation bool // call gif, mpeg4 format, can save to GIFs, no caption
	Audio     bool // or call music, can have caption, some music may as a document
	Document  bool // can have caption
	PaidMedia bool // photo or video, unknow caption
	Photo     bool // a list, sort by resolution
	Sticker   bool // sticker, but some .webp file maybe will send as sticker, actual file format and resolution may not match the limitations. no caption
	Story     bool
	Video     bool
	VideoNote bool // A circular video shot in Telegram
	Voice     bool // can have caption
	OnlyText  bool // just text message, todo
	Contact   bool
	Dice      bool
	Game      bool
	Poll      bool
	Venue     bool
	Location  bool
	Invoice   bool
	Giveaway   bool
}

type MessageAttribute struct {
	IsFromAnonymous      bool // anonymous admin or owner in group/supergroup
	IsFromLinkedChannel  bool // is automatic forward post from linked channel
	IsHasSenderChat      bool // sender of the message when sent on behalf of a chat, eg current group/supergroup or linked channel
	IsChatEnableForum    bool // group or supergroup is enable topic
	IsForwardMessage     bool // not a origin message, forward from somewhere
	IsTopicMessage       bool // the message is sent to a forum topic
	IsAutomaticForward   bool // is post from linked channel, auto forward by server
	IsReplyToMessage     bool // message reply to a another message
	IsExternalReply      bool // message reply from another chat, or call 'quote'
	IsQuoteToMessage     bool // reply from another chat or manual quote from current chat, maybe only true for text message
	IsQuoteHasEntities   bool // is quote message has entities
	IsManualQuote        bool // user manually select text to quote a message. if false, just use 'reply to other chat'
	IsReplyToStory       bool // TODO
	IsViaBot             bool // message by inline mode
	IsEdited             bool // message aready edited
	IsFromOffline        bool // eg scheduled message
	IsGroupedMedia       bool // media group, like select more than one file or photo to send
	IsTextHasEntities    bool // message has text entities
	IsMessageHasEffect   bool // message has effect
	IsCaptionHasEntities bool // message has caption entities
	IsCaptionAboveMedia  bool
	IsMediaHasSpoiler    bool
}

// 判断消息的类型
func getMessageType(msg *models.Message) MessageType {
	var msgType MessageType
	msgType.Attribute = getMessageAttribute(msg)
	if msg.Document != nil {
		if msg.Animation != nil && msg.Animation.FileID == msg.Document.FileID && msg.Document.MimeType == "video/mp4" {
			msgType.Animation = true
		} else {
			msgType.Document = true
		}
	}
	if msg.Audio != nil {
		msgType.Audio = true
	}
	if msg.PaidMedia != nil {
		msgType.PaidMedia = true
	}
	if msg.Photo != nil {
		msgType.Photo = true
	}
	if msg.Sticker != nil {
		msgType.Sticker = true
	}
	if msg.Story != nil {
		msgType.Story = true
	}
	if msg.Video != nil {
		msgType.Video = true
	}
	if msg.VideoNote != nil {
		msgType.VideoNote = true
	}
	if msg.Voice != nil {
		msgType.Voice = true
	}
	if msg.Contact != nil {
		msgType.Contact = true
	}
	if msg.Dice != nil {
		msgType.Dice = true
	}
	if msg.Game != nil {
		msgType.Game = true
	}
	if msg.Poll != nil {
		msgType.Poll = true
	}
	if msg.Venue != nil {
		msgType.Venue = true
	}
	if msg.Location != nil {
		msgType.Location = true
	}
	if msg.Invoice != nil {
		msgType.Invoice = true
	}
	if msg.Giveaway != nil {
		msgType.Giveaway = true
	}
	return msgType
}

func getMessageAttribute(msg *models.Message) MessageAttribute {
	var attribute MessageAttribute
	if msg.SenderChat != nil {
		attribute.IsHasSenderChat = true
		if msg.From.ID == 1087968824 && msg.From != nil && msg.From.IsBot && msg.SenderChat.ID == msg.Chat.ID {
			attribute.IsFromAnonymous = true
		}
		if msg.From.ID == 777000 && msg.ForwardOrigin != nil && msg.ForwardOrigin.MessageOriginChannel != nil && msg.SenderChat.ID == msg.ForwardOrigin.MessageOriginChannel.Chat.ID {
			attribute.IsFromLinkedChannel = true
		}
	}
	if msg.Chat.IsForum {
		attribute.IsChatEnableForum = true
	}
	if msg.ForwardOrigin != nil {
		attribute.IsForwardMessage = true
	}
	if msg.IsTopicMessage {
		attribute.IsTopicMessage = true
	}
	if msg.IsAutomaticForward {
		attribute.IsAutomaticForward = true
	}
	if msg.ReplyToMessage != nil {
		attribute.IsReplyToMessage = true
	}
	if msg.ExternalReply != nil {
		attribute.IsExternalReply = true
	}
	if msg.Quote != nil {
		attribute.IsQuoteToMessage = true
		if msg.Quote.Entities != nil {
			attribute.IsQuoteHasEntities = true
		}
		if msg.Quote.IsManual {
			attribute.IsManualQuote = true
		}
	}
	if msg.ReplyToStore != nil {
		attribute.IsReplyToStory = true
	}
	if msg.ViaBot != nil {
		attribute.IsViaBot = true
	}
	if msg.EditDate != 0 {
		attribute.IsEdited = true
	}
	if msg.IsFromOffline {
		attribute.IsFromOffline = true
	}
	if msg.MediaGroupID != "" {
		attribute.IsGroupedMedia = true
	}
	if msg.Entities != nil {
		attribute.IsTextHasEntities = true
	}
	if msg.EffectID != "" {
		attribute.IsMessageHasEffect = true
	}
	if msg.CaptionEntities != nil {
		attribute.IsCaptionHasEntities = true
	}
	if msg.ShowCaptionAboveMedia {
		attribute.IsCaptionAboveMedia = true
	}
	if msg.HasMediaSpoiler {
		attribute.IsMediaHasSpoiler = true
	}
	return attribute
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

func setUpWebhook(ctx context.Context, thebot *bot.Bot, params *bot.SetWebhookParams) {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != params.URL {
		if webHookInfo.URL == "" {
			log.Println("Webhook not set, setting it now...")
		} else {
			log.Printf("unsame Webhook URL [%s], save it and setting up new URL...", webHookInfo.URL)
			printLogAndSave(time.Now().Format(time.RFC3339) + " (unsame) old Webhook URL: " + webHookInfo.URL)
		}
		success, err := thebot.SetWebhook(ctx, params)
		if err != nil { log.Panicln("Set Webhook URL err:", err) }
		if success { log.Println("Webhook setup successfully:", params.URL) }

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
	file, err := os.OpenFile(logFile_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
	// 获取 commit 说明
	m, _ := exec.Command("git", "log", "-1", "--pretty=%s").Output()
	r := runtime.Version()
	grs := runtime.NumGoroutine()
	h, _ := os.Hostname()
	info := fmt.Sprintf("Branch: %sCommit: [%s - %s](https://gitea.trle5.xyz/trle5/trbot/commit/%s)\nRuntime: %s\nGoroutine: %d\nHostname: %s", b, m, c[:10], c, r, grs, h)
	return info
}

func showUserName(user *models.User) string {
	if user.LastName != "" {
		return user.FirstName + " " + user.LastName
	} else {
		return user.FirstName
	}
}

func showChatName(chat *models.Chat) string {
	if chat.Title != "" { // 群组
		return chat.Title
	} else if chat.LastName != "" { // 
		return chat.FirstName + " " + chat.LastName
	} else {
		return chat.FirstName
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

func InlineQueryMatchMultKeyword(queryFields []string, Keyword []string) bool {
	var allkeywords int
	if strings.HasPrefix(queryFields[len(queryFields)-1], InlinePaginationSymbol) {
		queryFields = queryFields[:len(queryFields) -1]
	} else {
		allkeywords = len(queryFields)
	}
	if strings.HasPrefix(queryFields[0], InlineSubCommandSymbol) {
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
