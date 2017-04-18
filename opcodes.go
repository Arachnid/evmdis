package evmdis

import (
	"fmt"
)

type OpCode byte

func (op OpCode) IsPush() bool {
	switch op {
	case PUSH1, PUSH2, PUSH3, PUSH4, PUSH5, PUSH6, PUSH7, PUSH8, PUSH9, PUSH10, PUSH11, PUSH12, PUSH13, PUSH14, PUSH15, PUSH16, PUSH17, PUSH18, PUSH19, PUSH20, PUSH21, PUSH22, PUSH23, PUSH24, PUSH25, PUSH26, PUSH27, PUSH28, PUSH29, PUSH30, PUSH31, PUSH32:
		return true
	}
	return false
}

func (op OpCode) IsDup() bool {
	switch op {
	case DUP1, DUP2, DUP3, DUP4, DUP5, DUP6, DUP7, DUP8, DUP9, DUP10, DUP11, DUP12, DUP13, DUP14, DUP15, DUP16:
		return true
	}
	return false
}

func (op OpCode) IsSwap() bool {
	switch op {
	case SWAP1, SWAP2, SWAP3, SWAP4, SWAP5, SWAP6, SWAP7, SWAP8, SWAP9, SWAP10, SWAP11, SWAP12, SWAP13, SWAP14, SWAP15, SWAP16:
		return true
	}
	return false
}

func (op OpCode) OperandSize() int {
	if !op.IsPush() {
		return 0
	}

	return int(byte(op) - byte(PUSH1) + 1)
}

func (op OpCode) IsJump() bool {
	return op == JUMP || op == JUMPI
}

const (
	// 0x0 range - arithmetic ops
	STOP OpCode = iota
	ADD
	MUL
	SUB
	DIV
	SDIV
	MOD
	SMOD
	ADDMOD
	MULMOD
	EXP
	SIGNEXTEND
)

const (
	LT OpCode = iota + 0x10
	GT
	SLT
	SGT
	EQ
	ISZERO
	AND
	OR
	XOR
	NOT
	BYTE

	SHA3 = 0x20
)

const (
	// 0x30 range - closure state
	ADDRESS OpCode = 0x30 + iota
	BALANCE
	ORIGIN
	CALLER
	CALLVALUE
	CALLDATALOAD
	CALLDATASIZE
	CALLDATACOPY
	CODESIZE
	CODECOPY
	GASPRICE
	EXTCODESIZE
	EXTCODECOPY
)

const (

	// 0x40 range - block operations
	BLOCKHASH OpCode = 0x40 + iota
	COINBASE
	TIMESTAMP
	NUMBER
	DIFFICULTY
	GASLIMIT
)

const (
	// 0x50 range - 'storage' and execution
	POP OpCode = 0x50 + iota
	MLOAD
	MSTORE
	MSTORE8
	SLOAD
	SSTORE
	JUMP
	JUMPI
	PC
	MSIZE
	GAS
	JUMPDEST
)

const (
	// 0x60 range
	PUSH1 OpCode = 0x60 + iota
	PUSH2
	PUSH3
	PUSH4
	PUSH5
	PUSH6
	PUSH7
	PUSH8
	PUSH9
	PUSH10
	PUSH11
	PUSH12
	PUSH13
	PUSH14
	PUSH15
	PUSH16
	PUSH17
	PUSH18
	PUSH19
	PUSH20
	PUSH21
	PUSH22
	PUSH23
	PUSH24
	PUSH25
	PUSH26
	PUSH27
	PUSH28
	PUSH29
	PUSH30
	PUSH31
	PUSH32
	DUP1
	DUP2
	DUP3
	DUP4
	DUP5
	DUP6
	DUP7
	DUP8
	DUP9
	DUP10
	DUP11
	DUP12
	DUP13
	DUP14
	DUP15
	DUP16
	SWAP1
	SWAP2
	SWAP3
	SWAP4
	SWAP5
	SWAP6
	SWAP7
	SWAP8
	SWAP9
	SWAP10
	SWAP11
	SWAP12
	SWAP13
	SWAP14
	SWAP15
	SWAP16
)

