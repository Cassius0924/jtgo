package parser

import (
	"context"

	"github.com/cassius0924/jtgo/engine/model"
)

// VarProcessor 处理 @var 关键字
type VarProcessor struct {
	BaseProcessor
}

// Process 实现变量处理逻辑
func (p *VarProcessor) Process(ctx context.Context, node *model.TNode, statement string, frame *model.ParseFrame, parser *Parser) bool {
	return parser.assignVariables(ctx, node, frame)
}
