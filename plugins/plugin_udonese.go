package plugins

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/handler_utils"
	"trbot/utils/plugin_utils"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

var UdoneseData *Udonese
var UdoneseErr   error

var Udonese_path string = consts.DB_path + "udonese/"
var UdonGroupID  int64  = -1002205667779
var UdoneseManagerIDs []int64 = []int64{
	872082796, // akaudon
	1086395364, // trle5
}

type Udonese struct {
	Count int           `yaml:"count"`
	List  []UdoneseWord `yaml:"list"`
}

// 获取全部的词
func (udonese Udonese) OnlyWord() []string {
	var Words []string
	for _, n := range udonese.List {
		Words = append(Words, n.Word)
	}
	return Words
}

type UdoneseWord struct {
	Word        string           `yaml:"Word,omitempty"`
	Used        int              `yaml:"Used"`
	MeaningList []UdoneseMeaning `yaml:"MeaningList,omitempty"`
}

// 从 UdoneseWord 列表中提取 Meaning 切片, 转换为小写
func (list UdoneseWord) OnlyMeaning() []string {
	var meanings []string
	for _, singleMeaning := range list.MeaningList {
		meanings = append(meanings, strings.ToLower(singleMeaning.Meaning))
	}
	return meanings
}

// 以 models.ParseModeHTML 的格式输出一个词和其对应的全部意思
func (list UdoneseWord) OutputMeanings() string {
	var pendingMessage = fmt.Sprintf("[ <code>%s</code> ] 已使用 %d 次，", list.Word, list.Used)
	if len(list.MeaningList) != 0 {
		pendingMessage += "它的意思有\n"
	} else {
		pendingMessage += "还没有添加任何意思\n"
	}
	for i, s := range list.MeaningList {
		// 先加意思
		pendingMessage += fmt.Sprintf("<code>%d</code>. [ %s ] ", i+1, s.Meaning)

		// 来源的用户或频道
		if s.FromUsername != "" {
			pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a> ", s.FromUsername, s.FromName)
		} else if s.FromID != 0 {
			if s.FromID < 0 {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(s.FromID), s.FromName)
			} else {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", s.FromID, s.FromName)
			}
		}

		// 由其他用户添加时的信息
		if s.ViaUsername != "" {
			pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", s.ViaUsername, s.ViaName)
		} else if s.ViaID != 0 {
			if s.ViaID < 0 {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(s.ViaID), s.ViaName)
			} else {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", s.ViaID, s.ViaName)
			}
		}

		// 末尾换行
		pendingMessage += "\n"
	}
	return pendingMessage
}

func (list UdoneseWord) buildUdoneseWordKeyboard() models.ReplyMarkup {
	var buttons [][]models.InlineKeyboardButton
	for index, singleMeaning := range list.MeaningList {
		buttons = append(buttons, []models.InlineKeyboardButton{{
			Text: singleMeaning.Meaning,
			CallbackData: fmt.Sprintf("udonese_meaning_%s_%d", list.Word, index),
		}})
	}

	buttons = append(buttons, []models.InlineKeyboardButton{
		{
			Text: "删除整个词",
			CallbackData: "udonese_delword_" + list.Word,
		},
		{
			Text: "关闭菜单",
			CallbackData: "udonese_done",
		},
	})

	return &models.InlineKeyboardMarkup{
		InlineKeyboard: buttons,
	}
}

type UdoneseMeaning struct {
	Meaning      string `yaml:"Meaning"`

	FromID       int64  `yaml:"FromID,omitempty"`
	FromUsername string `yaml:"FromUsername,omitempty"`
	FromName     string `yaml:"FromName,omitempty"`

	ViaID        int64  `yaml:"ViaID,omitempty"`
	ViaUsername  string `yaml:"ViaUsername,omitempty"`
	ViaName      string `yaml:"ViaName,omitempty"`
}

