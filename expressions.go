package evmdis

import (
	"fmt"
	"strings"
)

type Expression interface {
	String() string
}

type InstructionExpression struct {
	Inst Instruction
	Arguments []Expression
}

func (self *InstructionExpression) String() string {
	if self.Inst.Op.IsPush() {
		return fmt.Sprintf("0x%X", self.Inst.Arg)
	} else {
		args := make([]string, 0, len(self.Arguments))
		for _, arg := range self.Arguments {
			args = append(args, arg.String())
		}
		return fmt.Sprintf("%s(%s)", self.Inst.Op, strings.Join(args, ", "))
	}
}

type PopExpression struct {
}

func (self *PopExpression) String() string {
	return "POP()"
}

func BuildExpressions(prog *Program) {
	for _, block := range prog.Blocks {
		for _, inst := range block.Instructions {
			// Find all the definitions that reach each argument of this op
			var reaching ReachingDefinition
			inst.Annotations.Get(&reaching)
			if len(reaching) == inst.Op.StackReads() {
				args := make([]Expression, 0, inst.Op.StackReads())
				// For each argument
				for _, pointers := range reaching {
					if len(pointers) > 1 {
						// If there's more than one definition reaching the argument
						// represent it as a stack pop
						args = append(args, &PopExpression{})
					} else {
						sourceInst := pointers.First().Get()
						var reaches ReachesDefinition
						sourceInst.Annotations.Get(&reaches)
						if len(reaches) > 1 {
							// If the op for this argument is consumed in more than one place
							// represent ita s a stack pop
							args = append(args, &PopExpression{})
						} else {
							// Inline this argument's expression
							var sourceExpression Expression
							sourceInst.Annotations.Pop(&sourceExpression)
							args = append(args, sourceExpression)
						}
					}
				}
				var expression Expression = &InstructionExpression{inst, args}
				inst.Annotations.Set(&expression)
			}
		}
	}
}
