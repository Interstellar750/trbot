package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Metadata struct {
	Name string `yaml:"name"` // 语音包名称
	Voices []struct {
		ID       string `yaml:"ID"`       // 语音 ID
		Title    string `yaml:"Title"`    // 行内模式时显示的标题
		Caption  string `yaml:"Caption"`  // 发送后在语音下方的文字
		VoiceURL string `yaml:"VoiceURL"` // 音频文件网络链接
	} `yaml:"voices"`
}

func readMetadataFile(path string) (*Metadata, error) {
	file, err := os.Open(path)
	if err != nil { return nil, fmt.Errorf("error opening file: %v", err) }
	defer file.Close()

	var metadata Metadata
	decoder := yaml.NewDecoder(file)
	err = decoder.Decode(&metadata)
	if err != nil { return nil, fmt.Errorf("error decoding yaml: %v", err) }

	return &metadata, nil
}

func readMetadataFromDir(dir string) ([]*Metadata, error) {
	var metadataList []*Metadata

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil { return err }
		if info.Name() == "metadata.yaml" {
			metadata, err := readMetadataFile(path)
			if err != nil { return err }
			metadataList = append(metadataList, metadata)
		}
		return nil
	})
	if err != nil { return nil, err }
	
	return metadataList, nil
}
