package keywords

import (
	"strings"

	"github.com/samber/lo"
)

type Keyword string

const (
	// 关键词，全小写
	KeywordDefault  Keyword = "@default"
	KeywordExec     Keyword = "@exec"
	KeywordReturn   Keyword = "@return"
	KeywordVar      Keyword = "@var"
	KeywordFor      Keyword = "@for"
	KeywordIf       Keyword = "@if"
	KeywordElif     Keyword = "@elif"
	KeywordElse     Keyword = "@else"
	KeywordContinue Keyword = "@continue"
	KeywordCmt      Keyword = "@cmt"
)

var (
	// Keywords 关键词列表
	KeywordList = []Keyword{
		KeywordDefault,
		KeywordExec,
		KeywordReturn,
		KeywordVar,
		KeywordFor,
		KeywordIf,
		KeywordElif,
		KeywordElse,
		KeywordContinue,
		KeywordCmt,
	}

	KeywordDelimiters = []rune{
		' ',
		'\t',
	}
)

// detectKeyword 检测关键词
func DetectKeyword(input string) (Keyword, string) {
	// 不区分大小写
	lower := strings.ToLower(strings.TrimSpace(input))
	for _, kw := range KeywordList {
		if strings.HasPrefix(lower, string(kw)) {
			statement := input[len(kw):]
			// 关键词后面必须是空白
			if statement == "" || lo.Contains(KeywordDelimiters, rune(statement[0])) {
				return kw, strings.TrimSpace(statement)
			}
		}
	}
	return "", ""
}

func IsKeyword(input string, keyword Keyword) bool {
	kw, _ := DetectKeyword(input)
	return kw == keyword
}

func IsAnyKeyword(input string) bool {
	kw, _ := DetectKeyword(input)
	return kw != ""
}

func IsExecKeyword(input string) bool {
	return IsKeyword(input, KeywordExec)
}

func IsReturnKeyword(input string) bool {
	return IsKeyword(input, KeywordReturn)
}

func IsVarKeyword(input string) bool {
	return IsKeyword(input, KeywordVar)
}

func IsDefaultKeyword(input string) bool {
	return IsKeyword(input, KeywordDefault)
}

func IsForStatement(input string) bool {
	return IsKeyword(input, KeywordFor)
}

func IsIfStatement(input string) bool {
	return IsKeyword(input, KeywordIf)
}

func IsElifStatement(input string) bool {
	return IsKeyword(input, KeywordElif)
}

func IsElseKeyword(input string) bool {
	return IsKeyword(input, KeywordElse)
}

func IsContinueKeyword(input string) bool {
	return IsKeyword(input, KeywordContinue)
}

func IsCmtKeyword(input string) bool {
	return IsKeyword(input, KeywordCmt)
}
