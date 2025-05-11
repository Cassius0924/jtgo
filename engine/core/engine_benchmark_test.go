package core

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime/pprof"
	"testing"

	"github.com/samber/lo"
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

// BenchmarkEngine_ComplexTemplate 测试复杂模板的性能
func BenchmarkEngine_ComplexTemplate(b *testing.B) {
	// 准备模板和数据集
	tmpl := `
{
    "_main_": {
		"users": {
			"@cmt": "this is a comment",
			"@cmt": 2,
			"@cmt": true,
            "@for idx,user in userList": {
				"@cmt": "this is a comment",
				"num": "${idx + 1}",
                "name": "${user.first_name} ${user.last_name}",
                "age_group": {
                    "@if user.age < 18": {
						"@if user.age < 12": "teenager",
						"@else": "teenager"
					},
                    "@elif user.age < 60": "adult",
                    "@else": "senior"
                },
                "tags": "${user.tags}",
                "score": "${user.score * 100}",
                "status": {
                    "@if user.active && user.score > 0.8": "excellent",
                    "@elif user.active": "active",
					"@cmt": "this is a comment",
                    "@else": "inactive"
                }
            }
        },
		"@cmt": "this is a comment",
        "summary": {
			"total": "${len(userList)}",
			"pass_count": "${CalPassCount(userList)}"
        }
    }
}
        `
	userList := []map[string]any{
		{"last_name": "L", "first_name": "Alice", "age": 17, "tags": []string{"vip", "beta"}, "score": 0.9, "active": true},
		{"last_name": "H", "first_name": "Bob", "age": 25, "tags": []string{"new"}, "score": 0.7, "active": true},
		{"last_name": "Z", "first_name": "Tim", "age": 65, "tags": []string{}, "score": 0.5, "active": false},
		{"last_name": "S", "first_name": "Baby", "age": 9, "tags": []int{3, 2}, "score": 0.2, "active": true},
	}
	dataset := map[string]any{
		"userList": userList,
		"CalPassCount": func(arr any) int {
			cnt := lo.SumBy(arr.([]map[string]any), func(u map[string]any) int {
				if v, ok := u["score"].(float64); ok && v >= 0.6 {
					return 1
				}
				return 0
			})
			return cnt
		},
	}

	// 创建引擎
	engine, err := GetJSONTemplateEngine(context.Background(), "BenchmarkEngine_ComplexTemplate", tmpl)
	if err != nil {
		b.Fatalf("failed to create engine: %v", err)
	}

	// CPU Profile
	cpuFile, _ := os.Create("cpu_profile.prof")
	defer cpuFile.Close()
	pprof.StartCPUProfile(cpuFile)
	defer pprof.StopCPUProfile()

	// 内存 Profile
	memFile, _ := os.Create("mem_profile.prof")
	defer memFile.Close()
	defer pprof.WriteHeapProfile(memFile)

	// 重置计时器，避免引擎初始化的时间影响基准测试结果
	b.ResetTimer()

	// 执行基准测试
	for i := 0; i < b.N; i++ {
		engine.WithDataset(dataset).Run()
	}
}