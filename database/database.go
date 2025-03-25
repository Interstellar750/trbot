package database

import (
	"context"
	"fmt"
	"log"
	"trbot/database/database_struct"

	"github.com/go-telegram/bot/models"
)

type DatabaseBackend struct {
	// 数据库名称
	Name string

	// 数据库等级，低优先级的数据库不会实时同步更改，程序仅会在高优先级数据库不可用才会尝试使用其中的数据
	IsLowLevel bool

	// 是否已被成功初始化
	IsInitialized bool
	InitializedErr error

	// 操作数据库的函数
	InitUser              func(ctx context.Context, user *models.User) error
	InitChat              func(ctx context.Context, chat *models.Chat) error
	GetChatInfo           func(ctx context.Context, id int64) (*database_struct.ChatInfo, error)
	IncrementalUsageCount func(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_UsageCount) error
	RecordLatestData      func(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_LatestData, value string) error
	UpdateOperationStatus func(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_Status, value bool) error
	SetCustomFlag         func(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_CustomFlag, value string) error
}

// var AvailableDB AvailableDatabase
var DBBackends []DatabaseBackend
var DBBackends_LowLevel []DatabaseBackend

func AddDatabaseBackend(backends ...DatabaseBackend) int {
	if DBBackends == nil { DBBackends = []DatabaseBackend{} }
	if DBBackends_LowLevel == nil { DBBackends_LowLevel = []DatabaseBackend{} }

	var count int
	for _, backend := range backends {
		if backend.IsInitialized {
			if backend.IsLowLevel {
				DBBackends_LowLevel = append(DBBackends_LowLevel, backend)
			} else {
				DBBackends = append(DBBackends, backend)
			}
			log.Printf("Initialized database backend [%s]", backend.Name)
			count++
		} else {
			log.Printf("Failed to initialize database backend [%s]: %s", backend.Name, backend.InitializedErr)
		}
	}

	return count
}


func ListDatabaseCount() {
	// AddDatabaseBackend(databaseBackend{
	// 	name: "redis",
	// 	InitDatabase: db_redis.Init,
	// 	InitUser: db_redis.InitUser,
	// 	InitChat: db_redis.InitChat,
	// 	GetChatInfo: db_redis.GetChatInfo,
	// })

	for _, backend := range DBBackends {
		log.Printf("Database backend [%s] is available", backend.Name)
	}
	for _, backend := range DBBackends_LowLevel {
		log.Printf("Database backend [%s] is available", backend.Name)
	}

	if len(DBBackends) + len(DBBackends_LowLevel) == 0 {
		log.Fatalln("No database available")
	} else {
		log.Printf("Available databases: [H: %d, L: %d]", len(DBBackends), len(DBBackends_LowLevel))
	}
}


func InitChat(ctx context.Context, chat *models.Chat) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.InitChat(ctx, chat)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.InitChat(ctx, chat)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func InitUser(ctx context.Context, user *models.User) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.InitUser(ctx, user)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.InitUser(ctx, user)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func GetChatInfo(ctx context.Context, chatID int64) (*database_struct.ChatInfo, error) {
	// 优先从高优先级数据库获取数据
	for _, db := range DBBackends {
		return db.GetChatInfo(ctx, chatID)
	}
	for _, db := range DBBackends_LowLevel {
		return db.GetChatInfo(ctx, chatID)
	}
	return nil, fmt.Errorf("no database available")
}

func IncrementalUsageCount(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_UsageCount) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.IncrementalUsageCount(ctx, chatID, fieldName)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.IncrementalUsageCount(ctx, chatID, fieldName)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func RecordLatestData(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_LatestData, data string) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.RecordLatestData(ctx, chatID, fieldName, data)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.RecordLatestData(ctx, chatID, fieldName, data)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func UpdateOperationStatus(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_Status, value bool) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.UpdateOperationStatus(ctx, chatID, fieldName, value)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.UpdateOperationStatus(ctx, chatID, fieldName, value)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}

func SetCustomFlag(ctx context.Context, chatID int64, fieldName database_struct.ChatInfoField_CustomFlag, value string) error {
	var allErr error
	for _, db := range DBBackends {
		err := db.SetCustomFlag(ctx, chatID, fieldName, value)
		if err != nil {
			allErr = err
		}
	}
	for _, db := range DBBackends_LowLevel {
		err := db.SetCustomFlag(ctx, chatID, fieldName, value)
		if err != nil {
			allErr = fmt.Errorf("%s, %s", allErr, err)
		}
	}
	return allErr
}