const (
	LOG0 OpCode = 0xa0 + iota
	LOG1
	LOG2
	LOG3
	LOG4
)

const (
	// 0xf0 range - closures
	CREATE OpCode = 0xf0 + iota
	CALL
	CALLCODE
	RETURN
	DELEGATECALL

	INVALID      = 0xfe
	REVERT       = 0xfd
	SELFDESTRUCT = 0xff
)

// Since the opcodes aren't all in order we can't use a regular slice
var opCodeToString = map[OpCode]string{
	// 0x0 range - arithmetic ops
	STOP:       "STOP",
	ADD:        "ADD",
	MUL:        "MUL",
	SUB:        "SUB",
	DIV:        "DIV",
	SDIV:       "SDIV",
	MOD:        "MOD",
	SMOD:       "SMOD",
	EXP:        "EXP",
	NOT:        "NOT",
	LT:         "LT",
	GT:         "GT",
	SLT:        "SLT",
	SGT:        "SGT",
	EQ:         "EQ",
	ISZERO:     "ISZERO",
	SIGNEXTEND: "SIGNEXTEND",

	// 0x10 range - bit ops
	AND:    "AND",
	OR:     "OR",
	XOR:    "XOR",
	BYTE:   "BYTE",
	ADDMOD: "ADDMOD",
	MULMOD: "MULMOD",

	// 0x20 range - crypto
	SHA3: "SHA3",

	// 0x30 range - closure state
	ADDRESS:      "ADDRESS",
	BALANCE:      "BALANCE",
	ORIGIN:       "ORIGIN",
	CALLER:       "CALLER",
	CALLVALUE:    "CALLVALUE",
	CALLDATALOAD: "CALLDATALOAD",
	CALLDATASIZE: "CALLDATASIZE",
	CALLDATACOPY: "CALLDATACOPY",
	CODESIZE:     "CODESIZE",
	CODECOPY:     "CODECOPY",
	GASPRICE:     "TXGASPRICE",

	// 0x40 range - block operations
	BLOCKHASH:   "BLOCKHASH",
	COINBASE:    "COINBASE",
	TIMESTAMP:   "TIMESTAMP",
	NUMBER:      "NUMBER",
	DIFFICULTY:  "DIFFICULTY",
	GASLIMIT:    "GASLIMIT",
	EXTCODESIZE: "EXTCODESIZE",
	EXTCODECOPY: "EXTCODECOPY",

	// 0x50 range - 'storage' and execution
	POP: "POP",
	//DUP:     "DUP",
	//SWAP:    "SWAP",
	MLOAD:    "MLOAD",
	MSTORE:   "MSTORE",
	MSTORE8:  "MSTORE8",
	SLOAD:    "SLOAD",
	SSTORE:   "SSTORE",
	JUMP:     "JUMP",
	JUMPI:    "JUMPI",
	PC:       "PC",
	MSIZE:    "MSIZE",
	GAS:      "GAS",
	JUMPDEST: "JUMPDEST",

	// 0x60 range - push
	PUSH1:  "PUSH1",
	PUSH2:  "PUSH2",
	PUSH3:  "PUSH3",
	PUSH4:  "PUSH4",
	PUSH5:  "PUSH5",
	PUSH6:  "PUSH6",
	PUSH7:  "PUSH7",
	PUSH8:  "PUSH8",
	PUSH9:  "PUSH9",
	PUSH10: "PUSH10",
	PUSH11: "PUSH11",
	PUSH12: "PUSH12",
	PUSH13: "PUSH13",
	PUSH14: "PUSH14",
	PUSH15: "PUSH15",
	PUSH16: "PUSH16",
	PUSH17: "PUSH17",
	PUSH18: "PUSH18",
	PUSH19: "PUSH19",
	PUSH20: "PUSH20",
	PUSH21: "PUSH21",
	PUSH22: "PUSH22",
	PUSH23: "PUSH23",
	PUSH24: "PUSH24",
	PUSH25: "PUSH25",
	PUSH26: "PUSH26",
	PUSH27: "PUSH27",
	PUSH28: "PUSH28",
	PUSH29: "PUSH29",
	PUSH30: "PUSH30",
	PUSH31: "PUSH31",
	PUSH32: "PUSH32",

	DUP1:  "DUP1",
	DUP2:  "DUP2",
	DUP3:  "DUP3",
	DUP4:  "DUP4",
	DUP5:  "DUP5",
	DUP6:  "DUP6",
	DUP7:  "DUP7",
	DUP8:  "DUP8",
	DUP9:  "DUP9",
	DUP10: "DUP10",
	DUP11: "DUP11",
	DUP12: "DUP12",
	DUP13: "DUP13",
	DUP14: "DUP14",
	DUP15: "DUP15",
	DUP16: "DUP16",

	SWAP1:  "SWAP1",
	SWAP2:  "SWAP2",
	SWAP3:  "SWAP3",
	SWAP4:  "SWAP4",
	SWAP5:  "SWAP5",
	SWAP6:  "SWAP6",
	SWAP7:  "SWAP7",
	SWAP8:  "SWAP8",
	SWAP9:  "SWAP9",
	SWAP10: "SWAP10",
	SWAP11: "SWAP11",
	SWAP12: "SWAP12",
	SWAP13: "SWAP13",
	SWAP14: "SWAP14",
	SWAP15: "SWAP15",
	SWAP16: "SWAP16",
	LOG0:   "LOG0",
	LOG1:   "LOG1",
	LOG2:   "LOG2",
	LOG3:   "LOG3",
	LOG4:   "LOG4",

	// 0xf0 range
	CREATE:       "CREATE",
	CALL:         "CALL",
	RETURN:       "RETURN",
	CALLCODE:     "CALLCODE",
	DELEGATECALL: "DELEGATECALL",
	INVALID:      "INVALID",
	REVERT:       "REVERT",
	SELFDESTRUCT: "SELFDESTRUCT",
}

