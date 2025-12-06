package saved_message

import (
	"context"
	"errors"

	"trle5.xyz/gopkg/trbot/plugins/saved_message/channel"
	"trle5.xyz/gopkg/trbot/plugins/saved_message/common"
	"trle5.xyz/gopkg/trbot/plugins/saved_message/message_index"
	"trle5.xyz/gopkg/trbot/plugins/saved_message/personal"
	"trle5.xyz/gopkg/trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/meilisearch/meilisearch-go"
)

func Init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "Saved Message",
		Func: func(ctx context.Context, thebot *bot.Bot) error {
			err := common.ReadSavedMessageList(ctx)
			if err != nil {
				return err
			}
			if common.SavedMessageList.MeiliURL == "" {
				return errors.New("the Meilisearch URL is not set")
			} else {
				common.MeilisearchClient = meilisearch.New(common.SavedMessageList.MeiliURL, meilisearch.WithAPIKey(common.SavedMessageList.MeiliAPI))
				if common.SavedMessageList.ChannelID != 0 {
					err = channel.InitChannelPart(ctx)
					if err != nil {
						return err
					}
				}

				if common.SavedMessageList.AllowUserSave {
					err = personal.InitUserPart(ctx)
					if err != nil {
						return err
					}
				}

				message_index.Init(&common.MeilisearchClient)
			}
			return nil
		},
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Saved Message",
		Saver:  common.SaveSavedMessageList,
		Loader: common.ReadSavedMessageList,
	})



}
