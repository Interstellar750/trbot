package saved_message

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/message_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var SavedMessageSet map[int64]SavedMessage
var SavedMessageErr error

var SavedMessagePath string = filepath.Join(consts.YAMLDataBaseDir, "savedmessage/", consts.YAMLFileName)

var textExpandableLength int = 150

type SavedMessage struct {
	DiscussionID       int64 `yaml:"DiscussionID,omitempty"`
	// IsChannelMode      bool  `yaml:"IsChannelMode,omitempty"`
	ByUserID           int64 `yaml:"ByUserID,omitempty"`
	Count              int   `yaml:"Count"`                // 当前存储的数量
	SavedTimes         int   `yaml:"SavedTimes,omitempty"` // 一共存过多少个
	Limit              int   `yaml:"Limit,omitempty"`
	AgreePrivacyPolicy bool  `yaml:"AgreePrivacyPolicy,omitempty"`

	Item SavedMessageItems `yaml:"Item,omitempty"`
}

func SaveSavedMessageList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "SaveSavedMessageList").
		Logger()

	err := yaml.SaveYAML(SavedMessagePath, &SavedMessageSet)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", SavedMessagePath).
			Msg("Failed to save savedmessage list")
		SavedMessageErr = fmt.Errorf("failed to save savedmessage list: %w", err)
	} else {
		SavedMessageErr = nil
	}

	return SavedMessageErr
}

func ReadSavedMessageList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Saved Message").
		Str("funcName", "ReadSavedMessageList").
		Logger()

	err := yaml.LoadYAML(SavedMessagePath, &SavedMessageSet)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", SavedMessagePath).
				Msg("Not found savedmessage list file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(SavedMessagePath, &SavedMessageSet)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", SavedMessagePath).
					Msg("Failed to create empty savedmessage list file")
				SavedMessageErr = fmt.Errorf("failed to create empty savedmessage list file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", SavedMessagePath).
				Msg("Failed to load savedmessage list file")
			SavedMessageErr = fmt.Errorf("failed to load savedmessage list file: %w", err)
		}
	} else {
		SavedMessageErr = nil
	}

	if SavedMessageSet == nil {
		SavedMessageSet = map[int64]SavedMessage{}
	}

	buildSavedMessageByMessageHandlers()
	return SavedMessageErr
}

type sortstruct struct {
	sharedData *SavedMessageSharedData // 存放一些标准列表里没有的数据，方便搜索

	onlyText  *models.InlineQueryResultArticle
	audio     *models.InlineQueryResultCachedAudio
	document  *models.InlineQueryResultCachedDocument
	gif       *models.InlineQueryResultCachedGif
	photo     *models.InlineQueryResultCachedPhoto
	sticker   *models.InlineQueryResultCachedSticker
	video     *models.InlineQueryResultCachedVideo
	videoNote *models.InlineQueryResultCachedDocument
	voice     *models.InlineQueryResultCachedVoice
	mpeg4gif  *models.InlineQueryResultCachedMpeg4Gif
}

type SavedMessageItems struct {
	OnlyText  []SavedMessageTypeCachedOnlyText  `yaml:"OnlyText,omitempty"`
	Audio     []SavedMessageTypeCachedAudio     `yaml:"Audio,omitempty"`
	Document  []SavedMessageTypeCachedDocument  `yaml:"Document,omitempty"`
	Gif       []SavedMessageTypeCachedGif       `yaml:"Gif,omitempty"`
	Mpeg4gif  []SavedMessageTypeCachedMpeg4Gif  `yaml:"Mpeg4Gif,omitempty"`
	Photo     []SavedMessageTypeCachedPhoto     `yaml:"Photo,omitempty"`
	Sticker   []SavedMessageTypeCachedSticker   `yaml:"Sticker,omitempty"`
	Video     []SavedMessageTypeCachedVideo     `yaml:"Video,omitempty"`
	VideoNote []SavedMessageTypeCachedVideoNote `yaml:"VideoNote,omitempty"`
	Voice     []SavedMessageTypeCachedVoice     `yaml:"Voice,omitempty"`
}

