package util

import (
	"github.com/bytedance/sonic"
)

func SonicToString(data any) string {
	b, err := sonic.Marshal(data)
	if err != nil {
		return ""
	}
	return string(b)
}