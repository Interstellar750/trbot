package main

import (
	"context"
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
		bot.WithMessageTextHandler("/select", bot.MatchTypeExact, commandHandler),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, startHandler),
		bot.WithMessageTextHandler("/forwardonly", bot.MatchTypePrefix, addToWriteListHandler),
		bot.WithMessageTextHandler("", bot.MatchTypeContains, defaulthandler),
		bot.WithCallbackQueryDataHandler("btn_", bot.MatchTypePrefix, callbackHandler),
	}

	thebot, err := bot.New(botToken, opts...)
	if err != nil { panic(err) }

	// me, _ := thebot.GetMe()
	// log.Println(me.)

	log.Printf("starting %s\n", showBotID())
	log.Printf("logChat_ID: %v", logChat_ID)

	err = fwdonly_ReadMetadata()
	if err != nil {
		log.Println(err)
	}

	if usingWebhook() { // Webhook
		setUpWebhook(ctx, thebot, webhookURL)
		log.Println("Working at Webhook Mode")
		go thebot.StartWebhook(ctx)
		err := http.ListenAndServe(webhookPort, thebot.WebhookHandler())
		if err != nil { log.Panicln(err) }
	} else { // getUpdate, eg Long Polling
		// 保存并清理云端 Webhook URL，否则该模式会不生效 https://core.telegram.org/bots/api#getupdates
		saveAndCleanRemoteWebhookURL(ctx, thebot)
		log.Println("Working at Long Polling Mode")
		thebot.Start(ctx)
	}
}
