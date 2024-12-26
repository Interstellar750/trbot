package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

type forwardMetadata struct {
	EnabledForwardGroupID []struct {
		ID     int64 `yaml:"id"`
		Enable bool  `yaml:"enable"`
	} `yaml:"GroupID"`
}

func fwdonly_ReadMetadataFile(path string) (*forwardMetadata, error) {
	file, err := os.Open(path)
	if err != nil { return nil, fmt.Errorf("%w: %v", ErrFileOpen, err) }
	defer file.Close()

	var metadata forwardMetadata
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&metadata)
	if err != nil { return nil, fmt.Errorf("%w: %v", ErrYamlDecode, err) }

	return &metadata, nil
}

// 检查群组 ID 是否存在于配置中
func fwdonly_IsGroupEnabled(groupID int64, config *forwardMetadata) bool {
	for _, info := range config.EnabledForwardGroupID {
		if info.ID == groupID && info.Enable {
			return true
		}
	}
	return false
}

// 添加群组 ID 到配置中
func fwdonly_AddGroupID(groupID int64, config *forwardMetadata) {
	for _, group := range config.EnabledForwardGroupID {
		if group.ID == groupID {
			return // 群组已存在，不重复添加
		}
	}
	if !fwdonly_IsGroupEnabled(groupID, config) {
		config.EnabledForwardGroupID = append(
			config.EnabledForwardGroupID, struct {
				ID     int64 `yaml:"id"`
				Enable bool  `yaml:"enable"`
			}{
				ID:     groupID,
				Enable: false,
			},
		)
	}
}

// 启用或禁用群组的功能
func fwdonly_SetForwardOnly(groupID int64, config *forwardMetadata, enable bool) error {
	ID := fwdonly_FindGroupByID(groupID, config)
	if ID != -1 {
		config.EnabledForwardGroupID[ID].Enable = enable
		fmt.Println("[fwdonly_SetForwardOnly]", config.EnabledForwardGroupID[ID].Enable)
	} else {
		return fmt.Errorf("unknown groupID: %d", groupID)
	}
	return nil
}

// 查找群组 ID 是否已存在配置中
func fwdonly_FindGroupByID(groupID int64, config *forwardMetadata) int {
	for i := range config.EnabledForwardGroupID {
		if config.EnabledForwardGroupID[i].ID == groupID {
			return i
		}
	}
	return -1
}

func fwdonly_ReadMetadata() error {
	data, err := fwdonly_ReadMetadataFile(fwdonly_path + metadatafile_name);
	if data != nil {
		forwardonlylist = data
	}
	if err != nil {
		if errors.Is(err, ErrFileOpen) {
			log.Println("forwardonly: no config file found, create an new one")
			err = fwdonly_SaveMetadata(fwdonly_path, metadatafile_name, &forwardMetadata{})
			if err != nil {
				return err
			}
		} else if errors.Is(err, ErrYamlDecode) {
			return err
		}
	}
	return err
}

// 将群组配置保存到 YAML 文件
func fwdonly_SaveMetadata(path string, name string, Metadata *forwardMetadata) error {
	data, err := yaml.Marshal(Metadata)
	if err != nil { return err }

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("%w: %v", ErrDirectoryCreate, err)
		}
	}

	if _, err := os.Stat(path + name); os.IsNotExist(err) {
		_, err := os.Create(path + name)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrFileCreate, err)
		}
	}

	return os.WriteFile(path + name, data, 0644)
}

