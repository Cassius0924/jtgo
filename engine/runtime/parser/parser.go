package parser

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/engine/common"
	"github.com/cassius0924/jtgo/engine/exprs"
	"github.com/cassius0924/jtgo/engine/keywords"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/cassius0924/jtgo/util"
	"github.com/cassius0924/jtgo/util/ptr"
	"github.com/cassius0924/jtgo/werror"
	"github.com/expr-lang/expr/vm"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"
)

type Parser struct {
	exprHandler    *exprs.ExprHandler         // 表达式处理器
	loopMetas      map[string]*model.LoopMeta // 循环语句元数据
	dataset        map[string]any             // 数据集
	localVariables map[string]any             // 局部变量名称和值

	err error
}

// NewParser 创建一个新的解析器实例
func NewParser(exprHandler *exprs.ExprHandler, compiledExps map[string]*vm.Program, loopMetas map[string]*model.LoopMeta) *Parser {
	return &Parser{
		exprHandler:    exprHandler,
		loopMetas:      loopMetas,
		localVariables: make(map[string]any),
	}
}

// SetDataset 设置数据集
func (p *Parser) SetDataset(dataset map[string]any) {
	p.dataset = dataset
}

// Parse 解析JSON模板
func (p *Parser) Parse(ctx context.Context, template, entry string, target any) (string, error) {
	// 获取入口的模板
	entryTemplateNode := gjson.Get(template, entry)
	if !entryTemplateNode.Exists() {
		slog.ErrorContext(ctx, "[parser.Parse] entry not exists, please check whether entry name exists in the config JSON!", "entry", entry, "template", template)
		return "", werror.ErrEntryNotFound
	}

	// 循环解析方法
	result := p.iterativeParse(ctx, &entryTemplateNode, entry)
	resultStr := util.SonicToString(result)

	if target != nil {
		// 将模板解析结果反序列化给target
		if err := sonic.UnmarshalString(resultStr, target); err != nil {
			slog.ErrorContext(ctx, "[parser.Parse] UnmarshalFromString config error, please check if the template JSON field name matches the target structure field name!", "finalTarget", util.GenerateStructFormattedString(result), "error", err)
			return resultStr, werror.Join(werror.ErrParseToTargetFailed, err)
		}
	}

	return resultStr, p.err
}

