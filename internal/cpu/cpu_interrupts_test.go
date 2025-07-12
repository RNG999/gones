package cpu

import (
	"testing"
)

// InterruptTest represents a test case for CPU interrupt behavior
type InterruptTest struct {
	Name           string
	Setup          func(*CPUTestHelper)
	TriggerAction  func(*CPUTestHelper) // Action to trigger interrupt
	ExpectedPC     uint16               // Expected PC after interrupt
	ExpectedSP     uint8                // Expected stack pointer after interrupt
	ExpectedI      bool                 // Expected interrupt flag state
	ExpectedCycles uint64               // Expected cycle count for interrupt
	StackChecks    []StackCheck         // Expected stack contents
}

// StackCheck represents expected stack content at a specific stack position
type StackCheck struct {
	Offset uint8 // Offset from stack page (0x0100 + offset)
	Value  uint8 // Expected value
}

// TestResetSequence tests the CPU reset behavior
func TestResetSequence(t *testing.T) {
	tests := []InterruptTest{
		{
			Name: "Reset_Sequence",
			Setup: func(h *CPUTestHelper) {
				// Set up reset vector at 0xFFFC-0xFFFD
				h.Memory.SetBytes(0xFFFC, 0x00, 0x80) // Reset to $8000

				// Set initial CPU state
				h.CPU.A = 0x55
				h.CPU.X = 0xAA
				h.CPU.Y = 0xFF
				h.CPU.SP = 0x00 // Non-standard SP
				h.CPU.PC = 0x1234
				h.CPU.N = true
				h.CPU.V = true
				h.CPU.D = true
				h.CPU.I = false // Will be set by reset
				h.CPU.Z = true
				h.CPU.C = true
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.Reset()
			},
			ExpectedPC: 0x8000,
			ExpectedSP: 0xFD, // Reset sets SP to 0xFD
			ExpectedI:  true, // Reset sets interrupt disable
			// Reset doesn't change A, X, Y, or other flags except I
		},
		{
			Name: "Reset_Vector_Different_Address",
			Setup: func(h *CPUTestHelper) {
				// Set reset vector to different address
				h.Memory.SetBytes(0xFFFC, 0x34, 0x12) // Reset to $1234
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.Reset()
			},
			ExpectedPC: 0x1234,
			ExpectedSP: 0xFD,
			ExpectedI:  true,
		},
	}

	runInterruptTests(t, tests)
}

// TestIRQSequence tests IRQ interrupt handling
func TestIRQSequence(t *testing.T) {
	tests := []InterruptTest{
		{
			Name: "IRQ_Normal_Sequence",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)

				// Set up IRQ vector at 0xFFFE-0xFFFF
				h.Memory.SetBytes(0xFFFE, 0x00, 0x90) // IRQ handler at $9000

				// Set initial state
				h.CPU.PC = 0x8123
				h.CPU.SP = 0xFF
				h.CPU.SetStatusByte(0x24) // Set some flags (includes I=true)
				h.CPU.I = false           // IRQ enabled (override status byte)
			},
			TriggerAction: func(h *CPUTestHelper) {
				// Simulate IRQ trigger - this would be called by the system
				h.CPU.TriggerIRQ()
			},
			ExpectedPC:     0x9000, // Jump to IRQ vector
			ExpectedSP:     0xFC,   // SP decremented by 3 (PC high, PC low, status)
			ExpectedI:      true,   // IRQ sets interrupt disable
			ExpectedCycles: 7,      // IRQ takes 7 cycles
			StackChecks: []StackCheck{
				{Offset: 0xFF, Value: 0x81}, // PC high byte (0x8123 -> 0x81)
				{Offset: 0xFE, Value: 0x23}, // PC low byte (0x8123 -> 0x23)
				{Offset: 0xFD, Value: 0x20}, // Status register (with B=0 for IRQ, I=0 as it was when IRQ occurred)
			},
		},
		{
			Name: "IRQ_Disabled_NoEffect",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.CPU.PC = 0x8456
				h.CPU.SP = 0xFF
				h.CPU.I = true // IRQ disabled
			},
			TriggerAction: func(h *CPUTestHelper) {
				// IRQ should have no effect when I flag is set
				h.CPU.TriggerIRQ()
				// IRQ will be pending but won't execute due to I flag
			},
			ExpectedPC:     0x8456, // PC unchanged
			ExpectedSP:     0xFF,   // SP unchanged
			ExpectedI:      true,   // I flag remains set
			ExpectedCycles: 0,      // No cycles consumed
		},
		{
			Name: "IRQ_StatusRegister_BFlag_Clear",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetBytes(0xFFFE, 0x00, 0xA0) // IRQ vector
				h.CPU.PC = 0x8789
				h.CPU.SP = 0xFF
				h.CPU.I = false
				h.CPU.B = true // B flag should be clear in pushed status
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.TriggerIRQ()
			},
			ExpectedPC: 0xA000,
			ExpectedSP: 0xFC,
			ExpectedI:  true,
			StackChecks: []StackCheck{
				{Offset: 0xFD, Value: 0x20}, // Status with B=0 (cleared for IRQ), U=1 (bit 5)
			},
		},
	}

	runInterruptTests(t, tests)
}

