package parser

import (
	"context"

	"github.com/cassius0924/jtgo/engine/model"
	"github.com/tidwall/gjson"
)

// ExecProcessor 处理 @do 关键字
type ExecProcessor struct {
	BaseProcessor
}

// Process 执行操作处理逻辑
func (p *ExecProcessor) Process(ctx context.Context, node *gjson.Result, statement string, frame *model.ParseFrame, parser *Parser) bool {
	parser.execOperations(ctx, node)
	// 执行操作后不需要继续处理
	return false
}
