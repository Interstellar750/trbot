package signals

import (
	"context"
	"fmt"
	"log"
	"time"
	"trbot/database/yaml_db"
	"trbot/utils/consts"
	"trbot/utils/mess"
	"trbot/utils/plugin_utils"
)

func SignalsHandler(ctx context.Context, SIGNAL consts.SignalChannel) {
	every10Min := time.NewTicker(10 * time.Minute)
	defer every10Min.Stop()

	// additional.AdditionalDatas = additional.ReadAdditionalDatas(consts.AdditionalDatas_paths)

	for {
		select {
		case <-every10Min.C: // 每次 Ticker 触发时执行任务
			yaml_db.AutoSaveDatabaseHandler()
		case <-ctx.Done():
			log.Println("Cancle signal received")
			yaml_db.AutoSaveDatabaseHandler()
			log.Println("Database saved")
			SIGNAL.WorkDone <- true
			return
		case <-SIGNAL.Database_save:
			yaml_db.Database.UpdateTimestamp = time.Now().Unix()
			err := yaml_db.SaveYamlDB(consts.DB_path, consts.MetadataFileName, yaml_db.Database)
			if err != nil {
				mess.PrintLogAndSave(fmt.Sprintln("some issues happend when some function call save database now:", err))
			} else {
				mess.PrintLogAndSave("save at " + time.Now().Format(time.RFC3339))
			}
		case <-SIGNAL.PluginDB_reload:
			plugin_utils.ReloadPluginsDatabase()
			log.Println("Plugin Database reloaded")
		case <-SIGNAL.PluginDB_save:
			mess.PrintLogAndSave(plugin_utils.SavePluginsDatabase())
			// log.Println("Plugin Database saved")
		}
		
	}
}
