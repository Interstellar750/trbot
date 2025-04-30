package plugins

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_utils"
	"trbot/utils/plugin_utils"
	"unicode/utf8"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

var SavedMessageSet   map[int64]SavedMessage
var SavedMessageErr error

var SavedMessage_path string = consts.DB_path + "savedmessage/"

type SavedMessage struct {
	// ForUserID          int64 `yaml:"ForUserID"`
	Count              int   `yaml:"Count"`                // 当前存储的数量
	SavedTimes         int   `yaml:"SavedTimes,omitempty"` // 一共存过多少个
	Limit              int   `yaml:"Limit,omitempty"`
	AgreePrivacyPolicy bool  `yaml:"AgreePrivacyPolicy,omitempty"`

	Item SavedMessageItems `yaml:"Item,omitempty"`
}

func SaveSavedMessageList() error {
	data, err := yaml.Marshal(SavedMessageSet)
	if err != nil { return err }

	if _, err := os.Stat(SavedMessage_path); os.IsNotExist(err) {
		if err := os.MkdirAll(SavedMessage_path, 0755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(SavedMessage_path + consts.MetadataFileName); os.IsNotExist(err) {
		_, err := os.Create(SavedMessage_path + consts.MetadataFileName)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(SavedMessage_path + consts.MetadataFileName, data, 0644)
}

func ReadSavedMessageList() {
	var SavedMessages map[int64]SavedMessage

	file, err := os.Open(SavedMessage_path + consts.MetadataFileName)
	if err != nil {
		// 如果是找不到目录，新建一个
		log.Println("[SavedMessage]: Not found database file. Created new one")
		SaveSavedMessageList()
		SavedMessageSet, SavedMessageErr = map[int64]SavedMessage{}, err
		return
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&SavedMessages)
	if err != nil {
		if err == io.EOF {
			log.Println("[SavedMessage]: Saved Message list looks empty. now format it")
			SaveSavedMessageList()
			SavedMessageSet, SavedMessageErr = map[int64]SavedMessage{}, nil
			return
		}
		log.Println("(func)ReadSavedMessageList:", err)
		SavedMessageSet, SavedMessageErr = map[int64]SavedMessage{}, err
		return
	}
	SavedMessageSet, SavedMessageErr = SavedMessages, nil
}


type sortstruct struct {
	sharedData *SavedMessageSharedData // 存放一些标准列表里没有的数据，方便搜索

	onlyText  *models.InlineQueryResultArticle

	audio     *models.InlineQueryResultCachedAudio
	document  *models.InlineQueryResultCachedDocument
	gif       *models.InlineQueryResultCachedGif
	photo     *models.InlineQueryResultCachedPhoto
	sticker   *models.InlineQueryResultCachedSticker
	video     *models.InlineQueryResultCachedVideo
	videoNote *models.InlineQueryResultCachedDocument
	voice     *models.InlineQueryResultCachedVoice
	mpeg4gif  *models.InlineQueryResultCachedMpeg4Gif
}

type SavedMessageItems struct {
	OnlyText  []SavedMessageTypeCachedOnlyText  `yaml:"OnlyText,omitempty"`
	Audio     []SavedMessageTypeCachedAudio     `yaml:"Audio,omitempty"`
	Document  []SavedMessageTypeCachedDocument  `yaml:"Document,omitempty"`
	Gif       []SavedMessageTypeCachedGif       `yaml:"Gif,omitempty"`
	Photo     []SavedMessageTypeCachedPhoto     `yaml:"Photo,omitempty"`
	Sticker   []SavedMessageTypeCachedSticker   `yaml:"Sticker,omitempty"`
	Video     []SavedMessageTypeCachedVideo     `yaml:"Video,omitempty"`
	VideoNote []SavedMessageTypeCachedVideoNote `yaml:"VideoNote,omitempty"`
	Voice     []SavedMessageTypeCachedVoice     `yaml:"Voice,omitempty"`
	Mpeg4gif  []SavedMessageTypeCachedMpeg4Gif  `yaml:"Mpeg4Gif,omitempty"`
}

func (s *SavedMessageItems) All() []sortstruct {
	// var all []models.InlineQueryResult
	var list []sortstruct
	//  = make([]sortstruct, 100)
	for _, v := range s.OnlyText {
		index, err := strconv.Atoi(v.ID)
		if err != nil {
			log.Println("no an valid id", err)
			continue
		}
		if len(list) <= index {
			list = append(list, make([]sortstruct, index-len(list)+1)...)
		}
		if list[index].onlyText != nil {
			log.Println("duplicate id", v.ID)
			continue
		}
		// var pendingTitle string
		// if len(v.TitleAndMessageText) > 20 {
		// 	pendingTitle = v.TitleAndMessageText[:20] + "..."
		// }
		list[index].onlyText = &models.InlineQueryResultArticle{
			ID:                  v.ID,
			Title:               v.TitleAndMessageText,
			Description:         v.Description,
			InputMessageContent: models.InputTextMessageContent{
				MessageText:        v.TitleAndMessageText,
				Entities:           v.Entities,
				LinkPreviewOptions: v.LinkPreviewOptions,
			},
		}
	}
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
	for _, v := range s.VideoNote {
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
			ID:              v.ID,
			DocumentFileID:  v.FileID,
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
	FileName    string
	Description string
}

type SavedMessageTypeCachedOnlyText struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	ID                  string                     `yaml:"ID"`
	TitleAndMessageText string                     `yaml:"TitleAndMessageText"`
	Description         string                     `yaml:"Description,omitempty"`
	Entities            []models.MessageEntity     `yaml:"Entities,omitempty"`
	LinkPreviewOptions  *models.LinkPreviewOptions `yaml:"LinkPreviewOptions,omitempty"`
}

type SavedMessageTypeCachedAudio struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	Title       string `yaml:"Title,omitempty"`
	FileName    string `yaml:"FileName,omitempty"`
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
	Title           string                 `yaml:"Title"`                 // inline 标题
	Description     string                 `yaml:"Description,omitempty"` // inline 描述
	Caption         string                 `yaml:"Caption,omitempty"`     // 发送后图片携带的文本
	CaptionEntities []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
}

type SavedMessageTypeCachedVideoNote struct {
	IsDeleted bool `yaml:"IsDeleted,omitempty"`

	ID              string                 `yaml:"ID"`
	FileID          string                 `yaml:"FileID"`
	Title           string                 `yaml:"Title"`
	Description     string                 `yaml:"Description,omitempty"`
	Caption         string                 `yaml:"Caption,omitempty"` // 利用 bot 修改信息可以发出带文字的圆形视频，但是发送后不带文字
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

func saveMessageHandler(opts *handler_utils.SubHandlerOpts) {
	UserSavedMessage := SavedMessageSet[opts.Update.Message.From.ID]

	messageParams := &bot.SendMessageParams{
		ChatID: opts.Update.Message.Chat.ID,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
		ParseMode: models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击浏览您的收藏",
			SwitchInlineQueryCurrentChat: consts.InlineSubCommandSymbol + "saved ",
		}}}},
	}

	if !UserSavedMessage.AgreePrivacyPolicy {
		messageParams.Text = "此功能需要保存一些信息才能正常工作，在使用这个功能前，请先阅读一下我们会保存哪些信息"
		messageParams.ReplyMarkup = &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击查看",
			URL: fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy", consts.BotMe.Username),
		}}}}
		_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
		if err != nil {
			log.Printf("Error response /save command initial info: %v", err)
		}
		return
	}

	if UserSavedMessage.Limit == 0 && UserSavedMessage.Count == 0 {
		UserSavedMessage.Limit = 100
	}

	// 若不是初次添加，为 0 就是不限制
	if UserSavedMessage.Limit != 0 && UserSavedMessage.Count >= UserSavedMessage.Limit {
		messageParams.Text = "已达到限制，无法保存更多内容"
		_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
		if err != nil {
			log.Printf("Error response /save command reach limit: %v", err)
		}
		return
	}

	// var pendingMessage string
	if opts.Update.Message.ReplyToMessage == nil {
		messageParams.Text = "在回复一条消息的同时发送 <code>/save</code> 来添加"
	} else {
		var DescriptionText string
		// 获取使用命令保存时设定的描述
		if len(opts.Update.Message.Text) > len(opts.Fields[0]) + 1 {
			DescriptionText = opts.Update.Message.Text[len(opts.Fields[0]) + 1:]
		}

		if opts.Update.Message.ReplyToMessage.Text != "" {
			messageLength := utf8.RuneCountInString(opts.Update.Message.ReplyToMessage.Text)
			var pendingEntitites []models.MessageEntity = opts.Update.Message.ReplyToMessage.Entities
			if messageLength > 100 {
				pendingEntitites = append(pendingEntitites, models.MessageEntity{
					Type:   "expandable_blockquote",
					Offset: 0,
					Length: messageLength,
				})
			}

			UserSavedMessage.Item.OnlyText = append(UserSavedMessage.Item.OnlyText, SavedMessageTypeCachedOnlyText{
				ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
				TitleAndMessageText: opts.Update.Message.ReplyToMessage.Text,
				Description: DescriptionText,
				Entities: pendingEntitites,
				LinkPreviewOptions: opts.Update.Message.ReplyToMessage.LinkPreviewOptions,
			})
			UserSavedMessage.Count++
			UserSavedMessage.SavedTimes++
			SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
			SaveSavedMessageList()
			messageParams.Text = "已保存文本"
		} else if opts.Update.Message.ReplyToMessage.Audio != nil {
			UserSavedMessage.Item.Audio = append(UserSavedMessage.Item.Audio, SavedMessageTypeCachedAudio{
				ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
				FileID: opts.Update.Message.ReplyToMessage.Audio.FileID,
				Title: opts.Update.Message.ReplyToMessage.Audio.Title,
				FileName: opts.Update.Message.ReplyToMessage.Audio.FileName,
				Description: DescriptionText,
				Caption: opts.Update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.Update.Message.ReplyToMessage.CaptionEntities,
			})
			UserSavedMessage.Count++
			UserSavedMessage.SavedTimes++
			SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
			SaveSavedMessageList()
			messageParams.Text = "已保存音乐"
		} else if opts.Update.Message.ReplyToMessage.Animation != nil {
			UserSavedMessage.Item.Mpeg4gif = append(UserSavedMessage.Item.Mpeg4gif, SavedMessageTypeCachedMpeg4Gif{
				ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
				FileID: opts.Update.Message.ReplyToMessage.Animation.FileID,
				Title: opts.Update.Message.ReplyToMessage.Caption,
				Description: DescriptionText,
				Caption: opts.Update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.Update.Message.ReplyToMessage.CaptionEntities,
			})
			UserSavedMessage.Count++
			UserSavedMessage.SavedTimes++
			SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
			SaveSavedMessageList()
			messageParams.Text = "已保存 GIF"
		} else if opts.Update.Message.ReplyToMessage.Document != nil {
			if opts.Update.Message.ReplyToMessage.Document.MimeType == "image/gif" {
				UserSavedMessage.Item.Gif = append(UserSavedMessage.Item.Gif, SavedMessageTypeCachedGif{
					ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID: opts.Update.Message.ReplyToMessage.Document.FileID,
					Description: DescriptionText,
					Caption: opts.Update.Message.ReplyToMessage.Caption,
					CaptionEntities: opts.Update.Message.ReplyToMessage.CaptionEntities,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
				SaveSavedMessageList()
				messageParams.Text = "已保存 GIF (文件)"
			} else {
				UserSavedMessage.Item.Document = append(UserSavedMessage.Item.Document, SavedMessageTypeCachedDocument{
					ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID: opts.Update.Message.ReplyToMessage.Document.FileID,
					Title: opts.Update.Message.ReplyToMessage.Document.FileName,
					Description: DescriptionText,
					Caption: opts.Update.Message.ReplyToMessage.Caption,
					CaptionEntities: opts.Update.Message.ReplyToMessage.CaptionEntities,
				})
				UserSavedMessage.Count++
				UserSavedMessage.SavedTimes++
				SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
				SaveSavedMessageList()
				messageParams.Text = "已保存文件"
			}
		} else if opts.Update.Message.ReplyToMessage.Photo != nil {
			UserSavedMessage.Item.Photo = append(UserSavedMessage.Item.Photo, SavedMessageTypeCachedPhoto{
				ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
				FileID: opts.Update.Message.ReplyToMessage.Photo[len(opts.Update.Message.ReplyToMessage.Photo)-1].FileID,
				Title: opts.Update.Message.ReplyToMessage.Caption,
				Description: DescriptionText,
				Caption: opts.Update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.Update.Message.ReplyToMessage.CaptionEntities,
				CaptionAboveMedia: opts.Update.Message.ReplyToMessage.ShowCaptionAboveMedia,
			})
			UserSavedMessage.Count++
			UserSavedMessage.SavedTimes++
			SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
			SaveSavedMessageList()
			messageParams.Text = "已保存图片"
		} else if opts.Update.Message.ReplyToMessage.Sticker != nil {
			stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: opts.Update.Message.ReplyToMessage.Sticker.SetName })
			if err != nil {
				log.Printf("Error response /save command sticker no pack info: %v", err)
			}
			if stickerSet != nil {
				UserSavedMessage.Item.Sticker = append(UserSavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
					ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID: opts.Update.Message.ReplyToMessage.Sticker.FileID,
					SetName: stickerSet.Name,
					SetTitle: stickerSet.Title,
					Description: DescriptionText,
				})
			} else {
				UserSavedMessage.Item.Sticker = append(UserSavedMessage.Item.Sticker, SavedMessageTypeCachedSticker{
					ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
					FileID: opts.Update.Message.ReplyToMessage.Sticker.FileID,
					Description: DescriptionText,
				})
			}
			UserSavedMessage.Count++
			UserSavedMessage.SavedTimes++
			SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
			SaveSavedMessageList()
			messageParams.Text = "已保存贴纸"
		} else if opts.Update.Message.ReplyToMessage.Video != nil {
			if DescriptionText == "" {
				messageParams.Text = "保存失败：\n保存视频类型消息时需要一个描述\n请以 <code>/save 描述内容</code> 格式再次回复要收藏的视频"
				_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
				if err != nil {
					log.Printf("Error response /save command video no description: %v", err)
				}
				return
			}

			UserSavedMessage.Item.Video = append(UserSavedMessage.Item.Video, SavedMessageTypeCachedVideo{
				ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
				FileID: opts.Update.Message.ReplyToMessage.Video.FileID,
				Title: DescriptionText,
				Description: opts.Update.Message.ReplyToMessage.Caption,
				Caption: opts.Update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.Update.Message.ReplyToMessage.CaptionEntities,
			})
			UserSavedMessage.Count++
			UserSavedMessage.SavedTimes++
			SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
			SaveSavedMessageList()
			messageParams.Text = "已保存视频"
		} else if opts.Update.Message.ReplyToMessage.VideoNote != nil {
			UserSavedMessage.Item.VideoNote = append(UserSavedMessage.Item.VideoNote, SavedMessageTypeCachedVideoNote{
				ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
				FileID: opts.Update.Message.ReplyToMessage.VideoNote.FileID,
				Title: opts.Update.Message.ReplyToMessage.VideoNote.FileUniqueID,
				Description: DescriptionText,
			})
			UserSavedMessage.Count++
			UserSavedMessage.SavedTimes++
			SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
			SaveSavedMessageList()
			messageParams.Text = "已保存圆形录制视频"
		} else if opts.Update.Message.ReplyToMessage.Voice != nil {
			UserSavedMessage.Item.Voice = append(UserSavedMessage.Item.Voice, SavedMessageTypeCachedVoice{
				ID: fmt.Sprintf("%d", UserSavedMessage.SavedTimes),
				FileID: opts.Update.Message.ReplyToMessage.Voice.FileID,
				Title: DescriptionText,
				Description: opts.Update.Message.ReplyToMessage.Caption,
				Caption: opts.Update.Message.ReplyToMessage.Caption,
				CaptionEntities: opts.Update.Message.ReplyToMessage.CaptionEntities,
			})
			UserSavedMessage.Count++
			UserSavedMessage.SavedTimes++
			SavedMessageSet[opts.Update.Message.From.ID] = UserSavedMessage
			SaveSavedMessageList()
			messageParams.Text = "已保存语音"
		} else {
			messageParams.Text = "暂不支持的消息类型"
		}
	}

	// fmt.Println(opts.ChatInfo)

	_, err := opts.Thebot.SendMessage(opts.Ctx, messageParams)
	if err != nil {
		log.Printf("Error response /save command: %v", err)
	}
}

