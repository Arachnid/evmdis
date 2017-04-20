package evmdis

import (
	"fmt"
	"math/big"
	"strings"
)

type Expression interface {
	String() string
	Eval() *big.Int
}

var opcodeFormatStrings = map[OpCode]string{
	ADD:    "%v + %v",
	MUL:    "%v * %v",
	SUB:    "%v - %v",
	DIV:    "%v / %v",
	MOD:    "%v %% %v",
	EXP:    "%v ** %v",
	NOT:    "~%v",
	LT:     "%v < %v",
	GT:     "%v > %v",
	EQ:     "%v == %v",
	ISZERO: "!%v",
	AND:    "%v & %v",
	OR:     "%v | %v",
	XOR:    "%v ^ %v",
}

var operatorPrecedences = map[OpCode]int{
	NOT:    0,
	ISZERO: 0,
	EXP:    1,
	MUL:    2,
	DIV:    2,
	MOD:    2,
	ADD:    3,
	SUB:    3,
	AND:    4,
	XOR:    5,
	OR:     6,
	LT:     7,
	GT:     7,
	EQ:     7,
}

type InstructionExpression struct {
	Inst      *Instruction
	Arguments []Expression
}

func (self *InstructionExpression) Eval() *big.Int {
	return self.Inst.Arg
}

func (self *InstructionExpression) String() string {
	if self.Inst.Op.IsPush() {
		// Print push instructions as just their value
		return fmt.Sprintf("0x%X", self.Inst.Arg)
	} else if format, ok := opcodeFormatStrings[self.Inst.Op]; ok {
		args := make([]interface{}, 0, len(self.Arguments))
		for _, arg := range self.Arguments {
			if ie, ok := arg.(*InstructionExpression); ok && operatorPrecedences[ie.Inst.Op] > operatorPrecedences[self.Inst.Op] {
				args = append(args, fmt.Sprintf("(%s)", arg.String()))
			} else {
				args = append(args, arg.String())
			}
		}
		return fmt.Sprintf(format, args...)
	} else {
		// Format the opcode as a function call
		args := make([]string, 0, len(self.Arguments))
		for _, arg := range self.Arguments {
			args = append(args, arg.String())
		}
		return fmt.Sprintf("%s(%s)", self.Inst.Op, strings.Join(args, ", "))
	}
}

type PopExpression struct{}

func (self *PopExpression) String() string {
	return "POP()"
}

func (self *PopExpression) Eval() *big.Int {
	return nil
}

type SwapExpression struct {
	count int
}

func (self *SwapExpression) String() string {
	return fmt.Sprintf("SWAP%d", self.count)
}

func (self *SwapExpression) Eval() *big.Int {
	return nil
}

type DupExpression struct {
	count int
}

func (self *DupExpression) Eval() *big.Int {
	return nil
}

func (self *DupExpression) String() string {
	return fmt.Sprintf("DUP%d", self.count)
}

type JumpLabel struct {
	id       int
	refCount int
}

func (self *JumpLabel) Eval() *big.Int {
	return nil
}

func (self *JumpLabel) String() string {
	return fmt.Sprintf(":label%d", self.id)
}

func CreateLabels(prog *Program) {
	// Create initial labels, one per block
	for _, block := range prog.Blocks {
		label := &JumpLabel{}
		block.Annotations.Set(&label)
	}

	// Find all uses of labels and create references
	for _, block := range prog.Blocks {
	nextInstruction:
		for i, inst := range block.Instructions {
			if !inst.Op.IsPush() {
				continue
			}

			// Skip any pushes that aren't found in the jump table
			targetBlock := prog.JumpDestinations[int(inst.Arg.Int64())]
			if targetBlock == nil {
				continue
			}

			// Skip any pushes that aren't consumed exclusively as jump targets
			var reaches ReachesDefinition
			inst.Annotations.Get(&reaches)
			for _, pointer := range reaches {
				targetInst := pointer.Get()
				if targetInst.Op == JUMPI {
					// Check if it's the second argument
					var reaching ReachingDefinition
					targetInst.Annotations.Get(&reaching)
					if !reaching[0][InstructionPointer{block, i}] {
						continue nextInstruction
					}
				} else if targetInst.Op != JUMP {
					continue nextInstruction
				}
			}

			// Fetch the label and add a reference as an expression
			var label *JumpLabel
			targetBlock.Annotations.Get(&label)
			label.refCount += 1
			expression := Expression(label)
			inst.Annotations.Set(&expression)
		}
	}

	// Assign label numbers and delete unused labels
	count := 0
	for _, block := range prog.Blocks {
		var label *JumpLabel
		block.Annotations.Get(&label)
		if label.refCount == 0 {
			block.Annotations.Pop(&label)
		} else {
			label.id = count
			count += 1
		}
	}
}

