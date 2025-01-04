package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type VoicePack struct {
	Name string `yaml:"name,omitempty"` // 语音包名称
	Voices []struct {
		ID       string `yaml:"ID,omitempty"`       // 语音 ID
		Title    string `yaml:"Title,omitempty"`    // 行内模式时显示的标题
		Caption  string `yaml:"Caption,omitempty"`  // 发送后在语音下方的文字
		VoiceURL string `yaml:"VoiceURL,omitempty"` // 音频文件网络链接
	} `yaml:"voices,omitempty"`
}

// 读取指定目录下所有结尾为 .yaml 或 .yml 的语音文件
func readVoicePackFromPath(path string) ([]VoicePack, error) {
	var packs []VoicePack

	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Printf("No voices dir, create a new one: %s", voice_path)
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, err
		}
	}

	err := filepath.Walk(path, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		if strings.HasSuffix(info.Name(), ".yaml") || strings.HasSuffix(info.Name(), ".yml") {
			file, err := os.Open(path)
			if err != nil { log.Println("(func)readVoicesFromDir:", err) }
			defer file.Close()

			var singlePack VoicePack
			decoder := yaml.NewDecoder(file)
			err = decoder.Decode(&singlePack)
			if err != nil { log.Println("(func)readVoicesFromDir:", err) }
			packs = append(packs, singlePack)
		}
		return nil
	})
	if err != nil { return nil, err }
	
	return packs, nil
}
