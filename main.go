package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)


func main() {
	botToken = whereIsBotToken()

	IsDebugMode = os.Getenv("DEBUG") == "true"
	if IsDebugMode {
		log.Println("running in debug mode, all log will be printed to stdout")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	allowedUpdates := bot.AllowedUpdates{
		models.AllowedUpdateMessage,
		models.AllowedUpdateEditedMessage,
		models.AllowedUpdateChannelPost,
		models.AllowedUpdateEditedChannelPost,
		models.AllowedUpdateInlineQuery,
		models.AllowedUpdateChosenInlineResult,
		models.AllowedUpdateCallbackQuery,
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithAllowedUpdates(allowedUpdates),
	}

	thebot, err := bot.New(botToken, opts...)
	if err != nil { panic(err) }

	botMe, _ = thebot.GetMe(ctx)
	log.Printf("name[%s] [@%s] id[%d]", botMe.FirstName, botMe.Username, botMe.ID)

	log.Printf("starting %d\n", botMe.ID)
	log.Printf("logChat_ID: %v", logChat_ID)

	database, err = ReadYamlDB(db_path + metadataFileName)
	if err != nil {
		log.Println("read yaml db error: ", err)
	}

	go signalsHandler(SignalsChannel)

	// 初始化插件
	addPluginHandlers()

	// 检查是否设定了 webhookURL 环境变量
	if usingWebhook() { // Webhook
		setUpWebhook(ctx, thebot, &bot.SetWebhookParams{
			URL: webhookURL,
			AllowedUpdates: allowedUpdates,
		})
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
		if IsDebugMode {
			fmt.Printf("If in debug, visit https://api.telegram.org/bot%s/getWebhookInfo to check infos \n", botToken)
			fmt.Printf("If in debug, visit https://api.telegram.org/bot%s/setWebhook?url=https://api.trle5.xyz/webhook-trbot to reset webhook\n", botToken)
		}
		thebot.Start(ctx)
		<-ctx.Done() // 等待中断信号
	}
	// SignalsChannel.Database_save <- true // 退出之前保存一下数据库，似乎无效
	log.Println("manually stopped")
}
