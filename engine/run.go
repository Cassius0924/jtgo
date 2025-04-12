package engine

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cassius0924/jtgo/util"
	"github.com/cassius0924/jtgo/werror"
	"github.com/tidwall/gjson"
)

// Run 运行引擎，并且清空引擎状态
func (e *JTEngine) Run() error {
	slog.InfoContext(e.ctx, "[JSONTemplateEngine.Run](trace) Run function start")
	defer func() {
		// 清空调用链
		e.clear()
		slog.InfoContext(e.ctx, "[JSONTemplateEngine.Run](trace) Run function end")
	}()
	return e.keepStatusRun()
}

// keepStatusRun 保持状态运行
func (e *JTEngine) keepStatusRun() error {
	if err := e.checkBeforeRun(); err != nil {
		return err
	}

	// 获取入口的模板
	entryTemplateNode := gjson.Get(e.template, e.entry)
	if !entryTemplateNode.Exists() {
		slog.ErrorContext(e.ctx, "[JSONTemplateEngine.Run] entry not exists, please check whether entry name exists in the config JSON!", "entry", e.entry, "template", e.template)
		return werror.ErrEntryNotFound
	}

	target := make(map[string]any)
	result, hasResult := e.recursiveParse(entryTemplateNode, target, e.entry)

	var finalTarget any
	if hasResult { // 如果顶层就是表达式，这里会有值，其他情况 result 为空
		finalTarget = result
	} else {
		finalTarget = target
	}

	// 将config赋值给dest
	err := sonic.UnmarshalString(util.SonicToString(finalTarget), e.target)
	if err != nil {
		slog.ErrorContext(e.ctx, "[JSONTemplateEngine.Run] UnmarshalFromString config error, please check whether the template JSON field name matches the target structure field name!", "finalTarget", util.GenerateStructFormatedString(finalTarget), "error", err)
		return werror.Join(werror.ErrParseToTargetFailed, err)
	}

	// 恢复局部变量
	for k, v := range e.localVariables {
		e.dataset[k] = v
		slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.Run] restore variable,\nkey = %s,\nvalue = %v", k, v))
	}
	e.localVariables = make(map[string]any)

	return e.err
}

// recursiveParse 输入 templateNode 根据 e.dataset 最后解析到 target 中
func (e *JTEngine) recursiveParse(templateNode gjson.Result, target map[string]any, keyName string) (any, bool) {
	var (
		res, defaultRes *gjson.Result
		isMatch         bool
		matchedExpr     string
	)

	templateNode.ForEach(func(field, node gjson.Result) bool {
		if !node.Exists() {
			return true
		}

		// 去掉头尾空格
		fieldName := strings.TrimSpace(field.String())

		// fieldName 有五种情况：1. DEFAULT 2. VAR 3. DO 4. 普通字符串 5. 表达式
		switch DetectKeyword(fieldName) {
		case KeywordDo:
			// 是 DO 关键词，需要执行操作
			e.doOperations(&node)
			return true
		case KeywordVar:
			// 是 VAR 关键词，需要进行变量赋值
			e.varAssignment(&node)
			return true
		case KeywordDefault:
			// 是默认值DEFAULT
			defaultRes = &node // 记录下默认值
			return true
		default:
			// 其他情况，继续处理
		}

		expression, isExpression := extractExpression(fieldName)
		// 是普通字符串
		if !isExpression { // 不是表达式
			switch {
			case node.IsObject():
				// 是Object，需要继续递归解析
				target[fieldName] = map[string]any{}
				subTarget := target[fieldName].(map[string]any)
				// 递归解析
				result, hasResult := e.recursiveParse(node, subTarget, fieldName)
				if hasResult {
					target[fieldName] = result
				}
				return true
			default:
				// 其他类型直接复制
				target[fieldName] = e.replaceExpression(&node)
				slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.recursiveParse](trace) using default value,\nkey = %s,\nvalue = %s", fieldName, node.String()))
				return true
			}
		}

		isBoolResult, err := e.evaluateExpressionToBool(expression)
		if err != nil {
			e.err = err
			return true
		}

		if isBoolResult { // 表达式为true，替换值，并剪枝结束循环
			res = &node
			isMatch = true
			matchedExpr = fmt.Sprintf(expressionFormat, expression)
			return false
		}
		return true
	})

	// 遍历完毕，解析匹配到的值
	if isMatch {
		e.matchedExprTraces[keyName] = matchedExpr
		slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.recursiveParse](trace) using matched value,\nkey = %s,\nvalue = %s,\nexpr = %s", keyName, res.String(), matchedExpr))
		return e.replaceExpression(res), true
	} else if defaultRes != nil { // 没有匹配到值，使用DEFAULT默认值兜底
		e.matchedExprTraces[keyName] = `${DEFAULT}`
		slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.recursiveParse](trace) using default value,\nkey = %s,\nvalue = %s,\nexpr = ${DEFAULT},", keyName, defaultRes.String()))
		return e.replaceExpression(defaultRes), true
	}
	return nil, false
}

// checkBeforeRun 检查是否有必要的参数
func (e *JTEngine) checkBeforeRun() error {
	if e.template == "" {
		slog.ErrorContext(e.ctx, "[JSONTemplateEngine.check] configJSON is empty")
		return werror.ErrTemplateIsEmpty
	}
	if e.target == nil {
		slog.ErrorContext(e.ctx, "[JSONTemplateEngine.check] target is empty")
		return werror.ErrTargetIsNil
	}
	return nil
}
