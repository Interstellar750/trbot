package saved_message

import (
	"fmt"
	"trbot/utils"
	"trbot/utils/handler_params"
	"trbot/utils/meilisearch_utils"

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
		_, err := meilisearchClient.Index(SavedMessageList.ChannelIDStr()).AddDocuments(meilisearch_utils.BuildMessageData(opts.Ctx, opts.Thebot, opts.Message))
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
