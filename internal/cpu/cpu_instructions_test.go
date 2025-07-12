package cpu

import (
	"testing"
)

// InstructionTest represents a test case for a single instruction
type InstructionTest struct {
	Name           string
	Setup          func(*CPUTestHelper)
	Opcode         uint8
	Operands       []uint8
	ExpectedA      uint8
	ExpectedX      uint8
	ExpectedY      uint8
	ExpectedSP     uint8
	ExpectedPC     uint16
	ExpectedN      bool
	ExpectedV      bool
	ExpectedB      bool
	ExpectedD      bool
	ExpectedI      bool
	ExpectedZ      bool
	ExpectedC      bool
	ExpectedCycles uint64
	MemoryChecks   []MemoryCheck
}

// MemoryCheck represents an expected memory state after instruction execution
type MemoryCheck struct {
	Address uint16
	Value   uint8
}

// TestLoadStoreInstructions tests all load and store instructions
func TestLoadStoreInstructions(t *testing.T) {
	tests := []InstructionTest{
		// LDA - Load Accumulator
		{
			Name:     "LDA_Immediate_Zero",
			Opcode:   0xA9,
			Operands: []uint8{0x00},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF // Set non-zero to verify change
				h.CPU.Z = false
				h.CPU.N = true
			},
			ExpectedA:      0x00,
			ExpectedPC:     0x8002,
			ExpectedZ:      true,
			ExpectedN:      false,
			ExpectedCycles: 2,
		},
		{
			Name:     "LDA_Immediate_Negative",
			Opcode:   0xA9,
			Operands: []uint8{0x80},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x00
				h.CPU.Z = true
				h.CPU.N = false
			},
			ExpectedA:      0x80,
			ExpectedPC:     0x8002,
			ExpectedZ:      false,
			ExpectedN:      true,
			ExpectedCycles: 2,
		},
		{
			Name:     "LDA_ZeroPage",
			Opcode:   0xA5,
			Operands: []uint8{0x50},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x0050, 0x42)
			},
			ExpectedA:      0x42,
			ExpectedPC:     0x8002,
			ExpectedZ:      false,
			ExpectedN:      false,
			ExpectedCycles: 3,
		},
		{
			Name:     "LDA_ZeroPageX",
			Opcode:   0xB5,
			Operands: []uint8{0x50},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x05
				h.Memory.SetByte(0x0055, 0x33)
			},
			ExpectedA:      0x33,
			ExpectedX:      0x05,
			ExpectedPC:     0x8002,
			ExpectedCycles: 4,
		},
		{
			Name:     "LDA_Absolute",
			Opcode:   0xAD,
			Operands: []uint8{0x34, 0x12}, // Little endian: 0x1234
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x1234, 0x77)
			},
			ExpectedA:      0x77,
			ExpectedPC:     0x8003,
			ExpectedCycles: 4,
		},
		{
			Name:     "LDA_AbsoluteX",
			Opcode:   0xBD,
			Operands: []uint8{0x00, 0x20}, // 0x2000 + X
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x10
				h.Memory.SetByte(0x2010, 0x88)
			},
			ExpectedA:      0x88,
			ExpectedX:      0x10,
			ExpectedPC:     0x8003,
			ExpectedCycles: 4, // No page boundary crossed
		},
		{
			Name:     "LDA_AbsoluteX_PageBoundary",
			Opcode:   0xBD,
			Operands: []uint8{0xFF, 0x20}, // 0x20FF + X = 0x2100 (page boundary)
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x01
				h.Memory.SetByte(0x2100, 0x99)
			},
			ExpectedA:      0x99,
			ExpectedX:      0x01,
			ExpectedPC:     0x8003,
			ExpectedCycles: 5, // Page boundary crossed
		},
		{
			Name:     "LDA_AbsoluteY",
			Opcode:   0xB9,
			Operands: []uint8{0x00, 0x30},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0x08
				h.Memory.SetByte(0x3008, 0xAA)
			},
			ExpectedA:      0xAA,
			ExpectedY:      0x08,
			ExpectedPC:     0x8003,
			ExpectedCycles: 4,
		},
		{
			Name:     "LDA_IndirectX",
			Opcode:   0xA1,
			Operands: []uint8{0x20},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x04
				// ($20 + X) = $24, pointer at $24 points to $3456
				h.Memory.SetBytes(0x0024, 0x56, 0x34) // Little endian
				h.Memory.SetByte(0x3456, 0xBB)
			},
			ExpectedA:      0xBB,
			ExpectedX:      0x04,
			ExpectedPC:     0x8002,
			ExpectedCycles: 6,
		},
		{
			Name:     "LDA_IndirectY",
			Opcode:   0xB1,
			Operands: []uint8{0x86},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0x10
				// Pointer at $86 = $4028, ($4028) + Y = $4038
				h.Memory.SetBytes(0x0086, 0x28, 0x40)
				h.Memory.SetByte(0x4038, 0xCC)
			},
			ExpectedA:      0xCC,
			ExpectedY:      0x10,
			ExpectedPC:     0x8002,
			ExpectedCycles: 5,
		},

		// LDX - Load X Register
		{
			Name:     "LDX_Immediate",
			Opcode:   0xA2,
			Operands: []uint8{0x55},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
			},
			ExpectedX:      0x55,
			ExpectedPC:     0x8002,
			ExpectedCycles: 2,
		},
		{
			Name:     "LDX_ZeroPage",
			Opcode:   0xA6,
			Operands: []uint8{0x33},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x0033, 0xDD)
			},
			ExpectedX:      0xDD,
			ExpectedPC:     0x8002,
			ExpectedCycles: 3,
		},
		{
			Name:     "LDX_ZeroPageY",
			Opcode:   0xB6,
			Operands: []uint8{0x33},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0x02
				h.Memory.SetByte(0x0035, 0xEE)
			},
			ExpectedX:      0xEE,
			ExpectedY:      0x02,
			ExpectedPC:     0x8002,
			ExpectedCycles: 4,
		},
		{
			Name:     "LDX_Absolute",
			Opcode:   0xAE,
			Operands: []uint8{0x00, 0x50},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x5000, 0x11)
			},
			ExpectedX:      0x11,
			ExpectedPC:     0x8003,
			ExpectedCycles: 4,
		},
		{
			Name:     "LDX_AbsoluteY",
			Opcode:   0xBE,
			Operands: []uint8{0x00, 0x60},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0x05
				h.Memory.SetByte(0x6005, 0x22)
			},
			ExpectedX:      0x22,
			ExpectedY:      0x05,
			ExpectedPC:     0x8003,
			ExpectedCycles: 4,
		},

		// LDY - Load Y Register
		{
			Name:     "LDY_Immediate",
			Opcode:   0xA0,
			Operands: []uint8{0x77},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
			},
			ExpectedY:      0x77,
			ExpectedPC:     0x8002,
			ExpectedCycles: 2,
		},

		// STA - Store Accumulator
		{
			Name:     "STA_ZeroPage",
			Opcode:   0x85,
			Operands: []uint8{0x42},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x99
			},
			ExpectedA:      0x99,
			ExpectedPC:     0x8002,
			ExpectedCycles: 3,
			MemoryChecks: []MemoryCheck{
				{Address: 0x0042, Value: 0x99},
			},
		},
		{
			Name:     "STA_ZeroPageX",
			Opcode:   0x95,
			Operands: []uint8{0x42},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xAA
				h.CPU.X = 0x08
			},
			ExpectedA:      0xAA,
			ExpectedX:      0x08,
			ExpectedPC:     0x8002,
			ExpectedCycles: 4,
			MemoryChecks: []MemoryCheck{
				{Address: 0x004A, Value: 0xAA}, // 0x42 + 0x08
			},
		},
		{
			Name:     "STA_Absolute",
			Opcode:   0x8D,
			Operands: []uint8{0x00, 0x70},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xBB
			},
			ExpectedA:      0xBB,
			ExpectedPC:     0x8003,
			ExpectedCycles: 4,
			MemoryChecks: []MemoryCheck{
				{Address: 0x7000, Value: 0xBB},
			},
		},
		{
			Name:     "STA_AbsoluteX",
			Opcode:   0x9D,
			Operands: []uint8{0x00, 0x80},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xCC
				h.CPU.X = 0x10
			},
			ExpectedA:      0xCC,
			ExpectedX:      0x10,
			ExpectedPC:     0x8003,
			ExpectedCycles: 5, // Store instructions always take extra cycle
			MemoryChecks: []MemoryCheck{
				{Address: 0x8010, Value: 0xCC},
			},
		},
		{
			Name:     "STA_AbsoluteY",
			Opcode:   0x99,
			Operands: []uint8{0x00, 0x90},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xDD
				h.CPU.Y = 0x20
			},
			ExpectedA:      0xDD,
			ExpectedY:      0x20,
			ExpectedPC:     0x8003,
			ExpectedCycles: 5,
			MemoryChecks: []MemoryCheck{
				{Address: 0x9020, Value: 0xDD},
			},
		},
		{
			Name:     "STA_IndirectX",
			Opcode:   0x81,
			Operands: []uint8{0x40},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xEE
				h.CPU.X = 0x02
				// ($40 + X) = $42, pointer at $42 points to $A000
				h.Memory.SetBytes(0x0042, 0x00, 0xA0)
			},
			ExpectedA:      0xEE,
			ExpectedX:      0x02,
			ExpectedPC:     0x8002,
			ExpectedCycles: 6,
			MemoryChecks: []MemoryCheck{
				{Address: 0xA000, Value: 0xEE},
			},
		},
		{
			Name:     "STA_IndirectY",
			Opcode:   0x91,
			Operands: []uint8{0x50},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF
				h.CPU.Y = 0x04
				// Pointer at $50 = $B000, ($B000) + Y = $B004
				h.Memory.SetBytes(0x0050, 0x00, 0xB0)
			},
			ExpectedA:      0xFF,
			ExpectedY:      0x04,
			ExpectedPC:     0x8002,
			ExpectedCycles: 6,
			MemoryChecks: []MemoryCheck{
				{Address: 0xB004, Value: 0xFF},
			},
		},

		// STX - Store X Register
		{
			Name:     "STX_ZeroPage",
			Opcode:   0x86,
			Operands: []uint8{0x60},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x11
			},
			ExpectedX:      0x11,
			ExpectedPC:     0x8002,
			ExpectedCycles: 3,
			MemoryChecks: []MemoryCheck{
				{Address: 0x0060, Value: 0x11},
			},
		},

		// STY - Store Y Register
		{
			Name:     "STY_ZeroPage",
			Opcode:   0x84,
			Operands: []uint8{0x70},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0x22
			},
			ExpectedY:      0x22,
			ExpectedPC:     0x8002,
			ExpectedCycles: 3,
			MemoryChecks: []MemoryCheck{
				{Address: 0x0070, Value: 0x22},
			},
		},
	}

	runInstructionTests(t, tests)
}

