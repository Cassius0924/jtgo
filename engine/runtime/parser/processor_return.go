package parser

import (
	"context"

	"github.com/cassius0924/jtgo/engine/model"
)

// ReturnProcessor 处理 @return 注释关键字
type ReturnProcessor struct {
	BaseProcessor
}

// Process 实现返回处理逻辑
func (p *ReturnProcessor) Process(ctx context.Context, node *model.TNode, statement string, frame *model.ParseFrame, parser *Parser) bool {
	// 处理返回逻辑
	return true
}
