package engine

import (
	"github.com/tidwall/gjson"
)

// KeywordProcessor 关键字处理器接口
// 所有模板引擎关键字处理逻辑都需要实现此接口
type KeywordProcessor interface {
	// Process 处理关键字逻辑
	// node: 关键字对应的节点
	// statement: 关键字后的语句
	// frame: 当前解析帧
	// engine: 模板引擎实例
	// 返回值指示是否需要继续处理后续节点
	Process(node *gjson.Result, statement string, frame *ParseFrame, engine *JTEngine) bool
}

// BaseProcessor 处理器基础结构
type BaseProcessor struct{}

// MatchKeyword 检查是否匹配某个关键字
func (b *BaseProcessor) MatchKeyword(fieldName string, keyword Keyword) bool {
	return isKeyword(fieldName, keyword)
}
