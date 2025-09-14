package meilisearch_utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"
	"trbot/utils/origin_info"
	"trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
)

type MessageData struct {
	ID              int                `json:"id"`
	Type            message_utils.Type `json:"type,omitempty"`
	FileID          string             `json:"file_id,omitempty"`
	FileTitle       string             `json:"file_title,omitempty"`
	FileName        string             `json:"file_name,omitempty"`
	MediaGroupID    string             `json:"media_group_id,omitempty"`
	// MessageThreadID int                `json:"message_thread_id,omitempty"`
	Text            string             `json:"text,omitempty"`
	Desc            string             `json:"desc,omitempty"`

	Entities              []models.MessageEntity     `json:"entities,omitempty"`
	LinkPreviewOptions    *models.LinkPreviewOptions `json:"link_preview_options,omitempty"`
	ShowCaptionAboveMedia bool                       `json:"show_caption_above_media,omitempty"`
	// HasMediaSpoiler       bool                       `json:"has_media_spoiler,omitempty"`

	OriginInfo *origin_info.OriginInfo `json:"origin_info,omitempty"`
}

func (md MessageData) MsgIDStr() string {
	return strconv.Itoa(md.ID)
}

func (md MessageData) BuildButton(username string) models.ReplyMarkup {
	var buttons []models.InlineKeyboardButton

	// if md.OriginInfo != nil {
	// 	if md.OriginInfo.FromID < 0 && md.OriginInfo.MessageID != 0 {
	// 		buttons = append(buttons, models.InlineKeyboardButton{
	// 			Text: "来自 " + md.OriginInfo.FromName,
	// 			URL:  utils.MsgLinkPrivate(md.OriginInfo.FromID, md.OriginInfo.MessageID),
	// 		})
	// 	} else if md.OriginInfo.FromID > 0 {
	// 		buttons = append(buttons, models.InlineKeyboardButton{
	// 			Text: "来自 " + md.OriginInfo.FromName,
	// 			URL:  fmt.Sprintf("tg://user?id=%d", md.OriginInfo.FromID),
	// 		})
	// 	}
	// }

	if md.MediaGroupID != "" {
		buttons = append(buttons, models.InlineKeyboardButton{
			Text: "查看完整消息",
			URL:  fmt.Sprintf("https://t.me/%s/%d", username, md.ID),
		})
	}

	if len(buttons) == 0 {
		return nil
	}

	return &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{buttons}}
}

// Please pass in the parameter as a pointer
func UnmarshalMessageData(data, out any) error {
	temp, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return json.Unmarshal(temp, out)
}

func BuildMessageData(ctx context.Context, thebot *bot.Bot, msg *models.Message) MessageData {
	if msg == nil { return MessageData{} }
	var data = MessageData{
		ID:                    msg.ID,
		MediaGroupID:          msg.MediaGroupID,
		// MessageThreadID:       msg.MessageThreadID,
		LinkPreviewOptions:    msg.LinkPreviewOptions,
		ShowCaptionAboveMedia: msg.ShowCaptionAboveMedia,
	}

	if msg.Caption != "" {
		data.Text = msg.Caption
		data.Entities = msg.CaptionEntities
	} else if msg.Text != "" {
		data.Text = msg.Text
		data.Entities = msg.Entities
	}

	msgType := message_utils.GetMessageType(msg)
	data.Type = msgType.AsType()
	switch {
	case msgType.Text:
		// do nothing
	case msgType.Animation:
		data.FileID   = msg.Animation.FileID
		data.FileName = msg.Animation.FileName
	case msgType.Audio:
		data.FileID    = msg.Audio.FileID
		data.FileName  = msg.Audio.FileName
		data.FileTitle = msg.Audio.Title
	case msgType.Document:
		data.FileID   = msg.Document.FileID
		data.FileName = msg.Document.FileName
	case msgType.Photo:
		data.FileID = msg.Photo[len(msg.Photo)-1].FileID
	case msgType.Sticker:
		data.FileID   = msg.Sticker.FileID
		data.FileName = msg.Sticker.SetName

		if msg.Sticker.SetName != "" {
			stickerSet, _ := thebot.GetStickerSet(ctx, &bot.GetStickerSetParams{ Name: msg.Sticker.SetName })
			if stickerSet != nil {
				data.FileTitle = stickerSet.Title
			}
		}
	case msgType.Video:
		data.FileID   = msg.Video.FileID
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

func CreateChatIndex(ctx context.Context, client *meilisearch.ServiceManager, chatID string) error {
	taskinfo, err := (*client).CreateIndexWithContext(ctx, &meilisearch.IndexConfig{ Uid: chatID, PrimaryKey: "id" })
	if err != nil {
		return fmt.Errorf("failed to send create chat index request: %w", err)
	}
	task, err := WaitForTask(ctx, client, taskinfo.TaskUID, time.Second * 1)
	if err != nil {
		return fmt.Errorf("wait for create chat index task failed: %w", err)
	}
	if task.Status != meilisearch.TaskStatusSucceeded {
		return fmt.Errorf("create chat index failed: %s", task.Error.Message)
	}

	taskinfo, err = (*client).Index(chatID).UpdateFilterableAttributesWithContext(ctx, &[]string{"type"})
	if err != nil {
		return fmt.Errorf("failed to send update filterable attributes: %w", err)
	}
	task, err = WaitForTask(ctx, client, taskinfo.TaskUID, time.Second * 1)
	if err != nil {
		return fmt.Errorf("wait for update filterable attributes task failed: %w", err)
	}
	if task.Status != meilisearch.TaskStatusSucceeded {
		return fmt.Errorf("update filterable attributes failed: %s", task.Error.Message)
	}
	return nil
}

func WaitForTask(ctx context.Context, client *meilisearch.ServiceManager, taskUID int64, interval time.Duration) (*meilisearch.Task, error) {
	return (*client).WaitForTaskWithContext(ctx, taskUID, interval )
}
