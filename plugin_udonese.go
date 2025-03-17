package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

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
	var pendingMessage = fmt.Sprintf("[<code>%s</code>] 已使用 %d 次，它的意思有\n", list.Word, list.Used)
	for i, s := range list.MeaningList {
		// 先加意思
		pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] ", i+1, s.Meaning)

		// 来源的用户或频道
		if s.FromUsername != "" {
			pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a> ", s.FromUsername, s.FromName)
		} else if s.FromID != 0 {
			if s.FromID < 0 {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", strings.TrimPrefix(strconv.FormatInt(s.FromID, 10), "-100"), s.FromName)
			} else {
				pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", s.FromID, s.FromName)
			}
		}

		// 由其他用户添加时的信息
		if s.ViaUsername != "" {
			pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", s.ViaUsername, s.ViaName)
		} else if s.ViaID != 0 {
			if s.ViaID < 0 {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", strings.TrimPrefix(strconv.FormatInt(s.ViaID, 10), "-100"), s.ViaName)
			} else {
				pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", s.ViaID, s.ViaName)
			}
		}
		
		// 末尾换行
		pendingMessage += "\n"
	}
	return pendingMessage
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

func readUdonese(path, name string) (*Udonese, error) {
	var udonese *Udonese

	file, err := os.Open(path + name)
	if err != nil {
		// 如果是找不到目录，新建一个
		log.Println("[Udonese]: Not found database file. Created new one")
		SaveYamlDB(path, name, Udonese{})
		return &Udonese{}, err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&udonese)
	if err != nil {
		if err == io.EOF {
			log.Println("[Udonese]: Udonese list looks empty. now format it")
			SaveYamlDB(path, name, Udonese{})
			return &Udonese{}, nil
		}
		log.Println("(func)readUdonese:", err)
		return &Udonese{}, err
	}
	return udonese, nil
}

