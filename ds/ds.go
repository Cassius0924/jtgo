package ds

import (
	"github.com/liyue201/gostl/ds/deque"
	"github.com/liyue201/gostl/ds/queue"
	"github.com/liyue201/gostl/ds/set"
	"github.com/liyue201/gostl/ds/stack"
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
