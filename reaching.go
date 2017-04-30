package evmdis

import (
	"fmt"
	"github.com/Arachnid/evmdis/stack"
	"log"
	"strings"
)

type InstructionPointer struct {
	OriginBlock *BasicBlock
	OriginIndex int
}

func (self InstructionPointer) Get() *Instruction {
	return &self.OriginBlock.Instructions[self.OriginIndex]
}

func (self InstructionPointer) GetAddress() int {
	address := self.OriginBlock.Offset
	for i := 0; i < self.OriginIndex; i++ {
		address += self.OriginBlock.Instructions[i].Op.OperandSize() + 1
	}
	return address
}

func (self InstructionPointer) String() string {
	inst := self.Get()

	var expression Expression
	inst.Annotations.Get(&expression)
	switch expression := expression.(type) {
	case *InstructionExpression:
		if(expression.Inst.Op.IsPush()) {
			return fmt.Sprintf("0x%X", expression.Inst.Arg)
		}
		break;
	case *JumpLabel:
		return fmt.Sprintf("%v", expression)
	}

	return fmt.Sprintf("@0x%X", self.GetAddress())
}

type InstructionPointerSet map[InstructionPointer]bool

func (self InstructionPointerSet) String() string {
	pointers := make([]string, 0)
	for k := range self {
		pointers = append(pointers, k.String())
	}
	if len(pointers) == 1 {
		return pointers[0]
	} else {
		return fmt.Sprintf("[%v]", strings.Join(pointers, " | "))
	}
}

func (self InstructionPointerSet) First() *InstructionPointer {
	for pointer, _ := range self {
		return &pointer
	}
	return nil
}

type ReachingDefinition []InstructionPointerSet

type reachingState struct {
	program   *Program
	nextBlock *BasicBlock
	stack     stack.StackFrame
}

func PerformReachingAnalysis(prog *Program) error {
	initial := reachingState{
		program:   prog,
		nextBlock: prog.Blocks[0],
		stack:     stack.StackEnd{},
	}
	return ExecuteAbstractly(initial)
}

func updateBlockReachings(block *BasicBlock, stack stack.StackFrame) {
	var reachings ReachingDefinition
	block.Annotations.Get(&reachings)
	if reachings == nil {
		reachings = make([]InstructionPointerSet, stack.Height())
		for i := 0; i < len(reachings); i++ {
			reachings[i] = make(map[InstructionPointer]bool)
		}
	}

	frame := stack
	for i := 0; i < stack.Height(); i++ {
		if len(reachings) <= i {
			break
		}
		reachings[i][frame.Value().(InstructionPointer)] = true
		frame = frame.Up()
	}

	if stack.Height() < len(reachings) {
		reachings = reachings[:stack.Height()]
	}

	block.Annotations.Set(&reachings)
}

func updateReachings(inst *Instruction, operands []InstructionPointer) {
	var reachings ReachingDefinition
	inst.Annotations.Get(&reachings)
	if reachings == nil {
		reachings = make([]InstructionPointerSet, len(operands))
		for i := 0; i < len(reachings); i++ {
			reachings[i] = make(map[InstructionPointer]bool)
		}
	}

	for i, operand := range operands {
		reachings[i][operand] = true
	}
	inst.Annotations.Set(&reachings)
}

func (self reachingState) Advance() ([]EvmState, error) {
	log.Printf("Entering block at %d with stack height %v", self.nextBlock.Offset, self.stack.Height())
	updateBlockReachings(self.nextBlock, self.stack)
	pc := self.nextBlock.Offset
	st := self.stack
	for i := range self.nextBlock.Instructions {
		inst := &self.nextBlock.Instructions[i]
		op := inst.Op
		opFrames, newStack := stack.Popn(st, op.StackReads())
		operands := make([]InstructionPointer, len(opFrames))
		for i, frame := range opFrames {
			operands[i] = frame.Value().(InstructionPointer)
		}
		updateReachings(inst, operands)

		switch true {
		// Ops that terminate execution
		case op == STOP:
			fallthrough
		case op == RETURN:
			fallthrough
		case op == INVALID:
			fallthrough
		case op == REVERT:
			fallthrough
		case op == SELFDESTRUCT:
			return nil, nil
		case op.IsPush():
			newStack = stack.NewFrame(newStack, InstructionPointer{self.nextBlock, i})
		case op.IsDup():
			// Uses stack instead of newStack, because we don't actually want to pop all those elements
			newStack = stack.NewFrame(st, stack.UpBy(st, op.StackReads()-1).Value())
		case op.IsSwap():
			// Uses stack instead of newStack, because we don't actually want to pop all those elements
			newStack = stack.Swap(st, op.StackReads()-1)
		case op == JUMP:
			if !operands[0].Get().Op.IsPush() {
				return nil, fmt.Errorf("%v: Could not determine jump location statically; source is %v", pc, operands[0].GetAddress())
			}
			if dest, ok := self.program.JumpDestinations[int(operands[0].Get().Arg.Int64())]; ok {
				return []EvmState{
					reachingState{
						program:   self.program,
						nextBlock: dest,
						stack:     newStack,
					},
				}, nil
			}
			return nil, nil
		case op == JUMPI:
			if !operands[0].Get().Op.IsPush() {
				return nil, fmt.Errorf("%v: Could not determine jump location statically; source is %v", pc, operands[0].GetAddress())
			}
			var ret []EvmState
			if dest, ok := self.program.JumpDestinations[int(operands[0].Get().Arg.Int64())]; ok {
				ret = append(ret, reachingState{
					program:   self.program,
					nextBlock: dest,
					stack:     newStack,
				})
			}
			if self.nextBlock.Next != nil {
				ret = append(ret, reachingState{
					program:   self.program,
					nextBlock: self.nextBlock.Next,
					stack:     newStack,
				})
			}
			return ret, nil
		default:
			if op.StackWrites() == 1 {
				newStack = stack.NewFrame(newStack, InstructionPointer{self.nextBlock, i})
			} else if op.StackWrites() > 1 {
				return nil, fmt.Errorf("Unexpected op %v makes %v writes to the stack", op, op.StackWrites())
			}
		}

		// If the stack is too deep, abort
		if st.Height() > 1024 {
			return nil, nil
		}

		pc += op.OperandSize() + 1
		st = newStack
	}

	if self.nextBlock.Next != nil {
		return []EvmState{
			reachingState{
				program:   self.program,
				nextBlock: self.nextBlock.Next,
				stack:     st,
			},
		}, nil
	} else {
		return nil, nil
	}
}

type ReachesDefinition []InstructionPointer

func (self ReachesDefinition) String() string {
	parts := make([]string, 0)
	for _, pointer := range self {
		parts = append(parts, pointer.String())
	}
	return fmt.Sprintf("%v", parts)
}

func PerformReachesAnalysis(prog *Program) {
	for _, block := range prog.Blocks {
		for i, inst := range block.Instructions {
			if inst.Op.IsSwap() || inst.Op.IsDup() {
				continue
			}

			var reaching ReachingDefinition
			inst.Annotations.Get(&reaching)
			if reaching != nil {
				ptr := InstructionPointer{
					OriginBlock: block,
					OriginIndex: i,
				}
				for _, pointers := range reaching {
					for pointer := range pointers {
						var reaches ReachesDefinition
						pointer.Get().Annotations.Get(&reaches)
						reaches = append(reaches, ptr)
						pointer.Get().Annotations.Set(&reaches)
					}
				}
			}
		}
	}
}
