package engine

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/samber/lo"
	"github.com/spf13/cast"
)

const (
	// ${Expression}的前缀
	expressionFlagPrefix = "${"
	// ${Expression}的后缀
	expressionFlagSuffix = "}"

	expressionFormat = expressionFlagPrefix + "%s" + expressionFlagSuffix
	expressionRegexp = `\$\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`
)

var (
	expressionRe = regexp.MustCompile(expressionRegexp) // 匹配${Expression}的正则表达式
)

// evaluateExpressionsInText 计算文案中的表达式并且拼接，这个函数不一定返回 string，可能是任意类型
func (e *JTEngine) evaluateExpressionsInText(input string) any {
	text := strings.TrimSpace(input)
	exps := extractAllExpression(text) // 找到text中所有${Path.Var}中的Path.Var
	// 除表达式外，还有其他字符的场景，一定是字符串，例如 "活动名为${Activity.Title}"
	for _, exp := range exps {
		result, err := e.evaluateExpression(exp)
		if err != nil {
			input = strings.ReplaceAll(input, fmt.Sprintf(expressionFormat, exp), "")
			continue
		}

		// 如果是只有一个表达式，且无其他字符的场景，直接替换，例如 "assembleGameModuleActivity(PromoteGame)" 计算函数值然后返回一个结构体
		if len(exps) == 1 && fmt.Sprintf(expressionFormat, exps[0]) == strings.ReplaceAll(input, JSONTemplateEngineDatasetVariable, exprEnvVariable) {
			return result
		}

		resultStr := cast.ToString(result)
		// 将${Expr}替换为dataset中的值
		input = strings.ReplaceAll(input, fmt.Sprintf(expressionFormat, exp), resultStr)
	}
	if input == "" {
		return nil
	}
	return input
}

// extractExpression 将input字符串的所有variable符号去除，只留下表达式，strings库替换比正则替换更快
func extractExpression(input string) (string, bool) {
	result := strings.ReplaceAll(input, expressionFlagPrefix, "")
	if result == input {
		return result, false
	}
	result = strings.ReplaceAll(result, expressionFlagSuffix, "")
	if result == input {
		return result, false
	}

	// 将大写的DATASET 替换成 $env
	result = strings.ReplaceAll(result, JSONTemplateEngineDatasetVariable, exprEnvVariable)
	return result, true
}

func extractAllExpression(input string) []string {
	matches := expressionRe.FindAllStringSubmatch(input, -1)
	var result []string
	for _, match := range matches {
		if len(match) > 1 {
			r := strings.ReplaceAll(match[1], JSONTemplateEngineDatasetVariable, exprEnvVariable)
			result = append(result, r)
		}
	}
	return result
}

// exprCompileAsBool 编译表达式并且要求最终值为bool
func (e *JTEngine) exprCompileAsBool(expression string, opts ...expr.Option) (*vm.Program, error) {
	// 加上expr.AsBool()，确保表达式最终值为bool
	opts = append(opts, expr.Env(e.fns), expr.AsBool(), expr.AllowUndefinedVariables(), expr.WithContext("ctx"))
	return expr.Compile(expression, opts...)
}

// exprCompile 编译表达式，无最终值要求
func (e *JTEngine) exprCompile(expression string, opts ...expr.Option) (*vm.Program, error) {
	opts = append(opts, expr.Env(e.fns), expr.AllowUndefinedVariables(), expr.WithContext("ctx"))
	return expr.Compile(expression, opts...)
}

// exprRun 解析表达式
func (e *JTEngine) exprRun(program *vm.Program) (any, error) {
	// 把e.dataset和自定义函数合并到env中
	env := lo.Assign(e.dataset, e.fns)
	subEngine := &JTEngine{
		ctx:               e.ctx,
		templateID:        e.templateID,
		template:          e.template,
		compiledExps:      e.compiledExps,
		dataset:           e.dataset, // 继承数据集
		fns:               e.fns,
		localVariables:    make(map[string]any),
		exprTraces:        e.exprTraces,
		matchedExprTraces: e.matchedExprTraces,
	}
	e.ctx = context.WithValue(e.ctx, JSONTemplateEngineCtxKey, subEngine)
	env["ctx"] = e.ctx
	return expr.Run(program, env)
}

// evaluateExpression 计算表达式
func (e *JTEngine) evaluateExpression(expression string) (any, error) {
	program, ok := e.compiledExps[expression]
	if !ok {
		slog.ErrorContext(e.ctx, "[JsonTemplateEngine.evaluateExpression] compiledExps not found, please check code", "expression", expression)
		return nil, errors.ErrUnsupported
	}
	result, err := e.exprRun(program)
	if err != nil {
		// 这里用Warn，因为表达式可以不进行空指针判断，若出现空指针，这里会有err，但符合预期
		slog.WarnContext(e.ctx, "[JsonTemplateEngine.evaluateExpression] expr.Run err", "expression", expression, "error", err)
		return nil, err
	}
	e.exprTraces[expression] = result
	slog.InfoContext(e.ctx, fmt.Sprintf("[JsonTemplateEngine.evaluateExpression] expr.Run success,\nexpression = %s,\nresult = %v", expression, result))
	return result, nil
}

// evaluateExpressionToBool 计算表达式并且转换为bool
func (e *JTEngine) evaluateExpressionToBool(expression string) (bool, error) {
	result, err := e.evaluateExpression(expression)
	if err != nil {
		return false, err
	}
	// 把result转换为bool
	isBoolResult, ok := result.(bool) // double check
	if !ok {
		if boolPtr, ok := result.(*bool); ok && boolPtr != nil {
			isBoolResult = *boolPtr
		} else {
			slog.WarnContext(e.ctx, "[JsonTemplateEngine.recursiveParse] exprResult not bool, please check expression", "expression", expression, "result", result)
			return false, errors.ErrUnsupported
		}
	}
	return isBoolResult, nil
}
