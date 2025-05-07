package core

import "github.com/cassius0924/jtgo/util"

func registerFunction(fnMap map[string]any, fn any) {
	fnName := util.GetFunctionName(fn)
	fnMap[fnName] = fn
}

func RegisterFunction(templateID string, fn any) {
	if fn == nil || templateID == "" {
		return
	}
	templateIDToCustomFuncs.LoadOrStore(templateID, make(map[string]any))
	fnList, _ := templateIDToCustomFuncs.Load(templateID)
	registerFunction(fnList.(map[string]any), fn)
	templateIDToCustomFuncs.Store(templateID, fnList)
}

func RegisterFunctionWithAlias(fn any, fnName string) {
	builtInFns[fnName] = fn
}

func RegisterBuiltInFunction(fn any) {
	registerFunction(builtInFns, fn)
}

func RegisterBuiltInFunctionWithAlias(fn any, fnName string) {
	builtInFns[fnName] = fn
}
