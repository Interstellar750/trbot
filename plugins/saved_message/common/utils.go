package common

import (
	"path/filepath"
	"strconv"
	"unicode/utf16"

	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/configs"

	"github.com/go-telegram/bot/models"
	"github.com/meilisearch/meilisearch-go"
)

var SavedMessageList SavedMessage
var SavedMessageErr  error
var SavedMessagePath string = filepath.Join(configs.YAMLDatabaseDir, "savedmessage/", configs.YAMLFileName)

var MeilisearchClient meilisearch.ServiceManager

var TextExpandableLength int = 150

type SavedMessage struct {
	MeiliURL        string `yaml:"MeiliURL"`
	MeiliAPI        string `yaml:"MeiliAPI"`
	ChannelID       int64  `yaml:"ChannelID"`
	ChannelUsername string `yaml:"ChannelUsername"`
	NoticeChatID    int64  `yaml:"NoticeChatID"`
	AllowUserSave   bool   `yaml:"AllowUserSave"`

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

	DropOriginInfo bool `yaml:"DropOriginInfo,omitempty"` // 是否抛弃消息来源
	UseQuickSave   bool `yaml:"UseQuickSave,omitempty"`   // 是否包含公开的消息
}

func (u *SavedMessageUser) IDStr() string {
	return strconv.FormatInt(u.UserID, 10)
}

func (u *SavedMessageUser) ConfigButtons() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: utils.TextForTrueOrFalse(u.UseQuickSave, "✅ 显示快速保存按钮", "❌ 不显示快速保存按钮"),
		CallbackData: "savedmsg_switch_use_quick_save",
	}})

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: utils.TextForTrueOrFalse(u.DropOriginInfo, "❌ 不保留消息来源", "✅ 保留消息来源"),
		CallbackData: "savedmsg_switch_drop_origin_info",
	}})

	buttons = append(buttons, []models.InlineKeyboardButton{{
		Text: "关闭",
		CallbackData: "delete_this_message",
	}})

	return &models.InlineKeyboardMarkup{ InlineKeyboard: buttons }
}

func UTF16Length(s string) int {
	// 将字符串转为 rune 切片
	runes := []rune(s)
	// 转为 UTF-16 单元序列
	encoded := utf16.Encode(runes)
	// 返回长度
	return len(encoded)
}
