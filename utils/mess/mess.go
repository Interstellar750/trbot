package mess

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"runtime"

	"trbot/utils/configs"
	"trbot/utils/consts"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func PrintLogAndSave(message string) {
	log.Println(message)
	// 打开日志文件，如果不存在则创建
	file, err := os.OpenFile(consts.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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

func PrivateLogToChat(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	thebot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    configs.BotConfig.LogChatID,
		Text:      fmt.Sprintf("[%s %s](t.me/@id%d) say: \n%s", update.Message.From.FirstName, update.Message.From.LastName, update.Message.Chat.ID, update.Message.Text),
		ParseMode: models.ParseModeMarkdownV1,
	})
}

func OutputVersionInfo() string {
	hostname, _ := os.Hostname()
	var gitURL string = "https://gitea.trle5.xyz/trle5/trbot/commit/"
	var info   string
	if consts.BuildAt != "" {
		info += fmt.Sprintf("`Version:   `%s\n", consts.Version)
		info += fmt.Sprintf("`Branch:    `%s\n", consts.Branch)
		info += fmt.Sprintf("`Commit:    `[%s](%s%s) (%s)\n", consts.Commit[:10], gitURL, consts.Commit, consts.Changes)
		info += fmt.Sprintf("`BuildAt:   `%s\n", consts.BuildAt)
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
