package saved_message

import "github.com/go-telegram/bot/models"

// 用于在构建 inline result 列表后存放列表中没有的数据
type SavedMessageSharedData struct {
	Name        string
	Title       string
	FileName    string
	Description string
}

type SavedMessageTypeCachedOnlyText struct {
	ID                  string                     `yaml:"ID"`
	TitleAndMessageText string                     `yaml:"TitleAndMessageText"`
	Description         string                     `yaml:"Description,omitempty"`
	Entities            []models.MessageEntity     `yaml:"Entities,omitempty"`
	LinkPreviewOptions  *models.LinkPreviewOptions `yaml:"LinkPreviewOptions,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}

type SavedMessageTypeCachedAudio struct {
	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Caption           string                 `yaml:"Caption,omitempty"`
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`

	Title       string `yaml:"Title,omitempty"`
	FileName    string `yaml:"FileName,omitempty"`
	Description string `yaml:"Description,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}

type SavedMessageTypeCachedDocument struct {
	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Title             string                 `yaml:"Title"`
	Description       string                 `yaml:"Description,omitempty"`
	Caption           string                 `yaml:"Caption,omitempty"`
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}

type SavedMessageTypeCachedGif struct {
	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Title             string                 `yaml:"Title,omitempty"`
	Caption           string                 `yaml:"Caption,omitempty"`
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`

	Description string `yaml:"Description,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}

type SavedMessageTypeCachedPhoto struct {
	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Title             string                 `yaml:"Title,omitempty"`       // inline 标题
	Description       string                 `yaml:"Description,omitempty"` // inline 描述
	Caption           string                 `yaml:"Caption,omitempty"`     // 发送后图片携带的文本
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
	CaptionAboveMedia bool                   `yaml:"CaptionAboveMedia,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}

type SavedMessageTypeCachedSticker struct {
	ID     string `yaml:"ID"`
	FileID string `yaml:"FileID"`

	SetName     string `yaml:"SetName,omitempty"`
	SetTitle    string `yaml:"SetTitle,omitempty"`
	Description string `yaml:"Description,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}

type SavedMessageTypeCachedVideo struct {
	ID              string                 `yaml:"ID"`
	FileID          string                 `yaml:"FileID"`
	Title           string                 `yaml:"Title"`                 // inline 标题
	Description     string                 `yaml:"Description,omitempty"` // inline 描述
	Caption         string                 `yaml:"Caption,omitempty"`     // 发送后图片携带的文本
	CaptionEntities []models.MessageEntity `yaml:"CaptionEntities,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}

type SavedMessageTypeCachedVideoNote struct {
	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`

	ID              string                 `yaml:"ID"`
	FileID          string                 `yaml:"FileID"`
	Title           string                 `yaml:"Title"`
	Description     string                 `yaml:"Description,omitempty"`
	Caption         string                 `yaml:"Caption,omitempty"` // 利用 bot 修改信息可以发出带文字的圆形视频，但是发送后不带文字
	CaptionEntities []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
}

type SavedMessageTypeCachedVoice struct {
	ID              string                 `yaml:"ID"`
	FileID          string                 `yaml:"FileID"`
	Title           string                 `yaml:"Title"`
	Caption         string                 `yaml:"Caption,omitempty"`
	CaptionEntities []models.MessageEntity `yaml:"CaptionEntities,omitempty"`

	Description string `yaml:"Description,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}

type SavedMessageTypeCachedMpeg4Gif struct {
	ID              string                 `yaml:"ID"`
	FileID          string                 `yaml:"FileID"`
	Title           string                 `yaml:"Title,omitempty"`
	Caption         string                 `yaml:"Caption,omitempty"`
	CaptionEntities []models.MessageEntity `yaml:"CaptionEntities,omitempty"`

	Description string `yaml:"Description,omitempty"`

	IsDeleted  bool        `yaml:"IsDeleted,omitempty"`
	OriginInfo *OriginInfo `yaml:"OriginInfo,omitempty"`
}
