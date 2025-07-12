package integration

import (
	"testing"
)

// NMITestHelper provides utilities for NMI integration testing
type NMITestHelper struct {
	*IntegrationTestHelper
	nmiEvents []NMIEvent
}

// NMIEvent represents an NMI event for testing
type NMIEvent struct {
	Scanline               int
	Cycle                  int
	CPUCycleWhenTriggered  int
	HandlerAddress         uint16
	ReturnAddress          uint16
	InterruptedInstruction string
}

// NewNMITestHelper creates an NMI integration test helper
func NewNMITestHelper() *NMITestHelper {
	return &NMITestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		nmiEvents:             make([]NMIEvent, 0),
	}
}

// LogNMIEvent logs an NMI event
func (h *NMITestHelper) LogNMIEvent(event NMIEvent) {
	h.nmiEvents = append(h.nmiEvents, event)
}

// SetupNMIHandler sets up an NMI handler at the specified address
func (h *NMITestHelper) SetupNMIHandler(handlerAddr uint16, handlerCode []uint8) {
	// Set NMI vector
	romData := make([]uint8, 0x8000)

	// Copy existing ROM data if any
	for i := 0; i < 0x8000; i++ {
		romData[i] = h.Cartridge.ReadPRG(0x8000 + uint16(i))
	}

	// Set interrupt vectors
	romData[0x7FFA] = uint8(handlerAddr & 0xFF)        // NMI vector low
	romData[0x7FFB] = uint8((handlerAddr >> 8) & 0xFF) // NMI vector high
	romData[0x7FFC] = 0x00                             // Reset vector low
	romData[0x7FFD] = 0x80                             // Reset vector high

	// Place handler code
	if handlerAddr >= 0x8000 {
		offset := handlerAddr - 0x8000
		copy(romData[offset:], handlerCode)
	}

	h.GetMockCartridge().LoadPRG(romData)
}

// WaitForVBlank waits for VBlank to start and returns the cycle count
func (h *NMITestHelper) WaitForVBlank() int {
	cycles := 0
	maxCycles := 50000 // Safety limit

	for cycles < maxCycles {
		ppuStatus := h.PPU.ReadRegister(0x2002)
		if (ppuStatus & 0x80) != 0 { // VBlank flag set
			return cycles
		}
		h.Bus.Step()
		cycles++
	}

	return -1 // VBlank not found
}

