package main

import (
	"fmt"
)

// 检查群组 ID 是否存在于配置中
func isGroupEnabled(groupID int64, config *Metadata) bool {
	for _, info := range config.EnabledForwardGroupID {
		if info.ID == groupID && info.Enable {
			return true
		}
	}
	return false
}

// 添加群组 ID 到配置中
func addGroupID(groupID int64, config *Metadata) {
	for _, group := range config.EnabledForwardGroupID {
		if group.ID == groupID {
			return // 群组已存在，不重复添加
		}
	}
	if !isGroupEnabled(groupID, config) {
		config.EnabledForwardGroupID = append(
			config.EnabledForwardGroupID, struct {
				ID     int64 `yaml:"id,omitempty"`
				Enable bool  `yaml:"enable,omitempty"`
			}{
				ID:     groupID,
				Enable: false,
			},
		)
	}
}

// 启用或禁用群组的功能
func setForwardOnly(groupID int64, config *Metadata, enable bool) error {
	ID := findGroupByID(groupID, config)
	if ID != -1 {
		config.EnabledForwardGroupID[ID].Enable = enable
		fmt.Println(config.EnabledForwardGroupID[ID].Enable)
	} else {
		return fmt.Errorf("unknown groupID: %d", groupID)
	}
	return nil
}

// 查找群组 ID 是否已存在配置中
func findGroupByID(groupID int64, config *Metadata) int {
	for i := range config.EnabledForwardGroupID {
		if config.EnabledForwardGroupID[i].ID == groupID {
			return i
		}
	}
	return -1
}
