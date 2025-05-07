package parser

import (
	"context"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/cassius0924/jtgo/engine/keywords"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/cassius0924/jtgo/util"
	"github.com/cassius0924/jtgo/util/ptr"
	"github.com/cassius0924/jtgo/werror"
)

// execOperations 处理 exec 操作
func (p *Parser) execOperations(ctx context.Context, node *model.TNode, frame *model.ParseFrame) bool {
	frame.SharedMemo["executing_operation"] = true
	slog.InfoContext(ctx, fmt.Sprintf("[parser.execOperations](trace) start exec operations,\nkey = %s,\nvalue = %s", frame.FieldName, node.String()))
	return true
}

// returnResult 处理 return 操作
func (p *Parser) returnResult(ctx context.Context, node *model.TNode) any {
	result := p.replaceExpression(ctx, node)
	slog.InfoContext(ctx, fmt.Sprintf("[parser.returnResult](trace) return result,\nresult = %s", util.GenerateStructFormattedString(result)))
	return result
}

// assignVariables 处理 var 变量赋值
func (p *Parser) assignVariables(ctx context.Context, node *model.TNode, frame *model.ParseFrame) bool {
	frame.SharedMemo["assigning_variable"] = true
	slog.InfoContext(ctx, fmt.Sprintf("[parser.assignVariables](trace) start assign variables,\nkey = %s,\nvalue = %s", frame.FieldName, node.String()))
	return true
}

// judgeConditionalIf 处理if条件判断，返回是否命中该条件
func (p *Parser) judgeConditionalIf(ctx context.Context, node *model.TNode, expression string, frame *model.ParseFrame) bool {
	// 重置MatchedValue
	frame.ResetMatched()

	// 表达式计算为bool值
	matched, err := p.exprHandler.EvaluateExpressionToBool(ctx, expression)
	if err != nil {
		p.err = err
	}

	// 表达式为true，替换值，并剪枝结束循环
	if matched {
		frame.CondContext.MatchedValue = node
		frame.CondContext.IsMatched = true
	}

	frame.CondContext.HasIfBranch = true
	slog.InfoContext(ctx, fmt.Sprintf("[parser.judgeConditionalIf](trace) condition evaluate result,\nkey = %s,\nvalue = %s,\nexpr = %s", frame.FieldName, node.String(), expression))
	return matched
}

// judgeConditionalElif 处理elif条件判断，返回是否命中该条件
func (p *Parser) judgeConditionalElif(ctx context.Context, node *model.TNode, expression string, frame *model.ParseFrame) bool {
	if !frame.CondContext.HasIfBranch {
		// 是孤儿 elif 则视为 false
		slog.WarnContext(ctx, "[parser.judgeConditionalElif] this elif is orphan, please check if the elif belongs to an if!", "key", frame.FieldName)
		return false
	}

	// 如果当前条件分支已经命中，则不需要再执行 elif 语句
	if frame.CondContext.IsMatched {
		return false
	}

	matched, err := p.exprHandler.EvaluateExpressionToBool(ctx, expression)
	if err != nil {
		p.err = err
	}

	if matched {
		frame.CondContext.MatchedValue = node
		frame.CondContext.IsMatched = true
	}

	frame.CondContext.HasIfBranch = true
	slog.InfoContext(ctx, fmt.Sprintf("[parser.judgeConditionalIf](trace) condition evaluate result,\nkey = %s,\nvalue = %s,\nexpr = %s", frame.FieldName, node.String(), expression))
	return matched
}

// judgeConditionalElse 处理else条件判断
func (p *Parser) judgeConditionalElse(ctx context.Context, node *model.TNode, frame *model.ParseFrame) bool {
	// 判断当前 else 是否是孤儿else，即当前 else 是否属于某一个 if
	// TODO: 封装成函数
	if !frame.CondContext.HasIfBranch {
		// 是孤儿 else 则视为 false
		slog.WarnContext(ctx, "[parser.judgeConditionalElse] this else is orphan, please check if the else belongs to an if!", "key", frame.FieldName)
		return false
	}

	// 如果当前条件分支已经命中，则不需要再执行 else 语句
	if frame.CondContext.IsMatched {
		return false
	}

	frame.CondContext.MatchedValue = node
	frame.CondContext.IsMatched = true

	// 重置条件分支的情况
	frame.ResetBranches()
	slog.InfoContext(ctx, fmt.Sprintf("[parser.judgeConditionalElse](trace) condition evaluate result,\nkey = %s,\nvalue = %s", frame.FieldName, node.String()))
	return true
}

