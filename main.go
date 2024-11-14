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

var botToken string

func main() {
	botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		log.Println("No bot token in environment, trying to read it from the .env file")
		if godotenv.Load() != nil { log.Fatalln("Can't loading .env file") }
		botToken = os.Getenv("TELEGRAM_BOT_TOKEN")
		if botToken == "" {
			log.Fatalln("No bot token in .env file, try create a bot from @botfather https://core.telegram.org/bots/tutorial#obtain-your-bot-token")
		}
		log.Printf("Get token from .env file: %.10s", botToken)
	} else {
		log.Printf("Get token from environment: %.10s", botToken)
	}

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

	fmt.Printf("running %.10s\n", botToken)

	data, err := readMetadataFile("./forwardonly/metadata.yaml");
	if data != nil {
		forwardonlylist = data
	}
	if err != nil {
		log.Println(err)
		err = saveMetadata("./forwardonly/", "metadata.yaml", &Metadata{})
		if err != nil {
			log.Println(err)
		}
		// os.Mkdir("forwardonly", 7777)

	}
	thebot.Start(ctx)
}
