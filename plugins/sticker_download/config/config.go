package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"trbot/utils"
	"trbot/utils/configs"
	"trbot/utils/yaml"

	"github.com/rs/zerolog"
)

var CachedDir     string = filepath.Join(configs.CacheDir, "sticker/")
var ConvertedDir  string = filepath.Join(configs.CacheDir, "sticker_converted/")
var CompressedDir string = filepath.Join(configs.CacheDir, "sticker_compressed/")
var TempDir       string = filepath.Join(configs.CacheDir, "sticker_temp/")

var StickerConfigPath string = filepath.Join(configs.YAMLDatabaseDir, "sticker/", configs.YAMLFileName)

var Config StickerConfigs

type StickerConfigs struct {
	DisableConvert          bool `yaml:"DisableConvert"`
	UseCollcetSticker       bool `yaml:"UseCollcetSticker"`
	AllowDownloadStickerSet bool `yaml:"AllowDownloadStickerSet"`

	FFmpegPath      string `yaml:"FFmpegPath"`
	GifskiPath      string `yaml:"GifskiPath"`
	LottieToPNGPath string `yaml:"LottieToPNGPath"`

	TGSConvertFPS int `yaml:"TGSConvertFPS"`
}

func ReadStickerConfig(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "sticker").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(StickerConfigPath, &Config)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", StickerConfigPath).
				Msg("Not found sticker config file. Created new one")
			err = yaml.SaveYAML(StickerConfigPath, &Config)
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", StickerConfigPath).
					Msg("Failed to create empty sticker config file")
				return fmt.Errorf("failed to create empty sticker config file: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", StickerConfigPath).
				Msg("Failed to read sticker config file")
			return fmt.Errorf("failed to read sticker config file: %w", err)
		}
	}

	return nil
}

func SaveStickerConfig(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "sticker").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.SaveYAML(StickerConfigPath, &Config)
	if err != nil {
		logger.Error().
			Err(err).
			Str("path", StickerConfigPath).
			Msg("Failed to save sticker config file")
		return fmt.Errorf("failed to save sticker config file: %w", err)
	}
	return nil
}
