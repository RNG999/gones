package cpu

import (
	"testing"
)

// TimingTest represents a test case for CPU instruction timing
type TimingTest struct {
	Name           string
	Setup          func(*CPUTestHelper)
	Opcode         uint8
	Operands       []uint8
	ExpectedCycles uint64
	Description    string
	PageBoundary   bool // Whether this test specifically tests page boundary crossing
}

// TestBasicInstructionTiming tests fundamental instruction cycle counts
func TestBasicInstructionTiming(t *testing.T) {
	tests := []TimingTest{
		// Implied addressing mode (1 byte instructions)
		{
			Name:           "NOP",
			Opcode:         0xEA,
			ExpectedCycles: 2,
			Description:    "No operation - simplest instruction",
		},
		{
			Name:           "TAX",
			Opcode:         0xAA,
			ExpectedCycles: 2,
			Description:    "Transfer A to X",
		},
		{
			Name:           "TXA",
			Opcode:         0x8A,
			ExpectedCycles: 2,
			Description:    "Transfer X to A",
		},
		{
			Name:           "TAY",
			Opcode:         0xA8,
			ExpectedCycles: 2,
			Description:    "Transfer A to Y",
		},
		{
			Name:           "TYA",
			Opcode:         0x98,
			ExpectedCycles: 2,
			Description:    "Transfer Y to A",
		},
		{
			Name:           "TSX",
			Opcode:         0xBA,
			ExpectedCycles: 2,
			Description:    "Transfer SP to X",
		},
		{
			Name:           "TXS",
			Opcode:         0x9A,
			ExpectedCycles: 2,
			Description:    "Transfer X to SP",
		},
		{
			Name:           "INX",
			Opcode:         0xE8,
			ExpectedCycles: 2,
			Description:    "Increment X",
		},
		{
			Name:           "DEX",
			Opcode:         0xCA,
			ExpectedCycles: 2,
			Description:    "Decrement X",
		},
		{
			Name:           "INY",
			Opcode:         0xC8,
			ExpectedCycles: 2,
			Description:    "Increment Y",
		},
		{
			Name:           "DEY",
			Opcode:         0x88,
			ExpectedCycles: 2,
			Description:    "Decrement Y",
		},
		{
			Name:           "CLC",
			Opcode:         0x18,
			ExpectedCycles: 2,
			Description:    "Clear carry flag",
		},
		{
			Name:           "SEC",
			Opcode:         0x38,
			ExpectedCycles: 2,
			Description:    "Set carry flag",
		},
		{
			Name:           "CLI",
			Opcode:         0x58,
			ExpectedCycles: 2,
			Description:    "Clear interrupt flag",
		},
		{
			Name:           "SEI",
			Opcode:         0x78,
			ExpectedCycles: 2,
			Description:    "Set interrupt flag",
		},
		{
			Name:           "CLD",
			Opcode:         0xD8,
			ExpectedCycles: 2,
			Description:    "Clear decimal flag",
		},
		{
			Name:           "SED",
			Opcode:         0xF8,
			ExpectedCycles: 2,
			Description:    "Set decimal flag",
		},
		{
			Name:           "CLV",
			Opcode:         0xB8,
			ExpectedCycles: 2,
			Description:    "Clear overflow flag",
		},

		// Accumulator addressing mode
		{
			Name:           "ASL_A",
			Opcode:         0x0A,
			ExpectedCycles: 2,
			Description:    "Arithmetic shift left accumulator",
		},
		{
			Name:           "LSR_A",
			Opcode:         0x4A,
			ExpectedCycles: 2,
			Description:    "Logical shift right accumulator",
		},
		{
			Name:           "ROL_A",
			Opcode:         0x2A,
			ExpectedCycles: 2,
			Description:    "Rotate left accumulator",
		},
		{
			Name:           "ROR_A",
			Opcode:         0x6A,
			ExpectedCycles: 2,
			Description:    "Rotate right accumulator",
		},

		// Immediate addressing mode (2 byte instructions)
		{
			Name:           "LDA_Immediate",
			Opcode:         0xA9,
			Operands:       []uint8{0x42},
			ExpectedCycles: 2,
			Description:    "Load accumulator immediate",
		},
		{
			Name:           "LDX_Immediate",
			Opcode:         0xA2,
			Operands:       []uint8{0x42},
			ExpectedCycles: 2,
			Description:    "Load X immediate",
		},
		{
			Name:           "LDY_Immediate",
			Opcode:         0xA0,
			Operands:       []uint8{0x42},
			ExpectedCycles: 2,
			Description:    "Load Y immediate",
		},
		{
			Name:           "ADC_Immediate",
			Opcode:         0x69,
			Operands:       []uint8{0x10},
			ExpectedCycles: 2,
			Description:    "Add with carry immediate",
		},
		{
			Name:           "SBC_Immediate",
			Opcode:         0xE9,
			Operands:       []uint8{0x10},
			ExpectedCycles: 2,
			Description:    "Subtract with carry immediate",
		},
		{
			Name:           "AND_Immediate",
			Opcode:         0x29,
			Operands:       []uint8{0x0F},
			ExpectedCycles: 2,
			Description:    "Logical AND immediate",
		},
		{
			Name:           "ORA_Immediate",
			Opcode:         0x09,
			Operands:       []uint8{0xF0},
			ExpectedCycles: 2,
			Description:    "Logical OR immediate",
		},
		{
			Name:           "EOR_Immediate",
			Opcode:         0x49,
			Operands:       []uint8{0xFF},
			ExpectedCycles: 2,
			Description:    "Exclusive OR immediate",
		},
		{
			Name:           "CMP_Immediate",
			Opcode:         0xC9,
			Operands:       []uint8{0x80},
			ExpectedCycles: 2,
			Description:    "Compare accumulator immediate",
		},
		{
			Name:           "CPX_Immediate",
			Opcode:         0xE0,
			Operands:       []uint8{0x80},
			ExpectedCycles: 2,
			Description:    "Compare X immediate",
		},
		{
			Name:           "CPY_Immediate",
			Opcode:         0xC0,
			Operands:       []uint8{0x80},
			ExpectedCycles: 2,
			Description:    "Compare Y immediate",
		},
	}

	runTimingTests(t, tests)
}

