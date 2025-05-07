package exprs

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"strings"

	"github.com/cassius0924/jtgo/werror"
	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/spf13/cast"
)

type ExprHandler struct {
	templateID   string
	template     string
	compiledExps map[string]*vm.Program
	env          map[string]any
	fns          map[string]any
}

// NewExprHandler 创建一个新的ExprHandler实例
func NewExprHandler(templateID, template string, fns map[string]any) *ExprHandler {
	return &ExprHandler{
		templateID: templateID,
		template:   template,
		fns:        fns,
	}
}

// SetCompiledExps 设置编译过的表达式
func (h *ExprHandler) SetCompiledExps(compiledExps map[string]*vm.Program) {
	h.compiledExps = compiledExps
}

// SetDataset 设置数据集
func (h *ExprHandler) SetDataset(dataset map[string]any) {
	// 把数据集和函数f合并到env中
	maps.Copy(dataset, h.fns)
	h.env = dataset
}

// Run 解析表达式
func (h *ExprHandler) Run(ctx context.Context, program *vm.Program) (any, error) {
	h.env["ctx"] = ctx
	return expr.Run(program, h.env)
}

// EvaluateExpression 计算表达式
func (h *ExprHandler) EvaluateExpression(ctx context.Context, expression string) (any, error) {
	program, ok := h.compiledExps[expression]
	if !ok {
		slog.ErrorContext(ctx, "[exprs.EvaluateExpression] compiledExps not found, please check code", "expression", expression)
		return nil, werror.ErrCompiledExpressionNotFound
	}
	result, err := h.Run(ctx, program)
	if err != nil {
		// 这里用Warn，因为表达式可以不进行空指针判断，若出现空指针，这里会有err，但符合预期
		slog.WarnContext(ctx, "[exprs.EvaluateExpression] expr.Run err", "expression", expression, "error", err)
		return nil, err
	}
	slog.InfoContext(ctx, fmt.Sprintf("[exprs.EvaluateExpression] expr.Run success,\nexpression = %s,\nresult = %v", expression, result))
	return result, nil
}

// EvaluateExpressionsInText 计算文案中的表达式并且拼接
func (h *ExprHandler) EvaluateExpressionsInText(ctx context.Context, input string) any {
	text := strings.TrimSpace(input)
	exps := ExtractAllExpressions(text) // 找到text中所有${variable}中的variable
	// 除表达式外，还有其他字符的场景，一定是字符串
	for _, exp := range exps {
		result, err := h.EvaluateExpression(ctx, exp)
		if err != nil {
			input = strings.ReplaceAll(input, fmt.Sprintf(expressionFormat, exp), "")
			continue
		}

		// 如果是只有一个表达式，且无其他字符的场景，直接替换，例如 "assembleGameModuleActivity(PromoteGame)" 计算函数值然后返回一个结构体
		if len(exps) == 1 && fmt.Sprintf(expressionFormat, exps[0]) == input {
			return result
		}

		resultStr := cast.ToString(result)
		// 将${Expr}替换为dataset中的值
		input = strings.ReplaceAll(input, fmt.Sprintf(expressionFormat, exp), resultStr)
	}
	if input == "" {
		slog.InfoContext(ctx, "[exprs.EvaluateExpressionsInText] input is empty after replace", "input", input)
		return nil
	}
	return input
}

// EvaluateExpressionToBool 计算表达式并且转换为bool
func (h *ExprHandler) EvaluateExpressionToBool(ctx context.Context, expression string) (bool, error) {
	result, err := h.EvaluateExpression(ctx, expression)
	if err != nil {
		return false, err
	}
	// 把result转换为bool
	isBoolResult, castErr := cast.ToBoolE(result)
	if castErr != nil {
		slog.WarnContext(ctx, "[exprs.EvaluateExpressionToBool] exprResult not bool, please check expression", "expression", expression, "result", result)
		return false, werror.ErrExpressionResultNotBool
	}
	return isBoolResult, nil
}

// CompileAsBool 编译表达式并且要求最终值为bool
func (h *ExprHandler) CompileAsBool(ctx context.Context, expression string, opts ...expr.Option) (*vm.Program, error) {
	// 加上expr.AsBool()，确保表达式最终值为bool
	opts = append(opts, expr.Env(h.fns), expr.AsBool(), expr.AllowUndefinedVariables(), expr.WithContext("ctx"))
	return expr.Compile(expression, opts...)
}

// Compile 编译表达式，无最终值要求
func (h *ExprHandler) Compile(ctx context.Context, expression string, opts ...expr.Option) (*vm.Program, error) {
	opts = append(opts, expr.Env(h.fns), expr.AllowUndefinedVariables(), expr.WithContext("ctx"))
	return expr.Compile(expression, opts...)
}

// CompileStringExpressions 编译字符串中的表达式
func (h *ExprHandler) CompileStringExpressions(ctx context.Context, expression string, compiledExps map[string]*vm.Program) {
	exps := ExtractAllExpressions(expression)
	// 编译每一个表达式
	for _, exp := range exps {
		if compiledExps[exp] != nil { // 已经编译过跳过
			continue
		}
		program, err := h.Compile(ctx, exp)
		if err != nil {
			slog.ErrorContext(ctx, "[exprs.compileStringExpressions] expr.Compile err", "expression", exp, "error", err)
		} else {
			compiledExps[exp] = program
		}
	}
}
