package engine

import "strings"

type Keyword string

const (
	// 关键词
	KeywordDefualt Keyword = "default"
	KeywordDo      Keyword = "do"
	KeywordReturn  Keyword = "return"
	KeywordVar     Keyword = "var"
	KeywordFor     Keyword = "for"
	KeywordIf      Keyword = "if"
	KeywordElse    Keyword = "else"
)

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
	return strings.ToLower(input) == string(KeywordDefualt)
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
