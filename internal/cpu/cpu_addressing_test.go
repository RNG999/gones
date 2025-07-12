package cpu

import (
	"testing"
)

// AddressingModeTest represents a test case for addressing mode behavior
type AddressingModeTest struct {
	Name            string
	Setup           func(*CPUTestHelper)
	Opcode          uint8
	Operands        []uint8
	ExpectedAddress uint16 // Expected effective address calculated
	ExpectedValue   uint8  // Expected value read from effective address
	ExpectedCycles  uint64 // Expected cycle count
	PageBoundary    bool   // Whether test crosses page boundary
}

// TestImmediateAddressing tests immediate addressing mode
func TestImmediateAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:           "LDA_Immediate",
			Opcode:         0xA9,
			Operands:       []uint8{0x42},
			ExpectedValue:  0x42,
			ExpectedCycles: 2,
		},
		{
			Name:           "ADC_Immediate",
			Opcode:         0x69,
			Operands:       []uint8{0x33},
			ExpectedValue:  0x33,
			ExpectedCycles: 2,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			// Load instruction
			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}
		})
	}
}

// TestZeroPageAddressing tests zero page addressing mode
func TestZeroPageAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:            "LDA_ZeroPage",
			Opcode:          0xA5,
			Operands:        []uint8{0x80},
			ExpectedAddress: 0x0080,
			ExpectedValue:   0x55,
			ExpectedCycles:  3,
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x0080, 0x55)
			},
		},
		{
			Name:            "STA_ZeroPage",
			Opcode:          0x85,
			Operands:        []uint8{0x90},
			ExpectedAddress: 0x0090,
			ExpectedCycles:  3,
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0xAA
			},
		},
		{
			Name:            "INC_ZeroPage",
			Opcode:          0xE6,
			Operands:        []uint8{0xA0},
			ExpectedAddress: 0x00A0,
			ExpectedCycles:  5,
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x00A0, 0x7F)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}

			// For load instructions, verify the value was loaded correctly
			if test.ExpectedValue != 0 {
				switch test.Opcode {
				case 0xA5: // LDA
					if helper.CPU.A != test.ExpectedValue {
						t.Errorf("Expected A=0x%02X, got 0x%02X", test.ExpectedValue, helper.CPU.A)
					}
				}
			}

			// For store instructions, verify memory was written
			if test.Opcode == 0x85 { // STA
				if helper.Memory.Read(test.ExpectedAddress) != helper.CPU.A {
					t.Errorf("Expected memory[0x%04X]=0x%02X, got 0x%02X",
						test.ExpectedAddress, helper.CPU.A, helper.Memory.Read(test.ExpectedAddress))
				}
			}
		})
	}
}

// TestZeroPageIndexedAddressing tests zero page indexed addressing modes
func TestZeroPageIndexedAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:            "LDA_ZeroPageX",
			Opcode:          0xB5,
			Operands:        []uint8{0x80},
			ExpectedAddress: 0x0085, // 0x80 + 0x05
			ExpectedValue:   0x33,
			ExpectedCycles:  4,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x05
				h.Memory.SetByte(0x0085, 0x33)
			},
		},
		{
			Name:            "LDA_ZeroPageX_Wrap",
			Opcode:          0xB5,
			Operands:        []uint8{0xFF},
			ExpectedAddress: 0x0004, // 0xFF + 0x05 = 0x04 (wraps in zero page)
			ExpectedValue:   0x77,
			ExpectedCycles:  4,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x05
				h.Memory.SetByte(0x0004, 0x77)
			},
		},
		{
			Name:            "LDX_ZeroPageY",
			Opcode:          0xB6,
			Operands:        []uint8{0x70},
			ExpectedAddress: 0x0078, // 0x70 + 0x08
			ExpectedValue:   0x44,
			ExpectedCycles:  4,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x08
				h.Memory.SetByte(0x0078, 0x44)
			},
		},
		{
			Name:            "STY_ZeroPageX",
			Opcode:          0x94,
			Operands:        []uint8{0x60},
			ExpectedAddress: 0x0063, // 0x60 + 0x03
			ExpectedCycles:  4,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x03
				h.CPU.Y = 0x99
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}

			// Verify load operations
			if test.ExpectedValue != 0 {
				switch test.Opcode {
				case 0xB5: // LDA
					if helper.CPU.A != test.ExpectedValue {
						t.Errorf("Expected A=0x%02X, got 0x%02X", test.ExpectedValue, helper.CPU.A)
					}
				case 0xB6: // LDX
					if helper.CPU.X != test.ExpectedValue {
						t.Errorf("Expected X=0x%02X, got 0x%02X", test.ExpectedValue, helper.CPU.X)
					}
				}
			}

			// Verify store operations
			if test.Opcode == 0x94 { // STY
				if helper.Memory.Read(test.ExpectedAddress) != helper.CPU.Y {
					t.Errorf("Expected memory[0x%04X]=0x%02X, got 0x%02X",
						test.ExpectedAddress, helper.CPU.Y, helper.Memory.Read(test.ExpectedAddress))
				}
			}
		})
	}
}

