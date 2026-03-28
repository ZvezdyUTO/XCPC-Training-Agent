package tooling

import (
	stdctx "context"
	"encoding/json"

	agentmodel "aATA/internal/logic/agent/model"
)

// CallResult 是工具执行后的统一返回结构。
type CallResult struct {
	Name    string
	Result  any
	Summary map[string]any
}

// Toolbox 是 runtime 眼中的工具集合接口。
type Toolbox interface {
	Definitions() []agentmodel.ToolDefinition
	Call(ctx stdctx.Context, name string, input json.RawMessage) (CallResult, error)
}