// TestArithmeticInstructions tests ADC and SBC instructions
func TestArithmeticInstructions(t *testing.T) {
	tests := []InstructionTest{
		// ADC - Add with Carry
		{
			Name:     "ADC_Immediate_NoCarry",
			Opcode:   0x69,
			Operands: []uint8{0x50},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x30
				h.CPU.C = false
			},
			ExpectedA:      0x80,
			ExpectedPC:     0x8002,
			ExpectedN:      true, // 0x80 is negative
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedV:      true, // 0x30 + 0x50 = overflow in signed arithmetic
			ExpectedCycles: 2,
		},
		{
			Name:     "ADC_Immediate_WithCarry",
			Opcode:   0x69,
			Operands: []uint8{0x01},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFE
				h.CPU.C = true
			},
			ExpectedA:      0x00, // 0xFE + 0x01 + 1 = 0x100 -> 0x00 with carry
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      true,
			ExpectedC:      true, // Carry out
			ExpectedV:      false,
			ExpectedCycles: 2,
		},
		{
			Name:     "ADC_ZeroPage",
			Opcode:   0x65,
			Operands: []uint8{0x80},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x10
				h.Memory.SetByte(0x0080, 0x20)
				h.CPU.C = false
			},
			ExpectedA:      0x30,
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedV:      false,
			ExpectedCycles: 3,
		},

		// SBC - Subtract with Carry
		{
			Name:     "SBC_Immediate_NoBorrow",
			Opcode:   0xE9,
			Operands: []uint8{0x30},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x50
				h.CPU.C = true // Carry clear = borrow
			},
			ExpectedA:      0x20, // 0x50 - 0x30 = 0x20
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedC:      true, // No borrow
			ExpectedV:      false,
			ExpectedCycles: 2,
		},
		{
			Name:     "SBC_Immediate_WithBorrow",
			Opcode:   0xE9,
			Operands: []uint8{0x01},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x00
				h.CPU.C = false // Borrow needed
			},
			ExpectedA:      0xFE, // 0x00 - 0x01 - 1 = 0xFE
			ExpectedPC:     0x8002,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedC:      false, // Borrow occurred
			ExpectedV:      false,
			ExpectedCycles: 2,
		},
	}

	runInstructionTests(t, tests)
}

