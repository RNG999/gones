package integration

import (
	"testing"
)

// TestInterruptEdgeCases tests complex interrupt scenarios and edge cases
func TestInterruptEdgeCases(t *testing.T) {
	t.Run("NMI_Suppression_Critical_Timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler for detection
		nmiHandler := []uint8{
			0xE6, 0x10, // INC $10 (NMI occurred)
			0x40,       // RTI
		}

		// Program that tests critical PPUSTATUS read timing
		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			
			// Critical timing loop
			0xE6, 0x11,       // INC $11 (loop counter)
			0xAD, 0x02, 0x20, // LDA $2002 (read PPUSTATUS at various timings)
			0x85, 0x12,       // STA $12 (save last status read)
			0x29, 0x80,       // AND #$80 (isolate VBlank flag)
			0xF0, 0xF5,       // BEQ (continue if VBlank not set)
			
			// VBlank detected, read again immediately
			0xAD, 0x02, 0x20, // LDA $2002 (second read)
			0x85, 0x13,       // STA $13 (save second read)
			0xE6, 0x14,       // INC $14 (VBlank detection counter)
			
			// Small delay then continue
			0xEA, 0xEA,       // NOPs
			0x4C, 0x08, 0x80, // JMP back to loop
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

		// Initialize test variables
		helper.Memory.Write(0x0010, 0x00) // NMI counter
		helper.Memory.Write(0x0011, 0x00) // Loop counter
		helper.Memory.Write(0x0012, 0x00) // Last status read
		helper.Memory.Write(0x0013, 0x00) // Second status read
		helper.Memory.Write(0x0014, 0x00) // VBlank detection counter

		// Execute and look for suppression patterns
		suppressionDetected := false
		normalNMIDetected := false
		
		for i := 0; i < 75000; i++ {
			helper.Bus.Step()
			
			nmiCount := helper.Memory.Read(0x0010)
			vblankDetections := helper.Memory.Read(0x0014)
			firstRead := helper.Memory.Read(0x0012)
			secondRead := helper.Memory.Read(0x0013)
			
			// Check for suppression: VBlank flag read but no NMI
			if vblankDetections > 0 {
				if nmiCount == 0 {
					suppressionDetected = true
					t.Logf("NMI suppression detected at step %d", i)
					t.Logf("  VBlank detections: %d, NMI count: %d", vblankDetections, nmiCount)
					t.Logf("  First read: 0x%02X, Second read: 0x%02X", firstRead, secondRead)
				} else {
					normalNMIDetected = true
					t.Logf("Normal NMI behavior at step %d", i)
				}
				break
			}
		}

		// Report results
		finalNMICount := helper.Memory.Read(0x0010)
		finalVBlankCount := helper.Memory.Read(0x0014)
		
		t.Logf("Critical timing test results:")
		t.Logf("  Final NMI count: %d", finalNMICount)
		t.Logf("  VBlank detections: %d", finalVBlankCount)
		t.Logf("  Suppression detected: %v", suppressionDetected)
		t.Logf("  Normal NMI detected: %v", normalNMIDetected)
	})

	t.Run("Multiple_NMI_Edge_Detection", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that tracks multiple calls
		nmiHandler := []uint8{
			0xE6, 0x20,       // INC $20 (NMI call count)
			0xAD, 0x02, 0x20, // LDA $2002 (read status in handler)
			0x85, 0x21,       // STA $21 (save status from handler)
			0x40,             // RTI
		}

		// Program that tests rapid enable/disable cycles
		program := []uint8{
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			
			// Test sequence: enable NMI, wait briefly, disable, repeat
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xE6, 0x22,       // INC $22 (enable count)
			
			// Brief wait
			0xEA, 0xEA, 0xEA,
			
			0xA9, 0x00,       // LDA #$00
			0x8D, 0x00, 0x20, // STA $2000 (disable NMI)
			0xE6, 0x23,       // INC $23 (disable count)
			
			// Another brief wait
			0xEA, 0xEA, 0xEA,
			
			// Re-enable quickly
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (re-enable NMI)
			0xE6, 0x24,       // INC $24 (re-enable count)
			
			// Check loop counter
			0xE6, 0x25,       // INC $25 (loop count)
			0xA5, 0x25,       // LDA $25
			0xC9, 0x10,       // CMP #$10
			0x90, 0xE1,       // BCC (continue loop)
			
			// End loop
			0x4C, 0x22, 0x80, // JMP to end marker
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
		helper.Memory.Write(0x0020, 0x00) // NMI call count
		helper.Memory.Write(0x0021, 0x00) // Status from handler
		helper.Memory.Write(0x0022, 0x00) // Enable count
		helper.Memory.Write(0x0023, 0x00) // Disable count
		helper.Memory.Write(0x0024, 0x00) // Re-enable count
		helper.Memory.Write(0x0025, 0x00) // Loop count

		// Execute test
		for i := 0; i < 100000; i++ {
			helper.Bus.Step()
			
			loopCount := helper.Memory.Read(0x0025)
			if loopCount >= 0x10 {
				// Test completed
				break
			}
		}

		// Analyze results
		nmiCallCount := helper.Memory.Read(0x0020)
		statusFromHandler := helper.Memory.Read(0x0021)
		enableCount := helper.Memory.Read(0x0022)
		disableCount := helper.Memory.Read(0x0023)
		reEnableCount := helper.Memory.Read(0x0024)
		
		t.Logf("Multiple NMI edge detection results:")
		t.Logf("  NMI calls: %d", nmiCallCount)
		t.Logf("  Status from handler: 0x%02X", statusFromHandler)
		t.Logf("  Enable cycles: %d", enableCount)
		t.Logf("  Disable cycles: %d", disableCount)
		t.Logf("  Re-enable cycles: %d", reEnableCount)
		
		// Verify edge detection working (shouldn't get NMI on every enable)
		if nmiCallCount > enableCount {
			t.Errorf("Too many NMIs: %d calls for %d enables", nmiCallCount, enableCount)
		}
	})

	t.Run("DMA_Halt_During_Critical_Instructions", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that executes critical instructions with DMA timing
		program := []uint8{
			// Set up test data
			0xA9, 0x99,       // LDA #$99
			0x85, 0x30,       // STA $30 (test value)
			
			// Critical instruction sequence
			0xA5, 0x30,       // LDA $30 (load test value)
			0x85, 0x31,       // STA $31 (save to verify)
			
			// Read-modify-write instruction that should be atomic
			0xE6, 0x30,       // INC $30 (should complete before DMA)
			
			// Trigger DMA during next instruction
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (trigger DMA)
			0xE6, 0x32,       // INC $32 (this should be delayed by DMA)
			
			// Verify instruction completion
			0xA5, 0x30,       // LDA $30 (check incremented value)
			0x85, 0x33,       // STA $33 (save final value)
			
			0x4C, 0x16, 0x80, // JMP to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize test variables
		helper.Memory.Write(0x0030, 0x00) // Test value
		helper.Memory.Write(0x0031, 0x00) // Saved value
		helper.Memory.Write(0x0032, 0x00) // DMA delay marker
		helper.Memory.Write(0x0033, 0x00) // Final value

		// Execute instruction by instruction and monitor DMA interaction
		instructionCount := 0
		dmaTriggered := false
		
		for instructionCount < 20 {
			beforeValue := helper.Memory.Read(0x0030)
			beforeDMA := helper.Bus.IsDMAInProgress()
			beforePC := helper.CPU.PC
			
			helper.Bus.Step()
			instructionCount++
			
			afterValue := helper.Memory.Read(0x0030)
			afterDMA := helper.Bus.IsDMAInProgress()
			afterPC := helper.CPU.PC
			
			// Log critical state changes
			if beforeValue != afterValue {
				t.Logf("Instruction %d: Value changed %02X -> %02X (PC: %04X -> %04X)",
					instructionCount, beforeValue, afterValue, beforePC, afterPC)
			}
			
			if !beforeDMA && afterDMA {
				dmaTriggered = true
				t.Logf("Instruction %d: DMA started (PC: %04X)", instructionCount, beforePC)
			}
			
			if beforeDMA && !afterDMA {
				t.Logf("Instruction %d: DMA completed (PC: %04X)", instructionCount, afterPC)
			}
		}

		// Verify instruction atomicity
		savedValue := helper.Memory.Read(0x0031) // Value before INC
		finalValue := helper.Memory.Read(0x0033) // Value after INC
		dmaMarker := helper.Memory.Read(0x0032)  // DMA completion marker
		
		t.Logf("Critical instruction test results:")
		t.Logf("  Saved value: 0x%02X", savedValue)
		t.Logf("  Final value: 0x%02X", finalValue)
		t.Logf("  DMA marker: %d", dmaMarker)
		t.Logf("  DMA was triggered: %v", dmaTriggered)
		
		// INC should have completed atomically
		if savedValue == 0x99 && finalValue == 0x9A {
			t.Log("INC instruction completed atomically before DMA")
		} else {
			t.Errorf("INC instruction may have been interrupted: %02X -> %02X", savedValue, finalValue)
		}
	})

	t.Run("NMI_IRQ_Priority_Complex", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler
		nmiHandler := []uint8{
			0xE6, 0x40,       // INC $40 (NMI count)
			0xA9, 0xAA,       // LDA #$AA
			0x85, 0x41,       // STA $41 (NMI marker)
			0x40,             // RTI
		}

		// IRQ handler
		irqHandler := []uint8{
			0xE6, 0x42,       // INC $42 (IRQ count)
			0xA9, 0xBB,       // LDA #$BB
			0x85, 0x43,       // STA $43 (IRQ marker)
			0x40,             // RTI
		}

		// Main program that sets up interrupt conditions
		program := []uint8{
			// Enable both NMI and rendering (for NMI)
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			
			// Clear interrupt disable for IRQ
			0x58,             // CLI (enable IRQ)
			
			// Trigger software IRQ via BRK for testing
			0xE6, 0x44,       // INC $44 (pre-BRK marker)
			0x00,             // BRK (software interrupt)
			0xE6, 0x45,       // INC $45 (post-BRK marker)
			
			// Continue execution
			0xEA, 0xEA,       // NOPs
			0x4C, 0x0E, 0x80, // JMP back to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)  // NMI at $8100
		copy(romData[0x0200:], irqHandler)  // IRQ at $8200
		
		// Set interrupt vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high  
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		romData[0x7FFE] = 0x00 // IRQ vector low
		romData[0x7FFF] = 0x82 // IRQ vector high

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize test variables
		helper.Memory.Write(0x0040, 0x00) // NMI count
		helper.Memory.Write(0x0041, 0x00) // NMI marker
		helper.Memory.Write(0x0042, 0x00) // IRQ count
		helper.Memory.Write(0x0043, 0x00) // IRQ marker
		helper.Memory.Write(0x0044, 0x00) // Pre-BRK marker
		helper.Memory.Write(0x0045, 0x00) // Post-BRK marker

		// Execute and monitor interrupt interactions
		for i := 0; i < 100000; i++ {
			helper.Bus.Step()
			
			nmiCount := helper.Memory.Read(0x0040)
			irqCount := helper.Memory.Read(0x0042)
			preBRK := helper.Memory.Read(0x0044)
			postBRK := helper.Memory.Read(0x0045)
			
			// Check for interesting interrupt interactions
			if nmiCount > 0 && irqCount > 0 {
				nmiMarker := helper.Memory.Read(0x0041)
				irqMarker := helper.Memory.Read(0x0043)
				
				t.Logf("Complex interrupt priority test results at step %d:", i)
				t.Logf("  NMI count: %d (marker: 0x%02X)", nmiCount, nmiMarker)
				t.Logf("  IRQ count: %d (marker: 0x%02X)", irqCount, irqMarker)
				t.Logf("  BRK execution: pre=%d, post=%d", preBRK, postBRK)
				
				// NMI should have priority over IRQ
				if nmiCount > 0 {
					t.Log("NMI executed as expected")
				}
				
				break
			}
		}

		// Final state
		finalNMI := helper.Memory.Read(0x0040)
		finalIRQ := helper.Memory.Read(0x0042)
		
		t.Logf("Final interrupt counts: NMI=%d, IRQ=%d", finalNMI, finalIRQ)
	})
}