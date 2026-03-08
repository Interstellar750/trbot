package plugins

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/flaterr"
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/plugin_utils"
	"trle5.xyz/trbot/utils/type/message_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var photoCachedDir string = filepath.Join(configs.CacheDir, "photo/")
var imageBaseURL   string = "https://alist.trle5.xyz/d/cache/photo/"

func init() {
	plugin_utils.AddHandlerByMessageTypeHandlers(plugin_utils.ByMessageTypeHandler{
		PluginName:       "获取搜图链接",
		ChatType:         models.ChatTypePrivate,
		MessageType:      message_utils.Photo,
		AllowAutoTrigger: true,
		MessageHandler:   searchImageHandler,
	})
	plugin_utils.AddSlashCommandHandlers(plugin_utils.SlashCommand{
		SlashCommand:   "searchlinks",
		MessageHandler: sendSearchLinks,
	})
}

type SearchEngines struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

var searchURLs = []SearchEngines{
	{
		Name: "Google",
		URL:  "https://www.google.com/searchbyimage?client=app&image_url=%s",
	},
	{
		Name: "Google Lens",
		URL:  "https://lens.google.com/uploadbyurl?url=%s",
	},
	{
		Name: "Bing",
		URL:  "https://www.bing.com/images/search?q=imgurl:%s&view=detailv2&iss=sbi",
	},
	{
		Name: "Yandex.ru",
		URL:  "https://yandex.ru/images/search?rpt=imageview&url=%s",
	},
	{
		Name: "Yandex.com",
		URL:  "https://yandex.com/images/search?rpt=imageview&url=%s",
	},
	{
		Name: "SauceNAO",
		URL:  "https://saucenao.com/search.php?url=%s",
	},
	{
		Name: "ascii2d",
		URL:  "https://ascii2d.net/search/url/%s",
	},
	{
		Name: "Tineye",
		URL:  "https://tineye.com/search?url=%s",
	},
	{
		Name: "trace.moe",
		URL:  "https://trace.moe/?auto&url=%s",
	},
	// {
	// 	Name: "IQDB",
	// 	URL:  "http://iqdb.org/?url=%s",
	// },
	// {
	// 	Name: "3D-IQDB",
	// 	URL:  "http://3d.iqdb.org/?url=%s",
	// },
}

// sendSearchLinks 用于响应 /searchlinks 命令
func sendSearchLinks(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "search_images").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Dict(utils.GetChatDict(&opts.Message.Chat)).
		Logger()

	wrap := flaterr.NewWrapper(logger.Error())

	if opts.Message.ReplyToMessage == nil || opts.Message.ReplyToMessage.Photo == nil {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Message.Chat.ID,
			Text:      "使用此命令回复一张图片来获得搜索链接",
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
		})
		wrap.ErrIf(err).MsgT(flaterr.SendMessage, "need reply to a photo")
	} else {
		photoPath, err := downloadPhoto(opts.Ctx, opts.Thebot, opts.Message.ReplyToMessage)
		if err != nil {
			wrap.Err(err).Msg("error when download photo")
			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Message.Chat.ID,
				Text:   fmt.Sprintf("缓存图片时发生错误: <blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ReplyToMessage.ID },
				ParseMode: models.ParseModeHTML,
			})
			wrap.ErrIf(err).MsgT(flaterr.SendMessage, "photo cache error")
		} else {
			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Message.Chat.ID,
				Text: "选择一个搜索图片的搜索引擎\n此功能灵感来源于 @soutubot",
				ReplyMarkup: buildSearchLinksKeboard(photoPath),
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ReplyToMessage.ID },
			})
			wrap.ErrIf(err).MsgT(flaterr.SendMessage, "search images link buttons")
		}
	}

	return wrap.Flat()
}

