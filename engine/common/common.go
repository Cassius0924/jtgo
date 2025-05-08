package common

import (
	"strings"

	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/engine/model"
	"github.com/liyue201/gostl/ds/vector"
	"github.com/tidwall/gjson"
)

// NormalizeFieldName 规范化字段名称
func NormalizeFieldName(fieldName string) string {
	// 去掉头尾空格
	return strings.TrimSpace(fieldName)
}

// FlattenNode 将模板节点的所有子节点扁平化为一个动态数组
// TODO: 改成对象池，或改至编译期完成
func FlattenNode(node *model.TNode) *vector.Vector[*ds.Pair[*model.TNode, *model.TNode]] {
	var subNodes = ds.NewVectorWithCap[*ds.Pair[*model.TNode, *model.TNode]](len(node.Map()))
	node.ForEach(func(key, val gjson.Result) bool {
		subNodes.PushBack(ds.MakePair(model.NewTNode(key), model.NewTNode(val)))
		return true
	})
	return subNodes
}
