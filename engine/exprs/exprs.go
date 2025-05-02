package exprs

import (
	"context"
	"log/slog"
	"regexp"
	"strings"

	"github.com/cassius0924/jtgo/engine/model"
	"github.com/samber/lo"
)

const (
	// ${Expression}的前缀
	expressionFlagPrefix = "${"
	// ${Expression}的后缀
	expressionFlagSuffix = "}"

	expressionFormat = expressionFlagPrefix + "%s" + expressionFlagSuffix

	expressionRegexp = `\$\{([^{}]*(?:\{[^{}]*\}[^{}]*)*)\}`
)

var (
	// 匹配${Expression}的正则表达式
	expressionRe = regexp.MustCompile(expressionRegexp)

	// 循环语句的分隔符
	loopStatementDelimiters = []string{
		" in ",
		":=",
	}
)

// ExtractExpression 将input字符串的所有variable符号去除，只留下表达式，strings库替换比正则替换更快
func ExtractExpression(input string) (string, bool) {
	result := strings.ReplaceAll(input, expressionFlagPrefix, "")
	if result == input {
		return result, false
	}
	result = strings.ReplaceAll(result, expressionFlagSuffix, "")
	if result == input {
		return result, false
	}

	return result, true
}

// ExtractAllExpressions 提取所有表达式
func ExtractAllExpressions(input string) []string {
	matches := expressionRe.FindAllStringSubmatch(input, -1)
	var result []string
	for _, match := range matches {
		if len(match) > 1 {
			result = append(result, match[1])
		}
	}
	return result
}

// ParseLoopStatement 解析循环语句
func ParseLoopStatement(ctx context.Context, statement string) (*model.LoopMeta, bool) {
	// statement like "index,val:=list"、"key,val := map"、"_,val in list"、"key,val := map"
	statement = strings.TrimSpace(statement)

	var (
		keyAndValue        string
		key, value, object string
		found              bool
	)
	// 拆开kv和object
	for _, delimiter := range loopStatementDelimiters {
		keyAndValue, object, found = strings.Cut(statement, delimiter)
		if found {
			break
		}
	}
	if !found {
		slog.ErrorContext(ctx, "[JSONTemplateEngine.parseLoopStatement] parse loop statement error, please check the statement", "statement", statement)
		return nil, false
	}

	// 如果循环对象为空则报错
	object = strings.TrimSpace(object)
	if object == "" {
		slog.ErrorContext(ctx, "[JSONTemplateEngine.parseLoopStatement] parse loop statement error, object is empty", "statement", statement)
		return nil, false
	}

	keyAndValue = strings.ReplaceAll(keyAndValue, " ", "")

	// 解析key和value，没有两个值也需要报错
	key, value, found = strings.Cut(keyAndValue, ",")
	if !found {
		slog.ErrorContext(ctx, "[JSONTemplateEngine.parseLoopStatement] parse key and value error, not found comma", "keyAndValue", keyAndValue)
		return nil, false
	}

	// 如果是空白占位符，表示不需要key或value
	key = lo.Ternary(key == "_", "", key)
	value = lo.Ternary(value == "_", "", value)

	return &model.LoopMeta{
		Key:    key,
		Value:  value,
		Object: object,
	}, true
}
