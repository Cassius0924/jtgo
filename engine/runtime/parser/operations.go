package parser

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/cassius0924/jtgo/engine/flags"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/cassius0924/jtgo/util"
	"github.com/cassius0924/jtgo/util/ptr"
	"github.com/cassius0924/jtgo/werror"
	"github.com/samber/lo"
	"github.com/tidwall/gjson"
)

// executeOperations 处理 exec 操作
func (p *Parser) executeOperations(ctx context.Context, node *model.TNode, frame *model.ParseFrame) bool {
	var (
		processCurrentNode = false
	)
	// @exec 允许 node 为 Object、Array、String
	if node.IsObject() || node.IsArray() {
		frame.SharedMemo["executing_operation"] = true
		slog.InfoContext(ctx, fmt.Sprintf("[parser.executeOperations](trace) start exec operations,\nkey = %s,\nvalue = %s", frame.FieldName, node.String()))
		processCurrentNode = true
	} else if node.Type == gjson.String {
		// 对于字符串，由于不会创建新解析帧，所以这里使用 NodeFlag 来标记
		node.NodeFlag.Set(flags.NodeFlagExecOnce)
		slog.InfoContext(ctx, fmt.Sprintf("[parser.executeOperations](trace) set the exec once node flag,\nkey = %s,\nvalue = %s", frame.FieldName, node.String()))
		processCurrentNode = true
	}

	return processCurrentNode
}

// returnResult 处理 return 操作
func (p *Parser) returnResult(ctx context.Context, node *model.TNode) any {
	result := p.transformNodeToValue(ctx, node)
	slog.InfoContext(ctx, fmt.Sprintf("[parser.returnResult](trace) return result,\nresult = %s", util.GenerateStructFormattedString(result)))
	return result
}

// assignVariables 处理 var 变量赋值
func (p *Parser) assignVariables(ctx context.Context, node *model.TNode, frame *model.ParseFrame) bool {
	var (
		processCurrentNode = false
	)
	// @var 只允许 node 为 Object
	if node.IsObject() {
		frame.SharedMemo["assigning_variable"] = true
		processCurrentNode = true
		slog.InfoContext(ctx, fmt.Sprintf("[parser.assignVariables](trace) start assign variables,\nkey = %s,\nvalue = %s", frame.FieldName, node.String()))
	}
	return processCurrentNode
}

// judgeConditionalIf 处理if条件判断，返回是否命中该条件
func (p *Parser) judgeConditionalIf(ctx context.Context, node *model.TNode, expression string, frame *model.ParseFrame) bool {
	frame.LazyInitCondContextList()

	// 只在处理 if 时创建新的条件上下文
	node.CondContext = model.NewConditionalContext()

	// 将条件上下文添加到帧维度的链表中，以便 elif 和 else 语句可以访问
	frame.CondContextList.PushBack(node.CondContext)

	// 表达式计算为bool值
	matched, err := p.exprHandler.EvaluateExpressionToBool(ctx, expression)
	if err != nil {
		p.err = err
	}

	if matched {
		node.CondContext.MatchedValue = node
		node.CondContext.IsMatched = true
	}

	slog.InfoContext(ctx, fmt.Sprintf("[parser.judgeConditionalIf](trace) condition evaluate result,\nkey = %s,\nvalue = %s,\nexpr = %s", frame.FieldName, node.String(), expression))
	return matched
}

// judgeConditionalElif 处理elif条件判断，返回是否命中该条件
func (p *Parser) judgeConditionalElif(ctx context.Context, node *model.TNode, expression string, frame *model.ParseFrame) bool {
	// elif 寻找 if 所创建的条件上下文
	if frame.CondContextList == nil {
		slog.ErrorContext(ctx, "[parser.judgeConditionalElif] this elif is orphan, please check if the elif belongs to an if!", "key", frame.FieldName)
		return false
	}

	condContext := frame.CondContextList.Back()
	if condContext == nil || condContext.HasElseBranch {
		slog.ErrorContext(ctx, "[parser.judgeConditionalElif] this elif is orphan, please check if the elif belongs to an if!", "key", frame.FieldName)
		return false
	}
	condContext.HasElifBranch = true
	// 将条件上下文绑定到当前节点
	node.CondContext = condContext

	// 如果当前条件分支已经命中，则不需要再执行 elif 语句
	if condContext.IsMatched {
		return false
	}

	matched, err := p.exprHandler.EvaluateExpressionToBool(ctx, expression)
	if err != nil {
		p.err = err
	}

	if matched {
		condContext.MatchedValue = node
		condContext.IsMatched = true
	}

	slog.InfoContext(ctx, fmt.Sprintf("[parser.judgeConditionalIf](trace) condition evaluate result,\nkey = %s,\nvalue = %s,\nexpr = %s", frame.FieldName, node.String(), expression))
	return matched
}

