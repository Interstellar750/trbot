package signals

import (
	"context"
	"os"
	"time"
	"trbot/database"
	"trbot/database/yaml_db"
	"trbot/utils/mess"
	"trbot/utils/plugin_utils"

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
	every10Min := time.NewTicker(10 * time.Minute)
	defer every10Min.Stop()
	var saveDatabaseRetryCount int = 0
	var saveDatabaseRetryMax   int = 10

	for {
		select {
		case <-every10Min.C: // 每次 Ticker 触发时执行任务
			yaml_db.AutoSaveDatabaseHandler()
		case <-ctx.Done():
			if saveDatabaseRetryCount == 0 { logger.Warn().Msg("Cancle signal received") }
			err := database.SaveDatabase(ctx)
			if err != nil {
				saveDatabaseRetryCount++
				logger.Error().
					Err(err).
					Int("retryCount", saveDatabaseRetryCount).
					Int("maxRetry", saveDatabaseRetryMax).
					Msg("Save database failed")
				time.Sleep(2 * time.Second)
				if saveDatabaseRetryCount >= saveDatabaseRetryMax {
					logger.Error().Msg("Save database failed too many times, exiting")
					os.Exit(1)
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
			plugin_utils.ReloadPluginsDatabase()
			logger.Info().Msg("Plugin Database reloaded")
		case <-SIGNALS.PluginDB_save:
			mess.PrintLogAndSave(plugin_utils.SavePluginsDatabase())
		}
	}
}
