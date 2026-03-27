/*
定义统一接口，避免 controller 绑定 模型：
Complete(ctx, prompt) (string, error)
*/

package llm

import "context"

type Client interface {
	Complete(ctx context.Context, prompt string) (string, error)
}

type Descriptor interface {
	ModelName() string
}
