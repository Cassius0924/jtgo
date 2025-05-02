package parser

import (
	"context"

	"github.com/cassius0924/jtgo/engine/model"
	"github.com/tidwall/gjson"
)

// CmtProcessor 处理 @cmt 注释关键字
type CmtProcessor struct {
	BaseProcessor
}

// Process 实现注释处理逻辑
func (p *CmtProcessor) Process(ctx context.Context, node *gjson.Result, statement string, frame *model.ParseFrame, parser *Parser) bool {
	// 注释不做任何处理，直接跳过
	return false
}