// TestLogicalInstructions tests AND, ORA, EOR instructions
func TestLogicalInstructions(t *testing.T) {
	tests := []InstructionTest{
		// AND - Logical AND
		{
			Name:     "AND_Immediate",
			Opcode:   0x29,
			Operands: []uint8{0x0F},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF
			},
			ExpectedA:      0x0F,
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},
		{
			Name:     "AND_Immediate_Zero",
			Opcode:   0x29,
			Operands: []uint8{0x00},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF
			},
			ExpectedA:      0x00,
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      true,
			ExpectedCycles: 2,
		},

		// ORA - Logical OR
		{
			Name:     "ORA_Immediate",
			Opcode:   0x09,
			Operands: []uint8{0xF0},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x0F
			},
			ExpectedA:      0xFF,
			ExpectedPC:     0x8002,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},

		// EOR - Exclusive OR
		{
			Name:     "EOR_Immediate",
			Opcode:   0x49,
			Operands: []uint8{0xFF},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xAA
			},
			ExpectedA:      0x55, // 0xAA XOR 0xFF = 0x55
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},
	}

	runInstructionTests(t, tests)
}

// TestShiftRotateInstructions tests ASL, LSR, ROL, ROR instructions
func TestShiftRotateInstructions(t *testing.T) {
	tests := []InstructionTest{
		// ASL - Arithmetic Shift Left
		{
			Name:   "ASL_Accumulator",
			Opcode: 0x0A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x55 // 01010101
			},
			ExpectedA:      0xAA, // 10101010
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedCycles: 2,
		},
		{
			Name:   "ASL_Accumulator_Carry",
			Opcode: 0x0A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x80 // 10000000
			},
			ExpectedA:      0x00,
			ExpectedPC:     0x8001,
			ExpectedN:      false,
			ExpectedZ:      true,
			ExpectedC:      true, // Bit 7 shifted into carry
			ExpectedCycles: 2,
		},
		{
			Name:     "ASL_ZeroPage",
			Opcode:   0x06,
			Operands: []uint8{0x50},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x0050, 0x40) // 01000000
			},
			ExpectedPC:     0x8002,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedCycles: 5,
			MemoryChecks: []MemoryCheck{
				{Address: 0x0050, Value: 0x80}, // 10000000
			},
		},

		// LSR - Logical Shift Right
		{
			Name:   "LSR_Accumulator",
			Opcode: 0x4A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xAA // 10101010
			},
			ExpectedA:      0x55, // 01010101
			ExpectedPC:     0x8001,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedCycles: 2,
		},
		{
			Name:   "LSR_Accumulator_Carry",
			Opcode: 0x4A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x01 // 00000001
			},
			ExpectedA:      0x00,
			ExpectedPC:     0x8001,
			ExpectedN:      false,
			ExpectedZ:      true,
			ExpectedC:      true, // Bit 0 shifted into carry
			ExpectedCycles: 2,
		},

		// ROL - Rotate Left
		{
			Name:   "ROL_Accumulator_NoCarry",
			Opcode: 0x2A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x55 // 01010101
				h.CPU.C = false
			},
			ExpectedA:      0xAA, // 10101010
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedCycles: 2,
		},
		{
			Name:   "ROL_Accumulator_WithCarry",
			Opcode: 0x2A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x55 // 01010101
				h.CPU.C = true
			},
			ExpectedA:      0xAB, // 10101011 (carry rotated into bit 0)
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedCycles: 2,
		},

		// ROR - Rotate Right
		{
			Name:   "ROR_Accumulator_NoCarry",
			Opcode: 0x6A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xAA // 10101010
				h.CPU.C = false
			},
			ExpectedA:      0x55, // 01010101
			ExpectedPC:     0x8001,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedCycles: 2,
		},
		{
			Name:   "ROR_Accumulator_WithCarry",
			Opcode: 0x6A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xAA // 10101010
				h.CPU.C = true
			},
			ExpectedA:      0xD5, // 11010101 (carry rotated into bit 7)
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedC:      false,
			ExpectedCycles: 2,
		},
	}

	runInstructionTests(t, tests)
}

