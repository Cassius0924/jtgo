package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/lo"
	. "github.com/smartystreets/goconvey/convey"
)

type SubTestObj struct {
	Name string `json:"name"`
}

type TestObj struct {
	Name        string        `json:"name"`
	SubTest     *SubTestObj   `json:"sub_test"`
	IconURL     string        `json:"icon_url"`
	SubTestList []*SubTestObj `json:"sub_test_list"`

	AString    string            `json:"a_string"`
	PtrAString *string           `json:"ptr_a_string"`
	AInt       int64             `json:"a_int"`
	PtrAInt    *int64            `json:"ptr_a_int"`
	AFloat     float64           `json:"a_float"`
	PtrAFloat  *float64          `json:"ptr_a_float"`
	ABool      bool              `json:"a_bool"`
	PtrABool   *bool             `json:"ptr_a_bool"`
	AObject    AObject           `json:"a_object"`
	PtrAObject *AObject          `json:"ptr_a_object"`
	AArray     []int             `json:"a_array"`
	AMap       map[string]string `json:"a_map"`
	AEnum      AEnum             `json:"a_enum"`
	PtrAEnum   *AEnum            `json:"ptr_a_enum"`
}

func TestEngine_Simple(t *testing.T) {
	Convey("TestEngine_ConditionalIf", t, func() {
		tmpl := `
{
	"_main_": {
		"name": "${a}",
		"age": "${b}",
		"work": "coder",
		"key": {
			"a": "${a}",
			"b": "${b}"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalIf", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
			"b": 18,
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"name": true,
	"age": 18,
	"work": "coder",
	"key": {
		"a": true,
		"b": 18
	}
}
	`)

	})
}

func TestEngine_ConditionalIf(t *testing.T) {
	Convey("TestEngine_ConditionalIf", t, func() {
		tmpl := `
{
	"_main_": {
		"name": {
			"sub_name": {
				"@if a": "hello"
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalIf", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"name": {
		"sub_name": "hello"
	}
}
	`)

	})
}

func TestEngine_ConditionalIf2(t *testing.T) {
	Convey("TestEngine_ConditionalIf2", t, func() {
		{
			tmpl := `
{
	"_main_": {
		"name": {
			"wrong": "something",
			"@if a": "hello",
			"wrong": "something"
		}
	}
}
	`
			engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalIf2", tmpl)
			So(err, ShouldBeNil)

			dataset := map[string]any{
				"a": true,
			}

			target := &TestObj{}
			_, _ = engine.WithDataset(dataset).ParseTo(target).Run()

			So(target.Name, ShouldEqual, "hello")
		}
	})
}

func TestEngine_ConditionalIf3(t *testing.T) {
	Convey("TestEngine_ConditionalIf3", t, func() {
		tmpl := `
{
	"_main_": {
		"name": {
			"@if a": {
				"actual": "hello"
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalIf3", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"name": {
		"actual": "hello"
	}
}
	`)

	})
}