func (o OpCode) String() string {
	str := opCodeToString[o]
	if len(str) == 0 {
		return fmt.Sprintf("Missing opcode 0x%x", int(o))
	}

	return str
}

var opCodeToStackReads = map[OpCode]int{
	// 0x0 range - arithmetic ops
	STOP:       0,
	ADD:        2,
	MUL:        2,
	SUB:        2,
	DIV:        2,
	SDIV:       2,
	MOD:        2,
	SMOD:       2,
	EXP:        2,
	NOT:        1,
	LT:         2,
	GT:         2,
	SLT:        2,
	SGT:        2,
	EQ:         2,
	ISZERO:     1,
	SIGNEXTEND: 1,

	// 0x10 range - bit ops
	AND:    2,
	OR:     2,
	XOR:    2,
	BYTE:   2,
	ADDMOD: 3,
	MULMOD: 3,

	// 0x20 range - crypto
	SHA3: 2,

	// 0x30 range - closure state
	ADDRESS:      0,
	BALANCE:      1,
	ORIGIN:       0,
	CALLER:       0,
	CALLVALUE:    0,
	CALLDATALOAD: 1,
	CALLDATASIZE: 0,
	CALLDATACOPY: 3,
	CODESIZE:     0,
	CODECOPY:     3,
	GASPRICE:     0,

	// 0x40 range - block operations
	BLOCKHASH:   1,
	COINBASE:    0,
	TIMESTAMP:   0,
	NUMBER:      0,
	DIFFICULTY:  0,
	GASLIMIT:    0,
	EXTCODESIZE: 1,
	EXTCODECOPY: 4,

	// 0x50 range - 'storage' and execution
	POP: 1,
	//DUP:     "DUP",
	//SWAP:    "SWAP",
	MLOAD:    1,
	MSTORE:   2,
	MSTORE8:  2,
	SLOAD:    1,
	SSTORE:   2,
	JUMP:     1,
	JUMPI:    2,
	PC:       0,
	MSIZE:    0,
	GAS:      0,
	JUMPDEST: 0,

	// 0x60 range - push
	PUSH1:  0,
	PUSH2:  0,
	PUSH3:  0,
	PUSH4:  0,
	PUSH5:  0,
	PUSH6:  0,
	PUSH7:  0,
	PUSH8:  0,
	PUSH9:  0,
	PUSH10: 0,
	PUSH11: 0,
	PUSH12: 0,
	PUSH13: 0,
	PUSH14: 0,
	PUSH15: 0,
	PUSH16: 0,
	PUSH17: 0,
	PUSH18: 0,
	PUSH19: 0,
	PUSH20: 0,
	PUSH21: 0,
	PUSH22: 0,
	PUSH23: 0,
	PUSH24: 0,
	PUSH25: 0,
	PUSH26: 0,
	PUSH27: 0,
	PUSH28: 0,
	PUSH29: 0,
	PUSH30: 0,
	PUSH31: 0,
	PUSH32: 0,

	DUP1:  1,
	DUP2:  2,
	DUP3:  3,
	DUP4:  4,
	DUP5:  5,
	DUP6:  6,
	DUP7:  7,
	DUP8:  8,
	DUP9:  9,
	DUP10: 10,
	DUP11: 11,
	DUP12: 12,
	DUP13: 13,
	DUP14: 14,
	DUP15: 15,
	DUP16: 16,

	SWAP1:  2,
	SWAP2:  3,
	SWAP3:  4,
	SWAP4:  5,
	SWAP5:  6,
	SWAP6:  7,
	SWAP7:  8,
	SWAP8:  9,
	SWAP9:  10,
	SWAP10: 11,
	SWAP11: 12,
	SWAP12: 13,
	SWAP13: 14,
	SWAP14: 15,
	SWAP15: 16,
	SWAP16: 17,
	LOG0:   2,
	LOG1:   3,
	LOG2:   4,
	LOG3:   5,
	LOG4:   6,

	// 0xf0 range
	CREATE:       3,
	CALL:         7,
	RETURN:       2,
	CALLCODE:     7,
	DELEGATECALL: 6,
	INVALID:      0,
	REVERT:       0,
	SELFDESTRUCT: 1,
}

