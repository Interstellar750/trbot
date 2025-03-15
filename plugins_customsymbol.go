package main

type Plugin_CustomSymbolCommand struct {
	fullCommand string
	handler func(*subHandlerOpts)
}

func AddCustomSymbolCommandPlugins(Plugins ...Plugin_CustomSymbolCommand) int {
	if AllPugins.CustomSymbolCommand == nil { AllPugins.CustomSymbolCommand = []Plugin_CustomSymbolCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.CustomSymbolCommand = append(AllPugins.CustomSymbolCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
