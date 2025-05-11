package parser

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/engine/common"
	"github.com/cassius0924/jtgo/engine/exprs"
	"github.com/cassius0924/jtgo/engine/flags"
	"github.com/cassius0924/jtgo/engine/keywords"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/cassius0924/jtgo/util"
	"github.com/cassius0924/jtgo/util/ptr"
	"github.com/cassius0924/jtgo/werror"
	"github.com/expr-lang/expr/vm"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"
)

// Parser 模板解析器
type Parser struct {
	exprHandler    *exprs.ExprHandler         // 表达式处理器
	loopMetas      map[string]*model.LoopMeta // 循环语句元数据
	subNodeIters   map[string]*model.NodeIter // 子节点迭代器
	dataset        map[string]any             // 数据集
	localVariables map[string]any             // 局部变量名称和值

	err error
}

// NewParser 创建一个新的解析器实例
func NewParser(exprHandler *exprs.ExprHandler, compiledExps map[string]*vm.Program, loopMetas map[string]*model.LoopMeta, subNodeIters map[string]*model.NodeIter) *Parser {
	return &Parser{
		exprHandler:    exprHandler,
		loopMetas:      loopMetas,
		subNodeIters:   subNodeIters,
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
	result := p.iterativeParse(ctx, model.NewTNode(entryTemplateNode, nil), entry)
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
		return p.transformNodeToValue(ctx, templateNode)
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
	// 初始化栈，将根节点压入栈中
	templateFieldName = common.NormalizeFieldName(templateFieldName) // 规范化字段名
	if subNodeIter, ok := p.subNodeIters[templateFieldName]; ok {
		// 重置迭代器，以确保从头开始遍历子节点
		frameStack.Push(&model.ParseFrame{
			Node:        templateNode,
			FieldName:   templateFieldName,
			Target:      result,
			Result:      result,
			SharedMemo:  make(map[string]any, 2),
			Path:        templateFieldName,
			SubNodeIter: subNodeIter.IteratorAt(0).(*model.NodeIter),
		})
	} else {
		slog.ErrorContext(ctx, "[parser.iterativeParse] subNodeIter not found", "path", templateFieldName)
		return nil
	}

	// 迭代解析，直到栈为空
	for frameStack.Size() > 0 {
		frame := frameStack.Top() // 获取栈顶帧

		// 循环一遍结束，需要继续下一遍循环
		if frame.Node.NodeFlag.Has(flags.NodeFlagLooping) && !frame.SubNodeIter.IsValid() {
			// 重置迭代器，并且将索引加1
			frame.SubNodeIter = frame.SubNodeIter.IteratorAt(0).(*model.NodeIter)
			frame.Node.LoopContext.DealNextItem()
		}

		// 以下三种情况需要弹出当前帧:
		// 1. 在非处理循环时，子节点迭代器无效（已遍历完所有子节点）
		// 2. 在不需要遍历完所有子节点的情况下，匹配到条件语句（再需要遍历完所有节点的情况，即使匹配到条件语句，也需要继续解析，直至所有子节点解析完毕）
		// 3. 循环已遍历完毕（处理循环时）
		if (!frame.Node.NodeFlag.Has(flags.NodeFlagLooping) && !frame.SubNodeIter.IsValid()) || (frame.InNormalScope() && frame.IsConditionalMatched()) || frame.IsLoopDone() {
			frameStack.Pop()
			frame.SharedMemo["no_match"] = false
			// 离开 var 作用域，需要清空 变量赋值中 的标记
			if keywords.IsVarKeyword(frame.FieldName) {
				frame.SharedMemo["assigning_variable"] = false
				continue
			}
			// 离开 exec 作用域，需要清空 执行操作中 的标记
			if keywords.IsExecKeyword(frame.FieldName) {
				frame.SharedMemo["executing_operation"] = false
				continue
			}
			// 循环完成，将循环结果放入 Result
			if frame.IsLoopDone() {
				frame.Result = frame.Node.LoopContext.ResultList
			}

			if !frame.InNormalScope() {
				continue
			}

			// 如果栈为空，说明所有帧都已经遍历完毕，处理最终结果并返回
			if frameStack.Size() == 0 {
				break
			}

			// 如果父帧字段名是关键词，则需要将当前帧的结果传递给父帧，而不能直接复制给 Target
			if frame.Node.IsAnyKeyword() {
				// 对于关键字字段，确保辅助结果中存储了当前结果
				if resultMap, ok := frame.Result.(map[string]any); ok && len(resultMap) == 0 {
					//  如果当前结果为空，则说明父条件语句不成立，通过 no_match 标记
					frame.SharedMemo["no_match"] = true
				} else {
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
				// TODO: Target 可以删去，使用Parent的Result代替
				if parentNode := frame.Node.Parent; parentNode != nil && parentNode.NodeFlag.Has(flags.NodeFlagLooping) {
					result := parentNode.LoopContext.ResultList[parentNode.LoopContext.Index]
					util.AppendOrSet(result, frame.FieldName, resultValue)
				} else {
					frame.PutTarget(resultValue)
				}

			}
			continue
		}
		frame.SharedMemo["no_match"] = false

		// 取出当前子节点，并将迭代器指向下一个元素
		subNodeField, subNode := frame.ExtractSubNodePair()
		frame.SubNodeIter.Next()
		subNodeFieldName := common.NormalizeFieldName(subNodeField.String())

		// 处理模板语法关键字
		if subNode.IsAnyKeyword() {
			// 获取关键字处理器并执行处理
			processor := GetProcessor(subNode.Keyword)
			if processor == nil {
				slog.ErrorContext(ctx, "[parser.iterativeParse] keyword processor not found", "keyword", subNode.Keyword)
				p.err = werror.ErrProcessorNotRegistered
				return nil
			}

			// 如果处理器返回false，表示不需要继续处理当前节点
			processCurrentNode := processor.Process(ctx, subNode, subNode.Statement, frame, p)
			if !processCurrentNode {
				continue
			}
		}

		// 根据节点类型进行不同处理
		switch {
		case subNode.IsObject() || subNode.IsArray():
			// Object 和 Array，需要创建新的解析帧并压入栈中，继续深度遍历
			// TODO: 移动到编译期
			path := frame.BuildNodePath(subNodeFieldName)
			if subNodeIter, ok := p.subNodeIters[path]; ok {
				frameStack.Push(&model.ParseFrame{
					Node:        subNode,
					FieldName:   subNodeFieldName,
					Target:      frame.Result,
					Result:      lo.TernaryF(subNode.IsArray(), func() any { return ptr.Of(make([]any, 0)) }, func() any { return make(map[string]any) }),
					SharedMemo:  frame.SharedMemo,
					Path:        path,
					SubNodeIter: subNodeIter.IteratorAt(0).(*model.NodeIter),
				})
			} else {
				slog.ErrorContext(ctx, "[parser.iterativeParse] subNodeIter not found", "path", path)
				return nil
			}
			continue
		default:
			if frame.InExecScope() || subNode.NodeFlag.Has(flags.NodeFlagExecOnce) {
				if subNode.NodeFlag.HasAny(flags.NodeFlagGeneralObjectItem, flags.NodeFlagKeywordCmt, flags.NodeFlagKeywordReturn, flags.NodeFlagKeywordFor, flags.NodeFlagKeywordVar) {
					continue
				}
				// 在 exec 作用域
				p.executeOperationsOnNode(ctx, subNode)
				slog.InfoContext(ctx, fmt.Sprintf("[parser.iterativeParse](trace) execute operation,\nkey = %s,\nvalue = %s,\nexpr = %s", subNodeFieldName, util.GenerateStructFormattedString(p.dataset[subNodeFieldName]), subNode.String()))
			} else if subNode.CondContext != nil && subNode.IsConditionalMatched() {
				if !frame.InNormalScope() {
					continue
				}
				// 条件语句匹配成功，使用条件匹配值
				frame.Result = p.transformNodeToValue(ctx, subNode.CondContext.MatchedValue)
				slog.InfoContext(ctx, fmt.Sprintf("[parser.iterativeParse](trace) condition matched,\nkey = %s,\nvalue = %s", subNodeFieldName, util.GenerateStructFormattedString(frame.Result)))
			} else if frame.InVarScope() {
				// 在 var 作用域
				p.localVariables[subNodeFieldName] = p.dataset[subNodeFieldName]
				p.dataset[subNodeFieldName] = p.transformNodeToValue(ctx, subNode)
				slog.InfoContext(ctx, fmt.Sprintf("[parser.iterativeParse](trace) assign variable,\nkey = %s,\nvalue = %s,\nexpr = %s", subNodeFieldName, util.GenerateStructFormattedString(p.dataset[subNodeFieldName]), subNode.String()))
			} else if subNode.NodeFlag.Has(flags.NodeFlagLooping) && subNode.NodeFlag.Has(flags.NodeFlagNodeNonObjectOrArray) {
				// 在循环中，循环体为非对象或数组类型
				for !subNode.LoopContext.IsDone() {
					subNode.LoopContext.ResultList = append(subNode.LoopContext.ResultList, p.transformNodeToValue(ctx, subNode))
					subNode.LoopContext.DealNextItem()
				}
				frame.Result = subNode.LoopContext.ResultList
				slog.InfoContext(ctx, fmt.Sprintf("[parser.iterativeParse](trace) loop result,\nkey = %s,\nvalue = %s", subNodeFieldName, util.GenerateStructFormattedString(subNode.LoopContext.ResultList)))
			} else if frame.Node.NodeFlag.Has(flags.NodeFlagLooping) {
				// 在循环中
				result := frame.Node.LoopContext.ResultList[frame.Node.LoopContext.Index]
				util.AppendOrSet(result, subNodeFieldName, p.transformNodeToValue(ctx, subNode))
				slog.InfoContext(ctx, fmt.Sprintf("[parser.iterativeParse](trace) loop result,\nkey = %s,\nvalue = %s", subNodeFieldName, util.GenerateStructFormattedString(result)))
			} else {
				// 有字段名，设置结果的对应字段
				frame.PutResult(subNodeFieldName, p.transformNodeToValue(ctx, subNode))
			}
			continue
		}
	}

	return result
}

// transformNodeToValue 将节点转换为值
func (p *Parser) transformNodeToValue(ctx context.Context, node *model.TNode) any {
	switch {
	case node.IsObject(): // object
		slog.ErrorContext(ctx, "[parser.transformNodeToValue] object type should not be here.", "input", node.String())
		return nil
	case node.IsArray(): // array
		slog.ErrorContext(ctx, "[parser.transformNodeToValue] array type should not be here.", "input", node.String())
		return nil
	case node.IsBool(): // bool
		return node.Bool()
	case node.Type == gjson.Number: // number
		return node.Num
	case node.Type == gjson.Null: // null
		return nil
	default: // string
		return p.exprHandler.InterpolateString(ctx, node.String()) // 使用数据集中的值替换字符串中的变量
	}
}

// executeOperationsOnNode 执行操作
// TODO: 增加 error 返回
func (p *Parser) executeOperationsOnNode(ctx context.Context, node *model.TNode) {
	switch {
	case node.IsObject():
		slog.ErrorContext(ctx, "[parser.executeOperationsOnNode] object type should not be here.", "input", node.String())
		return
	case node.IsArray():
		slog.ErrorContext(ctx, "[parser.executeOperationsOnNode] array type should not be here.", "input", node.String())
		return
	case node.IsBool():
		slog.WarnContext(ctx, "[parser.executeOperationsOnNode] bool type should not be here.", "input", node.String())
		return
	case node.Type == gjson.Number:
		slog.WarnContext(ctx, "[parser.executeOperationsOnNode] number type should not be here.", "input", node.String())
		return
	case node.Type == gjson.Null:
		slog.WarnContext(ctx, "[parser.executeOperationsOnNode] null type should not be here.", "input", node.String())
		return
	default:
		// 执行操作
		p.exprHandler.ExecuteAllExpressionsInText(ctx, node.String())
		slog.InfoContext(ctx, fmt.Sprintf("[parser.executeOperationsOnNode](trace) execute operation,\nkey = %s,\nvalue = %s", node.String(), node.String()))
	}
}
