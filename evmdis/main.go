package main

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Arachnid/evmdis"
)

func main() {
	hexdata, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		log.Fatalf("Could not read from stdin: %v", err)
	}

	bytecode := make([]byte, hex.DecodedLen(len(hexdata)))
	hex.Decode(bytecode, hexdata)

	program := evmdis.NewProgram(bytecode)
	if err := evmdis.PerformReachingAnalysis(program); err != nil {
		log.Fatalf("Error performing reaching analysis: %v", err)
	}
	evmdis.PerformReachesAnalysis(program)
	evmdis.CreateLabels(program)
	evmdis.BuildExpressions(program)

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
			instruction.Annotations.Get(&reaching)

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
