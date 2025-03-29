package plugin_utils

import (
	"fmt"
	"log"
)

type DatabaseHandler struct {
	Name   string
	Loader func()
	Saver  func() error
}

func AddDataBaseHandler(InlineHandlerPlugins ...DatabaseHandler) int {
	if AllPugins.Databases == nil {
		AllPugins.Databases = []DatabaseHandler{}
	}
	var pluginCount int
	for _, originPlugin := range InlineHandlerPlugins {
		AllPugins.Databases = append(AllPugins.Databases, originPlugin)
		pluginCount++
	}
	return pluginCount
}

func ReloadPluginsDatabase() {
	for _, plugin := range AllPugins.Databases {
		if plugin.Loader == nil { continue }
		plugin.Loader()
	}
}

func SavePluginsDatabase() string {
	dbCount := len(AllPugins.Databases)
	successCount := 0
	for _, plugin := range AllPugins.Databases {
		if plugin.Saver == nil { continue }
		err := plugin.Saver()
		if err != nil {
			log.Println("Plugin", plugin.Name, "failed to save:", err)
		} else {
			successCount++
		}
	}
	return fmt.Sprintf("[plugin_utils] Saved %d/%d plugins database", successCount, dbCount)
}
