package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"trbot/database"
	"trbot/utils/consts"
	"trbot/utils/mess"
	"trbot/utils/plugin_init"
	"trbot/utils/signals"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)


func main() {
	consts.BotToken = mess.WhereIsBotToken()

	consts.IsDebugMode = os.Getenv("DEBUG") == "true"
	if consts.IsDebugMode {
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

	thebot, err := bot.New(consts.BotToken, opts...)
	if err != nil { panic(err) }

	consts.BotMe, _ = thebot.GetMe(ctx)
	log.Printf("name[%s] [@%s] id[%d]", consts.BotMe.FirstName, consts.BotMe.Username, consts.BotMe.ID)

	log.Printf("starting %d\n", consts.BotMe.ID)
	log.Printf("logChat_ID: %v", consts.LogChat_ID)

	database.ListDatabaseCount()

	go signals.SignalsHandler(ctx, consts.SignalsChannel)

	// 初始化插件
	plugin_init.RegisterPlugins()

	// 检查是否设定了 webhookURL 环境变量
	if mess.UsingWebhook() { // Webhook
		mess.SetUpWebhook(ctx, thebot, &bot.SetWebhookParams{
			URL: consts.WebhookURL,
			AllowedUpdates: allowedUpdates,
		})
		log.Println("Working at Webhook Mode")
		go thebot.StartWebhook(ctx)
		go func() {
			err := http.ListenAndServe(consts.WebhookPort, thebot.WebhookHandler())
			if err != nil { log.Panicln(err) }
		}()
	} else { // getUpdate, aka Long Polling
		// 保存并清理云端 Webhook URL，否则该模式会不生效 https://core.telegram.org/bots/api#getupdates
		mess.SaveAndCleanRemoteWebhookURL(ctx, thebot)
		log.Println("Working at Long Polling Mode")
		if consts.IsDebugMode {
			fmt.Printf("If in debug, visit https://api.telegram.org/bot%s/getWebhookInfo to check infos \n", consts.BotToken)
			fmt.Printf("If in debug, visit https://api.telegram.org/bot%s/setWebhook?url=https://api.trle5.xyz/webhook-trbot to reset webhook\n", consts.BotToken)
		}
		thebot.Start(ctx)
	}

	for {
		select {
		case <- consts.SignalsChannel.WorkDone:
			log.Println("manually stopped")
			return
		default:
			// log.Println("still waiting...") // 不在调式模式下，这个日志会非常频繁
			time.Sleep(1 * time.Second)
		}
	}

}
