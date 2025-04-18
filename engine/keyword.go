package engine

import (
	"strings"

	"github.com/samber/lo"
)

type Keyword string

const (
	// 关键词，全小写
	KeywordDefault  Keyword = "@default"
	KeywordDo       Keyword = "@do"
	KeywordReturn   Keyword = "@return"
	KeywordVar      Keyword = "@var"
	KeywordFor      Keyword = "@for"
	KeywordIf       Keyword = "@if"
	KeywordElif     Keyword = "@elif"
	KeywordElse     Keyword = "@else"
	KeywordContinue Keyword = "@continue"
	KeywordComment  Keyword = "@cmt"
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
		KeywordElif,
		KeywordElse,
		KeywordContinue,
		KeywordComment,
	}

	KeywordDelimiters = []rune{
		' ',
		'\t',
		':',
	}
)

// detectKeyword 检测关键词
func detectKeyword(input string) (Keyword, string) {
	// 不区分大小写
	lower := strings.ToLower(strings.TrimSpace(input))
	for _, kw := range Keywords {
		if strings.HasPrefix(lower, string(kw)) {
			statement := input[len(kw):]
			// 关键词后面必须是空白、冒号、或结束
			if statement == "" || lo.Contains(KeywordDelimiters, rune(statement[0])) {
				return kw, strings.TrimSpace(statement)
			}
		}
	}
	return "", ""
}

func isKeyword(input string, keyword Keyword) bool {
	kw, _ := detectKeyword(input)
	return kw == keyword
}

func IsAnyKeyword(input string) bool {
	return lo.Contains(Keywords, Keyword(strings.ToLower(input)))
}

func IsDoKeyword(input string) bool {
	return isKeyword(input, KeywordDo)
}

func IsReturnKeyword(input string) bool {
	return isKeyword(input, KeywordReturn)
}

func IsVarKeyword(input string) bool {
	return isKeyword(input, KeywordVar)
}

func IsDefaultKeyword(input string) bool {
	return isKeyword(input, KeywordDefault)
}

func IsForStatement(input string) bool {
	return isKeyword(input, KeywordFor)
}

func IsIfStatement(input string) bool {
	return isKeyword(input, KeywordIf)
}

func IsElifStatement(input string) bool {
	return isKeyword(input, KeywordElif)
}

func IsElseKeyword(input string) bool {
	return isKeyword(input, KeywordElse)
}

func IsContinueKeyword(input string) bool {
	return isKeyword(input, KeywordContinue)
}

func IsCommentKeyword(input string) bool {
	return isKeyword(input, KeywordComment)
}
