package mess

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"runtime"
	"time"
	"unicode"

	"trbot/utils/consts"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
)

// 查找 bot token，优先级为 环境变量 > .env 文件
func WhereIsBotToken() string {
	consts.BotToken = os.Getenv("BOT_TOKEN")
	if consts.BotToken == "" {
		// log.Printf("No bot token in environment, trying to read it from the .env file")
		godotenv.Load()
		consts.BotToken = os.Getenv("BOT_TOKEN")
		if consts.BotToken == "" {
			log.Fatalln("No bot token in environment and .env file, try create a bot from https://t.me/@botfather https://core.telegram.org/bots/tutorial#obtain-your-bot-token")
		}
		log.Printf("Get token from .env file: %s", ShowBotID())
	} else {
		log.Printf("Get token from environment: %s", ShowBotID())
	}
	return consts.BotToken
}

// 输出 bot 的 ID
func ShowBotID() string {
	var botID string
	for _, char := range consts.BotToken {
		if unicode.IsDigit(char) {
			botID += string(char)
		} else {
			break // 遇到非数字字符停止
		}
	}
	return botID
}

func UsingWebhook() bool {
	consts.WebhookURL = os.Getenv("WEBHOOK_URL")
	if consts.WebhookURL == "" {
		// 到这里可能变量没在环境里，试着读一下 .env 文件
		godotenv.Load()
		consts.WebhookURL = os.Getenv("WEBHOOK_URL")
		if consts.WebhookURL == "" {
			// 到这里就是 .env 文件里也没有，不启用
			log.Printf("No Webhook URL in environment and .env file, using getUpdate")
			return false
		}
		// 从 .env 文件中获取到了 URL，启用 Webhook
		log.Printf("Get Webhook URL from .env file: %s", consts.WebhookURL)
		return true
	} else {
		// 从环境变量中获取到了 URL，启用 Webhook
		log.Printf("Get Webhook URL from environment: %s", consts.WebhookURL)
		return true
	}
}

func SetUpWebhook(ctx context.Context, thebot *bot.Bot, params *bot.SetWebhookParams) {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != params.URL {
		if webHookInfo.URL == "" {
			log.Println("Webhook not set, setting it now...")
		} else {
			log.Printf("unsame Webhook URL [%s], save it and setting up new URL...", webHookInfo.URL)
			PrintLogAndSave(time.Now().Format(time.RFC3339) + " (unsame) old Webhook URL: " + webHookInfo.URL)
		}
		success, err := thebot.SetWebhook(ctx, params)
		if err != nil { log.Panicln("Set Webhook URL err:", err) }
		if success { log.Println("Webhook setup successfully:", params.URL) }

	} else {
		log.Println("Webhook is already set:", webHookInfo.URL)
	}
}

func SaveAndCleanRemoteWebhookURL(ctx context.Context, thebot *bot.Bot) *models.WebhookInfo {
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil { log.Println(err) }
	if webHookInfo.URL != "" {
		log.Printf("found Webhook URL [%s] set at api server, save and clean it...", webHookInfo.URL)
		PrintLogAndSave(time.Now().Format(time.RFC3339) + " (remote) old Webhook URL: " + webHookInfo.URL)
		thebot.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
			DropPendingUpdates: false,
		})
		return webHookInfo
	}
	return nil
}

func PrintLogAndSave(message string) {
	log.Println(message)
	// 打开日志文件，如果不存在则创建
	file, err := os.OpenFile(consts.LogFile_path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
func ReadLog() []string {
	// 打开日志文件
	file, err := os.Open(consts.LogFile_path)
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

func PrivateLogToChat(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	thebot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    consts.LogChat_ID,
		Text:      fmt.Sprintf("[%s %s](t.me/@id%d) say: \n%s", update.Message.From.FirstName, update.Message.From.LastName, update.Message.Chat.ID, update.Message.Text),
		ParseMode: models.ParseModeMarkdownV1,
	})
}

func OutputVersionInfo() string {
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