func (s *SavedMessageItems) All() []sortstruct {
	// var all []models.InlineQueryResult
	var list []sortstruct
	//  = make([]sortstruct, 100)
	for _, v := range s.OnlyText {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].onlyText != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		// var pendingTitle string
		// if len(v.TitleAndMessageText) > 20 {
		// 	pendingTitle = v.TitleAndMessageText[:20] + "..."
		// }
		list[index].onlyText = &models.InlineQueryResultArticle{
			ID:                  v.ID,
			Title:               v.TitleAndMessageText,
			Description:         v.Description,
			InputMessageContent: models.InputTextMessageContent{
				MessageText:        v.TitleAndMessageText,
				Entities:           v.Entities,
				LinkPreviewOptions: v.LinkPreviewOptions,
			},
			ReplyMarkup:         buildFromInfoButton(v.OriginInfo),
		}
	}
	for _, v := range s.Audio {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].audio != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		list[index].audio = &models.InlineQueryResultCachedAudio{
			ID:              v.ID,
			AudioFileID:     v.FileID,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
			ReplyMarkup:     buildFromInfoButton(v.OriginInfo),
		}

		list[index].sharedData = &SavedMessageSharedData{
			Title: v.Title,
			FileName: v.FileName,
			Description: v.Description,
		}
	}
	for _, v := range s.Document {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].document != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		list[index].document = &models.InlineQueryResultCachedDocument{
			ID:              v.ID,
			DocumentFileID:  v.FileID,
			Title:           v.Title,
			Description:     v.Description,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
			ReplyMarkup:     buildFromInfoButton(v.OriginInfo),
		}
	}
	for _, v := range s.Gif {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].gif != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		list[index].gif = &models.InlineQueryResultCachedGif{
			ID:              v.ID,
			GifFileID:       v.FileID,
			Title:           v.Title,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
			ReplyMarkup:     buildFromInfoButton(v.OriginInfo),
		}

		list[index].sharedData = &SavedMessageSharedData{
			Description: v.Description,
		}
	}
	for _, v := range s.Mpeg4gif {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].mpeg4gif != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		list[index].mpeg4gif = &models.InlineQueryResultCachedMpeg4Gif{
			ID:              v.ID,
			Mpeg4FileID:     v.FileID,
			Title:           v.Title,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
			ReplyMarkup:     buildFromInfoButton(v.OriginInfo),
		}
		list[index].sharedData = &SavedMessageSharedData{
			Description: v.Description,
		}
	}
	for _, v := range s.Photo {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].photo != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		list[index].photo = &models.InlineQueryResultCachedPhoto{
			ID:                    v.ID,
			PhotoFileID:           v.FileID,
			Title:                 v.Title,
			Description:           v.Description,
			Caption:               v.Caption,
			CaptionEntities:       v.CaptionEntities,
			ShowCaptionAboveMedia: v.CaptionAboveMedia,
			ReplyMarkup:           buildFromInfoButton(v.OriginInfo),
		}
	}
	for _, v := range s.Sticker {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].sticker != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		list[index].sticker = &models.InlineQueryResultCachedSticker{
			ID:            v.ID,
			StickerFileID: v.FileID,
			ReplyMarkup:   buildFromInfoButton(v.OriginInfo),
		}

		list[index].sharedData = &SavedMessageSharedData{
			Name:        v.SetName,
			Title:       v.SetTitle,
			Description: v.Description,
			FileName:    v.Emoji,
		}
	}
	for _, v := range s.Video {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].video != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		if v.Title == "" {
			v.Title = "video.mp4"
		}
		list[index].video = &models.InlineQueryResultCachedVideo{
			ID:              v.ID,
			VideoFileID:     v.FileID,
			Title:           v.Title,
			Description:     v.Description,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
			ReplyMarkup:     buildFromInfoButton(v.OriginInfo),
		}
	}
	for _, v := range s.VideoNote {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].document != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		list[index].document = &models.InlineQueryResultCachedDocument{
			ID:              v.ID,
			DocumentFileID:  v.FileID,
			Title:           v.Title,
			Description:     v.Description,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
			ReplyMarkup:     buildFromInfoButton(v.OriginInfo),
		}
	}
	for _, v := range s.Voice {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			fmt.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].voice != nil {
			fmt.Println("duplicate id", v.ID)
			continue
		}
		if v.Title == "" {
			v.Title = "audio"
		}
		list[index].voice = &models.InlineQueryResultCachedVoice{
			ID:              v.ID,
			VoiceFileID:     v.FileID,
			Title:           v.Title,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
			ReplyMarkup:     buildFromInfoButton(v.OriginInfo),
		}
		list[index].sharedData = &SavedMessageSharedData{
			Description: v.Description,
		}
	}


	// for _, n := range list {
	// 	if n.audio != nil {
	// 		all = append(all, n.audio)
	// 	} else if n.document != nil {
	// 		all = append(all, n.document)
	// 	} else if n.gif != nil {
	// 		all = append(all, n.gif)
	// 	} else if n.photo != nil {
	// 		all = append(all, n.photo)
	// 	} else if n.sticker != nil {
	// 		all = append(all, n.sticker)
	// 	} else if n.video != nil {
	// 		all = append(all, n.video)
	// 	} else if n.voice != nil {
	// 		all = append(all, n.voice)
	// 	} else if n.mpeg4gif != nil {
	// 		all = append(all, n.mpeg4gif)
	// 	}
	// }
	return list

}

