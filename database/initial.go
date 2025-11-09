package database

import (
	"context"
	"trbot/database/db_struct"
	"trbot/database/yaml_db"
	"trbot/utils/task"

	"github.com/go-telegram/bot/models"
	"github.com/reugn/go-quartz/job"
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
	logger := zerolog.Ctx(ctx)
	var err error

	database, err = yaml_db.Initialize(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to initialize database")
	}

	err = task.ScheduleTask(ctx, task.Task{
		Name:    "save_database",
		Group:   "trbot",
		Job:     job.NewFunctionJobWithDesc(
			func(ctx context.Context) (int, error) {
				return 0, database.SaveDatabase(ctx)
			},
			"Save database",
		),
		Trigger: nil,
	})
	if err != nil {
		logger.Fatal().
			Err(err).
			Str("taskName", "save_database").
			Msg("Failed to add save database task")
	}

	logger.Info().
		Str("name", database.Name()).
		Msg("Database initialized")
}
