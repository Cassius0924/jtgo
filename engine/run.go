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
		if err := sonic.UnmarshalString(resultStr, e.target); err != nil {
			slog.ErrorContext(e.ctx, "[JSONTemplateEngine.Run] UnmarshalFromString config error, please check if the template JSON field name matches the target structure field name!", "finalTarget", util.GenerateStructFormatedString(result), "error", err)
			return resultStr, werror.Join(werror.ErrParseToTargetFailed, err)
		}
	}

	return resultStr, e.err
}

// interativeParse 迭代解析方法
// 通过迭代方式解析JSON模板，将模板转换为最终输出结果
// templateFieldName: 模板节点的字段名
func (e *JTEngine) interativeParse(templateNode *gjson.Result, templateFieldName string) any {
	var (
		frameStack     = ds.NewStack[*ParseFrame]() // 解析帧栈，用于深度优先遍历JSON树
		result     any = make(map[string]any)       // 初始化结果为空map
	)

	// 解析栈的工作原理:
	// 1. 每个解析帧(ParseFrame)代表JSON模板中的一个层级节点
	// 2. 解析帧的 Target 指向的是上一解析帧的 Result，用于将当前结果传递给父级
	// 3. Result 为当前解析帧的最终结果，存储当前节点解析后的数据
	// 4. 当前帧处理完毕后，会将 Result 按照 FieldName 设置到父帧的 Target 中
	// 5. 整个过程是深度优先遍历，从根节点开始，依次处理每个子节点
	//
	// 栈结构示意图 (从左到右为栈底到栈顶):
	// |----------------------------------------------> Stack Top
	// | |-Frame01-|     |-Frame02-|     |-Frame03-|
	// | | Node    |     | Node    |     | Node    |   <- 当前处理的JSON节点
	// | | Result  |<-+  | Result  |<-+  | Result  |   <- 当前帧的解析结果
	// | | Target  |  +--| Target  |  +--| Target  |   <- 指向父帧的Result
	// | | ...     |     | ...     |     | ...     |
	// | |---------|     |---------|     |---------|
	// |----------------------------------------------> Stack Top
	//
	// 特殊情况处理:
	// - 条件语句(if/elif/else): 通过ConditionalCtx跟踪条件状态
	// - 循环语句(for): 通过LoopCtx管理循环迭代
	// - 变量操作(var/do): 直接执行不入栈
	//
	// 初始化栈，将根节点压入栈中
	frameStack.Push(&ParseFrame{
		Node:         templateNode,
		FieldName:    templateFieldName,
		Target:       result,
		Result:       result,
		AssistResult: make(map[string]any, 1),           // 辅助结果，用于存储临时数据
		SubNodeIter:  flattenNode(templateNode).First(), // 子节点迭代器
		ConditionalCtx: &ConditionalContext{ // 条件上下文
			IsMatched: false, // 初始状态未匹配
		},
	})

	// 迭代解析，直到栈为空
	for frameStack.Size() > 0 {
		var (
			frame                 = frameStack.Top() // 获取栈顶帧
			subNodeField, subNode gjson.Result       // 子节点字段名和子节点
		)

		// 以下三种情况需要弹出当前帧:
		// 1. 子节点迭代器无效（已遍历完所有子节点）
		// 2. 已匹配到条件语句
		// 3. 存在循环上下文（表示循环已处理完毕）
		if !frame.SubNodeIter.IsValid() || frame.ConditionalCtx.IsMatched || frame.LoopCtx != nil {
			frameStack.Pop()
			// 如果栈为空，说明所有帧都已经遍历完毕，处理最终结果并返回
			if frameStack.Size() == 0 {
				// 如果当前字段名为空或者是循环结果，则直接返回Result
				if frame.CurSubNodeFieldName == "" {
					result = frame.Result
				} else if frame.LoopCtx != nil && frame.LoopCtx.IsSerialFor {
					result = frame.Result
				}
				break
			}

			// 将当前帧的结果传递给父帧
			if frame.ConditionalCtx.IsNestedCondition {
				// 嵌套条件语句的处理
				frame.AssistResult["value"] = frame.Result
			} else if assistRes, ok := frame.AssistResult["value"]; ok {
				// 处理辅助结果
				for k := range frame.Target.(map[string]any) {
					delete(frame.Target.(map[string]any), k)
				}
				frame.Target.(map[string]any)[frame.FieldName] = assistRes
				delete(frame.AssistResult, "value")
			} else {
				// 常规结果处理
				frame.Target.(map[string]any)[frame.FieldName] = frame.Result
			}
			continue
		}

		// 取出当前子节点，并将迭代器指向下一个元素
		subNodePair := frame.SubNodeIter.Value()
		subNodeField, subNode = subNodePair.First, subNodePair.Second
		frame.SubNodeIter.Next()

		subNodeFieldName := normalizeFieldName(subNodeField.String())
		frame.CurSubNodeFieldName = subNodeFieldName

		// 处理模板语法关键字
		switch keyword, statement := detectKeyword(subNodeFieldName); keyword {
		case KeywordDo:
			e.doOperations(&subNode)
			continue
		case KeywordVar:
			e.varAssignment(&subNode)
			continue
		case KeywordIf:
			matched := e.judgeConditionalIf(&subNode, statement, frame)
			// 如果未命中当前条件，则继续尝试下一个条件
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
			// 注释，不做任何处理
			continue
		default:
			// 其他字段名，继续正常处理
		}

		// 根据节点类型进行不同处理
		switch {
		case subNode.IsObject():
			// 如果是对象类型，需要创建新的解析帧并压入栈中，继续深度遍历
			frameStack.Push(&ParseFrame{
				Node:         &subNode,
				FieldName:    subNodeFieldName,
				Target:       frame.Result,
				Result:       make(map[string]any),
				AssistResult: frame.AssistResult,
				SubNodeIter:  flattenNode(&subNode).First(),
				ConditionalCtx: &ConditionalContext{
					IsMatched: false,
				},
			})
			continue
		default:
			// 处理基本类型节点和其他情况
			if frame.isConditionalMatched() {
				// 条件语句匹配成功，使用条件匹配值
				frame.Result = e.replaceExpression(frame.ConditionalCtx.MatchedValue)
			} else if subNodeFieldName == "" {
				// 无字段名，直接设置结果
				frame.Result = e.replaceExpression(&subNode)
			} else {
				// 有字段名，设置结果的对应字段
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

// flattenNode 将 gjson.Result 节点的所有子节点扁平化为一个双端队列
// TODO: 改成内存池
func flattenNode(node *gjson.Result) *deque.Deque[*ds.Pair[gjson.Result, gjson.Result]] {
	var result = ds.NewDeque[*ds.Pair[gjson.Result, gjson.Result]]()
	node.ForEach(func(k, v gjson.Result) bool {
		result.PushBack(ds.MakePair(k, v))
		return true
	})
	return result
}