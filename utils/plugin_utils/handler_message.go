package plugin_utils

import (
	"strings"
	"time"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/flaterr"
	"trbot/utils/handler_params"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

func RunCommandHandlers(params *handler_params.Message) (bool, error) {
	if params.Message.Text == "" { return false, nil }

	logger := zerolog.Ctx(params.Ctx).
		With().
		Dict(utils.GetUserDict(params.Message.From)).
		Dict(utils.GetChatDict(&params.Message.Chat)).
		Str("text",    params.Message.Text).
		Str("caption", params.Message.Caption).
		Logger()


	if strings.HasPrefix(params.Message.Text, "/") {
		// 匹配默认的 `/xxx` 命令
		isCalled, err := RunSlashCommandHandlers(params)
		if isCalled {
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Error in slash symbol command handler")
			}
			return true, err
		}
		// 不存在以 `/` 作为前缀命令时的条件
		if params.Message.Chat.Type == models.ChatTypePrivate {
			// 非冗余条件，在私聊状态下应处理用户发送的所有开头为 / 的命令
			// 与群组中不同，群组中命令末尾不指定此 bot 回应的命令无须处理，以防与群组中的其他 bot 冲突
			_, err := params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
				ChatID:          params.Message.Chat.ID,
				Text:            "不存在的命令",
				ReplyParameters: &models.ReplyParameters{ MessageID: params.Message.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "no this command").
					Msg(flaterr.SendMessage.Str())
			}
			return true, nil
		} else if strings.HasSuffix(params.Fields[0], "@" + consts.BotMe.Username) {
			// 当使用一个不存在的命令，但是命令末尾指定为此 bot 处理
			// 为防止与其他 bot 的命令冲突，默认不会响应不在命令列表中的命令
			// 如果消息以 /xxx@examplebot 的形式指定此 bot 回应，且 /xxx 不在预设的命令中时，才发送该命令不可用的提示
			botMessage, err := params.Thebot.SendMessage(params.Ctx, &bot.SendMessageParams{
				ChatID:          params.Message.Chat.ID,
				Text:            "不存在的命令",
				ReplyParameters: &models.ReplyParameters{ MessageID: params.Message.ID },
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("content", "no this command").
					Msg(flaterr.SendMessage.Str())
			} else {
				time.Sleep(time.Second * 10)
				_, err = params.Thebot.DeleteMessages(params.Ctx, &bot.DeleteMessagesParams{
					ChatID:     params.Message.Chat.ID,
					MessageIDs: []int{
						params.Message.ID,
						botMessage.ID,
					},
				})
				if err != nil {
					logger.Error().
						Err(err).
						Str("content", "no this command").
						Msg(flaterr.DeleteMessages.Str())
				}
			}
			return true, nil
		}
		return false, nil
	} else if len(params.Message.Text) > 0 {
		// 没有 `/` 号作为前缀，检查是不是自定义命令
		isCalled, err := RunFullCommandHandlers(params)
		if isCalled {
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Error in full command handler")
			}
			return true, err
		}

		// 以后缀来触发的命令
		isCalled, err = RunSuffixCommandHandlers(params)
		if isCalled {
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Error in suffix command handler")
			}
			return true, err
		}
	}
	return false, nil
}
