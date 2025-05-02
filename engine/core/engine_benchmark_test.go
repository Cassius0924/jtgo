package core

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"testing"
)

func init() {
	slog.SetDefault(slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func BenchmarkEngine_LoopFor(b *testing.B) {
	// Template using @for loop
	tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for _,val := a": "${val}"
		}
	}
}
	`

	// Test data - array with 10 elements
	testData := make([]string, 10)
	for i := 0; i < 10; i++ {
		testData[i] = fmt.Sprintf("value%d", i)
	}
	dataset := map[string]any{
		"a": testData,
	}

	b.ResetTimer()
	engine, _ := GetJSONTemplateEngine(context.Background(), "BenchLoop", tmpl)
	for i := 0; i < b.N; i++ {
		_, _ = engine.WithDataset(dataset).Run()
	}
}

func BenchmarkEngine_FunctionMap(b *testing.B) {
	// Template using map function
	tmpl := `
{
	"_main_": {
		"sub_test_list": "${map(a, #)}"
	}
}
	`

	// Test data - array with 10 elements
	testData := make([]string, 1)
	for i := 0; i < 1; i++ {
		testData[i] = fmt.Sprintf("value%d", i)
	}
	dataset := map[string]any{
		"a": testData,
	}

	b.ResetTimer()
	engine, _ := GetJSONTemplateEngine(context.Background(), "BenchFunctionMap", tmpl)
	for i := 0; i < b.N; i++ {
		_, _ = engine.WithDataset(dataset).Run()
	}
}

func BenchmarkEngine_LoopForForTmpl(b *testing.B) {
	// Template using @for loop
	tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for idx,val := a": {
				"name": "${idx}_${val}"
			}
		}
	}
}
	`

	// Test data - array with 10 elements
	testData := make([]string, 10)
	for i := 0; i < 10; i++ {
		testData[i] = fmt.Sprintf("value%d", i)
	}
	dataset := map[string]any{
		"a": testData,
	}

	b.ResetTimer()
	engine, _ := GetJSONTemplateEngine(context.Background(), "BenchLoop", tmpl)
	for i := 0; i < b.N; i++ {
		_, _ = engine.WithDataset(dataset).Run()
	}
}

func BenchmarkEngine_FunctionMapForTmpl(b *testing.B) {
	// Template using map function
	tmpl := `
{
	"_main_": {
		"sub_test_list": "${map(a, Use('tmpl_sub_test', Var('idx', #index), Var('val', #)))}"
	},
	"tmpl_sub_test": {
		"name": "${idx}_${val}"
	}
}
	`

	// Test data - array with 10 elements
	testData := make([]string, 10)
	for i := 0; i < 10; i++ {
		testData[i] = fmt.Sprintf("value%d", i)
	}
	dataset := map[string]any{
		"a": testData,
	}

	b.ResetTimer()
	engine, _ := GetJSONTemplateEngine(context.Background(), "BenchFunctionMap", tmpl)
	for i := 0; i < b.N; i++ {
		_, _ = engine.WithDataset(dataset).Run()
	}
}
