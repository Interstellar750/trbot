package plugin_utils

type Plugin_Initializer struct {
	InitFunction func() (bool, error)
	IsInitialized bool
	Error error
}

func AddPluginsInitializer(Plugins ...Plugin_Initializer) int {
	if AllPugins.Initializer == nil {
		AllPugins.Initializer = []Plugin_Initializer{}
	}

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.Initializer = append(AllPugins.Initializer, originPlugin)
		pluginCount++
	}
	return pluginCount
}


func InitPlugins() {
	for _, plugin := range AllPugins.Initializer {
		plugin.IsInitialized, plugin.Error = plugin.InitFunction()
	}
}
