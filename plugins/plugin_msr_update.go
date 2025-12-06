package plugins

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"trle5.xyz/gopkg/trbot/utils"
	"trle5.xyz/gopkg/trbot/utils/configs"
	"trle5.xyz/gopkg/trbot/utils/plugin_utils"
	"trle5.xyz/gopkg/trbot/utils/task"
	"trle5.xyz/gopkg/trbot/utils/yaml"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
	"github.com/reugn/go-quartz/job"
	"github.com/reugn/go-quartz/quartz"
	"github.com/rs/zerolog"
)

var msrConfigPath string = filepath.Join(configs.YAMLDatabaseDir, "msr_update/", configs.YAMLFileName)

var msrConfig MSRUpdateConfig

var msrCachedDir string = filepath.Join(configs.CacheDir, "msr_conver/")

type MSRUpdateConfig struct {
	lock sync.Mutex
	botIns *bot.Bot

	ChannelID      int64  `yaml:"ChannelID"`
	SilentPost     bool   `yaml:"SilentPost"`
	APIBaseURL     string `yaml:"APIBaseURL"`
	PostedAlbumID  string `yaml:"PostedAlbumID"`

	TaskCron string `yaml:"TaskCron"`

	PostIntervalInMinute int `yaml:"PostIntervalInMinute"`
}

func DoAPIRequest(ctx context.Context, URI string, data any) error {
	resp, err := http.Get(msrConfig.APIBaseURL + URI)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return err
	}

	return nil
}

type AlbumsResponse struct {
	Code int     `json:"code"`
	Msg  string  `json:"msg"`
	Data []Album `json:"data"`
}

type Album struct {
	CID       string  `json:"cid"`
	Name      string  `json:"name"`
	ConverURL string  `json:"coverUrl"`
	Artistes  []string`json:"artistes"`
}

func GetAlbums(ctx context.Context) ([]Album, error) {
	var albumsResp AlbumsResponse

	err := DoAPIRequest(ctx, "albums", &albumsResp)
	if err != nil {
		return nil, fmt.Errorf("failed to get albums: %w", err)
	}

	if albumsResp.Code != 0 {
		return nil, fmt.Errorf("response code is not 0, msg: %s", albumsResp.Msg)
	}

	return albumsResp.Data, nil
}

func FindNewAlbums(ctx context.Context) ([]Album, error) {
	albums, err := GetAlbums(ctx)
	if err != nil {
		return nil, err
	}

	var needPost []Album

	for i := len(albums) - 1; i >= 0; i-- {
		if msrConfig.PostedAlbumID == albums[i].CID {
			needPost = albums[:i]
			break
		}
	}

	for i, j := 0, len(needPost)-1; i < j; i, j = i+1, j-1 {
		needPost[i], needPost[j] = needPost[j], needPost[i]
	}
	return needPost, nil
}

type AlbumDetailResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data AlbumDetail `json:"data"`
}

type AlbumDetail struct {
	CID        string `json:"cid"`
	Name       string `json:"name"`
	Intro      string `json:"intro"`
	Belong     string `json:"belong"`
	ConverURL  string `json:"coverUrl"`
	CoverDeURL string `json:"coverDeUrl"`

	// 暂时不需要
	Songs []Song `json:"songs"`
}

func (ad *AlbumDetail) Escape() {
	ad.Name = utils.IgnoreHTMLTags(strings.TrimSpace(ad.Name))
	ad.Intro = utils.IgnoreHTMLTags(strings.Trim(ad.Intro, "\n"))
}

