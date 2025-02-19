package main

import (
	"fmt"
	"io"
	"log"
	"os"
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
		if s.ViaID != 0 { // 通过回复添加
			pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] From <a href=\"https://t.me/@id%d\">%s</a> Via <a href=\"https://t.me/@id%d\">%s</a>\n",
				i+1, s.Meaning, s.FromID, s.FromName, s.ViaID, s.ViaName,
			)
		} else if s.FromID != 0 { // 有添加人信息
			pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] From <a href=\"https://t.me/@id%d\">%s</a>\n",
				i+1, s.Meaning, s.FromID, s.FromName,
			)
		} else {
			pendingMessage += fmt.Sprintf("<code>%d</code>. [%s]\n", i+1, s.Meaning)
		}
	}
	return pendingMessage
}

type UdoneseMeaning struct {
	Meaning  string `yaml:"Meaning"`
	FromID   int64  `yaml:"FromID,omitempty"`
	FromName string `yaml:"FromName,omitempty"`
	ViaID    int64  `yaml:"ViaID,omitempty"`
	ViaName  string `yaml:"ViaName,omitempty"`
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

		var fromName string
		var fromID   int64
		var viaName  string
		var viaID    int64

		if opts.update.Message.ReplyToMessage == nil {
			fromName = showUserName(opts.update.Message.From)
			fromID = opts.update.Message.From.ID
		} else {
			fromName = showUserName(opts.update.Message.ReplyToMessage.From)
			fromID = opts.update.Message.ReplyToMessage.From.ID
			viaName = showUserName(opts.update.Message.From)
			viaID = opts.update.Message.From.ID
		}

		var pendingMessage string
		var botMessage *models.Message

		oldMeaning := addUdonese(udon, &UdoneseWord{
			Word: opts.fields[1],
			MeaningList: []UdoneseMeaning{ {
				Meaning: meaning,
				FromID: fromID,
				FromName: fromName,
				ViaID: viaID,
				ViaName: viaName,
			}},
		})
		if oldMeaning != nil {
			pendingMessage += fmt.Sprintf("[%s] 意思已存在于 [%s] 中:\n", meaning, oldMeaning.Word)
			for i, s := range oldMeaning.MeaningList {
				if meaning == s.Meaning {
					if s.ViaID != 0 { // 通过回复添加
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] From <a href=\"https://t.me/@id%d\">%s</a> Via <a href=\"https://t.me/@id%d\">%s</a>\n",
							i + 1, s.Meaning, s.FromID, s.FromName, s.ViaID, s.ViaName,
						)
					} else if s.FromID == 0 { // 有添加人信息
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s] From <a href=\"https://t.me/@id%d\">%s</a>\n",
							i + 1, s.Meaning, s.FromID, s.FromName,
						)
					} else { // 只有意思
						pendingMessage += fmt.Sprintf("<code>%d</code>. [%s]\n", i + 1, s.Meaning)
					}
				}
			}
		} else {
			err = SaveYamlDB(udon_path, metadataFileName, *udon)
			if err != nil {
				pendingMessage += fmt.Sprintln("保存语句时似乎发生了一些错误:\n", err)
			} else {
				pendingMessage += fmt.Sprintf("已添加 [<code>%s</code>]\n", opts.fields[1])
				if viaID != 0 { // 通过回复添加
					pendingMessage += fmt.Sprintf("[%s] From <a href=\"https://t.me/@id%d\">%s</a> Via <a href=\"https://t.me/@id%d\">%s</a>\n",
						meaning, fromID, fromName, viaID, viaName,
					)
				} else if fromID != 0 { // 普通命令添加
					pendingMessage += fmt.Sprintf("[%s] From <a href=\"https://t.me/@id%d\">%s</a>\n",
						meaning, fromID, fromName,
					)
				}
			}
		}

		pendingMessage += fmt.Sprintln("<blockquote>发送的消息与此消息将在十秒后删除</blockquote>")
		botMessage, _= opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
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
			if strings.EqualFold(word.Word, opts.fields[1]) && len(word.MeaningList) > 0 {
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
		opts.thebot.SendMessage(opts.ctx, &bot.SendMessageParams{
			ChatID: opts.update.Message.Chat.ID,
			ReplyParameters: &models.ReplyParameters{ MessageID: opts.update.Message.ID },
			Text:   "这个词还没有记录，使用 `udonese <词> <意思>` 来添加吧",
			ParseMode: models.ParseModeMarkdownV1,
		})
	}
}

var udoneseInlineCommand string = "sms"

func udoneseInlineHandler(opts *subHandlerOpts) []models.InlineQueryResult {
	var udoneseResultList []models.InlineQueryResult

	// 查语句需要不区分大小写
	for i := 0; i < len(opts.fields); i++ {
		opts.fields[i] = strings.ToLower(opts.fields[i])
	}

	// 仅 :sms 参数，或带有分页符号，输出全部词
	if len(opts.fields) < 2 || len(opts.fields) == 2 && strings.HasPrefix(opts.fields[len(opts.fields)-1], InlinePaginationSymbol) {
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
			if InlineQueryMatchMultKeyword(opts.fields, []string{strings.ToLower(data.Word)}, true) {
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
			if InlineQueryMatchMultKeyword(opts.fields, data.OnlyMeaning(), true) {
				for _, n := range data.MeaningList {
					if InlineQueryMatchMultKeyword(opts.fields, []string{strings.ToLower(n.Meaning)}, true) {
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
				Description: fmt.Sprintf("没有找到包含 %s 的词或意思，若想查看添加方法，请点击这条内容", opts.fields[1:]),
				InputMessageContent: models.InputTextMessageContent{
					MessageText: "没有这个词，使用 `udonese <词> <意思>` 来添加吧",
					ParseMode: models.ParseModeMarkdownV1,
				},
			})
		}
	}
	return udoneseResultList
}
