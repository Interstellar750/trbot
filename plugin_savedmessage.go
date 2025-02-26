package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)



type SavedMessage struct {
	// ForUserID          int64 `yaml:"ForUserID"`
	Count              int   `yaml:"Count"`                // 当前存储的数量
	SavedTimes         int   `yaml:"SavedTimes,omitempty"` // 一共存过多少个
	Limit              int   `yaml:"Limit,omitempty"`
	AgreePrivacyPolicy bool  `yaml:"AgreePrivacyPolicy,omitempty"`

	Item SavedMessageItems `yaml:"Item,omitempty"`
}

type sortstruct struct {
	// 存放一些标准列表里没有的数据，方便搜索
	sharedData *SavedMessageSharedData

	audio    *models.InlineQueryResultCachedAudio
	document *models.InlineQueryResultCachedDocument
	gif      *models.InlineQueryResultCachedGif
	photo    *models.InlineQueryResultCachedPhoto
	sticker  *models.InlineQueryResultCachedSticker
	video    *models.InlineQueryResultCachedVideo
	voice    *models.InlineQueryResultCachedVoice
	mpeg4gif *models.InlineQueryResultCachedMpeg4Gif
}

type SavedMessageItems struct {
	Audio    []SavedMessageTypeCachedAudio    `yaml:"Audio,omitempty"`
	Document []SavedMessageTypeCachedDocument `yaml:"Document,omitempty"`
	Gif      []SavedMessageTypeCachedGif      `yaml:"Gif,omitempty"`
	Photo    []SavedMessageTypeCachedPhoto    `yaml:"Photo,omitempty"`
	Sticker  []SavedMessageTypeCachedSticker  `yaml:"Sticker,omitempty"`
	Video    []SavedMessageTypeCachedVideo    `yaml:"Video,omitempty"`
	Voice    []SavedMessageTypeCachedVoice    `yaml:"Voice,omitempty"`
	Mpeg4gif []SavedMessageTypeCachedMpeg4Gif `yaml:"Mpeg4Gif,omitempty"`
}


func (s *SavedMessageItems) All() []sortstruct {
	// var all []models.InlineQueryResult
	var list []sortstruct
	//  = make([]sortstruct, 100)
	for _, v := range s.Audio {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].audio != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		list[index].audio = &models.InlineQueryResultCachedAudio{
			ID:                    v.ID,
			AudioFileID:           v.FileID,
			Caption:               v.Caption,
			CaptionEntities:       v.CaptionEntities,
		}

		list[index].sharedData = &SavedMessageSharedData{
			Description: v.Description,
		}
	}
	for _, v := range s.Document {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].document != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		list[index].document = &models.InlineQueryResultCachedDocument{
			ID:                    v.ID,
			DocumentFileID:        v.FileID,
			Title:                 v.Title,
			Description:           v.Description,
			Caption:               v.Caption,
			CaptionEntities:       v.CaptionEntities,
		}
	}
	for _, v := range s.Gif {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].gif != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		list[index].gif = &models.InlineQueryResultCachedGif{
			ID:                    v.ID,
			GifFileID:             v.FileID,
			Title:                 v.Title,
			Caption:               v.Caption,
			CaptionEntities:       v.CaptionEntities,
		}

		list[index].sharedData = &SavedMessageSharedData{
			Description: v.Description,
		}
	}
	for _, v := range s.Photo {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].photo != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		list[index].photo = &models.InlineQueryResultCachedPhoto{
			ID:                    v.ID,
			PhotoFileID:           v.FileID,
			Title:                 v.Title,
			Description:           v.Description,
			Caption:               v.Caption,
			CaptionEntities:       v.CaptionEntities,
			ShowCaptionAboveMedia: v.CaptionAboveMedia,
		}
	}
	for _, v := range s.Sticker {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].sticker != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		list[index].sticker = &models.InlineQueryResultCachedSticker{
			ID:            v.ID,
			StickerFileID: v.FileID,
		}

		list[index].sharedData = &SavedMessageSharedData{
			Name: v.SetName,
			Title: v.SetTitle,
			Description: v.Description,
		}
	}
	for _, v := range s.Video {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].video != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		list[index].video = &models.InlineQueryResultCachedVideo{
			ID:              v.ID,
			VideoFileID:     v.FileID,
			Title:           v.Title,
			Description:     v.Description,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
		}
	}
	for _, v := range s.Voice {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].voice != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		list[index].voice = &models.InlineQueryResultCachedVoice{
			ID:              v.ID,
			VoiceFileID:     v.FileID,
			Title:           v.Title,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
		}
		list[index].sharedData = &SavedMessageSharedData{
			Description: v.Description,
		}
	}
	for _, v := range s.Mpeg4gif {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].mpeg4gif != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		list[index].mpeg4gif = &models.InlineQueryResultCachedMpeg4Gif{
			ID:              v.ID,
			Mpeg4FileID:     v.FileID,
			Title:           v.Title,
			Caption:         v.Caption,
			CaptionEntities: v.CaptionEntities,
		}
		list[index].sharedData = &SavedMessageSharedData{
			Description: v.Description,
		}
	}
	
	// for _, n := range list {
	// 	if n.audio != nil {
	// 		all = append(all, n.audio)
	// 	} else if n.document != nil {
	// 		all = append(all, n.document)
	// 	} else if n.gif != nil {
	// 		all = append(all, n.gif)
	// 	} else if n.photo != nil {
	// 		all = append(all, n.photo)
	// 	} else if n.sticker != nil {
	// 		all = append(all, n.sticker)
	// 	} else if n.video != nil {
	// 		all = append(all, n.video)
	// 	} else if n.voice != nil {
	// 		all = append(all, n.voice)
	// 	} else if n.mpeg4gif != nil {
	// 		all = append(all, n.mpeg4gif)
	// 	}
	// }
	return list

}

