package parser

import (
	"context"
	"sync"

	"github.com/cassius0924/jtgo/engine/keywords"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/tidwall/gjson"
)

// KeywordProcessor 关键字处理器接口
// 所有模板引擎关键字处理逻辑都需要实现此接口
type KeywordProcessor interface {
	// Process 处理关键字逻辑
	// node: 关键字对应的节点
	// statement: 关键字后的语句
	// frame: 当前解析帧
	// parser: 模板解析器
	// 返回值指示是否需要继续处理后续节点
	Process(ctx context.Context, node *gjson.Result, statement string, frame *model.ParseFrame, parser *Parser) bool
}

// BaseProcessor 处理器基础结构
type BaseProcessor struct{}

var (
	// procRegistry 关键字处理器注册表
	procRegistry     = make(map[keywords.Keyword]KeywordProcessor)
	procRegistryOnce sync.Once
)

// initProcessorRegistry 初始化处理器注册表
func initProcessorRegistry() {
	procRegistryOnce.Do(func() {
		// 注册条件处理器
		registerProcessor(keywords.KeywordIf, &IfProcessor{})
		registerProcessor(keywords.KeywordElif, &ElifProcessor{})
		registerProcessor(keywords.KeywordElse, &ElseProcessor{})

		// 注册循环处理器
		registerProcessor(keywords.KeywordFor, &ForProcessor{})
		registerProcessor(keywords.KeywordContinue, &ContinueProcessor{})

		// 注册变量处理器
		registerProcessor(keywords.KeywordVar, &VarProcessor{})

		// 注册操作处理器
		registerProcessor(keywords.KeywordExec, &ExecProcessor{})

		// 注册注释处理器
		registerProcessor(keywords.KeywordComment, &CommentProcessor{})
	})
}

// registerProcessor 注册处理器
func registerProcessor(keyword keywords.Keyword, processor KeywordProcessor) {
	procRegistry[keyword] = processor
}

// GetProcessor 获取关键字处理器
func GetProcessor(keyword keywords.Keyword) KeywordProcessor {
	initProcessorRegistry()
	return procRegistry[keyword]
}
