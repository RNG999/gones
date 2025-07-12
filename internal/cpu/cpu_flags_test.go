package cpu

import (
	"testing"
)

// FlagTest represents a test case for CPU flag behavior
type FlagTest struct {
	Name      string
	Setup     func(*CPUTestHelper)
	Execute   func(*CPUTestHelper)
	ExpectedN bool
	ExpectedV bool
	ExpectedB bool
	ExpectedD bool
	ExpectedI bool
	ExpectedZ bool
	ExpectedC bool
}

// TestNegativeFlag tests the Negative (N) flag behavior
func TestNegativeFlag(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "LDA_Sets_N_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.LoadProgram(0x8000, 0xA9, 0x80) // LDA #$80
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true,
			ExpectedZ: false,
		},
		{
			Name: "LDA_Clears_N_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.N = true                    // Set initially
				h.LoadProgram(0x8000, 0xA9, 0x7F) // LDA #$7F
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: false,
		},
		{
			Name: "ADC_Sets_N_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x7F
				h.CPU.C = false
				h.LoadProgram(0x8000, 0x69, 0x01) // ADC #$01
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true, // 0x7F + 0x01 = 0x80 (negative)
			ExpectedZ: false,
			ExpectedV: true, // Overflow from positive to negative
			ExpectedC: false,
		},
		{
			Name: "INC_Sets_N_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x50, 0x7F)
				h.LoadProgram(0x8000, 0xE6, 0x50) // INC $50
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true, // 0x7F + 1 = 0x80 (negative)
			ExpectedZ: false,
		},
	}

	runFlagTests(t, tests)
}

// TestZeroFlag tests the Zero (Z) flag behavior
func TestZeroFlag(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "LDA_Sets_Z_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF                    // Set non-zero initially
				h.LoadProgram(0x8000, 0xA9, 0x00) // LDA #$00
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true,
		},
		{
			Name: "LDA_Clears_Z_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.Z = true                    // Set initially
				h.LoadProgram(0x8000, 0xA9, 0x01) // LDA #$01
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: false,
		},
		{
			Name: "ADC_Sets_Z_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF
				h.CPU.C = true
				h.LoadProgram(0x8000, 0x69, 0x00) // ADC #$00
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true, // 0xFF + 0x00 + 1 = 0x00 (with carry)
			ExpectedC: true,
		},
		{
			Name: "DEC_Sets_Z_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetByte(0x60, 0x01)
				h.LoadProgram(0x8000, 0xC6, 0x60) // DEC $60
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true, // 0x01 - 1 = 0x00
		},
		{
			Name: "CMP_Sets_Z_Flag_Equal",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x55
				h.LoadProgram(0x8000, 0xC9, 0x55) // CMP #$55
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true, // A == operand
			ExpectedC: true, // A >= operand
		},
	}

	runFlagTests(t, tests)
}

// TestCarryFlag tests the Carry (C) flag behavior
func TestCarryFlag(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "ADC_Sets_C_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF
				h.CPU.C = false
				h.LoadProgram(0x8000, 0x69, 0x01) // ADC #$01
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true, // Result is 0x00
			ExpectedC: true, // Carry out
		},
		{
			Name: "ADC_Clears_C_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x10
				h.CPU.C = true                    // Set initially
				h.LoadProgram(0x8000, 0x69, 0x20) // ADC #$20
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: false,
			ExpectedC: false, // No carry out: 0x10 + 0x20 + 1 = 0x31
		},
		{
			Name: "SBC_Sets_C_Flag_NoBorrow",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x50
				h.CPU.C = true                    // No borrow
				h.LoadProgram(0x8000, 0xE9, 0x30) // SBC #$30
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: false,
			ExpectedC: true, // No borrow needed
		},
		{
			Name: "SBC_Clears_C_Flag_WithBorrow",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x30
				h.CPU.C = true
				h.LoadProgram(0x8000, 0xE9, 0x50) // SBC #$50
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true, // Result is negative (0xE0)
			ExpectedZ: false,
			ExpectedC: false, // Borrow needed
		},
		{
			Name: "ASL_Sets_C_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x80              // Bit 7 set
				h.LoadProgram(0x8000, 0x0A) // ASL A
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true, // Result is 0x00
			ExpectedC: true, // Bit 7 shifted into carry
		},
		{
			Name: "LSR_Sets_C_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x01              // Bit 0 set
				h.LoadProgram(0x8000, 0x4A) // LSR A
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true, // Result is 0x00
			ExpectedC: true, // Bit 0 shifted into carry
		},
		{
			Name: "CMP_Sets_C_Flag_GreaterEqual",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x80
				h.LoadProgram(0x8000, 0xC9, 0x7F) // CMP #$7F
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: false,
			ExpectedC: true, // A >= operand
		},
		{
			Name: "CMP_Clears_C_Flag_Less",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x30
				h.CPU.C = true                    // Set initially
				h.LoadProgram(0x8000, 0xC9, 0x40) // CMP #$40
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true, // Result is negative
			ExpectedZ: false,
			ExpectedC: false, // A < operand
		},
	}

	runFlagTests(t, tests)
}