type SavedMessageSharedData struct {
    Name        string
	Title       string
	Description string
}

type SavedMessageTypeCachedAudio struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	Description string `yaml:"Description,omitempty"`

	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Caption           string                 `yaml:"Caption,omitempty"`
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
}

type SavedMessageTypeCachedDocument struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Title             string                 `yaml:"Title"`
	Description       string                 `yaml:"Description,omitempty"`
	Caption           string                 `yaml:"Caption,omitempty"`
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
}

type SavedMessageTypeCachedGif struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	Description string `yaml:"Description,omitempty"`

	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Title             string                 `yaml:"Title,omitempty"`
	Caption           string                 `yaml:"Caption,omitempty"`
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
}

type SavedMessageTypeCachedPhoto struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Title             string                 `yaml:"Title,omitempty"`       // inline 标题
	Description       string                 `yaml:"Description,omitempty"` // inline 描述
	Caption           string                 `yaml:"Caption,omitempty"`     // 发送后图片携带的文本
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
	CaptionAboveMedia bool                   `yaml:"CaptionAboveMedia,omitempty"`
}

type SavedMessageTypeCachedSticker struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	SetName     string `yaml:"SetName,omitempty"`
	SetTitle    string `yaml:"SetTitle,omitempty"`
	Description string `yaml:"Description,omitempty"`

	ID     string `yaml:"ID"`
	FileID string `yaml:"FileID"`
}

type SavedMessageTypeCachedVideo struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	ID              string                 `yaml:"ID"`
	FileID          string                 `yaml:"FileID"`
	Title           string                 `yaml:"Title"`       // inline 标题
	Description     string                 `yaml:"Description,omitempty"` // inline 描述
	Caption         string                 `yaml:"Caption,omitempty"`     // 发送后图片携带的文本
	CaptionEntities []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
}

type SavedMessageTypeCachedVoice struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	Description string `yaml:"Description,omitempty"` // inline 描述

	ID              string                 `yaml:"ID"`
	FileID          string                 `yaml:"FileID"`
	Title           string                 `yaml:"Title"`   // inline 标题
	Caption         string                 `yaml:"Caption,omitempty"` // 发送后图片携带的文本
	CaptionEntities []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
}

type SavedMessageTypeCachedMpeg4Gif struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	Description string `yaml:"Description,omitempty"` // inline 描述

	ID              string                 `yaml:"ID"`
	FileID          string                 `yaml:"FileID"`
	Title           string                 `yaml:"Title,omitempty"`       // inline 标题
	Caption         string                 `yaml:"Caption,omitempty"`     // 发送后图片携带的文本
	CaptionEntities []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
}

