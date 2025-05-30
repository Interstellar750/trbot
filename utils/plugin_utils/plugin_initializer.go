package plugin_utils

import (
	"context"

	"github.com/rs/zerolog"
)

type Initializer struct {
	Name string
	Func func(ctx context.Context) error
}

func AddInitializer(initializers ...Initializer) int {
	if AllPlugins.Initializer == nil {
		AllPlugins.Initializer = []Initializer{}
	}
	var pluginCount int
	for _, initializer := range initializers {
		AllPlugins.Initializer = append(AllPlugins.Initializer, initializer)
		pluginCount++
	}
	return pluginCount
}

func RunPluginInitializers(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("funcName", "RunPluginInitializers").
		Logger()

	count := len(AllPlugins.Initializer)
	successCount := 0

	for _, initializer := range AllPlugins.Initializer {
		if initializer.Func == nil {
			logger.Warn().
				Str("pluginName", initializer.Name).
				Msg("Plugin has no initialize function, skipping")
			continue
		}
		err := initializer.Func(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Str("pluginName", initializer.Name).
				Msg("Failed to initialize plugin, skipping")
			continue
		} else {
			logger.Info().
				Str("pluginName", initializer.Name).
				Msg("Plugin initialized success")
			successCount++
		}
		
	}
	
	logger.Info().Msgf("Run (%d/%d) initializer success", successCount, count)
}
