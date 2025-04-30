package engine

import (
	"log/slog"

	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/werror"
	"github.com/tidwall/gjson"

)

func (e *JTEngine) preCompileExpressions(configJSON string) error {
	configResult := gjson.Parse(configJSON)
	if !configResult.Exists() {
		slog.ErrorContext(e.ctx, "[JSONTemplateEngine.preCompileExpressions] pre compile configJSON error, please check whether the config JSON is valid!")
		return werror.ErrPreCompileFail
	}

	e.iterativePreCompile(configResult)
	return nil
}

// iterativePreCompile 迭代预编译方法
// 通过迭代方式预编译JSON模板中的表达式，替代原来的递归方法
func (e *JTEngine) iterativePreCompile(configResult gjson.Result) {
	var frameStack = ds.NewStack[*PreCompileFrame]()

	// 初始化栈，将根节点压入栈中
	frameStack.Push(&PreCompileFrame{
		Node:        configResult,
		SubNodeIter: flattenNodeForPreCompile(configResult).First(),
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

		fieldName := normalizeFieldName(field.String())
		keyword, statement := DetectKeyword(fieldName)

		// 这里预编译，是表达式并且未被编译过
		switch keyword {
		case Keyword(""), KeywordComment:
			// 不是表达式，跳过
		case KeywordFor:
			if statement != "" && e.loopMetas[statement] == nil {
				// 从statement中抽取出循环的变量名
				loopMeta, ok := parseLoopStatement(e.ctx, statement)
				if !ok {
					slog.ErrorContext(e.ctx, "[JSONTemplateEngine.iterativePreCompile] parse loop statement error", "field", fieldName, "expression", statement)
					break
				}
				// 储存循环元数据
				e.loopMetas[statement] = loopMeta

				// 编译迭代对象
				if e.compiledExps[loopMeta.Object] == nil {
					program, err := e.exprCompile(loopMeta.Object)
					if err != nil {
						slog.ErrorContext(e.ctx, "[JSONTemplateEngine.iterativePreCompile] expr.Compile err", "field", fieldName, "expression", statement, "error", err)
						break
					}
					e.compiledExps[loopMeta.Object] = program
				}
			}
		default:
			if statement != "" && e.compiledExps[statement] == nil {
				// 编译表达式，然后缓存进compiledExps
				program, err := e.exprCompileAsBool(statement)
				if err != nil { // 说明expression不是以bool为最终值的表达式
					slog.ErrorContext(e.ctx, "[JSONTemplateEngine.iterativePreCompile] expr.Compile err", "field", fieldName, "expression", statement, "error", err)
					break
				}
				e.compiledExps[statement] = program
			}
		}

		switch {
		case object.IsObject():
			// 如果是对象类型，将其压入栈中继续处理
			frameStack.Push(&PreCompileFrame{
				Node:        object,
				SubNodeIter: flattenNodeForPreCompile(object).First(),
			})
		case object.IsArray():
			// 如果是数组类型，遍历数组中的每个元素并压入栈中
			object.ForEach(func(_, value gjson.Result) bool {
				if value.IsObject() || value.IsArray() {
					frameStack.Push(&PreCompileFrame{
						Node:        value,
						SubNodeIter: flattenNodeForPreCompile(value).First(),
					})
				} else if !value.IsBool() && value.Type != gjson.Number && value.Type != gjson.Null {
					// 处理字符串类型的值
					e.compileStringExpressions(value, fieldName)
				}
				return true
			})
		case object.IsBool() || object.Type == gjson.Number || object.Type == gjson.Null:
			// 基本类型，不需要处理
		default: // 字符串
			// 编译字符串中的表达式
			e.compileStringExpressions(object, fieldName)
		}
	}
}

// compileStringExpressions 编译字符串中的表达式
func (e *JTEngine) compileStringExpressions(object gjson.Result, fieldName string) {
	exps := extractAllExpression(object.String())
	// 编译每一个表达式
	for _, exp := range exps {
		if e.compiledExps[exp] != nil { // 已经编译过跳过
			continue
		}
		program, err := e.exprCompile(exp)
		if err != nil {
			slog.ErrorContext(e.ctx, "[JSONTemplateEngine.compileStringExpressions] expr.Compile err", "field", fieldName, "expression", exp, "error", err)
		} else {
			e.compiledExps[exp] = program
		}
	}
}
