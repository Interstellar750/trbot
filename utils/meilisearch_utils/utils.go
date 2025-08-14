package meilisearch_utils

import (
	"context"
	"encoding/json"
	"strconv"
	"trbot/utils/origin_info"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
)

type MessageData struct {
	MsgID     int                `json:"msg_id"`
	MsgType   message_utils.Type `json:"msg_type,omitempty"`
	FileID    string             `json:"file_id,omitempty"`
	FileTitle string             `json:"file_title,omitempty"`
	FileName  string             `json:"file_name,omitempty"`
	Text      string             `json:"text,omitempty"`
	Desc      string             `json:"desc,omitempty"`

	Entities              []models.MessageEntity     `json:"entities,omitempty"`
	LinkPreviewOptions    *models.LinkPreviewOptions `json:"link_preview_options,omitempty"`
	ShowCaptionAboveMedia bool                       `json:"show_caption_above_media,omitempty"`
	// HasMediaSpoiler       bool                       `json:"has_media_spoiler,omitempty"`

	OriginInfo *origin_info.OriginInfo `json:"origin_info,omitempty"`
}

func (md MessageData) MsgIDStr() string {
	return strconv.Itoa(md.MsgID)
}

// Please pass in the parameter as a pointer
func UnMarshalMessageData(data, out any) error {
	temp, err := json.Marshal(data)
	if err != nil { return err }
	err = json.Unmarshal(temp, out)
	if err != nil { return err }
	return nil
}

func BuildMessageData(ctx context.Context, thebot *bot.Bot, msg *models.Message) MessageData {
	if msg == nil { return MessageData{} }
	var data = MessageData{
		MsgID:                 msg.ID,
		LinkPreviewOptions:    msg.LinkPreviewOptions,
		ShowCaptionAboveMedia: msg.ShowCaptionAboveMedia,
		// HasMediaSpoiler:       msg.HasMediaSpoiler,
	}

	if msg.Caption != "" {
		data.Text = msg.Caption
		data.Entities = msg.CaptionEntities
	} else if msg.Text != "" {
		data.Text = msg.Text
		data.Entities = msg.Entities
	}

	msgType := message_utils.GetMessageType(msg)
	data.MsgType = msgType.AsType()
	switch {
	case msgType.Text:
		// do nothing
	case msgType.Animation:
		data.FileID = msg.Animation.FileID
		data.FileName = msg.Animation.FileName
	case msgType.Audio:
		data.FileID = msg.Audio.FileID
		data.FileName = msg.Audio.FileName
		data.FileTitle = msg.Audio.Title
	case msgType.Document:
		data.FileID = msg.Document.FileID
		data.FileName = msg.Document.FileName
	case msgType.Photo:
		data.FileID = msg.Photo[len(msg.Photo)-1].FileID
	case msgType.Sticker:
		data.FileID = msg.Sticker.FileID
		data.FileName = msg.Sticker.SetName

		if msg.Sticker.SetName != "" {
			stickerSet, _ := thebot.GetStickerSet(ctx, &bot.GetStickerSetParams{Name: msg.Sticker.SetName})
			if stickerSet != nil {
				data.FileTitle = stickerSet.Title
			}
		}
	case msgType.Video:
		data.FileID = msg.Video.FileID
		data.FileName = msg.Video.FileName

		if data.FileName == "" {
			data.FileName = "video.mp4"
		}
	case msgType.VideoNote:
		data.FileID   = msg.VideoNote.FileID
		data.FileName = "video_note.mp4"
	case msgType.Voice:
		data.FileID    = msg.Voice.FileID
		data.FileTitle = data.Text
		if data.FileTitle == "" {
			data.FileTitle = msg.Voice.MimeType
		}
	}
	return data
}

func CreateChatIndex(client *meilisearch.ServiceManager, chatID string) {
	(*client).CreateIndex(&meilisearch.IndexConfig{ Uid: chatID, PrimaryKey: "msg_id" })
	indexManager := (*client).Index(chatID)
	// indexManager.UpdateSearchableAttributes(&[]string{"msg_id", "msg_type", "file_title", "file_name", "text", "desc"})
	// indexManager.UpdateDisplayedAttributes(&[]string{"msg_id", "msg_type", "file_title", "file_name", "text", "desc"})
	indexManager.UpdateFilterableAttributes(&[]string{"msg_type"})
}