func TestEngine_ConditionalIfNested(t *testing.T) {
	Convey("TestEngine_ConditionalIfNested", t, func() {
		tmpl := `
{
	"_main_": {
		"count": {
			"@if true": {
				"@if true": {
					"@if true": 100
				}
			}
		},
		"age": 12,
		"name": {
			"@if a": {
				"@if b": "nothing",
				"@if c": "hello",
				"@if d": "world"
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalIfNested", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
			"b": false,
			"c": true,
			"d": true,
		}

		result, _ := engine.WithDataset(dataset).Run()
		So(result, ShouldEqualJSON, `
{
	"count": 100,
	"age": 12,
	"name": "hello"
}
	`)
	})
}

func TestEngine_ConditionalIfNestedNoMatch(t *testing.T) {
	Convey("TestEngine_ConditionalIfNestedNoMatch", t, func() {
		tmpl := `
{
	"_main_": {
		"age": 12,
		"name": {
			"@if a": {
				"@if b": "nothing"
			},
			"@if a": {
				"@if a": {
					"@if b": [
						"nothing"
					]
				}
			},
			"@if true": "hello"
		},
		"age": 12
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalIfNested2", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
			"b": false,
		}

		result, _ := engine.WithDataset(dataset).Run()
		So(result, ShouldEqualJSON, `
{
	"age": 12,
	"name": "hello"
}
	`)
	})
}

func TestEngine_ConditionalElif(t *testing.T) {
	Convey("TestEngine_ConditionalElif", t, func() {
		tmpl := `
{
	"_main_": {
		"name": {
			"@if a": "nothing",
			"@elif b": "nothing",
			"@elif c": "hello"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalElif", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": false,
			"b": false,
			"c": true,
		}

		target := &TestObj{}
		_, _ = engine.WithDataset(dataset).ParseTo(target).Run()

		So(target.Name, ShouldEqual, "hello")
	})
}

func TestEngine_ConditionalElifNested(t *testing.T) {
	Convey("TestEngine_ConditionalElifNested", t, func() {
		tmpl := `
{
	"_main_": {
		"name": {
			"@if a": "nothing",
			"wrong": "something",
			"@elif b": {
				"@if c + 1 == 2": "hello",
				"wrong": "something"
			},
			"wrong": "something"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalElifNested", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": false,
			"b": true,
			"c": 1,
		}

		target := &TestObj{}
		result, _ := engine.WithDataset(dataset).ParseTo(target).Run()

		So(result, ShouldEqualJSON, `
{
	"name": "hello"
}
	`)
		So(target.Name, ShouldEqual, "hello")

	})
}

func TestEngine_ConditionalElse(t *testing.T) {
	Convey("TestEngine_ConditionalElse", t, func() {
		tmpl := `
{
	"_main_": {
		"name": {
			"@else": "nothing",
			"@if a": "nothing",
			"@else": "hello",
			"@if true": "nothing"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalElse", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": false,
		}

		target := &TestObj{}
		result, _ := engine.WithDataset(dataset).ParseTo(target).Run()

		So(result, ShouldEqualJSON, `
{
	"name": "hello"
}
	`)
		So(target.Name, ShouldEqual, "hello")

	})
}

func TestEngine_LoopArray(t *testing.T) {
	Convey("TestEngine_Loop", t, func() {
		tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for key,val in a": {
				"name": "${key}_${val}"
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_Loop", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": [3]string{
				"json",
				"template",
				"with",
			},
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"sub_test_list": [
		{
			"name": "0_json"
		},
		{
			"name": "1_template"
		},
		{
			"name": "2_with"
		}
	]
} 
	`)
	})
}

func TestEngine_LoopSlice(t *testing.T) {
	Convey("TestEngine_LoopSlice", t, func() {
		tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for idx,val in b": {
				"name": "${idx}_${val}"
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopSlice", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"b": []string{
				"json",
				"template",
				"with",
				"go",
			},
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"sub_test_list": [
		{
			"name": "0_json"
		},
		{
			"name": "1_template"
		},
		{
			"name": "2_with"
		},
		{
			"name": "3_go"
		}
	]
}
	`)
	})
}

func TestEngine_LoopMap(t *testing.T) {
	Convey("TestEngine_LoopMap", t, func() {
		tmpl := `
{
	"_main_": {
		"name_list": {
			"@for key,val in a": {
				"name": "${key}_${val}"
			},
			"abc": "nothing"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopMap", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": map[string]string{
				"hello": "world",
				"hi":    "golang",
			},
		}

		var result map[string]any
		_, _ = engine.WithDataset(dataset).ParseTo(&result).Run()

		// 验证结果结构是否正确
		So(result, ShouldNotBeNil)
		So(result["name_list"], ShouldHaveSameTypeAs, []any{})

		nameList, _ := result["name_list"].([]any)
		So(len(nameList), ShouldEqual, 2) // 应该有2个元素 (map a的长度)

		// 检查所有可能的组合是否存在
		expectedCombinations := map[string]bool{
			"hello_world": false,
			"hi_golang":   false,
		}

		// 收集所有name值
		for _, item := range nameList {
			nameMap := item.(map[string]any)
			name, _ := nameMap["name"].(string)
			expectedCombinations[name] = true
		}

		// 验证所有期望的组合都存在
		for _, found := range expectedCombinations {
			So(found, ShouldBeTrue)
		}
	})
}

func TestEngine_LoopMapNoObject(t *testing.T) {
	Convey("TestEngine_LoopMapNoObject", t, func() {
		tmpl := `
{
	"_main_": {
		"name_list": {
			"@for _,val in a": "${val}"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopMapNoObject", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": map[string]string{
				"hello": "world",
				"hi":    "golang",
			},
		}

		var result map[string]any
		_, _ = engine.WithDataset(dataset).ParseTo(&result).Run()

		// 验证结果结构是否正确
		So(result, ShouldNotBeNil)
		So(result["name_list"], ShouldHaveSameTypeAs, []any{})

		nameList, _ := result["name_list"].([]any)
		So(len(nameList), ShouldEqual, 2) // 应该有2个元素 (map a的长度)

		// 检查所有可能的值是否存在
		expectedValues := map[string]bool{
			"world":  false,
			"golang": false,
		}

		// 收集所有值
		for _, val := range nameList {
			strVal, _ := val.(string)
			expectedValues[strVal] = true
		}

		// 验证所有期望的值都存在
		for _, found := range expectedValues {
			So(found, ShouldBeTrue)
		}
	})
}

func TestEngine_LoopMapNested(t *testing.T) {
	Convey("TestEngine_LoopMapNested", t, func() {
		tmpl := `
{
	"_main_": {
		"name_list": {
			"@for key,val in a": {
				"@for key2,val2 in b": {
					"name": "${key}_${val}_${key2}_${val2}"
				}
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopMapNested", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": map[string]string{
				"hello": "world",
				"hi":    "golang",
			},
			"b": map[string]string{
				"json": "template",
				"with": "go",
			},
		}

		var result map[string]any
		_, _ = engine.WithDataset(dataset).ParseTo(&result).Run()

		// 验证结果结构是否正确
		So(result, ShouldNotBeNil)
		So(result["name_list"], ShouldHaveSameTypeAs, []any{})

		nameList, _ := result["name_list"].([]any)
		So(len(nameList), ShouldEqual, 2) // 应该有2个元素 (a的长度)

		// 验证每个元素是数组且长度为2 (b的长度)
		for _, item := range nameList {
			innerList, ok := item.([]any)
			So(ok, ShouldBeTrue)
			So(len(innerList), ShouldEqual, 2)
		}

		// 检查所有可能的组合是否存在
		expectedCombinations := []string{
			"hello_world_json_template",
			"hello_world_with_go",
			"hi_golang_json_template",
			"hi_golang_with_go",
		}

		foundCombinations := map[string]bool{}

		// 收集所有name值
		for _, outerItem := range nameList {
			innerList := outerItem.([]any)
			for _, innerItem := range innerList {
				nameMap := innerItem.(map[string]any)
				name, _ := nameMap["name"].(string)
				foundCombinations[name] = true
			}
		}

		// 验证所有期望的组合都存在
		for _, expected := range expectedCombinations {
			So(foundCombinations[expected], ShouldBeTrue)
		}
	})
}

func TestEngine_LoopSliceNested(t *testing.T) {
	Convey("TestEngine_LoopSliceNested", t, func() {
		tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for idx,val in a": {
				"@for idx2,val2 in b": {
					"name": "${idx}_${val}_${idx2}_${val2}"
				}
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopSliceNested", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": []string{
				"a1",
			},
			"b": []string{
				"b1",
				"b2",
			},
		}

		result, _ := engine.WithDataset(dataset).Run()
		So(result, ShouldEqualJSON, `
{
	"sub_test_list": [
		[
			{
				"name": "0_a1_0_b1"
			},	
			{
				"name": "0_a1_1_b2"
			}
		]
	]
}
		`)
	})
}

func TestEngine_LoopWithError(t *testing.T) {
	Convey("TestEngine_LoopWithError", t, func() {
		tmpl := `
{
	"_main_": {
		"value": {
			"@for idx,val in a": {
				"name": "${idx}_${val}"
			},
			"name": "wrong"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopWithError", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": 1,
		}

		result, err := engine.WithDataset(dataset).Run()
		So(err, ShouldNotBeNil)
		So(result, ShouldEqualJSON, `
		{
			"value": {}
		}`)

	})
}

func TestEngine_ConditionalIfNestedLoop(t *testing.T) {
	Convey("TestEngine_IfNestedLoop", t, func() {
		tmpl := `
{
	"_main_": {
		"value": {
			"@if a": {
				"@for idx,val in b": {
					"name": "${idx}_${val}"
				}
			},
			"@else": "hello"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_IfNestedLoop", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
			"b": []string{
				"json",
				"template",
				"with",
				"go",
			},
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"value": [
		{
			"name": "0_json"
		},
		{
			"name": "1_template"
		},
		{
			"name": "2_with"
		},
		{
			"name": "3_go"
		}
	]
}
	`)

		dataset["a"] = false
		result, _ = engine.WithDataset(dataset).Run()
		So(result, ShouldEqualJSON, `
{
	"value": "hello"
}
	`)

	})
}

func TestEngine_Comment(t *testing.T) {
	Convey("TestEngine_Comment", t, func() {
		tmpl := `
{
	"_main_": {
		"@cmt": "this is a comment",
		"name": "hello",
		"@cmt": 2,
		"@cmt": null,
		"@cmt": true,
		"@cmt": {
			"@if a": "nothing"
		},
		"@cmt": [
			"this is a comment",
			"this is a comment too"
		]
	}
}
`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_Comment", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"name": "hello"
}
`)

	})
}

func TestEngine_Var(t *testing.T) {
	Convey("TestEngine_Var", t, func() {
		tmpl := `
{
	"_main_": {
		"@var": {
			"e": "ines"
		},
		"name": {
			"@var": {
				"a": "hello",
				"b": 1,
				"c": "num_${x + y}",
				"d": false
			},
			"value": "${a}_${b}_${c}_${d}_${e}"
		}
	}
}
`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_Var", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
			"x": 1.2,
			"y": 2,
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"name": {
		"value": "hello_1_num_3.2_false_ines"
	}
}
`)

	})
}

func TestEngine_VarWithError(t *testing.T) {
	Convey("TestEngine_VarWithError", t, func() {
		tmpl := `
{
	"_main_": {
		"@var": "wrong",
		"name": {
			"value": "${a}"
		}
	}
}
`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_VarWithError", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": "hello",
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"name": {
		"value": "hello"
	}
}
`)

	})
}

func TestEngine_VarWithIf(t *testing.T) {
	Convey("TestEngine_VarWithIf", t, func() {
		// TODO: 限制不能if套var，仅可var套if
		tmpl := `
{
	"_main_": {
		"name": {
			"@var": {
				"@if noExist": {
					"y": "nothing"
				},
				"@if true": {
					"d": "one"
				},
				"a": false,
				"@if a": {
					"@if false": {
						"z": "wrong"
					}
				},
				"@elif true": {
					"@else": {
						"x": "orphan else"
					},
					"@if true": {
						"b": "world"
					},
					"@if true": {
						"c": "json"
					}
				},
				"@else": {
					"x": "nothing"
				}
			},
			"value": "${a}_${b}_${c}_${d}_${x}_${y}_${z}"
		}
	}
}
`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_VarWithIf", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"name": {
		"value": "false_world_json_one___"
	}
}
`)

	})
}

func TestEngine_TemplateInList(t *testing.T) {
	Convey("TestEngine_TemplateInList", t, func() {
		tmpl := `
{
    "_main_": [
        "hello",
		[
            {
                "name": "${a}"
            },
            {
                "value": "${b}"
            }
        ],
        {
            "name": "hello",
            "value": {
                "@if a": "world"
            }
        },
        {
            "time": {
                "@if a": "now",
                "@else": "idk"
            },
            "author": "Cassius0924"
        }
    ]
}
`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_TemplateInList", tmpl)
		So(err, ShouldBeNil)

		dataset := map[string]any{
			"a": true,
			"b": "best",
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
[
	"hello",
	[
		{
			"name": true
		},
		{
			"value": "best"
		}
	],
	{
		"name": "hello",
		"value": "world"
	},
	{
		"time": "now",
		"author": "Cassius0924"
	}
]
`)

	})
}

