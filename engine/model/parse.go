package model

import (
	"fmt"

	"github.com/cassius0924/jtgo/util"
	"github.com/liyue201/gostl/ds/list/bidlist"
)

// ParseFrame 模板引擎解析帧
type ParseFrame struct {
	Node       *TNode         // 当前帧的节点
	FieldName  string         // 当前帧的字段名
	Target     any            // 父帧的解析结果，指向父帧的 Result
	Result     any            // 本帧的解析结果
	SharedMemo map[string]any // 帧与帧之间的共享数据的备忘录，类似一个全局变量
	Path       string         // 当前帧在模板中的路径

	SubNodeIter *NodeIter // 当前帧的子节点迭代器

	CondContextList *bidlist.List[*ConditionalContext] // 条件上下文链表，记录当前帧的所有条件上下文

	// CondContext *ConditionalContext // 当前帧的条件上下文信息
	// LoopContext *LoopContext        // 当前帧的循环上下文信息
}

// ExtractSubNodePair 提取子节点对
func (f *ParseFrame) ExtractSubNodePair() (*TNode, *TNode) {
	subNodePair := f.SubNodeIter.Value()
	subNodeField, subNode := subNodePair.First, subNodePair.Second
	return subNodeField, subNode
}

// LazyInitCondContextList 懒加载条件上下文链表
func (f *ParseFrame) LazyInitCondContextList() {
	if f.CondContextList == nil {
		f.CondContextList = bidlist.New[*ConditionalContext]()
	}
}

// IsConditionalMatched 是否匹配到了条件
func (f *ParseFrame) IsConditionalMatched() bool {
	if f.CondContextList == nil {
		return false
	}
	condContext := f.CondContextList.Back()
	return condContext != nil && condContext.IsMatched && !(f.SharedMemo["no_match"] == true)
}

// IsLoopDone 是否完成循环
func (f *ParseFrame) IsLoopDone() bool {
	return f.Node.LoopContext != nil && f.Node.LoopContext.IsDone()
}

// ShouldTraverseAllSubNodes 是否需要遍历完所有节点
func (f *ParseFrame) ShouldTraverseAllSubNodes() bool {
	return f.InVarScope() || f.InExecScope()
}

// InNormalScope 是否在普通作用域
func (f *ParseFrame) InNormalScope() bool {
	return !f.InVarScope() && !f.InExecScope()
}

// InVarScope 是否在变量赋值作用域
func (f *ParseFrame) InVarScope() bool {
	return f.SharedMemo["assigning_variable"] == true
}

// InExecScope 是否在操作执行作用域
func (f *ParseFrame) InExecScope() bool {
	return f.SharedMemo["executing_operation"] == true
}

// BuildNodePath 构建节点路径
func (f *ParseFrame) BuildNodePath(fieldName string) string {
	// TODO: 性能优化
	if f.Path == "" {
		return fmt.Sprintf("%d_%s", f.SubNodeIter.Position(), fieldName)
	}
	return fmt.Sprintf("%s.%d_%s", f.Path, f.SubNodeIter.Position(), fieldName)
}

// PutTarget 将值放入 Target
func (f *ParseFrame) PutTarget(value any) {
	util.AppendOrSet(f.Target, f.FieldName, value)
}

// PutResult 将值放入 Result
func (f *ParseFrame) PutResult(fieldName string, value any) {
	util.AppendOrSet(f.Result, fieldName, value)
}
