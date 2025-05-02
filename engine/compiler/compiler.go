package compiler

import (
	"context"
	"log/slog"

	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/engine/common"
	"github.com/cassius0924/jtgo/engine/exprs"
	"github.com/cassius0924/jtgo/engine/keywords"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/cassius0924/jtgo/werror"
	"github.com/expr-lang/expr/vm"
	"github.com/liyue201/gostl/ds/deque"
	"github.com/tidwall/gjson"
)

type Compiler struct {
	exprHandler  *exprs.ExprHandler
	compiledExps map[string]*vm.Program
	loopMetas    map[string]*model.LoopMeta
}

type CompileFrame struct {
	Node        *gjson.Result
	SubNodeIter *deque.DequeIterator[*ds.Pair[gjson.Result, gjson.Result]]
}

func NewCompiler(exprHandler *exprs.ExprHandler) *Compiler {
	return &Compiler{
		exprHandler:  exprHandler,
		compiledExps: make(map[string]*vm.Program),
		loopMetas:    make(map[string]*model.LoopMeta),
	}
}

func (c *Compiler) Compile(ctx context.Context, template string) error {
	templateNode := gjson.Parse(template)
	if !templateNode.Exists() {
		slog.ErrorContext(ctx, "[JSONTemplateEngine.preCompileExpressions] pre compile configJSON error, please check whether the config JSON is valid!")
		return werror.ErrTemplateIsEmpty
	}

	// 迭代编译模板
	err := c.iterativeCompile(ctx, &templateNode)
	if err != nil {
		slog.ErrorContext(ctx, "[JSONTemplateEngine.iterativeCompile] pre compile configJSON error", "error", err)
		return err
	}

	return nil
}

// iterativeCompile 迭代编译模板
func (c *Compiler) iterativeCompile(ctx context.Context, templateNode *gjson.Result) error {
	var frameStack = ds.NewStack[*CompileFrame]()

	// 初始化栈，将根节点压入栈中
	frameStack.Push(&CompileFrame{
		Node:        templateNode,
		SubNodeIter: flattenNode(templateNode).First(),
	})

	// 迭代处理，直到栈为空
	for frameStack.Size() > 0 {
		var frame = frameStack.Top()

		// 如果当前帧的子节点已经遍历完毕，弹出栈顶帧
		if !frame.SubNodeIter.IsValid() {
			frameStack.Pop()
			continue
		}

		// 取出当前子节点，并将迭代器指向下一个元素
		subNodePair := frame.SubNodeIter.Value()
		field, object := subNodePair.First, subNodePair.Second
		frame.SubNodeIter.Next()

		if !object.Exists() {
			continue
		}

		fieldName := common.NormalizeFieldName(field.String())
		keyword, statement := keywords.DetectKeyword(fieldName)

		// 这里预编译，是表达式并且未被编译过
		switch keyword {
		case keywords.Keyword(""), keywords.KeywordCmt:
			// 不是表达式，跳过
		case keywords.KeywordFor:
			if statement != "" && c.loopMetas[statement] == nil {
				// 从statement中抽取出循环的变量名
				loopMeta, ok := exprs.ParseLoopStatement(ctx, statement)
				if !ok {
					slog.ErrorContext(ctx, "[JSONTemplateEngine.iterativePreCompile] parse loop statement error", "field", fieldName, "expression", statement)
					break
				}
				// 储存循环元数据
				c.loopMetas[statement] = loopMeta

				// 编译迭代对象
				if c.compiledExps[loopMeta.Object] == nil {
					program, err := c.exprHandler.Compile(ctx, loopMeta.Object)
					if err != nil {
						slog.ErrorContext(ctx, "[JSONTemplateEngine.iterativePreCompile] expr.Compile err", "field", fieldName, "expression", statement, "error", err)
						break
					}
					c.compiledExps[loopMeta.Object] = program
				}
			}
		default:
			if statement != "" && c.compiledExps[statement] == nil {
				// 编译表达式，然后缓存进compiledExps
				program, err := c.exprHandler.CompileAsBool(ctx, statement)
				if err != nil { // 说明expression不是以bool为最终值的表达式
					slog.ErrorContext(ctx, "[JSONTemplateEngine.iterativePreCompile] expr.Compile err", "field", fieldName, "expression", statement, "error", err)
					break
				}
				c.compiledExps[statement] = program
			}
		}

		switch {
		case object.IsObject():
			// 如果是对象类型，将其压入栈中继续处理
			frameStack.Push(&CompileFrame{
				Node:        &object,
				SubNodeIter: flattenNode(&object).First(),
			})
		case object.IsArray():
			// 如果是数组类型，遍历数组中的每个元素并压入栈中
			object.ForEach(func(_, value gjson.Result) bool {
				if value.IsObject() || value.IsArray() {
					frameStack.Push(&CompileFrame{
						Node:        &value,
						SubNodeIter: flattenNode(&value).First(),
					})
				} else if !value.IsBool() && value.Type != gjson.Number && value.Type != gjson.Null {
					// 处理字符串类型的值
					c.exprHandler.CompileStringExpressions(ctx, value.String(), c.compiledExps)
				}
				return true
			})
		case object.IsBool() || object.Type == gjson.Number || object.Type == gjson.Null:
			// 基本类型，不需要处理
		default: // 字符串
			// 编译字符串中的表达式
			c.exprHandler.CompileStringExpressions(ctx, object.String(), c.compiledExps)
		}
	}

	return nil
}

func (c *Compiler) GetCompiledExps() map[string]*vm.Program {
	return c.compiledExps
}

func (c *Compiler) GetLoopMetas() map[string]*model.LoopMeta {
	return c.loopMetas
}

// TODO: 和parser合并成一个
func flattenNode(node *gjson.Result) *deque.Deque[*ds.Pair[gjson.Result, gjson.Result]] {
	var result = ds.NewDeque[*ds.Pair[gjson.Result, gjson.Result]]()
	node.ForEach(func(k, v gjson.Result) bool {
		result.PushBack(ds.MakePair(k, v))
		return true
	})
	return result
}
