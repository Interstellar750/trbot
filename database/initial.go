package database

import (
	"context"
	"trbot/database/db_struct"
	"trbot/database/redis_db"
	"trbot/database/yaml_db"
	"trbot/utils"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

type DatabaseBackend struct {
	// 数据库名称
	Name string

	// 数据库等级，低优先级的数据库不会实时同步更改，程序仅会在高优先级数据库不可用才会尝试使用其中的数据
	IsLowLevel bool

	// 是否已被成功初始化
	Initializer    func() (bool, error)
	IsInitialized  bool
	InitializedErr error

	// 数据库保存和读取函数
	SaveDatabase func(ctx context.Context) error
	ReadDatabase func(ctx context.Context) error

	// 操作数据库的函数
	InitUser              func(ctx context.Context, user *models.User) error
	InitChat              func(ctx context.Context, chat *models.Chat) error
	GetChatInfo           func(ctx context.Context, id int64) (*db_struct.ChatInfo, error)
	IncrementalUsageCount func(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_UsageCount) error
	RecordLatestData      func(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_LatestData, value string) error
	UpdateOperationStatus func(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_Status, value bool) error
	SetCustomFlag         func(ctx context.Context, chatID int64, fieldName db_struct.ChatInfoField_CustomFlag, value string) error
}

var DBBackends []DatabaseBackend
var DBBackends_LowLevel []DatabaseBackend

func AddDatabaseBackend(ctx context.Context, backends ...DatabaseBackend) int {
	logger := zerolog.Ctx(ctx)

	if DBBackends == nil { DBBackends = []DatabaseBackend{} }
	if DBBackends_LowLevel == nil { DBBackends_LowLevel = []DatabaseBackend{} }

	var count int
	for _, db := range backends {
		db.IsInitialized, db.InitializedErr = db.Initializer()
		if db.IsInitialized {
 			if db.IsLowLevel {
				DBBackends_LowLevel = append(DBBackends_LowLevel, db)
			} else {
				DBBackends = append(DBBackends, db)
			}
			logger.Info().
				Str("database", db.Name).
				Str("databaseLevel", utils.TextForTrueOrFalse(db.IsLowLevel, "low", "high")).
				Msg("Database initialized")
			count++
		} else {
			logger.Error().
				Err(db.InitializedErr).
				Str("database", db.Name).
				Msg("Database initialize failed")
		}
	}

	return count
}

func InitAndListDatabases(ctx context.Context) {
	logger := zerolog.Ctx(ctx)
	AddDatabaseBackend(ctx, DatabaseBackend{
		Name:        "redis",
		Initializer: redis_db.InitializeDB,

		InitUser:              redis_db.InitUser,
		InitChat:              redis_db.InitChat,
		GetChatInfo:           redis_db.GetChatInfo,
		IncrementalUsageCount: redis_db.IncrementalUsageCount,
		RecordLatestData:      redis_db.RecordLatestData,
		UpdateOperationStatus: redis_db.UpdateOperationStatus,
		SetCustomFlag:         redis_db.SetCustomFlag,
	})

	AddDatabaseBackend(ctx, DatabaseBackend{
		Name:        "yaml",
		IsLowLevel:  true,
		Initializer: yaml_db.InitializeDB,

		SaveDatabase: yaml_db.SaveDatabase,
		ReadDatabase: yaml_db.ReadDatabase,

		InitUser:              yaml_db.InitUser,
		InitChat:              yaml_db.InitChat,
		GetChatInfo:           yaml_db.GetChatInfo,
		IncrementalUsageCount: yaml_db.IncrementalUsageCount,
		RecordLatestData:      yaml_db.RecordLatestData,
		UpdateOperationStatus: yaml_db.UpdateOperationStatus,
		SetCustomFlag:         yaml_db.SetCustomFlag,
	})

	// for _, backend := range DBBackends {
	// 	logger.Info().
	// 		Str("database", backend.Name).
	// 		Str("level", "high").
	// 		Msg("database initialized")
	// }
	// for _, backend := range DBBackends_LowLevel {
	// 	logger.Info().
	// 		Str("database", backend.Name).
	// 		Str("level", "low").
	// 		Msg("database initialized")
	// }

	if len(DBBackends) + len(DBBackends_LowLevel) == 0 {
		logger.Fatal().
			Msg("No database available")
	} else {
		logger.Info().
			Int("High-level", len(DBBackends)).
			Int("Low-level", len(DBBackends_LowLevel)).
			Msg("Available databases")
	}
}
