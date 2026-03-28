package tooling

import (
	"context"
	"encoding/json"
	"fmt"
)

// registry 是工具名到工具实现的最小映射表。
type registry struct {
	tools map[string]Tool
}

// newRegistry 创建空的工具注册表。
func newRegistry() *registry {
	return &registry{tools: make(map[string]Tool)}
}

// Register 向注册表中加入一个工具；同名工具会直接 panic，避免运行期歧义。
func (r *registry) Register(t Tool) {
	if _, exists := r.tools[t.Name()]; exists {
		panic("duplicate tool: " + t.Name())
	}
	r.tools[t.Name()] = t
}

// List 返回当前已注册的全部工具。
func (r *registry) List() []Tool {
	out := make([]Tool, 0, len(r.tools))
	for _, t := range r.tools {
		out = append(out, t)
	}
	return out
}

// Call 根据工具名执行对应工具。
func (r *registry) Call(ctx context.Context, name string, raw json.RawMessage) (any, error) {
	tool, ok := r.tools[name]
	if !ok {
		return nil, fmt.Errorf("未知工具：%s", name)
	}
	return tool.Call(ctx, raw)
}

// DefaultToolbox 是当前默认的工具聚合实现。
type DefaultToolbox struct {
	registry *registry
	summary  ToolSummarizer
}

// NewToolbox 创建一个带默认摘要器的工具箱。
func NewToolbox() *DefaultToolbox {
	return &DefaultToolbox{
		registry: newRegistry(),
		summary:  NewDefaultToolSummarizer(),
	}
}

// Register 向工具箱中注册一个业务工具。
func (b *DefaultToolbox) Register(t Tool) {
	b.registry.Register(t)
}
