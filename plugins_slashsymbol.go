package main

type Plugin_SlashSymbolCommand struct {
	slashCommand string // 'command' in '/command'
	handler func(*subHandlerOpts)
}

func AddSlashSymbolCommandPlugins(Plugins ...Plugin_SlashSymbolCommand) int {
	if AllPugins.SlashSymbolCommand == nil { AllPugins.SlashSymbolCommand = []Plugin_SlashSymbolCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.SlashSymbolCommand = append(AllPugins.SlashSymbolCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
