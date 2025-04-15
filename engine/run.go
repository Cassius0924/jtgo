package engine

import (
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/util"
	"github.com/cassius0924/jtgo/werror"
	"github.com/liyue201/gostl/ds/deque"
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

	// 循环解析方法
	finalTarget := e.interativeParse(entryTemplateNode, e.entry)

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

type ParseFrame struct {
	node         gjson.Result
	fieldName    string
	target       any
	curSubNode   *deque.DequeIterator[*ds.Pair[gjson.Result, gjson.Result]]
	isMatchedRes bool
	result       *gjson.Result
}

// interativeParse 迭代解析方法
func (e *JTEngine) interativeParse(templateNode gjson.Result, entry string) any {
	var (
		stack  = ds.NewStack[*ParseFrame]()
		result any
	)
	// 初始化栈
	stack.Push(&ParseFrame{
		node:         templateNode,
		fieldName:    entry,
		target:       make(map[string]any),
		curSubNode:   flattenNode(templateNode).First(),
		isMatchedRes: false,
	})

	// 迭代解析
	for stack.Size() > 0 {
		var (
			frame       = stack.Top() // 获取栈顶帧
			field, node gjson.Result
		)

		// 如果子节点迭代器无效，或匹配到了值，说明当前帧的子节点已经遍历完毕，弹出栈顶元素
		if !frame.curSubNode.IsValid() || frame.isMatchedRes {
			stack.Pop()

			if frame.result != nil {
				e.setExpressionResult(frame, entry)
				slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.interativeParse](trace) using value,\nkey = %s,\nvalue = %s", frame.fieldName, frame.result.String()))
			}

			// 如果栈为空，说明所有帧都已经遍历完毕，返回结果
			if stack.Size() == 0 {
				result = frame.target
				break
			}
			continue
		}

		// 取出当前值后，继续遍历，迭代器指向下一个元素
		pair := frame.curSubNode.Value()
		field, node = pair.First, pair.Second
		frame.curSubNode.Next()

		// TODO: 这里是否需要？
		// if !node.Exists() {
		// 	continue
		// }

		fieldName := normalizeFieldName(field.String())

		switch keyword, _ := detectKeyword(fieldName); keyword {
		case KeywordDo:
			e.doOperations(&node)
			continue
		case KeywordVar:
			e.varAssignment(&node)
			continue
		case KeywordDefault:
			if !frame.isMatchedRes {
				frame.result = &node // 记录下默认值
			}
			continue
		case KeywordIf:
			// e.controlFlowIf(&node, extra)
			continue
		case KeywordElse:
			// e.controlFlowElse(&node, extra)
		case KeywordFor:
			// e.loop(&node, extra)
			continue

		default:
			// 其他情况，继续处理
		}

		expression, isExpression := extractExpression(fieldName)
		if !isExpression { // 不是表达式
			switch {
			case node.IsObject():
				// 是Object，需要继续解析
				var subTarget map[string]any
				if frame.fieldName == entry {
					// 如果是顶层节点，直接赋值给target
					subTarget = frame.target.(map[string]any)
				} else {
					// 不是顶层节点，赋值给当前帧的target
					frame.target.(map[string]any)[frame.fieldName] = make(map[string]any)
					subTarget = frame.target.(map[string]any)[frame.fieldName].(map[string]any)
				}

				stack.Push(&ParseFrame{
					node:         node,
					fieldName:    fieldName,
					target:       subTarget,
					curSubNode:   flattenNode(node).First(),
					isMatchedRes: false,
				})
				continue
			default:
				// 其他类型直接赋值
				frame.target.(map[string]any)[frame.fieldName] = e.replaceExpression(&node)
				slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.interativeParse](trace) using default value,\nkey = %s,\nvalue = %s", frame.fieldName, node.String()))
				continue
			}
		}

		// 是表达式
		isTrue, err := e.evaluateExpressionToBool(expression)
		if err != nil {
			e.err = err
			continue
		}

		if isTrue { // 表达式为true，替换值，并剪枝结束循环
			frame.result = &node
			frame.isMatchedRes = true
			slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.recursiveParse](trace) using matched value,\nkey = %s,\nvalue = %s,\nexpr = %s", frame.fieldName, node.String(), expression))
		}
	}

	return result
}

