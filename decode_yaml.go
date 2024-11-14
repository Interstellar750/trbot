package main

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Metadata struct {
	VoicesName string `yaml:"name,omitempty"` // 语音包名称
	Voices []struct {
		ID       string `yaml:"ID,omitempty"`       // 语音 ID
		Title    string `yaml:"Title,omitempty"`    // 行内模式时显示的标题
		Caption  string `yaml:"Caption,omitempty"`  // 发送后在语音下方的文字
		VoiceURL string `yaml:"VoiceURL,omitempty"` // 音频文件网络链接
	} `yaml:"voices,omitempty"`
	EnabledForwardGroupID []struct {
		ID     int64 `yaml:"id,omitempty"`
		Enable bool  `yaml:"enable,omitempty"`
	} `yaml:"GroupID,omitempty"`
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

// 将群组配置保存到 YAML 文件
func saveMetadata(path string, name string, Metadata *Metadata) error {
	data, err := yaml.Marshal(Metadata)
	if err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("failed to create directory: %v", err)
		}
	}

	if _, err := os.Stat(path + name); os.IsNotExist(err) {
		_, err := os.Create(path + name)
		if err != nil {
			return fmt.Errorf("failed to create file: %v", err)
		}
	}

	return os.WriteFile(path + name, data, 0644)
}