func ReadUdonese() {
	var udonese *Udonese

	file, err := os.Open(Udonese_path + consts.MetadataFileName)
	if err != nil {
		// 如果是找不到目录，新建一个
		log.Println("[Udonese]: Not found database file. Created new one")
		SaveUdonese()
		UdoneseData, UdoneseErr = &Udonese{}, err
		return 
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&udonese)
	if err != nil {
		if err == io.EOF {
			log.Println("[Udonese]: Udonese list looks empty. now format it")
			SaveUdonese()
			UdoneseData, UdoneseErr = &Udonese{}, nil
			return
		}
		log.Println("(func)ReadUdonese:", err)
		UdoneseData, UdoneseErr = &Udonese{}, err
		return
	}
	UdoneseData, UdoneseErr = udonese, nil
}

func SaveUdonese() error {
	data, err := yaml.Marshal(UdoneseData)
	if err != nil { return err }

	if _, err := os.Stat(Udonese_path); os.IsNotExist(err) {
		if err := os.MkdirAll(Udonese_path, 0755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(Udonese_path + consts.MetadataFileName); os.IsNotExist(err) {
		_, err := os.Create(Udonese_path + consts.MetadataFileName)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(Udonese_path + consts.MetadataFileName, data, 0644)
}

// 如果要添加的意思重复，返回对应意思的单个词结构体指针，否则返回空指针
// 设计之初可以添加多个意思，但现在不推荐这样
func addUdonese(udonese *Udonese, params *UdoneseWord) *UdoneseWord {
	for wordIndex, savedList := range udonese.List {
		if strings.EqualFold(savedList.Word, params.Word){
			log.Printf("发现已存在的词 [%s]，正在检查是否有新增的意思", savedList.Word)
			for _, newMeaning := range params.MeaningList {
				var isreallynew bool = true
				for _, oldmeanlist := range savedList.MeaningList {
					if newMeaning.Meaning == oldmeanlist.Meaning {
						isreallynew = false
					}
				}
				if isreallynew {
					udonese.List[wordIndex].MeaningList = append(udonese.List[wordIndex].MeaningList, newMeaning)
					log.Printf("正在为 [%s] 添加 [%s] 意思", udonese.List[wordIndex].Word, newMeaning.Meaning)
				} else {
					log.Println("存在的意思，跳过", newMeaning)
					return &savedList
				}
			}
			return nil
		}
	}
	log.Printf("发现新的词 [%s]，正在添加 %v", params.Word, params.MeaningList)
	udonese.List = append(udonese.List, *params)
	udonese.Count++
	return nil
}

func addUdoneseHandler(opts *handler_utils.SubHandlerOpts) {
	// 不响应来自转发的命令
	if opts.Update.Message.ForwardOrigin != nil {
		return
	}

	isManager := utils.AnyContains(opts.Update.Message.From.ID, UdoneseManagerIDs)

	if opts.Update.Message.Chat.ID != UdonGroupID {
		_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Update.Message.Chat.ID,
			Text:      "抱歉，此命令仅在部分群组可用",
			ParseMode: models.ParseModeMarkdownV1,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			DisableNotification: true,
		})
		if err != nil {
			log.Println("error sending /udonese not allowed group:", err)
		}
		return
	}


	if isManager && len(opts.Fields) < 3 {
		if len(opts.Fields) < 2 {
			opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:    opts.Update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				Text: "使用 `/udonese <词> <单个意思>` 来添加记录\n或使用 `/udonese <词>` 来管理记录",
				ParseMode: models.ParseModeMarkdownV1,
				DisableNotification: true,
			})
			return
		} else {
			checkWord := opts.Fields[1]
			var targetWord UdoneseWord
			for _, wordlist := range UdoneseData.List {
				if wordlist.Word == checkWord {
					targetWord = wordlist
				}
			}

			if targetWord.Word == "" {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID:    opts.Update.Message.Chat.ID,
					Text:      "似乎没有这个词呢...",
					ParseMode: models.ParseModeMarkdownV1,
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					DisableNotification: true,
				})
				if err != nil {
					log.Println("error sending /udonese word not found:", err)
				}
				return
			}

			var pendingMessage string = fmt.Sprintf("词: [ %s ]\n有 %d 个意思\n", targetWord.Word, len(targetWord.MeaningList))

			opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID:    opts.Update.Message.Chat.ID,
				Text:      pendingMessage,
				ParseMode: models.ParseModeHTML,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
				DisableNotification: true,
			})
			return
		}
	} else if len(opts.Fields) < 3 {
		opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID:    opts.Update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			Text: "使用 `/udonese <词> <单个意思>` 来添加记录",
			ParseMode: models.ParseModeMarkdownV1,
			DisableNotification: true,
		})
		return
	}

	meaning := strings.TrimSpace(opts.Update.Message.Text[len(opts.Fields[0])+len(opts.Fields[1])+2:])

	var (
		fromID       int64
		fromUsername string
		fromName     string
		viaID        int64
		viaUsername  string
		viaName      string

		isVia         bool
		isFromGroup   bool
		isViaGroup    bool
		isFromChannel bool
		isViaChannel  bool
	)

	if opts.Update.Message.ReplyToMessage != nil {
		// 有回复一条信息，通过回复消息添加词
		isVia = true
		if opts.Update.Message.ReplyToMessage.From.IsBot {
			if opts.Update.Message.ReplyToMessage.From.ID == 136817688 {
				// 频道身份信息
				isViaChannel = true
			} else if opts.Update.Message.ReplyToMessage.From.ID == 1087968824 {
				// 群组匿名身份
				isViaGroup = true
			} else {
				// 有 bot 标识，但不是频道身份也不是群组匿名，则是普通 bot
				isVia = false
			}
		}
	}
	// 发送命令的人信息
	if opts.Update.Message.From.IsBot {
		if opts.Update.Message.From.ID == 136817688 {
			// 用频道身份发言
			isFromChannel = true
		} else if opts.Update.Message.From.ID == 1087968824 {
			// 用群组匿名身份发言
			isFromGroup = true
		}
	}

	if isVia {
		if isViaChannel || isViaGroup {
			// 回复给一条频道身份的信息
			fromID = opts.Update.Message.ReplyToMessage.SenderChat.ID
			fromUsername = opts.Update.Message.ReplyToMessage.SenderChat.Username
			fromName = utils.ShowChatName(opts.Update.Message.ReplyToMessage.SenderChat)
		} else {
			// 回复给普通用户
			fromName = utils.ShowUserName(opts.Update.Message.ReplyToMessage.From)
			fromID = opts.Update.Message.ReplyToMessage.From.ID
		}
		if isFromChannel || isFromGroup {
			// 频道身份
			viaID = opts.Update.Message.SenderChat.ID
			viaUsername = opts.Update.Message.SenderChat.Username
			viaName = utils.ShowChatName(opts.Update.Message.SenderChat)
		} else {
			// 普通用户身份
			viaID = opts.Update.Message.From.ID
			viaName = utils.ShowUserName(opts.Update.Message.From)
		}
	} else {
		if isFromChannel || isFromGroup {
			// 频道身份
			fromID = opts.Update.Message.SenderChat.ID
			fromUsername = opts.Update.Message.SenderChat.Username
			fromName = utils.ShowChatName(opts.Update.Message.SenderChat)
		} else {
			// 普通用户身份
			fromID = opts.Update.Message.From.ID
			fromName = utils.ShowUserName(opts.Update.Message.From)
		}
	}

	// 来源和经过都是同一位用户，删除 via 信息
	if fromID == viaID {
		isVia = false
		viaID = 0
		viaUsername = ""
		viaName = ""
	}

	var pendingMessage string
	var botMessage *models.Message

	oldMeaning := addUdonese(UdoneseData, &UdoneseWord{
		Word: opts.Fields[1],
		MeaningList: []UdoneseMeaning{{
			Meaning:      meaning,
			FromID:       fromID,
			FromUsername: fromUsername,
			FromName:     fromName,
			ViaID:        viaID,
			ViaUsername:  viaUsername,
			ViaName:      viaName,
		}},
	})
	if oldMeaning != nil {
		pendingMessage += fmt.Sprintf("[%s] 意思已存在于 [%s] 中:\n", meaning, oldMeaning.Word)
		for i, s := range oldMeaning.MeaningList {
			if meaning == s.Meaning {
				pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] ", i + 1, s.Meaning)

				// 来源的用户或频道
				if s.FromUsername != "" {
					pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a> ", s.FromUsername, s.FromName)
				} else if s.FromID != 0 {
					if s.FromID < 0 {
						pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(s.FromID), s.FromName)
					} else {
						pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", s.FromID, s.FromName)
					}
				}

				// 由其他用户添加时的信息
				if s.ViaUsername != "" {
					pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", s.ViaUsername, s.ViaName)
				} else if s.ViaID != 0 {
					if s.ViaID < 0 {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(s.ViaID), s.ViaName)
					} else {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", s.ViaID, s.ViaName)
					}
				}

				// 末尾换行
				pendingMessage += "\n"
			}
		}
	} else {
		err := SaveUdonese()
		if err != nil {
			pendingMessage += fmt.Sprintln("保存语句时似乎发生了一些错误:\n", err)
		} else {
			pendingMessage += fmt.Sprintf("已添加 [<code>%s</code>]\n", opts.Fields[1])
			pendingMessage += fmt.Sprintf("[%s] ", meaning)

			// 来源的用户或频道
			if fromUsername != "" {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a> ", fromUsername, fromName)
			} else if fromID != 0 {
				if fromID < 0 {
					pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(fromID), fromName)
				} else {
					pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", fromID, fromName)
				}
			}

			// 由其他用户添加时的信息
			if viaUsername != "" {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", viaUsername, viaName)
			} else if viaID != 0 {
				if viaID < 0 {
					pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", utils.RemoveIDPrefix(viaID), viaName)
				} else {
					pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", viaID, viaName)
				}
			}
		}
	}

	pendingMessage += fmt.Sprintln("<blockquote>发送的消息与此消息将在十秒后删除</blockquote>")
	botMessage, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
		ChatID: opts.Update.Message.Chat.ID,
		Text: pendingMessage,
		ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
		ParseMode: models.ParseModeHTML,
		DisableNotification: true,
	})
	if err == nil {
		time.Sleep(time.Second * 10)
		opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
			ChatID: opts.Update.Message.Chat.ID,
			MessageIDs: []int{
				opts.Update.Message.ID,
				botMessage.ID,
			},
		})
	}
}