// iterativeParse 迭代解析方法
// 通过迭代方式解析JSON模板，将模板转换为最终输出结果
// templateFieldName: 模板节点的字段名
func (p *Parser) iterativeParse(ctx context.Context, templateNode *model.TNode, templateFieldName string) any {
	var (
		frameStack     = ds.NewStackWithListContainer[*model.ParseFrame]()                                                                             // 解析帧栈，用于深度优先遍历JSON树
		result     any = lo.TernaryF(templateNode.IsArray(), func() any { return ptr.Of(make([]any, 0)) }, func() any { return make(map[string]any) }) // 初始化结果为空map或slice
	)

	if !templateNode.IsObject() && !templateNode.IsArray() {
		return p.replaceExpression(ctx, templateNode)
	}

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
	frameStack.Push(&model.ParseFrame{
		Node:        templateNode,
		FieldName:   templateFieldName,
		Target:      result,
		Result:      result,
		SharedMemo:  make(map[string]any, 2),
		SubNodeIter: common.FlattenNode(templateNode).First(), // 子节点迭代器
		CondContext: &model.ConditionalContext{ // 条件上下文
			IsMatched: false, // 初始状态未匹配
		},
	})

	// 迭代解析，直到栈为空
	for frameStack.Size() > 0 {
		var (
			frame                 = frameStack.Top() // 获取栈顶帧
			subNodeField, subNode model.TNode        // 子节点字段名和子节点
		)

		// 以下三种情况需要弹出当前帧:
		// 1. 子节点迭代器无效（已遍历完所有子节点）
		// 2. 在不需要遍历完所有字节点的情况下，匹配到条件语句（再需要遍历完所有节点的情况，即使匹配到条件语句，也需要继续解析，直至所有子节点解析完毕）
		// 3. 存在循环上下文（表示循环已处理完毕）
		if !frame.SubNodeIter.IsValid() || (!frame.ShouldTraverseAllSubNodes() && frame.IsConditionalMatched()) || frame.HasLoopContext() {
			frame.SharedMemo["no_match"] = false
			// 此处用于清空 变量赋值中 的标记
			if keywords.IsVarKeyword(frame.FieldName) {
				frame.SharedMemo["assigning_variable"] = false
			}
			// 此处用于清空 执行操作中 的标记
			if keywords.IsExecKeyword(frame.FieldName) {
				frame.SharedMemo["executing_operation"] = false
			}

			frameStack.Pop()
			// 如果栈为空，说明所有帧都已经遍历完毕，处理最终结果并返回
			if frameStack.Size() == 0 {
				// 如果当前字段名为空或者是循环结果，则直接返回Result
				if frame.LoopContext != nil && frame.LoopContext.IsSerialFor {
					result = frame.Result
				}
				break
			}

			if frame.ShouldTraverseAllSubNodes() {
				continue
			}

			// 将当前帧的结果传递给父帧
			if keywords.IsAnyKeyword(frame.FieldName) {
				// 对于关键字字段，确保辅助结果中存储了当前结果
				if resultMap, ok := frame.Result.(map[string]any); ok && len(resultMap) == 0 {
					//  如果当前结果为空，则说明父条件语句不成立，通过 no_match 标记
					frame.SharedMemo["no_match"] = true
				} else if _, ok := frame.SharedMemo["assist_result"]; !ok {
					frame.SharedMemo["assist_result"] = frame.Result
				}
			} else {
				// 非关键字字段的处理
				var resultValue any
				if assistRes, ok := frame.SharedMemo["assist_result"]; ok {
					// 如果存在辅助结果，使用辅助结果作为值
					resultValue = assistRes
					delete(frame.SharedMemo, "assist_result")
				} else {
					resultValue = frame.Result
				}
				// 将结果设置到父帧的目标中
				if target, ok := frame.Target.(*[]any); ok {
					// 如果目标是数组类型，直接追加结果
					*target = append(*target, resultValue)
				} else {
					frame.Target.(map[string]any)[frame.FieldName] = resultValue
				}
			}
			continue
		}
		frame.SharedMemo["no_match"] = false

		// 取出当前子节点，并将迭代器指向下一个元素
		subNodePair := frame.SubNodeIter.Value()
		frame.SubNodeIter.Next()
		subNodeField, subNode = subNodePair.First, subNodePair.Second
		subNodeFieldName := common.NormalizeFieldName(subNodeField.String())

		// 处理模板语法关键字
		keyword, statement := keywords.DetectKeyword(subNodeFieldName)
		if keyword != "" {
			// 获取关键字处理器并执行处理
			processor := GetProcessor(keyword)
			if processor == nil {
				slog.ErrorContext(ctx, "[parser.iterativeParse] keyword processor not found", "keyword", keyword)
				p.err = werror.ErrProcessorNotRegistered
				return nil
			}

			// 如果处理器返回false，表示不需要继续处理当前节点
			processCurrentNode := processor.Process(ctx, &subNode, statement, frame, p)
			if !processCurrentNode {
				continue
			}
		}

		// 根据节点类型进行不同处理
		switch {
		case subNode.IsObject():
			// 如果是对象类型，需要创建新的解析帧并压入栈中，继续深度遍历
			frameStack.Push(&model.ParseFrame{
				Node:        &subNode,
				FieldName:   subNodeFieldName,
				Target:      frame.Result,
				Result:      lo.TernaryF(subNode.IsArray(), func() any { return ptr.Of(make([]any, 0)) }, func() any { return make(map[string]any) }),
				SharedMemo:  frame.SharedMemo,
				SubNodeIter: common.FlattenNode(&subNode).First(),
				CondContext: &model.ConditionalContext{
					IsMatched: false,
				},
			})
			continue
		case subNode.IsArray():
			// 如果是数组类型，需要反向遍历数组元素
			// 并且 target指向一个新切片
			var (
				arr        = subNode.Array()
				target any = ptr.Of(make([]any, 0))
			)
			if result, ok := frame.Result.(*[]any); ok {
				*result = append(*result, target)
			} else {
				frame.Result.(map[string]any)[subNodeFieldName] = target
			}

			for i := len(arr) - 1; i >= 0; i-- {
				frameStack.Push(&model.ParseFrame{
					Node:        &arr[i],
					FieldName:   subNodeFieldName,
					Target:      target,
					Result:      lo.TernaryF(arr[i].IsArray(), func() any { return ptr.Of(make([]any, 0)) }, func() any { return make(map[string]any) }),
					SharedMemo:  frame.SharedMemo,
					SubNodeIter: common.FlattenNode(&arr[i]).First(),
					CondContext: &model.ConditionalContext{
						IsMatched: false,
					},
				})
			}
		default:
			if frame.IsAssigningVariable() {
				// 正在进行变量赋值
				p.localVariables[subNodeFieldName] = p.dataset[subNodeFieldName]
				p.dataset[subNodeFieldName] = p.replaceExpression(ctx, &subNode)
				slog.InfoContext(ctx, fmt.Sprintf("[parser.assignVariables](trace) create variable,\nkey = %s,\nvalue = %s,\nexpr = %s", subNodeFieldName, util.GenerateStructFormattedString(p.dataset[subNodeFieldName]), subNode.String()))
			} else if frame.IsExecutingOperation() {
				// 正在执行操作语句
				

			} else if frame.IsConditionalMatched() {
				// 条件语句匹配成功，使用条件匹配值
				frame.Result = p.replaceExpression(ctx, frame.CondContext.MatchedValue)
			} else {
				// 有字段名，设置结果的对应字段
				if result, ok := frame.Result.(*[]any); ok {
					*result = append(*result, p.replaceExpression(ctx, &subNode))
				} else {
					frame.Result.(map[string]any)[subNodeFieldName] = p.replaceExpression(ctx, &subNode)
				}
			}
			continue
		}
	}

	return result
}

// replaceExpression 递归替换 object 中所有含有${Expression}的值
func (p *Parser) replaceExpression(ctx context.Context, input *model.TNode) any {
	if input == nil || !input.Exists() {
		return nil
	}
	// JSON 共有 4 种类型: bool, number, null, string
	switch {
	case input.IsBool(): // bool
		return input.Bool()
	case input.Type == gjson.Number: // number
		return input.Num
	case input.Type == gjson.Null: // null
		return nil
	default: // string
		return p.exprHandler.EvaluateExpressionsInText(ctx, input.String()) // 使用数据集中的值替换字符串中的变量
	}
}
