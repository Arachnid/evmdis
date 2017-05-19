package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/Arachnid/evmdis"

	"bufio"
	"bytes"
	"os/exec"
	"regexp"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type contract struct {
	name        string
	data        []byte
	solidity    []byte
	hexdata     []byte
	disassembly string
}

func ValidHexData(hexdata []byte) bool {
	if isHex, err := regexp.Match("^[0-9A-Fa-f]+$", hexdata); err != nil || !isHex {
		return false
	}

	return true
}

func ParseData(solc string, solcOptions string, source *contract) error {
	if !ValidHexData(source.data) {
		// if it doesn't look like hex, compile with solc

		solcArgs := strings.Split(solcOptions, " ")

		solcCmd := exec.Command(solc, solcArgs...)

		// get handles to stdin, stdout and stderr pipes for solc command
		stdin, err := solcCmd.StdinPipe()
		if err != nil {
			return err
		}

		stdout, err := solcCmd.StdoutPipe()
		if err != nil {
			return err
		}

		stderr, err := solcCmd.StderrPipe()
		if err != nil {
			return err
		}

		// start the command
		err = solcCmd.Start()
		if err != nil {
			return err
		}

		// write the solc source code and close stdin
		_, err = stdin.Write(source.data)
		if err != nil {
			return err
		}
		err = stdin.Close()
		if err != nil {
			return err
		}

		// read stdout and stderr
		solcErr := new(bytes.Buffer)
		_, err = solcErr.ReadFrom(stderr)
		if err != nil {
			return err
		}

		// loop until we have a line that looks like hex (solc tends to add extraneous info lines)
		scanner := bufio.NewScanner(stdout)

		foundHexData := false

		for scanner.Scan() {
			// check if line output was valid hex
			if ValidHexData(scanner.Bytes()) {
				source.solidity = source.data
				source.hexdata = make([]byte, len(scanner.Bytes())) // necessary? Because bufio doesn't allocate a byte slice
				copy(source.hexdata, scanner.Bytes())
				foundHexData = true

				// log.Printf("solc stdout - hexdata: '%v'", scanner.Text())
				break
			} else {
				// log.Printf("solc stdout: '%v'", scanner.Text())
			}
		}

		// wait for solc to finish
		err = solcCmd.Wait()
		if err != nil {
			_, ok := err.(*exec.ExitError)

			if ok {
				return fmt.Errorf("Problem compiling solidity: %v\n%v", err.Error(), solcErr.String())
			}

			return err
		}

		// ensure that after all of this we found valid compiled solidity
		if !foundHexData {
			return fmt.Errorf("Solidity didn't compile to valid hex: %v", solcErr.String())
		}
	} else {
		// otherwise set hexdata directly
		source.hexdata = source.data
	}

	return nil
}

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

func getDifferences(name1 string, data1 []byte, name2 string, data2 []byte, patch bool) (diff string) {
	// check for missing data
	missingData := []string{}
	if len(data1) == 0 {
		missingData = append(missingData, name1)
	}
	if len(data2) == 0 {
		missingData = append(missingData, name2)
	}

	if len(missingData) > 0 {
		diff = fmt.Sprintf("<Data not present for source(s): %q>", missingData)
		return
	}

	if bytes.Compare(data1, data2) == 0 {
		diff = "<No differences>"
		return
	}

	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(string(data1), string(data2), false)

	if !patch {
		diff = dmp.DiffPrettyText(diffs)
	} else {
		patches := dmp.PatchMake(diffs)
		diff = dmp.PatchToText(patches)
	}

	return
}

const maxSources = 2
const swarmHashLength = 43

var swarmHashProgramTrailer = [...]byte{0x00, 0x29}
var swarmHashHeader = [...]byte{0xa1, 0x65}

