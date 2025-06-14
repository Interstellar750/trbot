package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"time"

	"trbot/database"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/internal_plugin"
	"trbot/utils/signals"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// create a logger and attached it into ctx
	var logger zerolog.Logger
	file, err := os.OpenFile(consts.LogFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err == nil {
		fileWriter := &zerolog.FilteredLevelWriter{
			Writer: zerolog.MultiLevelWriter(file),
			Level: zerolog.WarnLevel,
		}

		multWriter := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stdout}, fileWriter)
		logger = zerolog.New(multWriter).With().Timestamp().Logger()
		logger.Info().
			Str("logFile", consts.LogFilePath).
			Str("levelForLogFile", zerolog.WarnLevel.String()).
			Msg("Use mult log writer")
	} else {
		logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
		logger.Error().
			Err(err).
			Str("logFile", consts.LogFilePath).
			Msg("Failed to open log file, use console writer only")
	}

	ctx = logger.WithContext(ctx)

	// read configs
	if err := configs.InitBot(ctx); err != nil {
		logger.Fatal().Err(err).Msg("Failed to read bot configs")
	}

	// set log level from config
	zerolog.SetGlobalLevel(configs.BotConfig.LevelForZeroLog())
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	logger.Warn().
		Str("version", consts.Version).
		Str("commit", consts.Commit[:13]).
		Str("buildTime", consts.BuildTime).
		Str("changes", consts.Changes).
		Str("runtime", runtime.Version()).
		Str("logLevel", zerolog.GlobalLevel().String()).
		Msg("trbot")

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
		bot.WithAllowedUpdates(configs.BotConfig.AllowedUpdates),
	}

	thebot, err := bot.New(configs.BotConfig.BotToken, opts...)
	if err != nil { logger.Fatal().Err(err).Msg("Failed to initialize bot") }

	consts.BotMe, err = thebot.GetMe(ctx)
	if err != nil { logger.Fatal().Err(err).Msg("Failed to get bot info") }

	logger.Info().
		Dict(utils.GetUserDict(consts.BotMe)).
		Msg("Bot initialized")
	if configs.BotConfig.LogChatID != 0 {
		logger.Info().
			Int64("LogChatID", configs.BotConfig.LogChatID).
			Msg("Enabled log to chat")
	}

	database.InitAndListDatabases(ctx)

	// start handler custom signals
	go signals.SignalsHandler(ctx)

	// register plugin (plugin use `init()` first, then plugin use `InitPlugins` second, and internal last)
	internal_plugin.Register(ctx)

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
			logger.Fatal().
				Err(err).
				Msg("Webhook server failed")
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
