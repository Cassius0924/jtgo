package model

import (
	"github.com/cassius0924/jtgo/ds"
	"github.com/cassius0924/jtgo/engine/flags"
	"github.com/cassius0924/jtgo/engine/keywords"
	"github.com/liyue201/gostl/ds/vector"
	"github.com/tidwall/gjson"
)

// TNode TNode(TemplateNode) 代表一个模板节点
type TNode struct {
	gjson.Result
	Parent      *TNode                   // 父节点
	NodeFlag    *ds.Flag[flags.NodeFlag] // 节点标志位
	Keyword     keywords.Keyword         // 关键字
	Statement   string                   // 语句
	CondContext *ConditionalContext      // 条件上下文信息
	LoopContext *LoopContext             // 循环上下文信息
	// Processor   Processor                // 处理器
}

// NewTNode 创建一个新的模板节点
func NewTNode(gjsonResult gjson.Result, parent *TNode) *TNode {
	nodeFlag := ds.NewFlag[flags.NodeFlag]()
	if !(gjsonResult.IsObject() || gjsonResult.IsArray()) {
		nodeFlag.Set(flags.NodeFlagNodeNonObjectOrArray)
	}
	return &TNode{
		Result:   gjsonResult,
		Parent:   parent,
		NodeFlag: nodeFlag,
	}
}

func (n *TNode) IsConditionalMatched() bool {
	return n.CondContext.IsMatched
}

func (n *TNode) IsAnyKeyword() bool {
	return n.NodeFlag.HasAny(
		flags.NodeFlagKeywordIf, flags.NodeFlagKeywordElif, flags.NodeFlagKeywordElse,
		flags.NodeFlagKeywordFor, flags.NodeFlagKeywordExec, flags.NodeFlagKeywordVar,
		flags.NodeFlagKeywordReturn, flags.NodeFlagKeywordCmt,
	)
}

// NodeIter 节点迭代器
type NodeIter = vector.VectorIterator[*ds.Pair[*TNode, *TNode]]

// ConditionalContext 条件判断上下文
type ConditionalContext struct {
	MatchedValue *TNode // 匹配到的值
	IsMatched    bool   // 是否匹配到条件

	HasElifBranch bool // 是否有elif分支
	HasElseBranch bool // 是否有else分支
}

// NewConditionalContext 创建条件判断上下文
func NewConditionalContext() *ConditionalContext {
	return &ConditionalContext{}
}

type LoopType int

const (
	LoopTypeForWithSlice LoopType = iota
	LoopTypeForWithArray
	LoopTypeForWithMap
)

// LoopContext 循环上下文
type LoopContext struct {
	Type         LoopType // 循环类型
	Length       int      // 循环对象的长度
	Index        int      // 当前循环索引
	Continued    bool     // 是否命中 continue
	OriginKey    any      // 循环的原始键
	OriginValue  any      // 循环的原始值
	DealNextItem func()   // 处理下一个循环项
	ResultList   []any    // 循环结果
}

// NewLoopContext 创建一个循环上下文
func NewLoopContext() *LoopContext {
	return &LoopContext{}
}

// IsDone 判断循环是否完成
func (l *LoopContext) IsDone() bool {
	return l.Index >= l.Length
}

// LoopMeta 循环元数据，用于存储循环的键、值和对象的变量名
type LoopMeta struct {
	Key    string // 循环的键，如果为 key 为 _ 则此处会为空
	Value  string // 循环的值，如果为 value 为 _ 则此处会为空
	Object string // 循环的对象
}