// TestCompareInstructions tests CMP, CPX, CPY instructions
func TestCompareInstructions(t *testing.T) {
	tests := []InstructionTest{
		// CMP - Compare Accumulator
		{
			Name:     "CMP_Immediate_Equal",
			Opcode:   0xC9,
			Operands: []uint8{0x55},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x55
			},
			ExpectedA:      0x55, // A unchanged
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      true, // Equal
			ExpectedC:      true, // A >= operand
			ExpectedCycles: 2,
		},
		{
			Name:     "CMP_Immediate_Greater",
			Opcode:   0xC9,
			Operands: []uint8{0x30},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x50
			},
			ExpectedA:      0x50,
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedC:      true, // A >= operand
			ExpectedCycles: 2,
		},
		{
			Name:     "CMP_Immediate_Less",
			Opcode:   0xC9,
			Operands: []uint8{0x80},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x30
			},
			ExpectedA:      0x30,
			ExpectedPC:     0x8002,
			ExpectedN:      true, // Result is negative
			ExpectedZ:      false,
			ExpectedC:      false, // A < operand
			ExpectedCycles: 2,
		},

		// CPX - Compare X Register
		{
			Name:     "CPX_Immediate",
			Opcode:   0xE0,
			Operands: []uint8{0x40},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x40
			},
			ExpectedX:      0x40,
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      true,
			ExpectedC:      true,
			ExpectedCycles: 2,
		},

		// CPY - Compare Y Register
		{
			Name:     "CPY_Immediate",
			Opcode:   0xC0,
			Operands: []uint8{0x60},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0x80
			},
			ExpectedY:      0x80,
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedC:      true, // Y >= operand
			ExpectedCycles: 2,
		},
	}

	runInstructionTests(t, tests)
}