// judgeConditionalElse 处理else条件判断
func (p *Parser) judgeConditionalElse(ctx context.Context, node *model.TNode, frame *model.ParseFrame) bool {
	// else 寻找 if 所创建的条件上下文
	if frame.CondContextList == nil {
		slog.ErrorContext(ctx, "[parser.judgeConditionalElse] this else is orphan, please check if the else belongs to an if!", "key", frame.FieldName)
		return false
	}

	condContext := frame.CondContextList.Back()
	if condContext == nil || condContext.HasElseBranch {
		slog.ErrorContext(ctx, "[parser.judgeConditionalElif] this elif is orphan, please check if the elif belongs to an if!", "key", frame.FieldName)
		return false
	}
	condContext.HasElseBranch = true
	node.CondContext = condContext

	// 如果当前条件分支已经命中，则不需要再执行 else 语句
	if node.CondContext.IsMatched {
		return false
	}

	condContext.MatchedValue = node
	condContext.IsMatched = true

	slog.InfoContext(ctx, fmt.Sprintf("[parser.judgeConditionalElse](trace) condition evaluate result,\nkey = %s,\nvalue = %s", frame.FieldName, node.String()))
	return true
}

// executeLoop 处理for循环
func (p *Parser) executeLoop(ctx context.Context, node *model.TNode, statement string, frame *model.ParseFrame) bool {
	// 如果循环上下文不存在，则是第一次执行循环，进行初始化
	node.NodeFlag.Set(flags.NodeFlagLooping)

	err := p.initLoopContext(ctx, node, statement)
	if err != nil {
		p.err = err
		return false
	}
	return true
}

// initLoopContext 初始化循环上下文
func (p *Parser) initLoopContext(ctx context.Context, node *model.TNode, statement string) error {
	node.LoopContext = model.NewLoopContext()
	// 取出编译期解析的循环元数据
	loopMeta, ok := p.loopMetas[statement]
	if !ok {
		slog.ErrorContext(ctx, "[parser.initLoopContext] loop meta not found", "statement", statement)
		return werror.ErrLoopMetaNotFound
	}

	// 取出编译期解析的循环对象
	object, err := p.exprHandler.EvaluateExpression(ctx, loopMeta.Object)
	if err != nil {
		slog.ErrorContext(ctx, "[parser.initLoopContext] exprRun when getting loop object", "statement", statement, "error", err)
		return err
	}

	var (
		objectRefl   = reflect.ValueOf(object)
		loopType     model.LoopType
		length       int
		dealNextItem func()
		resultList   []any
	)

	originKey := p.dataset[loopMeta.Key]
	originValue := p.dataset[loopMeta.Value]

	switch objectRefl.Kind() {
	case reflect.Slice, reflect.Array:
		loopType = lo.Ternary(objectRefl.Kind() == reflect.Array, model.LoopTypeForWithArray, model.LoopTypeForWithSlice)
		length = objectRefl.Len()

		dealNextItem = func() {
			node.LoopContext.Index++
			if !node.LoopContext.IsDone() {
				if loopMeta.Key != "" {
					p.dataset[loopMeta.Key] = node.LoopContext.Index
				}
				if loopMeta.Value != "" {
					p.dataset[loopMeta.Value] = objectRefl.Index(node.LoopContext.Index).Interface()
				}
			}
		}

	case reflect.Map:
		loopType = model.LoopTypeForWithMap
		mapIter := objectRefl.MapRange()
		length = objectRefl.Len()

		dealNextItem = func() {
			node.LoopContext.Index++
			if mapIter.Next() {
				if loopMeta.Key != "" {
					p.dataset[loopMeta.Key] = mapIter.Key().Interface()
				}
				if loopMeta.Value != "" {
					p.dataset[loopMeta.Value] = mapIter.Value().Interface()
				}
			}
		}

	default:
		slog.ErrorContext(ctx, "[parser.initLoopContext] the loop object is not rangeable", "object", object)
		return werror.ErrLoopObjectNotRangeable
	}

	resultList = make([]any, 0, length)
	if node.IsArray() {
		for i := 0; i < length; i++ {
			resultList = append(resultList, ptr.Of(make([]any, 0)))
		}
	} else if node.IsObject() {
		for i := 0; i < length; i++ {
			resultList = append(resultList, make(map[string]any))
		}
	}

	node.LoopContext.Type = loopType
	node.LoopContext.Length = length
	node.LoopContext.OriginKey = originKey
	node.LoopContext.OriginValue = originValue
	node.LoopContext.DealNextItem = dealNextItem
	node.LoopContext.ResultList = resultList
	node.LoopContext.Index = -1
	dealNextItem()
	return nil
}

// continueLoop 处理循环中的CONTINUE
func (p *Parser) continueLoop(ctx context.Context, node *model.TNode, frame *model.ParseFrame) {

}
