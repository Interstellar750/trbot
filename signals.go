package main

import (
	"context"
	"fmt"
	"log"
	"time"
)

type SignalChannel struct {
	Database_save          chan bool
	AdditionalDatas_reload chan bool
	WorkDone               chan bool
}

var SignalsChannel = SignalChannel{
	Database_save:          make(chan bool),
	AdditionalDatas_reload: make(chan bool),
	WorkDone:               make(chan bool),
}

func signalsHandler(ctx context.Context, SIGNAL SignalChannel) {
	every10Min := time.NewTicker(10 * time.Minute)
	defer every10Min.Stop()

	AdditionalDatas = readAdditionalDatas(AdditionalDatas_paths)

	for {
		select {
		case <-every10Min.C: // 每次 Ticker 触发时执行任务
			AutoSaveDatabaseHandler()
		case <-ctx.Done():
			log.Println("Cancle signal received")
			AutoSaveDatabaseHandler()
			log.Println("Database saved")
			SIGNAL.WorkDone <- true
			return
		case <-SIGNAL.Database_save:
			database.UpdateTimestamp = time.Now().Unix()
			err := SaveYamlDB(db_path, metadataFileName, database)
			if err != nil {
				printLogAndSave(fmt.Sprintln("some issues happend when some function call save database now:", err))
			} else {
				printLogAndSave("save at " + time.Now().Format(time.RFC3339))
			}
		case <-SIGNAL.AdditionalDatas_reload:
			AdditionalDatas = readAdditionalDatas(AdditionalDatas_paths)
			log.Println("AdditionalData reloaded")
		}
	}
}
