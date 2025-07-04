package configs

import (
	"context"
	"os"

	"github.com/go-telegram/bot"
	"github.com/rs/zerolog"
)

// 通过是否设定环境变量和配置文件中的 Webhook URL 来决定是否使用 Webhook 模式
func IsUsingWebhook(ctx context.Context) bool {
	logger := zerolog.Ctx(ctx)
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL != "" {
		BotConfig.WebhookURL = webhookURL
		logger.Info().
			Str("WebhookURL", BotConfig.WebhookURL).
			Msg("Get Webhook URL from environment")
		return true
	}

	// 从 yaml 配置文件中读取
	if BotConfig.WebhookURL != "" {
		logger.Info().
			Str("WebhookURL", BotConfig.WebhookURL).
			Msg("Get Webhook URL from config file")
		return true
	}

	logger.Info().
		Msg("No Webhook URL in environment and .env file, using getUpdate mode")
	return false
}

func SetUpWebhook(ctx context.Context, thebot *bot.Bot, params *bot.SetWebhookParams) bool {
	logger := zerolog.Ctx(ctx)
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Get Webhook info error")
		return false
	} else {
		if webHookInfo != nil && webHookInfo.URL != params.URL {
			if webHookInfo.URL == "" {
				logger.Info().
					Msg("Webhook not set, setting it now...")
			} else {
				logger.Warn().
					Str("remoteURL", webHookInfo.URL).
					Str("localURL", params.URL).
					Msg("The remote Webhook URL conflicts with the local one, overwriting the remote URL")
			}
			success, err := thebot.SetWebhook(ctx, params)
			if !success {
				logger.Error().
					Err(err).
					Str("localURL", params.URL).
					Msg("Set Webhook URL failed")
				return false
			} else {
				logger.Info().
					Str("remoteURL", params.URL).
					Msg("Set Webhook URL success")
			}
		} else {
			logger.Info().
				Str("remoteURL", params.URL).
				Msg("Webhook URL is already set")
		}
	}

	return true
}

func CleanRemoteWebhookURL(ctx context.Context, thebot *bot.Bot) bool {
	logger := zerolog.Ctx(ctx)
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get Webhook info")
		return false
	} else {
		if webHookInfo != nil && webHookInfo.URL != "" {
			logger.Warn().
				Str("remoteURL", webHookInfo.URL).
				Msg("There is a Webhook URL remotely, clearing it to use the getUpdate mode")
			ok, err := thebot.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
				DropPendingUpdates: false,
			})
			if !ok {
				logger.Error().
					Err(err).
					Str("remoteURL", webHookInfo.URL).
					Msg("Failed to delete Webhook URL")
				return false
			} else {
				logger.Info().
					Str("oldRemoteURL", webHookInfo.URL).
					Msg("Deleted Webhook URL")
			}
		}
	}

	return true
}
