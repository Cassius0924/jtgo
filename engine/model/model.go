package model

import (
	"fmt"
	"reflect"

	"github.com/cassius0924/jtgo/ds"
	"github.com/liyue201/gostl/ds/vector"
	"github.com/tidwall/gjson"
)

// TNode TemplateNode 代表一个模板节点
type TNode = gjson.Result

// ParseFrame 模板引擎解析帧
type ParseFrame struct {
	Node       *TNode         // 当前帧的节点
	FieldName  string         // 当前帧的字段名
	Target     any            // 父帧的解析结果，指向父帧的 Result
	Result     any            // 本帧的解析结果
	SharedMemo map[string]any // 帧与帧之间的共享数据的备忘录，类似一个全局变量
	Path       string         // 当前帧在模板中的路径

	SubNodeIter *vector.VectorIterator[*ds.Pair[TNode, TNode]] // 当前帧的子节点迭代器

	CondContext *ConditionalContext // 当前帧的条件上下文信息
	LoopContext *LoopContext        // 当前帧的循环上下文信息
}

// IsConditionalMatched 是否匹配到了条件
func (f *ParseFrame) IsConditionalMatched() bool {
	return f.CondContext != nil && f.CondContext.IsMatched && !(f.SharedMemo["no_match"] == true)
}

// IsLoopDone 是否完成循环
func (f *ParseFrame) IsLoopDone() bool {
	return f.LoopContext != nil && f.LoopContext.IsDone
}

// ShouldTraverseAllSubNodes 是否需要遍历完所有节点
func (f *ParseFrame) ShouldTraverseAllSubNodes() bool {
	return f.InVarScope() || f.InExecScope()
}

// InVarScope 是否在变量赋值作用域
func (f *ParseFrame) InVarScope() bool {
	return f.SharedMemo["assigning_variable"] == true
}

// InExecScope 是否在操作执行作用域
func (f *ParseFrame) InExecScope() bool {
	return f.SharedMemo["executing_operation"] == true
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

// BuildNodePath 构建节点路径
func (f *ParseFrame) BuildNodePath(fieldName string) string {
	// TODO: 性能优化
	return fmt.Sprintf("%s.%s", f.Path, fieldName)
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
	IsDone         bool             // 是否完成循环
}

func NewLoopContext() *LoopContext {
	return &LoopContext{}
}

// LoopMeta 循环元数据，用于存储循环的键、值和对象的变量名
type LoopMeta struct {
	Key    string // 循环的键，如果为 key 为 _ 则此处会为空
	Value  string // 循环的值，如果为 value 为 _ 则此处会为空
	Object string // 循环的对象
}
