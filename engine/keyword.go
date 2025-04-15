package engine

import "strings"

type Keyword string

const (
	// 关键词，全小写
	KeywordDefault Keyword = "@default"
	KeywordDo      Keyword = "@do"
	KeywordReturn  Keyword = "@return"
	KeywordVar     Keyword = "@var"
	KeywordFor     Keyword = "@for"
	KeywordIf      Keyword = "@if"
	KeywordElse    Keyword = "@else"
)

var (
	// Keywords 关键词列表
	Keywords = []Keyword{
		KeywordDefault,
		KeywordDo,
		KeywordReturn,
		KeywordVar,
		KeywordFor,
		KeywordIf,
		KeywordElse,
	}
)

// detectKeyword 检测关键词
func detectKeyword(input string) (Keyword, string) {
	// 不区分大小写
	input = strings.ToLower(strings.TrimSpace(input))
	for _, kw := range Keywords {
		extra, ok := strings.CutPrefix(input, string(kw))
		if ok {
			return kw, extra
		}
	}
	return "", ""
}

func IsKeywordDo(input string) bool {
	return strings.ToLower(input) == string(KeywordDo)
}

func IsKeywordReturn(input string) bool {
	return strings.ToLower(input) == string(KeywordReturn)
}

func IsKeywordVar(input string) bool {
	return strings.ToLower(input) == string(KeywordVar)
}

func IsKeywordDefault(input string) bool {
	return strings.ToLower(input) == string(KeywordDefault)
}

func IsKeywordFor(input string) bool {
	return strings.ToLower(input) == string(KeywordFor)
}
func IsKeywordIf(input string) bool {
	return strings.ToLower(input) == string(KeywordIf)
}

func IsKeywordElse(input string) bool {
	return strings.ToLower(input) == string(KeywordElse)
}
