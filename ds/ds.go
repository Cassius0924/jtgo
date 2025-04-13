package ds

import (
	"github.com/liyue201/gostl/ds/array"
	"github.com/liyue201/gostl/ds/bitmap"
	"github.com/liyue201/gostl/ds/deque"
	"github.com/liyue201/gostl/ds/list/bidlist"
	"github.com/liyue201/gostl/ds/list/simplelist"
	treemap "github.com/liyue201/gostl/ds/map"
	"github.com/liyue201/gostl/ds/queue"
	"github.com/liyue201/gostl/ds/set"
	"github.com/liyue201/gostl/ds/stack"
	"github.com/liyue201/gostl/ds/vector"
	"github.com/liyue201/gostl/utils/comparator"
)

// NewDeque 创建双端队列
func NewDeque[T any]() *deque.Deque[T] {
	return deque.New[T]()
}

// NewQueue 创建单向队列
func NewQueue[T any]() *queue.Queue[T] {
	return queue.New[T]()
}

// NewStack 创建栈
func NewStack[T any]() *stack.Stack[T] {
	return stack.New[T]()
}

// NewSet 创建集合
func NewSet[T any](cmp comparator.Comparator[T], opts ...set.Option) *set.Set[T] {
	return set.New(cmp, opts...)
}

// NewTreeMap 创建基于红黑树的有序映射，支持顺序访问
func NewTreeMap[K, V any](cmp comparator.Comparator[K], opts ...treemap.Option) *treemap.Map[K, V] {
	return treemap.New[K, V](cmp, opts...)
}

// NewBitmap 创建位图
func NewBitmap(size uint64) *bitmap.Bitmap {
	return bitmap.New(size)
}

// NewArray 创建定长数组
func NewArray[T any](size int) *array.Array[T] {
	return array.New[T](size)
}

// NewVector 创建动态数组
func NewVector[T any]() *vector.Vector[T] {
	return vector.New[T]()
}

// NewForwardList 创建单向链表
func NewForwardList[T any]() *simplelist.List[T] {
	return simplelist.New[T]()
}

// NewList 创建双向链表
func NewList[T any]() *bidlist.List[T] {
	return bidlist.New[T]()
}
