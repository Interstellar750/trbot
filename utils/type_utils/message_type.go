package type_utils

import "github.com/go-telegram/bot/models"

// 消息类型
type MessageType struct {
	// https://core.telegram.org/bots/api#message

	Animation bool `yaml:"Animation,omitempty"` // call gif, mpeg4 format, can save to GIFs, no caption
	Audio     bool `yaml:"Audio,omitempty"`     // or call music, can have caption, some music may as a document
	Document  bool `yaml:"Document,omitempty"`  // can have caption
	PaidMedia bool `yaml:"PaidMedia,omitempty"` // photo or video, unknow caption
	Photo     bool `yaml:"Photo,omitempty"`     // a list, sort by resolution
	Sticker   bool `yaml:"Sticker,omitempty"`   // sticker, but some .webp file maybe will send as sticker, actual file format and resolution may not match the limitations. no caption
	Story     bool `yaml:"Story,omitempty"`
	Video     bool `yaml:"Video,omitempty"`
	VideoNote bool `yaml:"VideoNote,omitempty"` // A circular video shot in Telegram
	Voice     bool `yaml:"Voice,omitempty"`     // can have caption
	OnlyText  bool `yaml:"OnlyText,omitempty"`  // just text message
	Contact   bool `yaml:"Contact,omitempty"`
	Dice      bool `yaml:"Dice,omitempty"`
	Game      bool `yaml:"Game,omitempty"`
	Poll      bool `yaml:"Poll,omitempty"`
	Venue     bool `yaml:"Venue,omitempty"`
	Location  bool `yaml:"Location,omitempty"`
	Invoice   bool `yaml:"Invoice,omitempty"`
	Giveaway  bool `yaml:"Giveaway,omitempty"`
}

// 判断消息的类型
func GetMessageType(msg *models.Message) MessageType {
	var msgType MessageType
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
	if msg.Giveaway != nil {
		msgType.Giveaway = true
	}
	if msg.Text != "" {
		msgType.OnlyText = true
	}
	return msgType
}