func udoneseInlineHandler(opts *handler_utils.SubHandlerOpts) []models.InlineQueryResult {
	var udoneseResultList []models.InlineQueryResult

	// 查语句需要不区分大小写
	for i := 0; i < len(opts.Fields); i++ {
		opts.Fields[i] = strings.ToLower(opts.Fields[i])
	}

	keywordFields := utils.InlineExtractKeywords(opts.Fields)

	// 仅 :sms 参数，或带有分页符号，输出全部词
	if len(keywordFields) == 0 {
		for _, data := range UdoneseData.List {
			var pendingMessage string
			if len(data.MeaningList) > 0 {
				pendingMessage = fmt.Sprintf("已使用 %d 次，有 %d 个意思: %s...", data.Used, len(data.MeaningList), data.MeaningList[0].Meaning)
			} else {
				pendingMessage = fmt.Sprintf("已使用 %d 次，暂无意思", data.Used)
			}
			udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
				ID:    data.Word + "-word",
				Title: data.Word,
				Description: pendingMessage,
				InputMessageContent: models.InputTextMessageContent{
					MessageText: data.OutputMeanings(),
					ParseMode: models.ParseModeHTML,
				},
			})
		}
	} else {
		for _, data := range UdoneseData.List {
			// 通过词查找意思
			var pendingMessage string
			if len(data.MeaningList) > 0 {
				pendingMessage = fmt.Sprintf("已使用 %d 次，有 %d 个意思: %s...", data.Used, len(data.MeaningList), data.MeaningList[0].Meaning)
			} else {
				pendingMessage = fmt.Sprintf("已使用 %d 次，暂无意思", data.Used)
			}
			if utils.InlineQueryMatchMultKeyword(keywordFields, []string{strings.ToLower(data.Word)}) {
				udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
					ID:    data.Word + "-word",
					Title: data.Word,
					Description: pendingMessage,
					InputMessageContent: models.InputTextMessageContent{
						MessageText: data.OutputMeanings(),
						ParseMode: models.ParseModeHTML,
					},
				})
			}
			// 通过意思查找词
			if utils.InlineQueryMatchMultKeyword(keywordFields, data.OnlyMeaning()) {
				for _, n := range data.MeaningList {
					if utils.InlineQueryMatchMultKeyword(keywordFields, []string{strings.ToLower(n.Meaning)}) {
						udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
							ID:    n.Meaning + "-meaning",
							Title: n.Meaning,
							Description: fmt.Sprintf("%s 对应的词是 %s", n.Meaning, data.Word),
							InputMessageContent: models.InputTextMessageContent{
								MessageText: fmt.Sprintf("%s 对应的词是 <code>%s</code>", n.Meaning, data.Word),
								ParseMode: models.ParseModeHTML,
							},
						})
					}
				}
			}
		}
		if len(udoneseResultList) == 0 {
			udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
				ID:       "none",
				Title:    "没有符合关键词的内容",
				Description: fmt.Sprintf("没有找到包含 %s 的词或意思，若想查看添加方法，请点击这条内容", keywordFields),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "没有这个词，使用 `udonese <词> <意思>` 来添加吧",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}
	return udoneseResultList
}

