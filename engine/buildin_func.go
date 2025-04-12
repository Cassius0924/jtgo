package engine

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"reflect"
	"strconv"
	"time"

	"github.com/cassius0924/jtgo/util"
	"github.com/samber/lo"
)

func init() {
	RegisterBuiltInFunction(ToInt)
	RegisterBuiltInFunction(ToString)
	RegisterBuiltInFunction(ToBool)
	RegisterBuiltInFunction(Object)
	RegisterBuiltInFunction(KV)
	RegisterBuiltInFunction(Use)
	RegisterBuiltInFunction(Var)
	RegisterBuiltInFunction(Inc)
	RegisterBuiltInFunction(Dec)
	RegisterBuiltInFunction(Set)
	RegisterBuiltInFunction(CompactSlice)

	// 时间日期函数
	RegisterBuiltInFunction(Unix)
	RegisterBuiltInFunction(time.Date)
	RegisterBuiltInFunction(DateYMD)
	RegisterBuiltInFunction(DateYMDHMS)

	// URL 操作相关函数
	RegisterBuiltInFunction(EncodeURL)
	RegisterBuiltInFunction(DecodeURL)
	RegisterBuiltInFunction(DeleteURLParams)
	RegisterBuiltInFunction(AddURLParams)
	RegisterBuiltInFunction(GetURLParamValue)

	// 类型转化函数
	RegisterBuiltInFunction(strconv.Atoi)

	// 日志打印函数
	RegisterBuiltInFunction(LogInfo)
	RegisterBuiltInFunction(LogWarn)
	RegisterBuiltInFunction(LogError)
}

// LogInfo 用于打印 info 级别日志
func LogInfo(ctx context.Context, str string, v ...any) bool {
	slog.InfoContext(ctx, str, v...)
	return true
}

// LogWarn 用于打印 warn 级别日志
func LogWarn(ctx context.Context, str string, v ...any) bool {
	slog.WarnContext(ctx, str, v...)
	return true
}

// LogError 用于打印 error 级别日志
func LogError(ctx context.Context, str string, v ...any) bool {
	slog.ErrorContext(ctx, str, v...)
	return true
}

// ToInt 用于将指针类型或普通 type int 类型转为 int
func ToInt(ctx context.Context, value any) any {
	if value == nil {
		slog.ErrorContext(ctx, "[ToInt] value is nil")
		return 0
	}
	v := reflect.ValueOf(value)

	// 检查是否为基于 int 的类型
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int()
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return int64(v.Uint())
	case reflect.Float64, reflect.Float32:
		return int64(v.Float())
	case reflect.String:
		i, _ := strconv.ParseInt(v.String(), 10, 64)
		return i
	case reflect.Invalid:
		slog.ErrorContext(ctx, "[ToInt] invalid type", "type", v, "value", value)
		return 0
	default:
		slog.ErrorContext(ctx, "[ToInt] unsupported type", "type", v.Type(), "value", value)
		return 0
	}
}

// ToString 用于将基本类型转为 string
func ToString(ctx context.Context, value any) string {
	if value == nil {
		slog.ErrorContext(ctx, "[ToString] value is nil")
		return ""
	}
	v := reflect.ValueOf(value)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return fmt.Sprintf("%f", v.Float())
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Invalid:
		slog.ErrorContext(ctx, "[ToString] invalid type", "type", v, "value", value)
		return ""
	default:
		slog.ErrorContext(ctx, "[ToString] unsupported type", "type", v.Type(), "value", value)
		return ""
	}
}

func ToBool(ctx context.Context, value any) bool {
	if value == nil {
		slog.ErrorContext(ctx, "[ToBool] value is nil")
		return false
	}
	v := reflect.ValueOf(value)

	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() != 0
	case reflect.Invalid:
		slog.ErrorContext(ctx, "[ToBool] invalid type", "type", v, "value", value)
		return false
	default:
		slog.InfoContext(ctx, "[ToBool] unsupported type", "type", v.Type(), "value", value)
		return false
	}
}

// Unix 时间戳转换为日期
func Unix(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

// DateYMD 通过年月日创建日期
func DateYMD(year, month, day int) time.Time {
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local)
}

// DateYMDHMS 通过年月日时分秒创建日期
func DateYMDHMS(year, month, day, hour, min, sec int) time.Time {
	return time.Date(year, time.Month(month), day, hour, min, sec, 0, time.Local)
}

type KeyValue struct {
	key   string
	value any
}

// Object 用于创建 map[string]any
func Object(kvs ...KeyValue) map[string]any {
	res := make(map[string]any)
	for _, kv := range kvs {
		res[kv.key] = kv.value
	}
	return res
}

// KV 用于创建 KeyValue
func KV(key string, value any) KeyValue {
	return KeyValue{key: key, value: value}
}