// TestIncrementDecrementInstructions tests INC, DEC, INX, DEX, INY, DEY
func TestIncrementDecrementInstructions(t *testing.T) {
	tests := []InstructionTest{
		// INC - Increment Memory
		{
			Name:     "INC_ZeroPage",
			Opcode:   0xE6,
			Operands: []uint8{0x90},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x0090, 0x7F)
			},
			ExpectedPC:     0x8002,
			ExpectedN:      true, // 0x80 is negative
			ExpectedZ:      false,
			ExpectedCycles: 5,
			MemoryChecks: []MemoryCheck{
				{Address: 0x0090, Value: 0x80},
			},
		},
		{
			Name:     "INC_ZeroPage_Wrap",
			Opcode:   0xE6,
			Operands: []uint8{0x90},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x0090, 0xFF)
			},
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      true, // Wrapped to zero
			ExpectedCycles: 5,
			MemoryChecks: []MemoryCheck{
				{Address: 0x0090, Value: 0x00},
			},
		},

		// DEC - Decrement Memory
		{
			Name:     "DEC_ZeroPage",
			Opcode:   0xC6,
			Operands: []uint8{0xA0},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x00A0, 0x01)
			},
			ExpectedPC:     0x8002,
			ExpectedN:      false,
			ExpectedZ:      true, // Decremented to zero
			ExpectedCycles: 5,
			MemoryChecks: []MemoryCheck{
				{Address: 0x00A0, Value: 0x00},
			},
		},

		// INX - Increment X Register
		{
			Name:   "INX",
			Opcode: 0xE8,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x7F
			},
			ExpectedX:      0x80,
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},

		// DEX - Decrement X Register
		{
			Name:   "DEX",
			Opcode: 0xCA,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x01
			},
			ExpectedX:      0x00,
			ExpectedPC:     0x8001,
			ExpectedN:      false,
			ExpectedZ:      true,
			ExpectedCycles: 2,
		},

		// INY - Increment Y Register
		{
			Name:   "INY",
			Opcode: 0xC8,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0xFE
			},
			ExpectedY:      0xFF,
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},

		// DEY - Decrement Y Register
		{
			Name:   "DEY",
			Opcode: 0x88,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0x00
			},
			ExpectedY:      0xFF, // Wrap around
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},
	}

	runInstructionTests(t, tests)
}

