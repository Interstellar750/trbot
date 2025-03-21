package database

import (
	"context"
	"fmt"
	"log"
	"trbot/database/database_struct"
	"trbot/database/db_redis"
	"trbot/database/db_yaml"

	"github.com/go-telegram/bot/models"
)

type databaseBackend struct {
	// 数据库名称
	name string

	// 数据库等级，低优先级的数据库不会实时同步更改，程序仅会在高优先级数据库不可用才会尝试使用其中的数据
	IsLowLevel bool

	// 初始化数据库
	InitDatabase func(ctx context.Context) bool

	// 操作数据库的函数
	InitUser func(ctx context.Context, user *models.User) error
	InitChat func(ctx context.Context, chat *models.Chat) error
	GetChatInfo func(ctx context.Context, id int64) (*database_struct.ChatInfo, error)
}

// type AvailableDatabase struct {
// 	redis bool
// 	yaml  bool
// }

// var AvailableDB AvailableDatabase
var DBBackends []databaseBackend
var DBBackends_LowLevel []databaseBackend

func InitChat(ctx context.Context, chat *models.Chat) error {
	for _, db := range DBBackends {
		err := db.InitChat(ctx, chat)
		if err != nil {
			return err
		}
	}
	return nil
}

func InitUser(ctx context.Context, user *models.User) error {
	for _, db := range DBBackends {
		err := db.InitUser(ctx, user)
		if err != nil {
			return err
		}
	}
	return nil
}

func GetChatInfo(ctx context.Context, chatID int64) (*database_struct.ChatInfo, error) {
	// 优先从高级数据库获取数据
	for _, db := range DBBackends {
		return db.GetChatInfo(ctx, chatID)
	}
	for _, db := range DBBackends_LowLevel {
		return db.GetChatInfo(ctx, chatID)
	}
	return nil, fmt.Errorf("no database available")
}

func InitDatabases(ctx context.Context) {
	addDatabaseBackend(databaseBackend{
		name: "redis",
		InitDatabase: db_redis.Init,
		InitUser: db_redis.InitUser,
		InitChat: db_redis.InitChat,
		GetChatInfo: db_redis.GetChatInfo,
	})

	addDatabaseBackend(databaseBackend{
		name: "yaml",
		IsLowLevel: true,
		InitDatabase: db_yaml.Init,
		InitUser: db_yaml.InitUser,
		InitChat: db_yaml.InitChat,
		GetChatInfo: db_yaml.GetChatInfo,
	})

	for _, backend := range DBBackends {
		backend.InitDatabase(ctx)
		log.Println("Initialized database backend: ", backend.name)
	}

	if len(DBBackends) + len(DBBackends_LowLevel) == 0 {
		log.Fatalln("No database available")
	}
}

func addDatabaseBackend(backends ...databaseBackend) int {
	if DBBackends == nil { DBBackends = []databaseBackend{} }

	var count int
	for _, backend := range backends {
		if backend.IsLowLevel {
			DBBackends_LowLevel = append(DBBackends_LowLevel, backend)
		} else {
			DBBackends = append(DBBackends, backend)
		}
		count++
	}

	return count
}
