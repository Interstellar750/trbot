package yaml_db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"
	"trbot/database/db_struct"
	"trbot/utils"
	"trbot/utils/consts"
	"trbot/utils/yaml"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var Database DataBaseYaml

var YAMLDatabasePath = filepath.Join(consts.YAMLDataBaseDir, consts.YAMLFileName)

// 需要重构，错误信息不足

type DataBaseYaml struct {
	// 如果运行中希望程序强制读取新数据，在 YAML 数据库文件的开头添加 FORCEOVERWRITE: true 即可
	ForceOverwrite bool `yaml:"FORCEOVERWRITE,omitempty"`

	UpdateTimestamp int64 `yaml:"UpdateTimestamp"`
	Data struct {
		ChatInfo []db_struct.ChatInfo `yaml:"ChatInfo"`
	} `yaml:"Data"`
}

func InitializeDB(ctx context.Context) error {
	if consts.YAMLDataBaseDir != "" {
		err := ReadDatabase(ctx)
		if err != nil {
			return fmt.Errorf("failed to read yaml database: %s", err)
		}
		return nil
	} else {
		return fmt.Errorf("yaml database path is empty")
	}
}

func SaveDatabase(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str("funcName", "SaveDatabase").
		Logger()

	Database.UpdateTimestamp = time.Now().Unix()
	err := yaml.SaveYAML(YAMLDatabasePath, &Database)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", YAMLDatabasePath).
			Msg("Failed to save database")
		return fmt.Errorf("failed to save database: %w", err)
	}

	return nil
}

func ReadDatabase(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str("funcName", "ReadDatabase").
		Logger()

	err := yaml.LoadYAML(YAMLDatabasePath, &Database)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", YAMLDatabasePath).
				Msg("Not found database file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(YAMLDatabasePath, &Database)
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

func ReadYamlDB(ctx context.Context, pathToFile string) (*DataBaseYaml, error) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str("funcName", "ReadYamlDB").
		Logger()

	var tempDatabase *DataBaseYaml
	err := yaml.LoadYAML(pathToFile, &tempDatabase)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", YAMLDatabasePath).
				Msg("Not found database file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(YAMLDatabasePath, &tempDatabase)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", YAMLDatabasePath).
					Msg("Failed to create empty database file")
				return nil, fmt.Errorf("failed to create empty database file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", YAMLDatabasePath).
				Msg("Failed to read database file")
			return nil, fmt.Errorf("failed to read database file: %w", err)
		}
	}

	return tempDatabase, nil
}

// 路径 文件名 YAML 数据结构体
func SaveYamlDB(ctx context.Context, path, name string, tempDatabase interface{}) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str("funcName", "SaveDatabase").
		Logger()

	err := yaml.SaveYAML(filepath.Join(path, name), &tempDatabase)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", YAMLDatabasePath).
			Msg("Failed to save database")
		return fmt.Errorf("failed to save database: %w", err)
	}

	return nil
}

