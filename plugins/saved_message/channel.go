package saved_message

import (
	"fmt"
	"trbot/utils"
	"trbot/utils/handler_params"
	"trbot/utils/meilisearch_utils"
	"trbot/utils/origin_info"
	"unicode/utf8"

	"github.com/go-telegram/bot/models"
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

	if meilisearchClient == nil {
		logger.Warn().
			Int("messageID", opts.Message.ID).
			Msg("Meilisearch client is not initialized, skipping indexing")
	} else {
		msgData := meilisearch_utils.BuildMessageData(opts.Ctx, opts.Thebot, opts.Message)

		var messageLength int
		var pendingEntitites []models.MessageEntity
		var needChangeEntitites bool = true

		if opts.Message.Caption != "" {
			messageLength = utf8.RuneCountInString(opts.Message.Caption)
			pendingEntitites = opts.Message.CaptionEntities
		} else if opts.Message.Text != "" {
			messageLength = utf8.RuneCountInString(opts.Message.Text)
			pendingEntitites = opts.Message.Entities
		} else {
			needChangeEntitites = false
		}

		// 若字符长度大于设定的阈值，添加折叠样式引用再保存
		if needChangeEntitites && messageLength > textExpandableLength && len(pendingEntitites) == 0 {
			msgData.Entities = []models.MessageEntity{{
				Type:   models.MessageEntityTypeExpandableBlockquote,
				Offset: 0,
				Length: messageLength,
			}}
		}

		msgData.OriginInfo = origin_info.GetOriginInfo(opts.Message)

		_, err := meilisearchClient.Index(SavedMessageList.ChannelIDStr()).AddDocuments(msgData)
		if err != nil {
			logger.Error().
				Err(err).
				Int("messageID", opts.Message.ID).
				Str("content", "add message to index failed").
				Msg("failed to add message to index")
			return fmt.Errorf("failed to add message to index: %w", err)
		}
	}

	return nil
}

// func channelChangeKeywordHandler(opts *handler_params.Message) error {
// 	var handlerErr flaterr.MultErr

// 	return handlerErr.Flat()
// }