func TestEngine_ExecString(t *testing.T) {
	Convey("TestEngine_ExecString", t, func() {
		tmpl := `
{
	"_main_": {
		"@exec": "${Inc(a)}",
		"value": {
			"@exec": "${Inc(a)}",
			"name": "${a}"
		}
	}
}
`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ExecString", tmpl)
		So(err, ShouldBeNil)

		a := 0
		dataset := map[string]any{
			"a": &a,
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"value": {
		"name": 2
	}
}
`)

	})
}

func TestEngine_ExecArray(t *testing.T) {
	Convey("TestEngine_ExecArray", t, func() {
		tmpl := `
{
	"_main_": {
		"@exec": [
			"${Inc(a)}",
			"${Inc(a)}"
		],
		"value": {
			"@exec": [
				"${Inc(a)}"
			],
			"name": "${a}"
		}
	}
}
`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ExecArray", tmpl)
		So(err, ShouldBeNil)

		a := 0
		dataset := map[string]any{
			"a": &a,
		}

		result, _ := engine.WithDataset(dataset).Run()

		So(result, ShouldEqualJSON, `
{
	"value": {
		"name": 3
	}
}
`)

	})
}

func TestEngine_ComplexTemplate(t *testing.T) {
	Convey("TestEngine_ComplexTemplate", t, func() {
		tmpl := `
{
    "_main_": {
		"@var": {
			"userCount": "${PtrInt(0)}"
		},
		"users": {
			"@cmt": "this is a comment",
			"@cmt": ["this is a comment", "this is a comment too"],
			"@exec": [
				"${LogInfo('start')}"
			],
            "@for idx,user in userList": {
				"@exec": [
					"${LogInfo('parse user', 'index', idx)}",
					"${Inc(userCount)}"
				],
				"@cmt": "this is a comment",
				"num": "${idx + 1}",
                "name": "${user.first_name} ${user.last_name}",
                "age_group": {
                    "@if user.age < 0": {
						"@if user.age == -1": "unknown",
						"@else": "invalid"
					},
                    "@if user.age < 18": {
						"@if user.age < 12": "kid",
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
			"pass_count": "${CalPassCount(userList)}",
			"user_count": "${userCount}"
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

		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ComplexTemplate", tmpl)
		So(err, ShouldBeNil)

		result, _ := engine.WithDataset(dataset).Run()
		fmt.Println(result)

		So(result, ShouldEqualJSON, `
{
    "users": [
        {
			"num": 1,
			"name": "Alice L",
            "age_group": "teenager",
            "tags": ["vip", "beta"],
            "score": 90,
            "status": "excellent"
        },
        {
			"num": 2,
            "name": "Bob H",
            "age_group": "adult",
            "tags": ["new"],
            "score": 70,
            "status": "active"
        },
        {
			"num": 3,
            "name": "Tim Z",
            "age_group": "senior",
            "tags": [],
            "score": 50,
            "status": "inactive"
        },
		{
			"num": 4,
			"name": "Baby S",
			"age_group": "kid",
			"tags": [3, 2],
			"score": 20,
			"status": "active"
		}
    ],
    "summary": {
        "total": 4,
		"pass_count": 2,
		"user_count": 4
    }
}
        `)
	})
}
