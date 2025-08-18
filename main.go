package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"time"

	"trbot/database"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/internal_plugin"
	"trbot/utils/signals"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack // set stack trace func
	logger := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
	ctx     = logger.WithContext(ctx) // attach logger into ctx

	// read bot configs
	err := configs.InitBot(ctx)
	if err != nil { logger.Fatal().Err(err).Msg("Failed to read bot configs") }

	// writer log to a file or only display on console
	if configs.IsUseMultiLogWriter(&logger) { ctx = logger.WithContext(ctx) } // re-attach logger into ctx
	configs.CheckConfig(ctx) // check and auto fill some config
	configs.ShowConst(ctx)   // show build info

	thebot, err := bot.New(configs.BotConfig.BotToken, []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithAllowedUpdates(configs.BotConfig.AllowedUpdates),
		bot.WithErrorsHandler(func(err error){ logger.Error().Err(err).Msg("go-telegram/bot") }),
	}...)
	if err != nil { logger.Fatal().Err(err).Msg("Failed to initialize bot") }

	consts.BotMe, err = thebot.GetMe(ctx)
	if err != nil { logger.Fatal().Err(err).Msg("Failed to get bot info") }

	logger.Info().
		Dict(utils.GetUserDict(consts.BotMe)).
		Msg("Bot initialized")

	database.InitAndListDatabases(ctx)

	// set log level after bot initialized
	zerolog.SetGlobalLevel(configs.BotConfig.LevelForZeroLog(false))

	// start handler custom signals
	go signals.SignalsHandler(ctx)

	// register plugin (plugin use `init()` first, then plugin use `InitPlugins` second, and internal is the last)
	internal_plugin.Register(ctx)

	// Select mode by Webhook config
	if configs.IsUsingWebhook(ctx) /* Webhook */ {
		if configs.SetUpWebhook(ctx, thebot, &bot.SetWebhookParams{
			URL: configs.BotConfig.WebhookURL,
			AllowedUpdates: configs.BotConfig.AllowedUpdates,
		}) {
			logger.Info().
				Str("listenAddress", consts.WebhookListenPort).
				Msg("Working at Webhook Mode")
			go thebot.StartWebhook(ctx)
			err := http.ListenAndServe(consts.WebhookListenPort, thebot.WebhookHandler())
			if err != nil {
				logger.Fatal().
					Err(err).
					Msg("Webhook server failed")
			}
		} else {
			logger.Fatal().
				Str("webhookURL", configs.BotConfig.WebhookURL).
				Msg("Failed to setup Webhook")
		}
	} else /* getUpdate, aka Long Polling */ {
		// remove Webhook URL befor using getUpdate https://core.telegram.org/bots/api#getupdates
		if configs.CleanRemoteWebhookURL(ctx, thebot) {
			logger.Info().Msg("Working at Long Polling Mode")
			// logger.Debug().Msgf("visit https://api.telegram.org/bot%s/getWebhookInfo to check infos", configs.BotConfig.BotToken)
			thebot.Start(ctx)
		} else {
			logger.Fatal().Msg("Failed to remove Webhook URL")
		}
	}

	// A loop wait for getUpdate mode, this program will exit in `utils\signals\signals.go`.
	// This loop will only run when the exit signal is received in getUpdate mode.
	// Webhook mode won't reach here, http.ListenAndServe() will keep program running until exit.
	// They use the same code to exit, this loop is to give some time to save the database when receive exit signal.
	for {
		time.Sleep(5 * time.Second)
		logger.Info().Msg("still waiting...")
	}
}
