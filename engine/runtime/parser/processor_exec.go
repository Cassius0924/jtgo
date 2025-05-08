package parser

import (
	"context"

	"github.com/cassius0924/jtgo/engine/model"
)

// ExecProcessor 处理 @do 关键字
type ExecProcessor struct {
	BaseProcessor
}

// Process 执行操作处理逻辑
func (p *ExecProcessor) Process(ctx context.Context, node *model.TNode, statement string, frame *model.ParseFrame, parser *Parser) bool {
	return parser.executeOperations(ctx, node, frame)
}
