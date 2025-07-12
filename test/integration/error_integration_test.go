package integration

import (
	"testing"
)

// ErrorTestHelper provides utilities for error condition integration testing
type ErrorTestHelper struct {
	*IntegrationTestHelper
	errorEvents []ErrorEvent
}

// ErrorEvent represents an error condition event
type ErrorEvent struct {
	EventType   string
	Description string
	Address     uint16
	Value       uint8
	Component   string
	Recovered   bool
}

// NewErrorTestHelper creates an error integration test helper
func NewErrorTestHelper() *ErrorTestHelper {
	return &ErrorTestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		errorEvents:           make([]ErrorEvent, 0),
	}
}

// LogErrorEvent logs an error condition event
func (h *ErrorTestHelper) LogErrorEvent(event ErrorEvent) {
	h.errorEvents = append(h.errorEvents, event)
}

// TestErrorConditions tests various error conditions and system robustness
func TestErrorConditions(t *testing.T) {
	t.Run("Invalid memory access patterns", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that accesses invalid memory regions
		program := []uint8{
			// Read from unmapped regions
			0xAD, 0x00, 0x50, // LDA $5000 (expansion ROM)
			0x85, 0x10, // STA $10 (save result)

			0xAD, 0x00, 0x45, // LDA $4500 (unmapped I/O)
			0x85, 0x11, // STA $11

			0xAD, 0x18, 0x40, // LDA $4018 (test mode)
			0x85, 0x12, // STA $12

			// Write to read-only regions (should be ignored)
			0xA9, 0x42, // LDA #$42
			0x8D, 0x02, 0x20, // STA $2002 (PPUSTATUS - read only)

			// Write to ROM (should be ignored)
			0x8D, 0x00, 0x80, // STA $8000 (ROM)

			// Continue execution
			0xEA,             // NOP
			0x4C, 0x14, 0x80, // JMP to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute invalid accesses
		for i := 0; i < 10; i++ {
			helper.Bus.Step()
		}

		// System should remain stable
		if helper.CPU.PC < 0x8000 {
			t.Errorf("PC left ROM area after invalid accesses: 0x%04X", helper.CPU.PC)
		}

		// Check that reads from unmapped areas returned 0 or safe values
		result1 := helper.Memory.Read(0x0010)
		result2 := helper.Memory.Read(0x0011)
		result3 := helper.Memory.Read(0x0012)

		t.Logf("Unmapped reads returned: 0x%02X, 0x%02X, 0x%02X", result1, result2, result3)

		// System should not crash from invalid accesses
		for i := 0; i < 100; i++ {
			helper.Bus.Step()
		}

		t.Log("Invalid memory access patterns test completed")
	})

	t.Run("Stack overflow/underflow conditions", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that causes stack overflow
		program := []uint8{
			// Overflow test: push until stack wraps
			0xA9, 0x55, // LDA #$55
			0x48,             // PHA (push A)
			0x4C, 0x02, 0x80, // JMP to PHA (infinite push)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		initialSP := helper.CPU.SP

		// Execute until stack wraps
		for i := 0; i < 300; i++ {
			helper.Bus.Step()

			// Check if stack wrapped around
			if helper.CPU.SP > initialSP {
				t.Logf("Stack overflow detected: SP wrapped from 0x%02X to 0x%02X after %d pushes",
					initialSP, helper.CPU.SP, i/3) // 3 steps per iteration
				break
			}
		}

		// System should handle stack overflow gracefully
		if helper.CPU.SP == initialSP {
			t.Error("Stack did not overflow as expected")
		}

		// Test stack underflow
		helper.Bus.Reset()

		// Program that causes stack underflow
		underflowProgram := []uint8{
			0x68,             // PLA (pop from empty stack)
			0x68,             // PLA (pop again)
			0x68,             // PLA (keep popping)
			0x4C, 0x00, 0x80, // JMP to continue popping
		}

		copy(romData, underflowProgram)
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		initialSP2 := helper.CPU.SP

		// Execute until stack underflows
		for i := 0; i < 300; i++ {
			helper.Bus.Step()

			// Check if stack wrapped around (underflow)
			if helper.CPU.SP < initialSP2 {
				t.Logf("Stack underflow detected: SP wrapped from 0x%02X to 0x%02X after %d pops",
					initialSP2, helper.CPU.SP, i/2) // 2 steps per iteration
				break
			}
		}

		t.Log("Stack overflow/underflow test completed")
	})

	t.Run("Infinite loop detection", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program with tight infinite loop
		program := []uint8{
			0x4C, 0x00, 0x80, // JMP $8000 (infinite loop)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Run for many cycles
		initialPC := helper.CPU.PC
		unchangedCycles := 0

		for i := 0; i < 10000; i++ {
			lastPC := helper.CPU.PC
			helper.Bus.Step()

			if helper.CPU.PC == lastPC {
				unchangedCycles++
			} else {
				unchangedCycles = 0
			}

			// Detect if PC hasn't changed for many cycles (infinite loop)
			if unchangedCycles > 100 {
				t.Logf("Detected infinite loop: PC stuck at 0x%04X for %d cycles",
					helper.CPU.PC, unchangedCycles)
				break
			}
		}

		// System should continue running even in infinite loop
		if helper.CPU.PC != initialPC {
			t.Error("PC should remain at loop address")
		}

		t.Log("Infinite loop detection test completed")
	})

	t.Run("Invalid instruction handling", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program with invalid/undefined opcodes
		program := []uint8{
			0x02,             // Illegal opcode (KIL)
			0xEA,             // NOP (should continue)
			0x12,             // Illegal opcode
			0xEA,             // NOP
			0x22,             // Illegal opcode
			0xEA,             // NOP
			0x4C, 0x06, 0x80, // JMP to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute through illegal opcodes
		for i := 0; i < 20; i++ {
			helper.Bus.Step()
		}

		// System should handle illegal opcodes gracefully
		// (typically by treating them as NOPs or 2-cycle instructions)
		if helper.CPU.PC < 0x8000 {
			t.Errorf("PC left ROM area after illegal opcodes: 0x%04X", helper.CPU.PC)
		}

		t.Log("Invalid instruction handling test completed")
	})
}

