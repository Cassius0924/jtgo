package engine

import (
	"github.com/tidwall/gjson"
)

// IfProcessor 处理 @if 关键字
type IfProcessor struct {
	BaseProcessor
}

// Process 实现 if 处理逻辑
func (p *IfProcessor) Process(node *gjson.Result, statement string, frame *ParseFrame, engine *JTEngine) bool {
	matched := engine.judgeConditionalIf(node, statement, frame)
	// 如果匹配成功，继续处理；否则跳过，寻找下一个条件
	return matched
}

// ElifProcessor 处理 @elif 关键字
type ElifProcessor struct {
	BaseProcessor
}

// Process 实现 elif 处理逻辑
func (p *ElifProcessor) Process(node *gjson.Result, statement string, frame *ParseFrame, engine *JTEngine) bool {
	matched := engine.judgeConditionalElif(node, statement, frame)
	// 如果匹配成功，继续处理；否则跳过，寻找下一个条件
	return matched
}

// ElseProcessor 处理 @else 关键字
type ElseProcessor struct {
	BaseProcessor
}

// Process 实现 else 处理逻辑
func (p *ElseProcessor) Process(node *gjson.Result, statement string, frame *ParseFrame, engine *JTEngine) bool {
	matched := engine.judgeConditionalElse(node, frame)
	// 如果匹配成功，继续处理；否则跳过
	return matched
}
