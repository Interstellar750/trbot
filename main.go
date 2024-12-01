package main

import (
	"context"
	"log"
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

	log.Printf("running %s\n", showBotID())
	log.Printf("logChat_ID: %v", logChat_ID)

	err = fwdonly_ReadMetadata()
	if err != nil {
		log.Println(err)
	}

	thebot.Start(ctx)
	// go thebot.StartWebhook(ctx)
	// http.ListenAndServe(":2000", thebot.WebhookHandler())
}
