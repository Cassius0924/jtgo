package engine

import (
	"log/slog"

	"github.com/cassius0924/jtgo/werror"
	"github.com/tidwall/gjson"
)

func (e *JTEngine) preCompileExpressions(configJSON string) error {
	configResult := gjson.Parse(configJSON)
	if !configResult.Exists() {
		slog.ErrorContext(e.ctx, "[JsonTemplateEngine.preCompileExpressions] pre compile configJSON error, please check whether the config JSON is valid!")
		return werror.ErrPreCompileFail
	}

	configResult.ForEach(func(field, variable gjson.Result) bool {
		e.recursivePreCompile(configResult)
		return true
	})

	return nil
}

func (e *JTEngine) recursivePreCompile(configResult gjson.Result) {
	configResult.ForEach(func(field, object gjson.Result) bool {
		if !object.Exists() {
			return true
		}

		fieldName := normalizeFieldName(field.String())
		// 去掉${}符号，提取表达式
		expression, isExpression := extractExpression(fieldName)

		// 这里预编译，是表达式并且未被编译过
		if isExpression && e.compiledExps[expression] == nil && !IsKeywordDefault(expression) {
			// 编译表达式，然后缓存进compiledExps
			program, err := e.exprCompileAsBool(expression)
			if err != nil { // 说明expression不是以bool为最终值的表达式
				slog.ErrorContext(e.ctx, "[JsonTemplateEngine.recursivePreCompile] expr.Compile err", "field", fieldName, "expression", expression, "error", err)
			} else {
				e.compiledExps[expression] = program
			}
		}

		switch {
		case object.IsObject():
			e.recursivePreCompile(object)
			return true
		case object.IsArray():
			object.ForEach(func(_, value gjson.Result) bool {
				e.recursivePreCompile(value)
				return true
			})
		case object.IsBool() || object.Type == gjson.Number || object.Type == gjson.Null:
			return true
		default: // 字符串
			// 这里编译value
			exps := extractAllExpression(object.String())
			// 编译每一个表达式
			for _, exp := range exps {
				if e.compiledExps[exp] != nil { // 已经编译过跳过
					continue
				}
				program, err := e.exprCompile(exp)
				if err != nil {
					slog.ErrorContext(e.ctx, "[JsonTemplateEngine.recursivePreCompile] expr.Compile err", "field", fieldName, "expression", exp, "error", err)
				} else {
					e.compiledExps[exp] = program
				}
			}
			return true
		}
		return true
	})
}
