package plugins

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"trbot/utils/consts"
	"trbot/utils/handler_params"
	"trbot/utils/inline_utils"
	"trbot/utils/plugin_utils"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var VoiceLists   []VoicePack
var VoiceListErr error
var VoiceListDir string = filepath.Join(consts.YAMLDataBaseDir, "voices/")

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "VoiceList",
		Func: ReadVoicePackFromPath,
	})
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "Voice List",
		Loader: ReadVoicePackFromPath,
	})
	plugin_utils.AddInlineHandlers(plugin_utils.InlineHandler{
		Command:       "voice",
		InlineHandler: VoiceListHandler,
		Description:   "一些语音列表",
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
func ReadVoicePackFromPath(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Voice List").
		Str("funcName", "ReadVoicePackFromPath").
		Logger()

	var packs []VoicePack

	_, err := os.Stat(VoiceListDir)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Str("directory", VoiceListDir).
				Msg("VoiceList directory not exist, now create it")
			err = os.MkdirAll(VoiceListDir, 0755)
			if err != nil {
				logger.Error().
					Err(err).
					Str("directory", VoiceListDir).
					Msg("Failed to create VoiceList data directory")
				VoiceListErr = err
				return err
			}
		} else {
			logger.Error().
				Err(err).
				Str("directory", VoiceListDir).
				Msg("Open VoiceList data directory failed")
			VoiceListErr = err
			return err
		}
	}


	err = filepath.Walk(VoiceListDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			logger.Error().
				Err(err).
				Str("path", path).
				Msg("Failed to read file use `filepath.Walk()`")
		}
		if strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml") {
			var singlePack VoicePack

			err = yaml.LoadYAML(path, &singlePack)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", path).
					Msg("Failed to decode file use `yaml.NewDecoder()`")
			}
			packs = append(packs, singlePack)
		}
		return nil
	})
	if err != nil {
		logger.Error().
			Err(err).
			Str("directory", VoiceListDir).
			Msg("Failed to read voice packs in VoiceList directory")
		VoiceListErr = err
		return err
	}

	VoiceLists = packs
	return nil
}

func VoiceListHandler(opts *handler_params.InlineQuery) []models.InlineQueryResult {
	// 将 metadata 转换为 Inline Query 结果
	var results []models.InlineQueryResult

	if VoiceLists == nil {
		zerolog.Ctx(opts.Ctx).Warn().
			Str("pluginName", "Voice List").
			Str("funcName", "VoiceListHandler").
			Str("VoiceListDir", VoiceListDir).
			Msg("No voices file in VoiceListDir")

		return []models.InlineQueryResult{&models.InlineQueryResultVoice{
			ID:       "none",
			Title:    "无法读取到语音文件，请联系机器人管理员",
			Caption:  "由于无法读取到语音文件，此处被替换为预设的 `♿otto: 我是说的道理~` ",
			VoiceURL: "https://alist.trle5.xyz/d/voices/otto/我是说的道理.ogg",
			ParseMode: models.ParseModeMarkdownV1,
		}}
	}

	keywordFields := inline_utils.ParseInlineFields(opts.Fields).Keywords

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
				if inline_utils.MatchMultKeyword(keywordFields, []string{voicePack.Name, voice.Title, voice.Caption}) {
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
				ID:                  "none",
				Title:               "没有符合关键词的内容",
				Description:         fmt.Sprintf("没有找到包含 %s 的内容", keywordFields),
				InputMessageContent: &models.InputTextMessageContent{
					MessageText: "用户在找不到想看的东西时无奈点击了提示信息...",
					ParseMode:    models.ParseModeMarkdownV1,
				},
			})
		}
	}

	if VoiceListErr != nil {
		return append([]models.InlineQueryResult{&models.InlineQueryResultArticle{
			ID:                  "none",
			Title:               "读取语音文件时发生错误，请联系机器人管理员",
			Description:         "点此显示错误信息",
			InputMessageContent: &models.InputTextMessageContent{
				MessageText: fmt.Sprintf("读取语音文件时发生错误<blockquote expandable>%s</blockquote>", VoiceListErr),
				ParseMode:   models.ParseModeHTML,
			},
		}}, results...)
	}

	return results
}
