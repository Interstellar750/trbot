package common

import (
	"strings"

	"trle5.xyz/gopkg/trbot/utils/type/message_utils"
)

var ResultCategorys = InlineCategorys{
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
	result, isExist = ResultCategorys[strings.ToLower(str)]
	return
}
