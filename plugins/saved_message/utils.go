package saved_message

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/consts"
	"trbot/utils/plugin_utils"
	"trbot/utils/type/message_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
	"github.com/rs/zerolog"
)

var SavedMessageList SavedMessage
var SavedMessageErr  error
var SavedMessagePath string = filepath.Join(consts.YAMLDataBaseDir, "savedmessage/", consts.YAMLFileName)

var meilisearchClient meilisearch.ServiceManager

var textExpandableLength int = 150

var stateMessageID   int
var editingMessageID int // 用于编辑消息时的消息ID

type SavedMessage struct {
	MeiliURL        string `yaml:"MeiliURL"`
	MeiliAPI        string `yaml:"MeiliAPI"`
	ChannelID       int64  `yaml:"ChannelID"`
	ChannelUsername string `yaml:"ChannelUsername"`
	NoticeChatID    int64  `yaml:"NoticeChatID"`

	User []SavedMessageUser `yaml:"User"`
}

func (sm *SavedMessage) ChannelIDStr() string {
	return strconv.FormatInt(sm.ChannelID, 10)
}

func (sm *SavedMessage) GetUser(userID int64) *SavedMessageUser {
	for i, user := range sm.User {
		if user.UserID == userID {
			return &sm.User[i]
		}
	}
	return nil
}

type SavedMessageUser struct {
	UserID     int64 `yaml:"UserID"`
	Count      int   `yaml:"Count"`                // 当前存储的数量
	SavedTimes int   `yaml:"SavedTimes,omitempty"` // 一共存过多少个
	Limit      int   `yaml:"Limit,omitempty"`

	IncludeChannel bool `yaml:"IncludeChannel,omitempty"` // 是否包含公开的消息
	DropOriginInfo bool `yaml:"DropOriginInfo,omitempty"` // 是否抛弃消息来源
}

func (u *SavedMessageUser) IDStr() string {
	return strconv.FormatInt(u.UserID, 10)
}

func (u *SavedMessageUser) buildUserConfigButtons() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	if SavedMessageList.ChannelUsername != "" {
		buttons = append(buttons, []models.InlineKeyboardButton{
			{
				Text: utils.TextForTrueOrFalse(u.IncludeChannel, "✅ 包含公共频道内容", "❌ 不包含公共频道内容"),
				CallbackData: "savedmsg_switch_include_channel",
			},
			{
				Text: "查看公共频道",
				URL: "https://t.me/" + SavedMessageList.ChannelUsername,
			},
		})
	}

	buttons = append(buttons, []models.InlineKeyboardButton{
		{
			Text: utils.TextForTrueOrFalse(u.DropOriginInfo, "❌ 不保留消息来源", "✅ 保留消息来源"),
			CallbackData: "savedmsg_switch_drop_origin_info",
		},
	})

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "关闭",
		CallbackData: "delete_this_message",
	}})

	return &models.InlineKeyboardMarkup{ InlineKeyboard: buttons }
}


func SaveSavedMessageList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.SaveYAML(SavedMessagePath, &SavedMessageList)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", SavedMessagePath).
			Msg("Failed to save savedmessage list")
		SavedMessageErr = fmt.Errorf("failed to save savedmessage list: %w", err)
	} else {
		SavedMessageErr = nil
	}

	return SavedMessageErr
}

func ReadSavedMessageList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(SavedMessagePath, &SavedMessageList)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", SavedMessagePath).
				Msg("Not found savedmessage list file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(SavedMessagePath, &SavedMessageList)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", SavedMessagePath).
					Msg("Failed to create empty savedmessage list file")
				SavedMessageErr = fmt.Errorf("failed to create empty savedmessage list file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", SavedMessagePath).
				Msg("Failed to load savedmessage list file")
			SavedMessageErr = fmt.Errorf("failed to load savedmessage list file: %w", err)
		}
	} else {
		SavedMessageErr = nil
	}

	if SavedMessageList.NoticeChatID == 0 && len(configs.BotConfig.AdminIDs) > 0 {
		SavedMessageList.NoticeChatID = configs.BotConfig.AdminIDs[0]
	}

	buildSavedMessageByMessageHandlers()
	return SavedMessageErr
}

func buildSavedMessageByMessageHandlers() {
	msgTypeList := []message_utils.Type{
		message_utils.Text,
		message_utils.Audio,
		message_utils.Animation,
		message_utils.Document,
		message_utils.Photo,
		message_utils.Sticker,
		message_utils.Video,
		message_utils.VideoNote,
		message_utils.Voice,
	}

	for _, user := range SavedMessageList.User {
		for _, msgType := range msgTypeList {
			plugin_utils.RemoveHandlerByMessageTypeHandler(
				models.ChatTypePrivate,
				msgType,
				user.UserID,
				"保存消息到收藏夹",
			)
		}
	}
	for _, user := range SavedMessageList.User {
		for _, msgType := range msgTypeList {
			plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
				PluginName:     "保存消息到收藏夹",
				ChatType:       models.ChatTypePrivate,
				ForChatID:      user.UserID,
				MessageType:    msgType,
				MessageHandler: saveMessageFromCallbackQueryHandler,
			})
		}
	}
}

var resultCategorys = InlineCategorys{
	"gif":       message_utils.Animation,
	"text":      message_utils.Text,
	"audio":     message_utils.Audio,
	"document":  message_utils.Document,
	"photo":     message_utils.Photo,
	"sticker":   message_utils.Sticker,
	"video":     message_utils.Video,
	"videonote": message_utils.VideoNote,
	"voice":     message_utils.Voice,
}

type InlineCategorys map[string]message_utils.Type

func (ic InlineCategorys) StrList() (list []string) {
	for key := range ic {
		list = append(list, key)
	}
	return list
}

func (ic InlineCategorys) GetCategory(str string) (result message_utils.Type, isExist bool) {
	result, isExist = resultCategorys[strings.ToLower(str)]
	return
}
