package plugin_utils

import "github.com/go-telegram/bot/models"

type HandlerHelp struct {
	Name        string // show in help list
	Description string
	ParseMode   models.ParseMode
	ReplyMarkup models.ReplyMarkup // button, if use callback, please register it manually
}

func BuildHandlerHelpKeyboard() models.ReplyMarkup {
	var button [][]models.InlineKeyboardButton
	for _, handler := range AllPlugins.HandlerHelp {
		button = append(button, []models.InlineKeyboardButton{
			{
				Text:         handler.Name,
				CallbackData: "help-handler_" + handler.Name,
			},
		})
	}
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: button,
	}
}

func AddHandlerHelpInfo(HandlerHelps ...HandlerHelp) int {
	if AllPlugins.HandlerHelp == nil { AllPlugins.HandlerHelp = []HandlerHelp{} }
	var Count int
	for _, handlerHelp := range HandlerHelps {
		if handlerHelp.Name == "" { continue }
		AllPlugins.HandlerHelp = append(AllPlugins.HandlerHelp, handlerHelp)
		Count++
	}
	return Count
}