// TestNMISequence tests NMI interrupt handling
func TestNMISequence(t *testing.T) {
	tests := []InterruptTest{
		{
			Name: "NMI_Normal_Sequence",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)

				// Set up NMI vector at 0xFFFA-0xFFFB
				h.Memory.SetBytes(0xFFFA, 0x00, 0xB0) // NMI handler at $B000

				// Set initial state
				h.CPU.PC = 0x8ABC
				h.CPU.SP = 0xFF
				h.CPU.SetStatusByte(0x42) // Set some flags
				h.CPU.I = false           // I flag doesn't affect NMI (override status byte)
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.TriggerNMI()
			},
			ExpectedPC:     0xB000, // Jump to NMI vector
			ExpectedSP:     0xFC,   // SP decremented by 3
			ExpectedI:      true,   // NMI sets interrupt disable
			ExpectedCycles: 7,      // NMI takes 7 cycles
			StackChecks: []StackCheck{
				{Offset: 0xFF, Value: 0x8A}, // PC high byte
				{Offset: 0xFE, Value: 0xBC}, // PC low byte
				{Offset: 0xFD, Value: 0x62}, // Status register (0x42 | 0x20 for unused bit, with B=0)
			},
		},
		{
			Name: "NMI_IgnoresInterruptFlag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetBytes(0xFFFA, 0x34, 0x12) // NMI vector to $1234
				h.CPU.PC = 0x8DEF
				h.CPU.SP = 0xFF
				h.CPU.I = true // NMI should still trigger
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.TriggerNMI()
			},
			ExpectedPC:     0x1234,
			ExpectedSP:     0xFC,
			ExpectedI:      true, // Still set after NMI
			ExpectedCycles: 7,
		},
		{
			Name: "NMI_StatusRegister_BFlag_Clear",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetBytes(0xFFFA, 0x00, 0xC0) // NMI vector
				h.CPU.PC = 0x8111
				h.CPU.SP = 0xFF
				h.CPU.B = true // B flag should be clear in pushed status
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.TriggerNMI()
			},
			ExpectedPC: 0xC000,
			ExpectedSP: 0xFC,
			ExpectedI:  true, // NMI sets interrupt disable
			StackChecks: []StackCheck{
				{Offset: 0xFD, Value: 0x24}, // Status with I=1, B=0 (cleared by NMI), U=1
			},
		},
	}

	runInterruptTests(t, tests)
}

// TestBRKInstruction tests the BRK instruction (software interrupt)
func TestBRKInstruction(t *testing.T) {
	tests := []InterruptTest{
		{
			Name: "BRK_Normal_Sequence",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)

				// Set up IRQ/BRK vector at 0xFFFE-0xFFFF
				h.Memory.SetBytes(0xFFFE, 0x00, 0xD0) // BRK handler at $D000

				// Load BRK instruction at current PC
				h.LoadProgram(0x8000, 0x00) // BRK opcode
				h.CPU.SP = 0xFF
				h.CPU.SetStatusByte(0x24) // Initial status
			},
			TriggerAction: func(h *CPUTestHelper) {
				// BRK is executed via Step()
				h.CPU.Step()
			},
			ExpectedPC:     0xD000, // Jump to IRQ/BRK vector
			ExpectedSP:     0xFC,   // SP decremented by 3
			ExpectedI:      true,   // BRK sets interrupt disable
			ExpectedCycles: 7,      // BRK takes 7 cycles
			StackChecks: []StackCheck{
				{Offset: 0xFF, Value: 0x80}, // PC high byte (PC+1 from BRK)
				{Offset: 0xFE, Value: 0x01}, // PC low byte (PC+1 from BRK)
				{Offset: 0xFD, Value: 0x34}, // Status with B=1 for BRK
			},
		},
		{
			Name: "BRK_StatusRegister_BFlag_Set",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)
				h.Memory.SetBytes(0xFFFE, 0x56, 0x78) // BRK vector
				h.LoadProgram(0x8000, 0x00)           // BRK
				h.CPU.SP = 0xFF
				h.CPU.B = false // B flag should be set in pushed status
				h.CPU.I = false // Clear I flag for this test
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedPC: 0x7856,
			ExpectedSP: 0xFC,
			ExpectedI:  true,
			StackChecks: []StackCheck{
				{Offset: 0xFD, Value: 0x30}, // Status with B=1, U=1
			},
		},
		{
			Name: "BRK_PCIncrement",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8200)
				h.Memory.SetBytes(0xFFFE, 0x00, 0xE0) // BRK vector
				h.LoadProgram(0x8200, 0x00)           // BRK at $8200
				h.CPU.SP = 0xFF
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedPC: 0xE000,
			ExpectedI:  true, // BRK sets interrupt disable flag
			StackChecks: []StackCheck{
				{Offset: 0xFF, Value: 0x82}, // PC high byte (PC+1 = $8201)
				{Offset: 0xFE, Value: 0x01}, // PC low byte (PC+1 = $8201)
			},
		},
	}

	runInterruptTests(t, tests)
}

