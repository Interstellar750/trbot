package saved_message

import (
	"fmt"
	"strconv"
	"trbot/utils"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"
	"trbot/utils/inline_utils"
	"trbot/utils/meilisearch_utils"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
	"github.com/rs/zerolog"
)

func channelSaveMessageHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "channelSaveMessageHandler").
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	indexManager := meilisearchClient.Index(strconv.FormatInt(SavedMessageList.Channel.ChatID, 10))

	_, err := indexManager.AddDocuments(meilisearch_utils.BuildMessageData(opts.Ctx, opts.Thebot, opts.Message))
	if err != nil {
		logger.Error().
			Err(err).
			Int("messageID", opts.Message.ID).
			Str("content", "add message to index failed").
			Msg("failed to add message to index")
		return fmt.Errorf("failed to add message to index: %w", err)
	}

	return nil
}

func InlineChannelHandler(opts *handler_params.InlineQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "InlineChannelHandler").
		Dict(utils.GetUserDict(opts.InlineQuery.From)).
		Str("query", opts.InlineQuery.Query).
		Logger()
	var handlerErr flaterr.MultErr

	parsedQuery := inline_utils.ParseInlineFields(opts.Fields)

	indexManager := meilisearchClient.Index(SavedMessageList.Channel.ChatIDStr())
	datas, err := indexManager.Search(parsedQuery.KeywordQuery(), &meilisearch.SearchRequest{Limit: 50})
	if err != nil {
		logger.Error().
			Err(err).
			Msg("failed to get message")
		handlerErr.Addf("failed to get message: %w", err)
	} else {
		var (
			onlyText  []models.InlineQueryResult
			audio     []models.InlineQueryResult
			document  []models.InlineQueryResult
			gif       []models.InlineQueryResult
			photo     []models.InlineQueryResult
			sticker   []models.InlineQueryResult
			video     []models.InlineQueryResult
			videoNote []models.InlineQueryResult
			voice     []models.InlineQueryResult
		)
		for _, data := range datas.Hits {
			msgData, err := meilisearch_utils.MarshalMessageData(data)
			if err != nil {
				return err
			}
			switch msgData.MsgType {
			case message_utils.OnlyText:
				onlyText = append(onlyText, &models.InlineQueryResultArticle{
					ID:          msgData.MsgIDStr(),
					Title:       msgData.Text,
					Description: msgData.Desc,
					InputMessageContent: &models.InputTextMessageContent{
						MessageText:        msgData.Text,
						Entities:           msgData.Entities,
						LinkPreviewOptions: msgData.LinkPreviewOptions,
					},
				})
			case message_utils.Audio:
				audio = append(audio, &models.InlineQueryResultCachedAudio{
					ID:              msgData.MsgIDStr(),
					AudioFileID:     msgData.FileID,
					Caption:         msgData.Text,
					CaptionEntities: msgData.Entities,
				})
			case message_utils.Document:
				document = append(document, &models.InlineQueryResultCachedDocument{
					ID:              msgData.MsgIDStr(),
					DocumentFileID:  msgData.FileID,
					Title:           msgData.FileName,
					Caption:         msgData.Text,
					CaptionEntities: msgData.Entities,
					Description:     msgData.Desc,
				})
			case message_utils.Animation:
				gif = append(gif, &models.InlineQueryResultCachedMpeg4Gif{
					ID:              msgData.MsgIDStr(),
					Mpeg4FileID:     msgData.FileID,
					Title:           msgData.FileName,
					Caption:         msgData.Text,
					CaptionEntities: msgData.Entities,
				})
			case message_utils.Photo:
				photo = append(photo, &models.InlineQueryResultCachedPhoto{
					ID:                    msgData.MsgIDStr(),
					PhotoFileID:           msgData.FileID,
					Caption:               msgData.Text,
					CaptionEntities:       msgData.Entities,
					Description:           msgData.Desc,
					ShowCaptionAboveMedia: msgData.ShowCaptionAboveMedia,
				})
			case message_utils.Sticker:
				sticker = append(sticker, &models.InlineQueryResultCachedSticker{
					ID:            msgData.MsgIDStr(),
					StickerFileID: msgData.FileID,
				})
			case message_utils.Video:
				video = append(video, &models.InlineQueryResultCachedVideo{
					ID:              msgData.MsgIDStr(),
					VideoFileID:     msgData.FileID,
					Title:           msgData.FileName,
					Description:     msgData.Desc,
					Caption:         msgData.Text,
					CaptionEntities: msgData.Entities,
				})
			case message_utils.VideoNote:
				videoNote = append(videoNote, &models.InlineQueryResultCachedDocument{
					ID:              msgData.MsgIDStr(),
					DocumentFileID:  msgData.FileID,
					Title:           msgData.FileName,
					Description:     msgData.Desc,
					Caption:         msgData.Text,
					CaptionEntities: msgData.Entities,
				})
			case message_utils.Voice:
				voice = append(voice, &models.InlineQueryResultCachedVoice{
					ID:              msgData.MsgIDStr(),
					VoiceFileID:     msgData.FileID,
					Title:           msgData.FileTitle,
					Caption:         msgData.Text,
					CaptionEntities: msgData.Entities,
				})
			}
		}
		if len(datas.Hits) == 0 {
			onlyText = append(onlyText, &models.InlineQueryResultArticle{
				ID:                  "none",
				Title:               "没有符合关键词的内容",
				InputMessageContent: &models.InputTextMessageContent{MessageText: "用户在找不到想看的东西时无奈点击了提示信息..."},
			})
		}

		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.InlineQuery.ID,
			Results: inline_utils.ResultCategory(parsedQuery, map[string][]models.InlineQueryResult{
				"text":      onlyText,
				"audio":     audio,
				"document":  document,
				"gif":       gif,
				"photo":     photo,
				"sticker":   sticker,
				"video":     video,
				"videoNote": videoNote,
				"voice":     voice,
			}),
			IsPersonal: true,
			CacheTime:  0,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "channel saved message result").
				Msg(flaterr.AnswerInlineQuery.Str())
			handlerErr.Addt(flaterr.AnswerInlineQuery, "channel saved message result", err)
		}
	}
	return handlerErr.Flat()
}