// TestZeroPageTiming tests zero page addressing mode timing
func TestZeroPageTiming(t *testing.T) {
	tests := []TimingTest{
		// Zero page loads (3 cycles)
		{
			Name:           "LDA_ZeroPage",
			Opcode:         0xA5,
			Operands:       []uint8{0x80},
			ExpectedCycles: 3,
			Description:    "Load accumulator from zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x0080, 0x42)
			},
		},
		{
			Name:           "LDX_ZeroPage",
			Opcode:         0xA6,
			Operands:       []uint8{0x90},
			ExpectedCycles: 3,
			Description:    "Load X from zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x0090, 0x33)
			},
		},
		{
			Name:           "LDY_ZeroPage",
			Opcode:         0xA4,
			Operands:       []uint8{0xA0},
			ExpectedCycles: 3,
			Description:    "Load Y from zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x00A0, 0x44)
			},
		},

		// Zero page stores (3 cycles)
		{
			Name:           "STA_ZeroPage",
			Opcode:         0x85,
			Operands:       []uint8{0x50},
			ExpectedCycles: 3,
			Description:    "Store accumulator to zero page",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x55
			},
		},
		{
			Name:           "STX_ZeroPage",
			Opcode:         0x86,
			Operands:       []uint8{0x60},
			ExpectedCycles: 3,
			Description:    "Store X to zero page",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x66
			},
		},
		{
			Name:           "STY_ZeroPage",
			Opcode:         0x84,
			Operands:       []uint8{0x70},
			ExpectedCycles: 3,
			Description:    "Store Y to zero page",
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x77
			},
		},

		// Zero page arithmetic (3 cycles)
		{
			Name:           "ADC_ZeroPage",
			Opcode:         0x65,
			Operands:       []uint8{0x80},
			ExpectedCycles: 3,
			Description:    "Add with carry from zero page",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x10
				h.Memory.SetByte(0x0080, 0x20)
			},
		},
		{
			Name:           "SBC_ZeroPage",
			Opcode:         0xE5,
			Operands:       []uint8{0x90},
			ExpectedCycles: 3,
			Description:    "Subtract with carry from zero page",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x50
				h.CPU.C = true
				h.Memory.SetByte(0x0090, 0x30)
			},
		},

		// Zero page bit test (3 cycles)
		{
			Name:           "BIT_ZeroPage",
			Opcode:         0x24,
			Operands:       []uint8{0xB0},
			ExpectedCycles: 3,
			Description:    "Bit test zero page",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0xFF
				h.Memory.SetByte(0x00B0, 0xC0)
			},
		},
	}

	runTimingTests(t, tests)
}

