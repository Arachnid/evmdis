package stack

import (
	"fmt"
)

type StackFrame interface {
	Up() StackFrame
	Height() int
	Value() interface{}
}

type stackFrame struct {
	up     StackFrame
	height int
	value  interface{}
}

func (f stackFrame) Up() StackFrame     { return f.up }
func (f stackFrame) Height() int        { return f.height }
func (f stackFrame) Value() interface{} { return f.value }

type StackEnd struct{}

func (s StackEnd) Up() StackFrame     { panic("Attempted to call Up() on StackEnd") }
func (s StackEnd) Height() int        { return 0 }
func (s StackEnd) Value() interface{} { panic("Attempted to call Value() on StackEnd") }

func NewFrame(up StackFrame, value interface{}) StackFrame {
	return stackFrame{up, up.Height() + 1, value}
}

func UpBy(stack StackFrame, num int) StackFrame {
	ret := stack
	for i := 0; i < num; i++ {
		ret = ret.Up()
	}
	return ret
}

func Replace(stack StackFrame, num int, value interface{}) (StackFrame, interface{}) {
	if num == 0 {
		return NewFrame(stack.Up(), value), stack.Value()
	}
	up, old := Replace(stack.Up(), num-1, value)
	return NewFrame(up, stack.Value()), old
}

func Swap(stack StackFrame, num int) StackFrame {
	up, old := Replace(stack.Up(), num-1, stack.Value())
	return NewFrame(up, old)
}

func String(stack StackFrame) string {
	values := make([]interface{}, 0, stack.Height() + 1)
	for frame := stack; frame.Height() > 0; frame = frame.Up() {
		values = append(values, frame.Value())
	}
	return fmt.Sprintf("%v", values)
}

func Popn(stack StackFrame, n int) (values []StackFrame, rest StackFrame) {
	rest = stack
	values = make([]StackFrame, n)
	for i := 0; i < n; i++ {
		values[i] = rest
		rest = rest.Up()
	}
	return values, rest
}
