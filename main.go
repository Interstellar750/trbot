package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
	"github.com/joho/godotenv"
)


func main() {
	err := godotenv.Load()
	if err != nil { log.Printf("Can't loading .env file") }
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(inlinehandler),
		// bot.WithMiddlewares(),
		bot.WithMessageTextHandler("/select", bot.MatchTypeExact, commandHandler),
		bot.WithMessageTextHandler("/start", bot.MatchTypeExact, startHandler),
		bot.WithMessageTextHandler("", bot.MatchTypeContains, defaulthandler),
		bot.WithCallbackQueryDataHandler("btn_", bot.MatchTypePrefix, callbackHandler),
		// bot.
	}

	thebot, err := bot.New(os.Getenv("TELEGRAM_BOT_TOKEN"), opts...)
	if err != nil { panic(err) }

	// thebot.RegisterHandler()
	// thebot.UnregisterHandler()

	fmt.Printf("running %.10s\n", os.Getenv("TELEGRAM_BOT_TOKEN"))

	thebot.Start(ctx)
}