func (ad AlbumDetail) GetLandscapeCoverImage(ctx context.Context) (io.Reader, error) {
	coverPath := filepath.Join(msrCachedDir, ad.CID + ".jpg")
	_, err := os.Stat(coverPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(msrCachedDir, 0755)
			if err != nil {
				return nil, fmt.Errorf("failed to create directory [%s] to cache cover image: %w", msrCachedDir, err)
			}

			resp, err := http.Get(ad.CoverDeURL)
			if err != nil {
				return nil, fmt.Errorf("failed to download cover image: %w", err)
			}
			defer resp.Body.Close()

			stickerfile, err := os.Create(coverPath)
			if err != nil {
				return nil, fmt.Errorf("failed to create cover image file [%s]: %w", coverPath, err)
			}
			defer stickerfile.Close()

			_, err = io.Copy(stickerfile, resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to writing data to cover image file [%s]: %w", coverPath, err)
			}
		} else {
			return nil, fmt.Errorf("failed to check cover image file [%s]: %w", coverPath, err)
		}
	}

	data, err := os.Open(coverPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open cover image file [%s]: %w", coverPath, err)
	}

	return data, nil
}

type Song struct {
	CID      string   `json:"cid"`
	Name     string   `json:"name"`
	Artistes []string `json:"artistes"`
}

func GetAlbumDetail(ctx context.Context, albumID string) (AlbumDetail, error) {
	var albumDetailResp AlbumDetailResponse

	err := DoAPIRequest(ctx, fmt.Sprintf("album/%s/detail", albumID), &albumDetailResp)
	if err != nil {
		return AlbumDetail{}, fmt.Errorf("failed to get album detail: %w", err)
	}

	if albumDetailResp.Code != 0 {
		return AlbumDetail{}, fmt.Errorf("failed to get album detail: %s", albumDetailResp.Msg)
	}

	return albumDetailResp.Data, nil
}

func init() {
	plugin_utils.AddInitializer(plugin_utils.Initializer{
		Name: "msr_update",
		Func: initMSRUpdate,
	})

	plugin_utils.AddDataBaseHandler(plugin_utils.DatabaseHandler{
		Name:   "msr_update",
		Loader: readMSRConfig,
		Saver:  saveMSRConfig,
	})
}

func initMSRUpdate(ctx context.Context, thebot *bot.Bot) error {
	msrConfig.botIns = thebot

	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "msr_update").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := readMSRConfig(ctx)
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to read msr update config")
		return fmt.Errorf("failed to read msr update config: %w", err)
	}

	trigger, err := quartz.NewCronTriggerWithLoc(msrConfig.TaskCron, time.FixedZone("CST", 8*3600))
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to create cron trigger")
		return fmt.Errorf("failed to create cron trigger: %w", err)
	}

	err = task.ScheduleTask(ctx, task.Task{
		Name:  "check_new_msr_album",
		Group: "msr_update",
		Job: job.NewFunctionJobWithDesc(
			MSRUpdateTask,
			"check new msr album and post",
		),
		Trigger: trigger,
	})
	if err != nil {
		logger.Error().
			Err(err).
			Msg("Failed to schedule task")
		return fmt.Errorf("failed to schedule task: %w", err)
	}

	return nil
}

func readMSRConfig(ctx context.Context) error {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "msr_update").
		Str(utils.GetCurrentFuncName()).
		Logger()

	err := yaml.LoadYAML(msrConfigPath, &msrConfig)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Warn().
				Err(err).
				Str("path", msrConfigPath).
				Msg("Not found msr update config file. Created new one")
			err = yaml.SaveYAML(msrConfigPath, &MSRUpdateConfig{
				APIBaseURL:           "https://monster-siren.hypergryph.com/api/",
				TaskCron:             "0 0 11,17 * * ?",
				PostIntervalInMinute: 5,
			})
			if err != nil {
				logger.Error().
					Err(err).
					Str("path", msrConfigPath).
					Msg("Failed to create empty config")
				return fmt.Errorf("failed to create empty config: %w", err)
			}
		} else {
			logger.Error().
				Err(err).
				Str("path", msrConfigPath).
				Msg("Failed to read config file")

			// 读取配置文件内容失败也不允许重新启动
			return fmt.Errorf("failed to read config file: %w", err)
		}
	}

	return err
}

