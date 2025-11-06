package yaml_db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"time"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/yaml"

	"github.com/rs/zerolog"
)

func readYAMLDB(ctx context.Context, pathToFile string) (*DataBaseYaml, error) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str(utils.GetCurrentFuncName()).
		Logger()

	var tempDatabase *DataBaseYaml
	err := yaml.LoadYAML(pathToFile, &tempDatabase)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", YAMLDatabasePath).
				Msg("Not found database file")
			// 找不到文件直接返回 nil
			return nil, nil
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
func saveYAMLDB(ctx context.Context, dir, name string, data any) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.SaveYAML(filepath.Join(dir, name), data)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", YAMLDatabasePath).
			Msg("Failed to save database")
		return fmt.Errorf("failed to save database: %w", err)
	}

	return nil
}

func (db *DataBaseYaml)AutoSaveDatabaseHandler(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("database", "yaml").
		Str(utils.GetCurrentFuncName()).
		Logger()

	// 先读取一下数据库文件
	databaseFile, err := readYAMLDB(ctx, YAMLDatabasePath)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", YAMLDatabasePath).
			Msg("Failed to read database file")
		return
	}

	// 加锁检查数据库
	db.rw.RLock()
	needRecover   := databaseFile == nil
	needOverwrite := databaseFile != nil && databaseFile.ForceOverwrite
	sameData      := databaseFile != nil && reflect.DeepEqual(databaseFile.Chats, db.Chats)
	sameTimestamp := databaseFile != nil && databaseFile.UpdateTimestamp == db.UpdateTimestamp
	fileNewer     := databaseFile != nil && databaseFile.UpdateTimestamp >= db.UpdateTimestamp
	db.rw.RUnlock()

	// 数据库文件为空，需要恢复
	if needRecover {
		db.rw.Lock()
		defer db.rw.Unlock()

		logger.Warn().
			Str("path", YAMLDatabasePath).
			Msg("The database file is empty, recover database file using current data")
		err = db.saveDatabaseNoLock(ctx)
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
		return
	}

	// 如果数据库文件中有设定专用的 `FORCEOVERWRITE: true` 覆写标记
	// 无论任何修改，先保存程序中的数据，再读取新的数据替换掉当前的数据并保存
	if needOverwrite {
		db.rw.Lock()
		defer db.rw.Unlock()

		logger.Warn().
			Str("path", YAMLDatabasePath).
			Msg("Detected `FORCEOVERWRITE: true` in database file, save current database to another file first")

		// 保存覆盖之前的数据库
		oldFileName := fmt.Sprintf("beforeOverwritten_%d_%s", time.Now().Unix(), configs.YAMLFileName)
		err := saveYAMLDB(ctx, configs.YAMLDatabaseDir, oldFileName, databaseFile)
		if err != nil {
			logger.Warn().
				Err(err).
				Str("dir", configs.YAMLDatabaseDir).
				Str("fileName", oldFileName).
				Msg("Failed to save the database before overwrite")
		} else {
			logger.Warn().
				Err(err).
				Str("dir", configs.YAMLDatabaseDir).
				Str("fileName", oldFileName).
				Msg("The database before overwrite is saved to another file")
		}

		// 以数据库文件中的数据覆盖当前的数据
		db.Chats           = databaseFile.Chats
		db.UpdateTimestamp = databaseFile.UpdateTimestamp
		err = db.saveDatabaseNoLock(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Str("path", YAMLDatabasePath).
				Msg("Failed to save the database after overwrite")
		} else {
			logger.Warn().
				Str("path", YAMLDatabasePath).
				Msg("Read new data and save it to the database file")
		}
		return
	}

	// 数据无变动
	if sameData && sameTimestamp {
		logger.Debug().Msg("Looks database no any change, skip autosave this time")
		return
	}

	// 数据库文件更新时间比程序中的更新时间更晚
	if fileNewer {
		db.rw.Lock()
		defer db.rw.Unlock()

		logger.Warn().Msg("The database file is newer than current data in the program")
		// 如果只是更新时间有差别，更新一下时间，再保存就行
		if sameData {
			logger.Warn().
				Msg("But current data and database is the same, updating UpdateTimestamp in the database only")
			db.UpdateTimestamp = time.Now().Unix()
			err := db.saveDatabaseNoLock(ctx)
			if err != nil {
				logger.Error().
					Err(err).
					Msg("Failed to save database after updating UpdateTimestamp")
			}
		} else {
			// 数据库文件与程序中的数据不同，提示不要在程序运行的时候乱动数据库文件
			logger.Warn().
				Str("notice", "Do not modify the database file while the program is running, If you want to overwrite the current database, please add the field `FORCEOVERWRITE: true` at the beginning of the file").
				Msg("The database file is different from the current database, saving modified file and recovering database file using current data in the program")

			// 将新的数据文件改名另存为 `edited_时间戳_文件名`，再使用程序中的数据还原数据文件
			editedFileName := fmt.Sprintf("edited_%d_%s", time.Now().Unix(), configs.YAMLFileName)
			err := saveYAMLDB(ctx, configs.YAMLDatabaseDir, editedFileName, databaseFile)
			if err != nil {
				logger.Error().
					Err(err).
					Str("dir", configs.YAMLDatabaseDir).
					Str("fileName", editedFileName).
					Msg("Failed to save modified database")
			} else {
				logger.Warn().
					Str("dir", configs.YAMLDatabaseDir).
					Str("fileName", editedFileName).
					Msg("The modified database is saved to another file")
			}
			err = db.saveDatabaseNoLock(ctx)
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
		return
	}

	// 正常保存流程
	db.rw.Lock()
	defer db.rw.Unlock()
	// 数据有更改，程序内的更新时间也比本地数据库晚，正常保存
	// 无论如何都尽量不要手动修改数据库文件，如果必要也请在开头添加专用的 `FORCEOVERWRITE: true` 覆写标记，或停止程序后再修改
	db.UpdateTimestamp = time.Now().Unix()
	err = db.saveDatabaseNoLock(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("Failed to auto save database")
	} else {
		logger.Debug().
			Str("path", YAMLDatabasePath).
			Msg("The database is auto saved")
	}
}
