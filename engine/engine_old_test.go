package engine

import (
	"context"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/cassius0924/jtgo/util/ptr"
	"github.com/smartystreets/goconvey/convey"
)

type SubTitle struct {
	SubTitleText string `json:"sub_title_text"`
}

type Button struct {
	ButtonText string `json:"button_text"`
}

type AObject struct {
	Name string `json:"name"`
}

type AEnum int64

type PromoteGameModuleInfo struct {
	Title      string      `json:"title"`
	SubTitle   *SubTitle `json:"sub_title"`
	Button     Button      `json:"button"`
	IconURL    string      `json:"icon_url"`
	ButtonList []*Button   `json:"button_list"`

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

func init() {
	RegisterBuiltInFunction(toInt)
}

// toInt 用于将指针类型 type int 类型转为 int
func toInt(ptrValue ...any) (any, error) {
	v := reflect.ValueOf(ptrValue)

	// 检查是否为基于 int 的类型
	v = v.Elem()
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint()), nil
	default:
		return 0, fmt.Errorf("unsupported type: %T", ptrValue)
	}
}

func TestConfigEngine_CreateEngine(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"sub_title": {
					"sub_title_text": {
						"toInt(${Request.Scene}) == 1 && ${Request.Type} == 'a'": null,
						"toInt(${Request.Scene}) == 2 && ${Request.Type} == 'b'": {},
						"toInt(${Request.Scene}) == 3 && ${Request.Type} == 'c'": "近期直播内容",
						"@default": "近期内容"
					}
				}
			}
		}
		`
		_, err0 := GetJSONTemplateEngine(context.Background(), "test", "{1")
		convey.So(err0, convey.ShouldBeError)

		_, err1 := GetJSONTemplateEngine(context.Background(), "test", "")
		convey.So(err1, convey.ShouldBeError)

		_, err2 := GetJSONTemplateEngine(context.Background(), "", configJSON)
		convey.So(err2, convey.ShouldBeError)

		_, err3 := GetJSONTemplateEngine(context.Background(), "test", "")
		convey.So(err3, convey.ShouldBeError)

		//=== 测试 configJSON 未变
		ctx := context.Background()
		engine1, err4 := GetJSONTemplateEngine(ctx, "test", configJSON)
		convey.So(err4, convey.ShouldBeNil)

		engine2, _ := GetJSONTemplateEngine(ctx, "test", configJSON)
		fnName1 := runtime.FuncForPC(reflect.ValueOf(engine1.fns["toInt"]).Pointer()).Name()
		fnName2 := runtime.FuncForPC(reflect.ValueOf(engine2.fns["toInt"]).Pointer()).Name()
		convey.So(fnName1, convey.ShouldEqual, fnName2)
		convey.So(engine1.dataset, convey.ShouldResemble, engine2.dataset)
		convey.So(engine1.template, convey.ShouldResemble, engine2.template)

		//=== 测试错误的 configJSON
		configJSON = `
		{
			"promote_game_module_info": {
				"sub_title": {
					"sub_title_text": {
						"toInt(${Request.Scene}) == 1 && ${Request.Type} == 'a'": null,
						"@default": "近期内容"
					}
				},
			}
		`

		engine3, _ := GetJSONTemplateEngine(context.Background(), "test", configJSON)
		fnName3 := runtime.FuncForPC(reflect.ValueOf(engine3.fns["toInt"]).Pointer()).Name()
		convey.So(fnName2, convey.ShouldEqual, fnName3)
		convey.So(engine2.dataset, convey.ShouldResemble, engine3.dataset)
		convey.So(engine2.template, convey.ShouldResemble, engine3.template)
	})
}

func TestConfigEngine_Run_CustomFunction(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		type Enum int
		const (
			EnumA Enum = 1
		)
		configJSON := `
			{
				"promote_game_module_info": {
					"sub_title": {
						"sub_title_text": {
							"toIntV1(${Request.Scene}) == 1 && ${Request.Type} == 'a'": "近期直播内容" ,
							"@default": "近期内容"
						}
					}
				}
			}
		`
		// === 测试未注册函数
		//engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)
		temp1 := EnumA

		dataset1 := map[string]any{
			"Request": map[string]any{
				"Scene": &temp1,
				"Type":  "a",
			},
		}
		//result1 := PromoteGameModuleInfo{}
		//err1 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result1).Run()
		//convey.So(err1, convey.ShouldBeError)

		// === 测试注册过函数
		RegisterFunction(
			"test_custom_function",
			toIntV1,
		)

		engine2, _ := GetJSONTemplateEngine(context.Background(), "test_custom_function", configJSON)
		result2 := PromoteGameModuleInfo{}
		err2 := engine2.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result2).Run()
		convey.So(err2, convey.ShouldBeNil)
		convey.So(result2.SubTitle.SubTitleText, convey.ShouldEqual, "近期直播内容")

		RegisterFunction(
			"test_custom_function",
			nil,
		)
		RegisterFunction(
			"",
			toIntV1,
		)

	})

}

func toIntV1(ptrValue any) any {
	v := reflect.ValueOf(ptrValue)

	// 检查是否为基于 int 的类型
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint())
	case reflect.String:
		i, _ := strconv.Atoi(v.String())
		return i
	default:
		return 0
	}
}

func TestConfigEngine_Run_Default(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
			{
				"promote_game_module_info": {
					"sub_title": {
						"sub_title_text": {
							"${Request.Scene} == 1 && ${Request.Type} == 'a'": null,
							"${Request.Scene} == 2 && ${Request.Type} == 'b'": {},
							"${Request.Scene} == 3 && ${Request.Type} == 'c'": "近期直播内容" ,
							"@default": "近期内容"
						}
					}
				}
			}
		`

		// === 测试命中 DEFAULT
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		dataset1 := map[string]any{
			"Request": map[string]any{
				"Scene": 4,
				"Type":  "w",
			},
		}
		result1 := PromoteGameModuleInfo{}
		err1 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result1).Run()
		convey.So(err1, convey.ShouldBeNil)
		convey.So(result1.SubTitle.SubTitleText, convey.ShouldEqual, "近期内容")

		// === 测试不命中 DEFAULT
		dataset2 := map[string]any{
			"Request": map[string]any{
				"Scene": 3,
				"Type":  "c",
			},
		}
		result2 := PromoteGameModuleInfo{}
		err2 := engine.WithDataset(dataset2).WithCustomEntry("promote_game_module_info").ParseTo(&result2).Run()
		convey.So(err2, convey.ShouldBeNil)
		convey.So(result2.SubTitle.SubTitleText, convey.ShouldEqual, "近期直播内容")
		configJSON = `
			{
				"promote_game_module_info": {
					"sub_title": {
						"sub_title_text": {
							"@default": "近期内容1",
							"${Request.Scene} == 1 && ${Request.Type} == 'a'": null,
							"${Request.Scene} == 2 && ${Request.Type} == 'b'": {},
							"@default": "近期内容2",
							"${Request.Scene} == 3 && ${Request.Type} == 'c'": "近期直播内容"
						}
					}
				}
			}
		`
		engine3, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)
		result3 := PromoteGameModuleInfo{}
		err3 := engine3.WithDataset(dataset2).WithCustomEntry("promote_game_module_info").ParseTo(&result3).Run()
		convey.So(err3, convey.ShouldBeNil)
		convey.So(result3.SubTitle.SubTitleText, convey.ShouldEqual, "近期直播内容")
	})
}