// executeLoop 处理for循环
func (p *Parser) executeLoop(ctx context.Context, node *model.TNode, statement string, frame *model.ParseFrame) {
	// 如果循环上下文不存在，则是第一次执行循环，进行初始化
	if frame.LoopContext == nil {
		err := p.initLoopContext(ctx, statement, frame)
		if err != nil {
			p.err = err
			return
		}
	}

	var (
		loopCtx  = frame.LoopContext
		loopMeta = loopCtx.Meta
		result   []any
	)
	switch loopCtx.Type {
	case model.LoopTypeForWithArray, model.LoopTypeForWithSlice:
		for ; loopCtx.Index < loopCtx.Length; loopCtx.Index++ {
			item := loopCtx.ObjectRefl.Index(loopCtx.Index).Interface()

			var (
				originKey, originValue any
			)
			if loopMeta.Key != "" {
				originKey = p.dataset[loopMeta.Key]
				p.dataset[loopMeta.Key] = loopCtx.Index
			}

			if loopMeta.Value != "" {
				originValue = p.dataset[loopMeta.Value]
				p.dataset[loopMeta.Value] = item
			}

			result = append(result, p.iterativeParse(ctx, node, string(keywords.KeywordFor)))

			// 处理完当前循环，恢复局部变量
			if loopMeta.Key != "" {
				p.dataset[loopMeta.Key] = originKey
			}
			if loopMeta.Value != "" {
				p.dataset[loopMeta.Value] = originValue
			}
		}
	case model.LoopTypeForWithMap:
		for loopCtx.MapIter.Next() {
			key := loopCtx.MapIter.Key().Interface()
			value := loopCtx.MapIter.Value().Interface()

			var (
				originKey, originValue any
			)
			if loopMeta.Key != "" {
				originKey = p.dataset[loopMeta.Key]
				p.dataset[loopMeta.Key] = key
			}
			if loopMeta.Value != "" {
				originValue = p.dataset[loopMeta.Value]
				p.dataset[loopMeta.Value] = value
			}

			result = append(result, p.iterativeParse(ctx, node, string(keywords.KeywordFor)))

			// 处理完当前循环，恢复局部变量
			if loopMeta.Key != "" {
				p.dataset[loopMeta.Key] = originKey
			}
			if loopMeta.Value != "" {
				p.dataset[loopMeta.Value] = originValue
			}
		}
	default:
		slog.ErrorContext(ctx, "[parser.executeLoop] loop object is not a supported rangeable type", "statement", statement)
		return
	}

	frame.Result = result
}

// initLoopContext 初始化循环上下文
func (p *Parser) initLoopContext(ctx context.Context, statement string, frame *model.ParseFrame) error {
	frame.LoopContext = model.NewLoopContext()
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
		objectRefl = reflect.ValueOf(object)
		loopType   model.LoopType
		mapIter    *reflect.MapIter
		length     int
	)

	switch objectRefl.Kind() {
	case reflect.Slice:
		loopType = model.LoopTypeForWithSlice
		length = objectRefl.Len()
	case reflect.Array:
		loopType = model.LoopTypeForWithArray
		length = objectRefl.Len()
	case reflect.Map:
		loopType = model.LoopTypeForWithMap
		mapIter = objectRefl.MapRange()
	default:
		slog.ErrorContext(ctx, "[parser.initLoopContext] the loop object is not rangeable", "object", object)
		return werror.ErrLoopObjectNotRangeable
	}

	frame.LoopContext.Meta = loopMeta
	frame.LoopContext.Object = object
	frame.LoopContext.ObjectRefl = ptr.Of(objectRefl)
	frame.LoopContext.Type = loopType
	frame.LoopContext.MapIter = mapIter
	frame.LoopContext.Length = length
	frame.LoopContext.IsSerialFor = keywords.IsForStatement(frame.FieldName)
	return nil
}

// continueLoop 处理循环中的CONTINUE
func (p *Parser) continueLoop(ctx context.Context, node *model.TNode, frame *model.ParseFrame) {

}