// TestAbsoluteAddressing tests absolute addressing mode
func TestAbsoluteAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:            "LDA_Absolute",
			Opcode:          0xAD,
			Operands:        []uint8{0x34, 0x12}, // Little endian: 0x1234
			ExpectedAddress: 0x1234,
			ExpectedValue:   0x66,
			ExpectedCycles:  4,
			Setup: func(h *CPUTestHelper) {
				h.Memory.SetByte(0x1234, 0x66)
			},
		},
		{
			Name:            "STA_Absolute",
			Opcode:          0x8D,
			Operands:        []uint8{0x00, 0x30}, // 0x3000
			ExpectedAddress: 0x3000,
			ExpectedCycles:  4,
			Setup: func(h *CPUTestHelper) {
				h.CPU.A = 0x88
			},
		},
		{
			Name:            "JMP_Absolute",
			Opcode:          0x4C,
			Operands:        []uint8{0x00, 0x40}, // Jump to 0x4000
			ExpectedAddress: 0x4000,
			ExpectedCycles:  3,
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}

			// Verify load operations
			if test.ExpectedValue != 0 && test.Opcode == 0xAD { // LDA
				if helper.CPU.A != test.ExpectedValue {
					t.Errorf("Expected A=0x%02X, got 0x%02X", test.ExpectedValue, helper.CPU.A)
				}
			}

			// Verify store operations
			if test.Opcode == 0x8D { // STA
				if helper.Memory.Read(test.ExpectedAddress) != helper.CPU.A {
					t.Errorf("Expected memory[0x%04X]=0x%02X, got 0x%02X",
						test.ExpectedAddress, helper.CPU.A, helper.Memory.Read(test.ExpectedAddress))
				}
			}

			// Verify jump operations
			if test.Opcode == 0x4C { // JMP
				if helper.CPU.PC != test.ExpectedAddress {
					t.Errorf("Expected PC=0x%04X, got 0x%04X", test.ExpectedAddress, helper.CPU.PC)
				}
			}
		})
	}
}