func TestConfigEngine_Run_ReplaceVariable(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"sub_title": {
					"@default": {
						"sub_title_text": "${SubTitleText}"
					}
				},
				"icon_url": {
					"@default": "${PromoteGame.GameInfo.Icon}"
				},
				"title": {
					"@default": "这是${PromoteGame.GameInfo.Name} ${PromoteGame.GameInfo.Type}游戏"    
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		dataset1 := map[string]any{
			"SubTitleText": "副标题",
			"PromoteGame": map[string]any{
				"GameInfo": map[string]any{
					"Icon": "https://www.bytedance.com",
					"Name": "王者荣耀",
					"Type": "MOBA",
				},
			},
		}
		result1 := PromoteGameModuleInfo{}
		err1 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result1).Run()
		convey.So(err1, convey.ShouldBeNil)
		convey.So(result1.SubTitle.SubTitleText, convey.ShouldEqual, "副标题")
		convey.So(result1.IconURL, convey.ShouldEqual, "https://www.bytedance.com")
		convey.So(result1.Title, convey.ShouldEqual, "这是王者荣耀 MOBA游戏")
	})
}

func TestConfigEngine_Run_MultiModule(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"icon_url": {
					"@default": "${PromoteGame.GameInfo.Icon}"
				}
			},
			"other_module": {
				"title": {
					"@default": "这是${PromoteGame.GameInfo.Name}游戏"    
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		dataset1 := map[string]any{
			"PromoteGame": map[string]any{
				"GameInfo": map[string]any{
					"Icon": "https://www.bytedance.com",
					"Name": "王者荣耀",
				},
			},
		}
		result1 := PromoteGameModuleInfo{}
		err1 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result1).Run()
		convey.So(err1, convey.ShouldBeNil)
		convey.So(result1.IconURL, convey.ShouldEqual, "https://www.bytedance.com")

		result2 := PromoteGameModuleInfo{}
		err2 := engine.WithDataset(dataset1).WithCustomEntry("other_module").ParseTo(&result2).Run()
		convey.So(err2, convey.ShouldBeNil)
		convey.So(result2.Title, convey.ShouldEqual, "这是王者荣耀游戏")
	})
}