func saveMessageHandler(opts *subHandlerOpts) {
	userData := getIDInfo(&opts.update.Message.From.ID)
	if userData == nil && AddUserInfo(opts.update.Message.From) {
		log.Printf("add (saveMessage)private \"%s\"(%s)[%d] in database",
			showUserName(opts.update.Message.From), opts.update.Message.From.Username, opts.update.Message.From.ID,
		)
		userData = getIDInfo(&opts.update.Message.From.ID)
	}
	SavedMessage := database.Data.SavedMessage[userData.ID]

	messageParams := &bot.SendMessageParams{
		ChatID: opts.update.Message.Chat.ID,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
		ParseMode: models.ParseModeHTML,
	}

	if !SavedMessage.AgreePrivacyPolicy {
		messageParams.Text = "此功能需要保存一些信息才能正常工作，在使用这个功能前，请先阅读一下我们会保存哪些信息"
		messageParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击查看",
			URL: fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy", botMe.Username),
		}}}}
		_, err := opts.thebot.SendMessage(opts.ctx, messageParams)
		if err != nil {
			log.Printf("Error response /save command initial info: %v", err)
		}
		return
	}

	if SavedMessage.Limit == 0 && SavedMessage.Count == 0 {
		SavedMessage.Limit = 100
	}

	// 若不是初次添加，为 0 就是不限制
	if SavedMessage.Limit != 0 && SavedMessage.Count >= SavedMessage.Limit {
		messageParams.Text = "已达到限制，无法保存更多内容"
		_, err := opts.thebot.SendMessage(opts.ctx, messageParams)
		if err != nil {
			log.Printf("Error response /save command reach limit: %v", err)
		}
		return
	}

	// var pendingMessage string
	if opts.update.Message.ReplyToMessage != nil {
		var DescriptionText string
		// 获取使用命令保存时设定的描述
		if len(opts.update.Message.Text) > len(opts.fields[0]) + 1 {
			DescriptionText = opts.update.Message.Text[len(opts.fields[0]) + 1:]
		}

		if opts.update.Message.ReplyToMessage.Audio != nil {
			SavedMessage.Item.Audio = append(SavedMessage.Item.Audio, SavedMessageTypeCachedAudio{
				ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
				FileID: opts.update.Message.ReplyToMessage.Audio.FileID,
				Description: DescriptionText,
				Caption: opts.update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.update.Message.ReplyToMessage.CaptionEntities,
			})
			SavedMessage.Count++
			SavedMessage.SavedTimes++
			database.Data.SavedMessage[userData.ID] = SavedMessage
			SignalsChannel.Database_save <- true
			messageParams.Text = "已保存音乐"
		} else if opts.update.Message.ReplyToMessage.Animation != nil {
			SavedMessage.Item.Mpeg4gif = append(SavedMessage.Item.Mpeg4gif, SavedMessageTypeCachedMpeg4Gif{
				ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
				FileID: opts.update.Message.ReplyToMessage.Animation.FileID,
				Title: opts.update.Message.ReplyToMessage.Caption,
				Description: DescriptionText,
				Caption: opts.update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.update.Message.ReplyToMessage.CaptionEntities,
			})
			SavedMessage.Count++
			SavedMessage.SavedTimes++
			database.Data.SavedMessage[userData.ID] = SavedMessage
			SignalsChannel.Database_save <- true
			messageParams.Text = "已保存 GIF"
		} else if opts.update.Message.ReplyToMessage.Document != nil {
			if opts.update.Message.ReplyToMessage.Document.MimeType == "image/gif" {
				SavedMessage.Item.Gif = append(SavedMessage.Item.Gif, SavedMessageTypeCachedGif{
					ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
					FileID: opts.update.Message.ReplyToMessage.Document.FileID,
					Description: DescriptionText,
					Caption: opts.update.Message.ReplyToMessage.Caption,
					CaptionEntities: opts.update.Message.ReplyToMessage.CaptionEntities,
				})
				SavedMessage.Count++
				SavedMessage.SavedTimes++
				database.Data.SavedMessage[userData.ID] = SavedMessage
				SignalsChannel.Database_save <- true
				messageParams.Text = "已保存 GIF (文件)"
			} else {
				SavedMessage.Item.Document = append(SavedMessage.Item.Document, SavedMessageTypeCachedDocument{
					ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
					FileID: opts.update.Message.ReplyToMessage.Document.FileID,
					Title: opts.update.Message.ReplyToMessage.Document.FileName,
					Description: opts.update.Message.ReplyToMessage.Caption,
					Caption: opts.update.Message.ReplyToMessage.Caption,
					CaptionEntities: opts.update.Message.ReplyToMessage.CaptionEntities,
				})
				SavedMessage.Count++
				SavedMessage.SavedTimes++
				database.Data.SavedMessage[userData.ID] = SavedMessage
				SignalsChannel.Database_save <- true
				messageParams.Text = "已保存文件"
			}
		} else if opts.update.Message.ReplyToMessage.Photo != nil {
			SavedMessage.Item.Photo = append(SavedMessage.Item.Photo, SavedMessageTypeCachedPhoto{
				ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
				FileID: opts.update.Message.ReplyToMessage.Photo[len(opts.update.Message.ReplyToMessage.Photo)-1].FileID,
				Title: opts.update.Message.ReplyToMessage.Caption,
				Description: DescriptionText,
				Caption: opts.update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.update.Message.ReplyToMessage.CaptionEntities,
				CaptionAboveMedia: opts.update.Message.ReplyToMessage.ShowCaptionAboveMedia,
			})
			SavedMessage.Count++
			SavedMessage.SavedTimes++
			database.Data.SavedMessage[userData.ID] = SavedMessage
			SignalsChannel.Database_save <- true
			messageParams.Text = "已保存图片"
		} else if opts.update.Message.ReplyToMessage.Sticker != nil {
			stickerSet, err := opts.thebot.GetStickerSet(opts.ctx, &bot.GetStickerSetParams{ Name: opts.update.Message.ReplyToMessage.Sticker.SetName })
			if err != nil {
				log.Printf("Error response /save command sticker no pack info: %v", err)
			}
			if stickerSet != nil {
				SavedMessage.Item.Sticker = append(SavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
					ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
					FileID: opts.update.Message.ReplyToMessage.Sticker.FileID,
					SetName: stickerSet.Name,
					SetTitle: stickerSet.Title,
					Description: DescriptionText,
				})
			} else {
				SavedMessage.Item.Sticker = append(SavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
					ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
					FileID: opts.update.Message.ReplyToMessage.Sticker.FileID,
					Description: DescriptionText,
				})
			}
			SavedMessage.Count++
			SavedMessage.SavedTimes++
			database.Data.SavedMessage[userData.ID] = SavedMessage
			SignalsChannel.Database_save <- true
			messageParams.Text = "已保存贴纸"
		} else if opts.update.Message.ReplyToMessage.Video != nil {
			if DescriptionText == "" {
				messageParams.Text = "保存失败：\n保存视频类型消息时需要一个描述\n请以 <code>/save 描述内容</code> 格式再次回复要收藏的视频"
				_, err := opts.thebot.SendMessage(opts.ctx, messageParams)
				if err != nil {
					log.Printf("Error response /save command video no description: %v", err)
				}
				return
			}

			SavedMessage.Item.Video = append(SavedMessage.Item.Video, SavedMessageTypeCachedVideo{
				ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
				FileID: opts.update.Message.ReplyToMessage.Video.FileID,
				Title: DescriptionText,
				Description: opts.update.Message.ReplyToMessage.Caption,
				Caption: opts.update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.update.Message.ReplyToMessage.CaptionEntities,
			})
			SavedMessage.Count++
			SavedMessage.SavedTimes++
			database.Data.SavedMessage[userData.ID] = SavedMessage
			SignalsChannel.Database_save <- true
			messageParams.Text = "已保存视频"
		} else if opts.update.Message.ReplyToMessage.Voice != nil {
			SavedMessage.Item.Voice = append(SavedMessage.Item.Voice, SavedMessageTypeCachedVoice{
				ID: fmt.Sprintf("%d", SavedMessage.SavedTimes),
				FileID: opts.update.Message.ReplyToMessage.Voice.FileID,
				Title: DescriptionText,
				Description: opts.update.Message.ReplyToMessage.Caption,
				Caption: opts.update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.update.Message.ReplyToMessage.CaptionEntities,
			})
			SavedMessage.Count++
			SavedMessage.SavedTimes++
			database.Data.SavedMessage[userData.ID] = SavedMessage
			SignalsChannel.Database_save <- true
			messageParams.Text = "已保存语音"
		} else {
			messageParams.Text = "暂不支持的消息类型"
		}
	} else {
		messageParams.Text = "在回复一条消息的同时发送 <code>/save</code> 来添加"
	}

	// fmt.Println(opts.chatInfo)

	_, err := opts.thebot.SendMessage(opts.ctx, messageParams)
	if err != nil {
		log.Printf("Error response /save command: %v", err)
	}
}

