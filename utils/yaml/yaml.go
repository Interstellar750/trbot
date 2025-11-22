package yaml

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// 一个通用的 yaml 结构体读取函数
//
// 如果 `out` 结构体中有字段没有导出（以大写开头），那这个字段不会被读取到
//
// If there is a field in the `out` structure that is not exported (starting with an uppercase letter), this field cannot be read.
func LoadYAML(pathToFile string, out any) error {
	file, err := os.ReadFile(pathToFile)
	if err == nil {
		err = yaml.Unmarshal(file, out)
	}

	return err
}

// 一个通用的 yaml 结构体保存函数，目录和文件不存在则创建，并以结构体类型保存
//
// 如果 `data` 结构体中有字段没有导出（以大写开头），那这个字段不会被保存
//
// If there is a field in the `data` structure that is not exported (starting with an uppercase letter), this field cannot be save.
func SaveYAML(pathToFile string, data any) error {
	out, err := yaml.Marshal(data)
	if err == nil {
		err = os.MkdirAll(filepath.Dir(pathToFile), 0755)
		if err == nil {
			err = os.WriteFile(pathToFile, out, 0644)
		}
	}

	return err
}
