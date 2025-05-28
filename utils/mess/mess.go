package mess

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
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
func ReadLog() []string {
	// 打开日志文件
	file, err := os.Open(consts.LogFilePath)
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
		ChatID:    configs.BotConfig.LogChatID,
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