// TestZeroPageIndexedTiming tests zero page indexed addressing timing
func TestZeroPageIndexedTiming(t *testing.T) {
	tests := []TimingTest{
		// Zero page indexed (4 cycles)
		{
			Name:           "LDA_ZeroPageX",
			Opcode:         0xB5,
			Operands:       []uint8{0x80},
			ExpectedCycles: 4,
			Description:    "Load accumulator zero page,X",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x05
				h.Memory.SetByte(0x0085, 0x42)
			},
		},
		{
			Name:           "LDX_ZeroPageY",
			Opcode:         0xB6,
			Operands:       []uint8{0x90},
			ExpectedCycles: 4,
			Description:    "Load X zero page,Y",
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x08
				h.Memory.SetByte(0x0098, 0x33)
			},
		},
		{
			Name:           "LDY_ZeroPageX",
			Opcode:         0xB4,
			Operands:       []uint8{0xA0},
			ExpectedCycles: 4,
			Description:    "Load Y zero page,X",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x0A
				h.Memory.SetByte(0x00AA, 0x44)
			},
		},
		{
			Name:           "STA_ZeroPageX",
			Opcode:         0x95,
			Operands:       []uint8{0x50},
			ExpectedCycles: 4,
			Description:    "Store accumulator zero page,X",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x55
				h.CPU.X = 0x03
			},
		},
		{
			Name:           "STY_ZeroPageX",
			Opcode:         0x94,
			Operands:       []uint8{0x60},
			ExpectedCycles: 4,
			Description:    "Store Y zero page,X",
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x77
				h.CPU.X = 0x04
			},
		},
		{
			Name:           "STX_ZeroPageY",
			Opcode:         0x96,
			Operands:       []uint8{0x70},
			ExpectedCycles: 4,
			Description:    "Store X zero page,Y",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x88
				h.CPU.Y = 0x05
			},
		},
	}

	runTimingTests(t, tests)
}

