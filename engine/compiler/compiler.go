package compiler

import (
	"context"
	"log/slog"

	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/engine/common"
	"github.com/cassius0924/jtgo/engine/exprs"
	"github.com/cassius0924/jtgo/engine/flags"
	"github.com/cassius0924/jtgo/engine/keywords"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/cassius0924/jtgo/werror"
	"github.com/expr-lang/expr/vm"
	"github.com/liyue201/gostl/ds/stack"
	"github.com/liyue201/gostl/ds/vector"
	"github.com/tidwall/gjson"
)

// Compiler 模板编译器
type Compiler struct {
	exprHandler  *exprs.ExprHandler
	compiledExps map[string]*vm.Program
	loopMetas    map[string]*model.LoopMeta
	subNodeIters map[string]*model.NodeIter
}

func NewCompiler(exprHandler *exprs.ExprHandler) *Compiler {
	return &Compiler{
		exprHandler:  exprHandler,
		compiledExps: make(map[string]*vm.Program),
		loopMetas:    make(map[string]*model.LoopMeta),
		subNodeIters: make(map[string]*model.NodeIter),
	}
}

func (c *Compiler) Compile(ctx context.Context, template string) error {
	templateNode := gjson.Parse(template)
	if !templateNode.Exists() {
		slog.ErrorContext(ctx, "[compiler.Compile] template is empty")
		return werror.ErrTemplateIsEmpty
	}

	// 迭代编译模板
	err := c.iterativeCompile(ctx, model.NewTNode(templateNode, nil))
	if err != nil {
		slog.ErrorContext(ctx, "[compiler.Compile] compile error", "error", err)
		return err
	}

	return nil
}

// iterativeCompile 迭代编译模板
func (c *Compiler) iterativeCompile(ctx context.Context, templateNode *model.TNode) error {
	var frameStack = ds.NewStackWithListContainer[*model.CompileFrame]()

	// 初始化栈，将根节点压入栈中
	templateNode.ForEach(func(subNodeField, subNode gjson.Result) bool {
		subNodeFieldName := common.NormalizeFieldName(subNodeField.String())
		c.PushFrameToStack(frameStack, model.NewTNode(subNode, nil), subNodeFieldName)
		return true
	})

	// 迭代处理，直到栈为空
	for frameStack.Size() > 0 {
		frame := frameStack.Top()

		// 如果当前帧的子节点已经遍历完毕，弹出栈顶帧
		if !frame.SubNodeIter.IsValid() {
			frameStack.Pop()
			continue
		}

		// 取出当前子节点，并将迭代器指向下一个元素
		subNodeField, subNode := frame.ExtractSubNodePair()
		frame.SubNodeIter.Next()
		subNodeFieldName := common.NormalizeFieldName(subNodeField.String())

		keyword, statement := keywords.DetectKeyword(subNodeFieldName)
		setKeywordNodeFlag(subNode, keyword)
		subNode.Keyword = keyword
		subNode.Statement = statement

		// 这里预编译，是表达式并且未被编译过
		switch keyword {
		case keywords.Keyword(""), keywords.KeywordCmt:
			// 不是表达式，跳过
		case keywords.KeywordFor:
			if statement != "" && c.loopMetas[statement] == nil {
				// 从statement中抽取出循环的变量名
				loopMeta, ok := exprs.ParseLoopStatement(ctx, statement)
				if !ok {
					slog.ErrorContext(ctx, "[compiler.iterativeCompile] parse loop statement error", "field", subNodeFieldName, "expression", statement)
					break
				}
				// 储存循环元数据
				c.loopMetas[statement] = loopMeta

				// 编译迭代对象
				if c.compiledExps[loopMeta.Object] == nil {
					program, err := c.exprHandler.Compile(ctx, loopMeta.Object)
					if err != nil {
						slog.ErrorContext(ctx, "[compiler.iterativeCompile] expr.Compile err", "field", subNodeFieldName, "expression", statement, "error", err)
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
					slog.ErrorContext(ctx, "[compiler.iterativeCompile] expr.Compile err", "field", subNodeFieldName, "expression", statement, "error", err)
					break
				}
				c.compiledExps[statement] = program
			}
		}

		switch {
		case subNode.IsObject(), subNode.IsArray():
			// 如果是对象类型，将其压入栈中继续处理
			c.PushFrameToStack(frameStack, subNode, frame.BuildNodePath(subNodeFieldName))
		case subNode.IsBool() || subNode.Type == gjson.Number || subNode.Type == gjson.Null:
			// 基本类型，不需要处理
		default: // 字符串
			// 编译字符串中的表达式
			c.exprHandler.CompileStringExpressions(ctx, subNode.String(), c.compiledExps)
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

func (c *Compiler) GetSubNodeIters() map[string]*vector.VectorIterator[*ds.Pair[*model.TNode, *model.TNode]] {
	return c.subNodeIters
}

func (c *Compiler) PushFrameToStack(stack *stack.Stack[*model.CompileFrame], node *model.TNode, path string) {
	subNodeIter := common.FlattenNode(node).First()
	if path != "" {
		// 缓存一份，key 为路径，根节点不做缓存，因为解析时不会直接操作根节点
		c.subNodeIters[path] = subNodeIter
	}
	stack.Push(&model.CompileFrame{
		Node:        node,
		SubNodeIter: subNodeIter,
		Path:        path,
	})
}

func setKeywordNodeFlag(node *model.TNode, keyword keywords.Keyword) {
	switch keyword {
	case keywords.KeywordExec:
		node.NodeFlag.Set(flags.NodeFlagKeywordExec)
	case keywords.KeywordReturn:
		node.NodeFlag.Set(flags.NodeFlagKeywordReturn)
	case keywords.KeywordVar:
		node.NodeFlag.Set(flags.NodeFlagKeywordVar)
	case keywords.KeywordFor:
		node.NodeFlag.Set(flags.NodeFlagKeywordFor)
	case keywords.KeywordIf:
		node.NodeFlag.Set(flags.NodeFlagKeywordIf)
	case keywords.KeywordElif:
		node.NodeFlag.Set(flags.NodeFlagKeywordElif)
	case keywords.KeywordElse:
		node.NodeFlag.Set(flags.NodeFlagKeywordElse)
	case keywords.KeywordCmt:
		node.NodeFlag.Set(flags.NodeFlagKeywordCmt)
	default:
		if node.Parent != nil && node.Parent.IsArray() {
			node.NodeFlag.Set(flags.NodeFlagGeneralArrayItem)
		} else {
			node.NodeFlag.Set(flags.NodeFlagGeneralObjectItem)
		}
	}
}
