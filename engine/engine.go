package engine

import (
	"context"
	"log/slog"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/cassius0924/jtgo/werror"
	"github.com/expr-lang/expr/vm"
	"github.com/samber/lo"
)

type CtxKey string

const (
	JSONTemplateEngineCtxKey CtxKey = "_JSONTemplateEngine_"
)

const (
	defaultEntry = "_main_" // 默认入口标志

	JSONTemplateEngineDatasetVariable = "DATASET" // 引擎数据集变量名
	exprEnvVariable                   = "$env"    // expr Env变量名
)

var (
	templateIDToTemplate     sync.Map // 原始WCC配置，用于感知配置是否更新
	templateIDToCompiledExps sync.Map // 缓存编译过的表达式集合
	templateIDToCustomFuncs  sync.Map // 自定义函数集合

	builtInFuncCollection = make(map[string]any) // 引擎内置函数集合
)

type JTEngine struct {
	ctx               context.Context
	templateID        string
	metricsTags       map[string]string
	template          string
	entry             string
	err               error
	dataset           map[string]any
	target            any
	compiledExps      map[string]*vm.Program // 缓存编译过的表达式
	fns               map[string]any         // 函数集合
	localVariables    map[string]any         // 局部变量名称和值
	exprTraces        map[string]any         // 详细trace
	matchedExprTraces map[string]any         // 条件语句分支trace
}

func (e *JTEngine) ExprTraces() map[string]any {
	return e.exprTraces
}

func (e *JTEngine) MatchedExprTraces() map[string]any {
	return e.matchedExprTraces
}

// WithDataset 设置数据集
func (e *JTEngine) WithDataset(dataset map[string]any) *JTEngine {
	e.dataset = dataset
	return e
}

// WithCustomEntry 指定入口字段名
func (e *JTEngine) WithCustomEntry(entry string) *JTEngine {
	e.entry = entry
	return e
}

// ParseTo 设置解析结果的目标结构体
func (e *JTEngine) ParseTo(target any) *JTEngine {
	e.target = target
	return e
}

// WithMetricsTags 设置Metrics Tags，用于上报 Run 函数耗时
func (e *JTEngine) WithMetricsTags(metricsTags map[string]string) *JTEngine {
	e.metricsTags = metricsTags
	return e
}

// GetDataset 获取数据集
func (e *JTEngine) GetDataset() map[string]any {
	return e.dataset
}

func (e *JTEngine) clear() {
	e.entry = defaultEntry
	e.dataset = nil
	e.target = nil
	e.err = nil
}

func GetJSONTemplateEngine(ctx context.Context, templateID, template string) (*JTEngine, error) {
	if templateID == "" {
		slog.ErrorContext(ctx, "[JSONTemplateEngine.GetJSONTemplateEngine] templateID is empty", "templateID", templateID)
		return nil, werror.ErrTemplateIDIsEmpty
	}
	if template == "" {
		slog.ErrorContext(ctx, "[JSONTemplateEngine.GetJSONTemplateEngine] template is empty", "template", template)
		return nil, werror.ErrTemplateIsEmpty
	}

	isTemplateValidJSON := sonic.ValidString(template)
	if !isTemplateValidJSON {
		slog.ErrorContext(ctx, "[JSONTemplateEngine.GetJSONTemplateEngine] template is not a valid json, please check template JSON!", "templateID", templateID, "template", template)
	}

	cachedTemplate, _ := templateIDToTemplate.Load(templateID)

	if template == cachedTemplate || !isTemplateValidJSON { // 模板无更新 或 模板格式不合法 则使用缓存
		if cachedTemplate == nil {
			slog.ErrorContext(ctx, "[JSONTemplateEngine.GetJSONTemplateEngine] template is empty", "templateID", templateID)
			return nil, werror.ErrTemplateIsEmpty
		}

		cachedCompiledExps, _ := templateIDToCompiledExps.LoadOrStore(templateID, make(map[string]*vm.Program))
		cachedCustomFuncs, _ := templateIDToCustomFuncs.LoadOrStore(templateID, make(map[string]any))

		// 使用缓存的编译过的表达式
		return &JTEngine{
			ctx:               ctx,
			templateID:        templateID,
			entry:             defaultEntry,
			template:          cachedTemplate.(string),                     // 使用原配置
			compiledExps:      cachedCompiledExps.(map[string]*vm.Program), // 使用原缓存编译过的表达式
			fns:               lo.Assign(builtInFuncCollection, cachedCustomFuncs.(map[string]any)),
			localVariables:    make(map[string]any),
			exprTraces:        make(map[string]any),
			matchedExprTraces: make(map[string]any),
		}, nil
	}

	// 如果模板更新了，则重新编译表达式
	return createJSONTemplateEngine(ctx, templateID, template)
}

// createJSONTemplateEngine 创建模板引擎
func createJSONTemplateEngine(ctx context.Context, templateID, template string) (*JTEngine, error) {
	customFns, _ := templateIDToCustomFuncs.LoadOrStore(templateID, make(map[string]any))

	slog.InfoContext(ctx, "[JSONTemplateEngine.GetJSONTemplateEngine] template is updated, running preCompileExpressions", "templateID", templateID, "template", template, "customFunction count", len(customFns.(map[string]any)))

	engine := &JTEngine{
		ctx:               ctx,
		templateID:        templateID,
		entry:             defaultEntry,
		template:          template,
		compiledExps:      make(map[string]*vm.Program),
		fns:               lo.Assign(builtInFuncCollection, customFns.(map[string]any)), // 合并内置函数和自定义函数
		localVariables:    make(map[string]any),
		exprTraces:        make(map[string]any),
		matchedExprTraces: make(map[string]any),
	}

	err := engine.preCompileExpressions(template)
	if err != nil {
		return nil, err
	}

	templateIDToTemplate.Store(templateID, template)
	templateIDToCompiledExps.Store(templateID, engine.compiledExps)

	slog.InfoContext(ctx, "[JSONTemplateEngine.GetJSONTemplateEngine] running preCompileExpressions finish")
	return engine, nil
}

// GetJSONTemplateEngineFromContext 从context中获取JSONTemplateEngine，用于嵌套解析
func GetJSONTemplateEngineFromContext(ctx context.Context) *JTEngine {
	return ctx.Value(JSONTemplateEngineCtxKey).(*JTEngine)
}
