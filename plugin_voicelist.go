package main

import (
	"fmt"
	"log"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func voiceListHandler(opts *subHandlerOpts) {
	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult

	keywordFields := InlineExtractKeywords(opts.fields)


	// 没有查询字符串或使用分页搜索符号，返回所有结果
	if len(keywordFields) == 0 {
		for _, voicePack := range AdditionalDatas.Voices {
			for _, voice := range voicePack.Voices {
				results = append(results, &models.InlineQueryResultVoice{
					ID:       voice.ID,
					Title:    voicePack.Name + ": " + voice.Title,
					Caption:  voice.Caption,
					VoiceURL: voice.VoiceURL,
				})
			}
		}
	} else {
		for _, voicePack := range AdditionalDatas.Voices {
			for _, voice := range voicePack.Voices {
				if InlineQueryMatchMultKeyword(keywordFields, []string{voicePack.Name, voice.Title, voice.Caption}) {
					results = append(results, &models.InlineQueryResultVoice{
						ID:       voice.ID,
						Title:    voicePack.Name + ": " + voice.Title,
						Caption:  voice.Caption,
						VoiceURL: voice.VoiceURL,
					})
				}
			}
		}
		if len(results) == 0 {
			results = append(results, &models.InlineQueryResultArticle{
				ID:    "none",
				Title: "没有符合关键词的内容",
				Description: fmt.Sprintf("没有找到包含 %s 的内容", keywordFields),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}

	if AdditionalDatas.VoiceErr != nil {
		log.Printf("Error when reading metadata files: %v", AdditionalDatas.VoiceErr)
		opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{
				&models.InlineQueryResultVoice{
					ID:       "none",
					Title:    "读取语音文件时发生错误，请联系机器人管理员",
					Caption:  "由于无法读取到语音文件，此处被替换为预设的 `♿otto: 我是说的道理~` ",
					VoiceURL: "https://otto-hzys.huazhiwan.xyz/static/ysddTokens/wssddl.mp3",
					ParseMode: models.ParseModeMarkdownV1,
				},
			},
			CacheTime: 0,
		})
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: logChat_ID,
			Text:   fmt.Sprintf("%s\nInline Mode: some user get error, %v", time.Now().Format(time.RFC3339), AdditionalDatas.VoiceErr),
		})
		return
	} else if AdditionalDatas.Voices == nil {
		log.Printf("No voices file in voices_path: %s", voice_path)
		opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.update.InlineQuery.ID,
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
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID:    logChat_ID,
			Text:      fmt.Sprintf("%s\nInline Mode: some user can't load voices", time.Now().Format(time.RFC3339)),
		})
		return
	}

	_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: opts.update.InlineQuery.ID,
		Results:       InlineResultPagination(opts.fields, results),
		// Button: inlineButton,
	})
	if err != nil {
		log.Printf("Error sending inline query response: %v", err)
		return
	}
}

var voiceListInlineHandler = Plugin_InlineManual{
	command: "voice",
	handler: voiceListHandler,
	description: "一些语音列表",
}