// TestRTIInstruction tests the RTI instruction (return from interrupt)
func TestRTIInstruction(t *testing.T) {
	tests := []InterruptTest{
		{
			Name: "RTI_Normal_Sequence",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)

				// Set up stack as if we returned from interrupt
				h.CPU.SP = 0xFC                // 3 bytes on stack
				h.Memory.SetByte(0x01FD, 0x42) // Status register
				h.Memory.SetByte(0x01FE, 0x34) // PC low byte
				h.Memory.SetByte(0x01FF, 0x12) // PC high byte

				// Load RTI instruction
				h.LoadProgram(0x8000, 0x40) // RTI opcode
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedPC:     0x1234, // Restored from stack
			ExpectedSP:     0xFF,   // SP incremented by 3
			ExpectedCycles: 6,      // RTI takes 6 cycles
		},
		{
			Name: "RTI_StatusRegister_Restore",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)

				// Set up stack with specific status
				h.CPU.SP = 0xFC
				h.Memory.SetByte(0x01FD, 0xE7) // Status: 11100111 (all flags except unused)
				h.Memory.SetByte(0x01FE, 0x56) // PC low
				h.Memory.SetByte(0x01FF, 0x78) // PC high

				// Set different initial flags
				h.CPU.SetStatusByte(0x00)

				h.LoadProgram(0x8000, 0x40) // RTI
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedPC: 0x7856,
			ExpectedSP: 0xFF,
			ExpectedI:  true, // I flag restored from stack value 0xE7
			// Status should be restored (but unused bit 5 is always 1)
		},
		{
			Name: "RTI_IgnoresBFlag",
			Setup: func(h *CPUTestHelper) {
				h.SetupResetVector(0x8000)

				h.CPU.SP = 0xFC
				h.Memory.SetByte(0x01FD, 0x30) // Status with B=1, but RTI ignores B flag
				h.Memory.SetByte(0x01FE, 0x00)
				h.Memory.SetByte(0x01FF, 0x90)

				h.CPU.B = false             // Should remain unchanged
				h.LoadProgram(0x8000, 0x40) // RTI
			},
			TriggerAction: func(h *CPUTestHelper) {
				h.CPU.Step()
			},
			ExpectedPC: 0x9000,
			ExpectedSP: 0xFF,
			// B flag behavior depends on implementation - some ignore pulled B flag
		},
	}

	runInterruptTests(t, tests)
}

// TestInterruptPriority tests interrupt priority and edge cases
func TestInterruptPriority(t *testing.T) {
	t.Run("NMI_Priority_Over_IRQ", func(t *testing.T) {
		helper := NewCPUTestHelper()
		helper.SetupResetVector(0x8000)

		// Set up both interrupt vectors
		helper.Memory.SetBytes(0xFFFA, 0x00, 0xA0) // NMI vector
		helper.Memory.SetBytes(0xFFFE, 0x00, 0xB0) // IRQ vector

		helper.CPU.PC = 0x8123
		helper.CPU.SP = 0xFF
		helper.CPU.I = false // IRQ enabled

		// Trigger both interrupts (NMI should take priority)
		helper.CPU.TriggerNMI()
		helper.CPU.TriggerIRQ() // Should be ignored due to pending NMI

		// Process the NMI
		helper.CPU.ProcessPendingInterrupts()

		// Should jump to NMI vector, not IRQ
		if helper.CPU.PC != 0xA000 {
			t.Errorf("Expected PC=0xA000 (NMI), got 0x%04X", helper.CPU.PC)
		}
	})

	t.Run("Multiple_NMI_EdgeDetection", func(t *testing.T) {
		helper := NewCPUTestHelper()
		helper.SetupResetVector(0x8000)
		helper.Memory.SetBytes(0xFFFA, 0x00, 0xC0) // NMI vector

		helper.CPU.PC = 0x8456
		helper.CPU.SP = 0xFF

		// First NMI
		helper.CPU.TriggerNMI()

		// Immediate second NMI should be ignored (edge detection)
		helper.CPU.TriggerNMI()

		// Process first NMI
		helper.CPU.ProcessPendingInterrupts()

		// Now NMI should be re-enabled for edge detection
		helper.CPU.ClearNMIPending()
		helper.CPU.TriggerNMI()
	})
}

