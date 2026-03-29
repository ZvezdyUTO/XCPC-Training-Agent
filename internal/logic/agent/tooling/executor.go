package tooling

import (
	"context"
	"encoding/json"
	"sort"
)

// Definitions 导出 tooling 域视角下的稳定工具规格列表。
func (b *DefaultToolbox) Definitions() []ToolSpec {
	tools := b.registry.List()
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name() < tools[j].Name()
	})

	defs := make([]ToolSpec, 0, len(tools))
	for _, tool := range tools {
		defs = append(defs, ToolSpec{
			Name:        tool.Name(),
			Description: tool.Description(),
			Schema:      tool.Schema(),
		})
	}
	return defs
}

// Call 执行指定工具，并返回原始工具结果。
func (b *DefaultToolbox) Call(ctx context.Context, name string, raw json.RawMessage) (CallResult, error) {
	result, err := b.registry.Call(ctx, name, raw)
	if err != nil {
		return CallResult{
			Name:   name,
			Result: map[string]string{"error": err.Error()},
		}, err
	}

	return CallResult{
		Name:   name,
		Result: result,
	}, nil
}
