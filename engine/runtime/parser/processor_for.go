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
	return parser.executeLoop(ctx, node, statement, frame)
}
