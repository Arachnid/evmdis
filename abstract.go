package evmdis

import (
    "fmt"
)

type StackFrame struct {
    Up *StackFrame
    Height int
    Value interface{}
}

func NewFrame(up *StackFrame, value interface{}) *StackFrame {
    if up != nil {
        return &StackFrame{up, up.Height + 1, value}
    } else {
        return &StackFrame{nil, 0, value}
    }
}

func (self *StackFrame) UpBy(num int) *StackFrame {
    ret := self
    for i := 0; i < num; i++ {
        ret = ret.Up
    }
    return ret
}

func (self *StackFrame) Replace(num int, value interface{}) (*StackFrame, interface{}) {
    if num == 0 {
        return NewFrame(self.Up, value), self.Value
    }
    up, old := self.Up.Replace(num - 1, value)
    return NewFrame(up, self.Value), old
}

func (self *StackFrame) Swap(num int) *StackFrame {
    up, old := self.Up.Replace(num - 1, self.Value)
    return NewFrame(up, old)
}

func (self *StackFrame) String() string {
    values := make([]interface{}, 0, self.Height + 1)
    for frame := self; frame != nil; frame = frame.Up {
        values = append(values, frame.Value)
    }
    return fmt.Sprintf("%v", values)
}

func (self *StackFrame) Popn(n int) (values []*StackFrame, stack *StackFrame) {
    stack = self
    values = make([]*StackFrame, n)
    for i := 0; i < n; i++ {
        values[i] = stack
        stack = stack.Up
    }
    return values, stack
}


type EvmState interface {
    Advance() ([]EvmState, error)
}

func ExecuteAbstractly(initial EvmState) error {
    stack := []EvmState{initial}
    seen := make(map[EvmState]bool)

    for len(stack) > 0 {
        var state EvmState
        state, stack = stack[len(stack) - 1], stack[:len(stack) - 1]
        nextStates, err := state.Advance()
        if err != nil {
            return err
        }
        for _, nextState := range nextStates {
            if !seen[nextState] {
                stack = append(stack, nextState)
                seen[nextState] = true
            }
        }
    }

    return nil
}
