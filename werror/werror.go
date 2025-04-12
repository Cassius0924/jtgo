package werror

import "errors"

var (
	// ErrPreCompileFail 模板引擎预编译失败
	ErrPreCompileFail = errors.New("JSON template engine pre compile failed")
	// ErrEntryNotFound 未找到模板入口
	ErrEntryNotFound = errors.New("entry of template not found")
	// ErrTemplateIsEmpty 模板为空
	ErrTemplateIsEmpty = errors.New("JSON template is empty")
	// ErrTemplateIDIsEmpty 模板ID为空
	ErrTemplateIDIsEmpty = errors.New("JSON template ID is empty")
	// ErrTargetIsNil 模板引擎解析目标为空
	ErrTargetIsNil = errors.New("target of JSON template engine is nil")
	// ErrParseToTargetFailed 模板引擎解析目标失败
	ErrParseToTargetFailed = errors.New("JSON template engine parse to target failed")
)

// 合并错误
func Join(err1, err2 error) error {
	return errors.Join(err1, err2)
}
