package engine

import "github.com/cassius0924/jtgo/util"

func registerFunction(fnMap map[string]any, fn any) {
	// 反射取出函数名，正常来说RegisterFunction只会在服务启动的时候运行一次，不用担心性能问题
	fnName := util.GetFunctionName(fn)
	fnMap[fnName] = fn
}

// RegisterFunction 注册业务自定义函数
func RegisterFunction(templateID string, fn any) {
	if fn == nil {
		return
	}
	templateIDToCustomFuncs.LoadOrStore(templateID, make(map[string]any))
	fnList, _ := templateIDToCustomFuncs.Load(templateID)
	registerFunction(fnList.(map[string]any), fn)
	templateIDToCustomFuncs.Store(templateID, fnList)
}

func RegisterFunctionWithAlias(fn any, fnName string) {
	builtInFuncCollection[fnName] = fn
}

// RegisterBuiltInFunction 注册引擎内置函数，只用于注册通用函数，如toInt等
func RegisterBuiltInFunction(fn any) {
	registerFunction(builtInFuncCollection, fn)
}

func RegisterBuiltInFunctionWithAlias(fn any, fnName string) {
	builtInFuncCollection[fnName] = fn
}