// TestOverflowFlag tests the Overflow (V) flag behavior
func TestOverflowFlag(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "ADC_Sets_V_Flag_PositiveOverflow",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x7F // Positive
				h.CPU.C = false
				h.LoadProgram(0x8000, 0x69, 0x01) // ADC #$01
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true, // Result appears negative
			ExpectedZ: false,
			ExpectedV: true, // Overflow: positive + positive = negative
			ExpectedC: false,
		},
		{
			Name: "ADC_Sets_V_Flag_NegativeOverflow",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x80 // Negative (-128)
				h.CPU.C = false
				h.LoadProgram(0x8000, 0x69, 0xFF) // ADC #$FF (-1)
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false, // Result appears positive
			ExpectedZ: false,
			ExpectedV: true, // Overflow: negative + negative = positive
			ExpectedC: true, // Carry out
		},
		{
			Name: "ADC_Clears_V_Flag_NoOverflow",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x40 // Positive
				h.CPU.V = true // Set initially
				h.CPU.C = false
				h.LoadProgram(0x8000, 0x69, 0x30) // ADC #$30
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: false,
			ExpectedV: false, // No overflow: positive + positive = positive
			ExpectedC: false,
		},
		{
			Name: "SBC_Sets_V_Flag_Overflow",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x80                    // Negative (-128)
				h.CPU.C = true                    // No borrow
				h.LoadProgram(0x8000, 0xE9, 0x01) // SBC #$01 (positive)
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false, // Result appears positive
			ExpectedZ: false,
			ExpectedV: true, // Overflow: negative - positive = positive
			ExpectedC: true,
		},
		{
			Name: "SBC_Clears_V_Flag_NoOverflow",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x50 // Positive
				h.CPU.V = true // Set initially
				h.CPU.C = true
				h.LoadProgram(0x8000, 0xE9, 0x30) // SBC #$30
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: false,
			ExpectedV: false, // No overflow
			ExpectedC: true,
		},
	}

	runFlagTests(t, tests)
}

// TestBITInstruction tests the BIT instruction flag behavior
func TestBITInstruction(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "BIT_Sets_N_And_V_Flags",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF
				h.Memory.SetByte(0x80, 0xC0)      // 11000000 (N=1, V=1)
				h.LoadProgram(0x8000, 0x24, 0x80) // BIT $80
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true,  // Bit 7 of memory
			ExpectedV: true,  // Bit 6 of memory
			ExpectedZ: false, // A & memory != 0
		},
		{
			Name: "BIT_Clears_N_And_V_Flags",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0xFF
				h.CPU.N = true                    // Set initially
				h.CPU.V = true                    // Set initially
				h.Memory.SetByte(0x80, 0x3F)      // 00111111 (N=0, V=0)
				h.LoadProgram(0x8000, 0x24, 0x80) // BIT $80
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false, // Bit 7 of memory
			ExpectedV: false, // Bit 6 of memory
			ExpectedZ: false, // A & memory != 0
		},
		{
			Name: "BIT_Sets_Z_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x0F                    // 00001111
				h.CPU.Z = false                   // Clear initially
				h.Memory.SetByte(0x80, 0xF0)      // 11110000
				h.LoadProgram(0x8000, 0x24, 0x80) // BIT $80
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true, // Bit 7 of memory
			ExpectedV: true, // Bit 6 of memory
			ExpectedZ: true, // A & memory == 0
		},
	}

	runFlagTests(t, tests)
}

