package context

// appliedMemoryNames 把已生效的 memory bundle 转成便于 trace 暴露的名称列表。
func appliedMemoryNames(bundle Bundle) []string {
	names := make([]string, 0, len(bundle.Rules)+1)
	if bundle.Project != "" {
		names = append(names, "project")
	}
	for _, rule := range bundle.Rules {
		if rule.Name != "" {
			names = append(names, "rule:"+rule.Name)
		}
	}
	return names
}