// TestAbsoluteTiming tests absolute addressing mode timing
func TestAbsoluteTiming(t *testing.T) {
	tests := []TimingTest{
		// Absolute loads (4 cycles)
		{
			Name:           "LDA_Absolute",
			Opcode:         0xAD,
			Operands:       []uint8{0x34, 0x12}, // $1234
			ExpectedCycles: 4,
			Description:    "Load accumulator absolute",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x1234, 0x42)
			},
		},
		{
			Name:           "LDX_Absolute",
			Opcode:         0xAE,
			Operands:       []uint8{0x56, 0x34}, // $3456
			ExpectedCycles: 4,
			Description:    "Load X absolute",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x3456, 0x33)
			},
		},
		{
			Name:           "LDY_Absolute",
			Opcode:         0xAC,
			Operands:       []uint8{0x78, 0x56}, // $5678
			ExpectedCycles: 4,
			Description:    "Load Y absolute",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x5678, 0x44)
			},
		},

		// Absolute stores (4 cycles)
		{
			Name:           "STA_Absolute",
			Opcode:         0x8D,
			Operands:       []uint8{0x00, 0x30}, // $3000
			ExpectedCycles: 4,
			Description:    "Store accumulator absolute",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x55
			},
		},
		{
			Name:           "STX_Absolute",
			Opcode:         0x8E,
			Operands:       []uint8{0x00, 0x40}, // $4000
			ExpectedCycles: 4,
			Description:    "Store X absolute",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x66
			},
		},
		{
			Name:           "STY_Absolute",
			Opcode:         0x8C,
			Operands:       []uint8{0x00, 0x50}, // $5000
			ExpectedCycles: 4,
			Description:    "Store Y absolute",
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x77
			},
		},

		// Absolute jumps (3 cycles)
		{
			Name:           "JMP_Absolute",
			Opcode:         0x4C,
			Operands:       []uint8{0x00, 0x80}, // $8000
			ExpectedCycles: 3,
			Description:    "Jump absolute",
		},

		// Absolute bit test (4 cycles)
		{
			Name:           "BIT_Absolute",
			Opcode:         0x2C,
			Operands:       []uint8{0x00, 0x60}, // $6000
			ExpectedCycles: 4,
			Description:    "Bit test absolute",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0xFF
				h.Memory.SetByte(0x6000, 0xC0)
			},
		},
	}

	runTimingTests(t, tests)
}

// TestAbsoluteIndexedTiming tests absolute indexed addressing timing
func TestAbsoluteIndexedTiming(t *testing.T) {
	tests := []TimingTest{
		// Absolute indexed loads - no page crossing (4 cycles)
		{
			Name:           "LDA_AbsoluteX_NoPageCrossing",
			Opcode:         0xBD,
			Operands:       []uint8{0x00, 0x20}, // $2000
			ExpectedCycles: 4,
			Description:    "Load accumulator absolute,X (no page crossing)",
			PageBoundary:   false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x10
				h.Memory.SetByte(0x2010, 0x42)
			},
		},
		{
			Name:           "LDA_AbsoluteY_NoPageCrossing",
			Opcode:         0xB9,
			Operands:       []uint8{0x00, 0x30}, // $3000
			ExpectedCycles: 4,
			Description:    "Load accumulator absolute,Y (no page crossing)",
			PageBoundary:   false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x08
				h.Memory.SetByte(0x3008, 0x33)
			},
		},
		{
			Name:           "LDX_AbsoluteY_NoPageCrossing",
			Opcode:         0xBE,
			Operands:       []uint8{0x00, 0x40}, // $4000
			ExpectedCycles: 4,
			Description:    "Load X absolute,Y (no page crossing)",
			PageBoundary:   false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x05
				h.Memory.SetByte(0x4005, 0x44)
			},
		},

		// Absolute indexed loads - page crossing (5 cycles)
		{
			Name:           "LDA_AbsoluteX_PageCrossing",
			Opcode:         0xBD,
			Operands:       []uint8{0xF0, 0x20}, // $20F0
			ExpectedCycles: 5,
			Description:    "Load accumulator absolute,X (page crossing)",
			PageBoundary:   true,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x20 // $20F0 + $20 = $2110 (crosses page)
				h.Memory.SetByte(0x2110, 0x55)
			},
		},
		{
			Name:           "LDA_AbsoluteY_PageCrossing",
			Opcode:         0xB9,
			Operands:       []uint8{0xFF, 0x30}, // $30FF
			ExpectedCycles: 5,
			Description:    "Load accumulator absolute,Y (page crossing)",
			PageBoundary:   true,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x01 // $30FF + $01 = $3100 (crosses page)
				h.Memory.SetByte(0x3100, 0x66)
			},
		},

		// Absolute indexed stores - always extra cycle (5 cycles)
		{
			Name:           "STA_AbsoluteX",
			Opcode:         0x9D,
			Operands:       []uint8{0x00, 0x50}, // $5000
			ExpectedCycles: 5,
			Description:    "Store accumulator absolute,X (always extra cycle)",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x77
				h.CPU.X = 0x10
			},
		},
		{
			Name:           "STA_AbsoluteY",
			Opcode:         0x99,
			Operands:       []uint8{0x00, 0x60}, // $6000
			ExpectedCycles: 5,
			Description:    "Store accumulator absolute,Y (always extra cycle)",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x88
				h.CPU.Y = 0x08
			},
		},
	}

	runTimingTests(t, tests)
}