func udoneseGroupHandler(opts *handler_utils.SubHandlerOpts) {
	// 不响应来自转发的命令
	if opts.Update.Message.ForwardOrigin != nil {
		return
	}
	// 空文本
	if len(opts.Fields) < 1 {
		return
	}

	if UdoneseErr != nil {
		log.Println("some error in while read udonese list: ", UdoneseErr)
		ReadUdonese()
	}

	// 统计词使用次数
	for i, n := range UdoneseData.OnlyWord() {
		if n == opts.Update.Message.Text || strings.HasPrefix(opts.Update.Message.Text, n) {
			UdoneseData.List[i].Used++
			err := SaveUdonese()
			if err != nil {
				log.Println("get some error when add udonese used count:", err)
			}
			// fmt.Println(udon.List[i].Word, "+1", udon.List[i].Used)
		}
	}

	if opts.Fields[0] == "sms" {
		// 参数过少，提示用法
		if len(opts.Fields) < 2 {
			opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
				ChatID: opts.Update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
				Text:   "使用方法：发送 `sms <词>` 来查看对应的意思",
				ParseMode: models.ParseModeMarkdownV1,
				DisableNotification: true,
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击浏览全部词与意思",
					SwitchInlineQueryCurrentChat: consts.InlineSubCommandSymbol + "sms ",
				}}}},
			})
			return
		}

		// 在数据库循环查找这个词
		for _, word := range UdoneseData.List {
			if strings.EqualFold(word.Word, opts.Fields[1]) && len(word.MeaningList) > 0 {
				_, err := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					ParseMode: models.ParseModeHTML,
					DisableNotification: true,
				})
				if err != nil {
					log.Println("get some error when answer udonese meaning:", err)
				}
				return
			}
		}

		// 到这里就是没找到，提示没有
		botMessage, _ := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
			DisableNotification: true,
		})

		time.Sleep(time.Second * 10)
		opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
			ChatID: opts.Update.Message.Chat.ID,
			MessageIDs: []int{
				botMessage.ID,
			},
		})

		return
	} else if len(opts.Fields) > 1 && strings.HasSuffix(opts.Update.Message.Text, "ssm") {
		// 在数据库循环查找这个词
		for _, word := range UdoneseData.List {
			if strings.EqualFold(word.Word, opts.Fields[0]) && len(word.MeaningList) > 0 {
				opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
					ChatID: opts.Update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
					ParseMode: models.ParseModeHTML,
					DisableNotification: true,
				})
				return
			}
		}

		// 到这里就是没找到，提示没有
		botMessage, _ := opts.Thebot.SendMessage(opts.Ctx, &bot.SendMessageParams{
			ChatID: opts.Update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.Update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
			DisableNotification: true,
		})

		time.Sleep(time.Second * 10)
		opts.Thebot.DeleteMessages(opts.Ctx, &bot.DeleteMessagesParams{
			ChatID: opts.Update.Message.Chat.ID,
			MessageIDs: []int{
				botMessage.ID,
			},
		})
	}
}

