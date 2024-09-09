package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/go-telegram/bot"
)


func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(inlinehandler),
		// bot.WithMiddlewares(),
	}

	rbot, err := bot.New(os.Getenv("TELEGRAM_BOT_TOKEN"), opts...)
	if err != nil {
		panic(err)
	}

	// rbot.RegisterHandler(bot.HandlerTypeMessageText, "/inline", bot.MatchTypeExact, handler)

	fmt.Printf("running %.10s\n", os.Getenv("TELEGRAM_BOT_TOKEN"))

	rbot.Start(ctx)
}