// TestIndirectTiming tests indirect addressing timing
func TestIndirectTiming(t *testing.T) {
	tests := []TimingTest{
		// Indirect jump (5 cycles)
		{
			Name:           "JMP_Indirect",
			Opcode:         0x6C,
			Operands:       []uint8{0x00, 0x30}, // ($3000)
			ExpectedCycles: 5,
			Description:    "Jump indirect",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetBytes(0x3000, 0x34, 0x12) // Jump to $1234
			},
		},

		// Indexed indirect (6 cycles)
		{
			Name:           "LDA_IndexedIndirect",
			Opcode:         0xA1,
			Operands:       []uint8{0x20},
			ExpectedCycles: 6,
			Description:    "Load accumulator ($zp,X)",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x04
				h.Memory.SetBytes(0x0024, 0x00, 0x50) // Pointer to $5000
				h.Memory.SetByte(0x5000, 0x42)
			},
		},
		{
			Name:           "STA_IndexedIndirect",
			Opcode:         0x81,
			Operands:       []uint8{0x30},
			ExpectedCycles: 6,
			Description:    "Store accumulator ($zp,X)",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x55
				h.CPU.X = 0x08
				h.Memory.SetBytes(0x0038, 0x00, 0x60) // Pointer to $6000
			},
		},

		// Indirect indexed - no page crossing (5 cycles)
		{
			Name:           "LDA_IndirectIndexed_NoPageCrossing",
			Opcode:         0xB1,
			Operands:       []uint8{0x40},
			ExpectedCycles: 5,
			Description:    "Load accumulator ($zp),Y (no page crossing)",
			PageBoundary:   false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x08
				h.Memory.SetBytes(0x0040, 0x00, 0x70) // Pointer to $7000
				h.Memory.SetByte(0x7008, 0x33)        // $7000 + $08
			},
		},

		// Indirect indexed - page crossing (6 cycles)
		{
			Name:           "LDA_IndirectIndexed_PageCrossing",
			Opcode:         0xB1,
			Operands:       []uint8{0x50},
			ExpectedCycles: 6,
			Description:    "Load accumulator ($zp),Y (page crossing)",
			PageBoundary:   true,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x10
				h.Memory.SetBytes(0x0050, 0xF0, 0x70) // Pointer to $70F0
				h.Memory.SetByte(0x7100, 0x44)        // $70F0 + $10 = $7100 (page cross)
			},
		},

		// Indirect indexed store - always extra cycle (6 cycles)
		{
			Name:           "STA_IndirectIndexed",
			Opcode:         0x91,
			Operands:       []uint8{0x60},
			ExpectedCycles: 6,
			Description:    "Store accumulator ($zp),Y (always extra cycle)",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x99
				h.CPU.Y = 0x04
				h.Memory.SetBytes(0x0060, 0x00, 0x80) // Pointer to $8000
			},
		},
	}

	runTimingTests(t, tests)
}