func InlineShowSavedMessageHandler(opts *handler_utils.SubHandlerOpts) []models.InlineQueryResult {
	var InlineSavedMessageResultList []models.InlineQueryResult

	SavedMessage := SavedMessageSet[opts.ChatInfo.ID]

	keywordFields := utils.InlineExtractKeywords(opts.Fields)

	if len(keywordFields) == 0 {
		var all []models.InlineQueryResult
		for _, n := range SavedMessage.Item.All() {
			if n.onlyText != nil {
				all = append(all, n.onlyText)
			} else if n.audio != nil {
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
			} else if n.videoNote != nil {
				all = append(all, n.videoNote)
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
			if n.onlyText != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.onlyText.Description, n.onlyText.Title}) {
				all = append(all, n.onlyText)
			} else if n.audio != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.audio.Caption, n.sharedData.Description}) {
				all = append(all, n.audio)
			} else if n.document != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.document.Title, n.document.Caption, n.document.Description}) {
				all = append(all, n.document)
			} else if n.gif != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.gif.Title, n.gif.Caption, n.sharedData.Description}) {
				all = append(all, n.gif)
			} else if n.photo != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.photo.Title, n.photo.Caption, n.photo.Description}) {
				all = append(all, n.photo)
			} else if n.sticker != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.sharedData.Title, n.sharedData.Name, n.sharedData.Description}) {
				all = append(all, n.sticker)
			} else if n.video != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.video.Title, n.video.Caption, n.video.Description}) {
				all = append(all, n.video)
			} else if n.videoNote != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.videoNote.Title, n.videoNote.Caption, n.videoNote.Description}) {
				all = append(all, n.videoNote)
			} else if n.voice != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.voice.Title, n.voice.Caption, n.sharedData.Description}) {
				all = append(all, n.voice)
			} else if n.mpeg4gif != nil && utils.InlineQueryMatchMultKeyword(keywordFields, []string{n.mpeg4gif.Title, n.mpeg4gif.Caption, n.sharedData.Description}) {
				all = append(all, n.mpeg4gif)
			}
		}
		InlineSavedMessageResultList = all

		if len(InlineSavedMessageResultList) == 0 {
			InlineSavedMessageResultList = append(InlineSavedMessageResultList, &models.InlineQueryResultArticle{
				ID:       "none",
				Title:    "没有符合关键词的内容",
				Description: fmt.Sprintf("没有找到包含 %s 的内容", keywordFields),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}

	if len(InlineSavedMessageResultList) == 0 {
		_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
			InlineQueryID: opts.Update.InlineQuery.ID,
			Results:       []models.InlineQueryResult{&models.InlineQueryResultArticle{
				ID:    "empty",
				Title: "没有保存内容（点击查看详细教程）",
				Description: "对一条信息回复 /save 来保存它",
				InputMessageContent: models.InputTextMessageContent{
					MessageText: fmt.Sprintf("您可以在任何聊天的输入栏中输入 <code>@%s +saved </code>来查看您的收藏\n若要添加，您需要确保机器人可以读取到您的指令，例如在群组中需要添加机器人，或点击 @%s 进入与机器人的聊天窗口，找到想要收藏的信息，然后对着那条信息回复 /save 即可\n若收藏成功，机器人会回复您并提示收藏成功，您也可以手动发送一条想要收藏的息，再使用 /save 命令回复它", consts.BotMe.Username, consts.BotMe.Username),
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

	_, err := opts.Thebot.AnswerInlineQuery(opts.Ctx, &bot.AnswerInlineQueryParams{
		InlineQueryID: opts.Update.InlineQuery.ID,
		Results:       utils.InlineResultPagination(opts.Fields, InlineSavedMessageResultList),
		IsPersonal:    true,
	})
	if err != nil {
		log.Println("Error when answering inline [saved] command", err)
	}

	return InlineSavedMessageResultList
}

func SendPrivacyPolicy(opts *handler_utils.SubHandlerOpts) {
	_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Update.Message.Chat.ID,
		Text: "目前此机器人仍在开发阶段中，此信息可能会有更改\n" +
			"<blockquote expandable>本机器人提供收藏信息功能，您可以在回复一条信息时输入 /save 来收藏它，之后在 inline 模式下随时浏览您的收藏内容并发送\n\n" +

			"我们会记录哪些数据？\n" +
			"1. 您的用户信息，例如 用户昵称、用户 ID、聊天类型（当您将此机器人添加到群组或频道中时）\n" +
			"2. 您的使用情况，例如 消息计数、inline 调用计数、inline 条目计数、最后向机器人发送的消息、callback_query、inline_query 以及选择的 inline 结果\n" +
			"3. 收藏信息内容，您需要注意这个，因为您是为了这个而阅读此内容，例如 存储的收藏信息数量、其图片上传到 Telegram 时的文件 ID、图片下方的文本，还有您在使用添加命令时所自定义的搜索关键词" +
			"\n\n" +

			"我的数据安全吗？\n" +
			"这是一个早期的项目，还有很多未发现的 bug 与漏洞，因此您不能也不应该将敏感的数据存储在此机器人中，若您觉得我们收集的信息不妥，您可以不点击底部的同意按钮，我们仅会收集一些基本的信息，防止对机器人造成滥用，基本信息为前一段的 1 至 2 条目" +
			"\n\n" +

			"我收藏的消息，有谁可以看到?\n" +
			"此功能被设计为每个人有单独的存储空间，如果您不手动从 inline 模式下选择信息并发送，其他用户是没法查看您的收藏列表的。不过，与上一个条目一样，为了防止滥用，我们是可以也有权利查看您收藏的内容的，请不要在其中保存隐私数据" +
			"</blockquote>" +

			"\n\n" +
			"内容待补充...",
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
			Text: "点击同意以上内容",
			URL: fmt.Sprintf("https://t.me/%s?start=savedmessage_privacy_policy_agree", consts.BotMe.Username),
		}}}},
		ParseMode: models.ParseModeHTML,
	})
	if err != nil {
		log.Println("error when send savedmessage_privacy_policy:", err)
		return
	}
}

