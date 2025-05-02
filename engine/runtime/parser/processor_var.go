package parser

import (
	"context"
	"log/slog"

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

// DoProcessor 处理 @do 关键字
type DoProcessor struct {
	BaseProcessor
}

// Process 实现操作处理逻辑
func (p *DoProcessor) Process(ctx context.Context, node *gjson.Result, statement string, frame *model.ParseFrame, parser *Parser) bool {
	slog.InfoContext(ctx, "[JSONTemplateEngine.Run] do statement", "statement", statement)
	parser.doOperations(ctx, node)
	// 执行操作后不需要继续处理
	return false
}

// CommentProcessor 处理 @cmt 注释关键字
type CommentProcessor struct {
	BaseProcessor
}

// Process 实现注释处理逻辑
func (p *CommentProcessor) Process(ctx context.Context, node *gjson.Result, statement string, frame *model.ParseFrame, parser *Parser) bool {
	// 注释不做任何处理，直接跳过
	return false
}