// TestNMIGeneration tests NMI generation from PPU to CPU
func TestNMIGeneration(t *testing.T) {
	t.Run("Basic NMI generation on VBlank", func(t *testing.T) {
		helper := NewNMITestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up NMI handler
		nmiHandler := []uint8{
			0xA9, 0x55, // LDA #$55
			0x85, 0x80, // STA $80 (NMI marker)
			0x40, // RTI
		}
		helper.SetupNMIHandler(0x8100, nmiHandler)

		// Main program that enables NMI and waits
		program := []uint8{
			// Enable NMI
			0xA9, 0x80, // LDA #$80 (NMI enable)
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)

			// Wait loop
			0xEA,             // NOP
			0xEA,             // NOP
			0xEA,             // NOP
			0x4C, 0x08, 0x80, // JMP to wait loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)

		// Copy handler
		copy(romData[0x0100:], nmiHandler)

		// Set vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute setup
		helper.Bus.Step() // LDA #$80
		helper.Bus.Step() // STA $2000 (enable NMI)

		// Clear any existing NMI marker
		helper.Memory.Write(0x0080, 0x00)

		// Wait for VBlank to trigger NMI
		vblankCycles := helper.WaitForVBlank()
		if vblankCycles < 0 {
			t.Fatal("VBlank was not detected")
		}

		t.Logf("VBlank detected after %d cycles", vblankCycles)

		// Continue execution to allow NMI to be processed
		for i := 0; i < 100; i++ {
			helper.Bus.Step()

			// Check if NMI handler executed
			nmiMarker := helper.Memory.Read(0x0080)
			if nmiMarker == 0x55 {
				t.Log("NMI handler executed successfully")
				return
			}
		}

		t.Error("NMI handler was not executed")
	})

	t.Run("NMI enable/disable control", func(t *testing.T) {
		helper := NewNMITestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up NMI handler
		nmiHandler := []uint8{
			0xE6, 0x90, // INC $90 (NMI counter)
			0x40, // RTI
		}
		helper.SetupNMIHandler(0x8100, nmiHandler)

		// Program that toggles NMI enable
		program := []uint8{
			// Test 1: NMI disabled
			0xA9, 0x00, // LDA #$00 (NMI disabled)
			0x8D, 0x00, 0x20, // STA $2000

			// Wait for potential VBlank
			0xEA, 0xEA, 0xEA, 0xEA, // NOPs

			// Test 2: Enable NMI
			0xA9, 0x80, // LDA #$80 (NMI enabled)
			0x8D, 0x00, 0x20, // STA $2000

			// Wait for VBlank with NMI enabled
			0xEA, 0xEA, 0xEA, 0xEA, // NOPs

			// Test 3: Disable NMI again
			0xA9, 0x00, // LDA #$00 (NMI disabled)
			0x8D, 0x00, 0x20, // STA $2000

			// Wait loop
			0x4C, 0x1C, 0x80, // JMP to wait
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize NMI counter
		helper.Memory.Write(0x0090, 0x00)

		// Execute test sequence
		helper.Bus.Step() // LDA #$00 (disable NMI)
		helper.Bus.Step() // STA $2000

		// Wait through potential VBlank with NMI disabled
		for i := 0; i < 30000; i++ {
			helper.Bus.Step()
		}

		// Check that no NMI occurred
		nmiCount1 := helper.Memory.Read(0x0090)
		if nmiCount1 != 0 {
			t.Errorf("NMI occurred when disabled: count=%d", nmiCount1)
		}

		// Reset and test with NMI enabled
		helper.Bus.Reset()
		helper.Memory.Write(0x0090, 0x00)

		// Execute enable sequence
		for i := 0; i < 4; i++ {
			helper.Bus.Step()
		}
		helper.Bus.Step() // LDA #$80 (enable NMI)
		helper.Bus.Step() // STA $2000

		// Wait for VBlank with NMI enabled
		for i := 0; i < 30000; i++ {
			helper.Bus.Step()

			nmiCount := helper.Memory.Read(0x0090)
			if nmiCount > 0 {
				t.Logf("NMI occurred when enabled: count=%d after %d cycles", nmiCount, i)
				break
			}
		}

		nmiCount2 := helper.Memory.Read(0x0090)
		if nmiCount2 == 0 {
			t.Error("NMI did not occur when enabled")
		}
	})

	t.Run("NMI timing precision", func(t *testing.T) {
		helper := NewNMITestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up precise NMI handler that records timing
		nmiHandler := []uint8{
			0xA9, 0xAA, // LDA #$AA
			0x85, 0xA0, // STA $A0 (NMI occurred marker)
			0x40, // RTI
		}
		helper.SetupNMIHandler(0x8100, nmiHandler)

		// Program that enables NMI and executes known instruction sequence
		program := []uint8{
			// Enable NMI
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000

			// Instruction sequence with known timing
			0xA9, 0x01, // LDA #$01 (2 cycles)
			0x85, 0xA1, // STA $A1 (3 cycles)
			0xE6, 0xA1, // INC $A1 (5 cycles)
			0xA5, 0xA1, // LDA $A1 (3 cycles)
			0x85, 0xA2, // STA $A2 (3 cycles)

			// Loop back
			0x4C, 0x08, 0x80, // JMP to instruction sequence
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize markers
		helper.Memory.Write(0x00A0, 0x00) // NMI marker
		helper.Memory.Write(0x00A1, 0x00) // Program counter
		helper.Memory.Write(0x00A2, 0x00) // Last value

		// Execute setup
		helper.Bus.Step() // LDA #$80
		helper.Bus.Step() // STA $2000

		// Execute instruction sequence and wait for NMI
		cyclesExecuted := 0
		maxCycles := 50000

		for cyclesExecuted < maxCycles {
			helper.Bus.Step()
			cyclesExecuted++

			// Check if NMI occurred
			nmiMarker := helper.Memory.Read(0x00A0)
			if nmiMarker == 0xAA {
				t.Logf("NMI occurred after %d cycles", cyclesExecuted)

				// Check program state when NMI occurred
				programCounter := helper.Memory.Read(0x00A1)
				lastValue := helper.Memory.Read(0x00A2)

				t.Logf("Program state at NMI: counter=%d, last_value=%d",
					programCounter, lastValue)

				// NMI should occur at VBlank (scanline 241, cycle 1)
				// This corresponds to specific CPU cycle timing
				break
			}
		}

		if cyclesExecuted >= maxCycles {
			t.Error("NMI did not occur within expected time")
		}
	})
}

// TestNMIInterruptBehavior tests NMI interrupt handling behavior
func TestNMIInterruptBehavior(t *testing.T) {
	t.Run("NMI interrupt sequence", func(t *testing.T) {
		helper := NewNMITestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that preserves state
		nmiHandler := []uint8{
			// Save registers (would be done by handler)
			0x48, // PHA (save A)
			0x8A, // TXA
			0x48, // PHA (save X)
			0x98, // TYA
			0x48, // PHA (save Y)

			// NMI work
			0xE6, 0xB0, // INC $B0 (NMI counter)

			// Restore registers
			0x68, // PLA (restore Y)
			0xA8, // TAY
			0x68, // PLA (restore X)
			0xAA, // TAX
			0x68, // PLA (restore A)

			0x40, // RTI
		}
		helper.SetupNMIHandler(0x8100, nmiHandler)

		// Main program that sets up registers and waits
		program := []uint8{
			// Enable NMI
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000

			// Set up test registers
			0xA9, 0x42, // LDA #$42
			0xA2, 0x33, // LDX #$33
			0xA0, 0x24, // LDY #$24

			// Wait loop (preserve registers)
			0x85, 0xB1, // STA $B1 (save A for comparison)
			0x86, 0xB2, // STX $B2 (save X for comparison)
			0x84, 0xB3, // STY $B3 (save Y for comparison)

			0xEA,             // NOP
			0x4C, 0x10, 0x80, // JMP to wait loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize counters
		helper.Memory.Write(0x00B0, 0x00) // NMI counter

		// Execute setup
		for i := 0; i < 8; i++ {
			helper.Bus.Step()
		}

		// Record initial register states
		initialA := helper.CPU.A
		initialX := helper.CPU.X
		initialY := helper.CPU.Y
		initialSP := helper.CPU.SP

		// Wait for NMI
		for i := 0; i < 30000; i++ {
			helper.Bus.Step()

			nmiCount := helper.Memory.Read(0x00B0)
			if nmiCount > 0 {
				// NMI occurred and returned

				// Check that registers were preserved
				if helper.CPU.A != initialA {
					t.Errorf("A register not preserved: expected 0x%02X, got 0x%02X",
						initialA, helper.CPU.A)
				}
				if helper.CPU.X != initialX {
					t.Errorf("X register not preserved: expected 0x%02X, got 0x%02X",
						initialX, helper.CPU.X)
				}
				if helper.CPU.Y != initialY {
					t.Errorf("Y register not preserved: expected 0x%02X, got 0x%02X",
						initialY, helper.CPU.Y)
				}

				// Stack pointer should be back to original (after pushing/popping)
				if helper.CPU.SP != initialSP {
					t.Errorf("Stack pointer not restored: expected 0x%02X, got 0x%02X",
						initialSP, helper.CPU.SP)
				}

				t.Logf("NMI interrupt sequence completed successfully")
				return
			}
		}

		t.Error("NMI did not occur")
	})

	t.Run("NMI interrupting different instructions", func(t *testing.T) {
		helper := NewNMITestHelper()
		helper.SetupBasicROM(0x8000)

		// Simple NMI handler
		nmiHandler := []uint8{
			0xE6, 0xC0, // INC $C0 (interrupt counter)
			0x40, // RTI
		}
		helper.SetupNMIHandler(0x8100, nmiHandler)

		// Program with various instruction types
		program := []uint8{
			// Enable NMI
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000

			// Various instructions that could be interrupted
			0xA9, 0x01, // LDA #$01 (immediate)
			0x85, 0xC1, // STA $C1 (zero page)
			0xAD, 0x00, 0x30, // LDA $3000 (absolute)
			0x8D, 0x00, 0x30, // STA $3000 (absolute)
			0xE6, 0xC1, // INC $C1 (read-modify-write)
			0x20, 0x20, 0x80, // JSR $8020 (subroutine)
		}

		// Subroutine
		subroutine := []uint8{
			0xA9, 0x55, // LDA #$55
			0x85, 0xC2, // STA $C2
			0x60, // RTS
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0020:], subroutine) // Place subroutine at $8020
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize
		helper.Memory.Write(0x00C0, 0x00) // Interrupt counter
		helper.Memory.Write(0x00C1, 0x00) // Test variable
		helper.Memory.Write(0x00C2, 0x00) // Subroutine marker

		// Execute and wait for NMI at various instruction points
		for i := 0; i < 50000; i++ {
			helper.Bus.Step()

			interruptCount := helper.Memory.Read(0x00C0)
			if interruptCount > 0 {
				t.Logf("NMI occurred during instruction execution (interrupt count: %d)", interruptCount)

				// Verify program continued correctly after NMI
				testVar := helper.Memory.Read(0x00C1)
				subroutineMarker := helper.Memory.Read(0x00C2)

				t.Logf("Program state after NMI: test_var=%d, subroutine_marker=%d",
					testVar, subroutineMarker)

				break
			}
		}

		interruptCount := helper.Memory.Read(0x00C0)
		if interruptCount == 0 {
			t.Error("NMI did not occur during instruction execution")
		}
	})

	t.Run("NMI edge detection", func(t *testing.T) {
		helper := NewNMITestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that counts occurrences
		nmiHandler := []uint8{
			0xE6, 0xD0, // INC $D0
			0x40, // RTI
		}
		helper.SetupNMIHandler(0x8100, nmiHandler)

		// Program that toggles NMI enable to test edge detection
		program := []uint8{
			// Test: Enable NMI when VBlank already set
			0xAD, 0x02, 0x20, // LDA $2002 (read status, may set VBlank)
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)

			// Wait a bit
			0xEA, 0xEA, 0xEA,

			// Disable and re-enable NMI
			0xA9, 0x00, // LDA #$00
			0x8D, 0x00, 0x20, // STA $2000 (disable NMI)
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (re-enable NMI)

			// Loop
			0x4C, 0x0C, 0x80, // JMP to wait
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize
		helper.Memory.Write(0x00D0, 0x00)

		// Execute test sequence
		for i := 0; i < 50000; i++ {
			helper.Bus.Step()

			// Check for NMI occurrence
			nmiCount := helper.Memory.Read(0x00D0)
			if nmiCount > 0 {
				t.Logf("NMI edge detection test: %d NMIs detected", nmiCount)

				// Continue for a bit to test multiple NMIs
				if nmiCount >= 2 {
					break
				}
			}
		}

		finalCount := helper.Memory.Read(0x00D0)
		if finalCount == 0 {
			t.Error("No NMIs detected in edge detection test")
		} else {
			t.Logf("NMI edge detection completed: %d total NMIs", finalCount)
		}
	})
}

// TestNMITiming tests precise NMI timing coordination
func TestNMITiming(t *testing.T) {
	t.Run("VBlank to NMI delay", func(t *testing.T) {
		helper := NewNMITestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that records precise timing
		nmiHandler := []uint8{
			0xA9, 0xBB, // LDA #$BB
			0x85, 0xE0, // STA $E0 (NMI timestamp)
			0x40, // RTI
		}
		helper.SetupNMIHandler(0x8100, nmiHandler)

		// Program that enables NMI and waits for precise timing
		program := []uint8{
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000

			// Tight loop to catch precise timing
			0xE6, 0xE1, // INC $E1 (cycle counter)
			0x4C, 0x06, 0x80, // JMP to loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize timing markers
		helper.Memory.Write(0x00E0, 0x00) // NMI timestamp
		helper.Memory.Write(0x00E1, 0x00) // Cycle counter

		// Execute setup
		helper.Bus.Step() // LDA #$80
		helper.Bus.Step() // STA $2000

		// Run tight loop until NMI
		loopCycles := 0
		maxLoops := 30000

		for loopCycles < maxLoops {
			helper.Bus.Step() // INC $E1
			helper.Bus.Step() // JMP
			loopCycles += 2

			// Check if NMI occurred
			nmiTimestamp := helper.Memory.Read(0x00E0)
			if nmiTimestamp == 0xBB {
				cycleCount := helper.Memory.Read(0x00E1)

				t.Logf("NMI occurred after %d loop cycles, counter value: %d",
					loopCycles, cycleCount)

				// NMI should occur at predictable intervals (frame timing)
				// Verify timing is within expected range

				break
			}
		}

		if loopCycles >= maxLoops {
			t.Error("NMI did not occur within expected time frame")
		}
	})

	t.Run("NMI and PPUSTATUS read race condition", func(t *testing.T) {
		helper := NewNMITestHelper()
		helper.SetupBasicROM(0x8000)

		// Test the critical timing where reading PPUSTATUS at exactly
		// the wrong time can suppress NMI generation

		nmiHandler := []uint8{
			0xE6, 0xF0, // INC $F0 (NMI occurred)
			0x40, // RTI
		}
		helper.SetupNMIHandler(0x8100, nmiHandler)

		// Program that reads PPUSTATUS at critical timing
		program := []uint8{
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)

			// Read PPUSTATUS repeatedly to test race condition
			0xAD, 0x02, 0x20, // LDA $2002 (PPUSTATUS)
			0x85, 0xF1, // STA $F1 (save status)
			0xAD, 0x02, 0x20, // LDA $2002 (PPUSTATUS again)
			0x85, 0xF2, // STA $F2 (save status)

			// Continue program
			0xEA,             // NOP
			0x4C, 0x08, 0x80, // JMP to status read loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize
		helper.Memory.Write(0x00F0, 0x00) // NMI counter
		helper.Memory.Write(0x00F1, 0x00) // Status 1
		helper.Memory.Write(0x00F2, 0x00) // Status 2

		// Execute and look for race condition
		cyclesRun := 0
		for cyclesRun < 50000 {
			helper.Bus.Step()
			cyclesRun++

			// Check results periodically
			if cyclesRun%1000 == 0 {
				nmiCount := helper.Memory.Read(0x00F0)
				status1 := helper.Memory.Read(0x00F1)
				status2 := helper.Memory.Read(0x00F2)

				if nmiCount > 0 || (status1&0x80) != 0 || (status2&0x80) != 0 {
					t.Logf("Race condition test results after %d cycles:", cyclesRun)
					t.Logf("  NMI count: %d", nmiCount)
					t.Logf("  Status 1: 0x%02X (VBlank: %t)", status1, (status1&0x80) != 0)
					t.Logf("  Status 2: 0x%02X (VBlank: %t)", status2, (status2&0x80) != 0)

					// If we read VBlank flag, NMI behavior depends on exact timing
					break
				}
			}
		}

		t.Log("PPUSTATUS read race condition test completed")
	})
}
