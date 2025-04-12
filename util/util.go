package util

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

func init() {
	// 用于美化结构体的输出
	spew.Config.Indent = "    "
}

// GenerateStructFormatedString 生成结构体格式化字符串
func GenerateStructFormatedString(raw any) string {
	return fmt.Sprint(spew.Sdump(raw))
}
