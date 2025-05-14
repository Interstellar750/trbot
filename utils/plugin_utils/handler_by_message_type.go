package plugin_utils

import "trbot/utils/handler_structs"

type HandlerByMessageType struct {
	Private    *HandlerByMessageTypeFunctions
	Group      *HandlerByMessageTypeFunctions
	Supergroup *HandlerByMessageTypeFunctions
	Channel    *HandlerByMessageTypeFunctions
}

type HandlerByMessageTypeFunctions struct {
	Animation []handler
	Audio     []handler
	Document  []handler
	PaidMedia []handler
	Photo     []handler
	Sticker   []handler
	Story     []handler
	Video     []handler
	VideoNote []handler
	Voice     []handler
	OnlyText  []handler
	Contact   []handler
	Dice      []handler
	Game      []handler
	Poll      []handler
	Venue     []handler
	Location  []handler
	Invoice   []handler
	Giveaway  []handler
}

type handler struct {
	Name    string
	Handler func(*handler_structs.SubHandlerParams)
}