func TestConfigEngine_Run_InvalidParams(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"icon_url": {
					"@default": "${PromoteGame.GameInfo.Icon}"
				}
			}
		}
		`
		// === 测试非法的 module
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		dataset1 := map[string]any{
			"PromoteGame": map[string]any{
				"GameInfo": map[string]any{
					"Icon": "https://www.bytedance.com",
				},
			},
		}
		result1 := PromoteGameModuleInfo{}
		err1 := engine.WithDataset(dataset1).WithCustomEntry("other_module").ParseTo(&result1).Run()
		convey.So(err1, convey.ShouldBeError)

		// === 测试参数校验
		result2 := PromoteGameModuleInfo{}
		err2 := engine.WithCustomEntry("promote_game_module_info").ParseTo(&result2).Run()
		convey.So(err2, convey.ShouldBeNil)
		err3 := engine.WithDataset(dataset1).ParseTo(&result2).Run()
		convey.So(err3, convey.ShouldBeError)
		err4 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").Run()
		convey.So(err4, convey.ShouldBeError)

		// === 测试非法的 destination
		result3 := 1
		err5 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result3).Run()
		convey.So(err5, convey.ShouldBeError)
	})
}

func TestConfigEngine_Run_BaseType(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"a_string": {
					"@default": "string"
				},
				"ptr_a_string": {
					"@default": "ptr_string"
				},
				"a_int": {	
					"@default": 1
				},
				"ptr_a_int": {
					"@default": 2
				},	
				"a_float": {	
					"@default": 3.1
				},
				"ptr_a_float": {	
					"@default": 4.1
				},
				"a_bool": {	
					"@default": true
				},	
				"ptr_a_bool": {
					"@default": false
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		result := PromoteGameModuleInfo{}
		err := engine.WithDataset(nil).WithCustomEntry("promote_game_module_info").ParseTo(&result).Run()
		convey.So(err, convey.ShouldBeNil)
		convey.So(result.AString, convey.ShouldEqual, "string")
		convey.So(*result.PtrAString, convey.ShouldEqual, "ptr_string")
		convey.So(result.AInt, convey.ShouldEqual, int64(1))
		convey.So(*result.PtrAInt, convey.ShouldEqual, int64(2))
		convey.So(result.AFloat, convey.ShouldEqual, 3.1)
		convey.So(*result.PtrAFloat, convey.ShouldEqual, 4.1)
		convey.So(result.ABool, convey.ShouldEqual, true)
		convey.So(*result.PtrABool, convey.ShouldEqual, false)
	})
}

func TestConfigEngine_Run_StructType(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"a_object": {
					"@default": {
						"name": "object"
					}
				},
				"ptr_a_object": {
					"@default": {
						"name": "ptr_object"
					}
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		result := PromoteGameModuleInfo{}
		err := engine.WithDataset(nil).WithCustomEntry("promote_game_module_info").ParseTo(&result).Run()
		convey.So(err, convey.ShouldBeNil)
		convey.So(result.AObject.Name, convey.ShouldEqual, "object")
		convey.So(result.PtrAObject.Name, convey.ShouldEqual, "ptr_object")
	})
}

func TestConfigEngine_Run_ArrayType(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"a_array": {
					"@default": [1, 2, 3, 4]
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		result := PromoteGameModuleInfo{}
		err := engine.WithDataset(nil).WithCustomEntry("promote_game_module_info").ParseTo(&result).Run()
		convey.So(err, convey.ShouldBeNil)
		convey.So(result.AArray, convey.ShouldResemble, []int{1, 2, 3, 4})
	})
}

func TestConfigEngine_Run_MapType(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"a_map": {
					"@default": {
						"key": "value"
					}
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		result := PromoteGameModuleInfo{}
		err := engine.WithDataset(nil).WithCustomEntry("promote_game_module_info").ParseTo(&result).Run()
		convey.So(err, convey.ShouldBeNil)
		convey.So(result.AMap, convey.ShouldResemble, map[string]string{"key": "value"})
	})
}

func TestConfigEngine_Run_Enum(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"a_enum": {
					"@default": 1
				},
				"ptr_a_enum": {
					"@default": 2
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		result := PromoteGameModuleInfo{}
		err := engine.WithDataset(nil).WithCustomEntry("promote_game_module_info").ParseTo(&result).Run()
		convey.So(err, convey.ShouldBeNil)
		convey.So(result.AEnum, convey.ShouldEqual, AEnum(1))
		convey.So(*result.PtrAEnum, convey.ShouldEqual, AEnum(2))
	})
}

func TestConfigEngine_Run_Expression(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"sub_title": {
					"${Request.Scene} == 1 && ${Request.Type} == 'a'": null,
					"${Request.Scene} == 2 && ${Request.Type} == 'b'": {},
					"${Request.Scene} == 3 && ${Request.Type} == 'c'": {
						"sub_title_text": "近期直播内容"
					},	
					"@default": {
						"sub_title_text": "近期内容"
					}
				},
				"button": {
					"${Request.Scene} == 2 && ${Request.Type} == 'b'": {
						"button_text": "添加"
					},
					"${Request.Scene} == 3 && ${Request.Type} == 'c'": {
						"button_text": "已添加"
					},
					"@default": {
						"button_text": "默认添加"
					}
				},
				"icon_url": {
					"${Request.Scene} == 1 && ${Request.Type} == 'a'": null,
					"@default": "HTTP"
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		dataset1 := map[string]any{
			"Request": map[string]any{
				"Scene": 1,
				"Type":  "a",
			},
		}
		result1 := PromoteGameModuleInfo{
			SubTitle: &SubTitle{
				SubTitleText: "测试测试",
			},
			IconURL: "https://www.bytedance.com",
		}
		err1 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result1).Run()
		convey.So(err1, convey.ShouldBeNil)
		convey.So(result1.SubTitle, convey.ShouldBeNil)
		convey.So(result1.IconURL, convey.ShouldEqual, "https://www.bytedance.com")
		convey.So(result1.Button.ButtonText, convey.ShouldEqual, "默认添加")

		dataset2 := map[string]any{
			"Request": map[string]any{
				"Scene": 2,
				"Type":  "b",
			},
		}
		result2 := PromoteGameModuleInfo{
			SubTitle: &SubTitle{
				SubTitleText: "测试测试",
			},
		}
		err2 := engine.WithDataset(dataset2).WithCustomEntry("promote_game_module_info").ParseTo(&result2).Run()
		convey.So(err2, convey.ShouldBeNil)
		convey.So(result2.SubTitle.SubTitleText, convey.ShouldEqual, "测试测试")
		convey.So(result2.Button.ButtonText, convey.ShouldEqual, "添加")

		dataset3 := map[string]any{
			"Request": map[string]any{
				"Scene": 3,
				"Type":  "c",
			},
		}
		result3 := PromoteGameModuleInfo{}
		err3 := engine.WithDataset(dataset3).WithCustomEntry("promote_game_module_info").ParseTo(&result3).Run()
		convey.So(err3, convey.ShouldBeNil)
		convey.So(result3.SubTitle.SubTitleText, convey.ShouldEqual, "近期直播内容")
		convey.So(result3.Button.ButtonText, convey.ShouldEqual, "已添加")
	})
}

func TestConfigEngine_Run_WrongExpression(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"sub_title": {
					"${Request.Scene == 1 && ${Request.Type} == 'a'": "a",
					"${Request.Scene} == 2 + ${Request.Type} == 'b'": "b",
					"${Request.Scene} == 3s ${Request.Type} == 'c'": "c",
					"${Request.Scene}": "d",
					"@default": "e"
				}
			}
		}	
		`
		engine, err := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)
		convey.So(err, convey.ShouldBeNil)

		dataset1 := map[string]any{
			"Request": map[string]any{
				"Scene": 1,
			},
		}
		result1 := PromoteGameModuleInfo{}
		err1 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result1).Run()
		convey.So(err1, convey.ShouldBeError)
	})
}

