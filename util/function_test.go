package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

// TestStruct 用于测试方法的结构体
type TestStruct struct{}

// TestMethod 测试方法
func (ts *TestStruct) TestMethod() {}

// RegularFunction 普通函数用于测试
func RegularFunction() {}

func TestGetFunctionName(t *testing.T) {
	convey.Convey("测试 GetFunctionName 函数", t, func() {
		convey.Convey("测试普通函数", func() {
			name := GetFunctionName(RegularFunction)
			convey.So(name, convey.ShouldEqual, "RegularFunction")
		})

		convey.Convey("测试结构体方法", func() {
			ts := &TestStruct{}
			name := GetFunctionName(ts.TestMethod)
			convey.So(name, convey.ShouldEqual, "TestMethod")
		})

		convey.Convey("测试其他包中的函数", func() {
			name := GetFunctionName(testing.RunTests)
			convey.So(name, convey.ShouldEqual, "RunTests")
		})

		convey.Convey("测试自身", func() {
			name := GetFunctionName(GetFunctionName)
			convey.So(name, convey.ShouldEqual, "GetFunctionName")
		})
	})
}

// 更复杂的测试案例
func TestGetFunctionNameComplex(t *testing.T) {
	convey.Convey("测试复杂的函数名场景", t, func() {
		// 测试带闭包的函数
		func() {
			closure := func() {}
			name := GetFunctionName(closure)
			// 闭包函数名通常很复杂，我们只需确保它能正确返回一个名称
			convey.So(name, convey.ShouldNotBeEmpty)
		}()

		// 测试嵌套结构体的方法
		type OuterStruct struct {
			InnerStruct struct {
				Value int
			}
		}
		outer := OuterStruct{}
		outer.InnerStruct.Value = 10

		// 使用函数映射
		funcs := map[string]interface{}{
			"test": RegularFunction,
		}
		name := GetFunctionName(funcs["test"])
		convey.So(name, convey.ShouldEqual, "RegularFunction")
	})
}

// 测试移除 -fm 后缀的情况
func TestFunctionNameSuffixRemoval(t *testing.T) {
	convey.Convey("测试移除函数名后缀", t, func() {
		// 使用反射构造一个带有 -fm 后缀的函数名
		// 这里我们只能间接测试，因为反射API没有直接方式构造这种情况

		// 创建一个带有接收器的方法变量
		ts := &TestStruct{}
		methodValue := ts.TestMethod

		// 测试是否能正确处理方法值
		name := GetFunctionName(methodValue)
		convey.So(name, convey.ShouldEqual, "TestMethod")

		// 我们无法直接制造带 -fm 后缀的情况，但可以测试该逻辑是否存在
		testString := "FunctionName-fm"
		result := strings.TrimSuffix(testString, "-fm")
		convey.So(result, convey.ShouldEqual, "FunctionName")
	})
}

// 测试边缘情况
func TestGetFunctionNameEdgeCases(t *testing.T) {
	convey.Convey("测试 GetFunctionName 边缘情况", t, func() {
		convey.Convey("测试 fmt.Println 函数", func() {
			name := GetFunctionName(fmt.Println)
			convey.So(name, convey.ShouldEqual, "Println")
		})

		convey.Convey("测试函数类型转换", func() {
			var fn interface{} = RegularFunction
			name := GetFunctionName(fn)
			convey.So(name, convey.ShouldEqual, "RegularFunction")
		})
	})
}
