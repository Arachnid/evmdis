package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/arachnid/evmdis"
)

func main() {
    bytecode, err := ioutil.ReadAll(os.Stdin)
    if err != nil {
        log.Fatalf("Could not read from stdin: %v", err)
    }

    program := evmdis.NewProgram(bytecode)
    if err := evmdis.PerformReachingAnalysis(program); err != nil {
    	log.Fatalf("Error performing reaching analysis: %v", err)
    }
    evmdis.PerformReachesAnalysis(program)
    evmdis.BuildExpressions(program)

    for _, block := range program.Blocks {
    	offset := block.Offset
    	for _, instruction := range block.Instructions {
    		var reaching evmdis.ReachingDefinition
    		instruction.Annotations.Get(&reaching)

    		var reaches evmdis.ReachesDefinition
    		instruction.Annotations.Get(&reaches)

    		//fmt.Printf("0x%X\t%v\t%v\t%v\n", offset, instruction.String(), reaching, reaches)
    		var expression evmdis.Expression
    		instruction.Annotations.Get(&expression)

    		if expression != nil {
    			fmt.Printf("0x%X\t%v\t%v\t%v\n", offset, expression, reaching, reaches)
    		}
    		offset += instruction.Op.OperandSize() + 1
    	}
    	fmt.Printf("\n")
    }
}
