# evmdis
evmdis is an EVM disassembler. It performs static analysis on the bytecode to provide a higher level of abstraction for the bytecode than raw EVM operations.

Features include:
 - Separates bytecode into basic blocks.
 - Jump target analysis, assigning labels to jump targets and replacing addresses with label names.
 - Composes individual operations into compound expressions where possible.
 
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

    0x4     MSTORE(0x40, 0x60)
    0xD     PUSH(CALLDATALOAD(0x0) / 0x2 ** 0xE0)
    0x13    DUP1
    0x17    JUMPI(:label0, POP() == 0xEEE97206)

    0x18    DUP1
    0x21    JUMPI(:label2, 0xF40A049D == POP())

    0x23    STOP()

    :label0
    0x25    PUSH(:label3)
    0x29    PUSH(CALLDATALOAD(0x4))
    0x2A    PUSH(0x0)
    0x2C    PUSH(:label4)
    0x2E    DUP3
    0x2F    PUSH(0x2)

    :label1
    0x32    PUSH(POP() * POP())
    0x33    SWAP1
    0x34    JUMP(POP())

    :label2
    0x36    PUSH(:label3)
    0x3A    PUSH(CALLDATALOAD(0x4))
    0x3B    PUSH(0x0)
    0x3D    PUSH(:label4)
    0x3F    DUP3
    0x40    PUSH(0x3)
    0x44    JUMP(:label1)

    :label3
    0x46    PUSH(0x60)
    0x48    SWAP1
    0x49    DUP2
    0x4A    MSTORE(POP(), POP())
    0x4E    RETURN(POP(), 0x20)

    :label4
    0x50    SWAP3
    0x51    SWAP2
    0x52    POP()
    0x53    POP()
    0x54    JUMP(POP())