// TestTransferInstructions tests register transfer instructions
func TestTransferInstructions(t *testing.T) {
	tests := []InstructionTest{
		// TAX - Transfer A to X
		{
			Name:   "TAX",
			Opcode: 0xAA,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x80
			},
			ExpectedA:      0x80,
			ExpectedX:      0x80,
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},

		// TXA - Transfer X to A
		{
			Name:   "TXA",
			Opcode: 0x8A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x00
			},
			ExpectedA:      0x00,
			ExpectedX:      0x00,
			ExpectedPC:     0x8001,
			ExpectedN:      false,
			ExpectedZ:      true,
			ExpectedCycles: 2,
		},

		// TAY - Transfer A to Y
		{
			Name:   "TAY",
			Opcode: 0xA8,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x55
			},
			ExpectedA:      0x55,
			ExpectedY:      0x55,
			ExpectedPC:     0x8001,
			ExpectedN:      false,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},

		// TYA - Transfer Y to A
		{
			Name:   "TYA",
			Opcode: 0x98,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Y = 0xFF
			},
			ExpectedA:      0xFF,
			ExpectedY:      0xFF,
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},

		// TSX - Transfer Stack Pointer to X
		{
			Name:   "TSX",
			Opcode: 0xBA,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.SP = 0x80
			},
			ExpectedX:      0x80,
			ExpectedSP:     0x80,
			ExpectedPC:     0x8001,
			ExpectedN:      true,
			ExpectedZ:      false,
			ExpectedCycles: 2,
		},

		// TXS - Transfer X to Stack Pointer
		{
			Name:   "TXS",
			Opcode: 0x9A,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0xFF
			},
			ExpectedX:      0xFF,
			ExpectedSP:     0xFF,
			ExpectedPC:     0x8001,
			ExpectedCycles: 2,
			// TXS does not affect flags
		},
	}

	runInstructionTests(t, tests)
}

// TestMiscellaneousInstructions tests BIT, NOP instructions
func TestMiscellaneousInstructions(t *testing.T) {
	tests := []InstructionTest{
		// BIT - Bit Test
		{
			Name:     "BIT_ZeroPage",
			Opcode:   0x24,
			Operands: []uint8{0x80},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF
				h.Memory.SetByte(0x0080, 0xC0) // 11000000
			},
			ExpectedA:      0xFF, // A is unchanged
			ExpectedPC:     0x8002,
			ExpectedN:      true,  // Bit 7 of memory
			ExpectedV:      true,  // Bit 6 of memory
			ExpectedZ:      false, // A & memory != 0
			ExpectedCycles: 3,
		},
		{
			Name:     "BIT_ZeroPage_Zero",
			Opcode:   0x24,
			Operands: []uint8{0x80},
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x0F
				h.Memory.SetByte(0x0080, 0x30) // 00110000
			},
			ExpectedA:      0x0F,
			ExpectedPC:     0x8002,
			ExpectedN:      false, // Bit 7 of memory
			ExpectedV:      false, // Bit 6 of memory
			ExpectedZ:      true,  // A & memory == 0
			ExpectedCycles: 3,
		},

		// NOP - No Operation
		{
			Name:   "NOP",
			Opcode: 0xEA,
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x55
				h.CPU.X = 0xAA
				h.CPU.Y = 0xFF
			},
			ExpectedA:      0x55, // All registers unchanged
			ExpectedX:      0xAA,
			ExpectedY:      0xFF,
			ExpectedPC:     0x8001,
			ExpectedCycles: 2,
		},
	}

	runInstructionTests(t, tests)
}

