package database

import (
	"context"
	"trbot/database/db_struct"
	"trbot/database/yaml_db"

	"github.com/go-telegram/bot/models"
	"github.com/rs/zerolog"
)

var database DatabaseBackend

type DatabaseBackend interface {
	Name() string

	// 数据库保存和读取函数
	SaveDatabase(ctx context.Context) error
	ReadDatabase(ctx context.Context) error

	// 操作数据库的函数
	GetChatInfo(ctx context.Context, id int64) (*db_struct.ChatInfo, error)

	InitUser(ctx context.Context, user *models.User) error
	InitChat(ctx context.Context, chat *models.Chat) error

	SetCustomFlag        (ctx context.Context, chatID int64, fieldName db_struct.Flag,       value string) error
	RecordLatestData     (ctx context.Context, chatID int64, fieldName db_struct.LatestData, value string) error
	UpdateOperationStatus(ctx context.Context, chatID int64, fieldName db_struct.Status,     value bool  ) error
	IncrementalUsageCount(ctx context.Context, chatID int64, fieldName db_struct.UsageCount              ) error
}

func InitDatabase(ctx context.Context) {
	var err error

	database, err = yaml_db.Initialize(ctx)
	if err != nil {
		zerolog.Ctx(ctx).Fatal().
			Err(err).
			Msg("Failed to initialize database")
	} else {
		zerolog.Ctx(ctx).Info().
			Str("name", database.Name()).
			Msg("Database initialized")
	}
}
