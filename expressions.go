package evmdis

import (
	"fmt"
	"log"
	"strings"
)

type Expression interface {
	String() string
}

var opcodeFormatStrings = map[OpCode]string {
	ADD: "%v + %v",
	MUL: "%v * %v",
	SUB: "%v - %v",
	DIV: "%v / %v",
	MOD: "%v %% %v",
	EXP: "%v ** %v",
	NOT: "!%v",
	LT: "%v < %v",
	GT: "%v > %v",
	EQ: "%v == %v",
	ISZERO: "%v == 0",
	AND: "%v & %v",
	OR: "%v | %v",
	XOR: "%v ^ %v",
}

var operatorPrecedences = map[OpCode]int {
	NOT: 0,
	EXP: 1,
	MUL: 2,
	DIV: 2,
	MOD: 2,
	ADD: 3,
	SUB: 3,
	AND: 4,
	XOR: 5,
	OR: 6,
	LT: 7,
	GT: 7,
	EQ: 7,
	ISZERO: 7,
}

type InstructionExpression struct {
	Inst Instruction
	Arguments []Expression
}

func (self *InstructionExpression) String() string {
	if self.Inst.Op.IsPush() {
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
		args := make([]string, 0, len(self.Arguments))
		for _, arg := range self.Arguments {
			args = append(args, arg.String())
		}
		return fmt.Sprintf("%s(%s)", self.Inst.Op, strings.Join(args, ", "))
	}
}

type PopExpression struct {}

func (self *PopExpression) String() string {
	return "POP()"
}

type SwapExpression struct {
	count int
}

func (self *SwapExpression) String() string {
	return fmt.Sprintf("SWAP%d", self.count)
}

type DupExpression struct {
	count int
}

func (self *DupExpression) String() string {
	return fmt.Sprintf("DUP%d", self.count)
}

type JumpLabel struct {
	id int
}

func (self *JumpLabel) String() string {
	return fmt.Sprintf(":label%d", self.id)
}

func CreateLabels(prog *Program) {
	counter := 0

	for _, block := range prog.Blocks {
		nextInstruction: for i, inst := range block.Instructions {
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

			// Fetch or create a jump label
			var label *JumpLabel
			targetBlock.Annotations.Get(&label)
			if label == nil {
				label = &JumpLabel{counter}
				targetBlock.Annotations.Set(&label)
				counter += 1
			}

			expression := Expression(label)
			inst.Annotations.Set(&expression)
		}
	}
}

func BuildExpressions(prog *Program) {
	for _, block := range prog.Blocks {
		lifted := make(InstructionPointerSet)
		for i, inst := range block.Instructions {
			// Find all the definitions that reach each argument of this op
			var reaching ReachingDefinition
			inst.Annotations.Get(&reaching)
			if len(reaching) != inst.Op.StackReads() {
				log.Fatalf("Expected number of stack reads (%v) to equal reaching definition length (%v)", inst.Op.StackReads(), len(reaching))
			}

			if inst.Op.IsSwap() {
				swapFrom, swapTo := reaching[0], reaching[len(reaching) - 1]
				leftLifted := len(swapFrom) == 1 && lifted[*swapFrom.First()]
				rightLifted := len(swapTo) == 1 && lifted[*swapTo.First()]
				if len(reaching) > 2 || (!leftLifted && !rightLifted) {
					if leftLifted && !rightLifted {
						// One side only is lifted; resolve by making arg explicit again
						delete(lifted, *swapFrom.First())
					} else if !leftLifted && rightLifted {
						delete(lifted, *swapTo.First())
					}

					if !leftLifted || !rightLifted {
						// Count number of non-lifted elements between the operands
						count := 0
						for i := 1; i < len(reaching) - 1; i++ {
							if len(reaching[i]) != 1 || !lifted[*reaching[i].First()] {
								count += 1
							}
						}
						var expression Expression = &SwapExpression{count + 1}
						inst.Annotations.Set(&expression)
					}
				}
			} else if inst.Op.IsDup() {
				dupOf := reaching[len(reaching) - 1]
				if len(dupOf) == 1 && lifted[*dupOf.First()] {
					delete(lifted, *dupOf.First())
				}

				// Count number of non-lifted elements between the operands
				count := 0
				for i := 0; i < len(reaching) - 1; i++ {
					if len(reaching[i]) != 1 || !lifted[*reaching[i].First()] {
						count += 1
					}
				}

				var expression Expression = &DupExpression{count + 1}
				inst.Annotations.Set(&expression)
			} else if inst.Op == POP && (len(reaching[0]) > 1 || !lifted[*reaching[0].First()]) {
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
					ptr := InstructionPointer{block, i}
					log.Printf("Lifting %v; only consumed at %v", ptr.String(), reaches[0].String())
					// 'Lift' this definition out of the stack, since we know it'll be consumed
					// later in this block (and only there)
					lifted[InstructionPointer{block, i}] = true
				}
			}
		}
		if len(lifted) != 0 {
			log.Fatalf("Expected all lifted arguments to be consumed by end of block: %v", lifted)
		}
	}
}
