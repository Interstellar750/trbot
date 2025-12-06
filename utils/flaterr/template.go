package flaterr

type Msg string

const (
	// LogTemplate is the template for log messages.
	SendMessage            Msg = "Failed to send message"
	SendDocument           Msg = "Failed to send document"
	EditMessageText        Msg = "Failed to edit message text"
	EditMessageMedia       Msg = "Failed to edit message media"
	EditMessageCaption     Msg = "Failed to edit message caption"
	EditMessageReplyMarkup Msg = "Failed to edit message reply markup"
	DeleteMessage          Msg = "Failed to delete message"
	DeleteMessages         Msg = "Failed to delete messages"
	AnswerCallbackQuery    Msg = "Failed to answer callback query"
	AnswerInlineQuery      Msg = "Failed to answer inline query"
	GetFile                Msg = "Failed to get file info"
	PinChatMessage         Msg = "Failed to pin chat message"
	UnpinChatMessage       Msg = "Failed to unpin chat message"
	GetStickerSet          Msg = "Failed to get sticker set info"
	ForwardMessage         Msg = "Failed to forward message"
)

// Str return message as string
func (m Msg) Str() string {
	return string(m)
}

// Fmt return a format string contains %s and %w
//
// %s is error content, %w is for error
//
// example: "failed to send [%s] message: %w"
func (m Msg) Fmt() string {
	switch m {
	case SendMessage:            return "failed to send [%s] message: %w"
	case SendDocument:           return "failed to send [%s] document: %w"
	case EditMessageText:        return "failed to edit message text to [%s]: %w"
	case EditMessageMedia:       return "failed to edit message media to [%s]: %w"
	case EditMessageCaption:     return "failed to edit message caption to [%s]: %w"
	case EditMessageReplyMarkup: return "failed to edit message reply markup to [%s]: %w"
	case DeleteMessage:          return "failed to delete [%s] message: %w"
	case DeleteMessages:         return "failed to delete [%s] messages: %w"
	case AnswerCallbackQuery:    return "failed to send [%s] callback answer: %w"
	case AnswerInlineQuery:      return "failed to send [%s] inline answer: %w"
	case GetFile:                return "failed to get [%s] file info: %w"
	case PinChatMessage:         return "failed to pin [%s] message: %w"
	case UnpinChatMessage:       return "failed to unpin [%s] message: %w"
	case GetStickerSet:          return "failed to get [%s] sticker set info: %w"
	case ForwardMessage:         return "failed to forward [%s] message: %w"
	default:
		return "unknown err content [%s]: %w"
	}
}
