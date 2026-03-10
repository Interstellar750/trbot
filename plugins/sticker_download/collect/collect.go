package collect

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"trle5.xyz/trbot/plugins/sticker_download/config"
	"trle5.xyz/trbot/plugins/sticker_download/download"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/flaterr"
	"trle5.xyz/trbot/utils/handler_params"
	"trle5.xyz/trbot/utils/type/contain"
	"trle5.xyz/trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var stickerCollect     collectedSticker
var stickerCollectPath string = filepath.Join(configs.YAMLDatabaseDir, "collectsticker/", configs.YAMLFileName)

type stickerSetInfo struct {
	Title string `yaml:"Title"` // 贴纸包的名称
	Name  string `yaml:"Name"`  // 贴纸包的 urlname
	MsgID int    `yaml:"MsgID"` // 发送到频道的消息 ID
	Count int    `yaml:"Count"` // 贴纸包中的贴纸数量
}

type collectedSticker struct {
	ChannelID  int64            `yaml:"ChannelID"`
	StickerSet []stickerSetInfo `yaml:"StickerSet"` // 已收藏的贴纸包列表
}

func (cs collectedSticker) GetStickerSetByName(name string) *stickerSetInfo {
	for i, set := range cs.StickerSet {
		if set.Name == name {
			return &cs.StickerSet[i]
		}
	}
	return nil
}

func ReadCollectStickerList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "sticker").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(stickerCollectPath, &stickerCollect)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", stickerCollectPath).
				Msg("Not found collect sticker list file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(stickerCollectPath, &stickerCollect)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", stickerCollectPath).
					Msg("Failed to create empty collect sticker list file")
				return fmt.Errorf("failed to create empty collect sticker list file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", stickerCollectPath).
				Msg("Failed to load collect sticker list file")
			return fmt.Errorf("failed to load collect sticker list file: %w", err)
		}
	}

	if stickerCollect.ChannelID == 0 {
		return errors.New("channelID is empty")
	}

	return nil
}

func SaveCollectStickerList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "sticker").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.SaveYAML(stickerCollectPath, &stickerCollect)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", stickerCollectPath).
			Msg("Failed to save collect sticker list")
		return fmt.Errorf("failed to save collect sticker list: %w", err)
	}
	return nil
}

