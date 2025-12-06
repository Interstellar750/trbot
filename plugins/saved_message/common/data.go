package common

import (
	"context"
	"fmt"
	"os"

	"trle5.xyz/gopkg/trbot/utils"
	"trle5.xyz/gopkg/trbot/utils/configs"
	"trle5.xyz/gopkg/trbot/utils/yaml"

	"github.com/rs/zerolog"
)

func SaveSavedMessageList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.SaveYAML(SavedMessagePath, &SavedMessageList)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", SavedMessagePath).
			Msg("Failed to save savedmessage list")
		SavedMessageErr = fmt.Errorf("failed to save savedmessage list: %w", err)
	} else {
		SavedMessageErr = nil
	}

	return SavedMessageErr
}

func ReadSavedMessageList(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "Saved Message").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(SavedMessagePath, &SavedMessageList)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", SavedMessagePath).
				Msg("Not found savedmessage list file. Created new one")
			// 如果是找不到文件，新建一个
			err = yaml.SaveYAML(SavedMessagePath, &SavedMessageList)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", SavedMessagePath).
					Msg("Failed to create empty savedmessage list file")
				SavedMessageErr = fmt.Errorf("failed to create empty savedmessage list file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", SavedMessagePath).
				Msg("Failed to load savedmessage list file")
			SavedMessageErr = fmt.Errorf("failed to load savedmessage list file: %w", err)
		}
	} else {
		SavedMessageErr = nil
	}

	if SavedMessageList.NoticeChatID == 0 && len(configs.BotConfig.AdminIDs) > 0 {
		SavedMessageList.NoticeChatID = configs.BotConfig.AdminIDs[0]
	}

	// buildSavedMessageByMessageHandlers()
	return SavedMessageErr
}
