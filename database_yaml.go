package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"time"

	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

type DataBaseYaml struct {
	// 如果运行中希望程序强制读取新数据，在 YAML 数据库文件的开头添加 FORCEOVERWRITE: true 即可
	ForceOverwrite bool `yaml:"FORCEOVERWRITE,omitempty"`

	UpdateTimestamp int64 `yaml:"UpdateTimestamp"`
	Data struct {
		IDs []IDInfo `yaml:"IDs"`
		Admin []int64 `yaml:"Admin,omitempty"`
		// SavedMessage []SavedMessage `yaml:"SavedMessage,omitempty"`
	} `yaml:"Data"`
}

type IDInfo struct {
	ID       int64           `yaml:"ID"`
	ChatName string          `yaml:"ChatName"`
	ChatType models.ChatType `yaml:"ChatType"`
	AddTime  string          `yaml:"AddTime,omitempty"`

	MessageCount int `yaml:"MessageCount,omitempty"`
	InlineCount  int `yaml:"InlineCount,omitempty"`

	IsBlackList         bool `yaml:"IsBlackList,omitempty"`
	IsBotAdmin          bool `yaml:"IsBotAdmin,omitempty"`
	IsEnableForwardonly bool `yaml:"IsEnableForwardonly,omitempty"`

	// nil/0 voice, 1 sticker， 2 photo
	DefaultInlineMode int `yaml:"DefaultInlineMode,omitempty"`

	LatestMessage      string `yaml:"LatestMessage,omitempty"`
	LatestInlineQuery  string `yaml:"LatestInlineQuery,omitempty"`
	LatestInlineResult string `yaml:"LatestInlineResult,omitempty"`


	HasPendingCallbackQuery bool   `yaml:"HasPendingCallbackQuery,omitempty"`
	LatestCallbackQueryData string `yaml:"LatestCallbackQueryData,omitempty"`

	SavedMessage   SavedMessage `yaml:"SavedMessage,omitempty"`
	IsSavedChannel bool         `yaml:"IsSavedChannel,omitempty"`
	SavedForUserID int64        `yaml:"SavedForUserID,omitempty"`

	InlineAlias    InlineAlias    `yaml:"InlineAliases,omitempty"`
	CustomCommands CustomCommands `yaml:"CustomCommands,omitempty"`
}

type SavedMessage struct {
	// ForUserID int64 `yaml:"ForUserID"`
	Count int `yaml:"Count"` // 当前存储的数量
	SavedTimes int `yaml:"SavedTimes,omitempty"` // 一共存过多少个
	Limit int `yaml:"Limit,omitempty"`
	AgreePrivacyPolicy bool `yaml:"AgreePrivacyPolicy,omitempty"`

	Item SavedMessageItems `yaml:"Item,omitempty"`
}

type SavedMessageItems struct {
	Photo   []SavedMessageTypeCachedPhoto   `yaml:"Photo,omitempty"`
	Video   []SavedMessageTypeCachedVideo   `yaml:"Video,omitempty"`
	Sticker []SavedMessageTypeCachedSticker `yaml:"Sticker,omitempty"`
	Voice   []SavedMessageTypeCachedVoice   `yaml:"Voice,omitempty"`
}

type SavedMessageTypeCachedPhoto struct {
	IsDeleted   bool   `yaml:"IsDeleted,omitempty"`

	ID                string                 `yaml:"ID"`
	FileID            string                 `yaml:"FileID"`
	Title             string                 `yaml:"Title,omitempty"` // inline 标题
	Description       string                 `yaml:"Description,omitempty"` // inline 描述
	Caption           string                 `yaml:"Caption,omitempty"` // 发送后图片携带的文本
	CaptionEntities   []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
	CaptionAboveMedia bool                   `yaml:"CaptionAboveMedia,omitempty"`
}

type SavedMessageTypeCachedVideo struct {
	IsDeleted   bool   `yaml:"IsDeleted,omitempty"`

	ID          string `yaml:"ID"`
	FileID      string `yaml:"FileID"`
	Title       string `yaml:"Title,omitempty"` // inline 标题
	Description string `yaml:"Description,omitempty"` // inline 描述
	Caption     string `yaml:"Caption,omitempty"` // 发送后图片携带的文本
}

type SavedMessageTypeCachedSticker struct {
	IsDeleted   bool   `yaml:"IsDeleted,omitempty"`
	
	ID          string `yaml:"ID"`
	FileID      string `yaml:"FileID"`
	SetName     string `yaml:"SetName,omitempty"`
	SetTitle    string `yaml:"SetTitle,omitempty"`
	Description string `yaml:"Description,omitempty"` // inline 描述
	// Title       string `yaml:"Title,omitempty"` // inline 标题
	// Caption     string `yaml:"Caption,omitempty"` // 发送后图片携带的文本
	// CaptionEntities    []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
	// CaptionAboveMedia bool `yaml:"CaptionAboveMedia,omitempty"`
}

