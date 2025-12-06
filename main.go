package main

import (
	"net/http"
	"time"

	"trle5.xyz/gopkg/trbot/database"
	"trle5.xyz/gopkg/trbot/utils"
	"trle5.xyz/gopkg/trbot/utils/configs"
	"trle5.xyz/gopkg/trbot/utils/internal_plugin"
	"trle5.xyz/gopkg/trbot/utils/signals"
	"trle5.xyz/gopkg/trbot/utils/task"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog"
)

func main() {
	ctx, cancel, logger := configs.InitBot()
	defer cancel()

	thebot, err := bot.New(configs.BotConfig.BotToken, []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithAllowedUpdates(configs.BotConfig.AllowedUpdates),
		bot.WithErrorsHandler(func(err error){ logger.Error().Err(err).Msg("go-telegram/bot") }),
		// bot.WithDebugHandler(func(format string, args ...any){ logger.Debug().Msgf(format, args...) }),
		bot.WithSkipGetMe(),
		// bot.WithDebug(),
	}...)
	if err != nil { logger.Fatal().Err(err).Msg("Failed to create bot instance") }

	configs.BotMe, err = thebot.GetMe(ctx)
	if err != nil { logger.Fatal().Err(err).Msg("Failed to get bot info") }

	logger.Info().
		Dict(utils.GetUserDict(configs.BotMe)).
		Msg("Bot initialized")

	// init task handler
	task.InitTaskHandler(ctx)

	database.InitDatabase(ctx)

	// set log level after bot initialized
	zerolog.SetGlobalLevel(configs.BotConfig.LevelForZeroLog(false))

	// start handler custom signals
	go signals.SignalsHandler(ctx)

	// register plugin (plugin use `init()` first, internal is the last)
	internal_plugin.Register(ctx, thebot)

	// Select mode by Webhook config
	if configs.BotConfig.WebhookURL != "" /* Webhook */ {
		if configs.SetUpWebhook(ctx, thebot, &bot.SetWebhookParams{
			URL:            configs.BotConfig.WebhookURL,
			AllowedUpdates: configs.BotConfig.AllowedUpdates,
		}) {
			logger.Info().
				Str("listenAddress", configs.BotConfig.WebhookListenAddress).
				Msg("Working at Webhook Mode")
			go thebot.StartWebhook(ctx)
			err := http.ListenAndServe(configs.BotConfig.WebhookListenAddress, thebot.WebhookHandler())
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
		// remove Webhook URL before using getUpdate https://core.telegram.org/bots/api#getupdates
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
