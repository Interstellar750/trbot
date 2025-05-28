package configs

import (
	"context"
	"os"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
)

// 通过是否设定环境变量和配置文件中的 webhook URL 来决定是否使用 webhook 模式
func IsUsingWebhook(ctx context.Context) bool {
	logger := zerolog.Ctx(ctx)
	// 通过 godotenv 库读取 .env 文件后再尝试读取
	godotenv.Load()
	webhookURL := os.Getenv("WEBHOOK_URL")
	if webhookURL != "" {
		BotConfig.WebhookURL = webhookURL
		logger.Info().
			Str("Webhook URL", BotConfig.WebhookURL).
			Msg("Get Webhook URL from environment or .env file")
		return true
	}

	// 从 yaml 配置文件中读取
	if BotConfig.WebhookURL != "" {
		logger.Info().
			Str("Webhook URL", BotConfig.WebhookURL).
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
			Msg("Get WebHook info error")
	}
	if webHookInfo != nil && webHookInfo.URL != params.URL {
		if webHookInfo.URL == "" {
			logger.Info().
				Msg("Webhook not set, setting it now...")
		} else {
			logger.Warn().
				Str("Remote URL", webHookInfo.URL).
				Str("Local URL", params.URL).
				Msg("The remote webhook URL conflicts with the local one, saving and overwriting the remote URL")
		}
		success, err := thebot.SetWebhook(ctx, params)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Set WebHook URL failed")
			return false
		}
		if success {
			logger.Info().
				Str("Webhook URL", params.URL).
				Msg("Set Webhook URL success")
			return true
		}
	} else {
		logger.Info().
			Str("Webhook URL", params.URL).
			Msg("Webhook URL is already set")
		return true
	}

	return false
}

func SaveAndCleanRemoteWebhookURL(ctx context.Context, thebot *bot.Bot) *models.WebhookInfo {
	logger := zerolog.Ctx(ctx)
	webHookInfo, err := thebot.GetWebhookInfo(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Get WebHook info error")
	}
	if webHookInfo != nil && webHookInfo.URL != "" {
		logger.Warn().
			Str("Remote URL", webHookInfo.URL).
			Msg("There is a webhook URL remotely, saving and clearing it to use the getUpdate mode")
		ok, err := thebot.DeleteWebhook(ctx, &bot.DeleteWebhookParams{
			DropPendingUpdates: false,
		})
		if !ok {
			logger.Error().
				Err(err).
				Msg("Delete Webhook URL failed")
		}
		return webHookInfo
	}
	
	return nil
}
