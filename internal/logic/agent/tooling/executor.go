package tooling

import (
	agentmodel "aATA/internal/logic/agent/model"
	"context"
	"encoding/json"
	"sort"
)

// Definitions 导出适合直接传给模型层的工具定义列表。
func (b *DefaultToolbox) Definitions() []agentmodel.ToolDefinition {
	tools := b.registry.List()
	sort.Slice(tools, func(i, j int) bool {
		return tools[i].Name() < tools[j].Name()
	})

	defs := make([]agentmodel.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		defs = append(defs, agentmodel.ToolDefinition{
			Type: "function",
			Function: agentmodel.FunctionDefinition{
				Name:        tool.Name(),
				Description: tool.Description(),
				Parameters:  buildToolParameters(tool.Schema()),
			},
		})
	}
	return defs
}

// Call 执行指定工具，并统一补齐对模型友好的摘要结果。
func (b *DefaultToolbox) Call(ctx context.Context, name string, raw json.RawMessage) (CallResult, error) {
	result, err := b.registry.Call(ctx, name, raw)
	if err != nil {
		return CallResult{
			Name:    name,
			Result:  map[string]string{"error": err.Error()},
			Summary: b.summary.Summarize(name, map[string]any{"error": err.Error()}),
		}, err
	}

	return CallResult{
		Name:    name,
		Result:  result,
		Summary: b.summary.Summarize(name, result),
	}, nil
}

// buildToolParameters 把内部 ToolSchema 转换成模型层所需的 JSON Schema 片段。
func buildToolParameters(schema ToolSchema) map[string]any {
	properties := make(map[string]any, len(schema.Parameters))
	for name, param := range schema.Parameters {
		property := map[string]any{
			"type":        param.Type,
			"description": param.Description,
		}
		if len(param.Enum) > 0 {
			property["enum"] = param.Enum
		}
		properties[name] = property
	}

	return map[string]any{
		"type":                 "object",
		"properties":           properties,
		"required":             schema.Required,
		"additionalProperties": false,
	}
}