func AutoSaveDatabaseHandler(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str("funcName", "AutoSaveDatabaseHandler").
		Logger()

	// 先读取一下数据库文件
	savedDatabase, err := ReadYamlDB(ctx, YAMLDatabasePath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", YAMLDatabasePath).
			Msg("Failed to read database file")
	} else {
		// 如果读取数据库文件时发现数据库为空，使用当前的数据重建数据库文件
		if savedDatabase == nil {
			logger.Warn().
				Str("path", YAMLDatabasePath).
				Msg("The database file is empty, recover database file using current data")
			err = SaveYamlDB(ctx, consts.YAMLDataBaseDir, consts.YAMLFileName, Database)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", YAMLDatabasePath).
					Msg("Failed to recover database file using current data")
			} else {
				logger.Warn().
					Str("path", YAMLDatabasePath).
					Msg("The database file is recovered using current data")
			}
		} else if reflect.DeepEqual(*savedDatabase, Database) {
			// 没有修改就跳过保存
			logger.Debug().Msg("Looks database no any change, skip autosave this time")
		} else {
			// 如果数据库文件中有设定专用的 `FORCEOVERWRITE: true` 覆写标记
			// 无论任何修改，先保存程序中的数据，再读取新的数据替换掉当前的数据并保存
			if savedDatabase.ForceOverwrite {
				logger.Warn().
					Str("path", YAMLDatabasePath).
					Msg("Detected `FORCEOVERWRITE: true` in database file, save current database to another file first")

				oldFileName := fmt.Sprintf("beforeOverwritten_%d_%s", time.Now().Unix(), consts.YAMLFileName)
				oldFilePath := filepath.Join(consts.YAMLDataBaseDir, oldFileName)

				err := SaveYamlDB(ctx, consts.YAMLDataBaseDir, oldFileName, savedDatabase)
				if err != nil {
					logger.Warn().
						Err(err).
						Str("oldPath", oldFilePath).
						Msg("Failed to save the database before overwrite")
				} else {
					logger.Warn().
						Err(err).
						Str("oldPath", oldFilePath).
						Msg("The database before overwrite is saved to another file")
				}
				Database = *savedDatabase
				Database.ForceOverwrite = false // 移除强制覆盖标记
				err = SaveYamlDB(ctx, consts.YAMLDataBaseDir, consts.YAMLFileName, Database)
				if err != nil {
					logger.Error().
						Err(err).
						Str("path", YAMLDatabasePath).
						Msg("Failed to save the database after overwrite")
				} else {
					logger.Warn().
						Str("path", YAMLDatabasePath).
						Msg("Read new database file and save to the old file")
				}
			} else {
				// 没有设定覆写标记，检测到本地的数据库更新时间比程序中的更新时间更晚
				if savedDatabase.UpdateTimestamp >= Database.UpdateTimestamp {
					logger.Warn().
						Msg("The database file is newer than current data in the program")
					// 如果只是更新时间有差别，更新一下时间，再保存就行
					if reflect.DeepEqual(savedDatabase.Data, Database.Data) {
						logger.Warn().
							Msg("But current data and database is the same, updating UpdateTimestamp in the database only")
						Database.UpdateTimestamp = time.Now().Unix()
						err := SaveYamlDB(ctx, consts.YAMLDataBaseDir, consts.YAMLFileName, Database)
						if err != nil {
							logger.Error().
								Err(err).
								Msg("Failed to save database after updating UpdateTimestamp")
						}
					} else {
						// 数据库文件与程序中的数据不同，提示不要在程序运行的时候乱动数据库文件
						logger.Warn().
							Str("notice", "Do not modify the database file while the program is running, If you want to overwrite the current database, please add the field `FORCEOVERWRITE: true` at the beginning of the file.").
							Msg("The database file is different from the current database, saving modified file and recovering database file using current data in the program")

						// 将新的数据文件改名另存为 `edited_时间戳_文件名`，再使用程序中的数据还原数据文件
						editedFileName := fmt.Sprintf("edited_%d_%s", time.Now().Unix(), consts.YAMLFileName)
						editedFilePath := filepath.Join(consts.YAMLDataBaseDir, editedFileName)

						err := SaveYamlDB(ctx, consts.YAMLDataBaseDir, editedFileName, savedDatabase)
						if err != nil {
							logger.Error().
								Err(err).
								Str("editedPath", editedFilePath).
								Msg("Failed to save modified database")
						} else {
							logger.Warn().
								Str("editedPath", editedFilePath).
								Msg("The modified database is saved to another file")
						}
						err = SaveYamlDB(ctx, consts.YAMLDataBaseDir, consts.YAMLFileName, Database)
						if err != nil {
							logger.Error().
								Err(err).
								Str("path", YAMLDatabasePath).
								Msg("Failed to recover database file")
						} else {
							logger.Warn().
								Str("path", YAMLDatabasePath).
								Msg("The database file is recovered using current data in the program")
						}
					}
				} else {
					// 数据有更改，程序内的更新时间也比本地数据库晚，正常保存
					// 无论如何都尽量不要手动修改数据库文件，如果必要也请在开头添加专用的 `FORCEOVERWRITE: true` 覆写标记，或停止程序后再修改
					Database.UpdateTimestamp = time.Now().Unix()
					err := SaveYamlDB(ctx, consts.YAMLDataBaseDir, consts.YAMLFileName, Database)
					if err != nil {
						logger.Error().
							Err(err).
							Msg("Failed to auto save database")
					} else {
						logger.Debug().
							Str("path", YAMLDatabasePath).
							Msg("The database is auto saved")
					}
				}
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
	Database.Data.ChatInfo = append(Database.Data.ChatInfo, db_struct.ChatInfo{
		ID:       chat.ID,
		ChatType: chat.Type,
		ChatName: utils.ShowChatName(chat),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	return SaveDatabase(ctx)
}

func InitUser(ctx context.Context, user *models.User) error {
	for _, data := range Database.Data.ChatInfo {
		if data.ID == user.ID {
			return nil // 用户已存在，不重复添加
		}
	}
	Database.Data.ChatInfo = append(Database.Data.ChatInfo, db_struct.ChatInfo{
		ID:       user.ID,
		ChatType: models.ChatTypePrivate,
		ChatName: utils.ShowUserName(user),
		AddTime:  time.Now().Format(time.RFC3339),
	})
	return SaveDatabase(ctx)
}

// 获取 ID 信息
func GetChatInfo(ctx context.Context, id int64) (*db_struct.ChatInfo, error) {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == id {
			return &Database.Data.ChatInfo[Index], nil
		}
	}
	return nil, fmt.Errorf("ChatInfo not found")
}

func IncrementalUsageCount(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_UsageCount) error {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == chatID {
			Database.UpdateTimestamp = time.Now().Unix() + 1
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

func RecordLatestData(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_LatestData, value string) error {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == chatID {
			Database.UpdateTimestamp = time.Now().Unix() + 1
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

func UpdateOperationStatus(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_Status, value bool) error {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == chatID {
			Database.UpdateTimestamp = time.Now().Unix() + 1
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

func SetCustomFlag(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_CustomFlag, value string) error {
	for Index, Data := range Database.Data.ChatInfo {
		if Data.ID == chatID {
			Database.UpdateTimestamp = time.Now().Unix() + 1
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
