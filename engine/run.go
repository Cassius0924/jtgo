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
func (e *JTEngine) Run() (string, error) {
	slog.InfoContext(e.ctx, "[JSONTemplateEngine.Run](trace) Run function start")
	defer func() {
		// 清空调用链
		e.clear()
		slog.InfoContext(e.ctx, "[JSONTemplateEngine.Run](trace) Run function end")
	}()
	return e.keepStatusRun()
}

// keepStatusRun 保持状态运行
func (e *JTEngine) keepStatusRun() (string, error) {
	if err := e.checkBeforeRun(); err != nil {
		return "", err
	}

	defer func() {
		// 恢复局部变量
		for k, v := range e.localVariables {
			e.dataset[k] = v
			slog.InfoContext(e.ctx, fmt.Sprintf("[JSONTemplateEngine.Run] restore variable,\nkey = %s,\nvalue = %v", k, v))
		}
		e.localVariables = make(map[string]any)
	}()

	// 获取入口的模板
	entryTemplateNode := gjson.Get(e.template, e.entry)
	if !entryTemplateNode.Exists() {
		slog.ErrorContext(e.ctx, "[JSONTemplateEngine.Run] entry not exists, please check whether entry name exists in the config JSON!", "entry", e.entry, "template", e.template)
		return "", werror.ErrEntryNotFound
	}

	// 循环解析方法
	result := e.interativeParse(&entryTemplateNode, e.entry)

	resultStr := util.SonicToString(result)

	if e.target != nil {
		// 将模板解析结果反序列化给target
		err := sonic.UnmarshalString(resultStr, e.target)
		if err != nil {
			slog.ErrorContext(e.ctx, "[JSONTemplateEngine.Run] UnmarshalFromString config error, please check whether the template JSON field name matches the target structure field name!", "finalTarget", util.GenerateStructFormatedString(result), "error", err)
			return resultStr, werror.Join(werror.ErrParseToTargetFailed, err)
		}
	}

	return resultStr, e.err
}

// interativeParse 迭代解析方法
func (e *JTEngine) interativeParse(templateNode *gjson.Result, templateFieldName string) any {
	var (
		frameStack     = ds.NewStack[*ParseFrame]()
		result     any = make(map[string]any)
	)
	// 初始化栈
	frameStack.Push(&ParseFrame{
		Node:           templateNode,
		FieldName:      templateFieldName,
		Target:         result,
		Result:         result,
		AssistResult:   make(map[string]any, 1),
		SubNodeIter:    flattenNode(templateNode).First(),
		ConditionalCtx: &ConditionalContext{
			IsMatched: false,
		},
	})

	// 迭代解析
	for frameStack.Size() > 0 {
		var (
			frame                 = frameStack.Top() // 获取栈顶帧
			subNodeField, subNode gjson.Result
		)

		// 如果子节点迭代器无效，或匹配到了值，说明当前帧的子节点已经遍历完毕，弹出栈顶元素
		if !frame.SubNodeIter.IsValid() || frame.ConditionalCtx.IsMatched || frame.LoopCtx != nil {
			frameStack.Pop()
			// 如果栈为空，说明所有帧都已经遍历完毕，返回结果
			if frameStack.Size() == 0 {

				if frame.CurSubNodeFieldName == "" {
					result = frame.Result
				} else if frame.LoopCtx != nil && frame.LoopCtx.IsSerialFor {
					result = frame.Result
				}

				break
			}

			if frame.ConditionalCtx.IsNestedCondition {
				frame.AssistResult["value"] = frame.Result
			} else if assistRes, ok := frame.AssistResult["value"]; ok {
				for k := range frame.Target.(map[string]any) {
					delete(frame.Target.(map[string]any), k)
				}
				frame.Target.(map[string]any)[frame.FieldName] = assistRes
				delete(frame.AssistResult, "value")
			} else {
				frame.Target.(map[string]any)[frame.FieldName] = frame.Result
			}
			continue
		}

		// 取出当前值后，继续遍历，迭代器指向下一个元素
		subNodePair := frame.SubNodeIter.Value()
		subNodeField, subNode = subNodePair.First, subNodePair.Second
		frame.SubNodeIter.Next()

		subNodeFieldName := normalizeFieldName(subNodeField.String())
		frame.CurSubNodeFieldName = subNodeFieldName

		switch keyword, statement := detectKeyword(subNodeFieldName); keyword {
		case KeywordDo:
			e.doOperations(&subNode)
			continue
		case KeywordVar:
			e.varAssignment(&subNode)
			continue
		case KeywordDefault:
			if !frame.ConditionalCtx.IsMatched {
				// frame.ConditionalCtx.ResultValue = &subNode // 记录下默认值
			}
			continue
		case KeywordIf:
			// if、elif、else和for都与当前帧相关，需要传入当前帧用以记录相关上下文
			matched := e.judgeConditionalIf(&subNode, statement, frame)
			// 如果未命中当前条件，则需要继续尝试下一个条件
			if !matched {
				continue
			}
		case KeywordElif:
			matched := e.judgeConditionalElif(&subNode, statement, frame)
			if !matched {
				continue
			}
		case KeywordElse:
			matched := e.judgeConditionalElse(&subNode, frame)
			if !matched {
				continue
			}
		case KeywordFor:
			e.executeLoop(&subNode, statement, frame)
			continue
		case KeywordContinue:
			e.continueLoop(&subNode, frame)
			continue
		case KeywordComment:
			// 注释，跳过
			continue
		default:
			// 其他情况，继续处理
		}

		switch {
		case subNode.IsObject():
			// 是Object，需要继续解析，将解析帧压入栈中
			frameStack.Push(&ParseFrame{
				Node:           &subNode,
				FieldName:      subNodeFieldName,
				Target:         frame.Result,
				Result:         make(map[string]any),
				AssistResult:   frame.AssistResult,
				SubNodeIter:    flattenNode(&subNode).First(),
				ConditionalCtx: &ConditionalContext{
					IsMatched: false,
				},
			})
			continue
		default:

			if frame.isConditionalMatched() {
				frame.Result = e.replaceExpression(frame.ConditionalCtx.MatchedValue)
			} else if subNodeFieldName == "" {
				frame.Result = e.replaceExpression(&subNode)
			} else {
				frame.Result.(map[string]any)[subNodeFieldName] = e.replaceExpression(&subNode)
			}
			continue
		}
	}

	return result
}

// checkBeforeRun 检查是否有必要的参数
func (e *JTEngine) checkBeforeRun() error {
	if e.template == "" {
		slog.ErrorContext(e.ctx, "[JSONTemplateEngine.check] configJSON is empty")
		return werror.ErrTemplateIsEmpty
	}
	return nil
}

// isFrameAtTopLevel 判断当前帧是否在模板的顶层
func (e *JTEngine) isFrameAtTopLevel(frame *ParseFrame, entry string) bool {
	return frame.FieldName == entry
}

func flattenNode(node *gjson.Result) *deque.Deque[*ds.Pair[gjson.Result, gjson.Result]] {
	var result = ds.NewDeque[*ds.Pair[gjson.Result, gjson.Result]]()
	node.ForEach(func(k, v gjson.Result) bool {
		result.PushBack(ds.MakePair(k, v))
		return true
	})
	return result
}
