package util

import (
	"github.com/bytedance/sonic"
)

// SonicToString 使用 sonic 序列化数据
func SonicToString(data any) string {
	b, err := sonic.Marshal(data)
	if err != nil {
		return ""
	}
	return string(b)
}