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
	finalTarget := e.interativeParse(entryTemplateNode)

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

// interativeParse 迭代解析方法
func (e *JTEngine) interativeParse(templateNode gjson.Result) any {
	var (
		frameStack = ds.NewStack[*ParseFrame]()
		result     any
	)
	// 初始化栈
	frameStack.Push(&ParseFrame{
		node:       templateNode,
		fieldName:  e.entry,
		target:     make(map[string]any),
		curSubNode: flattenNode(templateNode).First(),
		conditionalCtx: &ConditionalContext{
			isMatched: false,
		},
	})

	// 迭代解析
	for frameStack.Size() > 0 {
		var (
			frame                 = frameStack.Top() // 获取栈顶帧
			subNodeField, subNode gjson.Result
		)

		// 如果子节点迭代器无效，或匹配到了值，说明当前帧的子节点已经遍历完毕，弹出栈顶元素
		if !frame.curSubNode.IsValid() || frame.conditionalCtx.isMatched {
			frameStack.Pop()

			// TODO: 封装成 judgeConditionalIf 类似的函数，例如叫做 setConditionalResult
			// if frame.conditionalCtx.resultValue != nil {
			// 	e.setExpressionResult(frame)
			// 	slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.interativeParse](trace) using value,\nkey = %s,\nvalue = %s", frame.fieldName, frame.conditionalCtx.resultValue.String()))
			// }

			// 如果栈为空，说明所有帧都已经遍历完毕，返回结果
			if frameStack.Size() == 0 {
				result = frame.target
				break
			}
			continue
		}

		// 取出当前值后，继续遍历，迭代器指向下一个元素
		subNodePair := frame.curSubNode.Value()
		subNodeField, subNode = subNodePair.First, subNodePair.Second
		frame.curSubNode.Next()

		subNodeFieldName := normalizeFieldName(subNodeField.String())

		switch keyword, extra := detectKeyword(subNodeFieldName); keyword {
		case KeywordDo:
			e.doOperations(&subNode)
			continue
		case KeywordVar:
			e.varAssignment(&subNode)
			continue
		case KeywordDefault:
			if !frame.conditionalCtx.isMatched {
				frame.conditionalCtx.resultValue = &subNode // 记录下默认值
			}
			continue
		case KeywordIf:
			// if、elif、else和for都与当前帧相关，需要传入当前帧用以记录相关上下文
			// if、elif和else的extra都是条件表达式
			matched := e.judgeConditionalIf(&subNode, extra, frame)
			// 如果未命中当前条件，则需要继续尝试下一个条件
			if !matched {
				continue
			}
		case KeywordElif:
			matched := e.judgeConditionalElif(&subNode, extra, frame)
			if !matched {
				continue
			}
		case KeywordElse:
			e.judgeConditionalElse(&subNode, extra, frame)
		case KeywordFor:
			e.executeLoop(&subNode, extra, frame)
			continue
		case KeywordContinue:
			e.continueLoop(&subNode, extra, frame)
			continue
		default:
			// 其他情况，继续处理
		}

		switch {
		case subNode.IsObject():
			// 是Object，需要继续解析，将解析帧压入栈中
			var (
				target    any
				fieldName string
			)

			if e.isFrameAtTopLevel(frame) {
				// 如果是顶层节点，直接赋值给target，fieldName为正常的子节点fieldName
				target = frame.target
				fieldName = subNodeFieldName
			} else if frame.isConditionalMatched() {
				// 如果是条件匹配，fieldName应该为当前帧的fieldName，因为子节点fieldName是关键词+条件表达式
				target = frame.target
				fieldName = frame.fieldName
			} else {
				// 不是顶层节点，赋值给当前帧的target
				frame.target.(map[string]any)[frame.fieldName] = make(map[string]any)
				target = frame.target.(map[string]any)[frame.fieldName]
				fieldName = subNodeFieldName
			}

			frameStack.Push(&ParseFrame{
				node:       subNode,
				fieldName:  fieldName,
				target:     target,
				curSubNode: flattenNode(subNode).First(),
				conditionalCtx: &ConditionalContext{
					isMatched: false,
				},
			})
			continue
		default:
			// 其他类型直接赋值
			if frame.isConditionalMatched() {
				frame.target.(map[string]any)[frame.fieldName] = e.replaceExpression(&subNode)
			} else {
				frame.target.(map[string]any)[subNodeFieldName] = e.replaceExpression(&subNode)
			}
			continue
		}
	}

	return result
}

func (e *JTEngine) setExpressionResult(frame *ParseFrame) {
	result := e.replaceExpression(frame.conditionalCtx.resultValue)
	if e.isFrameAtTopLevel(frame) {
		frame.target = result
	} else {
		frame.target.(map[string]any)[frame.fieldName] = result
	}
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

// isFrameAtTopLevel 判断当前帧是否在模板的顶层
func (e *JTEngine) isFrameAtTopLevel(frame *ParseFrame) bool {
	return frame.fieldName == e.entry
}

func flattenNode(node gjson.Result) *deque.Deque[*ds.Pair[gjson.Result, gjson.Result]] {
	var result = ds.NewDeque[*ds.Pair[gjson.Result, gjson.Result]]()
	node.ForEach(func(k, v gjson.Result) bool {
		result.PushBack(ds.MakePair(k, v))
		return true
	})
	return result
}
