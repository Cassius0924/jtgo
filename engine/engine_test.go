package engine

import (
	"context"
	"testing"

	"github.com/smartystreets/goconvey/convey"
)

type SubTestObj struct {
	Name string `json:"name"`
}

type TestObj struct {
	Name        string        `json:"name"`
	SubTest     *SubTestObj   `json:"sub_test"`
	Button      Button        `json:"button"`
	IconURL     string        `json:"icon_url"`
	SubTestList []*SubTestObj `json:"button_list"`

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

func TestEngine_ConditionalIf(t *testing.T) {
	convey.Convey("TestEngine_ConditionalIf", t, func() {
		tmpl := `
{
	"_main_": {
		"name": {
			"@if a": "hello"
		}
	}
}
	`
		engine, err := GetJSONTemplateEngine(context.Background(), "TestEngine_ConditionalIf", tmpl)
		convey.So(err, convey.ShouldBeNil)

		dataset := map[string]any{
			"a": true,
		}

		target := &TestObj{}
		_ = engine.WithDataset(dataset).ParseTo(target).Run()

		convey.So(target.Name, convey.ShouldEqual, "hello")
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
			_ = engine.WithDataset(dataset).ParseTo(target).Run()

			convey.So(target.Name, convey.ShouldEqual, "hello")
		}
	})
}

func TestEngine_ConditionalIfNested(t *testing.T) {
	convey.Convey("TestEngine_ConditionalIfNested", t, func() {
		tmpl := `
{
	"_main_": {
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

		target := &TestObj{}
		_ = engine.WithDataset(dataset).ParseTo(target).Run()

		convey.So(target.Name, convey.ShouldEqual, "hello")
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
		_ = engine.WithDataset(dataset).ParseTo(target).Run()

		convey.So(target.Name, convey.ShouldEqual, "hello")
	})
}
