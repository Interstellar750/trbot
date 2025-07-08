package flate

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
	GetFile                Msg = "Failed to get file"
)

const (
	FormatSendMessage            string = "failed to send [%s] message: %w"
	FormatSendDocument           string = "failed to send [%s] document: %w"
	FormatEditMessageText        string = "failed to edit message text to [%s]: %w"
	FormatEditMessageMedia       string = "failed to edit message media to [%s]: %w"
	FormatEditMessageCaption     string = "failed to edit message caption to [%s]: %w"
	FormatEditMessageReplyMarkup string = "failed to edit message reply markup to [%s]: %w"
	FormatDeleteMessage          string = "failed to delete [%s] message: %w"
	FormatDeleteMessages         string = "failed to delete [%s] messages: %w"
	FormatAnswerCallbackQuery    string = "failed to send [%s] callback answer: %w"
	FormatAnswerInlineQuery      string = "failed to send [%s] inline answer: %w"
	FormatGetFile                string = "failed to get [%s] file: %w"
)

// return message as string
func (m Msg) Str() string {
	return string(m)
}

// return a format string contains %s and %w
func (m Msg) Template() string {
	switch m {
	case SendMessage:            return FormatSendMessage
	case SendDocument:           return FormatSendDocument
	case EditMessageText:        return FormatEditMessageText
	case EditMessageMedia:       return FormatEditMessageMedia
	case EditMessageCaption:     return FormatEditMessageCaption
	case EditMessageReplyMarkup: return FormatEditMessageReplyMarkup
	case DeleteMessage:          return FormatDeleteMessage
	case DeleteMessages:         return FormatDeleteMessages
	case AnswerCallbackQuery:    return FormatAnswerCallbackQuery
	case AnswerInlineQuery:      return FormatAnswerInlineQuery
	case GetFile:                return FormatGetFile
	default:
		return "unknown error content [%s], err: %w"
	}
}
