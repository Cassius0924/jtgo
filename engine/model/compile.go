package model

import (
	"fmt"

	"github.com/cassius0924/jtgo/ds"
	"github.com/liyue201/gostl/ds/vector"
)

type CompileFrame struct {
	Node        *TNode
	SubNodeIter *vector.VectorIterator[*ds.Pair[*TNode, *TNode]]
	Path        string
}

// ExtractSubNodePair 提取子节点对
func (f *CompileFrame) ExtractSubNodePair() (*TNode, *TNode) {
	subNodePair := f.SubNodeIter.Value()
	subNodeField, subNode := subNodePair.First, subNodePair.Second
	return subNodeField, subNode
}

// BuildNodePath 构建节点路径
func (f *CompileFrame) BuildNodePath(fieldName string) string {
	// TODO: 性能优化
	if f.Path == "" {
		return fmt.Sprintf("%d_%s", f.SubNodeIter.Position(), fieldName)
	}
	return fmt.Sprintf("%s.%d_%s", f.Path, f.SubNodeIter.Position(), fieldName)
}
