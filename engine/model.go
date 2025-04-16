package engine

import (
	"github.com/cassius0924/jtgo/ds"
	"github.com/liyue201/gostl/ds/deque"
	"github.com/tidwall/gjson"
)

// ParseFrame 模板引擎解析帧
type ParseFrame struct {
	node       gjson.Result
	fieldName  string
	target     any
	curSubNode *deque.DequeIterator[*ds.Pair[gjson.Result, gjson.Result]]

	conditionalCtx *ConditionalContext
	loopCtx        *LoopContext
}

func (f *ParseFrame) isConditionalMatched() bool {
	if f == nil {
		return false
	}
	return f.conditionalCtx != nil && f.conditionalCtx.isMatched
}

// ConditionalContext 条件判断上下文
type ConditionalContext struct {
	resultValue *gjson.Result
	isMatched   bool
	groupNum    *ds.Counter // 当前条件组的序号
}

// LoopContext 循环上下文
type LoopContext struct {
	index     int           // 当前循环索引
	key       string        // 当前循环键
	value     any           // 当前循环值
	object    *gjson.Result // 当前循环对象
	continued bool          // 是否命中 continue
	localVars map[string]any
}