// runInstructionTests executes a list of instruction tests
func runInstructionTests(t *testing.T, tests []InstructionTest) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()

			// Run setup
			if test.Setup != nil {
				test.Setup(helper)
			}

			// Load instruction at PC
			operands := make([]uint8, len(test.Operands))
			copy(operands, test.Operands)
			instruction := append([]uint8{test.Opcode}, operands...)
			helper.LoadProgram(helper.CPU.PC, instruction...)

			// Execute instruction
			cycles := helper.CPU.Step()

			// Check results - only check registers that are explicitly expected
			if test.ExpectedA != 0 || test.Name == "LDA_Immediate_Zero" {
				if helper.CPU.A != test.ExpectedA {
					t.Errorf("%s: Expected A=0x%02X, got 0x%02X", test.Name, test.ExpectedA, helper.CPU.A)
				}
			}
			if test.ExpectedX != 0 || (test.ExpectedX == 0 && (test.Name == "TXA" || test.Name == "LDX_ZeroPage")) {
				if helper.CPU.X != test.ExpectedX {
					t.Errorf("%s: Expected X=0x%02X, got 0x%02X", test.Name, test.ExpectedX, helper.CPU.X)
				}
			}
			if test.ExpectedY != 0 || (test.ExpectedY == 0 && test.Name == "LDY_Immediate") {
				if helper.CPU.Y != test.ExpectedY {
					t.Errorf("%s: Expected Y=0x%02X, got 0x%02X", test.Name, test.ExpectedY, helper.CPU.Y)
				}
			}
			if test.ExpectedSP != 0 {
				if helper.CPU.SP != test.ExpectedSP {
					t.Errorf("%s: Expected SP=0x%02X, got 0x%02X", test.Name, test.ExpectedSP, helper.CPU.SP)
				}
			}
			if test.ExpectedPC != 0 {
				if helper.CPU.PC != test.ExpectedPC {
					t.Errorf("%s: Expected PC=0x%04X, got 0x%04X", test.Name, test.ExpectedPC, helper.CPU.PC)
				}
			}

			// Check specific flags that are expected to change
			// N and Z flags for load and logical operations
			if test.Name == "LDA_Immediate_Zero" || test.Name == "LDA_Immediate_Negative" ||
				test.Name == "AND_Immediate" || test.Name == "AND_Immediate_Zero" ||
				test.Name == "ORA_Immediate" || test.Name == "EOR_Immediate" {
				if helper.CPU.N != test.ExpectedN {
					t.Errorf("%s: Expected N=%v, got %v", test.Name, test.ExpectedN, helper.CPU.N)
				}
				if helper.CPU.Z != test.ExpectedZ {
					t.Errorf("%s: Expected Z=%v, got %v", test.Name, test.ExpectedZ, helper.CPU.Z)
				}
			}

			// Check other flags only if they are explicitly expected to be different from default
			if test.ExpectedV {
				if helper.CPU.V != test.ExpectedV {
					t.Errorf("%s: Expected V=%v, got %v", test.Name, test.ExpectedV, helper.CPU.V)
				}
			}
			if test.ExpectedC {
				if helper.CPU.C != test.ExpectedC {
					t.Errorf("%s: Expected C=%v, got %v", test.Name, test.ExpectedC, helper.CPU.C)
				}
			}
			// Only check B, D, I flags if they're set to true (indicating an explicit expectation)
			if test.ExpectedB {
				if helper.CPU.B != test.ExpectedB {
					t.Errorf("%s: Expected B=%v, got %v", test.Name, test.ExpectedB, helper.CPU.B)
				}
			}
			if test.ExpectedD {
				if helper.CPU.D != test.ExpectedD {
					t.Errorf("%s: Expected D=%v, got %v", test.Name, test.ExpectedD, helper.CPU.D)
				}
			}
			if test.ExpectedI {
				if helper.CPU.I != test.ExpectedI {
					t.Errorf("%s: Expected I=%v, got %v", test.Name, test.ExpectedI, helper.CPU.I)
				}
			}

			if test.ExpectedCycles != 0 {
				if cycles != test.ExpectedCycles {
					t.Errorf("%s: Expected %d cycles, got %d", test.Name, test.ExpectedCycles, cycles)
				}
			}

			// Check memory modifications
			for _, check := range test.MemoryChecks {
				helper.AssertMemory(t, test.Name, check.Address, check.Value)
			}
		})
	}
}
