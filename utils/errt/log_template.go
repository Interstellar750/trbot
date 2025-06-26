package errt

const (
	// LogTemplate is the template for log messages.
	SendMessage            string = "Failed to send message"
	SendDocument           string = "Failed to send document"
	EditMessageText        string = "Failed to edit message text"
	EditMessageCaption     string = "Failed to edit message caption"
	EditMessageReplyMarkup string = "Failed to edit message reply markup"
	DeleteMessage          string = "Failed to delete message"
	DeleteMessages         string = "Failed to delete messages"
	AnswerCallbackQuery    string = "Failed to answer callback query"
	AnswerInlineQuery      string = "Failed to answer inline query"
)
