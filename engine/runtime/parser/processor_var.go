package parser

import (
	"context"

	"github.com/cassius0924/jtgo/engine/model"
	"github.com/tidwall/gjson"
)

// VarProcessor 处理 @var 关键字
type VarProcessor struct {
	BaseProcessor
}

// Process 实现变量处理逻辑
func (p *VarProcessor) Process(ctx context.Context, node *gjson.Result, statement string, frame *model.ParseFrame, parser *Parser) bool {
	parser.varAssignment(ctx, node)
	// 变量赋值后不需要继续处理
	return false
}