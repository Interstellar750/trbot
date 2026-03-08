package yaml_db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"trle5.xyz/trbot/database/db_struct"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/configs"
	"trle5.xyz/trbot/utils/task"
	"trle5.xyz/trbot/utils/yaml"

	"github.com/go-telegram/bot/models"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
	"github.com/rs/zerolog"
)

var YAMLDatabasePath = filepath.Join(configs.YAMLDatabaseDir, configs.YAMLFileName)

func Initialize(ctx context.Context) (*DataBaseYaml, error) {
	var db DataBaseYaml
	if configs.YAMLDatabaseDir == "" {
		return nil, fmt.Errorf("yaml database path is empty")
	}

	err := db.ReadDatabase(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to read yaml database: %w", err)
	}

	err = task.ScheduleTask(ctx, task.Task{
		Name:    "save_yaml_database",
		Group:   "trbot",
		Job:     job.NewFunctionJobWithDesc(
			func(ctx context.Context) (int, error) {
				db.AutoSaveDatabaseHandler(ctx)
				return 0, nil
			},
			"Save yaml database every 10 minutes",
		),
		Trigger: quartz.NewSimpleTrigger(10 * time.Minute),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to add auto save database task: %w", err)
	}

	return &db, nil
}

type DataBaseYaml struct {
	rw sync.RWMutex
	// 如果运行中希望程序强制读取新数据，在 YAML 数据库文件的开头添加 FORCEOVERWRITE: true 即可
	ForceOverwrite  bool  `yaml:"FORCEOVERWRITE,omitempty"`
	UpdateTimestamp int64 `yaml:"UpdateTimestamp"`

	Chats []db_struct.ChatInfo `yaml:"Chats"`
}

func (db *DataBaseYaml)Name() string {
	return "YAML"
}

func (db *DataBaseYaml)saveDatabaseNoLock(ctx context.Context) error {
	db.UpdateTimestamp = time.Now().Unix()
	err := yaml.SaveYAML(YAMLDatabasePath, &db)
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Err(err).
			Str("database", "yaml").
			Str(utils.GetCurrentFuncName()).
			Str("path", YAMLDatabasePath).
			Msg("Failed to save database")
		return fmt.Errorf("failed to save database: %w", err)
	}

	return nil
}

func (db *DataBaseYaml)readDatabaseNoLock(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(YAMLDatabasePath, &db)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", YAMLDatabasePath).
				Msg("Not found database file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(YAMLDatabasePath, &db)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", YAMLDatabasePath).
					Msg("Failed to create empty database file")
				return fmt.Errorf("failed to create empty database file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", YAMLDatabasePath).
				Msg("Failed to read database file")
			return  fmt.Errorf("failed to read database file: %w", err)
		}
	}

	return nil
}

func (db *DataBaseYaml)SaveDatabase(ctx context.Context) error {
	db.rw.RLock()
	defer db.rw.RUnlock()
	return db.saveDatabaseNoLock(ctx)
}

func (db *DataBaseYaml)ReadDatabase(ctx context.Context) error {
	db.rw.Lock()
	defer db.rw.Unlock()
	return db.readDatabaseNoLock(ctx)
}

// 获取 ID 信息
func (db *DataBaseYaml)GetChatInfo(ctx context.Context, id int64) (*db_struct.ChatInfo, error) {
	db.rw.RLock()
	defer db.rw.RUnlock()

	for _, data := range db.Chats {
		if data.ID == id {
			return &data, nil
		}
	}
	return nil, fmt.Errorf("ChatInfo not found")
}

// 初次添加群组时，获取必要信息
func (db *DataBaseYaml)InitChat(ctx context.Context, chat *models.Chat) error {
	db.rw.Lock()
	defer db.rw.Unlock()

	for _, data := range db.Chats {
		if data.ID == chat.ID {
			return nil // 群组已存在，不重复添加
		}
	}

	db.Chats = append(db.Chats, db_struct.ChatInfo{
		ID:       chat.ID,
		ChatType: chat.Type,
		ChatName: utils.ShowChatName(chat),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	return db.saveDatabaseNoLock(ctx)
}

func (db *DataBaseYaml)InitUser(ctx context.Context, user *models.User) error {
	db.rw.Lock()
	defer db.rw.Unlock()

	for _, data := range db.Chats {
		if data.ID == user.ID {
			return nil // 用户已存在，不重复添加
		}
	}
	db.Chats = append(db.Chats, db_struct.ChatInfo{
		ID:       user.ID,
		ChatType: models.ChatTypePrivate,
		ChatName: utils.ShowUserName(user),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	return db.saveDatabaseNoLock(ctx)
}

func (db *DataBaseYaml)IncrementalUsageCount(ctx context.Context, chatID int64, fieldName db_struct.UsageCount) error {
	db.rw.Lock()
	defer db.rw.Unlock()

	for index, data := range db.Chats {
		if data.ID == chatID {
			db.UpdateTimestamp = time.Now().Unix() + 1
			if data.UsageCount == nil { db.Chats[index].UsageCount = map[db_struct.UsageCount]int{} }
			usage, isExist := data.UsageCount[fieldName]
			if isExist {
				db.Chats[index].UsageCount[fieldName] = usage + 1
			} else {
				db.Chats[index].UsageCount[fieldName] = 1
			}
			return nil
		}
	}
	return fmt.Errorf("ChatInfo not found")
}

func (db *DataBaseYaml)RecordLatestData(ctx context.Context, chatID int64, fieldName db_struct.LatestData, value string) error {
	db.rw.Lock()
	defer db.rw.Unlock()

	for index, data := range db.Chats {
		if data.ID == chatID {
			db.UpdateTimestamp = time.Now().Unix() + 1
			if data.LatestData == nil { db.Chats[index].LatestData = map[db_struct.LatestData]string{} }
			db.Chats[index].LatestData[fieldName] = value
			return nil
		}
	}
	return fmt.Errorf("ChatInfo not found")
}

func (db *DataBaseYaml)UpdateOperationStatus(ctx context.Context, chatID int64, fieldName db_struct.Status, value bool) error {
	db.rw.Lock()
	defer db.rw.Unlock()

	for index, data := range db.Chats {
		if data.ID == chatID {
			db.UpdateTimestamp = time.Now().Unix() + 1
			if data.Status == nil { db.Chats[index].Status = map[db_struct.Status]bool{} }
			db.Chats[index].Status[fieldName] = value
			return nil
		}
	}
	return fmt.Errorf("ChatInfo not found")
}

func (db *DataBaseYaml)SetCustomFlag(ctx context.Context, chatID int64, fieldName db_struct.Flag, value string) error {
	db.rw.Lock()
	defer db.rw.Unlock()
	for index, data := range db.Chats {
		if data.ID == chatID {
			db.UpdateTimestamp = time.Now().Unix() + 1
			if data.Flag == nil { db.Chats[index].Flag = map[db_struct.Flag]string{} }
			db.Chats[index].Flag[fieldName] = value
			return nil
		}
	}
	return fmt.Errorf("ChatInfo not found")
}