func InlineShowSavedMessageHandler(opts *subHandlerOpts) []models.InlineQueryResult {
	var InlineSavedMessageResultList []models.InlineQueryResult

	SavedMessage := database.Data.SavedMessage[opts.chatInfo.ID]
	if len(opts.fields) < 2 || len(opts.fields) == 2 && strings.HasPrefix(opts.fields[len(opts.fields)-1], InlinePaginationSymbol) {
		var all []models.InlineQueryResult
		for _, n := range SavedMessage.Item.All() {
			if n.audio != nil {
				all = append(all, n.audio)
			} else if n.document != nil {
				all = append(all, n.document)
			} else if n.gif != nil {
				all = append(all, n.gif)
			} else if n.photo != nil {
				all = append(all, n.photo)
			} else if n.sticker != nil {
				all = append(all, n.sticker)
			} else if n.video != nil {
				all = append(all, n.video)
			} else if n.voice != nil {
				all = append(all, n.voice)
			} else if n.mpeg4gif != nil {
				all = append(all, n.mpeg4gif)
			}
		}
		InlineSavedMessageResultList = all
	} else {
		var all []models.InlineQueryResult
		for _, n := range SavedMessage.Item.All() {
			if n.audio != nil && InlineQueryMatchMultKeyword(opts.fields, []string{n.audio.Caption, n.sharedData.Description}) {
				all = append(all, n.audio)
			} else if n.document != nil && InlineQueryMatchMultKeyword(opts.fields, []string{n.document.Title, n.document.Caption, n.document.Description}) {
				all = append(all, n.document)
			} else if n.gif != nil && InlineQueryMatchMultKeyword(opts.fields, []string{n.gif.Title, n.gif.Caption, n.sharedData.Description}) {
				all = append(all, n.gif)
			} else if n.photo != nil && InlineQueryMatchMultKeyword(opts.fields, []string{n.photo.Title, n.photo.Caption, n.photo.Description}) {
				all = append(all, n.photo)
			} else if n.sticker != nil && InlineQueryMatchMultKeyword(opts.fields, []string{n.sharedData.Title, n.sharedData.Name, n.sharedData.Description}) {
				all = append(all, n.sticker)
			} else if n.video != nil && InlineQueryMatchMultKeyword(opts.fields, []string{n.video.Title, n.video.Caption, n.video.Description}) {
				all = append(all, n.video)
			} else if n.voice != nil && InlineQueryMatchMultKeyword(opts.fields, []string{n.voice.Title, n.voice.Caption, n.sharedData.Description}) {
				all = append(all, n.voice)
			} else if n.mpeg4gif != nil && InlineQueryMatchMultKeyword(opts.fields, []string{n.mpeg4gif.Title, n.mpeg4gif.Caption, n.sharedData.Description}) {
				all = append(all, n.mpeg4gif)
			}
		}
		InlineSavedMessageResultList = all

		if len(InlineSavedMessageResultList) == 0 {
			InlineSavedMessageResultList = append(InlineSavedMessageResultList, &models.InlineQueryResultArticle{
				ID:       "none",
				Title:    "没有符合关键词的内容",
				Description: fmt.Sprintf("没有找到包含 %s 的内容", opts.fields[1:]),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}

	if len(InlineSavedMessageResultList) == 0 {
		_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:    "empty",
				Title: "没有保存内容（点击查看详细教程）",
				Description: "对一条信息回复 /save 来保存它",
				InputMessageContent: models.InputTextMessageContent{
					MessageText: fmt.Sprintf("您可以在任何聊天的输入栏中输入 <code>@%s +saved </code>来查看您的收藏\n若要添加，您需要确保机器人可以读取到您的指令，例如在群组中需要添加机器人，或点击 @%s 进入与机器人的聊天窗口，找到想要收藏的信息，然后对着那条信息回复 /save 即可\n若收藏成功，机器人会回复您并提示收藏成功，您也可以手动发送一条想要收藏的息，再使用 /save 命令回复它", botMe.Username, botMe.Username),
					ParseMode: models.ParseModeHTML,
				},
			}},
			Button: &models.InlineQueryResultsButton{
				Text: "点击此处快速跳转到机器人",
				StartParameter: "via-inline_noreply",
			},

		})
		if err != nil {
			log.Println("Error when answering inline [saved] command", err)
		}
	}

	_, err := opts.thebot.AnswerInlineQuery(opts.ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: opts.update.InlineQuery.ID,
		Results:       InlineResultPagination(opts.fields, InlineSavedMessageResultList),
		IsPersonal:    true,
	})
	if err != nil {
		log.Println("Error when answering inline [saved] command", err)
	}

	return InlineSavedMessageResultList
}

