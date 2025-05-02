package parser

import (
	"context"
	"log/slog"

	"github.com/bytedance/sonic"
	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/engine/common"
	"github.com/cassius0924/jtgo/engine/exprs"
	"github.com/cassius0924/jtgo/engine/keywords"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/cassius0924/jtgo/util"
	"github.com/cassius0924/jtgo/werror"
	"github.com/expr-lang/expr/vm"
	"github.com/tidwall/gjson"
)

type Parser struct {
	exprHandler    *exprs.ExprHandler         // 表达式处理器
	compiledExps   map[string]*vm.Program     // 缓存编译过的表达式
	loopMetas      map[string]*model.LoopMeta // 循环语句元数据
	dataset        map[string]any             // 数据集
	localVariables map[string]any             // 局部变量名称和值

	err error
}

// NewParser 创建一个新的解析器实例
func NewParser(exprHandler *exprs.ExprHandler, compiledExps map[string]*vm.Program, loopMetas map[string]*model.LoopMeta) *Parser {
	return &Parser{
		exprHandler:    exprHandler,
		compiledExps:   compiledExps,
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
		slog.ErrorContext(ctx, "[JSONTemplateEngine.Run] entry not exists, please check whether entry name exists in the config JSON!", "entry", entry, "template", template)
		return "", werror.ErrEntryNotFound
	}

	// 循环解析方法
	result := p.interativeParse(ctx, &entryTemplateNode, entry)
	resultStr := util.SonicToString(result)

	if target != nil {
		// 将模板解析结果反序列化给target
		if err := sonic.UnmarshalString(resultStr, target); err != nil {
			slog.ErrorContext(ctx, "[JSONTemplateEngine.Run] UnmarshalFromString config error, please check if the template JSON field name matches the target structure field name!", "finalTarget", util.GenerateStructFormatedString(result), "error", err)
			return resultStr, werror.Join(werror.ErrParseToTargetFailed, err)
		}
	}

	return resultStr, p.err
}

// interativeParse 迭代解析方法
// 通过迭代方式解析JSON模板，将模板转换为最终输出结果
// templateFieldName: 模板节点的字段名
func (p *Parser) interativeParse(ctx context.Context, templateNode *gjson.Result, templateFieldName string) any {
	var (
		frameStack     = ds.NewStack[*model.ParseFrame]() // 解析帧栈，用于深度优先遍历JSON树
		result     any = make(map[string]any)             // 初始化结果为空map
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
	frameStack.Push(&model.ParseFrame{
		Node:         templateNode,
		FieldName:    templateFieldName,
		Target:       result,
		Result:       result,
		AssistResult: make(map[string]any, 1),                  // 辅助结果，用于存储临时数据
		SubNodeIter:  common.FlattenNode(templateNode).First(), // 子节点迭代器
		ConditionalCtx: &model.ConditionalContext{ // 条件上下文
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

		subNodeFieldName := common.NormalizeFieldName(subNodeField.String())
		frame.CurSubNodeFieldName = subNodeFieldName

		// 处理模板语法关键字
		keyword, statement := keywords.DetectKeyword(subNodeFieldName)
		if keyword != "" {
			// 获取关键字处理器并执行处理
			processor := GetProcessor(keyword)
			if processor != nil {
				// 如果处理器返回false，表示不需要继续处理当前节点
				continueProcess := processor.Process(ctx, &subNode, statement, frame, p)
				if !continueProcess {
					continue
				}
			}
		}

		// 根据节点类型进行不同处理
		switch {
		case subNode.IsObject():
			// 如果是对象类型，需要创建新的解析帧并压入栈中，继续深度遍历
			frameStack.Push(&model.ParseFrame{
				Node:         &subNode,
				FieldName:    subNodeFieldName,
				Target:       frame.Result,
				Result:       make(map[string]any),
				AssistResult: frame.AssistResult,
				SubNodeIter:  common.FlattenNode(&subNode).First(),
				ConditionalCtx: &model.ConditionalContext{
					IsMatched: false,
				},
			})
			continue
		default:
			// 处理基本类型节点和其他情况
			if frame.IsConditionalMatched() {
				// 条件语句匹配成功，使用条件匹配值
				frame.Result = p.replaceExpression(ctx, frame.ConditionalCtx.MatchedValue)
			} else if subNodeFieldName == "" {
				// 无字段名，直接设置结果
				frame.Result = p.replaceExpression(ctx, &subNode)
			} else {
				// 有字段名，设置结果的对应字段
				frame.Result.(map[string]any)[subNodeFieldName] = p.replaceExpression(ctx, &subNode)
			}
			continue
		}
	}

	return result
}

// replaceExpression 递归替换 object 中所有含有${Expression}的值
func (p *Parser) replaceExpression(ctx context.Context, input *gjson.Result) any {
	if input == nil || !input.Exists() {
		return nil
	}
	// JSON 共有 6 种类型：input, array, bool, number, null, string
	switch {
	case input.IsObject(): // input
		var (
			result          = make(map[string]any)
			resultForReturn any
			isExpression    bool
		)

		input.ForEach(func(field, node gjson.Result) bool {
			fieldName := common.NormalizeFieldName(field.String())

			switch keyword, _ := keywords.DetectKeyword(fieldName); keyword {
			case keywords.KeywordReturn:
				// 遇到 RETURN 关键词，直接返回
				resultForReturn = p.returnResult(ctx, &node)
				return false
			case keywords.KeywordDo:
				// 是 DO 关键词，需要执行操作
				p.doOperations(ctx, &node)
				return true
			case keywords.KeywordVar:
				// 是 VAR 关键词，需要进行变量赋值
				p.varAssignment(ctx, &node)
				return true
			default:
				// 其他情况，继续处理
			}
			_, isExpression = exprs.ExtractExpression(fieldName)
			if !isExpression {
				result[fieldName] = p.replaceExpression(ctx, &node)
				return true
			}

			return true
		})

		if resultForReturn != nil {
			return resultForReturn
		}
		return result
	case input.IsArray(): // array
		var result []any
		for _, value := range input.Array() {
			result = append(result, p.replaceExpression(ctx, &value))
		}
		return result
	case input.IsBool(): // bool
		return input.Bool()
	case input.Type == gjson.Number: // number
		return input.Num
	case input.Type == gjson.Null: // null
		return nil
	default: // string
		return p.exprHandler.EvaluateExpressionsInText(ctx, p.compiledExps, input.String()) // 使用数据集中的值替换text中的${Path.Var}
	}
}
