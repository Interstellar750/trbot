package teamspeak

import (
	"context"
	"fmt"
	"time"

	"github.com/go-telegram/bot"
	"github.com/multiplay/go-ts3"
	"github.com/rs/zerolog"
	"trle5.xyz/trbot/utils"
	"trle5.xyz/trbot/utils/flaterr"
)

// Connect 检查配置并尝试连接到 TeamSpeak 服务器
func (sc *ServerConfig) Connect(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Int64("GroupID", sc.GroupID).
		Logger()

	var err error
	sc.rw.Lock()
	defer sc.rw.Unlock()

	// 如果服务器地址为空不允许重新启动
	if sc.URL == "" {
		logger.Error().
			Str("path", tsConfigPath).
			Msg("No URL in config")
		return fmt.Errorf("no URL in config")
	}

	if sc.c != nil {
		logger.Info().Msg("Closing client...")
		err = sc.c.Close()
		if err != nil {
			logger.Error().
				Err(err).
				Str("path", tsConfigPath).
				Msg("Failed to close client")
		}
		// 尽管它有时会返回错误，但可能还是会正常关闭，所以等一等，然后抛弃旧的客户端
		time.Sleep(10 * time.Second)
		sc.c = nil
	}

	logger.Info().Msg("Starting client...")
	sc.c, err = ts3.NewClient(sc.URL)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsConfigPath).
			Msg("Failed to connect to server")
		return fmt.Errorf("failed to connnect to server: %w", err)
	}

	logger.Info().Msg("Checking credentials...")
	// ServerQuery 账号名或密码为空也不允许重新启动
	if sc.Name == "" || sc.Password == "" {
		logger.Error().
			Str("path", tsConfigPath).
			Msg("No Name/Password in config")
		return fmt.Errorf("no Name/Password in config")
	}

	logger.Info().Msg("Logining...")
	err = sc.c.Login(sc.Name, sc.Password)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsConfigPath).
			Msg("Failed to login to server")
		return fmt.Errorf("failed to login to server: %w", err)
	}

	logger.Info().Msg("Checking Group ID...")
	// 检查要设定通知的群组 ID 是否存在
	if sc.GroupID == 0 {
		logger.Error().
			Str("path", tsConfigPath).
			Msg("No GroupID in config")
		return fmt.Errorf("no GroupID in config")
	}

	logger.Info().Msg("Testing connection...")
	// 显示服务端版本测试一下连接
	v, err := sc.c.Version()
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", tsConfigPath).
			Msg("Failed to get server version")
		return fmt.Errorf("failed to get server version: %w", err)
	}

	logger.Info().
		Str("version", v.Version).
		Str("platform", v.Platform).
		Int("build", v.Build).
		Msg("TeamSpeak server connected")

	logger.Info().Msg("Switching virtual servers...")

	// 切换默认虚拟服务器
	err = sc.c.Use(1)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to switch server")
		return fmt.Errorf("failed to switch server: %w", err)
	}

	logger.Info().Msg("Checking nickname...")

	// 改一下 bot 自己的 nickname，使得在检测用户列表时默认不显示自己
	m, err := sc.c.Whoami()
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to get bot info")
		return fmt.Errorf("failed to get bot info: %w", err)
	} else if m != nil && m.ClientName != botNickName {
		logger.Info().Msg("Setting nickname...")
		// 当 bot 自己的 nickname 不等于配置文件中的 nickname 时，才进行修改
		err = sc.c.SetNick(botNickName)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to set bot nickname")
			return fmt.Errorf("failed to set nickname: %w", err)
		}
	}

	logger.Info().Msg("Successfully connected!")

	if !sc.s.IsCheckClientTaskScheduled {
		err = sc.ScheduleCheckClientTask(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to schedule check client task")
			return fmt.Errorf("failed to schedule check client task: %w", err)
		}
	}

	if !sc.s.IsDeleteMessageTaskScheduled {
		err = sc.ScheduleDeleteMessageTask(ctx)
		if err != nil {
			logger.Error().
				Err(err).
				Msg("Failed to schedule delete message task")
			return fmt.Errorf("failed to schedule delete message task: %w", err)
		}
	}

	return nil
}

// RetryLoop 将循环尝试连接到服务器，直到成功为止
func (sc *ServerConfig) RetryLoop(ctx context.Context) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "teamspeak3").
		Str(utils.GetCurrentFuncName()).
		Logger()

	sc.s.IsInRetryLoop = true
	defer func() {
		sc.s.IsInRetryLoop = false
		logger.Info().Msg("reconnect loop exited")
	}()
	sc.s.RetryCount = 0

	retryTicker := time.NewTicker(time.Second * 5)
	defer retryTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-sc.s.ResetTicker:
			sc.ResumeCheckClientTask(ctx)
			logger.Info().Msg("reconnect by command success")
			return
		case <-retryTicker.C:
			logger.Info().
				Int("retryCount", sc.s.RetryCount).
				Msg("try reconnect...")

			// 实际上并不生效...
			timeoutCtx, cancel := context.WithTimeout(ctx, time.Second * 30)
			defer cancel()

			err := sc.Connect(timeoutCtx)
			if err != nil {
				// 出现错误时，先降低 ticker 速度，然后尝试重新初始化
				// 无法成功则等待下一个周期继续尝试
				if sc.s.RetryCount < 15 {
					sc.s.RetryCount++
					retryTicker.Reset(time.Duration(sc.s.RetryCount * 20) * time.Second)
				}

				logger.Warn().
					Int("retryCount", sc.s.RetryCount).
					Time("nextRetry", time.Now().Add(time.Duration(sc.s.RetryCount * 20) * time.Second)).
					Msg("reconnect failed")
			} else {
				// 重新初始化成功则恢复查询任务
				sc.ResumeCheckClientTask(ctx)
				logger.Info().Msg("reconnect success")
				botMessage, err := botInstance.SendMessage(ctx, &bot.SendMessageParams{
					ChatID: sc.GroupID,
					Text:   "已成功与服务器重新建立连接",
				})
				if err != nil {
					logger.Error().
						Err(err).
						Int64("chatID", sc.GroupID).
						Str("content", "success reconnect to server notice").
						Msg(flaterr.SendMessage.Str())
				} else {
					sc.s.OldMessageID = append(tsConfig.s.OldMessageID, OldMessageID{
						Date: int(time.Now().Unix()),
						ID:   botMessage.ID,
					})
				}
				return
			}
		}
	}
}
