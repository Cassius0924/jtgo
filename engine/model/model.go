package model

import (
	"reflect"

	"github.com/cassius0924/jtgo/ds"
	"github.com/liyue201/gostl/ds/deque"
	"github.com/tidwall/gjson"
)

// TNode TemplateNode 代表一个模板节点
type TNode = gjson.Result

// ParseFrame 模板引擎解析帧
type ParseFrame struct {
	Node       *TNode
	FieldName  string
	Target     any
	Result     any            // 本帧的解析结果
	SharedMemo map[string]any // 帧与帧之间的共享备忘录，类似一个全局变量

	SubNodeIter         *deque.DequeIterator[*ds.Pair[TNode, TNode]]
	CurSubNodeFieldName string

	CondContext *ConditionalContext
	LoopContext *LoopContext
}

func (f *ParseFrame) IsConditionalMatched() bool {
	return f.CondContext != nil && f.CondContext.IsMatched && !(f.SharedMemo["nomatch"] == true)
}

func (f *ParseFrame) HasLoopContext() bool {
	return f.LoopContext != nil
}

// ConditionalContext 条件判断上下文
type ConditionalContext struct {
	MatchedValue  *TNode
	IsMatched     bool
	HasIfBranch   bool // 是否有if分支
	HasElseBranch bool // 是否有else分支
}

// ResetBranchs 重置分支状态
func (c *ConditionalContext) ResetBranchs() {
	c.HasIfBranch = false
	c.HasElseBranch = false
}

type LoopType int

const (
	LoopTypeForWithSlice LoopType = iota
	LoopTypeForWithArray
	LoopTypeForWithMap
)

// LoopContext 循环上下文
type LoopContext struct {
	Type           LoopType         // 循环类型
	MapIter        *reflect.MapIter // map遍历时的迭代器，当 Type == LoopTypeForWithMap 时才拥有
	Length         int              // 循环对象的长度，当 Type == LoopTypeForWithSlice 或 LoopTypeForWithArray 时才拥有
	Meta           *LoopMeta        // 循环元数据
	Index          int              // 当前循环索引
	Key            any              // 当前循环键
	Value          any              // 当前循环值
	Object         any              // 当前循环对象
	ObjectRefl     *reflect.Value   // 当前循环对象的反射
	OwnerFieldName string           // 当前循环所属的字段名
	Continued      bool             // 是否命中 continue
	LocalVars      map[string]any   // 局部变量，TODO: 确定这个变量的含义
	IsSerialFor    bool             // 是否是串行for循环
}

func NewLoopContext() *LoopContext {
	return &LoopContext{
		LocalVars: make(map[string]any),
	}
}

// LoopMeta 循环元数据，用于存储循环的键、值和对象的变量名
type LoopMeta struct {
	Key    string // 循环的键，如果为 key 为 _ 则此处会为空
	Value  string // 循环的值，如果为 value 为 _ 则此处会为空
	Object string // 循环的对象
}
