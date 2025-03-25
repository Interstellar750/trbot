package db_yaml

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"reflect"
	"time"
	"trbot/database"
	"trbot/database/database_struct"
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
		ChatInfo []database_struct.ChatInfo `yaml:"ChatInfo"`
	} `yaml:"Data"`
}

func init() {
	var IsInitialized bool = false
	var InitializedErr error

	if consts.DB_path != "" {
		var err error
		Database, err = ReadYamlDB(consts.DB_path + consts.MetadataFileName)
		if err != nil {
			InitializedErr = fmt.Errorf("read yaml db error: %s", err)
			IsInitialized = false
		}
		IsInitialized = true
	} else {
		InitializedErr = fmt.Errorf("DB path is empty")
		IsInitialized = false
	}

	database.AddDatabaseBackend(database.DatabaseBackend{
		Name:           "yaml",
		IsLowLevel:     true,
		IsInitialized:  IsInitialized,
		InitializedErr: InitializedErr,

		InitUser:              InitUser,
		InitChat:              InitChat,
		GetChatInfo:           GetChatInfo,
		IncrementalUsageCount: IncrementalUsageCount,
		RecordLatestData:      RecordLatestData,
		UpdateOperationStatus: UpdateOperationStatus,
		SetCustomFlag:         SetCustomFlag,
	})
}

func ReadYamlDB(pathToFile string) (DataBaseYaml, error) {
	file, err := os.Open(pathToFile)
	if err != nil {
		log.Println("[Database]: Not found Database file. Created new one")
		err = SaveYamlDB(consts.DB_path, consts.MetadataFileName, DataBaseYaml{})
		if err != nil {
			return DataBaseYaml{}, err
		} else {
			return DataBaseYaml{}, nil
		}
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
func addToYamlDB(params *database_struct.ChatInfo) {
	Database.Data.ChatInfo = append(Database.Data.ChatInfo, *params)
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
		log.Println("looks Database no any change, skip autosave this time")
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
func InitChat(ctx context.Context, chat *models.Chat) error {
	for _, data := range Database.Data.ChatInfo {
		if data.ID == chat.ID {
			return nil // 群组已存在，不重复添加
		}
	}
	addToYamlDB(&database_struct.ChatInfo{
		ID:       chat.ID,
		ChatType: chat.Type,
		ChatName: utils.ShowChatName(chat),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	consts.SignalsChannel.Database_save <- true
	return nil
}

func InitUser(ctx context.Context, user *models.User) error {
	for _, data := range Database.Data.ChatInfo {
		if data.ID == user.ID {
			return nil // 用户已存在，不重复添加
		}
	}
	addToYamlDB(&database_struct.ChatInfo{
		ID:       user.ID,
		ChatType: models.ChatTypePrivate,
		ChatName: utils.ShowUserName(user),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	consts.SignalsChannel.Database_save <- true
	return nil
}

// 获取 ID 信息
func GetChatInfo(ctx context.Context, id int64) (*database_struct.ChatInfo, error) {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == id {
			return &Database.Data.ChatInfo[Index], nil
		}
	}
	return nil, fmt.Errorf("ChatInfo not found")
}

func IncrementalUsageCount(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_UsageCount) error {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == chatID {
			v := reflect.ValueOf(&Database.Data.ChatInfo[Index]).Elem()
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).Name == string(fieldName) {
					v.Field(i).SetInt(v.Field(i).Int() + 1)
					return nil
				}
			}
		}
	}
	return fmt.Errorf("ChatInfo not found")
}

func RecordLatestData(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_LatestData, value string) error {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == chatID {
			v := reflect.ValueOf(&Database.Data.ChatInfo[Index]).Elem()
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).Name == string(fieldName) {
					v.Field(i).SetString(value)
					return nil
				}
			}
		}
	}
	return fmt.Errorf("ChatInfo not found")
}

func UpdateOperationStatus(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_Status, value bool) error {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == chatID {
			v := reflect.ValueOf(&Database.Data.ChatInfo[Index]).Elem()
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).Name == string(fieldName) {
					v.Field(i).SetBool(value)
					return nil
				}
			}
		}
	}
	return fmt.Errorf("ChatInfo not found")
}

func SetCustomFlag(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_CustomFlag, value string) error {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == chatID {
			v := reflect.ValueOf(&Database.Data.ChatInfo[Index]).Elem()
			for i := 0; i < v.NumField(); i++ {
				if v.Type().Field(i).Name == string(fieldName) {
					v.Field(i).SetString(value)
					return nil
				}
			}
		}
	}
	return fmt.Errorf("ChatInfo not found")
}
