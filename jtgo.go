package jtgo

import (
	"context"
	"log/slog"
	"os"

	"github.com/cassius0924/jtgo/engine/core"
)

func init() {
	// 初始化日志
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     slog.LevelDebug,
		AddSource: true,
	}))

	slog.SetDefault(logger)
}

func GetJSONTemplateEngine(ctx context.Context, sceneKey, newConfigJSON string) (*core.JTEngine, error) {
	return core.GetJSONTemplateEngine(ctx, sceneKey, newConfigJSON)
}
