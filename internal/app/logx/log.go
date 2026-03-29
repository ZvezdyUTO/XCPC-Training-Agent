package logx

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"
)

// Error 以统一格式输出内部错误日志。
// 它只负责把关键排障信息打到标准输出，不负责持久化或日志分级策略。
func Error(op string, err error, fields map[string]any) {
	log.Printf(
		"[error] time=%s op=%s err=%s fields=%s",
		time.Now().Format(time.RFC3339),
		strings.TrimSpace(op),
		errorText(err),
		encodeFields(fields),
	)
}

// Info 以统一格式输出普通运行日志。
// 当前只用于保留少量高价值运行信息，避免日志系统过度膨胀。
func Info(op string, fields map[string]any) {
	log.Printf(
		"[info] time=%s op=%s fields=%s",
		time.Now().Format(time.RFC3339),
		strings.TrimSpace(op),
		encodeFields(fields),
	)
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func encodeFields(fields map[string]any) string {
	if len(fields) == 0 {
		return "{}"
	}

	keys := make([]string, 0, len(fields))
	for key := range fields {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	ordered := make(map[string]any, len(fields))
	for _, key := range keys {
		ordered[key] = fields[key]
	}

	body, err := json.Marshal(ordered)
	if err != nil {
		return fmt.Sprintf(`{"marshal_error":%q}`, err.Error())
	}
	return string(body)
}