// TestStackTiming tests stack operation timing
func TestStackTiming(t *testing.T) {
	tests := []TimingTest{
		// Stack pushes (3 cycles)
		{
			Name:           "PHA",
			Opcode:         0x48,
			ExpectedCycles: 3,
			Description:    "Push accumulator",
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x55
				h.CPU.SP = 0xFF
			},
		},
		{
			Name:           "PHP",
			Opcode:         0x08,
			ExpectedCycles: 3,
			Description:    "Push processor status",
			Setup: func(h *CPUTestHelper) {
				h.CPU.SP = 0xFF
			},
		},

		// Stack pulls (4 cycles)
		{
			Name:           "PLA",
			Opcode:         0x68,
			ExpectedCycles: 4,
			Description:    "Pull accumulator",
			Setup: func(h *CPUTestHelper) {
				h.CPU.SP = 0xFE
				h.Memory.SetByte(0x01FF, 0x42)
			},
		},
		{
			Name:           "PLP",
			Opcode:         0x28,
			ExpectedCycles: 4,
			Description:    "Pull processor status",
			Setup: func(h *CPUTestHelper) {
				h.CPU.SP = 0xFE
				h.Memory.SetByte(0x01FF, 0x33)
			},
		},
	}

	runTimingTests(t, tests)
}

// TestBranchTiming tests branch instruction timing
func TestBranchTiming(t *testing.T) {
	tests := []TimingTest{
		// Branch not taken (2 cycles)
		{
			Name:           "BNE_NotTaken",
			Opcode:         0xD0,
			Operands:       []uint8{0x10},
			ExpectedCycles: 2,
			Description:    "Branch if not equal (not taken)",
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = true // Branch will not be taken
			},
		},
		{
			Name:           "BEQ_NotTaken",
			Opcode:         0xF0,
			Operands:       []uint8{0x20},
			ExpectedCycles: 2,
			Description:    "Branch if equal (not taken)",
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = false // Branch will not be taken
			},
		},

		// Branch taken, no page crossing (3 cycles)
		{
			Name:           "BNE_Taken_NoPageCrossing",
			Opcode:         0xD0,
			Operands:       []uint8{0x10}, // +16 bytes
			ExpectedCycles: 3,
			Description:    "Branch if not equal (taken, no page crossing)",
			PageBoundary:   false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = false // Branch will be taken
			},
		},
		{
			Name:           "BEQ_Taken_PageCrossing",
			Opcode:         0xF0,
			Operands:       []uint8{0xF0}, // -16 bytes (backward, crosses page)
			ExpectedCycles: 4,
			Description:    "Branch if equal (taken, page crossing)",
			PageBoundary:   true,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = true // Branch will be taken
			},
		},

		// Branch taken, page crossing (4 cycles)
		{
			Name:           "BNE_Taken_NoPageCrossing",
			Opcode:         0xD0,
			Operands:       []uint8{0x7F}, // +127 bytes (no page crossing from $8000)
			ExpectedCycles: 3,
			Description:    "Branch if not equal (taken, no page crossing)",
			PageBoundary:   false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = false // Branch will be taken
			},
		},
		{
			Name:           "BCS_Taken_PageCrossing",
			Opcode:         0xB0,
			Operands:       []uint8{0x80}, // -128 bytes (backward page cross)
			ExpectedCycles: 4,
			Description:    "Branch if carry set (taken, page crossing)",
			PageBoundary:   true,
			Setup: func(h *CPUTestHelper) {
				h.CPU.C = true // Branch will be taken
			},
		},
	}

	runTimingTests(t, tests)
}

