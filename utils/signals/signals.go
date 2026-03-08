package signals

import (
	"context"
	"os"
	"time"

	"trle5.xyz/trbot/database"
	"trle5.xyz/trbot/utils/plugin_utils"

	"github.com/rs/zerolog"
)

type SignalChannel struct {
	Database_save   chan bool
	PluginDB_save   chan bool
	PluginDB_reload chan bool
}

var SIGNALS = SignalChannel{
	Database_save:   make(chan bool),
	PluginDB_save:   make(chan bool),
	PluginDB_reload: make(chan bool),
}

func SignalsHandler(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	var saveDatabaseRetryCount int = 0
	var saveDatabaseRetryMax   int = 10

	for {
		select {
		case <-ctx.Done():
			if saveDatabaseRetryCount == 0 { logger.Warn().Msg("Cancle signal received") }
			plugin_utils.SavePluginsDatabase(ctx)
			err := database.SaveDatabase(ctx)
			if err != nil {
				saveDatabaseRetryCount++
				logger.Error().
					Err(err).
					Int("retryCount", saveDatabaseRetryCount).
					Int("maxRetry", saveDatabaseRetryMax).
					Msg("Failed to save database, retrying...")
				time.Sleep(2 * time.Second)
				if saveDatabaseRetryCount >= saveDatabaseRetryMax {
					logger.Fatal().
						Err(err).
						Msg("Failed to save database too many times, exiting")
				}
				continue
			}
			logger.Info().Msg("Database saved")
			time.Sleep(1 * time.Second)
			logger.Warn().Msg("manually stopped")
			os.Exit(0)
		case <-SIGNALS.Database_save:
			database.SaveDatabase(ctx)
		case <-SIGNALS.PluginDB_reload:
			plugin_utils.ReloadPluginsDatabase(ctx)
		case <-SIGNALS.PluginDB_save:
			plugin_utils.SavePluginsDatabase(ctx)
		}
	}
}