func (o OpCode) StackReads() int {
	return opCodeToStackReads[o]
}

var opCodeToStackWrites = map[OpCode]int{
	// 0x0 range - arithmetic ops
	STOP:       0,
	ADD:        1,
	MUL:        1,
	SUB:        1,
	DIV:        1,
	SDIV:       1,
	MOD:        1,
	SMOD:       1,
	EXP:        1,
	NOT:        1,
	LT:         1,
	GT:         1,
	SLT:        1,
	SGT:        1,
	EQ:         1,
	ISZERO:     1,
	SIGNEXTEND: 1,

	// 0x10 range - bit ops
	AND:    1,
	OR:     1,
	XOR:    1,
	BYTE:   1,
	ADDMOD: 1,
	MULMOD: 1,

	// 0x20 range - crypto
	SHA3: 1,

	// 0x30 range - closure state
	ADDRESS:      1,
	BALANCE:      1,
	ORIGIN:       1,
	CALLER:       1,
	CALLVALUE:    1,
	CALLDATALOAD: 1,
	CALLDATASIZE: 1,
	CALLDATACOPY: 0,
	CODESIZE:     1,
	CODECOPY:     0,
	GASPRICE:     1,

	// 0x40 range - block operations
	BLOCKHASH:   1,
	COINBASE:    1,
	TIMESTAMP:   1,
	NUMBER:      1,
	DIFFICULTY:  1,
	GASLIMIT:    1,
	EXTCODESIZE: 1,
	EXTCODECOPY: 0,

	// 0x50 range - 'storage' and execution
	POP: 0,
	//DUP:     "DUP",
	//SWAP:    "SWAP",
	MLOAD:    1,
	MSTORE:   0,
	MSTORE8:  0,
	SLOAD:    1,
	SSTORE:   0,
	JUMP:     0,
	JUMPI:    0,
	PC:       1,
	MSIZE:    1,
	GAS:      1,
	JUMPDEST: 0,

	// 0x60 range - push
	PUSH1:  1,
	PUSH2:  1,
	PUSH3:  1,
	PUSH4:  1,
	PUSH5:  1,
	PUSH6:  1,
	PUSH7:  1,
	PUSH8:  1,
	PUSH9:  1,
	PUSH10: 1,
	PUSH11: 1,
	PUSH12: 1,
	PUSH13: 1,
	PUSH14: 1,
	PUSH15: 1,
	PUSH16: 1,
	PUSH17: 1,
	PUSH18: 1,
	PUSH19: 1,
	PUSH20: 1,
	PUSH21: 1,
	PUSH22: 1,
	PUSH23: 1,
	PUSH24: 1,
	PUSH25: 1,
	PUSH26: 1,
	PUSH27: 1,
	PUSH28: 1,
	PUSH29: 1,
	PUSH30: 1,
	PUSH31: 1,
	PUSH32: 1,

	DUP1:  2,
	DUP2:  3,
	DUP3:  4,
	DUP4:  5,
	DUP5:  6,
	DUP6:  7,
	DUP7:  8,
	DUP8:  9,
	DUP9:  10,
	DUP10: 11,
	DUP11: 12,
	DUP12: 13,
	DUP13: 14,
	DUP14: 15,
	DUP15: 16,
	DUP16: 17,

	SWAP1:  2,
	SWAP2:  3,
	SWAP3:  4,
	SWAP4:  5,
	SWAP5:  6,
	SWAP6:  7,
	SWAP7:  8,
	SWAP8:  9,
	SWAP9:  10,
	SWAP10: 11,
	SWAP11: 12,
	SWAP12: 13,
	SWAP13: 14,
	SWAP14: 15,
	SWAP15: 16,
	SWAP16: 17,
	LOG0:   0,
	LOG1:   0,
	LOG2:   0,
	LOG3:   0,
	LOG4:   0,

	// 0xf0 range
	CREATE:       1,
	CALL:         1,
	RETURN:       0,
	CALLCODE:     1,
	DELEGATECALL: 1,
	INVALID:      0,
	REVERT:       0,
	SELFDESTRUCT: 0,
}