// TestResourceExhaustion tests system behavior under resource pressure
func TestResourceExhaustion(t *testing.T) {
	t.Run("Memory stress test", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that rapidly accesses all memory regions
		program := []uint8{
			// Stress test all memory areas
			0xA2, 0x00, // LDX #$00

			// Write to all RAM
			0x9D, 0x00, 0x00, // STA $0000,X (zero page)
			0x9D, 0x00, 0x01, // STA $0100,X (stack)
			0x9D, 0x00, 0x02, // STA $0200,X (RAM)
			0x9D, 0x00, 0x03, // STA $0300,X (RAM)
			0x9D, 0x00, 0x04, // STA $0400,X (RAM)
			0x9D, 0x00, 0x05, // STA $0500,X (RAM)
			0x9D, 0x00, 0x06, // STA $0600,X (RAM)
			0x9D, 0x00, 0x07, // STA $0700,X (RAM)

			// Read from all regions
			0xBD, 0x00, 0x20, // LDA $2000,X (PPU)
			0xBD, 0x00, 0x40, // LDA $4000,X (APU)
			0xBD, 0x00, 0x80, // LDA $8000,X (ROM)

			0xE8,       // INX
			0xD0, 0xDF, // BNE loop (256 iterations)

			// Continue stress test
			0x4C, 0x02, 0x80, // JMP to restart
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Run stress test
		for i := 0; i < 10000; i++ {
			helper.Bus.Step()

			// Check system stability periodically
			if i%1000 == 0 {
				if helper.CPU.PC < 0x8000 || helper.CPU.PC > 0xFFFF {
					t.Errorf("PC out of bounds during stress test: 0x%04X", helper.CPU.PC)
					break
				}
			}
		}

		// Verify memory integrity after stress test
		testValue := uint8(0xAB)
		helper.Memory.Write(0x0300, testValue)
		readValue := helper.Memory.Read(0x0300)

		if readValue != testValue {
			t.Errorf("Memory integrity compromised: wrote 0x%02X, read 0x%02X",
				testValue, readValue)
		}

		t.Log("Memory stress test completed")
	})

	t.Run("Rapid PPU access", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that rapidly accesses PPU registers
		program := []uint8{
			// Rapid PPU register access
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK)
			0xAD, 0x02, 0x20, // LDA $2002 (PPUSTATUS)
			0x8D, 0x03, 0x20, // STA $2003 (OAMADDR)
			0x8D, 0x04, 0x20, // STA $2004 (OAMDATA)
			0xAD, 0x02, 0x20, // LDA $2002 (PPUSTATUS)
			0x8D, 0x05, 0x20, // STA $2005 (PPUSCROLL)
			0x8D, 0x05, 0x20, // STA $2005 (PPUSCROLL)
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR)
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR)
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)

			0x4C, 0x02, 0x80, // JMP to repeat
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute rapid PPU access
		for i := 0; i < 5000; i++ {
			helper.Bus.Step()
		}

		// System should remain stable despite rapid PPU access
		if helper.CPU.PC < 0x8000 {
			t.Errorf("System became unstable during rapid PPU access: PC=0x%04X", helper.CPU.PC)
		}

		t.Log("Rapid PPU access test completed")
	})

	t.Run("Excessive DMA transfers", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up DMA source data
		for i := 0; i < 256; i++ {
			helper.Memory.Write(0x0200+uint16(i), uint8(i))
		}

		// Program that triggers excessive DMA
		program := []uint8{
			// Rapid DMA transfers
			0xA9, 0x02, // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (OAM DMA)
			0x8D, 0x14, 0x40, // STA $4014 (another DMA)
			0x8D, 0x14, 0x40, // STA $4014 (another DMA)

			// Continue with more transfers
			0x4C, 0x02, 0x80, // JMP to repeat
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute excessive DMA
		for i := 0; i < 1000; i++ {
			helper.Bus.Step()
		}

		// System should handle multiple DMA requests gracefully
		if helper.CPU.PC < 0x8000 {
			t.Errorf("System failed during excessive DMA: PC=0x%04X", helper.CPU.PC)
		}

		t.Log("Excessive DMA transfers test completed")
	})
}

