package disassemble

import (
	"fmt"

	"github.com/Arachnid/evmdis"
)

const swarmHashLength = 43

var swarmHashProgramTrailer = [...]byte{0x00, 0x29}
var swarmHashHeader = [...]byte{0xa1, 0x65}

func Disassemble(bytecode []byte, withSwarmHash bool, ctorMode bool) (disassembly string, err error) {
	// detect swarm hash and remove it from bytecode, see http://solidity.readthedocs.io/en/latest/miscellaneous.html?highlight=swarm#encoding-of-the-metadata-hash-in-the-bytecode
	bytecodeLength := uint64(len(bytecode))
	if bytecode[bytecodeLength-1] == swarmHashProgramTrailer[1] &&
		bytecode[bytecodeLength-2] == swarmHashProgramTrailer[0] &&
		bytecode[bytecodeLength-43] == swarmHashHeader[0] &&
		bytecode[bytecodeLength-42] == swarmHashHeader[1] &&
		withSwarmHash {

		bytecodeLength -= swarmHashLength // remove swarm part
	}

	program := evmdis.NewProgram(bytecode[:bytecodeLength])
	AnalyzeProgram(program)

	if ctorMode {
		var codeEntryPoint = FindNextCodeEntryPoint(program)

		if codeEntryPoint == 0 {
			return disassembly, fmt.Errorf("no code entrypoint found in ctor")
		} else if codeEntryPoint >= bytecodeLength {
			return disassembly, fmt.Errorf("code entrypoint outside of currently available code")
		}

		ctor := evmdis.NewProgram(bytecode[:codeEntryPoint])
		code := evmdis.NewProgram(bytecode[codeEntryPoint:bytecodeLength])

		AnalyzeProgram(ctor)
		disassembly += fmt.Sprintln("# Constructor part -------------------------")
		disassembly += PrintAnalysisResult(ctor)

		AnalyzeProgram(code)
		disassembly += fmt.Sprintln("# Code part -------------------------")
		disassembly += PrintAnalysisResult(code)

	} else {
		disassembly += PrintAnalysisResult(program)
	}

	return disassembly, nil
}

func FindNextCodeEntryPoint(program *evmdis.Program) uint64 {
	var lastPos uint64 = 0
	for _, block := range program.Blocks {
		for _, instruction := range block.Instructions {
			if instruction.Op == evmdis.CODECOPY {
				var expression evmdis.Expression

				instruction.Annotations.Get(&expression)

				arg := expression.(*evmdis.InstructionExpression).Arguments[1].Eval()

				if arg != nil {
					lastPos = arg.Uint64()
				}
			}
		}
	}
	return lastPos
}

func PrintAnalysisResult(program *evmdis.Program) (disassembly string) {
	for _, block := range program.Blocks {
		offset := block.Offset

		// Print out the jump label for the block, if there is one
		var label *evmdis.JumpLabel
		block.Annotations.Get(&label)
		if label != nil {
			disassembly += fmt.Sprintf("%v\n", label)
		}

		// Print out the stack prestate for this block
		var reaching evmdis.ReachingDefinition
		block.Annotations.Get(&reaching)

		blockDisassembly := fmt.Sprintf("# Stack: %v\n", reaching)
		blockRealInstructions := 0

		for _, instruction := range block.Instructions {
			var expression evmdis.Expression
			instruction.Annotations.Get(&expression)

			if expression != nil {
				if instruction.Op.StackWrites() == 1 && !instruction.Op.IsDup() {
					blockDisassembly += fmt.Sprintf("0x%X\tPUSH(%v)\n", offset, expression)
				} else {
					blockDisassembly += fmt.Sprintf("0x%X\t%v\n", offset, expression)
				}

				blockRealInstructions++
			}
			offset += instruction.Op.OperandSize() + 1
		}

		blockDisassembly += fmt.Sprintf("\n")

		// avoid printing empty stack frames with no instructions in the block
		if len(reaching) > 0 || blockRealInstructions > 0 {
			disassembly += blockDisassembly
		}
	}

	return disassembly
}

func AnalyzeProgram(program *evmdis.Program) (err error) {
	if err := evmdis.PerformReachingAnalysis(program); err != nil {
		return fmt.Errorf("Error performing reaching analysis: %v", err)
	}
	evmdis.PerformReachesAnalysis(program)
	evmdis.CreateLabels(program)
	if err := evmdis.BuildExpressions(program); err != nil {
		return fmt.Errorf("Error building expressions: %v", err)
	}

	return nil
}
