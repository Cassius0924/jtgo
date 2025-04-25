package util

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
)

// GetFunctionName 获取函数的名称，仅保留函数名部分
func GetFunctionName(f any) string {
	var funcName string
	// 检查 f 是否为函数类型
	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	fmt.Println("fullName:", fullName)
	lastSlash := strings.LastIndex(fullName, "/")
	if lastSlash != -1 {
		fullName = fullName[lastSlash+1:]
	}
	lastDot := strings.LastIndex(fullName, ".")
	if lastDot != -1 {
		funcName = fullName[lastDot+1:]
	} else {
		funcName = fullName
	}
	// 去掉尾部的 -fm
	funcName = strings.TrimSuffix(funcName, "-fm")
	return funcName
}
