package engine

import (
	"github.com/tidwall/gjson"
)

// ForProcessor 处理 @for 关键字
type ForProcessor struct {
	BaseProcessor
}

// Process 实现 for 处理逻辑
func (p *ForProcessor) Process(node *gjson.Result, statement string, frame *ParseFrame, engine *JTEngine) bool {
	engine.executeLoop(node, statement, frame)
	// for 循环处理完后不需要继续处理该节点
	return false
}

// ContinueProcessor 处理 @continue 关键字
type ContinueProcessor struct {
	BaseProcessor
}

// Process 实现 continue 处理逻辑
func (p *ContinueProcessor) Process(node *gjson.Result, statement string, frame *ParseFrame, engine *JTEngine) bool {
	engine.continueLoop(node, frame)
	// continue 关键字处理后不需要继续处理
	return false
}
