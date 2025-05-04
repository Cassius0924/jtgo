package util

import (
	"errors"

	"github.com/bytedance/sonic"
	"github.com/bytedance/sonic/decoder"
)

// SonicToString 使用 sonic 序列化数据
func SonicToString(data any) string {
	b, err := sonic.Marshal(data)
	if err != nil {
		return ""
	}
	return string(b)
}

// ValidateJSON 检查JSON是否合法
func ValidateJSON(template string) error {
	var data any
	err := sonic.UnmarshalString(template, &data)
	if serr, ok := err.(decoder.SyntaxError); ok {
		return errors.New(serr.Description())
	} else if merr, ok := err.(*decoder.MismatchTypeError); ok {
		return errors.New(merr.Description())
	}
	return err
}
