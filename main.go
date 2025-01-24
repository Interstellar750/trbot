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
		bot.WithDefaultHandler(defaultHandler),
		// bot.WithMiddlewares(),
		// bot.WithMessageTextHandler("/select", bot.MatchTypeExact, commandHandler),
		// bot.WithMessageTextHandler("", bot.MatchTypeContains, catchAllHandler),
		// bot.WithCallbackQueryDataHandler("btn_", bot.MatchTypePrefix, callbackHandler),
		// bot.WithWebhookSecretToken("dqwdefgertghytyjyiuy"),
		bot.WithAllowedUpdates(bot.AllowedUpdates{
			"message",
			"edited_message",
			"channel_post",
			"edited_channel_post",
			"inline_query",
			"chosen_inline_result",
			"callback_query",
			"shipping_query",
			"pre_checkout_query",
			"poll",
			"poll_answer",
			"my_chat_member",
			"chat_member",
			"chat_join_request",
			"chat_boost",
			"removed_chat_boost",
			"message_reaction",
			"message_reaction_count",
			"business_connection",
			"business_message",
			"edited_business_message",
			"deleted_business_messages",
			"purchased_paid_media",
		}),
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

	go AdditionalDataReloader(ADR_reload, &AdditionalDataPath{
		Voice: voice_path,
		Udonese: udon_path,
	})

	go mainDatabaseHandler(DB_savenow)

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
		fmt.Printf("If in debug, visit https://api.telegram.org/bot%s/getWebhookInfo to check infos \n", botToken)
		fmt.Printf("If in debug, visit https://api.telegram.org/bot%s/setWebhook?url=https://api.trle5.xyz/webhook-trbot to reset webhook\n", botToken)
		thebot.Start(ctx)
		<-ctx.Done() // 等待中断信号
	}
	DB_savenow <- true // 退出之前保存一下数据库，似乎无效
	log.Println("manually stopped")
}
