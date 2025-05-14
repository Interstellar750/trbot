package plugin_utils

import (
	"log"
	"trbot/utils/handler_structs"
	"trbot/utils/type_utils"

	"github.com/go-telegram/bot/models"
)

type HandlerByMessageTypeFor struct {
	// 改成 map[msgType][]HandlerByMessageType
	// Private    map[type_utils.MessageTypeList][]HandlerByMessageType
	Private    *HandlerByMessageTypeFunctions
	Group      *HandlerByMessageTypeFunctions
	Supergroup *HandlerByMessageTypeFunctions
	Channel    *HandlerByMessageTypeFunctions
}

type HandlerByMessageTypeFunctions struct {
	Animation []HandlerByMessageType
	Audio     []HandlerByMessageType
	Document  []HandlerByMessageType
	PaidMedia []HandlerByMessageType
	Photo     []HandlerByMessageType
	Sticker   []HandlerByMessageType
	Story     []HandlerByMessageType
	Video     []HandlerByMessageType
	VideoNote []HandlerByMessageType
	Voice     []HandlerByMessageType
	OnlyText  []HandlerByMessageType
	Contact   []HandlerByMessageType
	Dice      []HandlerByMessageType
	Game      []HandlerByMessageType
	Poll      []HandlerByMessageType
	Venue     []HandlerByMessageType
	Location  []HandlerByMessageType
	Invoice   []HandlerByMessageType
	Giveaway  []HandlerByMessageType
}

type HandlerByMessageType struct {
	Name        string
	ChatType    models.ChatType
	MessageType type_utils.MessageTypeList
	Handler     func(*handler_structs.SubHandlerParams)
}

func AddHandlerByMessageTypePlugin(plugins ...HandlerByMessageType) {
	if AllPlugins.HandlerByMessageTypeFor == nil {
		AllPlugins.HandlerByMessageTypeFor = &HandlerByMessageTypeFor{}
	}

	var targetChatType *HandlerByMessageTypeFunctions

	for _, plugin := range plugins {
		switch plugin.ChatType {
		case models.ChatTypePrivate:
			targetChatType = AllPlugins.HandlerByMessageTypeFor.Private
		case models.ChatTypeGroup:
			targetChatType = AllPlugins.HandlerByMessageTypeFor.Group
		case models.ChatTypeSupergroup:
			targetChatType = AllPlugins.HandlerByMessageTypeFor.Supergroup
		case models.ChatTypeChannel:
			targetChatType = AllPlugins.HandlerByMessageTypeFor.Channel
		default:
			log.Println("unknown chat type", plugin)
			continue
		}

		if targetChatType == nil {
			targetChatType = &HandlerByMessageTypeFunctions{}
		}

		switch plugin.MessageType {
		case type_utils.Animation:
			targetChatType.Animation = append(targetChatType.Animation, plugin)
		case type_utils.Audio:
			targetChatType.Audio = append(targetChatType.Audio, plugin)
		case type_utils.Document:
			targetChatType.Document = append(targetChatType.Document, plugin)
		case type_utils.PaidMedia:
			targetChatType.PaidMedia = append(targetChatType.PaidMedia, plugin)
		case type_utils.Photo:
			targetChatType.Photo = append(targetChatType.Photo, plugin)
		case type_utils.Sticker:
			targetChatType.Sticker = append(targetChatType.Sticker, plugin)
		case type_utils.Story:
			targetChatType.Story = append(targetChatType.Story, plugin)
		case type_utils.Video:
			targetChatType.Video = append(targetChatType.Video, plugin)
		case type_utils.VideoNote:
			targetChatType.VideoNote = append(targetChatType.VideoNote, plugin)
		case type_utils.Voice:
			targetChatType.Voice = append(targetChatType.Voice, plugin)
		case type_utils.OnlyText:
			targetChatType.OnlyText = append(targetChatType.OnlyText, plugin)
		case type_utils.Contact:
			targetChatType.Contact = append(targetChatType.Contact, plugin)
		case type_utils.Dice:
			targetChatType.Dice = append(targetChatType.Dice, plugin)
		case type_utils.Game:
			targetChatType.Game = append(targetChatType.Game, plugin)
		case type_utils.Poll:
			targetChatType.Poll = append(targetChatType.Poll, plugin)
		case type_utils.Venue:
			targetChatType.Venue = append(targetChatType.Venue, plugin)
		case type_utils.Location:
			targetChatType.Location = append(targetChatType.Location, plugin)
		case type_utils.Invoice:
			targetChatType.Invoice = append(targetChatType.Invoice, plugin)
		case type_utils.Giveaway:
			targetChatType.Giveaway = append(targetChatType.Giveaway, plugin)
		}
	}

	
}