// 如果要添加的意思重复，返回对应意思的单个词结构体指针，否则返回空指针
// 设计之初可以添加多个意思，但现在不推荐这样
func addUdonese(udonese *Udonese, params *UdoneseWord) *UdoneseWord {
	for wordIndex, savedList := range udonese.List {
		if savedList.Word == params.Word {
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

func udoneseInlineHandler(opts *subHandlerOpts) []models.InlineQueryResult {
	var udoneseResultList []models.InlineQueryResult

	// 查语句需要不区分大小写
	for i := 0; i < len(opts.fields); i++ {
		opts.fields[i] = strings.ToLower(opts.fields[i])
	}

	keywordFields := InlineExtractKeywords(opts.fields)

	// 仅 :sms 参数，或带有分页符号，输出全部词
	if len(keywordFields) == 0{
		for _, data := range AdditionalDatas.Udonese.List {
			udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
				ID:    data.Word + "-word",
				Title: data.Word,
				Description: fmt.Sprintf("已使用 %d 次，有 %d 个意思: %s...", data.Used, len(data.MeaningList), data.MeaningList[0].Meaning),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: data.OutputMeanings(),
					ParseMode: models.ParseModeHTML,
				},
			})
		}
	} else {
		for _, data := range AdditionalDatas.Udonese.List {
			// 通过词查找意思
			if InlineQueryMatchMultKeyword(keywordFields, []string{strings.ToLower(data.Word)}) {
				udoneseResultList = append(udoneseResultList, &models.InlineQueryResultArticle{
					ID:    data.Word + "-word",
					Title: data.Word,
					Description: fmt.Sprintf("已使用 %d 次，有 %d 个意思: %s...", data.Used, len(data.MeaningList), data.MeaningList[0].Meaning),
					InputMessageContent: models.InputTextMessageContent{
						MessageText: data.OutputMeanings(),
						ParseMode: models.ParseModeHTML,
					},
				})
			}
			// 通过意思查找词
			if InlineQueryMatchMultKeyword(keywordFields, data.OnlyMeaning()) {
				for _, n := range data.MeaningList {
					if InlineQueryMatchMultKeyword(keywordFields, []string{strings.ToLower(n.Meaning)}) {
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

var Udonese_InlineCommandHandler = Plugin_Inline{
	command: "sms",
	handler: udoneseInlineHandler,
}

func udoneseHandler(opts *subHandlerOpts) {
	// 不响应来自转发的命令
	if opts.update.Message.ForwardOrigin != nil {
		return
	}

	udon, err := AdditionalDatas.Udonese, AdditionalDatas.UdoneseErr
	if err != nil {
		log.Println("some error in while read udonese list: ", err)
	}

	// 统计词使用次数
	for i, n := range udon.OnlyWord() {
		if n == opts.update.Message.Text || strings.HasPrefix(opts.update.Message.Text, n) {
			udon.List[i].Used++
			err = SaveYamlDB(udon_path, metadataFileName, *udon)
			if err != nil {
				log.Println("get some error when add udonese used count:", err)
			}
			// fmt.Println(udon.List[i].Word, "+1", udon.List[i].Used)
		} 
	}

	if opts.fields[0] == "sms" {
		// 参数过少，提示用法
		if len(opts.fields) < 2 {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID: opts.update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				Text:   "使用方法：发送 `sms <词>` 来查看对应的意思",
				ParseMode: models.ParseModeMarkdownV1,
				ReplyMarkup: &models.InlineKeyboardMarkup{ InlineKeyboard: [][]models.InlineKeyboardButton{{{
					Text: "点击浏览全部词与意思",
					SwitchInlineQueryCurrentChat: InlineSubCommandSymbol + "sms ",
				}}}},
			})
			return
		}

		// 在数据库循环查找这个词
		for _, word := range udon.List {
			if strings.EqualFold(word.Word, opts.fields[1]) && len(word.MeaningList) > 0 {
				_, err := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ParseMode: models.ParseModeHTML,
				})
				if err != nil {
					log.Println("get some error when answer udonese meaning:", err)
				}
				return
			}
		}

		// 到这里就是没找到，提示没有
		botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
		})

		time.Sleep(time.Second * 10)
		opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			ChatID: opts.update.Message.Chat.ID,
			MessageIDs: []int{
				botMessage.ID,
			},
		})

		return
	} else if opts.fields[0] == "udonese" {
		if len(opts.fields) < 3 {
			opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
				ChatID:    opts.update.Message.Chat.ID,
				ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
				Text: "使用 `udonese <词> <单个意思>` 来添加记录",
				ParseMode: models.ParseModeMarkdownV1,
			})
			return
		}

		meaning := strings.TrimSpace(opts.update.Message.Text[len(opts.fields[0])+len(opts.fields[1])+2:])

		var fromID       int64
		var fromUsername string
		var fromName     string
		var viaID        int64
		var viaUsername  string
		var viaName      string

		var isVia         bool
		var isFromGroup   bool
		var isViaGroup    bool
		var isFromChannel bool
		var isViaChannel  bool

		if opts.update.Message.ReplyToMessage != nil {
			// 有回复一条信息，通过回复消息添加词
			isVia = true
			if opts.update.Message.ReplyToMessage.From.IsBot {
				if opts.update.Message.ReplyToMessage.From.ID == 136817688 {
					// 频道身份信息
					isViaChannel = true
				} else if opts.update.Message.ReplyToMessage.From.ID == 1087968824 {
					// 群组匿名身份
					isViaGroup = true
				} else {
					// 有 bot 标识，但不是频道身份也不是群组匿名，则是普通 bot
					isVia = false
				}
			}
		}
		// 发送命令的人信息
		if opts.update.Message.From.IsBot {
			if opts.update.Message.From.ID == 136817688 {
				// 用频道身份发言
				isFromChannel = true
			} else if opts.update.Message.From.ID == 1087968824 {
				// 用群组匿名身份发言
				isFromGroup = true
			}
		}


		if isVia {
			if isViaChannel || isViaGroup {
				// 回复给一条频道身份的信息
				fromID = opts.update.Message.ReplyToMessage.SenderChat.ID
				fromUsername = opts.update.Message.ReplyToMessage.SenderChat.Username
				fromName = showChatName(opts.update.Message.ReplyToMessage.SenderChat)
			} else {
				// 回复给普通用户
				fromName = showUserName(opts.update.Message.ReplyToMessage.From)
				fromID = opts.update.Message.ReplyToMessage.From.ID
			}
			if isFromChannel || isFromGroup {
				// 频道身份
				viaID = opts.update.Message.SenderChat.ID
				viaUsername = opts.update.Message.SenderChat.Username
				viaName = showChatName(opts.update.Message.SenderChat)
			} else { 
				// 普通用户身份
				viaID = opts.update.Message.From.ID
				viaName = showUserName(opts.update.Message.From)
			}
		} else {
			if isFromChannel || isFromGroup {
				// 频道身份
				fromID = opts.update.Message.SenderChat.ID
				fromUsername = opts.update.Message.SenderChat.Username
				fromName = showChatName(opts.update.Message.SenderChat)
			} else {
				// 普通用户身份
				fromID = opts.update.Message.From.ID
				fromName = showUserName(opts.update.Message.From)
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

		oldMeaning := addUdonese(udon, &UdoneseWord{
			Word: opts.fields[1],
			MeaningList: []UdoneseMeaning{ {
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
							pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", strings.TrimPrefix(strconv.FormatInt(s.FromID, 10), "-100"), s.FromName)
						} else {
							pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", s.FromID, s.FromName)
						}
					}

					// 由其他用户添加时的信息
					if s.ViaUsername != "" {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", s.ViaUsername, s.ViaName)
					} else if s.ViaID != 0 {
						if s.ViaID < 0 {
							pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", strings.TrimPrefix(strconv.FormatInt(s.ViaID, 10), "-100"), s.ViaName)
						} else {
							pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", s.ViaID, s.ViaName)
						}
					}
					
					// 末尾换行
					pendingMessage += "\n"
				}
			}
		} else {
			err = SaveYamlDB(udon_path, metadataFileName, *udon)
			if err != nil {
				pendingMessage += fmt.Sprintln("保存语句时似乎发生了一些错误:\n", err)
			} else {
				pendingMessage += fmt.Sprintf("已添加 [<code>%s</code>]\n", opts.fields[1])
				pendingMessage += fmt.Sprintf("[%s] ", meaning)
				
				// 来源的用户或频道
				if fromUsername != "" {
					pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/%s\">%s</a> ", fromUsername, fromName)
				} else if fromID != 0 {
					if fromID < 0 {
						pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/c/%s/0\">%s</a> ", strings.TrimPrefix(strconv.FormatInt(fromID, 10), "-100"), fromName)
					} else {
						pendingMessage += fmt.Sprintf("From <a href=\"https://t.me/@id%d\">%s</a> ", fromID, fromName)
					}
				}

				// 由其他用户添加时的信息
				if viaUsername != "" {
					pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/%s\">%s</a> ", viaUsername, viaName)
				} else if viaID != 0 {
					if viaID < 0 {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/c/%s/0\">%s</a> ", strings.TrimPrefix(strconv.FormatInt(viaID, 10), "-100"), viaName)
					} else {
						pendingMessage += fmt.Sprintf("Via <a href=\"https://t.me/@id%d\">%s</a> ", viaID, viaName)
					}
				}
			}
		}

		pendingMessage += fmt.Sprintln("<blockquote>发送的消息与此消息将在十秒后删除</blockquote>")
		botMessage, _ = opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			Text: pendingMessage,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			ParseMode: models.ParseModeHTML,
		})
		if err == nil {
			time.Sleep(time.Second * 10)
			opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
				ChatID: opts.update.Message.Chat.ID,
				MessageIDs: []int{
					opts.update.Message.ID,
					botMessage.ID,
				},
			})
		}
		return
	} else if len(opts.fields) > 1 && strings.HasSuffix(opts.update.Message.Text, "ssm") {
		// 在数据库循环查找这个词
		for _, word := range udon.List {
			if strings.EqualFold(word.Word, opts.fields[0]) && len(word.MeaningList) > 0 {
				opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
					ChatID: opts.update.Message.Chat.ID,
					Text:   word.OutputMeanings(),
					ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
					ParseMode: models.ParseModeHTML,
				})
				return
			}
		}

		// 到这里就是没找到，提示没有
		botMessage, _ := opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
		})

		time.Sleep(time.Second * 10)
		opts.thebot.DeleteMessages(opts.ctx, &bot.DeleteMessagesParams{
			ChatID: opts.update.Message.Chat.ID,
			MessageIDs: []int{
				botMessage.ID,
			},
		})
	}
}

var Udonese_SlashCommandHandlers = []Plugin_CustomSymbolCommand{
	{
		fullCommand: "udonese",
		handler:     udoneseHandler,
	},
	{
		fullCommand: "sms",
		handler:     udoneseHandler,
	},
}

var Udonese_SuffixCommandHandler = Plugin_SuffixCommand{
	suffixCommand: "ssm",
	handler:       udoneseHandler,
}