func main() {
	// sources to disassemble (and compare)
	// max two sources ("left side and right side") for now
	sources := []contract{}

	// command line options
	options := struct {
		withSwarmHash bool
		ctorMode      bool
		logging       bool
		solc          string
		solcOptions   string
		stdin         bool
		cmpsol        bool
		cmpbc         bool
		cmpasm        bool
		patch         bool
	}{
		withSwarmHash: true,
		ctorMode:      false,
		logging:       true,
		solc:          "solc",
		solcOptions:   "--optimize --bin-runtime",
		stdin:         false,
		cmpsol:        false,
		cmpbc:         false,
		cmpasm:        true,
		patch:         false,
	}

	flag.BoolVar(&options.withSwarmHash, "swarm", options.withSwarmHash, "solc adds a reference to the Swarm API description to the generated bytecode, if this flag is set it removes this reference before analysis")
	flag.BoolVar(&options.ctorMode, "ctor", options.ctorMode, "Indicates that the provided bytecode has construction(ctor) code included. (needs to be analyzed seperatly)")
	flag.BoolVar(&options.logging, "log", options.logging, "Print logging output.")
	flag.StringVar(&options.solc, "solc", options.solc, "Path to solc Solidity compiler.")
	flag.StringVar(&options.solcOptions, "solcoptions", options.solcOptions, "Options to pass to solc.")
	flag.BoolVar(&options.stdin, "stdin", options.stdin, "Force stdin as one of the input methods. Required if stdin desired in addition to a single command line parameter passed.")
	flag.BoolVar(&options.cmpsol, "cmpsol", options.cmpsol, "Compare solidity source code (if available).")
	flag.BoolVar(&options.cmpbc, "cmpbc", options.cmpbc, "Compare solidity bytecode.")
	flag.BoolVar(&options.cmpasm, "cmpasm", options.cmpasm, "Compare disassembled solidity bytecode.")
	flag.BoolVar(&options.patch, "patch", options.patch, "Show differences in patch format instead of by colour.")
	flag.Parse()

	if !options.logging {
		log.SetOutput(ioutil.Discard)
	}

	// validate number of command line arguments
	namedSources := flag.Args()
	numSources := len(namedSources)

	if options.stdin {
		numSources++
	}

	if numSources > maxSources {
		log.Fatalf("Invalid number of sources: %v, max %v\n", numSources, maxSources)
	}

	// load sources (stdin or file; TODO: smart contract address)

	// stdin
	if len(namedSources) == 0 || options.stdin {
		source := contract{}

		data, err := ioutil.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Could not read from stdin: %v", err)
		}

		source.name = "stdin"
		source.data = data

		sources = append(sources, source)
	}

	// named sources
	for _, namedSource := range namedSources {
		var err error
		source := contract{}

		// source is in file
		source.data, err = ioutil.ReadFile(namedSource)
		if err != nil {
			log.Fatalf("Could not read from file '%v': %v", namedSource, err)
		}

		source.name = namedSource

		sources = append(sources, source)
	}

	// parse and compile (if necessary) then disassemble each source
	for i, source := range sources {
		// parse and compile (if necessary)
		if err := ParseData(options.solc, options.solcOptions, &source); err != nil {
			log.Fatalf("Could not parse source '%v': %v", source.name, err)
		}

		// since source is a copy of sources[i]
		sources[i].solidity = source.solidity
		sources[i].hexdata = source.hexdata

		// disassemble
		bytecode := make([]byte, hex.DecodedLen(len(source.hexdata)))
		hex.Decode(bytecode, source.hexdata)

		var err error
		if sources[i].disassembly, err = Disassemble(bytecode, options.withSwarmHash, options.ctorMode); err != nil {
			log.Fatalf("Unable to disassemble %v: %v", source.name, err)
		}
	}

	// compare bytecode (hex data) and output results
	multiCmp := (options.cmpsol && options.cmpbc) || (options.cmpsol && options.cmpasm) || (options.cmpbc && options.cmpasm)
	if len(sources) == 1 {
		// single source: just display it
		src1 := sources[0]

		if options.cmpsol {
			if multiCmp {
				fmt.Println("=== SOLIDITY ===")
			}

			if len(src1.solidity) > 0 {
				fmt.Println(string(src1.solidity))
			} else {
				fmt.Println("<Solidity source code not present>")
			}
		}

		if options.cmpbc {
			if multiCmp {
				fmt.Println("=== BYTECODE ===")
			}

			fmt.Println(string(src1.hexdata))
		}

		if options.cmpasm {
			if multiCmp {
				fmt.Println("=== DISASSEMBLY ===")
			}

			fmt.Println(string(src1.disassembly))
		}

	} else {
		// multiple sources: display diffs
		src1 := sources[0]
		src2 := sources[1]

		if options.cmpsol {
			if multiCmp {
				fmt.Println("=== SOLIDITY ===")
			}

			fmt.Println(getDifferences(src1.name, src1.solidity, src2.name, src2.solidity, options.patch))
		}

		if options.cmpbc {
			if multiCmp {
				fmt.Println("=== BYTECODE ===")
			}

			fmt.Println(getDifferences(src1.name, src1.hexdata, src2.name, src2.hexdata, options.patch))
		}

		if options.cmpasm {
			if multiCmp {
				fmt.Println("=== DISASSEMBLY BY ABSTRACT INTERPRETATION ===")
			}

			fmt.Println(getDifferences(src1.name, []byte(src1.disassembly), src2.name, []byte(src2.disassembly), options.patch))
		}
	}
}