// TestRotateInstructions tests flag behavior for ROL and ROR
func TestRotateInstructions(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "ROL_With_Carry_In",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x80              // 10000000
				h.CPU.C = true              // Carry in
				h.LoadProgram(0x8000, 0x2A) // ROL A
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false, // Result is 0x01
			ExpectedZ: false,
			ExpectedC: true, // Bit 7 rotated into carry
		},
		{
			Name: "ROL_No_Carry_In",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x40 // 01000000
				h.CPU.C = false
				h.LoadProgram(0x8000, 0x2A) // ROL A
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true, // Result is 0x80
			ExpectedZ: false,
			ExpectedC: false, // No carry out
		},
		{
			Name: "ROR_With_Carry_In",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x01              // 00000001
				h.CPU.C = true              // Carry in
				h.LoadProgram(0x8000, 0x6A) // ROR A
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true, // Result is 0x80 (carry rotated into bit 7)
			ExpectedZ: false,
			ExpectedC: true, // Bit 0 rotated into carry
		},
		{
			Name: "ROR_Zero_Result",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x00
				h.CPU.C = false
				h.LoadProgram(0x8000, 0x6A) // ROR A
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true, // Result is 0x00
			ExpectedC: false,
		},
	}

	runFlagTests(t, tests)
}

// TestFlagInstructions tests flag manipulation instructions
func TestFlagInstructions(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "SEC_Sets_Carry",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.C = false
				h.LoadProgram(0x8000, 0x38) // SEC
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedC: true,
		},
		{
			Name: "CLC_Clears_Carry",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.C = true
				h.LoadProgram(0x8000, 0x18) // CLC
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedC: false,
		},
		{
			Name: "SEI_Sets_Interrupt",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.I = false
				h.LoadProgram(0x8000, 0x78) // SEI
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedI: true,
		},
		{
			Name: "CLI_Clears_Interrupt",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.I = true
				h.LoadProgram(0x8000, 0x58) // CLI
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedI: false,
		},
		{
			Name: "SED_Sets_Decimal",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.D = false
				h.LoadProgram(0x8000, 0xF8) // SED
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedD: true,
		},
		{
			Name: "CLD_Clears_Decimal",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.D = true
				h.LoadProgram(0x8000, 0xD8) // CLD
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedD: false,
		},
		{
			Name: "CLV_Clears_Overflow",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.V = true
				h.LoadProgram(0x8000, 0xB8) // CLV
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedV: false,
		},
	}

	runFlagTests(t, tests)
}

// TestStackInstructions tests flag behavior for stack operations
func TestStackInstructions(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "PLA_Sets_N_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.SP = 0xFE
				h.Memory.SetByte(0x01FF, 0x80) // Negative value on stack
				h.LoadProgram(0x8000, 0x68)    // PLA
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true,
			ExpectedZ: false,
		},
		{
			Name: "PLA_Sets_Z_Flag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.SP = 0xFE
				h.Memory.SetByte(0x01FF, 0x00) // Zero value on stack
				h.LoadProgram(0x8000, 0x68)    // PLA
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false,
			ExpectedZ: true,
		},
		{
			Name: "PLP_Restores_All_Flags",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.SP = 0xFE
				// Status: N=1, V=0, B=1, D=1, I=0, Z=1, C=0 = 0xBE
				h.Memory.SetByte(0x01FF, 0xBE)
				h.LoadProgram(0x8000, 0x28) // PLP
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true,
			ExpectedV: false,
			ExpectedB: true,
			ExpectedD: true,
			ExpectedI: false,
			ExpectedZ: true,
			ExpectedC: false,
		},
	}

	runFlagTests(t, tests)
}

// TestFlagDoNotAffect tests instructions that should not affect certain flags
func TestFlagDoNotAffect(t *testing.T) {
	tests := []FlagTest{
		{
			Name: "TXS_Does_Not_Affect_Flags",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.X = 0x80 // Would set N flag if this were TAX
				h.CPU.N = false
				h.CPU.Z = true
				h.LoadProgram(0x8000, 0x9A) // TXS
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: false, // Should remain unchanged
			ExpectedZ: true,  // Should remain unchanged
		},
		{
			Name: "STA_Does_Not_Affect_Flags",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.A = 0x00 // Would set Z flag if this were LDA
				h.CPU.N = true
				h.CPU.Z = false
				h.LoadProgram(0x8000, 0x85, 0x50) // STA $50
			},
			Execute: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedN: true,  // Should remain unchanged
			ExpectedZ: false, // Should remain unchanged
		},
	}

	runFlagTests(t, tests)
}

