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

	HasPendingCallbackQuery bool `yaml:"HasPendingCallbackQuery,omitempty"`
	LastedCallbackQueryData string `yaml:"LastedCallbackQueryData,omitempty"`

	SavedMessage   SavedMessage   `yaml:"SavedMessage,omitempty"`
	InlineAlias    InlineAlias    `yaml:"InlineAliases,omitempty"`
	CustomCommands CustomCommands `yaml:"CustomCommands,omitempty"`
}

type SavedMessage struct {
	Count int `yaml:"Count"`
	Limit int `yaml:"Limit,omitempty"`

	Item []SavedMessageItems `yaml:"Item,omitempty"`
}

type SavedMessageItems struct {
	Type     string `yaml:"Type"`
	URL      string `yaml:"Url"`
	Text     string `yaml:"Text"`
	Describe string `yaml:"Describe"`
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
	DB_savenow <- true
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
	DB_savenow <- true
	return true
}

func mainDatabaseHandler(DB_savenow chan bool) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C: // 每次 Ticker 触发时执行任务
			// 先读取一下数据库文件
			savedDatabase, err := ReadYamlDB(db_path + metadataFileName)
			if err != nil {
				log.Println("some issues when read database file", err)
				// 如果读取数据库文件时发现数据库为空，使用当前的数据重建数据库文件
				if reflect.DeepEqual(savedDatabase, DataBaseYaml{}){
					printLogAndSave(fmt.Sprintln("The database file is empty, recovering database file using current data"))
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
			if reflect.DeepEqual(savedDatabase, database) {
				fmt.Printf("\r%s looks database no any change, skip autosave this time", time.Now().Format(time.RFC3339))
			} else {
				// 如果检测到本地的数据库更新时间比程序中的更新时间更晚
				if savedDatabase.UpdateTimestamp > database.UpdateTimestamp {
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
						// 数据库文件与程序中的数据不同，需要处理
						nowtimestamp := time.Now().Unix()

						log.Println("Saved database is different from the current database")
						// 如果数据库文件中有设定专用的 `FORCEOVERWRITE: true` 覆写标记，保存程序中的数据，读取新的数据替换掉当前的数据并保存
						if savedDatabase.ForceOverwrite {
							printLogAndSave(fmt.Sprintf("The `FORCEOVERWRITE: true` in %s is detected", db_path + metadataFileName))
							oldFileName := fmt.Sprintf("beforeOverwritten_%d_%s", nowtimestamp, metadataFileName)
							err := SaveYamlDB(db_path, oldFileName, savedDatabase)
							if err != nil {
								printLogAndSave(fmt.Sprintln("some issues happend when saving the database before overwritten:", err))
							} else {
								printLogAndSave(fmt.Sprintf("The database before overwritten is saved to %s", db_path + oldFileName))
							}
							database = savedDatabase
							err = SaveYamlDB(db_path, metadataFileName, database)
							if err != nil {
								printLogAndSave(fmt.Sprintln("some issues happend when recreat database using new database:", err))
							} else {
								printLogAndSave(fmt.Sprintf("Success read data from the new file and saved to %s", db_path + metadataFileName))
							}
						} else {
							// 没有设定覆写标记，就将新的数据文件改名另存为 `edited_时间戳_文件名`，再使用程序中的数据还原数据文件
							editedFileName := fmt.Sprintf("edited_%d_%s", nowtimestamp, metadataFileName)

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
					}

				} else { // 数据有更改，程序内的更新时间也比本地数据库晚，正常保存
					database.UpdateTimestamp = time.Now().Unix()
					err := SaveYamlDB(db_path, metadataFileName, database)
					if err != nil {
						printLogAndSave(fmt.Sprintln("some issues happend when auto saving database:", err))
					} else {
						printLogAndSave("auto save at " + time.Now().Format(time.RFC3339))
					}
				}
			}
		case <-DB_savenow:
			database.UpdateTimestamp = time.Now().Unix()
			err := SaveYamlDB(db_path, metadataFileName, database)
			if err != nil {
				printLogAndSave(fmt.Sprintln("some issues happend when some function call save database now:", err))
			} else {
				printLogAndSave("save at " + time.Now().Format(time.RFC3339))
			}
		}
	}
}

// 获取 ID 信息与在数据库中的位置
func getIDInfoAndIndex(id *int64) (*IDInfo, int) {
	for Index, Data := range database.Data.IDs {
		if Data.ID == *id {
			return &database.Data.IDs[Index], Index
		}
	}
	return nil, -1
}
