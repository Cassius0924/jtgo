package common

import (
	"strings"

	"github.com/cassius0924/jtgo/ds"
	"github.com/liyue201/gostl/ds/deque"
	"github.com/tidwall/gjson"
)

// NormalizeFieldName 规范化字段名称
func NormalizeFieldName(fieldName string) string {
	// 去掉头尾空格
	return strings.TrimSpace(fieldName)
}

// FlattenNode 将 gjson.Result 节点的所有子节点扁平化为一个双端队列
// TODO: 改成对象池
func FlattenNode(node *gjson.Result) *deque.Deque[*ds.Pair[gjson.Result, gjson.Result]] {
	var result = ds.NewDeque[*ds.Pair[gjson.Result, gjson.Result]]()
	node.ForEach(func(k, v gjson.Result) bool {
		result.PushBack(ds.MakePair(k, v))
		return true
	})
	return result
}