type SavedMessageTypeCachedVoice struct {
	IsDeleted   bool   `yaml:"IsDeleted,omitempty"`
	
	ID          string `yaml:"ID"`
	FileID      string `yaml:"FileID"`
	Title       string `yaml:"Title,omitempty"` // inline 标题
	Caption     string `yaml:"Caption,omitempty"` // 发送后图片携带的文本
	CaptionEntities    []models.MessageEntity `yaml:"CaptionEntities,omitempty"`
	Description string `yaml:"Description,omitempty"` // inline 描述
	// CaptionAboveMedia bool `yaml:"CaptionAboveMedia,omitempty"`
}

type InlineAlias struct {
	Text    string `yaml:"Text,omitempty"`
	Image   string `yaml:"Image,omitempty"`
	Voice   string `yaml:"Voice,omitempty"`
	Video   string `yaml:"Video,omitempty"`
	File    string `yaml:"File,omitempty"`
	Sticker string `yaml:"Sticker,omitempty"`
}

type CustomCommands struct {
	Count int `yaml:"Count"`
	Limit int `yaml:"Limit,omitempty"`

	Item []CustomCommandsItems `yaml:"Item,omitempty"`
}

type CustomCommandsItems struct {
	Text       string `yaml:"Text"`
	WithPrefix bool   `yaml:"WithPrefix,omitempty"`
	Prefix     string `yaml:"Prefix,omitempty"`
}

func ReadYamlDB(pathToFile string) (DataBaseYaml, error) {
	file, err := os.Open(pathToFile)
	if err != nil {
		log.Println("[Database]: Not found database file. Created new one")
		SaveYamlDB(db_path, metadataFileName, DataBaseYaml{})
		return DataBaseYaml{}, err
	}
	defer file.Close()

	var database DataBaseYaml
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&database)
	if err != nil {
		if err == io.EOF {
			log.Println("[Database]: Database looks empty. now format it")
			SaveYamlDB(db_path, metadataFileName, DataBaseYaml{})
			return DataBaseYaml{}, nil
		}
		return DataBaseYaml{}, err
	}

	return database, nil
}

// 路径 文件名 YAML 数据结构体
func SaveYamlDB(path string, name string, database interface{}) error {
	data, err := yaml.Marshal(database)
	if err != nil { return err }

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return err
		}
	}

	if _, err := os.Stat(path + name); os.IsNotExist(err) {
		_, err := os.Create(path + name)
		if err != nil {
			return err
		}
	}

	return os.WriteFile(path + name, data, 0644)
}

// 添加数据
func addToYamlDB(params *IDInfo) {
	database.Data.IDs = append(database.Data.IDs, *params)
}