// searchImageHandler 作为 ByMessageTypeHandler 自动处理图片
func searchImageHandler(opts *handler_params.Message) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "search_images").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(opts.Message.From)).
		Logger()

	wrap := flaterr.NewWrapper(logger.Error())

	photoPath, err := downloadPhoto(opts.Ctx, opts.Thebot, opts.Message)
	if err != nil {
		wrap.Err(err).Str("path", photoPath).Msg("Error when cache photo")
		_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Message.Chat.ID,
			Text:   fmt.Sprintf("缓存图片时发生错误: <blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
			ParseMode: models.ParseModeHTML,
		})
		wrap.ErrIf(err).MsgT(flaterr.SendMessage, "photo cache error")
		return wrap.Flat()
	}

	_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Message.Chat.ID,
		Text: "选择一个搜索图片的搜索引擎\n此功能灵感来源于 @soutubot",
		ReplyMarkup: buildSearchLinksKeboard(photoPath),
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Message.ID },
	})
	wrap.ErrIf(err).MsgT(flaterr.SendMessage, "search images link buttons")

	return wrap.Flat()
}

func downloadPhoto(ctx context.Context, thebot *bot.Bot, msg *models.Message) (string, error) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "search_images").
		Str(utils.GetCurrentFuncName()).
		Logger()

	wrap := flaterr.NewWrapper(logger.Error())

	var photoFileName string = fmt.Sprintf("%d-%s.jpg", msg.From.ID, msg.Photo[len(msg.Photo)-1].FileID)
	var photoFullPath string = filepath.Join(photoCachedDir, photoFileName)

	_, err := os.Stat(photoFullPath) // 检查图片源文件是否已缓存
	if err != nil {
		// 如果图片源文件未缓存，则下载
		if os.IsNotExist(err) {
			fileInfo, err := thebot.GetFile(ctx, &bot.GetFileParams{
				FileID: msg.Photo[len(msg.Photo)-1].FileID,
			})
			if err != nil {
				wrap.Err(err).
					Str("fileID", msg.Photo[len(msg.Photo)-1].FileID).
					MsgT(flaterr.GetFile, "photo file")
				return "", wrap.Flat()
			} else {
				// 组合链接下载图片源文件
				resp, err := http.Get(thebot.FileDownloadLink(fileInfo))
				if err != nil {
					wrap.Err(err).
						Str("filePath", fileInfo.FilePath).
						Msg("Failed to download photo file")
					return "", wrap.Flat()
				}
				defer resp.Body.Close()

				// 创建目录
				err = os.MkdirAll(photoCachedDir, 0755)
				if err != nil {
					wrap.Err(err).
						Str("photoCachedDir", photoCachedDir).
						Msg("Failed to create directory to cache photo")
					return "", wrap.Flat()
				}

				downloadedPhoto, err := os.Create(photoFullPath)
				if err != nil {
					wrap.Err(err).
						Str("photoFullPath", photoFullPath).
						Msg("Failed to create photo file")
					return "", wrap.Flat()
				}
				defer downloadedPhoto.Close()

				// 将下载的原图片写入空文件
				_, err = io.Copy(downloadedPhoto, resp.Body)
				if err != nil {
					wrap.Err(err).
						Str("photoFullPath", photoFullPath).
						Msg("Failed to writing photo data to file")
					return "", wrap.Flat()
				}
			}
		} else {
			wrap.Err(err).
				Str("photoFullPath", photoFullPath).
				Msg("Failed to read cached photo file info")
			return "", wrap.Flat()
		}
	} else {
		// 文件已存在，跳过下载
		logger.Debug().
			Str("photoFullPath", photoFullPath).
			Msg("photo file already cached")
	}

	return photoFileName, nil
}

func buildSearchLinksKeboard(photoPath string) models.ReplyMarkup {
	var button     [][]models.InlineKeyboardButton
	var tempButton   []models.InlineKeyboardButton

	for _, url := range searchURLs {
		if len (tempButton) >= 3 {
			button = append(button, tempButton)
			tempButton = []models.InlineKeyboardButton{}
		}
		tempButton = append(tempButton, models.InlineKeyboardButton{
			Text: url.Name,
			URL:  fmt.Sprintf(url.URL, imageBaseURL + photoPath),
		})
	}

	if len(tempButton) > 0 { button = append(button, tempButton) }

	button = append(button, []models.InlineKeyboardButton{{
		Text:         "🚫 关闭菜单",
		CallbackData: "delete_this_message",
	}})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: button,
	}
}
