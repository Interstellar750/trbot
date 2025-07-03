package yaml

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// 一个通用的 yaml 结构体读取函数
func LoadYAML(pathToFile string, out interface{}) error {
	file, err := os.ReadFile(pathToFile)
	if err == nil {
		err = yaml.Unmarshal(file, out)
	}

	return err
}

// 一个通用的 yaml 结构体保存函数，目录和文件不存在则创建，并以结构体类型保存
func SaveYAML(pathToFile string, data interface{}) error {
	out, err := yaml.Marshal(data)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(pathToFile), 0755)
		if err == nil {
			err = os.WriteFile(pathToFile, out, 0644)
		}
	}

	return err
}
