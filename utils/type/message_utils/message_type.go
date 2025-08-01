package message_utils

import (
	"reflect"

	"github.com/go-telegram/bot/models"
)

// 消息类型
type Message struct {
	// https://core.telegram.org/bots/api#message

	Animation     bool `yaml:"Animation,omitempty"` // call gif, mpeg4 format, can save to GIFs, no caption
	Audio         bool `yaml:"Audio,omitempty"`     // or call music, can have caption, some music may as a document
	Document      bool `yaml:"Document,omitempty"`  // can have caption
	PaidMedia     bool `yaml:"PaidMedia,omitempty"` // photo or video, unknow caption
	Photo         bool `yaml:"Photo,omitempty"`     // a list, sort by resolution
	Sticker       bool `yaml:"Sticker,omitempty"`   // sticker, but some .webp file maybe will send as sticker, actual file format and resolution may not match the limitations. no caption
	Story         bool `yaml:"Story,omitempty"`
	Video         bool `yaml:"Video,omitempty"`
	VideoNote     bool `yaml:"VideoNote,omitempty"` // A circular video shot in Telegram
	Voice         bool `yaml:"Voice,omitempty"`     // can have caption
	OnlyText      bool `yaml:"OnlyText,omitempty"`  // just text message
	Checklist     bool `yaml:"Checklist,omitempty"`
	Contact       bool `yaml:"Contact,omitempty"`
	Dice          bool `yaml:"Dice,omitempty"`
	Game          bool `yaml:"Game,omitempty"`
	Poll          bool `yaml:"Poll,omitempty"`
	Venue         bool `yaml:"Venue,omitempty"`
	Location      bool `yaml:"Location,omitempty"`
	Invoice       bool `yaml:"Invoice,omitempty"`
	PinnedMessage bool `yaml:"PinnedMessage,omitempty"`
	Giveaway      bool `yaml:"Giveaway,omitempty"`
}

// 将消息类型结构体转换为 MessageTypeList(string) 类型
func (m Message)AsType() Type {
	val := reflect.ValueOf(m)
	typ := reflect.TypeOf(m)

	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).Bool() {
			return Type(typ.Field(i).Name)
		}
	}

	return ""
}

func (m Message)Str() string {
	val := reflect.ValueOf(m)
	typ := reflect.TypeOf(m)

	for i := 0; i < val.NumField(); i++ {
		if val.Field(i).Bool() {
			return string(Type(typ.Field(i).Name))
		}
	}

	return ""
}

type Type string

const (
	Animation     Type = "Animation"
	Audio         Type = "Audio"
	Document      Type = "Document"
	PaidMedia     Type = "PaidMedia"
	Photo         Type = "Photo"
	Sticker       Type = "Sticker"
	Story         Type = "Story"
	Video         Type = "Video"
	VideoNote     Type = "VideoNote"
	Voice         Type = "Voice"
	OnlyText      Type = "OnlyText"
	Checklist     Type = "Checklist"
	Contact       Type = "Contact"
	Dice          Type = "Dice"
	Game          Type = "Game"
	Poll          Type = "Poll"
	Venue         Type = "Venue"
	Location      Type = "Location"
	Invoice       Type = "Invoice"
	PinnedMessage Type = "PinnedMessage"
	Giveaway      Type = "Giveaway"
)

func (t Type)Str() string  {
	return string(t)
}

// 判断消息的类型
func GetMessageType(msg *models.Message) (msgType Message) {
	if msg.Document != nil {
		if msg.Animation != nil && msg.Animation.FileID == msg.Document.FileID && msg.Document.MimeType == "video/mp4" {
			msgType.Animation = true
		} else {
			msgType.Document = true
		}
	}
	if msg.Audio != nil {
		msgType.Audio = true
	}
	if msg.PaidMedia != nil {
		msgType.PaidMedia = true
	}
	if msg.Photo != nil {
		msgType.Photo = true
	}
	if msg.Sticker != nil {
		msgType.Sticker = true
	}
	if msg.Story != nil {
		msgType.Story = true
	}
	if msg.Video != nil {
		msgType.Video = true
	}
	if msg.VideoNote != nil {
		msgType.VideoNote = true
	}
	if msg.Voice != nil {
		msgType.Voice = true
	}
	if msg.Checklist != nil {
		msgType.Checklist = true
	}
	if msg.Contact != nil {
		msgType.Contact = true
	}
	if msg.Dice != nil {
		msgType.Dice = true
	}
	if msg.Game != nil {
		msgType.Game = true
	}
	if msg.Poll != nil {
		msgType.Poll = true
	}
	if msg.Venue != nil {
		msgType.Venue = true
	}
	if msg.Location != nil {
		msgType.Location = true
	}
	if msg.Invoice != nil {
		msgType.Invoice = true
	}
	if msg.PinnedMessage != nil {
		msgType.PinnedMessage = true
	}
	if msg.Giveaway != nil {
		msgType.Giveaway = true
	}
	if msg.Text != "" {
		msgType.OnlyText = true
	}

	return
}