// TestTimingViolations tests timing-critical edge cases
func TestTimingViolations(t *testing.T) {
	t.Run("PPU register access during rendering", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Program that accesses PPU during rendering
		program := []uint8{
			// Enable rendering
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			// Access PPU registers during rendering (bad practice)
			0xA9, 0x20, // LDA #$20
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR during rendering)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006
			0xA9, 0x55, // LDA #$55
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA during rendering)

			// Continue
			0x4C, 0x0C, 0x80, // JMP to continue violations
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute timing violations
		for i := 0; i < 5000; i++ {
			helper.Bus.Step()
		}

		// System should survive timing violations (may cause visual glitches but not crash)
		if helper.CPU.PC < 0x8000 {
			t.Errorf("System crashed during PPU timing violations: PC=0x%04X", helper.CPU.PC)
		}

		t.Log("PPU register access during rendering test completed")
	})

	t.Run("Interrupt timing edge cases", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up NMI handler
		nmiHandler := []uint8{
			0xE6, 0x50, // INC $50 (NMI counter)
			0x40, // RTI
		}

		// Program that tests interrupt edge cases
		program := []uint8{
			// Enable/disable NMI rapidly
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x00, 0x20, // STA $2000 (disable NMI)
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)

			// Read PPUSTATUS at critical timing
			0xAD, 0x02, 0x20, // LDA $2002 (may suppress NMI)
			0xAD, 0x02, 0x20, // LDA $2002 (again)

			0x4C, 0x02, 0x80, // JMP to repeat
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00 // NMI vector
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00 // Reset vector
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize NMI counter
		helper.Memory.Write(0x0050, 0x00)

		// Execute interrupt timing tests
		for i := 0; i < 10000; i++ {
			helper.Bus.Step()
		}

		// Check NMI counter
		nmiCount := helper.Memory.Read(0x0050)
		t.Logf("NMI count during timing edge case test: %d", nmiCount)

		// System should handle interrupt timing edge cases
		if helper.CPU.PC < 0x8000 {
			t.Errorf("System failed during interrupt timing test: PC=0x%04X", helper.CPU.PC)
		}

		t.Log("Interrupt timing edge cases test completed")
	})

	t.Run("Reset during critical operations", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up test data
		for i := 0; i < 256; i++ {
			helper.Memory.Write(0x0300+uint16(i), uint8(i))
		}

		// Program that does critical operations
		program := []uint8{
			// Start DMA
			0xA9, 0x03, // LDA #$03
			0x8D, 0x14, 0x40, // STA $4014 (start DMA)

			// Critical PPU operations
			0xA9, 0x3F, // LDA #$3F
			0x8D, 0x06, 0x20, // STA $2006
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006

			// Write palette
			0xA9, 0x0F, // LDA #$0F
			0x8D, 0x07, 0x20, // STA $2007

			0xEA,             // NOP
			0x4C, 0x0E, 0x80, // JMP to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute partway through critical operations
		for i := 0; i < 5; i++ {
			helper.Bus.Step()
		}

		// Reset during critical operations
		helper.Bus.Reset()

		// System should recover properly
		if helper.CPU.PC != 0x8000 {
			t.Errorf("Reset did not properly restore PC: got 0x%04X", helper.CPU.PC)
		}
		if helper.CPU.SP != 0xFD {
			t.Errorf("Reset did not properly restore SP: got 0x%02X", helper.CPU.SP)
		}

		// Continue execution after reset
		for i := 0; i < 100; i++ {
			helper.Bus.Step()
		}

		t.Log("Reset during critical operations test completed")
	})
}