func AutoSaveDatabaseHandler() {
	// 先读取一下数据库文件
	savedDatabase, err := ReadYamlDB(db_path + metadataFileName)
	if err != nil {
		log.Println("some issues when read database file", err)
		// 如果读取数据库文件时发现数据库为空，使用当前的数据重建数据库文件
		if reflect.DeepEqual(savedDatabase, DataBaseYaml{}){
			printLogAndSave("The database file is empty, recovering database file using current data")
			err = SaveYamlDB(db_path, metadataFileName, database)
			if err != nil {
				printLogAndSave(fmt.Sprintln("some issues happend when recovering empty database:", err))
			} else {
				printLogAndSave(fmt.Sprintf("The database is recovered to %s", db_path + metadataFileName))
			}
			return
		}
	}
	// 没有修改就跳过保存
	if reflect.DeepEqual(savedDatabase, database) && IsDebugMode {
		fmt.Printf("\r%s looks database no any change, skip autosave this time", time.Now().Format(time.RFC3339))
	} else {
		// 如果数据库文件中有设定专用的 `FORCEOVERWRITE: true` 覆写标记，无论任何修改，先保存程序中的数据，再读取新的数据替换掉当前的数据并保存
		if savedDatabase.ForceOverwrite {
			printLogAndSave(fmt.Sprintf("The `FORCEOVERWRITE: true` in %s is detected", db_path + metadataFileName))
			oldFileName := fmt.Sprintf("beforeOverwritten_%d_%s", time.Now().Unix(), metadataFileName)
			err := SaveYamlDB(db_path, oldFileName, savedDatabase)
			if err != nil {
				printLogAndSave(fmt.Sprintln("some issues happend when saving the database before overwritten:", err))
			} else {
				printLogAndSave(fmt.Sprintf("The database before overwritten is saved to %s", db_path + oldFileName))
			}
			database = savedDatabase
			database.ForceOverwrite = false // 移除强制覆盖标记
			err = SaveYamlDB(db_path, metadataFileName, database)
			if err != nil {
				printLogAndSave(fmt.Sprintln("some issues happend when recreat database using new database:", err))
			} else {
				printLogAndSave(fmt.Sprintf("Success read data from the new file and saved to %s", db_path + metadataFileName))
			}
		} else if savedDatabase.UpdateTimestamp > database.UpdateTimestamp { // 没有设定覆写标记，检测到本地的数据库更新时间比程序中的更新时间更晚
			log.Println("The saved database is newer than current data in the program")
			// 如果只是更新时间有差别，更新一下时间，再保存就行
			if reflect.DeepEqual(savedDatabase.Data, database.Data) {
				log.Println("But current data and database is the same, updating UpdateTimestamp in the database only")
				database.UpdateTimestamp = time.Now().Unix()
				err := SaveYamlDB(db_path, metadataFileName, database)
				if err != nil {
					printLogAndSave(fmt.Sprintln("some issues happend when update Timestamp in database:", err))
				} else {
					printLogAndSave("Update Timestamp in database at " + time.Now().Format(time.RFC3339))
				}
			} else {
				// 数据库文件与程序中的数据不同，将新的数据文件改名另存为 `edited_时间戳_文件名`，再使用程序中的数据还原数据文件
				log.Println("Saved database is different from the current database")
				editedFileName := fmt.Sprintf("edited_%d_%s", time.Now().Unix(), metadataFileName)

				// 提示不要在程序运行的时候乱动数据库文件
				log.Println("Do not modify the database file while the program is running, saving modified file and recovering database file using current data")
				err := SaveYamlDB(db_path, editedFileName, savedDatabase)
				if err != nil {
					printLogAndSave(fmt.Sprintln("some issues happend when saving modified database:", err))
				} else {
					printLogAndSave(fmt.Sprintf("The modified database is saved to %s", db_path + editedFileName))
				}
				err = SaveYamlDB(db_path, metadataFileName, database)
				if err != nil {
					printLogAndSave(fmt.Sprintln("some issues happend when recovering database:", err))
				} else {
					printLogAndSave(fmt.Sprintf("The database is recovered to %s", db_path + metadataFileName))
				}
			}
		} else { // 数据有更改，程序内的更新时间也比本地数据库晚，正常保存
			// 正常情况下更新时间就是会比程序内的时间晚，手动修改数据库途中如果有数据变动，而手动修改的时候没有修改时间戳，不会触发上面的保护机制，会直接覆盖手动修改的内容
			// 所以无论如何都尽量不要手动修改数据库文件，如果必要也请在开头添加专用的 `FORCEOVERWRITE: true` 覆写标记，或停止程序后再修改
			database.UpdateTimestamp = time.Now().Unix()
			err := SaveYamlDB(db_path, metadataFileName, database)
			if err != nil {
				printLogAndSave(fmt.Sprintln("some issues happend when auto saving database:", err))
			} else if IsDebugMode {
				printLogAndSave("auto save at " + time.Now().Format(time.RFC3339))
			}
		}
	}
}

// 初次添加群组时，获取必要信息
func AddChatInfo(chat *models.Chat) bool {
	for _, data := range database.Data.IDs {
		if data.ID == chat.ID {
			return false // 群组已存在，不重复添加
		}
	}
	addToYamlDB(&IDInfo{
		ID:       chat.ID,
		ChatType: chat.Type,
		ChatName: showChatName(chat),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	SignalsChannel.Database_save <- true
	return true
}

func AddUserInfo(user *models.User) bool {
	for _, data := range database.Data.IDs {
		if data.ID == user.ID {
			return false // 用户已存在，不重复添加
		}
	}
	addToYamlDB(&IDInfo{
		ID:       user.ID,
		ChatType: models.ChatTypePrivate,
		ChatName: showUserName(user),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	SignalsChannel.Database_save <- true
	return true
}

type SignalChannel struct {
	Database_save          chan bool
	AdditionalDatas_reload chan bool
}

func signalsHandler(SIGNAL SignalChannel) {
	every10Min := time.NewTicker(10 * time.Minute)
	defer every10Min.Stop()

	AdditionalDatas = readAdditionalDatas(AdditionalDatas_paths)

	for {
		select {
		case <-every10Min.C: // 每次 Ticker 触发时执行任务
			AutoSaveDatabaseHandler()
		case <-SIGNAL.Database_save:
			database.UpdateTimestamp = time.Now().Unix()
			err := SaveYamlDB(db_path, metadataFileName, database)
			if err != nil {
				printLogAndSave(fmt.Sprintln("some issues happend when some function call save database now:", err))
			} else {
				printLogAndSave("save at " + time.Now().Format(time.RFC3339))
			}
		case <-SIGNAL.AdditionalDatas_reload:
			AdditionalDatas = readAdditionalDatas(AdditionalDatas_paths)
			log.Println("AdditionalData reloaded")
		}
	}
}

// 获取 ID 信息
func getIDInfo(id *int64) *IDInfo {
	for Index, Data := range database.Data.IDs {
		if Data.ID == *id {
			return &database.Data.IDs[Index]
		}
	}
	return nil
}