// TestAbsoluteIndexedAddressing tests absolute indexed addressing modes
func TestAbsoluteIndexedAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:            "LDA_AbsoluteX_NoPageCrossing",
			Opcode:          0xBD,
			Operands:        []uint8{0x00, 0x20}, // 0x2000 + X
			ExpectedAddress: 0x2010,              // 0x2000 + 0x10
			ExpectedValue:   0x42,
			ExpectedCycles:  4, // No page boundary crossed
			PageBoundary:    false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x10
				h.Memory.SetByte(0x2010, 0x42)
			},
		},
		{
			Name:            "LDA_AbsoluteX_PageCrossing",
			Opcode:          0xBD,
			Operands:        []uint8{0xFF, 0x20}, // 0x20FF + X
			ExpectedAddress: 0x2110,              // 0x20FF + 0x11 = 0x2110 (page boundary)
			ExpectedValue:   0x55,
			ExpectedCycles:  5, // Page boundary crossed
			PageBoundary:    true,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x11
				h.Memory.SetByte(0x2110, 0x55)
			},
		},
		{
			Name:            "LDA_AbsoluteY_NoPageCrossing",
			Opcode:          0xB9,
			Operands:        []uint8{0x00, 0x30}, // 0x3000 + Y
			ExpectedAddress: 0x3008,              // 0x3000 + 0x08
			ExpectedValue:   0x77,
			ExpectedCycles:  4,
			PageBoundary:    false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x08
				h.Memory.SetByte(0x3008, 0x77)
			},
		},
		{
			Name:            "LDA_AbsoluteY_PageCrossing",
			Opcode:          0xB9,
			Operands:        []uint8{0xF0, 0x30}, // 0x30F0 + Y
			ExpectedAddress: 0x3100,              // 0x30F0 + 0x10 = 0x3100 (page boundary)
			ExpectedValue:   0x99,
			ExpectedCycles:  5,
			PageBoundary:    true,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x10
				h.Memory.SetByte(0x3100, 0x99)
			},
		},
		{
			Name:            "STA_AbsoluteX_AlwaysExtraCycle",
			Opcode:          0x9D,
			Operands:        []uint8{0x00, 0x40}, // 0x4000 + X
			ExpectedAddress: 0x4005,              // 0x4000 + 0x05
			ExpectedCycles:  5,                   // Store always takes extra cycle
			PageBoundary:    false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x05
				h.CPU.A = 0xAA
			},
		},
		{
			Name:            "STA_AbsoluteY_AlwaysExtraCycle",
			Opcode:          0x99,
			Operands:        []uint8{0x00, 0x50}, // 0x5000 + Y
			ExpectedAddress: 0x500A,              // 0x5000 + 0x0A
			ExpectedCycles:  5,
			PageBoundary:    false,
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x0A
				h.CPU.A = 0xBB
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}

			// Verify load operations
			if test.ExpectedValue != 0 {
				switch test.Opcode {
				case 0xBD, 0xB9: // LDA variants
					if helper.CPU.A != test.ExpectedValue {
						t.Errorf("Expected A=0x%02X, got 0x%02X", test.ExpectedValue, helper.CPU.A)
					}
				}
			}

			// Verify store operations
			switch test.Opcode {
			case 0x9D, 0x99: // STA variants
				if helper.Memory.Read(test.ExpectedAddress) != helper.CPU.A {
					t.Errorf("Expected memory[0x%04X]=0x%02X, got 0x%02X",
						test.ExpectedAddress, helper.CPU.A, helper.Memory.Read(test.ExpectedAddress))
				}
			}
		})
	}
}

// TestIndirectAddressing tests indirect addressing modes
func TestIndirectAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:            "JMP_Indirect",
			Opcode:          0x6C,
			Operands:        []uint8{0x00, 0x30}, // Indirect address at 0x3000
			ExpectedAddress: 0x4567,              // Target address stored at 0x3000
			ExpectedCycles:  5,
			Setup: func(h *CPUTestHelper) {
				// Store target address at 0x3000 (little endian)
				h.Memory.SetBytes(0x3000, 0x67, 0x45) // 0x4567
			},
		},
		{
			Name:            "JMP_Indirect_PageBoundaryBug",
			Opcode:          0x6C,
			Operands:        []uint8{0xFF, 0x30}, // Indirect address at 0x30FF
			ExpectedAddress: 0x4500,              // Bug: high byte from 0x3000, not 0x3100
			ExpectedCycles:  5,
			Setup: func(h *CPUTestHelper) {
				// Set up the famous 6502 indirect JMP bug
				h.Memory.SetByte(0x30FF, 0x00) // Low byte
				h.Memory.SetByte(0x3000, 0x45) // High byte (should be 0x3100)
				h.Memory.SetByte(0x3100, 0x67) // What high byte should be
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}

			// Verify jump target
			if test.Opcode == 0x6C { // JMP indirect
				if helper.CPU.PC != test.ExpectedAddress {
					t.Errorf("Expected PC=0x%04X, got 0x%04X", test.ExpectedAddress, helper.CPU.PC)
				}
			}
		})
	}
}

