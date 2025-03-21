package signals

import (
	"context"
	"fmt"
	"log"
	"time"
	"trbot/utils/consts"
	"trbot/database/db_yaml"
	"trbot/utils/mess"
)

func SignalsHandler(ctx context.Context, SIGNAL consts.SignalChannel) {
	every10Min := time.NewTicker(10 * time.Minute)
	defer every10Min.Stop()

	// additional.AdditionalDatas = additional.ReadAdditionalDatas(consts.AdditionalDatas_paths)

	for {
		select {
		// case <-every10Min.C: // 每次 Ticker 触发时执行任务
		// 	additional.AutoSaveDatabaseHandler()
		case <-ctx.Done():
			log.Println("Cancle signal received")
			db_yaml.AutoSaveDatabaseHandler()
			log.Println("Database saved")
			SIGNAL.WorkDone <- true
			return
		case <-SIGNAL.Database_save:
			db_yaml.Database.UpdateTimestamp = time.Now().Unix()
			err := db_yaml.SaveYamlDB(consts.DB_path, consts.MetadataFileName, db_yaml.Database)
			if err != nil {
				mess.PrintLogAndSave(fmt.Sprintln("some issues happend when some function call save database now:", err))
			} else {
				mess.PrintLogAndSave("save at " + time.Now().Format(time.RFC3339))
			}
		// case <-SIGNAL.AdditionalDatas_reload:
		// 	additional.AdditionalDatas = additional.ReadAdditionalDatas(consts.AdditionalDatas_paths)
		// 	log.Println("AdditionalData reloaded")
		}
	}
}
