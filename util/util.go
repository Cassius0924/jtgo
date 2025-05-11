package util

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

func init() {
	// 用于美化结构体的输出
	spew.Config.Indent = "    "
}

// GenerateStructFormattedString 生成结构体格式化字符串
func GenerateStructFormattedString(raw any) string {
	return fmt.Sprint(spew.Sdump(raw))
}

// AppendOrSet 根据容器类型追加或设置值
func AppendOrSet(container any, key string, value any) {
	if result, ok := container.(*[]any); ok {
		*result = append(*result, value)
	} else {
		container.(map[string]any)[key] = value
	}
}