// TestIndexedIndirectAddressing tests indexed indirect addressing (zp,X)
func TestIndexedIndirectAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:            "LDA_IndexedIndirect",
			Opcode:          0xA1,
			Operands:        []uint8{0x20},
			ExpectedAddress: 0x5678,
			ExpectedValue:   0x42,
			ExpectedCycles:  6,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x04
				// ($20 + X) = $24, pointer at $24-$25 points to $5678
				h.Memory.SetBytes(0x0024, 0x78, 0x56) // Little endian: 0x5678
				h.Memory.SetByte(0x5678, 0x42)
			},
		},
		{
			Name:            "LDA_IndexedIndirect_ZeroPageWrap",
			Opcode:          0xA1,
			Operands:        []uint8{0xFF},
			ExpectedAddress: 0x1234,
			ExpectedValue:   0x55,
			ExpectedCycles:  6,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x01
				// ($FF + X) = $00 (wraps), pointer at $00-$01 points to $1234
				h.Memory.SetBytes(0x0000, 0x34, 0x12) // Little endian: 0x1234
				h.Memory.SetByte(0x1234, 0x55)
			},
		},
		{
			Name:            "STA_IndexedIndirect",
			Opcode:          0x81,
			Operands:        []uint8{0x40},
			ExpectedAddress: 0x9ABC,
			ExpectedCycles:  6,
			Setup: func(h *CPUTestHelper) {
				h.CPU.X = 0x08
				h.CPU.A = 0x77
				// ($40 + X) = $48, pointer at $48-$49 points to $9ABC
				h.Memory.SetBytes(0x0048, 0xBC, 0x9A) // Little endian: 0x9ABC
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}

			// Verify load operations
			if test.ExpectedValue != 0 && test.Opcode == 0xA1 { // LDA
				if helper.CPU.A != test.ExpectedValue {
					t.Errorf("Expected A=0x%02X, got 0x%02X", test.ExpectedValue, helper.CPU.A)
				}
			}

			// Verify store operations
			if test.Opcode == 0x81 { // STA
				if helper.Memory.Read(test.ExpectedAddress) != helper.CPU.A {
					t.Errorf("Expected memory[0x%04X]=0x%02X, got 0x%02X",
						test.ExpectedAddress, helper.CPU.A, helper.Memory.Read(test.ExpectedAddress))
				}
			}
		})
	}
}

// TestIndirectIndexedAddressing tests indirect indexed addressing (zp),Y
func TestIndirectIndexedAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:            "LDA_IndirectIndexed_NoPageCrossing",
			Opcode:          0xB1,
			Operands:        []uint8{0x60},
			ExpectedAddress: 0x2008, // 0x2000 + 0x08
			ExpectedValue:   0x33,
			ExpectedCycles:  5, // No page boundary crossed
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x08
				// Pointer at $60-$61 = $2000, ($2000) + Y = $2008
				h.Memory.SetBytes(0x0060, 0x00, 0x20) // Little endian: 0x2000
				h.Memory.SetByte(0x2008, 0x33)
			},
		},
		{
			Name:            "LDA_IndirectIndexed_PageCrossing",
			Opcode:          0xB1,
			Operands:        []uint8{0x70},
			ExpectedAddress: 0x3100, // 0x30FF + 0x01 = 0x3100 (page boundary)
			ExpectedValue:   0x44,
			ExpectedCycles:  6, // Page boundary crossed
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x01
				// Pointer at $70-$71 = $30FF, ($30FF) + Y = $3100
				h.Memory.SetBytes(0x0070, 0xFF, 0x30) // Little endian: 0x30FF
				h.Memory.SetByte(0x3100, 0x44)
			},
		},
		{
			Name:            "STA_IndirectIndexed_AlwaysExtraCycle",
			Opcode:          0x91,
			Operands:        []uint8{0x80},
			ExpectedAddress: 0x4010, // 0x4000 + 0x10
			ExpectedCycles:  6,      // Store always takes extra cycle
			Setup: func(h *CPUTestHelper) {
				h.CPU.Y = 0x10
				h.CPU.A = 0x88
				// Pointer at $80-$81 = $4000, ($4000) + Y = $4010
				h.Memory.SetBytes(0x0080, 0x00, 0x40) // Little endian: 0x4000
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}

			// Verify load operations
			if test.ExpectedValue != 0 && test.Opcode == 0xB1 { // LDA
				if helper.CPU.A != test.ExpectedValue {
					t.Errorf("Expected A=0x%02X, got 0x%02X", test.ExpectedValue, helper.CPU.A)
				}
			}

			// Verify store operations
			if test.Opcode == 0x91 { // STA
				if helper.Memory.Read(test.ExpectedAddress) != helper.CPU.A {
					t.Errorf("Expected memory[0x%04X]=0x%02X, got 0x%02X",
						test.ExpectedAddress, helper.CPU.A, helper.Memory.Read(test.ExpectedAddress))
				}
			}
		})
	}
}