func BuildExpressions(prog *Program) error {
	for _, block := range prog.Blocks {
		var reaching ReachingDefinition
		block.Annotations.Get(&reaching)

		// If reaching is nil, this block is unreachable; skip processing it
		if reaching == nil {
			continue
		}

		// Lifted is a set of subexpressions that can be incorporated into larger expressions;
		// they have been 'lifted' out of the stack.
		lifted := make(InstructionPointerSet)
		for i := 0; i < len(block.Instructions); i++ {
			inst := &block.Instructions[i]

			// Find all the definitions that reach each argument of this op
			var reaching ReachingDefinition
			inst.Annotations.Get(&reaching)
			if len(reaching) != inst.Op.StackReads() {
				return fmt.Errorf("Processing %v@0x%X: expected number of stack reads (%v) to equal reaching definition length (%v)", inst, block.OffsetOf(inst), inst.Op.StackReads(), len(reaching))
			}

			if inst.Op.IsSwap() {
				// Try and reduce the size of swap operations to account for lifted arguments
				swapFrom, swapTo := reaching[0], reaching[len(reaching)-1]
				leftLifted := len(swapFrom) == 1 && lifted[*swapFrom.First()]
				rightLifted := len(swapTo) == 1 && lifted[*swapTo.First()]
				if len(reaching) > 2 || (!leftLifted && !rightLifted) {
					// One side only is lifted; resolve by making arg explicit again
					if leftLifted && !rightLifted {
						delete(lifted, *swapFrom.First())
					} else if !leftLifted && rightLifted {
						delete(lifted, *swapTo.First())
					}

					if !leftLifted || !rightLifted {
						// Count number of non-lifted elements between the operands
						count := 0
						for i := 1; i < len(reaching)-1; i++ {
							if len(reaching[i]) != 1 || !lifted[*reaching[i].First()] {
								count += 1
							}
						}
						var expression Expression = &SwapExpression{count + 1}
						inst.Annotations.Set(&expression)
					}
				}
			} else if inst.Op.IsDup() {
				// Try and reduce the size of dup operations to account for lifted arguments

				dupOf := reaching[len(reaching)-1]
				if len(dupOf) == 1 && lifted[*dupOf.First()] {
					// 'unlift' any operations that are consumed by DUPs
					delete(lifted, *dupOf.First())
				}

				// Count number of non-lifted elements between the operands
				count := 0
				for i := 0; i < len(reaching)-1; i++ {
					if len(reaching[i]) != 1 || !lifted[*reaching[i].First()] {
						count += 1
					}
				}

				var expression Expression = &DupExpression{count + 1}
				inst.Annotations.Set(&expression)
			} else if inst.Op == POP && (len(reaching[0]) > 1 || !lifted[*reaching[0].First()]) {
				// Represent POPs explicitly if the argument is consumed in more than one place
				var expression Expression = &PopExpression{}
				inst.Annotations.Set(&expression)
			} else {
				var expression Expression
				inst.Annotations.Get(&expression)

				// Don't recalculate expressions found by previous passes
				if expression == nil {
					args := make([]Expression, 0, inst.Op.StackReads())
					// Assemble a subexpression for each argument
					for _, pointers := range reaching {
						if len(pointers) > 1 || !lifted[*pointers.First()] {
							// If there's more than one definition reaching the argument
							// or it's not in our set of expression fragments, represent it
							// as a stack pop.
							args = append(args, &PopExpression{})
						} else {
							// Inline this argument's expression
							sourcePointer := pointers.First()
							var sourceExpression Expression
							sourcePointer.Get().Annotations.Pop(&sourceExpression)
							args = append(args, sourceExpression)
							delete(lifted, *sourcePointer)
						}
					}

					expression = &InstructionExpression{inst, args}
					inst.Annotations.Set(&expression)
				}

				var reaches ReachesDefinition
				inst.Annotations.Get(&reaches)
				if len(reaches) == 1 && reaches[0].OriginBlock == block {
					// 'Lift' this definition out of the stack, since we know it'll be consumed
					// later in this block (and only there)
					ptr := InstructionPointer{block, i}
					lifted[ptr] = true
				}
			}
		}
		if len(lifted) != 0 {
			return fmt.Errorf("Expected all lifted arguments to be consumed by end of block: %v", lifted)
		}
	}

	return nil
}
