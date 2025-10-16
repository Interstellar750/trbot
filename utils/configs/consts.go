package configs

import (
	"time"

	"github.com/go-telegram/bot/models"
)

var BotMe *models.User // 用于存储 bot 信息

var (
	Commit  string
	Branch  string
	Version string
	BuildAt string
	BuildOn string
	Changes string // uncommit files when build
	StartAt time.Time = time.Now()
)
