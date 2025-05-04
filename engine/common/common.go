package common

import (
	"strings"

	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/liyue201/gostl/ds/deque"
)

// NormalizeFieldName 规范化字段名称
func NormalizeFieldName(fieldName string) string {
	// 去掉头尾空格
	return strings.TrimSpace(fieldName)
}

// FlattenNode 将 model.TNode 节点的所有子节点扁平化为一个双端队列
// TODO: 改成对象池
func FlattenNode(node *model.TNode) *deque.Deque[*ds.Pair[model.TNode, model.TNode]] {
	var result = ds.NewDeque[*ds.Pair[model.TNode, model.TNode]]()
	node.ForEach(func(k, v model.TNode) bool {
		result.PushBack(ds.MakePair(k, v))
		return true
	})
	return result
}
