package tooling

import (
	"context"
	"encoding/json"
)

// ToolSchema 描述一个工具对外暴露的输入参数结构。
type ToolSchema struct {
	Parameters map[string]Parameter
	Required   []string
}

// Parameter 描述单个工具参数的元信息。
type Parameter struct {
	Type        string
	Description string
	Enum        []string
}

// Tool 是业务工具需要实现的最小能力接口。
type Tool interface {
	Name() string
	Description() string
	Schema() ToolSchema
	Call(ctx context.Context, input json.RawMessage) (any, error)
}
