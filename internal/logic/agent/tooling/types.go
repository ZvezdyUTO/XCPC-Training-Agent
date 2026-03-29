package tooling

import (
	stdctx "context"
	"encoding/json"
)

// CallResult 是工具执行后的统一返回结构。
type CallResult struct {
	Name   string
	Result any
}

// ToolSpec 是 tooling 域对外暴露的稳定工具规格。
// 它描述工具是什么，而不是 provider 期望的协议 JSON 长什么样。
type ToolSpec struct {
	Name        string
	Description string
	Schema      ToolSchema
}

// Toolbox 是 runtime 眼中的工具集合接口。
type Toolbox interface {
	Definitions() []ToolSpec
	Call(ctx stdctx.Context, name string, input json.RawMessage) (CallResult, error)
}
