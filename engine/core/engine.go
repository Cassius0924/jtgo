package core

import (
	"context"
	"log/slog"
	"sync"

	"github.com/cassius0924/jtgo/engine/compiler"
	"github.com/cassius0924/jtgo/engine/exprs"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/cassius0924/jtgo/engine/runtime/parser"
	"github.com/cassius0924/jtgo/util"
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
)

var (
	templateIDToTemplate     sync.Map // 缓存模板节点
	templateIDToCompiledExps sync.Map // 缓存编译过的表达式集合
	templateIDToLoopMeta     sync.Map // 缓存循环元数据集合
	templateIDToCustomFuncs  sync.Map // 自定义函数集合

	builtInFns = make(map[string]any) // 引擎内置函数集合
)

type JTEngine struct {
	ctx          context.Context
	templateID   string
	template     string
	entry        string
	dataset      map[string]any
	target       any
	compiledExps map[string]*vm.Program     // 缓存编译过的表达式
	loopMetas    map[string]*model.LoopMeta // 循环语句元数据

	exprHandler *exprs.ExprHandler // 表达式处理器
	compiler    *compiler.Compiler // 模板编译器
	parser      *parser.Parser     // 模板解析器
}

// WithDataset 设置数据集
func (e *JTEngine) WithDataset(dataset map[string]any) *JTEngine {
	e.dataset = dataset
	e.exprHandler.SetDataset(dataset)
	e.parser.SetDataset(dataset)

	// subEngine := GetJSONTemplateEngineFromContext(e.ctx)
	// subEngine.dataset = dataset
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

// GetDataset 获取数据集
func (e *JTEngine) GetDataset() map[string]any {
	return e.dataset
}

func (e *JTEngine) clear() {
	e.entry = defaultEntry
	e.dataset = nil
	e.target = nil
}

func GetJSONTemplateEngine(ctx context.Context, templateID, template string) (*JTEngine, error) {
	if templateID == "" {
		slog.ErrorContext(ctx, "[core.GetJSONTemplateEngine] templateID is empty", "templateID", templateID)
		return nil, werror.ErrTemplateIDIsEmpty
	}
	if template == "" {
		slog.ErrorContext(ctx, "[core.GetJSONTemplateEngine] template is empty", "template", template)
		return nil, werror.ErrTemplateIsEmpty
	}

	var cachedTemplate string
	validateErr := util.ValidateJSON(template)
	if cachedTemplate, ok := templateIDToTemplate.Load(templateID); ok && cachedTemplate != nil {
		cachedTemplate = cachedTemplate.(string)
	}

	if validateErr != nil {
		slog.ErrorContext(ctx, "[core.GetJSONTemplateEngine] template is invalid JSON", "templateID", templateID, "template", template, "error", validateErr)
	}

	if equalTemplate(template, cachedTemplate) || validateErr != nil { // 模板无更新 或 模板格式不合法 则使用缓存
		if cachedTemplate == "" {
			if validateErr != nil {
				return nil, werror.Join(werror.ErrTemplateIsInvalidJSON, validateErr)
			}
			slog.ErrorContext(ctx, "[core.GetJSONTemplateEngine] template is empty", "templateID", templateID)
			return nil, werror.ErrTemplateIsEmpty
		}

		var (
			cachedCompiledExps map[string]*vm.Program
			cachedCustomFns  map[string]any
			cachedLoopMeta     map[string]*model.LoopMeta
		)
		if compiledExps, ok := templateIDToCompiledExps.LoadOrStore(templateID, make(map[string]*vm.Program)); ok {
			cachedCompiledExps = compiledExps.(map[string]*vm.Program)
		} else {
			slog.ErrorContext(ctx, "[core.GetJSONTemplateEngine] cached compiled expressions not found", "templateID", templateID)
			return nil, werror.ErrCachedCompiledExpressionsNotFound
		}
		if customFuncs, ok := templateIDToCustomFuncs.LoadOrStore(templateID, make(map[string]any)); ok {
			cachedCustomFns = customFuncs.(map[string]any)
		} else {
			slog.ErrorContext(ctx, "[core.GetJSONTemplateEngine] cached custom functions not found", "templateID", templateID)
			return nil, werror.ErrCachedCustomFunctionsNotFound
		}
		if loopMeta, ok := templateIDToLoopMeta.LoadOrStore(templateID, make(map[string]*model.LoopMeta)); ok {
			cachedLoopMeta = loopMeta.(map[string]*model.LoopMeta)
		} else {
			slog.ErrorContext(ctx, "[core.GetJSONTemplateEngine] cached loop meta not found", "templateID", templateID)
			return nil, werror.ErrCachedLoopMetaNotFound
		}

		fns := lo.Assign(builtInFns, cachedCustomFns)
		exprHandler := exprs.NewExprHandler(templateID, template, fns)

		// 使用缓存的编译过的表达式
		return &JTEngine{
			ctx:          ctx,
			templateID:   templateID,
			entry:        defaultEntry,
			template:     template,
			compiledExps: cachedCompiledExps, // 使用原缓存编译过的表达式
			loopMetas:    cachedLoopMeta,     // 使用原缓存循环元数据

			exprHandler: exprHandler,
			parser:      parser.NewParser(exprHandler, cachedCompiledExps, cachedLoopMeta),
		}, nil
	}

	// 如果模板更新了，则重新编译表达式
	return createJSONTemplateEngine(ctx, templateID, template)
}

// createJSONTemplateEngine 创建模板引擎
func createJSONTemplateEngine(ctx context.Context, templateID, template string) (*JTEngine, error) {
	var (
		customFns, _ = templateIDToCustomFuncs.LoadOrStore(templateID, make(map[string]any))
		fns          = lo.Assign(builtInFns, customFns.(map[string]any)) // 合并内置函数和自定义函数
		exprHandler  = exprs.NewExprHandler(templateID, template, fns)
	)

	engine := &JTEngine{
		ctx:        ctx,
		templateID: templateID,
		entry:      defaultEntry,
		template:   template,

		exprHandler: exprHandler,
		compiler:    compiler.NewCompiler(exprHandler),
	}

	slog.InfoContext(ctx, "[core.GetJSONTemplateEngine] template is updated, running iterativePreCompile", "templateID", templateID, "template", template, "custom function count", len(customFns.(map[string]any)))
	err := engine.compiler.Compile(ctx, template)
	if err != nil {
		return nil, err
	}

	// 迭代编译完成后，获取编译过的表达式和循环元数据
	engine.compiledExps = engine.compiler.GetCompiledExps()
	engine.loopMetas = engine.compiler.GetLoopMetas()
	exprHandler.SetCompiledExps(engine.compiledExps)
	// 创建模板解析器
	engine.parser = parser.NewParser(exprHandler, engine.compiledExps, engine.loopMetas)

	// 缓存
	templateIDToTemplate.Store(templateID, template)
	templateIDToCompiledExps.Store(templateID, engine.compiledExps)
	templateIDToLoopMeta.Store(templateID, engine.loopMetas)

	slog.InfoContext(ctx, "[core.GetJSONTemplateEngine] create JSON template engine success", "templateID", templateID)
	return engine, nil
}

// GetJSONTemplateEngineFromContext 从context中获取JSONTemplateEngine，用于嵌套解析
func GetJSONTemplateEngineFromContext(ctx context.Context) *JTEngine {
	return ctx.Value(JSONTemplateEngineCtxKey).(*JTEngine)
}

// Run 运行引擎，并且清空引擎状态
func (e *JTEngine) Run() (string, error) {
	slog.InfoContext(e.ctx, "[core.Run](trace) Run function start")
	defer func() {
		// 清空调用链
		e.clear()
		slog.InfoContext(e.ctx, "[core.Run](trace) Run function end")
	}()
	return e.keepStatusRun()
}

// checkBeforeRun 检查是否有必要的参数
func (e *JTEngine) checkBeforeRun() error {
	if e.template == "" {
		slog.ErrorContext(e.ctx, "[core.checkBeforeRun] configJSON is empty")
		return werror.ErrTemplateIsEmpty
	}
	return nil
}

// keepStatusRun 保持状态运行
func (e *JTEngine) keepStatusRun() (string, error) {
	if err := e.checkBeforeRun(); err != nil {
		return "", err
	}

	defer func() {
		// 恢复局部变量
		// for k, v := range e.localVariables {
		// 	e.dataset[k] = v
		// 	slog.InfoContext(e.ctx, fmt.Sprintf("[core.keepStatusRun] restore variable,\nkey = %s,\nvalue = %v", k, v))
		// }
		// e.localVariables = make(map[string]any)
	}()

	return e.parser.Parse(e.ctx, e.template, e.entry, e.target)
}

func equalTemplate(templateA, templateB string) bool {
	return templateA == templateB
}