// TestRelativeAddressing tests relative addressing mode for branch instructions
func TestRelativeAddressing(t *testing.T) {
	tests := []AddressingModeTest{
		{
			Name:            "BNE_Forward_NoPageCrossing",
			Opcode:          0xD0,
			Operands:        []uint8{0x10}, // +16 bytes
			ExpectedAddress: 0x8012,        // 0x8000 + 2 + 16
			ExpectedCycles:  3,             // Branch taken, no page crossing
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = false // Branch will be taken
			},
		},
		{
			Name:            "BNE_Forward_NoPageCrossing",
			Opcode:          0xD0,
			Operands:        []uint8{0x7F}, // +127 bytes
			ExpectedAddress: 0x8081,        // 0x8000 + 2 + 127 = 0x8081 (no page crossing)
			ExpectedCycles:  3,             // Branch taken, no page crossing
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = false // Branch will be taken
			},
		},
		{
			Name:            "BEQ_Backward",
			Opcode:          0xF0,
			Operands:        []uint8{0xFE}, // -2 bytes (two's complement)
			ExpectedAddress: 0x8000,        // 0x8000 + 2 - 2
			ExpectedCycles:  3,             // Branch taken, no page crossing
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = true // Branch will be taken
			},
		},
		{
			Name:            "BEQ_Backward_PageCrossing",
			Opcode:          0xF0,
			Operands:        []uint8{0x80}, // -128 bytes
			ExpectedAddress: 0x7F82,        // 0x8000 + 2 - 128 = 0x7F82 (page boundary)
			ExpectedCycles:  4,             // Branch taken, page crossing
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = true // Branch will be taken
			},
		},
		{
			Name:            "BNE_NotTaken",
			Opcode:          0xD0,
			Operands:        []uint8{0x20}, // Would jump +32 bytes
			ExpectedAddress: 0x8002,        // PC just advances past instruction
			ExpectedCycles:  2,             // Branch not taken
			Setup: func(h *CPUTestHelper) {
				h.CPU.Z = true // Branch will NOT be taken
			},
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()
			helper.SetupResetVector(0x8000)

			if test.Setup != nil {
				test.Setup(helper)
			}

			instruction := append([]uint8{test.Opcode}, test.Operands...)
			helper.LoadProgram(0x8000, instruction...)

			cycles := helper.CPU.Step()

			if test.ExpectedCycles != 0 && cycles != test.ExpectedCycles {
				t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, cycles)
			}

			// Verify PC is at expected address
			if helper.CPU.PC != test.ExpectedAddress {
				t.Errorf("Expected PC=0x%04X, got 0x%04X", test.ExpectedAddress, helper.CPU.PC)
			}
		})
	}
}