func getMessageOriginData(msgOrigin *models.MessageOrigin) *OriginInfo {
	if msgOrigin == nil { return nil }

	switch msgOrigin.Type {
	case models.MessageOriginTypeUser:
		return &OriginInfo{
			FromName: utils.ShowUserName(&msgOrigin.MessageOriginUser.SenderUser),
			FromID: msgOrigin.MessageOriginUser.SenderUser.ID,
		}
	// 不再保存匿名的来源，已在调用处排除
	case models.MessageOriginTypeHiddenUser:
		return &OriginInfo{
			FromName: msgOrigin.MessageOriginHiddenUser.SenderUserName,
		}
	case models.MessageOriginTypeChat:
		return &OriginInfo{
			FromName: utils.ShowChatName(&msgOrigin.MessageOriginChat.SenderChat),
			FromID: msgOrigin.MessageOriginChat.SenderChat.ID,
		}
	case models.MessageOriginTypeChannel:
		return &OriginInfo{
			FromName: utils.ShowChatName(&msgOrigin.MessageOriginChannel.Chat),
			FromID: msgOrigin.MessageOriginChannel.Chat.ID,
			MessageID: msgOrigin.MessageOriginChannel.MessageID,
		}
	default:
		return nil
	}
}

func getMessageLink(msg *models.Message) *OriginInfo {
	// if msg.From.ID == msg.Chat.ID {
	// }
	attr := message_utils.GetMessageAttribute(msg)
	if attr.IsFromLinkedChannel || attr.IsFromAnonymous || attr.IsUserAsChannel {
		return &OriginInfo{
			FromName: utils.ShowChatName(msg.SenderChat),
			FromID: msg.SenderChat.ID,
			ChatID: msg.Chat.ID,
			MessageID: msg.ReplyToMessage.ID,
		}
	} else {
		return &OriginInfo{
			FromName: utils.ShowUserName(msg.ReplyToMessage.From),
			FromID: msg.ReplyToMessage.From.ID,
			ChatID: msg.Chat.ID,
			MessageID: msg.ReplyToMessage.ID,
		}
	}
}

type OriginInfo struct {
	FromName string `yaml:"FromName,omitempty"`
	FromID   int64  `yaml:"FromID,omitempty"`
	// FromChatID int64  `yaml:"FromChatID,omitempty"`

	// 用于查看消息来源
	ChatID    int64 `yaml:"ChatID,omitempty"`
	MessageID int   `yaml:"MessageID,omitempty"`
}

func buildFromInfoButton(o *OriginInfo) models.ReplyMarkup {
	if o == nil {
		return nil
	}
	var buttons []models.InlineKeyboardButton

	if o.FromID != 0 {
		if o.FromID < 0 {
			// -100 开头的 ID，为群组或频道
			buttons = append(buttons, models.InlineKeyboardButton{
				Text: "来自 " + o.FromName,
				URL:  fmt.Sprintf("https://t.me/c/%s/0", utils.RemoveIDPrefix(o.FromID)),
			})
		} else {
			buttons = append(buttons, models.InlineKeyboardButton{
				Text: "来自用户 " + o.FromName,
				URL:  fmt.Sprintf("https://t.me/@id%d", o.FromID),
			})
		}
	}
	if o.MessageID != 0 {
		if o.ChatID == 0 {
			// 保存来源是频道
			buttons = append(buttons, models.InlineKeyboardButton{
				Text: "查看消息",
				URL:  fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(o.FromID), o.MessageID),
			})
		} else {
			// 从群组中保存的消息
			buttons = append(buttons, models.InlineKeyboardButton{
				Text: "查看消息",
				URL:  fmt.Sprintf("https://t.me/c/%s/%d", utils.RemoveIDPrefix(o.ChatID), o.MessageID),
			})
		}

	}

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			buttons,
		},
	}

}

func buildSavedMessageByMessageHandlers() {
	for chatID := range SavedMessageSet {
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.OnlyText,
			chatID,
			"saved_message",
		)
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.Audio,
			chatID,
			"saved_message",
		)
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.Animation,
			chatID,
			"saved_message",
		)
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.Document,
			chatID,
			"saved_message",
		)
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.Photo,
			chatID,
			"saved_message",
		)
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.Sticker,
			chatID,
			"saved_message",
		)
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.Video,
			chatID,
			"saved_message",
		)
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.VideoNote,
			chatID,
			"saved_message",
		)
		plugin_utils.RemoveHandlerByMessageTypeHandler(
			models.ChatTypePrivate,
			message_utils.Voice,
			chatID,
			"saved_message",
		)
	}
	for chatID, user := range SavedMessageSet {
		if user.AgreePrivacyPolicy {
			plugin_utils.AddHandlerByMessageTypeHandlers([]plugin_utils.ByMessageTypeHandler{
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.OnlyText,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.Audio,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.Animation,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.Document,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.Photo,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.Sticker,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.Video,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.VideoNote,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
				{
					PluginName: "saved_message",
					ChatType: models.ChatTypePrivate,
					ForChatID: chatID,
					MessageType: message_utils.Voice,
					UpdateHandler: saveMessageFromCallbackQuery,
				},
			}...)
		}
	}
}
