package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var currentOptions = []bool{false, false, false}

func startHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	thebot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      fmt.Sprintf("Hello, *%s %s*\n\nThis robot doesn't currently support chat mode, please use [inline mode](https://telegram.org/blog/inline-bots?setln=en) for interactive operations.", update.Message.From.FirstName, update.Message.From.LastName),
		ParseMode: models.ParseModeMarkdownV1,
	})
}


func defaulthandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	// if update.Message != nil {
	// 	thebot.SendMessage(ctx, &bot.SendMessageParams{
	// 		ChatID: update.Message.Chat.ID,
	// 		Text:   update.Message.Text,
	// 	})
	// }

	// if update.Message.Sticker != nil {
	// 	thebot.SendSticker(ctx, &bot.SendStickerParams{
	// 		ChatID:  update.Message.Chat.ID,
	// 		Sticker: &models.InputFileString{update.Message.Sticker.FileID},
	// 	})
	// }


	thebot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "No operations available",
		ParseMode: models.ParseModeMarkdown,
	})
	
	time.Sleep(time.Second * 5)

	thebot.DeleteMessages(ctx, &bot.DeleteMessagesParams{
		ChatID:     update.Message.Chat.ID,
		MessageIDs: []int{
			update.Message.ID,
			update.Message.ID + 1,
		},
	})
}


func inlinehandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	if update.InlineQuery == nil { return }

	metadataList, err := readMetadataFromDir("./voices")
	if err != nil {
		log.Printf("Error reading metadata files: %v", err)
		thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{
				&models.InlineQueryResultVoice{
					ID:       "none",
					Title:    "无法读取到语音文件，请联系机器人管理员",
					Caption:  "由于无法读取到语音文件，此处被替换为预设的 `♿otto: 我是说的道理~` ",
					VoiceURL: "https://otto-hzys.huazhiwan.xyz/static/ysddTokens/wssddl.mp3",
					ParseMode: models.ParseModeMarkdownV1,
				},
			},
			CacheTime: 0,
		})
		return
	}

	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult
	for _, metadata := range metadataList {
		for _, voice := range metadata.Voices {
			result := &models.InlineQueryResultVoice{
				ID:       voice.ID,
				Title:    metadata.Name + ": " + voice.Title,
				Caption:  voice.Caption,
				VoiceURL: voice.VoiceURL,
			}
			results = append(results, result)
		}
	}

	_, err = thebot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: update.InlineQuery.ID,
		Results:       results,
		CacheTime:     0,
	})
	if err != nil {
		log.Printf("Error sending inline query response: %v", err)
		return
	}
}

func callbackHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	// answering callback query first to let Telegram know that we received the callback query,
	// and we're handling it. Otherwise, Telegram might retry sending the update repetitively
	// as it thinks the callback query doesn't reach to our application. learn more by
	// reading the footnote of the https://core.telegram.org/bots/api#callbackquery type.
	thebot.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: update.CallbackQuery.ID,
		ShowAlert:       false,
	})

	switch update.CallbackQuery.Data {
	case "btn_opt1":
		currentOptions[0] = !currentOptions[0]
	case "btn_opt2":
		currentOptions[1] = !currentOptions[1]
	case "btn_opt3":
		currentOptions[2] = !currentOptions[2]
	case "btn_select":
		thebot.EditMessageText(ctx, &bot.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: update.CallbackQuery.Message.Message.ID,
			Text:   fmt.Sprintf("Selected options: %v", currentOptions),
		})
		// b.SendMessage(ctx, &bot.SendMessageParams{
		// 	ChatID: update.CallbackQuery.Message.Message.Chat.ID,
		// })
		return
	}

	thebot.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
		ChatID:      update.CallbackQuery.Message.Message.Chat.ID,
		MessageID:   update.CallbackQuery.Message.Message.ID,
		ReplyMarkup: buildKeyboard(),
	})
}

func buildKeyboard() models.ReplyMarkup {
	kb := &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: buttonText("Option 1", currentOptions[0]), CallbackData: "btn_opt1"},
				{Text: buttonText("Option 2", currentOptions[1]), CallbackData: "btn_opt2"},
				{Text: buttonText("Option 3", currentOptions[2]), CallbackData: "btn_opt3"},
			},
			{
				{Text: "Select", CallbackData: "btn_select"},
			},
		},
	}

	return kb
}

func buttonText(text string, opt bool) string {
	if opt { return "✅ " + text }
	return "❌ " + text
}

func commandHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      update.Message.Chat.ID,
		Text:        "Select multiple options",
		ReplyMarkup: buildKeyboard(),
	})
}