func (o OpCode) StackWrites() int {
	return opCodeToStackWrites[o]
}

var stringToOp = map[string]OpCode{
	"STOP":         STOP,
	"ADD":          ADD,
	"MUL":          MUL,
	"SUB":          SUB,
	"DIV":          DIV,
	"SDIV":         SDIV,
	"MOD":          MOD,
	"SMOD":         SMOD,
	"EXP":          EXP,
	"NOT":          NOT,
	"LT":           LT,
	"GT":           GT,
	"SLT":          SLT,
	"SGT":          SGT,
	"EQ":           EQ,
	"ISZERO":       ISZERO,
	"SIGNEXTEND":   SIGNEXTEND,
	"AND":          AND,
	"OR":           OR,
	"XOR":          XOR,
	"BYTE":         BYTE,
	"ADDMOD":       ADDMOD,
	"MULMOD":       MULMOD,
	"SHA3":         SHA3,
	"ADDRESS":      ADDRESS,
	"BALANCE":      BALANCE,
	"ORIGIN":       ORIGIN,
	"CALLER":       CALLER,
	"CALLVALUE":    CALLVALUE,
	"CALLDATALOAD": CALLDATALOAD,
	"CALLDATASIZE": CALLDATASIZE,
	"CALLDATACOPY": CALLDATACOPY,
	"DELEGATECALL": DELEGATECALL,
	"CODESIZE":     CODESIZE,
	"CODECOPY":     CODECOPY,
	"GASPRICE":     GASPRICE,
	"BLOCKHASH":    BLOCKHASH,
	"COINBASE":     COINBASE,
	"TIMESTAMP":    TIMESTAMP,
	"NUMBER":       NUMBER,
	"DIFFICULTY":   DIFFICULTY,
	"GASLIMIT":     GASLIMIT,
	"EXTCODESIZE":  EXTCODESIZE,
	"EXTCODECOPY":  EXTCODECOPY,
	"POP":          POP,
	"MLOAD":        MLOAD,
	"MSTORE":       MSTORE,
	"MSTORE8":      MSTORE8,
	"SLOAD":        SLOAD,
	"SSTORE":       SSTORE,
	"JUMP":         JUMP,
	"JUMPI":        JUMPI,
	"PC":           PC,
	"MSIZE":        MSIZE,
	"GAS":          GAS,
	"JUMPDEST":     JUMPDEST,
	"PUSH1":        PUSH1,
	"PUSH2":        PUSH2,
	"PUSH3":        PUSH3,
	"PUSH4":        PUSH4,
	"PUSH5":        PUSH5,
	"PUSH6":        PUSH6,
	"PUSH7":        PUSH7,
	"PUSH8":        PUSH8,
	"PUSH9":        PUSH9,
	"PUSH10":       PUSH10,
	"PUSH11":       PUSH11,
	"PUSH12":       PUSH12,
	"PUSH13":       PUSH13,
	"PUSH14":       PUSH14,
	"PUSH15":       PUSH15,
	"PUSH16":       PUSH16,
	"PUSH17":       PUSH17,
	"PUSH18":       PUSH18,
	"PUSH19":       PUSH19,
	"PUSH20":       PUSH20,
	"PUSH21":       PUSH21,
	"PUSH22":       PUSH22,
	"PUSH23":       PUSH23,
	"PUSH24":       PUSH24,
	"PUSH25":       PUSH25,
	"PUSH26":       PUSH26,
	"PUSH27":       PUSH27,
	"PUSH28":       PUSH28,
	"PUSH29":       PUSH29,
	"PUSH30":       PUSH30,
	"PUSH31":       PUSH31,
	"PUSH32":       PUSH32,
	"DUP1":         DUP1,
	"DUP2":         DUP2,
	"DUP3":         DUP3,
	"DUP4":         DUP4,
	"DUP5":         DUP5,
	"DUP6":         DUP6,
	"DUP7":         DUP7,
	"DUP8":         DUP8,
	"DUP9":         DUP9,
	"DUP10":        DUP10,
	"DUP11":        DUP11,
	"DUP12":        DUP12,
	"DUP13":        DUP13,
	"DUP14":        DUP14,
	"DUP15":        DUP15,
	"DUP16":        DUP16,
	"SWAP1":        SWAP1,
	"SWAP2":        SWAP2,
	"SWAP3":        SWAP3,
	"SWAP4":        SWAP4,
	"SWAP5":        SWAP5,
	"SWAP6":        SWAP6,
	"SWAP7":        SWAP7,
	"SWAP8":        SWAP8,
	"SWAP9":        SWAP9,
	"SWAP10":       SWAP10,
	"SWAP11":       SWAP11,
	"SWAP12":       SWAP12,
	"SWAP13":       SWAP13,
	"SWAP14":       SWAP14,
	"SWAP15":       SWAP15,
	"SWAP16":       SWAP16,
	"LOG0":         LOG0,
	"LOG1":         LOG1,
	"LOG2":         LOG2,
	"LOG3":         LOG3,
	"LOG4":         LOG4,
	"CREATE":       CREATE,
	"CALL":         CALL,
	"RETURN":       RETURN,
	"CALLCODE":     CALLCODE,
	"INVALID":      INVALID,
	"REVERT":       REVERT,
	"SELFDESTRUCT": SELFDESTRUCT,
}

func StringToOp(str string) OpCode {
	return stringToOp[str]
}
