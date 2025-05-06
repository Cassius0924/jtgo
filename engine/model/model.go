package model

import (
	"reflect"

	"github.com/cassius0924/jtgo/ds"
	"github.com/liyue201/gostl/ds/vector"
	"github.com/tidwall/gjson"
)

// TNode TemplateNode 代表一个模板节点
type TNode = gjson.Result

// ParseFrame 模板引擎解析帧
type ParseFrame struct {
	Node          *TNode         // 当前帧的节点
	FieldName     string         // 当前帧的字段名
	Target        any            // 父帧的解析结果，指向父帧的 Result
	Result        any            // 本帧的解析结果
	IsArrayResult bool           // 本帧的解析结果是否是数组
	SharedMemo    map[string]any // 帧与帧之间的共享数据的备忘录，类似一个全局变量

	SubNodeIter *vector.VectorIterator[*ds.Pair[TNode, TNode]] // 当前帧的子节点迭代器

	CondContext *ConditionalContext // 当前帧的条件上下文信息
	LoopContext *LoopContext        // 当前帧的循环上下文信息
}

func (f *ParseFrame) IsConditionalMatched() bool {
	return f.CondContext != nil && f.CondContext.IsMatched && !(f.SharedMemo["no_match"] == true)
}

func (f *ParseFrame) HasLoopContext() bool {
	return f.LoopContext != nil
}

func (f *ParseFrame) IsVarAssigning() bool {
	return f.SharedMemo["var_assigning"] == true
}

// ResetBranches 重置分支状态
func (f *ParseFrame) ResetBranches() {
	if f.CondContext == nil {
		return
	}
	f.CondContext.HasIfBranch = false
	f.CondContext.HasElseBranch = false
}

// ResetMatched 重置匹配状态
func (f *ParseFrame) ResetMatched() {
	if f.CondContext == nil {
		return
	}
	f.CondContext.MatchedValue = nil
	f.CondContext.IsMatched = false
}

// ConditionalContext 条件判断上下文
type ConditionalContext struct {
	MatchedValue  *TNode
	IsMatched     bool
	HasIfBranch   bool // 是否有if分支
	HasElseBranch bool // 是否有else分支
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
	IsSerialFor    bool             // 是否是串行for循环
}

func NewLoopContext() *LoopContext {
	return &LoopContext{ }
}

// LoopMeta 循环元数据，用于存储循环的键、值和对象的变量名
type LoopMeta struct {
	Key    string // 循环的键，如果为 key 为 _ 则此处会为空
	Value  string // 循环的值，如果为 value 为 _ 则此处会为空
	Object string // 循环的对象
}
