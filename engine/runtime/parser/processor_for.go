package parser

import (
	"context"

	"github.com/cassius0924/jtgo/engine/model"
)

// ForProcessor 处理 @for 关键字
type ForProcessor struct {
	BaseProcessor
}

// Process 实现 for 处理逻辑
func (p *ForProcessor) Process(ctx context.Context, node *model.TNode, statement string, frame *model.ParseFrame, parser *Parser) bool {
	parser.executeLoop(ctx, node, statement, frame)
	// for 循环处理完后不需要继续处理该节点
	return false
}

// ContinueProcessor 处理 @continue 关键字
type ContinueProcessor struct {
	BaseProcessor
}

// Process 实现 continue 处理逻辑
func (p *ContinueProcessor) Process(ctx context.Context, node *model.TNode, statement string, frame *model.ParseFrame, parser *Parser) bool {
	parser.continueLoop(ctx, node, frame)
	// continue 关键字处理后不需要继续处理
	return false
}
