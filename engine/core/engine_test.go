package core

import (
	"context"
	"fmt"
	"testing"

	"github.com/samber/lo"
	"github.com/smartystreets/goconvey/convey"
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
	convey.Convey("TestEngine_ConditionalIf", t, func() {
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
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": true,
			"b": 18,
		}

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
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
	convey.Convey("TestEngine_ConditionalIf", t, func() {
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
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": true,
		}

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
{
	"name": {
		"sub_name": "hello"
	}
}
	`)

	})
}

func TestEngine_ConditionalIf2(t *testing.T) {
	convey.Convey("TestEngine_ConditionalIf2", t, func() {
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
			convey.So(err, convey.ShouldBeNil)

			dataset := map[string]any{
				"a": true,
			}

			target := &TestObj{}
			_, _ = engine.WithDataset(dataset).ParseTo(target).Run()

			convey.So(target.Name, convey.ShouldEqual, "hello")
		}
	})
}

func TestEngine_ConditionalIf3(t *testing.T) {
	convey.Convey("TestEngine_ConditionalIf3", t, func() {
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
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": true,
		}

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
{
	"name": {
		"actual": "hello"
	}
}
	`)

	})
}

func TestEngine_ConditionalIfNested(t *testing.T) {
	convey.Convey("TestEngine_ConditionalIfNested", t, func() {
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
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": true,
			"b": false,
			"c": true,
			"d": true,
		}

		result, _ := engine.WithDataset(dataset).Run()
		convey.So(result, convey.ShouldEqualJSON, `
{
	"count": 100,
	"age": 12,
	"name": "hello"
}
	`)
	})
}

func TestEngine_ConditionalElif(t *testing.T) {
	convey.Convey("TestEngine_ConditionalElif", t, func() {
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
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": false,
			"b": false,
			"c": true,
		}

		target := &TestObj{}
		_, _ = engine.WithDataset(dataset).ParseTo(target).Run()

		convey.So(target.Name, convey.ShouldEqual, "hello")
	})
}

func TestEngine_ConditionalElifNested(t *testing.T) {
	convey.Convey("TestEngine_ConditionalElifNested", t, func() {
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
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": false,
			"b": true,
			"c": 1,
		}

		target := &TestObj{}
		result, _ := engine.WithDataset(dataset).ParseTo(target).Run()

		convey.So(result, convey.ShouldEqualJSON, `
{
	"name": "hello"
}
	`)
		convey.So(target.Name, convey.ShouldEqual, "hello")

	})
}

func TestEngine_ConditionalElse(t *testing.T) {
	convey.Convey("TestEngine_ConditionalElse", t, func() {
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
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": false,
		}

		target := &TestObj{}
		result, _ := engine.WithDataset(dataset).ParseTo(target).Run()

		convey.So(result, convey.ShouldEqualJSON, `
{
	"name": "hello"
}
	`)
		convey.So(target.Name, convey.ShouldEqual, "hello")

	})
}

func TestEngine_LoopArray(t *testing.T) {
	convey.Convey("TestEngine_Loop", t, func() {
		tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for key,val := a": {
				"name": "${key}_${val}"
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_Loop", tmpl)
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": [3]string{
				"json",
				"template",
				"with",
			},
		}

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
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
	convey.Convey("TestEngine_LoopSlice", t, func() {
		tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for idx,val := b": {
				"name": "${idx}_${val}"
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopSlice", tmpl)
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"b": []string{
				"json",
				"template",
				"with",
				"go",
			},
		}

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
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
	convey.Convey("TestEngine_LoopMap", t, func() {
		tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for key,val := a": {
				"name": "${key}_${val}"
			},
			"abc": "nothing"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopMap", tmpl)
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": map[string]string{
				"hello": "world",
				"hi":    "golang",
			},
		}

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
{
	"sub_test_list": [
		{
			"name": "hello_world"
		},
		{
			"name": "hi_golang"
		}
	]
}
		`)
	})
}

func TestEngine_LoopMapNoObject(t *testing.T) {
	convey.Convey("TestEngine_LoopMap", t, func() {
		tmpl := `
{
	"_main_": {
		"name_list": {
			"@for _,val := a": "${val}"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopMap", tmpl)
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": map[string]string{
				"hello": "world",
				"hi":    "golang",
			},
		}

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
{
	"name_list": [
		"world",
		"golang"
	]
}
		`)
	})
}

func TestEngine_LoopMapNested(t *testing.T) {
	convey.Convey("TestEngine_LoopMapNested", t, func() {
		tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for key,val := a": {
				"@for key2,val2 := b": {
					"name": "${key}_${val}_${key2}_${val2}"
				}
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopMapNested", tmpl)
		convey.So(err, convey.ShouldBeNil)

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

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
{
	"sub_test_list": [
		[
			{
				"name": "hello_world_json_template"
			},	
			{
				"name": "hello_world_with_go"
			}
		],
		[
			{
				"name": "hi_golang_json_template"
			},
			{
				"name": "hi_golang_with_go"
			}
		]
	]
}
		`)
	})
}

func TestEngine_LoopSliceNested(t *testing.T) {
	convey.Convey("TestEngine_LoopSliceNested", t, func() {
		tmpl := `
{
	"_main_": {
		"sub_test_list": {
			"@for idx,val := a": {
				"@for idx2,val2 := b": {
					"name": "${idx}_${val}_${idx2}_${val2}"
				}
			}
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_LoopSliceNested", tmpl)
		convey.So(err, convey.ShouldBeNil)

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
		convey.So(result, convey.ShouldEqualJSON, `
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

func TestEngine_Comment(t *testing.T) {
	convey.Convey("TestEngine_Comment", t, func() {
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
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": true,
		}

		result, _ := engine.WithDataset(dataset).Run()

		convey.So(result, convey.ShouldEqualJSON, `
{
	"name": "hello"
}
`)

	})
}

func TestEngine_ComplexTemplate(t *testing.T) {
	convey.Convey("TestEngine_ComplexTemplate", t, func() {
		tmpl := `
{
    "_main_": {
		"users": {
			"@cmt": "this is a comment",
			"@cmt": 2,
			"@cmt": true,
            "@for idx,user := userList": {
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

		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ComplexTemplate", tmpl)
		convey.So(err, convey.ShouldBeNil)

		result, _ := engine.WithDataset(dataset).Run()
		fmt.Println(result)

		convey.So(result, convey.ShouldEqualJSON, `
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
			"age_group": "teenager",
			"tags": [3, 2],
			"score": 20,
			"status": "active"
		}
    ],
    "summary": {
        "total": 4,
		"pass_count": 2
    }
}
        `)
	})
}