var SavedMessage_InlineCommandHandler = Plugin_Inline{
	command: "saved",
	handler: InlineShowSavedMessageHandler,
}


func SendPrivacyPolicy(opts *subHandlerOpts) {
	_, err := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
		ChatID: opts.update.Message.Chat.ID,
		Text: "<blockquote>目前此机器人仍在开发阶段中，此信息可能会有更改</blockquote>\n" +
			"本机器人提供收藏信息功能，您可以在回复一条信息时输入 /save 来收藏它，之后在 inline 模式下随时浏览您的收藏内容并发送\n\n" +

			"我们会记录哪些数据？\n" +
			"<blockquote>" +
			"1. 您的用户信息，例如 用户昵称、用户 ID、聊天类型（当您将此机器人添加到群组或频道中时）\n" +
			"2. 您的使用情况，例如 消息计数、inline 调用计数、inline 条目计数、最后向机器人发送的消息、callback_query、inline_query 以及选择的 inline 结果\n" +
			"3. 收藏信息内容，您需要注意这个，因为您是为了这个而阅读此内容，例如 存储的收藏信息数量、其图片上传到 Telegram 时的文件 ID、图片下方的文本，还有您在使用添加命令时所自定义的搜索关键词" +
			"</blockquote>\n\n" +

			"我的数据安全吗？\n" +
			"<blockquote>" +
			"这是一个早期的项目，还有很多未发现的 bug 与漏洞，因此您不能也不应该将敏感的数据存储在此机器人中，若您觉得我们收集的信息不妥，您可以不点击底部的同意按钮，我们仅会收集一些基本的信息，防止对机器人造成滥用，基本信息为前一段的 1 至 2 条目" +
			"</blockquote>\n\n" +

			"我收藏的消息，有谁可以看到?" +
			"<blockquote>" +
			"此功能被设计为每个人有单独的存储空间，如果您不手动从 inline 模式下选择信息并发送，其他用户是没法查看您的收藏列表的。不过，与上一个条目一样，为了防止滥用，我们是可以也有权利查看您收藏的内容的，请不要在其中保存隐私数据" +
			"</blockquote>\n\n" +

			"内容待补充...",
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击同意以上内容",
			URL: fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy_agree", botMe.Username),
		}}}},
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		log.Println("error when send savedmessage_privacy_policy:", err)
		return
	}
}

