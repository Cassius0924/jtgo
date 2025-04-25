package util

import (
	"fmt"
	"strings"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

// 测试用的结构体
type TestStruct struct{}

// 测试用的方法
func (t TestStruct) TestMethod() {}

// 测试用的普通函数
func testFunction() {}

func TestGetFunctionName(t *testing.T) {
	convey.Convey("测试GetFunctionName函数", t, func() {
		convey.Convey("测试普通函数", func() {
			name := GetFunctionName(testFunction)
			convey.So(name, convey.ShouldEqual, "testFunction")
		})

		convey.Convey("测试结构体方法", func() {
			ts := TestStruct{}
			name := GetFunctionName(ts.TestMethod)
			convey.So(name, convey.ShouldEqual, "TestMethod")
		})

		convey.Convey("测试标准库函数", func() {
			name := GetFunctionName(strings.Contains)
			convey.So(name, convey.ShouldEqual, "Contains")
		})

		convey.Convey("测试嵌套包中的函数", func() {
			name := GetFunctionName(fmt.Sprintf)
			convey.So(name, convey.ShouldEqual, "Sprintf")
		})

		convey.Convey("测试nil值处理", func() {
			defer func() {
				r := recover()
				convey.So(r, convey.ShouldNotBeNil) // 应该发生panic
			}()
			_ = GetFunctionName(nil)
		})
	})
}