// TestModifyInstructions tests read-modify-write instruction timing
func TestModifyInstructions(t *testing.T) {
	tests := []TimingTest{
		// Zero page modify (5 cycles)
		{
			Name:           "INC_ZeroPage",
			Opcode:         0xE6,
			Operands:       []uint8{0x80},
			ExpectedCycles: 5,
			Description:    "Increment zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x0080, 0x40)
			},
		},
		{
			Name:           "DEC_ZeroPage",
			Opcode:         0xC6,
			Operands:       []uint8{0x90},
			ExpectedCycles: 5,
			Description:    "Decrement zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x0090, 0x50)
			},
		},
		{
			Name:           "ASL_ZeroPage",
			Opcode:         0x06,
			Operands:       []uint8{0xA0},
			ExpectedCycles: 5,
			Description:    "Arithmetic shift left zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x00A0, 0x55)
			},
		},
		{
			Name:           "LSR_ZeroPage",
			Opcode:         0x46,
			Operands:       []uint8{0xB0},
			ExpectedCycles: 5,
			Description:    "Logical shift right zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x00B0, 0xAA)
			},
		},
		{
			Name:           "ROL_ZeroPage",
			Opcode:         0x26,
			Operands:       []uint8{0xC0},
			ExpectedCycles: 5,
			Description:    "Rotate left zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x00C0, 0x80)
				h.CPU.C = true
			},
		},
		{
			Name:           "ROR_ZeroPage",
			Opcode:         0x66,
			Operands:       []uint8{0xD0},
			ExpectedCycles: 5,
			Description:    "Rotate right zero page",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x00D0, 0x01)
				h.CPU.C = true
			},
		},

		// Zero page indexed modify (6 cycles)
		{
			Name:           "INC_ZeroPageX",
			Opcode:         0xF6,
			Operands:       []uint8{0x80},
			ExpectedCycles: 6,
			Description:    "Increment zero page,X",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x05
				h.Memory.SetByte(0x0085, 0x60)
			},
		},
		{
			Name:           "DEC_ZeroPageX",
			Opcode:         0xD6,
			Operands:       []uint8{0x90},
			ExpectedCycles: 6,
			Description:    "Decrement zero page,X",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x08
				h.Memory.SetByte(0x0098, 0x70)
			},
		},

		// Absolute modify (6 cycles)
		{
			Name:           "INC_Absolute",
			Opcode:         0xEE,
			Operands:       []uint8{0x00, 0x30}, // $3000
			ExpectedCycles: 6,
			Description:    "Increment absolute",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x3000, 0x80)
			},
		},
		{
			Name:           "DEC_Absolute",
			Opcode:         0xCE,
			Operands:       []uint8{0x00, 0x40}, // $4000
			ExpectedCycles: 6,
			Description:    "Decrement absolute",
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x4000, 0x90)
			},
		},

		// Absolute indexed modify (7 cycles)
		{
			Name:           "INC_AbsoluteX",
			Opcode:         0xFE,
			Operands:       []uint8{0x00, 0x50}, // $5000
			ExpectedCycles: 7,
			Description:    "Increment absolute,X",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x10
				h.Memory.SetByte(0x5010, 0xA0)
			},
		},
		{
			Name:           "DEC_AbsoluteX",
			Opcode:         0xDE,
			Operands:       []uint8{0x00, 0x60}, // $6000
			ExpectedCycles: 7,
			Description:    "Decrement absolute,X",
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x20
				h.Memory.SetByte(0x6020, 0xB0)
			},
		},
	}

	runTimingTests(t, tests)
}

// runTimingTests executes a list of timing tests
func runTimingTests(t *testing.T, tests []TimingTest) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			// Run setup
			if test.Setup != nil {
				test.Setup(helper)
			}

			// Load instruction at PC
			operands := make([]uint8, len(test.Operands))
			copy(operands, test.Operands)
			instruction := append([]uint8{test.Opcode}, operands...)
			helper.LoadProgram(helper.CPU.PC, instruction...)

			// Clear cycle counter and execute
			helper.CPU.cycles = 0
			cycles := helper.CPU.Step()

			// Check cycle count
			if cycles != test.ExpectedCycles {
				t.Errorf("%s: Expected %d cycles, got %d - %s",
					test.Name, test.ExpectedCycles, cycles, test.Description)
			}

			// Verify CPU internal cycle counter
			if helper.CPU.cycles != test.ExpectedCycles {
				t.Errorf("%s: Expected internal cycle count %d, got %d",
					test.Name, test.ExpectedCycles, helper.CPU.cycles)
			}
		})
	}
}