func AgreePrivacyPolicy(opts *subHandlerOpts) {
	var SavedMessages SavedMessage
		// , ok := database.Data.SavedMessage[opts.chatInfo.ID]
		if len(database.Data.SavedMessage) == 0 {
			database.Data.SavedMessage = make(map[int64]SavedMessage)
			// database.Data.SavedMessage[opts.chatInfo.ID] = SavedMessages
		}
		SavedMessages.AgreePrivacyPolicy = true
		database.Data.SavedMessage[opts.chatInfo.ID] = SavedMessages
		SignalsChannel.Database_save <- true
		_, err :=opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text: "您已成功开启收藏信息功能，回复一条信息的时候发送 /save 来使用收藏功能吧！\n由于服务器性能原因，每个人的收藏数量上限默认为 100 个，您也可以从机器人的个人信息中寻找管理员来申请更高的上限\n点击下方按钮来浏览您的收藏内容",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "点击浏览你的收藏",
				SwitchInlineQueryCurrentChat: InlineSubCommandSymbol + "saved ",
			}}}},
		})
		if err != nil {
			log.Println("error when send savedmessage_privacy_policy_agree:", err)
		}
}

var SavedMessage_StartCommandHandlers = []SlashStartHandler{
	{
		"savedmessage_privacy_policy",
		SendPrivacyPolicy,
	},
	{
		"savedmessage_privacy_policy_agree",
		AgreePrivacyPolicy,
	},
}

var SavedMessage_StartCommandWithPrefixHandlers = []SlashStartWithPrefixHandler{
	{
		"via-inline",
		"noreply",
		nil,
	},
	{
		"via-inline",
		"savedmessage-help",
		saveMessageHandler,
	},
}