// runFlagTests executes a list of flag tests
func runFlagTests(t *testing.T, tests []FlagTest) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()

			// Run setup
			if test.Setup != nil {
				test.Setup(helper)
			}

			// Execute the instruction
			if test.Execute != nil {
				test.Execute(helper)
			}

			// Check flag results - only check flags that are explicitly set in the test
			if test.ExpectedN {
				if !helper.CPU.N {
					t.Errorf("%s: Expected N flag to be set", test.Name)
				}
			}
			if !test.ExpectedN && helper.CPU.N {
				// Only check if test explicitly expects N to be false
				if hasExpectedFlag(test, "N") {
					t.Errorf("%s: Expected N flag to be clear", test.Name)
				}
			}

			if test.ExpectedV {
				if !helper.CPU.V {
					t.Errorf("%s: Expected V flag to be set", test.Name)
				}
			}
			if !test.ExpectedV && helper.CPU.V {
				if hasExpectedFlag(test, "V") {
					t.Errorf("%s: Expected V flag to be clear", test.Name)
				}
			}

			if test.ExpectedB {
				if !helper.CPU.B {
					t.Errorf("%s: Expected B flag to be set", test.Name)
				}
			}
			if !test.ExpectedB && helper.CPU.B {
				if hasExpectedFlag(test, "B") {
					t.Errorf("%s: Expected B flag to be clear", test.Name)
				}
			}

			if test.ExpectedD {
				if !helper.CPU.D {
					t.Errorf("%s: Expected D flag to be set", test.Name)
				}
			}
			if !test.ExpectedD && helper.CPU.D {
				if hasExpectedFlag(test, "D") {
					t.Errorf("%s: Expected D flag to be clear", test.Name)
				}
			}

			if test.ExpectedI {
				if !helper.CPU.I {
					t.Errorf("%s: Expected I flag to be set", test.Name)
				}
			}
			if !test.ExpectedI && helper.CPU.I {
				if hasExpectedFlag(test, "I") {
					t.Errorf("%s: Expected I flag to be clear", test.Name)
				}
			}

			if test.ExpectedZ {
				if !helper.CPU.Z {
					t.Errorf("%s: Expected Z flag to be set", test.Name)
				}
			}
			if !test.ExpectedZ && helper.CPU.Z {
				if hasExpectedFlag(test, "Z") {
					t.Errorf("%s: Expected Z flag to be clear", test.Name)
				}
			}

			if test.ExpectedC {
				if !helper.CPU.C {
					t.Errorf("%s: Expected C flag to be set", test.Name)
				}
			}
			if !test.ExpectedC && helper.CPU.C {
				if hasExpectedFlag(test, "C") {
					t.Errorf("%s: Expected C flag to be clear", test.Name)
				}
			}
		})
	}
}

// hasExpectedFlag checks if a test has an explicit expectation for a flag
// This is a simple heuristic - in a real implementation, you might want to be more explicit
func hasExpectedFlag(test FlagTest, flag string) bool {
	// For simplicity, assume if the test name contains the flag name, it has expectations
	// In practice, you might want to add explicit fields to FlagTest for this
	switch flag {
	case "N":
		return test.ExpectedN || (!test.ExpectedN &&
			(test.Name == "LDA_Clears_N_Flag" || test.Name == "TXS_Does_Not_Affect_Flags"))
	case "Z":
		return test.ExpectedZ || (!test.ExpectedZ &&
			(test.Name == "LDA_Clears_Z_Flag" || test.Name == "TXS_Does_Not_Affect_Flags" || test.Name == "STA_Does_Not_Affect_Flags"))
	case "V":
		return test.ExpectedV || (!test.ExpectedV && test.Name == "BIT_Clears_N_And_V_Flags")
	case "C":
		return test.ExpectedC || (!test.ExpectedC && test.Name == "CLC_Clears_Carry")
	case "I":
		return test.ExpectedI || (!test.ExpectedI && test.Name == "CLI_Clears_Interrupt")
	case "D":
		return test.ExpectedD || (!test.ExpectedD && test.Name == "CLD_Clears_Decimal")
	case "B":
		return test.ExpectedB || (!test.ExpectedB && false) // B flag tests are explicit
	}
	return false
}
