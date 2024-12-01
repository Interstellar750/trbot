package main

var botToken string // 全局 bot token
var logChat_ID int = -1002499888124 // 用于接收日志的聊天 ID，可以是 用户 群聊 频道

var metadatafile_name string = "metadata.yaml"

var db_path      string = "./db_yaml/"
var voice_path   string = db_path + "voices/"
var fwdonly_path string = db_path + "forwardonly/"
