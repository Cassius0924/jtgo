package engine

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/cassius0924/jtgo/util"
	"github.com/tidwall/gjson"
)

// replaceExpression 递归替换 object 中所有含有${Expression}的值
func (e *JTEngine) replaceExpression(input *gjson.Result) any {
	if input == nil || !input.Exists() {
		return nil
	}
	// JSON 共有 6 种类型：input, array, bool, number, null, string
	switch {
	case input.IsObject(): // input
		var (
			result          = make(map[string]any)
			resultForReturn any
			isExpression    bool
		)

		input.ForEach(func(field, node gjson.Result) bool {
			fieldName := strings.TrimSpace(field.String())

			switch DetectKeyword(fieldName) {
			case KeywordReturn:
				// 遇到 RETURN 关键词，直接返回
				resultForReturn = e.returnResult(&node)
				return false
			case KeywordDo:
				// 是 DO 关键词，需要执行操作
				e.doOperations(&node)
				return true
			case KeywordVar:
				// 是 VAR 关键词，需要进行变量赋值
				e.varAssignment(&node)
				return true
			default:
				// 其他情况，继续处理
			}
			_, isExpression = extractExpression(fieldName)
			if !isExpression {
				result[fieldName] = e.replaceExpression(&node)
				return true
			}

			return true
		})

		if resultForReturn != nil {
			return resultForReturn
		}
		return result
	case input.IsArray(): // array
		var result []any
		for _, value := range input.Array() {
			result = append(result, e.replaceExpression(&value))
		}
		return result
	case input.IsBool(): // bool
		return input.Bool()
	case input.Type == gjson.Number: // number
		return input.Num
	case input.Type == gjson.Null: // null
		return nil
	default: // string
		return e.evaluateExpressionsInText(input.String()) // 使用数据集中的值替换text中的${Path.Var}
	}
}

// doOperations 处理DO操作
func (e *JTEngine) doOperations(object *gjson.Result) {
	if !object.Exists() {
		return
	}
	switch {
	case object.IsArray(): // 是数组，则遍历处理
		object.ForEach(func(_, value gjson.Result) bool {
			if value.Type != gjson.String {
				return true
			}
			slog.InfoContext(e.ctx, fmt.Sprintf("[JsonTemplateEngine.doOperations](trace) do operation in array,\noperation = %s", value.String()))
			e.evaluateExpressionsInText(value.String())
			return true
		})
	case object.IsObject():
		object.ForEach(func(key, value gjson.Result) bool {
			// key是表达式，value是操作
			keyName := strings.TrimSpace(key.String())
			expression, isExpression := extractExpression(keyName)
			if !isExpression { // 不是表达式，则跳过
				slog.WarnContext(e.ctx, "[JsonTemplateEngine.doOperations] key is not an expression, please check whether the key is an expression!", "key", keyName)
				return true
			}
			isBoolResult, err := e.evaluateExpressionToBool(expression)
			if err != nil {
				e.err = err
				return true
			}
			if isBoolResult {
				slog.InfoContext(e.ctx, "[JsonTemplateEngine.doOperations] matched expression, nested do operation", "matchedExpression", expression)
				e.doOperations(&value)
				return false
			}
			return true
		})
	case object.Type == gjson.String:
		slog.InfoContext(e.ctx, fmt.Sprintf("[JsonTemplateEngine.doOperations](trace) do operation.\noperation = %s", object.String()))
		e.evaluateExpressionsInText(object.String())
	default:
		return
	}
}

func (e *JTEngine) returnResult(object *gjson.Result) any {
	result := e.replaceExpression(object)
	slog.InfoContext(e.ctx, fmt.Sprintf("[JsonTemplateEngine.returnResult](trace) return result,\nresult = %s", util.GenerateStructFormatedString(result)))
	return result
}

// varAssignment 处理VAR变量赋值
func (e *JTEngine) varAssignment(object *gjson.Result) {
	if !object.Exists() {
		return
	}

	switch {
	// 只有Object类型才能进行变量赋值，其他类型均属于语法错误
	case object.IsObject():
		object.ForEach(func(key, value gjson.Result) bool {
			// key是变量名或表达式，value是变量值
			keyName := strings.TrimSpace(key.String())
			expression, isExpression := extractExpression(keyName)
			// 不是表达式，是变量名，则创建变量
			if !isExpression {
				if keyName == "" {
					return true
				}
				if e.dataset == nil {
					e.dataset = make(map[string]any)
				}
				e.localVariables[keyName] = e.dataset[keyName]
				e.dataset[keyName] = e.replaceExpression(&value)
				slog.InfoContext(e.ctx, fmt.Sprintf("[JsonTemplateEngine.varAssignment](trace) create variable,\nkey = %s,\nvalue = %s,\nexpr = %s", keyName, util.GenerateStructFormatedString(e.dataset[keyName]), value.String()))
				return true
			}
			// 如果表达式对应的不是一个 Object，则属于语法错误，跳过
			if !value.IsObject() {
				slog.ErrorContext(e.ctx, "[JsonTemplateEngine.varAssignment] expression value is not an object, please check whether the expression value is an object!", "key", keyName, "value", value.String())
				return true
			}

			// 是表达式，计算表达式
			isBoolResult, err := e.evaluateExpressionToBool(expression)
			if err != nil {
				e.err = err
				return true
			}
			if isBoolResult {
				slog.InfoContext(e.ctx, "[JsonTemplateEngine.varAssignment] matched expression, nested var assignment", "matchedExpression", expression)
				e.varAssignment(&value)
				return true
			}
			return true
		})
	default:
		slog.ErrorContext(e.ctx, "[JsonTemplateEngine.varAssignment] do value is not an object, please check whether the do value is an object!", "doValue", object.String())
	}
}