// Use 用于引用一个 Object，计算并返回该 Object 的值
func Use(ctx context.Context, path string, variables ...*Variable) any {
	engine := GetJSONTemplateEngineFromContext(ctx)
	// 数据集会继承
	dataset := engine.GetDataset()
	slog.InfoContext(ctx, "[configengine.Use](trace) start", "path", path)
	// 将变量放入数据集
	for _, v := range variables {
		if dataset == nil {
			dataset = make(map[string]any)
		}
		dataset[v.name] = v.value
		slog.InfoContext(ctx, fmt.Sprintf("[configengine.Use](trace) create variable,\nkey = %s,\nvalue = %s", v.name, util.GenerateStructFormatedString(v.value)))
	}
	// 从引擎中获取对象
	var result any
	// 这里需要保持状态，因为可能会有多个 Use 在一个 for 循环中，如果不保持状态，会导致第2个及其以后的 Use 的数据集被清空
	_ = engine.WithDataset(dataset).WithCustomEntry(path).ParseTo(&result).keepStatusRun()
	slog.InfoContext(ctx, "[configengine.Use](trace) end", "path", path, "result", result)
	return result
}

type Variable struct {
	name  string
	value any
}

func Var(name string, value any) *Variable {
	return &Variable{
		name:  name,
		value: value,
	}
}

// Inc 自增函数，需要传入指针类型数值
func Inc(ctx context.Context, input any, addNum ...int) bool {
	if input == nil {
		slog.InfoContext(ctx, "[Inc] input is nil")
		return false
	}
	v := reflect.ValueOf(input)
	if v.Kind() != reflect.Ptr {
		slog.ErrorContext(ctx, "[Inc] input should be a pointer", "type", v.Type())
		return false
	}
	v = v.Elem()
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if len(addNum) == 0 {
			v.SetInt(v.Int() + 1)
		} else {
			v.SetInt(v.Int() + int64(addNum[0]))
		}
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if len(addNum) == 0 {
			v.SetUint(v.Uint() + 1)
		} else {
			v.SetUint(v.Uint() + uint64(addNum[0]))
		}
		return true
	default:
		slog.ErrorContext(ctx, "[Inc] unsupported type", "type", v.Type())
		return false
	}
}

// Dec 自减函数，需要传入指针类型数值
func Dec(ctx context.Context, input any, decNum ...int) bool {
	if input == nil {
		slog.ErrorContext(ctx, "[Dec] input is nil")
		return false
	}
	v := reflect.ValueOf(input)
	if v.Kind() != reflect.Ptr {
		slog.ErrorContext(ctx, "[Dec] input should be a pointer", "type", v.Type())
		return false
	}
	v = v.Elem()
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if len(decNum) == 0 {
			v.SetInt(v.Int() - 1)
		} else {
			v.SetInt(v.Int() - int64(decNum[0]))
		}
		return true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if len(decNum) == 0 {
			v.SetUint(v.Uint() - 1)
		} else {
			v.SetUint(v.Uint() - uint64(decNum[0]))
		}
		return true
	default:
		slog.ErrorContext(ctx, "[Dec] unsupported type", "type", v.Type())
		return false
	}
}

// CompactSlice 用于去除数组中的 nil 值
func CompactSlice(slice []any) []any {
	result := lo.Compact(slice)
	return result
}

// Set 用于设置数据集中的值
func Set(ctx context.Context, key string, value any) bool {
	engine := GetJSONTemplateEngineFromContext(ctx)
	dataset := engine.GetDataset()
	if dataset == nil {
		dataset = make(map[string]any)
	}
	dataset[key] = value
	slog.InfoContext(ctx, fmt.Sprintf("[configengine.Set](trace) set value.\nkey = %s,\nvalue = %s", key, util.GenerateStructFormatedString(value)))
	return true
}

// DeleteURLParams 删除 URL 中的参数
func DeleteURLParams(ctx context.Context, inputURL string, keys ...string) string {
	u, err := url.Parse(inputURL)
	if err != nil {
		slog.ErrorContext(ctx, "[DeleteURLParams] parse url failed", "url", inputURL)
		return inputURL
	}
	q := u.Query()
	for _, key := range keys {
		q.Del(key)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// AddURLParams 添加 URL 参数
func AddURLParams(ctx context.Context, inputURL string, kvs ...KeyValue) string {
	u, err := url.Parse(inputURL)
	if err != nil {
		slog.ErrorContext(ctx, "[AddURLParams] parse url failed", "url", inputURL)
		return inputURL
	}
	q := u.Query()
	for _, kv := range kvs {
		q.Set(kv.key, fmt.Sprintf("%v", kv.value))
	}
	u.RawQuery = q.Encode()
	return u.String()
}

// EncodeURL 编码 URL
func EncodeURL(ctx context.Context, inputURL string) string {
	return url.QueryEscape(inputURL)
}

// DecodeURL 解码 URL
func DecodeURL(ctx context.Context, inputURL string) string {
	decodeURL, err := url.QueryUnescape(inputURL)
	if err != nil {
		slog.ErrorContext(ctx, "[DecodeURL] decode url failed", "url", inputURL)
	}
	return decodeURL
}

// GetURLParamValue 获取 URL 参数值
func GetURLParamValue(ctx context.Context, inputURL, key string) string {
	u, err := url.Parse(inputURL)
	if err != nil {
		slog.ErrorContext(ctx, "[GetURLParamValue] parse url failed", "url", inputURL)
		return ""
	}
	return u.Query().Get(key)
}
