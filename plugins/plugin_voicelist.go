package plugins

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_structs"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

var VoiceLists   []VoicePack
var VoiceListErr error

var VoiceList_path string = consts.DB_path + "voices/"

func init() {
	ReadVoicePackFromPath()
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name: "Voice List",
		Loader: ReadVoicePackFromPath,
	})
	plugin_utils.AddInlineHandlerPlugins(plugin_utils.InlineHandler{
		Command: "voice",
		Handler: VoiceListHandler,
		Description: "一些语音列表",
	})
}

type VoicePack struct {
	Name string `yaml:"name,omitempty"` // 语音包名称
	Voices []struct {
		ID       string `yaml:"ID,omitempty"`       // 语音 ID
		Title    string `yaml:"Title,omitempty"`    // 行内模式时显示的标题
		Caption  string `yaml:"Caption,omitempty"`  // 发送后在语音下方的文字
		VoiceURL string `yaml:"VoiceURL,omitempty"` // 音频文件网络链接
	} `yaml:"voices,omitempty"`
}

// 读取指定目录下所有结尾为 .yaml 或 .yml 的语音文件
func ReadVoicePackFromPath() {
	var packs []VoicePack

	if _, err := os.Stat(VoiceList_path); os.IsNotExist(err) {
		log.Printf("No voices dir, create a new one: %s", VoiceList_path)
		if err := os.MkdirAll(VoiceList_path, 0755); err != nil {
			VoiceLists, VoiceListErr = nil, err
			return 
		}
	}

	err := filepath.Walk(VoiceList_path, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		if strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml") {
			file, err := os.Open(path)
			if err != nil { log.Println("(func)readVoicesFromDir:", err) }
			defer file.Close()

			var singlePack VoicePack
			decoder := yaml.NewDecoder(file)
			err = decoder.Decode(&singlePack)
			if err != nil { log.Println("(func)readVoicesFromDir:", err) }
			packs = append(packs, singlePack)
		}
		return nil
	})
	if err != nil {
		VoiceLists, VoiceListErr = nil, err
		return
	}

	VoiceLists, VoiceListErr = packs, nil
}

func VoiceListHandler(opts *handler_structs.SubHandlerParams) []models.InlineQueryResult {
	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult

	if VoiceLists == nil {
		log.Printf("No voices file in voices_path: %s", VoiceList_path)
		opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    consts.LogChat_ID,
			Text:      fmt.Sprintf("%s\nInline Mode: some user can't load voices", time.Now().Format(time.RFC3339)),
		})
		return []models.InlineQueryResult{&models.InlineQueryResultVoice{
			ID:       "none",
			Title:    "无法读取到语音文件，请联系机器人管理员",
			Caption:  "由于无法读取到语音文件，此处被替换为预设的 `♿otto: 我是说的道理~` ",
			VoiceURL: "https://alist.trle5.xyz/d/voices/otto/我是说的道理.ogg",
			ParseMode: models.ParseModeMarkdownV1,
		}}
	}

	keywordFields := utils.InlineExtractKeywords(opts.Fields)

	// 没有查询字符串或使用分页搜索符号，返回所有结果
	if len(keywordFields) == 0 {
		for _, voicePack := range VoiceLists {
			for _, voice := range voicePack.Voices {
				results = append(results, &models.InlineQueryResultVoice{
					ID:       voice.ID,
					Title:    voicePack.Name + ": " + voice.Title,
					Caption:  voice.Caption,
					VoiceURL: voice.VoiceURL,
				})
			}
		}
	} else {
		for _, voicePack := range VoiceLists {
			for _, voice := range voicePack.Voices {
				if utils.InlineQueryMatchMultKeyword(keywordFields, []string{voicePack.Name, voice.Title, voice.Caption}) {
					results = append(results, &models.InlineQueryResultVoice{
						ID:       voice.ID,
						Title:    voicePack.Name + ": " + voice.Title,
						Caption:  voice.Caption,
						VoiceURL: voice.VoiceURL,
					})
				}
			}
		}
		if len(results) == 0 {
			results = append(results, &models.InlineQueryResultArticle{
				ID:    "none",
				Title: "没有符合关键词的内容",
				Description: fmt.Sprintf("没有找到包含 %s 的内容", keywordFields),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}

	if VoiceListErr != nil {
		return append([]models.InlineQueryResult{&models.InlineQueryResultArticle{
			ID:       "none",
			Title:    "读取语音文件时发生错误，请联系机器人管理员",
			Description: "点此显示错误信息",
			InputMessageContent: models.InputTextMessageContent{
				MessageText: fmt.Sprintf("读取语音文件时发生错误<blockquote expandable>%s</blockquote>", VoiceListErr),
				ParseMode: models.ParseModeHTML,
			},
		}}, results...)
	}

	return results
}
