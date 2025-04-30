package jtgo

import (
	"context"
	"log/slog"
	"os"

	"github.com/cassius0924/jtgo/engine"
)

func init() {
	// 初始化日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))

	slog.SetDefault(logger)
}

func GetJSONTemplateEngine(ctx context.Context, sceneKey, newConfigJSON string) (*engine.JTEngine, error) {
	return engine.GetJSONTemplateEngine(ctx, sceneKey, newConfigJSON)
}

// Agent，中文译为智能体。在 Agent 模式下，Copilot 就相当于一个你的高级私人助手，可以帮助你完成一些复杂的任务，它能够自动分解任务、调用 MCP 工具、编写代码、测试并修复代码等等。