func TestConfigEngine_Run_WrongDataset(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"promote_game_module_info": {
				"sub_title": {
					"sub_title_text": {
						"${Request} != nil && ${Request.Scene} == 1": "游戏名为${Name_Wrong}",
						"@default": "游戏"
					}
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

		dataset1 := map[string]any{
			"Request": map[string]any{
				"Scene": 1,
			},
			"Name": "守望先锋",
		}
		result1 := PromoteGameModuleInfo{}
		err1 := engine.WithDataset(dataset1).WithCustomEntry("promote_game_module_info").ParseTo(&result1).Run()
		convey.So(err1, convey.ShouldBeNil)
		convey.So(result1.SubTitle.SubTitleText, convey.ShouldEqual, "游戏名为")

		dataset2 := map[string]any{
			"Request": nil,
		}
		result2 := PromoteGameModuleInfo{}
		err2 := engine.WithDataset(dataset2).WithCustomEntry("promote_game_module_info").ParseTo(&result2).Run()
		convey.So(err2, convey.ShouldBeNil)
	})
}

func TestConfigEngine_Run_ExpressionInValue(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"just_test": {
				"sub_title": {
					"sub_title_text": {
						"${Request.Scene == 1}": "${GetName(DATASET)}",
						"${Request.Scene == 2}": "你好${Echo}，测试不存在字段：${NotExist.Var}，测试不存在函数：${NotExistFunction()}，测试List：${GenerateList()}，测试Object：${GenerateObject()}，测试Bool：${GenerateBool()}，测试Int：${GenerateInt()}，测试String：${GenerateString()}",
						"@default": "???"
					}
				}
			}
		}
		`
		RegisterFunction("test_key", GetName)
		RegisterFunction("test_key", GenerateList)
		RegisterFunction("test_key", GenerateObject)
		RegisterFunction("test_key", GenerateBool)
		RegisterFunction("test_key", GenerateInt)
		RegisterFunction("test_key", GenerateString)
		engine, _ := GetJSONTemplateEngine(context.Background(), "test_key", configJSON)

		{
			dataset := map[string]any{
				"Request": map[string]any{
					"Scene": 2,
				},
				"Name": "Linux",
				"Echo": "HaHa",
			}
			result := PromoteGameModuleInfo{}
			err := engine.WithDataset(dataset).WithCustomEntry("just_test").ParseTo(&result).Run()
			convey.So(err, convey.ShouldBeNil)
			convey.So(result.SubTitle.SubTitleText, convey.ShouldEqual, "你好HaHa，测试不存在字段：，测试不存在函数：，测试List：，测试Object：，测试Bool：true，测试Int：1，测试String：this is string")
		}
	})
}

func GetName(params ...any) (any, error) {
	superDataset := params[0].(map[string]any)
	return superDataset["Name"], nil
}

func GenerateList(params ...any) (any, error) {
	return []int{1, 2, 3, 4}, nil
}

func GenerateObject(params ...any) (any, error) {
	return map[string]any{
		"Name": "1",
	}, nil
}

func GenerateBool(params ...any) (any, error) {
	return true, nil
}

func GenerateInt(params ...any) (any, error) {
	return 1, nil
}

func GenerateString(params ...any) (any, error) {
	return "this is string", nil
}

func TestConfigEngine_Run_MainEntry(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"_main_": {
				"sub_title": {
					"sub_title_text": {
						"${Request.Scene == 1}": "123123",
						"@default": "???"
					}
				}
			}
		}
		`
		engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)
		dataset := map[string]any{}

		result := PromoteGameModuleInfo{}
		_ = engine.WithDataset(dataset).ParseTo(&result).Run()
		convey.So(result.SubTitle.SubTitleText, convey.ShouldEqual, "???")
	})
}