func (e *JTEngine) setExpressionResult(frame *ParseFrame, entry string) {
	if frame.fieldName == entry {
		frame.target = e.replaceExpression(frame.result)
	} else {
		frame.target.(map[string]any)[frame.fieldName] = e.replaceExpression(frame.result)
	}
}

// recursiveParse 输入 templateNode 根据 e.dataset 最后解析到 target 中
// func (e *JTEngine) recursiveParse(templateNode gjson.Result, target map[string]any, keyName string) (any, bool) {
// 	var (
// 		res, defaultRes *gjson.Result
// 		isMatch         bool
// 		matchedExpr     string
// 	)

// 	templateNode.ForEach(func(field, node gjson.Result) bool {
// 		if !node.Exists() {
// 			return true
// 		}

// 		fieldName := normalizeFieldName(field.String())

// 		// fieldName 有五种情况：1. DEFAULT 2. VAR 3. DO 4. 普通字符串 5. 表达式
// 		switch detectKeyword(fieldName) {
// 		case KeywordDo:
// 			// 是 DO 关键词，需要执行操作
// 			e.doOperations(&node)
// 			return true
// 		case KeywordVar:
// 			// 是 VAR 关键词，需要进行变量赋值
// 			e.varAssignment(&node)
// 			return true
// 		case KeywordDefault:
// 			// 是默认值DEFAULT
// 			defaultRes = &node // 记录下默认值
// 			return true
// 		default:
// 			// 其他情况，继续处理
// 		}

// 		expression, isExpression := extractExpression(fieldName)
// 		// 是普通字符串
// 		if !isExpression { // 不是表达式
// 			switch {
// 			case node.IsObject():
// 				// 是Object，需要继续递归解析
// 				target[fieldName] = make(map[string]any)
// 				subTarget := target[fieldName].(map[string]any)
// 				// 递归解析
// 				result, hasResult := e.recursiveParse(node, subTarget, fieldName)
// 				if hasResult {
// 					target[fieldName] = result
// 				}
// 				return true
// 			default:
// 				// 其他类型直接赋值
// 				target[fieldName] = e.replaceExpression(&node)
// 				slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.recursiveParse](trace) using default value,\nkey = %s,\nvalue = %s", fieldName, node.String()))
// 				return true
// 			}
// 		}

// 		isBoolResult, err := e.evaluateExpressionToBool(expression)
// 		if err != nil {
// 			e.err = err
// 			return true
// 		}

// 		if isBoolResult { // 表达式为true，替换值，并剪枝结束循环
// 			res = &node
// 			isMatch = true
// 			return false
// 		}
// 		return true
// 	})

// 	// 遍历完毕，解析匹配到的值
// 	if isMatch {
// 		slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.recursiveParse](trace) using matched value,\nkey = %s,\nvalue = %s,\nexpr = %s", keyName, res.String(), matchedExpr))
// 		return e.replaceExpression(res), true
// 	} else if defaultRes != nil { // 没有匹配到值，使用DEFAULT默认值兜底
// 		slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.recursiveParse](trace) using default value,\nkey = %s,\nvalue = %s,\nexpr = ${DEFAULT},", keyName, defaultRes.String()))
// 		return e.replaceExpression(defaultRes), true
// 	}
// 	return nil, false
// }

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

func flattenNode(node gjson.Result) *deque.Deque[*ds.Pair[gjson.Result, gjson.Result]] {
	var result = ds.NewDeque[*ds.Pair[gjson.Result, gjson.Result]]()
	node.ForEach(func(k, v gjson.Result) bool {
		result.PushBack(ds.MakePair(k, v))
		return true
	})
	return result
}