func saveMSRConfig(ctx context.Context) error {
	err := yaml.SaveYAML(msrConfigPath, &msrConfig)
	if err != nil {
		zerolog.Ctx(ctx).Error().
			Str("pluginName", "msr_update").
			Str(utils.GetCurrentFuncName()).
			Err(err).
			Str("path", msrConfigPath).
			Msg("Failed to save msr update config")
		return fmt.Errorf("failed to save msr update config: %w", err)
	}

	return nil
}

func MSRUpdateTask(ctx context.Context) (int, error) {
	logger := zerolog.Ctx(ctx).
		With().
		Str("pluginName", "msr_update").
		Str(utils.GetCurrentFuncName()).
		Logger()

	if msrConfig.lock.TryLock() {
		defer msrConfig.lock.Unlock()
	} else {
		logger.Info().Msg("Another task is running, skip this run")
		return 0, nil
	}

	newAlbums, err := FindNewAlbums(ctx)
	if err != nil {
		logger.Err(err).
			Str("PostedAlbumID", msrConfig.PostedAlbumID).
			Msg("Failed to get new albums")
		return 1, fmt.Errorf("failed to get new albums: %w", err)
	}

	if len(newAlbums) == 0 {
		logger.Info().
			Str("PostedAlbumID", msrConfig.PostedAlbumID).
			Msg("No new albums found")
		return 0, nil
	}

	logger.Info().
		Str("PostedAlbumID", msrConfig.PostedAlbumID).
		Int("count", len(newAlbums)).
		// Interface("newAlbums", newAlbums).
		Msg("New albums found")

	for _, album := range newAlbums {
		logger.Info().
			Str("album", album.Name).
			Str("albumID", album.CID).
			Msg("New album found")
		detail, err := GetAlbumDetail(ctx, album.CID)
		if err != nil {
			logger.Err(err).
				Str("album", album.Name).
				Str("albumID", album.CID).
				Msg("failed to get album detail")
			return 1, fmt.Errorf(`failed to get new %s:"%s" album detail: %w`, album.CID, album.Name, err)
		}

		detail.Escape()

		logger.Info().
			Str("Name", detail.Name).
			Str("Intro", detail.Intro).
			Str("CoverDeURL", detail.CoverDeURL).
			Msg("Album detail found")

		// 把图片下载到本地再发出去
		data, err := detail.GetLandscapeCoverImage(ctx)
		if err != nil {
			logger.Err(err).
				Str("album", album.Name).
				Str("albumID", album.CID).
				Msg("failed to get album cover image")
			return 1, fmt.Errorf(`failed to get new %s:"%s" album cover image: %w`, album.CID, album.Name, err)
		}

		_, err = msrConfig.botIns.SendPhoto(ctx, &bot.SendPhotoParams{
			ChatID: msrConfig.ChannelID,
			Photo: &models.InputFileUpload{
				Filename: fmt.Sprintf("%s.jpg", album.CID),
				Data: data,
			},
			ParseMode: models.ParseModeHTML,
			DisableNotification: msrConfig.SilentPost,
			Caption: fmt.Sprintf(
				"<b>%s</b>\n\n%s\n\nListen on\n<a href=\"https://monster-siren.hypergryph.com/m/music/%s\"><u>Monster Siren Records</u></a>",
				detail.Name, detail.Intro, detail.Songs[0].CID,
			),
		})
		if err != nil {
			logger.Err(err).
				Str("album", album.Name).
				Str("albumID", album.CID).
				Msg("Failed to send new album")
			return 1, fmt.Errorf(`failed to send new %s:"%s" album: %w`, album.CID, album.Name, err)
		}

		logger.Info().
			Str("album", album.Name).
			Str("albumID", album.CID).
			Msg("New album sent")

		msrConfig.PostedAlbumID = album.CID

		time.Sleep(time.Duration(msrConfig.PostIntervalInMinute) * time.Minute)
	}

	err = saveMSRConfig(ctx)
	if err != nil {
		logger.Err(err).
			Msg("Failed to save msr config after post")
		return 1, fmt.Errorf("failed to save msr config after post: %w", err)
	}

	logger.Info().
		Msg("MSR update task done")
	return 0, nil
}