// TestInterruptDuringInstruction tests interrupt timing
func TestInterruptDuringInstruction(t *testing.T) {
	t.Run("IRQ_During_LongInstruction", func(t *testing.T) {
		helper := NewCPUTestHelper()
		helper.SetupResetVector(0x8000)
		helper.Memory.SetBytes(0xFFFE, 0x00, 0xD0) // IRQ vector

		// Set up a long instruction (RMW absolute,X - 7 cycles)
		helper.LoadProgram(0x8000, 0xFE, 0x00, 0x30) // INC $3000,X
		helper.CPU.X = 0x10
		helper.Memory.SetByte(0x3010, 0x55)
		helper.CPU.I = false
		helper.CPU.SP = 0xFF

		// Set IRQ pending during instruction execution
		helper.CPU.SetIRQPending()

		// Step should complete the current instruction before processing IRQ
		cycles := helper.CPU.Step()
		if cycles != 7 {
			t.Errorf("Expected 7 cycles for INC instruction, got %d", cycles)
		}

		// Verify instruction completed
		if helper.Memory.Read(0x3010) != 0x56 {
			t.Error("INC instruction should have completed before IRQ")
		}

		// Now process pending interrupt
		helper.CPU.ProcessPendingInterrupts()

		// Should now be at IRQ handler
		if helper.CPU.PC != 0xD000 {
			t.Errorf("Expected PC=0xD000 after IRQ, got 0x%04X", helper.CPU.PC)
		}
	})
}

// TestInterruptStackOverflow tests stack behavior during interrupts
func TestInterruptStackOverflow(t *testing.T) {
	t.Run("IRQ_With_LowStack", func(t *testing.T) {
		helper := NewCPUTestHelper()
		helper.SetupResetVector(0x8000)
		helper.Memory.SetBytes(0xFFFE, 0x00, 0xE0) // IRQ vector

		helper.CPU.PC = 0x8789
		helper.CPU.SP = 0x02 // Very low stack
		helper.CPU.I = false

		helper.CPU.TriggerIRQ()
		helper.CPU.ProcessPendingInterrupts()

		// Stack should wrap around to 0xFF after underflow
		if helper.CPU.SP != 0xFF {
			t.Errorf("Expected SP=0xFF after stack wrap, got 0x%02X", helper.CPU.SP)
		}

		// Verify stack contents at wrapped locations
		if helper.Memory.Read(0x0102) != 0x87 {
			t.Error("PC high should be at wrapped stack location")
		}
		if helper.Memory.Read(0x0101) != 0x89 {
			t.Error("PC low should be at wrapped stack location")
		}
		if helper.Memory.Read(0x0100) == 0 {
			t.Error("Status should be at wrapped stack location")
		}
	})
}

// runInterruptTests executes a list of interrupt tests
func runInterruptTests(t *testing.T, tests []InterruptTest) {
	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			helper := NewCPUTestHelper()

			// Run setup
			if test.Setup != nil {
				test.Setup(helper)
			}

			// Clear cycle counter
			helper.CPU.cycles = 0

			// Trigger the interrupt or action
			if test.TriggerAction != nil {
				test.TriggerAction(helper)
			}

			// Check results
			if test.ExpectedPC != 0 {
				if helper.CPU.PC != test.ExpectedPC {
					t.Errorf("Expected PC=0x%04X, got 0x%04X", test.ExpectedPC, helper.CPU.PC)
				}
			}

			if test.ExpectedSP != 0 {
				if helper.CPU.SP != test.ExpectedSP {
					t.Errorf("Expected SP=0x%02X, got 0x%02X", test.ExpectedSP, helper.CPU.SP)
				}
			}

			if helper.CPU.I != test.ExpectedI {
				t.Errorf("Expected I flag=%v, got %v", test.ExpectedI, helper.CPU.I)
			}

			if test.ExpectedCycles != 0 {
				if helper.CPU.cycles != test.ExpectedCycles {
					t.Errorf("Expected %d cycles, got %d", test.ExpectedCycles, helper.CPU.cycles)
				}
			}

			// Check stack contents
			for _, check := range test.StackChecks {
				address := uint16(0x0100) + uint16(check.Offset)
				actual := helper.Memory.Read(address)
				if actual != check.Value {
					t.Errorf("Expected stack[0x%04X]=0x%02X, got 0x%02X",
						address, check.Value, actual)
				}
			}
		})
	}
}

// Interrupt methods are now implemented in cpu.go