func CollectStickerSet(opts *handler_params.CallbackQuery) error {
	logger := zerolog.Ctx(opts.Ctx).
		With().
		Str("pluginName", "StickerDownload").
		Str(utils.GetCurrentFuncName()).
		Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
		Str("callbackQueryData", opts.CallbackQuery.Data).
		Logger()

	var handlerErr flaterr.MultErr

	if stickerCollect.ChannelID == 0 {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "未设置贴纸包收集频道",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "collect channel ID not set").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "collect channel ID not set", err)
		}
	} else if !contain.Int64(opts.CallbackQuery.From.ID, configs.BotConfig.AdminIDs...) {
		_, err := opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.CallbackQuery.ID,
			Text:            "只有管理员才能使用此功能",
			ShowAlert:       true,
		})
		if err != nil {
			logger.Error().
				Err(err).
				Str("content", "only admin can use this function").
				Msg(flaterr.AnswerCallbackQuery.Str())
			handlerErr.Addt(flaterr.AnswerCallbackQuery, "only admin can use this function", err)
		}
	} else {
		stickerSetName := strings.TrimPrefix(opts.CallbackQuery.Data, "c_")

		// 有下划线开头即代表这是一个超出了长度的贴纸包，下划线后面的数据是一个 ID
		// 解析这个 ID 并从 config.Config.OversizeSets 中拿到对应的贴纸包名称
		if stickerSetName[0:1] == "_" {
			setID, err := strconv.Atoi(strings.TrimPrefix(stickerSetName, "_"))
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to parser oversize sticker set ID")
				handlerErr.Addf("failed to parser oversize sticker set ID [%s]: %w", stickerSetName, err)
				_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "解析贴纸包 ID 失败，请重新尝试发送此贴纸再点击按钮",
					ShowAlert:       true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "failed to parser oversize sticker set ID notice").
						Msg(flaterr.AnswerCallbackQuery.Str())
					handlerErr.Addt(flaterr.AnswerCallbackQuery, "failed to parser oversize sticker set ID notice", err)
				}
				return handlerErr.Flat()
			}
			stickerSetName = config.Config.GetOversizeSetNameByID(setID)
			if stickerSetName == "" {
				logger.Error().
					Int("setID", setID).
					Msg("Failed to find oversize sticker set")
				handlerErr.Addf("Failed to find oversize sticker set [%d]", setID)
				_, err = opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
					CallbackQueryID: opts.CallbackQuery.ID,
					Text:            "未找到 ID 对应的贴纸包，请重新尝试发送此贴纸再点击按钮",
					ShowAlert:       true,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "no sticker set found for set ID notice").
						Msg(flaterr.AnswerCallbackQuery.Str())
					handlerErr.Addt(flaterr.AnswerCallbackQuery, "no sticker set found for set ID notice", err)
				}
				return handlerErr.Flat()
			}
		}

		stickerSet, err := opts.Thebot.GetStickerSet(opts.Ctx, &bot.GetStickerSetParams{ Name: stickerSetName })
		if err != nil {
			logger.Error().
				Err(err).
				Str("stickerSetName", stickerSetName).
				Msg(flaterr.GetStickerSet.Str())
			handlerErr.Addt(flaterr.GetStickerSet, stickerSetName, err)

			_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.CallbackQuery.From.ID,
				Text:   fmt.Sprintf("获取贴纸包时发生了一些错误\n<blockquote expandable>Failed to get sticker set info: %s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
				ParseMode: models.ParseModeHTML,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "get sticker set info error").
					Msg(flaterr.SendMessage.Str())
				handlerErr.Addt(flaterr.SendMessage, "get sticker set info error", err)
			}
		} else {
			logger.Info().
				Dict("stickerSet", zerolog.Dict().
					Str("title", stickerSet.Title).
					Str("name", stickerSet.Name).
					Int("allCount", len(stickerSet.Stickers)),
				).
				Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
				Msg("Start download and collect sticker set")

			if opts.CallbackQuery.Message.Message.Text != "" {
				_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
					ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					Text:      fmt.Sprintf("正在收藏 <a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包，请稍候...", stickerSet.Name, stickerSet.Title),
					ParseMode: models.ParseModeHTML,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
						Str("content", "start download stickerset notice").
						Msg(flaterr.EditMessageText.Str())
					handlerErr.Addt(flaterr.EditMessageText, "start download stickerset notice", err)
				}
			} else {
				_, err = opts.Thebot.EditMessageCaption(opts.Ctx, &bot.EditMessageCaptionParams{
					ChatID:    opts.CallbackQuery.Message.Message.Chat.ID,
					MessageID: opts.CallbackQuery.Message.Message.ID,
					Caption:   fmt.Sprintf("正在收藏 <a href=\"https://t.me/addstickers/%s\">%s</a> 贴纸包，请稍候...", stickerSet.Name, stickerSet.Title),
					ParseMode: models.ParseModeHTML,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Dict(utils.GetUserDict(&opts.CallbackQuery.From)).
						Str("content", "start download stickerset notice").
						Msg(flaterr.EditMessageCaption.Str())
					handlerErr.Addt(flaterr.EditMessageCaption, "start download stickerset notice", err)
				}
			}

			stickerData, err := download.GetStickerPack(opts.Ctx, opts.Thebot, stickerSet, false)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to download sticker set")
				handlerErr.Addf("failed to download sticker set: %w", err)

				_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.CallbackQuery.From.ID,
					Text:   fmt.Sprintf("下载贴纸包时发生了一些错误\n<blockquote expandable>Failed to download sticker set: %s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
					ParseMode: models.ParseModeHTML,
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "download sticker set error").
						Msg(flaterr.SendMessage.Str())
					handlerErr.Addt(flaterr.SendMessage, "download sticker set error", err)
				}
			} else {
				var reply *models.ReplyParameters
				collected := stickerCollect.GetStickerSetByName(stickerSetName)
				if collected != nil {
					// 已经收藏过了，更新贴纸包信息
					reply = &models.ReplyParameters{
						MessageID: collected.MsgID,
						AllowSendingWithoutReply: true, // 如果旧消息被删除了也允许发送
					}
				}
				var pendingMessage string
				if stickerData.WebP > 0 { pendingMessage += fmt.Sprintf(" %d(静态)", stickerData.WebP) }
				if stickerData.WebM > 0 { pendingMessage += fmt.Sprintf(" %d(动态)", stickerData.WebM) }
				if stickerData.TGS  > 0 { pendingMessage += fmt.Sprintf(" %d(矢量)", stickerData.TGS) }
				channelMessage, err := opts.Thebot.SendDocument(opts.Ctx, &bot.SendDocumentParams{
					ChatID:          stickerCollect.ChannelID,
					ParseMode:       models.ParseModeMarkdownV1,
					ReplyParameters: reply,
					Caption:         fmt.Sprintf("[%s](https://t.me/addstickers/%s)\n共 %d 个贴纸:%s\n存档时间 %s", stickerData.StickerSetTitle, stickerData.StickerSetName, stickerData.StickerCount, pendingMessage, time.Now().Format(time.RFC3339)),
					Document:        &models.InputFileUpload{ Filename: fmt.Sprintf("%s(%d).zip", stickerData.StickerSetName, stickerData.StickerCount), Data: stickerData.Data },
					ReplyMarkup:     &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
						Text: "查看贴纸包", URL: "https://t.me/addstickers/" + stickerData.StickerSetName },
					}}},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("channelID", stickerCollect.ChannelID).
						Str("stickerSetName", stickerSetName).
						Str("content", "collect sticker set file").
						Msg(flaterr.SendDocument.Str())
					handlerErr.Addt(flaterr.SendDocument, "collect sticker set", err)
					_, err = opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
						ChatID:    opts.CallbackQuery.From.ID,
						Text:      fmt.Sprintf("将贴纸包发送到收藏频道失败: <blockquote expandable>%s</blockquote>", utils.IgnoreHTMLTags(err.Error())),
						ParseMode: models.ParseModeHTML,
					})
					if err != nil {
						logger.Error().
							Err(err).
							Int64("channelID", stickerCollect.ChannelID).
							Str("stickerSetName", stickerSetName).
							Str("content", "collect sticker set failed notice").
							Msg(flaterr.SendMessage.Str())
						handlerErr.Addt(flaterr.SendMessage, "collect sticker set failed notice", err)
					}
				} else {
					if collected == nil {
						stickerCollect.StickerSet = append(stickerCollect.StickerSet, stickerSetInfo{
							Title: stickerSet.Title,
							Name:  stickerSet.Name,
							MsgID: channelMessage.ID,
							Count: stickerData.StickerCount,
						})
					} else {
						// 更新已收藏的贴纸包信息
						collected.Title = stickerSet.Title
						collected.MsgID = channelMessage.ID
						collected.Count = stickerData.StickerCount
					}

					err = SaveCollectStickerList(opts.Ctx)
					if err != nil {
						logger.Error().
							Err(err).
							Dict("collectSticker", zerolog.Dict().
								Str("title", collected.Title).
								Str("name", collected.Name),
							).
							Msg("Failed to save collect sticker list")
						handlerErr.Addf("failed to save collect sticker list: %w", err)
					}

					_, err = opts.Thebot.EditMessageReplyMarkup(opts.Ctx, &bot.EditMessageReplyMarkupParams{
						ChatID:    opts.CallbackQuery.From.ID,
						MessageID: opts.CallbackQuery.Message.Message.ID,
						ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{{
							Text: "✅ 已收藏至频道", URL: utils.MsgLinkPrivate(channelMessage.Chat.ID, channelMessage.ID),
						}}}},
					})
					if err != nil {
						logger.Err(err).
							Int64("channelID", stickerCollect.ChannelID).
							Str("stickerSetName", stickerSetName).
							Str("content", "collect sticker set success notice").
							Msg(flaterr.EditMessageReplyMarkup.Str())
						handlerErr.Addt(flaterr.EditMessageReplyMarkup, "collect sticker set success notice", err)
					}
				}
			}
		}
	}

	return handlerErr.Flat()
}

func AddButton(setName string, newCount int, isManager bool, button *[][]models.InlineKeyboardButton) {
	if stickerCollect.ChannelID != 0 {
		set := stickerCollect.GetStickerSetByName(setName)
		if set != nil {
			if isManager {
				*button = append(*button, []models.InlineKeyboardButton{
					{
						Text:         fmt.Sprintf("🔁 更新 (%d>%d)", set.Count, newCount),
						CallbackData: fmt.Sprintf("c_%s", setName),
					},
					{
						Text: "✅ 已收藏至频道",
						URL:  utils.MsgLinkPrivate(stickerCollect.ChannelID, set.MsgID),
					},
				})
			} else {
				*button = append(*button, []models.InlineKeyboardButton{{
					Text: "✅ 已收藏至频道",
					URL:  utils.MsgLinkPrivate(stickerCollect.ChannelID, set.MsgID),
				}})
			}
		} else if isManager {
			*button = append(*button, []models.InlineKeyboardButton{{
				Text: "⭐️ 收藏至频道",
				CallbackData: fmt.Sprintf("c_%s", setName),
			}})
		}
	}
}