// TestSystemRecovery tests system recovery from error conditions
func TestSystemRecovery(t *testing.T) {
	t.Run("Recovery from stack corruption", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that corrupts stack and recovers
		program := []uint8{
			// Save initial state
			0xBA,       // TSX (save SP to X)
			0x86, 0x60, // STX $60 (save original SP)

			// Corrupt stack
			0xA2, 0x80, // LDX #$80
			0x9A, // TXS (corrupt SP)

			// Try to use corrupted stack
			0x48,             // PHA (push to wrong location)
			0x20, 0x20, 0x80, // JSR subroutine (will corrupt return)

			// Subroutine should never return properly
			0xEA,             // NOP
			0x4C, 0x14, 0x80, // JMP to error handler

			// Subroutine at $8020
			0x60, // RTS (return to corrupted stack)
		}

		// Error handler at $8014
		errorHandler := []uint8{
			// Recover stack
			0xA6, 0x60, // LDX $60 (load original SP)
			0x9A, // TXS (restore SP)

			// Mark recovery
			0xA9, 0xAA, // LDA #$AA
			0x85, 0x61, // STA $61 (recovery marker)

			// Continue normally
			0xEA,             // NOP
			0x4C, 0x1A, 0x80, // JMP to normal operation
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0014:], errorHandler)
		romData[0x0020] = 0x60 // RTS at subroutine
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize recovery marker
		helper.Memory.Write(0x0061, 0x00)

		// Execute corruption and recovery
		for i := 0; i < 100; i++ {
			helper.Bus.Step()

			// Check if recovery occurred
			recoveryMarker := helper.Memory.Read(0x0061)
			if recoveryMarker == 0xAA {
				t.Log("Stack corruption recovery successful")
				break
			}
		}

		// Verify recovery
		recoveryMarker := helper.Memory.Read(0x0061)
		if recoveryMarker != 0xAA {
			t.Error("System did not recover from stack corruption")
		}

		t.Log("Stack corruption recovery test completed")
	})

	t.Run("Recovery from timing violations", func(t *testing.T) {
		helper := NewErrorTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that violates timing then recovers
		program := []uint8{
			// Enable rendering
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			// Violate timing (access VRAM during rendering)
			0xA9, 0x20, // LDA #$20
			0x8D, 0x06, 0x20, // STA $2006
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006
			0x8D, 0x07, 0x20, // STA $2007 (bad timing)

			// Recovery: wait for VBlank
			0xAD, 0x02, 0x20, // LDA $2002
			0x10, 0xFB, // BPL -5 (wait for VBlank)

			// Safe VRAM access during VBlank
			0xA9, 0x3F, // LDA #$3F
			0x8D, 0x06, 0x20, // STA $2006
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006
			0xA9, 0x0F, // LDA #$0F
			0x8D, 0x07, 0x20, // STA $2007 (safe timing)

			// Mark recovery
			0xA9, 0xBB, // LDA #$BB
			0x85, 0x70, // STA $70

			0x4C, 0x1E, 0x80, // JMP to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize recovery marker
		helper.Memory.Write(0x0070, 0x00)

		// Execute timing violation and recovery
		for i := 0; i < 50000; i++ {
			helper.Bus.Step()

			// Check if recovery occurred
			recoveryMarker := helper.Memory.Read(0x0070)
			if recoveryMarker == 0xBB {
				t.Logf("Timing violation recovery successful after %d cycles", i)
				break
			}
		}

		// Verify recovery
		recoveryMarker := helper.Memory.Read(0x0070)
		if recoveryMarker != 0xBB {
			t.Error("System did not recover from timing violations")
		}

		t.Log("Timing violation recovery test completed")
	})
}
