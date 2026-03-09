package teamspeak

// diffSlices 比较两个 OnlineClient 类型切片，返回新增和删除的字符串类型切片
func diffSlices(before, now []OnlineClient) (added, removed []string) {
	beforeMap := make(map[OnlineClient]bool)
	nowMap    := make(map[OnlineClient]bool)

	// 把 A 和 B 转成 map
	for _, item := range before { beforeMap[item] = true }
	for _, item := range now    { nowMap[item]    = true }

	// 删除
	for item := range nowMap {
		if !beforeMap[item] { added = append(added, item.Username)}
	}

	// 新增
	for item := range beforeMap {
		if !nowMap[item] { removed = append(removed, item.Username) }
	}

	return
}
