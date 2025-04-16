package engine

import (
	"fmt"
	"log/slog"

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
			fieldName := normalizeFieldName(field.String())

			switch keyword, _ := detectKeyword(fieldName); keyword {
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
func (e *JTEngine) doOperations(node *gjson.Result) {
	if !node.Exists() {
		return
	}
	switch {
	case node.IsArray(): // 是数组，则遍历处理
		node.ForEach(func(_, value gjson.Result) bool {
			if value.Type != gjson.String {
				return true
			}
			slog.InfoContext(e.ctx, fmt.Sprintf("[JsonTemplateEngine.doOperations](trace) do operation in array,\noperation = %s", value.String()))
			e.evaluateExpressionsInText(value.String())
			return true
		})
	case node.IsObject():
		node.ForEach(func(key, value gjson.Result) bool {
			// key是表达式，value是操作
			keyName := normalizeFieldName(key.String())
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
	case node.Type == gjson.String:
		slog.InfoContext(e.ctx, fmt.Sprintf("[JsonTemplateEngine.doOperations](trace) do operation.\noperation = %s", node.String()))
		e.evaluateExpressionsInText(node.String())
	default:
		return
	}
}

func (e *JTEngine) returnResult(node *gjson.Result) any {
	result := e.replaceExpression(node)
	slog.InfoContext(e.ctx, fmt.Sprintf("[JsonTemplateEngine.returnResult](trace) return result,\nresult = %s", util.GenerateStructFormatedString(result)))
	return result
}

// varAssignment 处理VAR变量赋值
func (e *JTEngine) varAssignment(node *gjson.Result) {
	if !node.Exists() {
		return
	}

	switch {
	// 只有Object类型才能进行变量赋值，其他类型均属于语法错误
	case node.IsObject():
		node.ForEach(func(key, value gjson.Result) bool {
			// key是变量名或表达式，value是变量值
			keyName := normalizeFieldName(key.String())
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
		slog.ErrorContext(e.ctx, "[JsonTemplateEngine.varAssignment] do value is not an object, please check whether the do value is an object!", "doValue", node.String())
	}
}

// judgeConditionalIf 处理if条件判断，返回是否命中该条件
func (e *JTEngine) judgeConditionalIf(node *gjson.Result, expression string, frame *ParseFrame) bool {
	// 表达式计算为bool值
	matched, err := e.evaluateExpressionToBool(expression)
	if err != nil {
		e.err = err
	}

	// 表达式为true，替换值，并剪枝结束循环
	if matched {
		frame.conditionalCtx.resultValue = node
		frame.conditionalCtx.isMatched = true
	}

	// 自增条件组序号
	frame.conditionalCtx.groupNum.Inc()
	slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.judgeConditionalIf](trace) condition evaluate result,\nkey = %s,\nvalue = %s,\nexpr = %s", frame.fieldName, node.String(), expression))
	return matched
}

// judgeConditionalElif 处理elif条件判断，返回是否命中该条件
func (e *JTEngine) judgeConditionalElif(node *gjson.Result, expression string, frame *ParseFrame) bool {
	return e.judgeConditionalIf(node, expression, frame)
}

// judgeConditionalElse 处理else条件判断
func (e *JTEngine) judgeConditionalElse(node *gjson.Result, frame *ParseFrame) bool {
	// 判断当前 else 是否是孤儿else，即当前 else 是否属于某一个 if
	// TODO: 封装成函数
	if frame.conditionalCtx.groupNum.Value() == 0 {
		// 是孤儿 else 则视为 false
		slog.WarnContext(e.ctx, "[JSONTemplateEngine.judgeConditionalElse] this else is orphan, please check whether the else belongs to an if!", "key", frame.fieldName)
		return false
	}

	frame.conditionalCtx.resultValue = node
	frame.conditionalCtx.isMatched = true

	slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.judgeConditionalIf](trace) condition evaluate result,\nkey = %s,\nvalue = %s", frame.fieldName, node.String()))
	return true
}

// executeLoop 处理for循环
func (e *JTEngine) executeLoop(node *gjson.Result, extra string, frame *ParseFrame) {

}

// continueLoop 处理循环中的CONTINUE
func (e *JTEngine) continueLoop(node *gjson.Result, extra string, frame *ParseFrame) {

}
