package model

type NodeFlag int8

const (
	// NodeFlagExecOnce 只执行一次
	NodeFlagExecOnce NodeFlag = iota + 1
)