func addToWriteListHandler(ctx context.Context, thebot *bot.Bot, update *models.Update) {
	if update.Message.Chat.Type == "private" {
		botMessage, _ := thebot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "仅限转发模式被设计为仅在群组中可用",
			ReplyParameters: &models.ReplyParameters{ MessageID: update.Message.ID },
		})
		time.Sleep(time.Second * 10)
		thebot.DeleteMessages(ctx, &bot.DeleteMessagesParams{
			ChatID:     update.Message.Chat.ID,
			MessageIDs: []int{
				update.Message.ID,
				botMessage.ID,
			},
		})
	} else if userIsAdmin(ctx, thebot, update.Message.Chat.ID, update.Message.From.ID) {
		// var groupIsInlist  bool = false
		var groupIsEnabled bool = false

		if forwardonlylist != nil {
			for _, data := range forwardonlylist.EnabledForwardGroupID {
				if data.ID == update.Message.Chat.ID {
					// groupIsInlist = true
					if data.Enable { groupIsEnabled = true }
				}
			}
		} else {
			err := fwdonly_SaveMetadata(fwdonly_path, metadatafile_name, &forwardMetadata{})
			if err != nil {
				log.Print(err)
			}
			thebot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "仅限转发群组列表数据异常，正在重建列表，请再次发送 /forwardonly 来尝试启用仅限转发模式",
				ParseMode: models.ParseModeMarkdownV1,
			})
		}

		// fmt.Println(update.Message.Text)

		if !groupIsEnabled && update.Message.Text == fmt.Sprintf("/forwardonly %d", update.Message.Chat.ID) {
			if err := fwdonly_SetForwardOnly(update.Message.Chat.ID, forwardonlylist, true); err != nil {
				log.Println(err)
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "未知的群组 ID，初次使用请发送 `/forwardonly` 记录群组 ID",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			}
			if err := fwdonly_SaveMetadata("./forwardonly/", "metadata.yaml", forwardonlylist); err != nil {
				log.Printf("Failed to save group config: %v", err)
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   fmt.Sprintf("发生了错误，无法将群组 ID 添加到列表：\n%v", err),
					ParseMode: models.ParseModeMarkdownV1,
				})
			} else {
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "仅限转发模式已启用，若要关闭，请发送 `/forwardonly disable` 来关闭它",
					ParseMode: models.ParseModeMarkdownV1,
				})
			}
		} else if update.Message.Text == "/forwardonly disable" {
			if err := fwdonly_SetForwardOnly(update.Message.Chat.ID, forwardonlylist, false); err != nil {
				log.Println(err)
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "未知的群组 ID，初次使用请发送 `/forwardonly` 记录群组 ID",
					ParseMode: models.ParseModeMarkdownV1,
				})
				return
			}
			if err := fwdonly_SaveMetadata("./forwardonly/", "metadata.yaml", forwardonlylist); err != nil {
				log.Printf("Failed to save group config: %v", err)
			} else {
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   fmt.Sprintf("仅限转发模式已关闭，重新启用请发送 `/forwardonly %d`", update.Message.Chat.ID),
					ParseMode: models.ParseModeMarkdownV1,
				})
			}
		} else if strings.HasPrefix(update.Message.Text, "/forwardonly") {
			if userIsAdmin(ctx, thebot, update.Message.Chat.ID, showBotID()) && userHavePermissionDeleteMessage(ctx, thebot, update.Message.Chat.ID, showBotID()) {
				fwdonly_AddGroupID(update.Message.Chat.ID, forwardonlylist)
				if err := fwdonly_SaveMetadata(fwdonly_path, metadatafile_name, forwardonlylist); err != nil {
					log.Printf("Failed to save group config: %v", err)
				} else {
					thebot.SendMessage(ctx, &bot.SendMessageParams{
						ChatID: update.Message.Chat.ID,
						Text:   fmt.Sprintf("群组 ID 已添加到列表，继续发送 `/forwardonly %d` 以启用仅限转发模式", update.Message.Chat.ID),
						ParseMode: models.ParseModeMarkdownV1,
					})
				}
			} else {
				thebot.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: update.Message.Chat.ID,
					Text:   "启用此功能前，请先将机器人设为管理员\n如果还是提示本消息，请检查机器人是否有删除消息的权限",
				})
			}
		}
	} else {
		botMessage, _ := thebot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "抱歉，您不是群组的管理员，无法为群组更改此功能",
		})
		time.Sleep(time.Second * 2)
		thebot.DeleteMessages(ctx, &bot.DeleteMessagesParams{
			ChatID:     update.Message.Chat.ID,
			MessageIDs: []int{
				update.Message.ID,
				botMessage.ID,
			},
		})
	}
}
