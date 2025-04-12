package util

import (
	"reflect"
	"runtime"
	"strings"
)

func GetFunctionName(f any) string {
	var funcName string

	fullName := runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
	parts := strings.Split(fullName, "/")
	if len(parts) > 0 {
		funcName = parts[len(parts)-1]
	}
	parts = strings.Split(funcName, ".")
	if len(parts) > 0 {
		funcName = parts[len(parts)-1]
	}
	return funcName
}
