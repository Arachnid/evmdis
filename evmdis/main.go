package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"log"
	"strconv"
	"flag"
	"github.com/Arachnid/evmdis"
)

func main() {
	
	withSwarmHash := flag.Bool("swarm", true, "a bool")
	ctorMode := flag.Bool("ctor", false, "a bool")
	logging := flag.Bool("log", false, "a bool")

	flag.Parse()

	if !*logging {
		log.SetOutput(ioutil.Discard)
	}

	hexdata, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		panic(fmt.Sprintf("Could not read from stdin: %v", err))
	}

	bytecodeLength := uint64(hex.DecodedLen(len(hexdata)))
	bytecode := make([]byte, bytecodeLength)

	hex.Decode(bytecode, hexdata)

	if bytecode[bytecodeLength-1] == 0x29 &&
			bytecode[bytecodeLength-2] == 0x00 &&
			bytecode[bytecodeLength-43] == 0xa1 &&
			bytecode[bytecodeLength-42] == 0x65 && *withSwarmHash {
		//swarm hash
		bytecodeLength -= 43
	} 

	program := evmdis.NewProgram(bytecode[:bytecodeLength])
	AnalizeProgram(program)

	if *ctorMode {
		var codeEntryPoint = FindNextCodeEntryPoint(program)

		log.Printf("Entrypoint code at index %v\n", codeEntryPoint)

		if codeEntryPoint == 0 {
			panic("no code entrypoint found in ctor")
		} else if codeEntryPoint >= bytecodeLength {
			panic("code entrypoint outside of currently available code")
		}

		ctor := evmdis.NewProgram(bytecode[:codeEntryPoint])
		code := evmdis.NewProgram(bytecode[codeEntryPoint:bytecodeLength])

		AnalizeProgram(ctor)
		fmt.Println("# Constructor part -------------------------")
		PrintAnalysisResult(ctor)


		AnalizeProgram(code)
		fmt.Println("# Code part -------------------------")
		PrintAnalysisResult(code)


	} else {
		PrintAnalysisResult(program)
	}	
}

func FindNextCodeEntryPoint(program *evmdis.Program) uint64 {
	var lastPos uint64 = 0
	for _, block := range program.Blocks {
		for _, instruction := range block.Instructions {
			if instruction.Op == evmdis.CODECOPY {
				var expression evmdis.Expression

				instruction.Annotations.Get(&expression)

				log.Printf("0x%X\tPUSH(%v)\n", 0, expression)

				if i, err := strconv.ParseUint(expression.GetArgString(1)[2:], 16, 32); err == nil {
					lastPos = i
				}
			}
		}
	}
	return lastPos
}

func PrintAnalysisResult(program *evmdis.Program) {
	for _, block := range program.Blocks {
		offset := block.Offset

		// Print out the jump label for the block, if there is one
		var label *evmdis.JumpLabel
		block.Annotations.Get(&label)
		if label != nil {
			fmt.Printf("%v\n", label)
		}

		// Print out the stack prestate for this block
		var reaching evmdis.ReachingDefinition
		block.Annotations.Get(&reaching)
		fmt.Printf("# Stack: %v\n", reaching)

		for _, instruction := range block.Instructions {
			// instruction.Annotations.Get(&reaching)

			log.Println(fmt.Sprintf("op: %s\n", instruction.Op))

			var reaches evmdis.ReachesDefinition
			instruction.Annotations.Get(&reaches)

			var expression evmdis.Expression
			instruction.Annotations.Get(&expression)

			if expression != nil {
				if instruction.Op.StackWrites() == 1 && !instruction.Op.IsDup() {
					fmt.Printf("0x%X\tPUSH(%v)\n", offset, expression)
				} else {
					fmt.Printf("0x%X\t%v\n", offset, expression)
				}
			}
			offset += instruction.Op.OperandSize() + 1
		}
		fmt.Printf("\n")
	}
}

func AnalizeProgram(program *evmdis.Program) {
	if err := evmdis.PerformReachingAnalysis(program); err != nil {
		panic(fmt.Sprintf("Error performing reaching analysis: %v", err))
	}
	evmdis.PerformReachesAnalysis(program)
	evmdis.CreateLabels(program)
	if err := evmdis.BuildExpressions(program); err != nil {
		panic(fmt.Sprintf("Error building expressions: %v", err))
	}
}