func AgreePrivacyPolicy(opts *handler_utils.SubHandlerOpts) {
	var UserSavedMessage SavedMessage
		// , ok := consts.Database.Data.SavedMessage[opts.ChatInfo.ID]
		if len(SavedMessageSet) == 0 {
			SavedMessageSet = map[int64]SavedMessage{}
			// consts.Database.Data.SavedMessage[opts.ChatInfo.ID] = SavedMessages
		}
		UserSavedMessage.AgreePrivacyPolicy = true
		SavedMessageSet[opts.ChatInfo.ID] = UserSavedMessage
		SaveSavedMessageList()
		_, err :=opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			Text: "您已成功开启收藏信息功能，回复一条信息的时候发送 /save 来使用收藏功能吧！\n由于服务器性能原因，每个人的收藏数量上限默认为 100 个，您也可以从机器人的个人信息中寻找管理员来申请更高的上限\n点击下方按钮来浏览您的收藏内容",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "点击浏览您的收藏",
				SwitchInlineQueryCurrentChat: consts.InlineSubCommandSymbol + "saved ",
			}}}},
		})
		if err != nil {
			log.Println("error when send savedmessage_privacy_policy_agree:", err)
		}
}

func init() {
	ReadSavedMessageList()
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Saved Message",
		Saver:  SaveSavedMessageList,
		Loader: ReadSavedMessageList,
	})
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "save",
		Handler:      saveMessageHandler,
	})
	plugin_utils.AddInlineHandlerPlugins(plugin_utils.InlineHandler{
		Command:     "saved",
		Handler:     InlineShowSavedMessageHandler,
		Description: "显示自己保存的消息",
	})
	plugin_utils.AddSlashStartCommandPlugins([]plugin_utils.SlashStartHandler{
		{
			Argument: "savedmessage_privacy_policy",
			Handler:  SendPrivacyPolicy,
		},
		{
			Argument: "savedmessage_privacy_policy_agree",
			Handler:  AgreePrivacyPolicy,
		},
	}...)
	plugin_utils.AddSlashStartWithPrefixCommandPlugins(plugin_utils.SlashStartWithPrefixHandler{
		Prefix:   "via-inline",
		Argument: "savedmessage-help",
		Handler:  saveMessageHandler,
	})
	plugin_utils.AddHandlerHelpInfo(plugin_utils.HandlerHelp{
		Name:        "收藏消息",
		Description: "此功能可以收藏用户指定的消息，之后使用 inline 模式查看并发送保存的内容\n\n保存消息：\n向机器人发送要保存的消息，然后使用 <code>/save 关键词</code> 命令回复要保存的消息，关键词可以忽略。若机器人在群组中，也可以直接使用 <code>/save 关键词</code> 命令回复要保存的消息。\n\n发送保存的消息：点击下方的按钮来使用 inline 模式，当您多次在 inline 模式下使用此 bot 时，在输入框中输入 <code>@</code> 即可看到 bot 会出现在列表中",
		ParseMode:   models.ParseModeHTML,
		ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{
			{{
				Text: "点击浏览您的收藏",
				SwitchInlineQueryCurrentChat: consts.InlineSubCommandSymbol + "saved ",
			}},
			{{
				Text: "将此功能设定为您的默认 inline 命令",
				CallbackData: "inline_default_noedit_saved",
			}},
		}},
	})
}
