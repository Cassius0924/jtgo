package flags

type NodeFlag int64

const (
	// NodeFlagGeneralObjectItem 普通对象字段
	NodeFlagGeneralObjectItem NodeFlag = 1 << iota
	// NodeFlagGeneralArrayItem 普通数组元素
	NodeFlagGeneralArrayItem
	// NodeFlagKeywordIf 关键字if
	NodeFlagKeywordIf
	// NodeFlagKeywordElif 关键字elif
	NodeFlagKeywordElif
	// NodeFlagKeywordElse 关键字else
	NodeFlagKeywordElse
	// NodeFlagKeywordFor 关键字for
	NodeFlagKeywordFor
	// NodeFlagKeywordVar 关键字var
	NodeFlagKeywordVar
	// NodeFlagKeywordExec 关键字exec
	NodeFlagKeywordExec
	// NodeFlagKeywordReturn 关键字return
	NodeFlagKeywordReturn
	// NodeFlagKeywordCmt 关键字cmt
	NodeFlagKeywordCmt
	// NodeFlagNodeNonObjectOrArray 非对象或数组节点
	NodeFlagNodeNonObjectOrArray
	// NodeFlagExecOnce 只执行一次
	NodeFlagExecOnce
	// NodeFlagLoop 处理循环时
	NodeFlagLooping
)
