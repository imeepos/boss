package apiroutes

// ByName 按端名取路由目录;不存在返回 nil。
// 手写辅助(生成文件 routes_gen.go 勿改)。
func ByName(name string) *Portal {
	for i := range Portals {
		if Portals[i].Name == name {
			return &Portals[i]
		}
	}
	return nil
}
