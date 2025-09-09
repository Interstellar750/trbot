package sticker_download

import (
	"os"
	"trbot/plugins/sticker_download/config"
	"trbot/utils/handler_params"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func init() {
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "cachedsticker",
		MessageHandler: showCachedStickers,
	})
}

func showCachedStickers(opts *handler_params.Message) error {
	var button     [][]models.InlineKeyboardButton
	var tempButton []models.InlineKeyboardButton

	entries, err := os.ReadDir(config.CachedDir)
	if err != nil { return err }

	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != "-custom" {
			if len(tempButton) == 4 {
				button = append(button, tempButton)
				tempButton = []models.InlineKeyboardButton{}
			}
			tempButton = append(tempButton, models.InlineKeyboardButton{
				Text: entry.Name(),
				URL:  "https://t.me/addstickers/" + entry.Name(),
			})
		}
	}

	if len(tempButton) > 0 {
		button = append(button, tempButton)
	}

	_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID:          opts.Message.Chat.ID,
		Text:            "请选择要查看的贴纸包",
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: button },
		// MessageEffectID: "5104841245755180586",
	})
	return err
}
