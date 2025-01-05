package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
)


func main() {
	botToken = whereIsBotToken()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(inlinehandler),
		// bot.WithMiddlewares(),
		// bot.WithMessageTextHandler("/select", bot.MatchTypeExact, commandHandler),
		bot.WithMessageTextHandler("", bot.MatchTypeContains, catchAllHandler),
		// bot.WithCallbackQueryDataHandler("btn_", bot.MatchTypePrefix, callbackHandler),
	}

	thebot, err := bot.New(botToken, opts...)
	if err != nil { panic(err) }

	botMe, _ = thebot.GetMe(ctx)
	log.Printf("name[%s] [@%s] id[%d]", botMe.FirstName, botMe.Username, botMe.ID)

	log.Printf("starting %d\n", botMe.ID)
	log.Printf("logChat_ID: %v", logChat_ID)

	database, err = ReadYamlDB(db_path + metadatafile_name)
	if err != nil {
		log.Println("read yaml db error: ", err)
	}

	go saveDatabase(savenow)

	// 检查是否设定了 webhookURL 环境变量
	if usingWebhook() { // Webhook
		setUpWebhook(ctx, thebot, webhookURL)
		log.Println("Working at Webhook Mode")
		go thebot.StartWebhook(ctx)
		go func() {
			err := http.ListenAndServe(webhookPort, thebot.WebhookHandler())
			if err != nil { log.Panicln(err) }
		}()
		<-ctx.Done() // 等待中断信号
		// log.Println("manually stopped")
	} else { // getUpdate, aka Long Polling
		// 保存并清理云端 Webhook URL，否则该模式会不生效 https://core.telegram.org/bots/api#getupdates
		saveAndCleanRemoteWebhookURL(ctx, thebot)
		log.Println("Working at Long Polling Mode")
		thebot.Start(ctx)
		<-ctx.Done() // 等待中断信号
	}
	log.Println("manually stopped")

}
