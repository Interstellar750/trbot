package database_yaml

import (
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"time"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/mess"

	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

var Database DataBaseYaml

type DataBaseYaml struct {
	// 如果运行中希望程序强制读取新数据，在 YAML 数据库文件的开头添加 FORCEOVERWRITE: true 即可
	ForceOverwrite bool `yaml:"FORCEOVERWRITE,omitempty"`

	UpdateTimestamp int64 `yaml:"UpdateTimestamp"`
	Data struct {
		IDs []IDInfo `yaml:"IDs"`
		Admin []int64 `yaml:"Admin,omitempty"`
		// SavedMessage map[int64]plugins.SavedMessage `yaml:"SavedMessage,omitempty"`
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

	// nil/0 voice, 1 saved
	DefaultInlineMode int `yaml:"DefaultInlineMode,omitempty"`

	LatestMessage      string `yaml:"LatestMessage,omitempty"`
	LatestInlineQuery  string `yaml:"LatestInlineQuery,omitempty"`
	LatestInlineResult string `yaml:"LatestInlineResult,omitempty"`

	HasPendingCallbackQuery bool   `yaml:"HasPendingCallbackQuery,omitempty"`
	LatestCallbackQueryData string `yaml:"LatestCallbackQueryData,omitempty"`

	// SavedMessage   SavedMessage `yaml:"SavedMessage,omitempty"`
	IsSavedChannel bool         `yaml:"IsSavedChannel,omitempty"`
	SavedForUserID int64        `yaml:"SavedForUserID,omitempty"`
}


func ReadYamlDB(pathToFile string) (DataBaseYaml, error) {
	file, err := os.Open(pathToFile)
	if err != nil {
		log.Println("[Database]: Not found Database file. Created new one")
		SaveYamlDB(consts.DB_path, consts.MetadataFileName, DataBaseYaml{})
		return DataBaseYaml{}, err
	}
	defer file.Close()

	var Database DataBaseYaml
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&Database)
	if err != nil {
		if err == io.EOF {
			log.Println("[Database]: Database looks empty. now format it")
			SaveYamlDB(consts.DB_path, consts.MetadataFileName, DataBaseYaml{})
			return DataBaseYaml{}, nil
		}
		return DataBaseYaml{}, err
	}

	return Database, nil
}

// 路径 文件名 YAML 数据结构体
func SaveYamlDB(path string, name string, Database interface{}) error {
	data, err := yaml.Marshal(Database)
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
	Database.Data.IDs = append(Database.Data.IDs, *params)
}

func AutoSaveDatabaseHandler() {
	// 先读取一下数据库文件
	savedDatabase, err := ReadYamlDB(consts.DB_path + consts.MetadataFileName)
	if err != nil {
		log.Println("some issues when read Database file", err)
		// 如果读取数据库文件时发现数据库为空，使用当前的数据重建数据库文件
		if reflect.DeepEqual(savedDatabase, DataBaseYaml{}){
			mess.PrintLogAndSave("The Database file is empty, recovering Database file using current data")
			err = SaveYamlDB(consts.DB_path, consts.MetadataFileName, Database)
			if err != nil {
				mess.PrintLogAndSave(fmt.Sprintln("some issues happend when recovering empty Database:", err))
			} else {
				mess.PrintLogAndSave(fmt.Sprintf("The Database is recovered to %s", consts.DB_path + consts.MetadataFileName))
			}
			return
		}
	}
	// 没有修改就跳过保存
	if reflect.DeepEqual(savedDatabase, Database) && consts.IsDebugMode {
		fmt.Printf("\r%s looks Database no any change, skip autosave this time", time.Now().Format(time.RFC3339))
	} else {
		// 如果数据库文件中有设定专用的 `FORCEOVERWRITE: true` 覆写标记，无论任何修改，先保存程序中的数据，再读取新的数据替换掉当前的数据并保存
		if savedDatabase.ForceOverwrite {
			mess.PrintLogAndSave(fmt.Sprintf("The `FORCEOVERWRITE: true` in %s is detected", consts.DB_path + consts.MetadataFileName))
			oldFileName := fmt.Sprintf("beforeOverwritten_%d_%s", time.Now().Unix(), consts.MetadataFileName)
			err := SaveYamlDB(consts.DB_path, oldFileName, savedDatabase)
			if err != nil {
				mess.PrintLogAndSave(fmt.Sprintln("some issues happend when saving the Database before overwritten:", err))
			} else {
				mess.PrintLogAndSave(fmt.Sprintf("The Database before overwritten is saved to %s", consts.DB_path + oldFileName))
			}
			Database = savedDatabase
			Database.ForceOverwrite = false // 移除强制覆盖标记
			err = SaveYamlDB(consts.DB_path, consts.MetadataFileName, Database)
			if err != nil {
				mess.PrintLogAndSave(fmt.Sprintln("some issues happend when recreat Database using new Database:", err))
			} else {
				mess.PrintLogAndSave(fmt.Sprintf("Success read data from the new file and saved to %s", consts.DB_path + consts.MetadataFileName))
			}
		} else if savedDatabase.UpdateTimestamp > Database.UpdateTimestamp { // 没有设定覆写标记，检测到本地的数据库更新时间比程序中的更新时间更晚
			log.Println("The saved Database is newer than current data in the program")
			// 如果只是更新时间有差别，更新一下时间，再保存就行
			if reflect.DeepEqual(savedDatabase.Data, Database.Data) {
				log.Println("But current data and Database is the same, updating UpdateTimestamp in the Database only")
				Database.UpdateTimestamp = time.Now().Unix()
				err := SaveYamlDB(consts.DB_path, consts.MetadataFileName, Database)
				if err != nil {
					mess.PrintLogAndSave(fmt.Sprintln("some issues happend when update Timestamp in Database:", err))
				} else {
					mess.PrintLogAndSave("Update Timestamp in Database at " + time.Now().Format(time.RFC3339))
				}
			} else {
				// 数据库文件与程序中的数据不同，将新的数据文件改名另存为 `edited_时间戳_文件名`，再使用程序中的数据还原数据文件
				log.Println("Saved Database is different from the current Database")
				editedFileName := fmt.Sprintf("edited_%d_%s", time.Now().Unix(), consts.MetadataFileName)

				// 提示不要在程序运行的时候乱动数据库文件
				log.Println("Do not modify the Database file while the program is running, saving modified file and recovering Database file using current data")
				err := SaveYamlDB(consts.DB_path, editedFileName, savedDatabase)
				if err != nil {
					mess.PrintLogAndSave(fmt.Sprintln("some issues happend when saving modified Database:", err))
				} else {
					mess.PrintLogAndSave(fmt.Sprintf("The modified Database is saved to %s", consts.DB_path + editedFileName))
				}
				err = SaveYamlDB(consts.DB_path, consts.MetadataFileName, Database)
				if err != nil {
					mess.PrintLogAndSave(fmt.Sprintln("some issues happend when recovering Database:", err))
				} else {
					mess.PrintLogAndSave(fmt.Sprintf("The Database is recovered to %s", consts.DB_path + consts.MetadataFileName))
				}
			}
		} else { // 数据有更改，程序内的更新时间也比本地数据库晚，正常保存
			// 正常情况下更新时间就是会比程序内的时间晚，手动修改数据库途中如果有数据变动，而手动修改的时候没有修改时间戳，不会触发上面的保护机制，会直接覆盖手动修改的内容
			// 所以无论如何都尽量不要手动修改数据库文件，如果必要也请在开头添加专用的 `FORCEOVERWRITE: true` 覆写标记，或停止程序后再修改
			Database.UpdateTimestamp = time.Now().Unix()
			err := SaveYamlDB(consts.DB_path, consts.MetadataFileName, Database)
			if err != nil {
				mess.PrintLogAndSave(fmt.Sprintln("some issues happend when auto saving Database:", err))
			} else if consts.IsDebugMode {
				mess.PrintLogAndSave("auto save at " + time.Now().Format(time.RFC3339))
			}
		}
	}
}

// 初次添加群组时，获取必要信息
func AddChatInfo(chat *models.Chat) bool {
	for _, data := range Database.Data.IDs {
		if data.ID == chat.ID {
			return false // 群组已存在，不重复添加
		}
	}
	addToYamlDB(&IDInfo{
		ID:       chat.ID,
		ChatType: chat.Type,
		ChatName: utils.ShowChatName(chat),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	consts.SignalsChannel.Database_save <- true
	return true
}

func AddUserInfo(user *models.User) bool {
	for _, data := range Database.Data.IDs {
		if data.ID == user.ID {
			return false // 用户已存在，不重复添加
		}
	}
	addToYamlDB(&IDInfo{
		ID:       user.ID,
		ChatType: models.ChatTypePrivate,
		ChatName: utils.ShowUserName(user),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	consts.SignalsChannel.Database_save <- true
	return true
}

// 获取 ID 信息
func GetIDInfo(id *int64) *IDInfo {
	for Index, Data := range Database.Data.IDs {
		if Data.ID == *id {
			return &Database.Data.IDs[Index]
		}
	}
	return nil
}
