# evmdis
evmdis is an EVM disassembler. It performs static analysis on the bytecode to provide a higher level of abstraction for the bytecode than raw EVM operations.

Features include:
 - Separates bytecode into basic blocks.
 - Jump target analysis, assigning labels to jump targets and replacing addresses with label names.
 - Composes individual operations into compound expressions where possible.
 - Provides insight into the state of the stack at the start of each block.
 
## Example
The following contract, compiled with `solc --optimize`:

    contract Test {
        function double(uint a) returns (uint) {
            return multiply(a, 2);
        }
    
        function triple(uint a) returns (uint) {
            return multiply(a, 3);
        }
    
        function multiply(uint a, uint b) internal returns (uint) {
            return a * b;
        }
    }

Produces the following disassembly:

    # Stack: []
    0x4 MSTORE(0x40, 0x60)
    0xD PUSH(CALLDATALOAD(0x0) / 0x2 ** 0xE0)
    0x13    DUP1
    0x17    JUMPI(:label0, POP() == 0xEEE97206)

    # Stack: [@0xD]
    0x18    DUP1
    0x21    JUMPI(:label2, 0xF40A049D == POP())

    # Stack: [@0xD]
    0x25    JUMP(0x2)

    :label0
    # Stack: [@0xD]
    0x2A    JUMPI(0x2, CALLVALUE())

    # Stack: [@0xD]
    0x2B    PUSH(:label3)
    0x2F    PUSH(CALLDATALOAD(0x4))
    0x30    PUSH(0x0)
    0x32    PUSH(:label4)
    0x34    DUP3
    0x35    PUSH(0x2)

    :label1
    # Stack: [[0x3 | 0x2] [@0x44 | @0x2F] [:label4 | :label4] [0x0 | 0x0] [@0x44 | @0x2F] [:label3 | :label3] @0xD]
    0x38    PUSH(POP() * POP())
    0x39    SWAP1
    0x3A    JUMP(POP())

    :label2
    # Stack: [@0xD]
    0x3F    JUMPI(0x2, CALLVALUE())

    # Stack: [@0xD]
    0x40    PUSH(:label3)
    0x44    PUSH(CALLDATALOAD(0x4))
    0x45    PUSH(0x0)
    0x47    PUSH(:label4)
    0x49    DUP3
    0x4A    PUSH(0x3)
    0x4E    JUMP(:label1)

    :label3
    # Stack: [@0x38 @0xD]
    0x50    PUSH(0x40)
    0x52    DUP1
    0x53    PUSH(MLOAD(POP()))
    0x54    SWAP2
    0x55    DUP3
    0x56    MSTORE(POP(), POP())
    0x57    PUSH(MLOAD(POP()))
    0x58    SWAP1
    0x59    DUP2
    0x5A    SWAP1
    0x60    RETURN(POP(), 0x20 + POP() - POP())

    :label4
    # Stack: [@0x38 [0x0 | 0x0] [@0x44 | @0x2F] [:label3 | :label3] @0xD]
    0x62    SWAP3
    0x63    SWAP2
    0x64    POP()
    0x65    POP()
    0x66    JUMP(POP())

## How it works

evmdis works on the principle of static analysis by abstract execution. Abstract execution is the process of simulating the execution of a program, substituting the 'concrete' values for abstract ones representing some useful property. Done properly, this is guaranteed to terminate (unlike regular execution).

Abstract execution is implemented in abstract.go. Callers provide an `EvmState` object that represents the current state of execution for the analysis in question, with an `Advance` method that performs some execution before returning zero or more new states. `EvmState` implementations must be hashable, since they're used as keys in a Go map.

Abstract execution starts with a single initial state. After calling Advance, each of the output states is compared to a map keeping track of all previously seen states. If the output state is not in the map, it is added to the map and to a stack of pending states. The procedure then pops a pending state off the stack and repeats until the stack of pending states is empty.

For example, evmdis performs reaching definition analysis. The goal of this is to generate, for each instruction that produces output, a list of all the places that the output is consumed. It does this by simulating execution of the code, replacing actual stack values with the address of the operation that generated them. Each time we encounter an instruction, we take the top elements of the current stack, and union each element with a set of previous definitions we've seen. The abstract execution semantics ensure that we continue executing for as long as we keep encountering new arrangements of the execution stack.

## Analyses

evmdis implements a number of different analyses, each building on the others. Only some of these require abstract execution.

Each analysis annotates instructions or basic blocks (described below) with objects that provide additional data about the code.

### Parsing and basic block creation

First, the code is parsed and split up into basic blocks. A basic block is a series of sequential EVM operations that do not contain any control flow (jumps in or out). Each basic block may optionally start with a JUMPDEST, and may optionally end with a JUMP or JUMPI; these operations will never occur inside a block. The sequential nature of basic blocks makes them useful building blocks for analysis.

### Reaching analysis

Next, we perform reaching definition analysis on the code, as described above in "How it works". This produces a `ReachingDefinition` annotation on each reachable basic block and each reachable instruction. A `ReachingDefinition` is a list of sets of `InstructionPointer`s, pointing to the source of each definition that reaches the given argument.

`ReachingDefinition` annotations on instructions have the same number of elements as the opcode has input arguments. `ReachingDefinition` annotations on basic blocks have elements for every stack slot that can be statically determined to be always present.

### Reaches analysis

Reaches analysis is the inverse of reaching definition analysis; for each instruction it annotates all the locations that its output reaches. This step does not require symbolic execution; it simply iterates over the reaching definition analysis and inverts it. This produces a `ReachesDefinition` annotation on each instruction. A `ReachesDefinition` is a list of instruction pointers.

### Jump target creation

This step finds all PUSH instructions whose output are consumed exclusively by JUMP or JUMPI instructions, and converts them to symbolic jump targets, inserting jump targets at the beginning of the appropriate basic blocks.

Relevant PUSH instructions are annotated with an `Expression` that is an instance of `JumpLabel`, and relevant basic blocks are annotated with the same `JumpLabel`.

### Expression construction

Expression construction iterates over each basic block, identifying operations that are consumed only at a single location, constructing composite expressions for them. For instance, `PUSH1 3 PUSH1 1 PUSH1 2 ADD MUL` is rewritten to the expression `(1 + 2) * 3`. POP operations are elided when possible, and explicitly included as annotations where not. DUP and SWAP operations are eliminated where possible, and adjusted to show the correct number of stack slots when 'lifting' removes elements from the (virtual) stack.

Each instruction that is not part of a subexpression is annotated with an `Expression` instance.
