package engine

import (
	"github.com/cassius0924/jtgo/ds"
	"github.com/liyue201/gostl/ds/deque"
	"github.com/tidwall/gjson"
)

// PreCompileFrame represents a frame for iterative precompilation
type PreCompileFrame struct {
	Node        gjson.Result
	SubNodeIter *deque.DequeIterator[*ds.Pair[gjson.Result, gjson.Result]]
}

// flattenNodeForPreCompile flattens a gjson.Result node into a deque of key-value pairs
func flattenNodeForPreCompile(node gjson.Result) *deque.Deque[*ds.Pair[gjson.Result, gjson.Result]] {
	var result = ds.NewDeque[*ds.Pair[gjson.Result, gjson.Result]]()
	node.ForEach(func(k, v gjson.Result) bool {
		result.PushBack(ds.MakePair(k, v))
		return true
	})
	return result
}
