package main

type Plugin_SlashStart struct {
	handler           []SlashStartHandler           // 例如 /start subcommand
	withPrefixHandler []SlashStartWithPrefixHandler // 例如 /start subcommand_augument
}

type SlashStartHandler struct {
	argument string
	handler  func(*subHandlerOpts)
}

func AddSlashStartCommandPlugins(SlashStartCommandPlugins ...SlashStartHandler) int {
	if AllPugins.SlashStart == nil {
		AllPugins.SlashStart = &Plugin_SlashStart{}
	}
	if AllPugins.SlashStart.handler == nil {
		AllPugins.SlashStart.handler = []SlashStartHandler{}
	}

	var pluginCount int
	for _, originPlugin := range SlashStartCommandPlugins {
		AllPugins.SlashStart.handler = append(AllPugins.SlashStart.handler, originPlugin)
		pluginCount++
	}
	return pluginCount
}

type SlashStartWithPrefixHandler struct {
	prefix   string
	argument string
	handler  func(*subHandlerOpts)
}

func AddSlashStartWithPrefixCommandPlugins(SlashStartWithPrefixCommandPlugins ...SlashStartWithPrefixHandler) int {
	if AllPugins.SlashStart == nil {
		AllPugins.SlashStart = &Plugin_SlashStart{}
	}
	if AllPugins.SlashStart.withPrefixHandler == nil {
		AllPugins.SlashStart.withPrefixHandler = []SlashStartWithPrefixHandler{}
	}

	var pluginCount int
	for _, originPlugin := range SlashStartWithPrefixCommandPlugins {
		AllPugins.SlashStart.withPrefixHandler = append(AllPugins.SlashStart.withPrefixHandler, originPlugin)
		pluginCount++
	}
	return pluginCount
}
