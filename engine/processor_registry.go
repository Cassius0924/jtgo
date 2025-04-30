package engine

import (
	"sync"
)

var (
	// processorRegistry 关键字处理器注册表
	processorRegistry     = make(map[Keyword]KeywordProcessor)
	processorRegistryOnce sync.Once
)

// initProcessorRegistry 初始化处理器注册表
func initProcessorRegistry() {
	processorRegistryOnce.Do(func() {
		// 注册条件处理器
		registerProcessor(KeywordIf, &IfProcessor{})
		registerProcessor(KeywordElif, &ElifProcessor{})
		registerProcessor(KeywordElse, &ElseProcessor{})

		// 注册循环处理器
		registerProcessor(KeywordFor, &ForProcessor{})
		registerProcessor(KeywordContinue, &ContinueProcessor{})

		// 注册变量和操作处理器
		registerProcessor(KeywordVar, &VarProcessor{})
		registerProcessor(KeywordDo, &DoProcessor{})
	})
}

// registerProcessor 注册处理器
func registerProcessor(keyword Keyword, processor KeywordProcessor) {
	processorRegistry[keyword] = processor
}

// GetProcessor 获取关键字处理器
func GetProcessor(keyword Keyword) KeywordProcessor {
	initProcessorRegistry()
	return processorRegistry[keyword]
}

// RegisterCustomProcessor 注册自定义处理器
// 用于扩展引擎功能，允许外部注册自定义处理器
func RegisterCustomProcessor(keyword Keyword, processor KeywordProcessor) {
	initProcessorRegistry()
	registerProcessor(keyword, processor)
}
