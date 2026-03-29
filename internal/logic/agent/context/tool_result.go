package context

// summarizeToolResult 在 context 内生成工具结果摘要。
// 摘要属于上下文状态维护策略，因此放在 context 中，而不是放在 tooling 中。
func summarizeToolResult(toolName string, patch ToolResultPatch) map[string]any {
	summary := map[string]any{
		"tool": toolName,
	}
	if len(patch.Args) > 0 {
		summary["args"] = cloneMap(patch.Args)
	}

	obj, ok := patch.Result.(map[string]any)
	if !ok {
		return summary
	}

	scalars := make(map[string]any)
	arrays := make(map[string]any)
	for key, value := range obj {
		switch typed := value.(type) {
		case nil, string, bool, int, int8, int16, int32, int64, float32, float64:
			scalars[key] = typed
		case []any:
			arrays[key] = len(typed)
		}
	}
	if len(scalars) > 0 {
		summary["scalars"] = scalars
	}
	if len(arrays) > 0 {
		summary["arrays"] = arrays
	}
	return summary
}

// cloneMap 复制一层顶层 map，避免外部继续修改参数结构导致状态漂移。
func cloneMap(input map[string]any) map[string]any {
	if len(input) == 0 {
		return nil
	}

	out := make(map[string]any, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}
