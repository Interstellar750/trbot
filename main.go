package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"trbot/database"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/internal_plugin"
	"trbot/utils/signals"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// create a logger and attached it into ctx
	logger := log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	ctx = logger.WithContext(ctx)

	// read configs
	if err := configs.InitBot(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to read bot configs")
	}

	// set log level from config
	zerolog.SetGlobalLevel(configs.BotConfig.LevelForZeroLog())
	if zerolog.GlobalLevel() == zerolog.DebugLevel {
		consts.IsDebugMode = true
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithAllowedUpdates(configs.BotConfig.AllowedUpdates),
	}

	thebot, err := bot.New(configs.BotConfig.BotToken, opts...)
	if err != nil { logger.Panic().Err(err).Msg("Failed to initialize bot") }

	consts.BotMe, err = thebot.GetMe(ctx)
	if err != nil { logger.Panic().Err(err).Msg("Failed to get bot info") }
	logger.Info().
		Str("name", consts.BotMe.FirstName).
		Str("username", consts.BotMe.Username).
		Int64("ID", consts.BotMe.ID).
		Msg("bot initialized")
	if configs.BotConfig.LogChatID != 0 {
		logger.Info().
			Int64("LogChatID", configs.BotConfig.LogChatID).
			Msg("Enabled log to chat")
	}

	database.InitAndListDatabases()

	// start handler custom signals
	go signals.SignalsHandler(ctx)

	// register plugin (internal first, then external)
	internal_plugin.Register()

	// Select mode by Webhook config
	if configs.IsUsingWebhook(ctx) { // Webhook
		configs.SetUpWebhook(ctx, thebot, &bot.SetWebhookParams{
			URL: configs.BotConfig.WebhookURL,
			AllowedUpdates: configs.BotConfig.AllowedUpdates,
		})
		logger.Info().
			Str("listenAddress", consts.WebhookListenPort).
			Msg("Working at Webhook Mode")
		go thebot.StartWebhook(ctx)
		err := http.ListenAndServe(consts.WebhookListenPort, thebot.WebhookHandler())
		if err != nil {
			logger.Panic().
				Err(err).
				Msg("webhook server failed")
		}
	} else { // getUpdate, aka Long Polling
		// save and clean remove Webhook URL befor using getUpdate https://core.telegram.org/bots/api#getupdates
		configs.SaveAndCleanRemoteWebhookURL(ctx, thebot)
		logger.Info().
			Msg("Working at Long Polling Mode")
		logger.Debug().
			Msgf("visit https://api.telegram.org/bot%s/getWebhookInfo to check infos", configs.BotConfig.BotToken)
		thebot.Start(ctx)
	}

	// a loop wait for getUpdate mode, this program will exit in `utils\signals\signals.go`.
	// This loop will only run when the exit signal is received in getUpdate mode.
	// Webhook won't reach here, http.ListenAndServe() will keep program running till exit.
	// They use the same code to exit, this loop is to give some time to save the database when receive exit signal.
	for {
		logger.Info().Msg("still waiting...")
		time.Sleep(2 * time.Second)
	}
}
