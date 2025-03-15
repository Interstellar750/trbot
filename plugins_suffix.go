package main

type Plugin_SuffixCommand struct {
	suffixCommand string
	handler       func(*subHandlerOpts)
}

func AddSuffixCommandPlugins(Plugins ...Plugin_SuffixCommand) int {
	if AllPugins.SuffixCommand == nil { AllPugins.SuffixCommand = []Plugin_SuffixCommand{} }

	var pluginCount int
	for _, originPlugin := range Plugins {
		AllPugins.SuffixCommand = append(AllPugins.SuffixCommand, originPlugin)
		pluginCount++
	}
	return pluginCount
}
