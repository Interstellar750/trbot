package main

import (
	"context"
	"log"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)


// func defaulthandler(ctx context.Context, rbot *bot.Bot, update *models.Update) {
// 	if update.Message != nil {
// 		rbot.SendMessage(ctx, &bot.SendMessageParams{
// 			ChatID: update.Message.Chat.ID,
// 			Text:   update.Message.Text,
// 		})
// 	}

// 	if update.Message.Sticker != nil {
// 		rbot.SendSticker(ctx, &bot.SendStickerParams{
// 			ChatID:  update.Message.Chat.ID,
// 			Sticker: &models.InputFileString{update.Message.Sticker.FileID},
// 		})
// 	}
// }


func inlinehandler(ctx context.Context, rbot *bot.Bot, update *models.Update) {
	if update.InlineQuery == nil {
		return
	}

	metadataList, err := readMetadataFromDir("./voices")
	if err != nil {
		log.Fatalf("Error reading metadata files: %v", err)
	}

	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult
	for _, metadata := range metadataList {
		for _, voice := range metadata.Voices {
			result := &models.InlineQueryResultVoice{
				ID:       voice.ID,
				Title:    metadata.Name + ":" + voice.Title,
				Caption:  voice.Caption,
				VoiceURL: voice.VoiceURL,
			}
			results = append(results, result)
		}
	}

	rbot.AnswerInlineQuery(ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: update.InlineQuery.ID,
		Results:       results,
		CacheTime:     0,
	})

	if err != nil {
		log.Printf("Error sending inline query response: %v", err)
	}
}