func init() {
	ReadUdonese()
	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name: "Udonese",
		Saver: SaveUdonese,
		Loader: ReadUdonese,
	})
	plugin_utils.AddInlineHandlerPlugins(plugin_utils.InlineHandler{
		Command: "sms",
		Handler: udoneseInlineHandler,
		Description: "查询 Udonese 词典",
	})
	plugin_utils.AddSlashSymbolCommandPlugins(plugin_utils.SlashSymbolCommand{
		SlashCommand: "udonese",
		Handler:      addUdoneseHandler,
	})
	plugin_utils.AddHandlerByChatIDPlugins(plugin_utils.HandlerByChatID{
		ChatID:  UdonGroupID,
		Handler: udoneseGroupHandler,
	})
	plugin_utils.AddCallbackQueryCommandPlugins(plugin_utils.CallbackQuery{
		CommandChar: "udonese",
		Handler:      udoneseCallbackHandler,
	})
	// plugin_utils.AddSuffixCommandPlugins(plugin_utils.SuffixCommand{
	// 	SuffixCommand: "ssm",
	// 	Handler:       udoneseHandler,
	// })
}

func udoneseCallbackHandler(opts *handler_utils.SubHandlerOpts) {
	if !utils.AnyContains(opts.Update.CallbackQuery.From.ID, UdoneseManagerIDs) {
		opts.Thebot.AnswerCallbackQuery(opts.Ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: opts.Update.CallbackQuery.ID,
			Text: "不可以！",
			ShowAlert: true,
		})
		return
	}
	
	if opts.Update.CallbackQuery.Data == "udonese_done" {
		opts.Thebot.DeleteMessage(opts.Ctx, &bot.DeleteMessageParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
		})
		return
	}

	if strings.HasPrefix(opts.Update.CallbackQuery.Data, "udonese_word_") {
		word := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "udonese_word_")
		var targetWord UdoneseWord
		for _, wordlist := range UdoneseData.List {
			if wordlist.Word == word {
				targetWord = wordlist
			}
		}

		opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text:      fmt.Sprintf("词: %s\n有 %d 个意思\n", targetWord.Word, len(targetWord.MeaningList)),
			ParseMode: models.ParseModeMarkdownV1,
			ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
		})
		return
	} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "udonese_meaning_") {
		wordAndIndex := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "udonese_meaning_")
		wordAndIndexList := strings.Split(wordAndIndex, "_")
		meanningIndex, err := strconv.Atoi(wordAndIndexList[1])
		if err != nil {
			log.Println("covert meanning index error:", err)
		}

		var targetMeaning UdoneseMeaning

		for _, udonese := range UdoneseData.List {
			if udonese.Word == wordAndIndexList[0] {
				for i, meaning := range udonese.MeaningList {
					if i == meanningIndex {
						targetMeaning = meaning
					}
				}
			}
		}

		var pendingMessage string = fmt.Sprintf("意思 [ %s ]\n", targetMeaning.Meaning)

		// 来源的用户或频道
		if targetMeaning.FromUsername != "" {
			pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a>\n", targetMeaning.FromUsername, targetMeaning.FromName)
		} else if targetMeaning.FromID != 0 {
			if targetMeaning.FromID < 0 {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a>\n", utils.RemoveIDPrefix(targetMeaning.FromID), targetMeaning.FromName)
			} else {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a>\n", targetMeaning.FromID, targetMeaning.FromName)
			}
		}

		// 由其他用户添加时的信息
		if targetMeaning.ViaUsername != "" {
			pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a>\n", targetMeaning.ViaUsername, targetMeaning.ViaName)
		} else if targetMeaning.ViaID != 0 {
			if targetMeaning.ViaID < 0 {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a>\n", utils.RemoveIDPrefix(targetMeaning.ViaID), targetMeaning.ViaName)
			} else {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a>\n", targetMeaning.ViaID, targetMeaning.ViaName)
			}
		}

		_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text: pendingMessage,
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: &models.InlineKeyboardMarkup{InlineKeyboard: [][]models.InlineKeyboardButton{{
				{
					Text: "删除此意思",
					CallbackData: fmt.Sprintf("udonese_delmeaning_%s_%d", wordAndIndexList[0], meanningIndex),
				},
				{
					Text: "返回",
					CallbackData: "udonese_word_" + wordAndIndexList[0],
				},
			}}},
		})
		if err != nil {
			log.Println(err)
		}

		return
	} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "udonese_delmeaning_") {
		wordAndIndex := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "udonese_delmeaning_")
		wordAndIndexList := strings.Split(wordAndIndex, "_")
		meanningIndex, err := strconv.Atoi(wordAndIndexList[1])
		if err != nil {
			log.Println("covert meanning index error:", err)
		}
		var newMeaningList []UdoneseMeaning
		var targetWord UdoneseWord
		var deletedMeaning string

		for index, udonese := range UdoneseData.List {
			if udonese.Word == wordAndIndexList[0] {
				for i, meaning := range udonese.MeaningList {
					if i == meanningIndex {
						deletedMeaning = meaning.Meaning
					} else {
						newMeaningList = append(newMeaningList, meaning)
					}
				}
				UdoneseData.List[index].MeaningList = newMeaningList
				targetWord = UdoneseData.List[index]
			}
		}

		var pendingMessage string = fmt.Sprintf("词: [ %s ]\n有 %d 个意思\n\n<blockquote>已删除 [ %s ] 词中的 %s 意思</blockquote>", targetWord.Word, len(targetWord.MeaningList), wordAndIndexList[0], deletedMeaning)

		_, err = opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text:      pendingMessage,
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: targetWord.buildUdoneseWordKeyboard(),
		})
		if err != nil {
			log.Println("error when edit deleted meaning keyboard:", err)
		}

		SaveUdonese()
		return
	} else if strings.HasPrefix(opts.Update.CallbackQuery.Data, "udonese_delword_") {
		word := strings.TrimPrefix(opts.Update.CallbackQuery.Data, "udonese_delword_")
		var newWordList []UdoneseWord
		for _, udonese := range UdoneseData.List {
			if udonese.Word != word {
				newWordList = append(newWordList, udonese)
			}
		}
		UdoneseData.List = newWordList

		var pendingMessage string = fmt.Sprintf("<blockquote>已删除 [ %s ] 词</blockquote>", word)

		_, err := opts.Thebot.EditMessageText(opts.Ctx, &bot.EditMessageTextParams{
			ChatID: opts.Update.CallbackQuery.Message.Message.Chat.ID,
			MessageID: opts.Update.CallbackQuery.Message.Message.ID,
			Text:      pendingMessage,
			ParseMode: models.ParseModeHTML,
			ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
				Text: "关闭菜单",
				CallbackData: "udonese_done",
			}}}},
		})
		if err != nil {
			log.Println("error when edit deleted word message:", err)
		}

		SaveUdonese()
	}
}