func TestConfigEngine_Run_calculateExpressionByExps(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"_main_": {
				"title": {
					"@default": "${GetAInt()} + ${GetAFloat()} + ${GetABool()} + ${GetAStr()}"
				}
			}
		}
		`

		RegisterFunction("Test", GetAInt)
		RegisterFunction("Test", GetAFloat)
		RegisterFunction("Test", GetABool)
		RegisterFunction("Test", GetAStr)

		result := PromoteGameModuleInfo{}
		engine, _ := GetJSONTemplateEngine(context.Background(), "Test", configJSON)
		_ = engine.ParseTo(&result).Run()
		convey.So(result.Title, convey.ShouldEqual, "123 + 123.1 + true + abc")
	})
}

func GetAInt() int64 {
	return 123
}

func GetAFloat() float64 {
	return 123.1
}

func GetABool() bool {
	return true
}

func GetAStr() string {
	return "abc"
}

func TestConfigEngine_Run_BuiltInDate(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		configJSON := `
		{
			"_main_": {
				"title": {
					"@default": "${ToString(now().Year())}"
				}
			}
		}
		`

		result := PromoteGameModuleInfo{}
		engine, _ := GetJSONTemplateEngine(context.Background(), "Test", configJSON)
		_ = engine.ParseTo(&result).Run()
		convey.So(result.Title, convey.ShouldEqual, time.Now().Format("2006"))
	})
}

func TestConfigEngine_Run_BoolPtrDataset(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		{
			configJSON := `
			{
				"_main_": {
					"icon_url": "http",
					"title": {
						"@default": "${IsBoolPtr == true ? '123' : 'abc'}"
					},
					"sub_title": {
						"sub_title_text":{
							"@default": "${ToBool(Env.IsBoolPtr) ? '123' : 'abc'}"
						}
					}
				}
			}
			`

			type Env struct {
				IsBoolPtr *bool
			}

			isTestTrue := true
			isTestFalse := false

			env := map[string]any{
				"IsBoolPtr": &isTestTrue,
				"Env": &Env{
					IsBoolPtr: &isTestFalse,
				},
			}

			result := PromoteGameModuleInfo{}
			engine, _ := GetJSONTemplateEngine(context.Background(), "BoolPtrDataset", configJSON)
			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.IconURL, convey.ShouldEqual, "http")
			convey.So(result.Title, convey.ShouldEqual, "123")
			convey.So(result.SubTitle.SubTitleText, convey.ShouldEqual, "abc")
		}

		{
			configJSON := `
			{
				"_main_": {
					"title": {
						"@default": "${IsIntPtr == 1 ? 'webcast' : 'game'}"
					},
					"sub_title": {
						"sub_title_text":{
							"@default": "${Env.IsIntPtr != 1 ? 'webcast' : 'game'}"
						}
					}
				}
			}
			`

			type Env struct {
				IsIntPtr *int
			}

			IsIntPtr := 1

			env := map[string]any{
				"IsIntPtr": &IsIntPtr,
				"Env": &Env{
					IsIntPtr: &IsIntPtr,
				},
			}

			result := PromoteGameModuleInfo{}
			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)
			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "webcast")
			convey.So(result.SubTitle.SubTitleText, convey.ShouldEqual, "game")
		}
	})
}

func TestConfigEngine_Run_ObjectAndKV(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		{
			configJSON := `
			{
				"_main_": {
					"title": {
						"@default": "${toJSON(Object( KV('name', Data.Name), KV('age', 30), KV('object', Object( KV('ha', 'ha') ) ) ))}"
					}
				}
			}
			`

			result := PromoteGameModuleInfo{}
			RegisterBuiltInFunction(Object)
			RegisterBuiltInFunction(KV)

			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

			env := map[string]any{
				"Data": map[string]any{
					"Name": ptr.Of("John"),
				},
			}

			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqualJSON, "{\"age\":30,\"name\":\"John\",\"object\":{\"ha\":\"ha\"}}")
		}
	})

}

func TestConfigEngine_Run_Use(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		{
			configJSON := `
			{
				"_main_": {
					"title": {
						"@default": "${toJSON(Use('test'))}"
					}
				},
				"test": {
					"@default": {
						"name": "John",
						"age": "${Age}"
					}
				}
			}
			`

			result := PromoteGameModuleInfo{}

			env := map[string]any{
				"Age": 30,
			}

			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqualJSON, "{\"age\":30,\"name\":\"John\"}")
		}

		{
			configJSON := `
			{
				"_main_": {
					"title": {
						"@default": "${ToString(Use('test.sub_test'))}"
					}
				},
				"test": {
					"sub_test": {
						"@default": "${Yes}"
					}
				}
			}
			`

			result := PromoteGameModuleInfo{}

			env := map[string]any{
				"Yes": true,
			}

			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "true")
		}

		{
			configJSON := `
			{
				"_main_": {
					"title": {	
						"@var": {
							"List1": "${map(Acts, Use('test', Var('Item', #) ) )}"
						},
						"@default": "${join(List1)}"
					}
				},
				"test": {
					"@default": "${Item + Base}"
				}
			}
			`

			result := PromoteGameModuleInfo{}
			env := map[string]any{
				"Base": "ok",
				"Acts": []string{
					"1", "2", "3",
				},
			}
			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)
			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "1ok2ok3ok")
		}

	})
}

func TestConfigEngine_Run_ReturnAndDo(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		{
			configJSON := `
			{
				"_main_": {
					"@do": "${Dec(Age, 5)}",
					"@do": {
						"${Age > 1000}": "${Dec(Age, 1000)}",
						"${Age < 1000}": "${Inc(Age, 1)}",
						"${Age < 1000}": "${Inc(Age, 1)}",
						"${true}": "${Inc(Age, 1)}"
					},
					"title": {
						"@default": {
							"@do": [
								"${Inc(Age)}",
								"${Dec(Age)}",
								"${Inc(Age, 20)}"
							],
							"@do": [
								"${Dec(Age, 10)}"
							],
							"@do": "${Dec(Age, 5) && Inc(Age, 5)}",
							"@do": "${Dec(ABool) && Inc(Age, 2)}",
							"@do": "${Dec(NotPtrAge, 10)}",
							"@return": "haha",
							"@do": "${Dec(Age, 100)}"
						}
					},
					"sub_title": {
						"sub_title_text":{
							"${Age in [24]}": "haha",
							"@default": {
								"@return": "ok"
							}
						}
					}
				}
			}
			`

			result := PromoteGameModuleInfo{}

			age := 18
			env := map[string]any{
				"Age":       &age,
				"ABool":     false,
				"NotPtrAge": 18,
			}

			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "haha")
			convey.So(*env["Age"].(*int), convey.ShouldEqual, 24)
			convey.So(env["ABool"], convey.ShouldEqual, false)
			convey.So(env["NotPtrAge"], convey.ShouldEqual, 18)
			convey.So(result.SubTitle.SubTitleText, convey.ShouldEqual, "haha")
		}

		{
			configJSON := `
			{
				"_main_": {
					"title": {
						"@default": {
							"@var": {
								"Num": "${Int64Ptr(0)}"
							},
							"DO": "${Inc(Num)}",
							"@return": "${'haha' + ToString(Num)}"
						}
					}
				}
			}
			`

			result := PromoteGameModuleInfo{}

			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

			_ = engine.ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "haha")
		}

	})
}

func TestConfigEngine_Run_CreateVar(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		{
			configJSON := `
			{
				"_main_": {
					"title": {	
						"@var": {
							"A": 18,
							"B": "${A + 2}"
						},
						"@default": "${string(A)}"
					},
					"sub_title": {
						"sub_title_text": {
							"@default": "${string(B)}"
						}
					}
				}
			}
			`

			result := PromoteGameModuleInfo{}
			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)
			_ = engine.ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "18")
			convey.So(result.SubTitle.SubTitleText, convey.ShouldEqual, "20")
		}

		{
			configJSON := `
			{
				"_main_": {
					"title": {	
						"@var": {
							"P": "${Persons}"
						},
						"@default": "${join(P)}"
					},
					"sub_title": {
						"sub_title_text": {
							"@default": {
								"@var": {
									"P": "${concat(P, ['sim'])}"
								},
								"@return": "${join(P)}"
							}
						}
					}
				}
			}
			`

			result := PromoteGameModuleInfo{}
			env := map[string]any{
				"Persons": []string{"tom", "jack"},
			}
			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "tomjack")
			convey.So(result.SubTitle.SubTitleText, convey.ShouldEqual, "tomjacksim")
		}

		{
			configJSON := `
			{
				"_main_": {
					"title": {	
						"@var": {
							"A": "123"
						},
						"@default": "${A + Use('test')}"
					}
				},
				"test": {
					"@var": {
						"A": "456"
					},
					"@default": "${A}"
				}
			}
			`

			result := PromoteGameModuleInfo{}
			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)

			_ = engine.ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "123456")
		}
	})
}

func TestConfigEngine_Run_VarNestedExpression(t *testing.T) {
	convey.Convey("Test GetJSONTemplateEngine", t, func() {
		{
			configJSON := `
			{
				"_main_": {
					"title": {	
						"@var": {
							"Skd": "abc",
							"${ATrue}": {
								"Tek": "123",
								"${ATrue}": {
									"Fjs": "123"
								}
							},
							"${AFalse}": {
								"Sek": "456"
							},
							"${Error}": 123,
							"Vas": "233",
							"Abc": "haha"
						},
						"@default": "${Use('test', Var('U3', 'uuu')) + Skd + Tek + Fjs + ToString(Sek) + Vas + Abc}"
					}
				},
				"test": {
					"@var": {
						"SSR": "ssr"
					},
					"@default": "${U3}"
				}
			}
			`

			result := PromoteGameModuleInfo{}
			env := map[string]any{
				"ATrue":  true,
				"AFalse": false,
			}
			engine, _ := GetJSONTemplateEngine(context.Background(), time.Now().String(), configJSON)
			_ = engine.WithDataset(env).ParseTo(&result).Run()
			convey.So(result.Title, convey.ShouldEqual, "uuuabc123123233haha")
			convey.So(env["SSR"], convey.ShouldBeNil)
			convey.So(env["Sek"], convey.ShouldBeNil)
			convey.So(env["Skd"], convey.ShouldBeNil)
			convey.So(env["Abc"], convey.ShouldBeNil)
		}
	})
}
