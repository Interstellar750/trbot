package plugin_utils

import (
	"context"

	"trle5.xyz/trbot/utils"

	"github.com/rs/zerolog"
)

type DatabaseHandler struct {
	Name   string
	Loader func(ctx context.Context) error
	Saver  func(ctx context.Context) error
}

func AddDataBaseHandler(InlineHandlerPlugins ...DatabaseHandler) int {
	if AllPlugins.Databases == nil {
		AllPlugins.Databases = []DatabaseHandler{}
	}
	var pluginCount int
	for _, originPlugin := range InlineHandlerPlugins {
		AllPlugins.Databases = append(AllPlugins.Databases, originPlugin)
		pluginCount++
	}
	return pluginCount
}

func ReloadPluginsDatabase(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Logger()

	dbCount := len(AllPlugins.Databases)
	successCount := 0
	for _, plugin := range AllPlugins.Databases {
		if plugin.Loader == nil {
			logger.Warn().
				Str("pluginName", plugin.Name).
				Msg("Plugin has no loader function, skipping")
			continue
		}
		err := plugin.Loader(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Str("pluginName", plugin.Name).
				Msg("Plugin failed to reload database")
		} else {
			successCount++
		}
	}

	logger.Info().Msgf("Reloaded (%d/%d) plugins database", successCount, dbCount)
}

func SavePluginsDatabase(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str(utils.GetCurrentFuncName()).
		Logger()

	dbCount := len(AllPlugins.Databases)
	successCount := 0
	for _, plugin := range AllPlugins.Databases {
		if plugin.Saver == nil {
			logger.Warn().
				Str("pluginName", plugin.Name).
				Msg("Plugin has no saver function, skipping")
			successCount++
			continue
		}
		err := plugin.Saver(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Str("pluginName", plugin.Name).
				Msg("Plugin failed to reload database")
		} else {
			successCount++
		}
	}

	logger.Info().Msgf("Saved (%d/%d) plugins database", successCount, dbCount)
}
