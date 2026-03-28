package tooling

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// ToolSummarizer 负责把工具原始结果压缩成适合继续喂给模型的摘要。
type ToolSummarizer interface {
	Summarize(toolName string, result any) map[string]any
}

// DefaultToolSummarizer 是当前默认的轻量摘要策略实现。
type DefaultToolSummarizer struct{}

// NewDefaultToolSummarizer 创建默认摘要器。
func NewDefaultToolSummarizer() ToolSummarizer {
	return DefaultToolSummarizer{}
}

// Summarize 根据结果类型提取关键标量、身份字段、对象结构和数组预览。
func (DefaultToolSummarizer) Summarize(toolName string, result any) map[string]any {
	summary := map[string]any{
		"tool":        toolName,
		"result_type": typeName(result),
	}

	if result == nil {
		summary["status"] = "empty"
		return summary
	}

	if obj, ok := normalizeObject(result); ok {
		mergeSummary(summary, summarizeObject(obj, 0))
		return summary
	}

	if list, ok := normalizeArray(result); ok {
		summary["array"] = summarizeArray(list, 0)
		return summary
	}

	summary["value"] = summarizeScalar(result)
	return summary
}

// summarizeObject 压缩对象结构，优先保留身份字段和少量关键标量。
func summarizeObject(obj map[string]any, depth int) map[string]any {
	keys := sortedKeys(obj)
	out := map[string]any{
		"key_count": len(keys),
		"keys":      keys,
	}

	identity := make(map[string]any)
	scalars := make(map[string]any)
	objects := make(map[string]any)
	arrays := make(map[string]any)

	for _, key := range keys {
		value := obj[key]
		switch {
		case isIdentityKey(key) && isScalarLike(value):
			identity[key] = summarizeScalar(value)
		case isScalarLike(value):
			scalars[key] = summarizeScalar(value)
		default:
			if child, ok := normalizeObject(value); ok {
				objects[key] = summarizeNestedObject(child, depth+1)
				continue
			}
			if list, ok := normalizeArray(value); ok {
				arrays[key] = summarizeArray(list, depth+1)
				continue
			}
			scalars[key] = summarizeScalar(value)
		}
	}

	if len(identity) > 0 {
		out["identity"] = identity
	}
	if len(scalars) > 0 {
		out["scalars"] = limitMap(scalars, 8)
	}
	if len(objects) > 0 {
		out["objects"] = limitMap(objects, 4)
	}
	if len(arrays) > 0 {
		out["arrays"] = limitMap(arrays, 4)
	}

	return out
}

// summarizeNestedObject 用于处理对象中的嵌套对象，避免递归展开过深。
func summarizeNestedObject(obj map[string]any, depth int) map[string]any {
	keys := sortedKeys(obj)
	out := map[string]any{
		"key_count": len(keys),
		"keys":      limitStrings(keys, 8),
	}

	if depth > 1 {
		return out
	}

	scalars := make(map[string]any)
	for _, key := range keys {
		if len(scalars) >= 6 {
			break
		}
		if isScalarLike(obj[key]) {
			scalars[key] = summarizeScalar(obj[key])
		}
	}
	if len(scalars) > 0 {
		out["scalars"] = scalars
	}

	return out
}

// summarizeArray 生成数组长度、元素类型和少量预览项。
func summarizeArray(list []any, depth int) map[string]any {
	out := map[string]any{
		"count": len(list),
	}
	if len(list) == 0 {
		return out
	}

	if depth > 1 {
		out["item_type"] = typeName(list[0])
		return out
	}

	previews := make([]any, 0, min(2, len(list)))
	for _, item := range list[:min(2, len(list))] {
		if obj, ok := normalizeObject(item); ok {
			previews = append(previews, summarizeNestedObject(obj, depth+1))
			continue
		}
		previews = append(previews, summarizeScalar(item))
	}

	out["preview"] = previews
	out["item_type"] = typeName(list[0])
	return out
}

// summarizeScalar 将标量压缩成短字符串或原始数值。
func summarizeScalar(v any) any {
	switch x := v.(type) {
	case string:
		if len(x) <= 64 {
			return x
		}
		return x[:64] + "..."
	case bool, int, int8, int16, int32, int64, float32, float64:
		return x
	default:
		b, err := json.Marshal(x)
		if err != nil {
			return fmt.Sprintf("%T", x)
		}
		text := string(b)
		if len(text) <= 64 {
			return text
		}
		return text[:64] + "..."
	}
}

// normalizeObject 尝试把任意可编码值归一化为 map 结构。
func normalizeObject(v any) (map[string]any, bool) {
	switch x := v.(type) {
	case map[string]any:
		return x, true
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var out map[string]any
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, false
		}
		return out, true
	}
}

// normalizeArray 尝试把任意可编码值归一化为切片结构。
func normalizeArray(v any) ([]any, bool) {
	switch x := v.(type) {
	case []any:
		return x, true
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return nil, false
		}
		var out []any
		if err := json.Unmarshal(b, &out); err != nil {
			return nil, false
		}
		return out, true
	}
}

// sortedKeys 返回对象的有序键列表，保证摘要输出稳定。
func sortedKeys(obj map[string]any) []string {
	keys := make([]string, 0, len(obj))
	for key := range obj {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// isIdentityKey 判断一个字段名是否更适合作为身份信息保留。
func isIdentityKey(key string) bool {
	key = strings.ToLower(strings.TrimSpace(key))
	switch key {
	case "student_id", "contest_id", "platform", "from", "to", "date", "contest_date", "contest_name", "name":
		return true
	default:
		return strings.HasSuffix(key, "_id")
	}
}

// isScalarLike 判断一个值是否可以直接作为轻量标量保留。
func isScalarLike(v any) bool {
	switch v.(type) {
	case nil, string, bool, int, int8, int16, int32, int64, float32, float64:
		return true
	default:
		return false
	}
}

// limitMap 裁剪 map，避免摘要里保留过多字段。
func limitMap(input map[string]any, limit int) map[string]any {
	if len(input) <= limit {
		return input
	}
	keys := sortedKeys(input)
	out := make(map[string]any, limit)
	for _, key := range keys[:limit] {
		out[key] = input[key]
	}
	return out
}

// limitStrings 裁剪字符串列表。
func limitStrings(items []string, limit int) []string {
	if len(items) <= limit {
		return items
	}
	return append([]string(nil), items[:limit]...)
}

// mergeSummary 把子摘要合并回父摘要。
func mergeSummary(dst, src map[string]any) {
	for key, value := range src {
		dst[key] = value
	}
}

// typeName 返回值的 Go 类型名，用于摘要和调试。
func typeName(v any) string {
	if v == nil {
		return "nil"
	}
	return fmt.Sprintf("%T", v)
}
