package teamspeak

// diffSlices 比较两个 OnlineClient 类型切片，返回新增和删除的字符串类型切片
func diffSlices(before, now []OnlineClient) (added, removed []string) {
	beforeMap := make(map[string]bool)
	nowMap    := make(map[string]bool)

	// 把 A 和 B 转成 map
	for _, item := range before { beforeMap[item.Username] = true }
	for _, item := range now    { nowMap[item.Username]    = true }

	// 删除
	for item := range nowMap {
		if !beforeMap[item] { added = append(added, item)}
	}

	// 新增
	for item := range beforeMap {
		if !nowMap[item] { removed = append(removed, item) }
	}

	return
}
