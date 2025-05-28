package yaml

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// 一个通用的 yaml 结构体读取函数
func LoadYAML(pathToFile string, out interface{}) error {
	file, err := os.ReadFile(pathToFile)
	if err != nil {
		return fmt.Errorf("read file failed: %w", err)
	}

	if err := yaml.Unmarshal(file, out); err != nil {
		return fmt.Errorf("decode yaml failed: %w", err)
	}

	return nil
}

// 一个通用的 yaml 结构体保存函数，目录和文件不存在则创建，并以结构体类型保存
func SaveYAML(pathToFile string, data interface{}) error {
	out, err := yaml.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML: %w", err)
	}

	dir := filepath.Dir(pathToFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create directory failed: %w", err)
	}

	if err := os.WriteFile(pathToFile, out, 0644); err != nil {
		return fmt.Errorf("write data failed: %w", err)
	}

	return nil
}
