package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/go-telegram/bot/models"
	"gopkg.in/yaml.v3"
)

type DataBaseYaml struct {
	Data struct {
		IDs []IDInfo `yaml:"IDs"`
		Admin []int64 `yaml:"Admin,omitempty"`
	} `yaml:"Data"`
}

type IDInfo struct {
	ID       int64           `yaml:"ID"`
	ChatType models.ChatType `yaml:"chatType"`
	AddTime  string          `yaml:"addTime,omitempty"`

	IsBlackList         bool `yaml:"isBlackList,omitempty"`
	IsBotAdmin          bool `yaml:"isBotAdmin,omitempty"`
	IsEnableForwardonly bool `yaml:"isEnableForwardonly,omitempty"`

	SavedMessage SavedMessage `yaml:"savedMessage,omitempty"`
	InlineAlias  InlineAlias  `yaml:"inlineAliases,omitempty"`
	CustomCommands CustomCommands `yaml:"customCommands,omitempty"`
}

type SavedMessage struct {
	Count int `yaml:"count"`
	Limit int `yaml:"limit,omitempty"`

	Item []SavedMessageItems `yaml:"item,omitempty"`
}

type SavedMessageItems struct {
	Type string `yaml:"type"`
	URL  string `yaml:"url"`
	Text string `yaml:"text"`
	Describe string `yaml:"describe"`
}

type InlineAlias struct {
	Text    string `yaml:"text,omitempty"`
	Image   string `yaml:"image,omitempty"`
	Voice   string `yaml:"voice,omitempty"`
	Video   string `yaml:"video,omitempty"`
	File    string `yaml:"file,omitempty"`
	Sticker string `yaml:"sticker,omitempty"`
}

type CustomCommands struct {
	Count int `yaml:"count"`
	Limit int `yaml:"limit,omitempty"`

	Item []CustomCommandsItems `yaml:"item,omitempty"`
}

type CustomCommandsItems struct {
	Text string `yaml:"text"`
	WithPrefix bool `yaml:"withPrefix,omitempty"`
	Prefix string `yaml:"prefix,omitempty"`
}

func ReadYamlDB(pathToFile string) (DataBaseYaml, error) {
	file, err := os.Open(pathToFile)
	if err != nil {
		log.Println("[Database]: Not found database file. Created new one")
		SaveYamlDB(db_path, metadatafile_name, DataBaseYaml{})
		return DataBaseYaml{}, err
	}
	defer file.Close()

	var database DataBaseYaml
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&database)
	if err != nil {
		if err == io.EOF {
			log.Println("[Database]: Database looks empty. now format it")
			SaveYamlDB(db_path, metadatafile_name, DataBaseYaml{})
			return DataBaseYaml{}, nil
		}
		return DataBaseYaml{}, err
	}

	return database, nil
}

func SaveYamlDB(path string, name string, database DataBaseYaml) error {
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

func listdatabase(db DataBaseYaml) {
	fmt.Println(db.Data.Admin)
	for index, data := range db.Data.IDs {
		// fmt.Printf("%d ID[%d]  ChatType[%s]  Admin[%v]  ForwardOnly[%v]\n", index, data.ID, data.ChatType, data.IsBotAdmin, data.IsEnableForwardonly)
		fmt.Printf("%3d ID[%15d]  ChatType[%-10s]  Admin[%-5v]  ForwardOnly[%-5v]\n", index, data.ID, data.ChatType, data.IsBotAdmin, data.IsEnableForwardonly)
	}
}

// 添加数据
func addToYamlDB(params *IDInfo) {
	database.Data.IDs = append(database.Data.IDs, *params)
}

// 初次添加群组时，获取必要信息
func AddChatID(chat models.Chat) bool {
	for _, data := range database.Data.IDs {
		if data.ID == chat.ID {
			return false // 群组已存在，不重复添加
		}
	}
	addToYamlDB(&IDInfo{
		ID: chat.ID,
		ChatType: chat.Type,
		AddTime: time.Now().Format(time.RFC3339),
	})
	savenow <- true
	return true
}

func saveDatabase(savenow chan bool) {
		// 创建一个 Ticker，每隔 1 秒触发一次
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop() // 确保程序退出时释放资源

		for {
			select {
			case <-ticker.C: // 每次 Ticker 触发时执行任务
				fmt.Println("auto save at", time.Now().Format(time.RFC3339))
				logToFile("auto save at " + time.Now().Format(time.RFC3339))
				SaveYamlDB(db_path, metadatafile_name, database)
			case <-savenow: // 收到停止信号时退出循环
				fmt.Println("save at", time.Now().Format(time.RFC3339))
				logToFile("save at " + time.Now().Format(time.RFC3339))
				SaveYamlDB(db_path, metadatafile_name, database)
				savenow <- false
			}
		}
	}

// 获取 ID 信息与在数据库中的位置
func getIDInfoAndIndex(id int64) (*IDInfo, int) {
	for Index, Data := range database.Data.IDs {
		if Data.ID == id {
			return &database.Data.IDs[Index], Index
		}
	}
	return nil, -1
}
