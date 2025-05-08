package example

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"testing"

	"github.com/cassius0924/jtgo"
	"github.com/samber/lo"
)

func TestEngine_InputOutput(t *testing.T) {
	// 1. 获取数据集
	dataset := fetchDataset("dataset.json", t)
	// 2. 读取模板
	tmpl := readTemplate("input_template.jtgo", t)

	// 3. 创建引擎
	engine, _ := jtgo.GetEngine(context.Background(), "TestEngine_InputOutput", tmpl)

	// 4. 传入数据集并解析模板，获得JSON内容
	result, _ := engine.WithDataset(dataset).Run()

	// 5. 将结果写入文件
	writeOutput("output_json.json", result, t)
}

func fetchDataset(filename string, t *testing.T) map[string]any {
	datasetBytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read dataset: %v", err)
	}

	dataset := make(map[string]any)
	err = json.Unmarshal(datasetBytes, &dataset)
	if err != nil {
		t.Fatalf("failed to unmarshal dataset: %v", err)
	}

	dataset["CalPassCount"] = func(arr any) int {
		var tmp []map[string]any
		for _, v := range arr.([]any) {
			if m, ok := v.(map[string]any); ok {
				tmp = append(tmp, m)
			}
		}
		cnt := lo.SumBy(tmp, func(u map[string]any) int {
			if v, ok := u["score"].(float64); ok && v >= 0.6 {
				return 1
			}
			return 0
		})
		return cnt
	}

	return dataset
}

func readTemplate(filename string, t *testing.T) string {
	tmplBytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatalf("failed to read template: %v", err)
	}
	return string(tmplBytes)
}

func writeOutput(filename string, result string, t *testing.T) {
	var prettyJSON bytes.Buffer
	if err := json.Indent(&prettyJSON, []byte(result), "", "    "); err != nil {
		t.Fatalf("failed to prettify json: %v", err)
	}
	err := os.WriteFile(filename, prettyJSON.Bytes(), 0644)
	if err != nil {
		t.Fatalf("failed to write output: %v", err)
	}
}
