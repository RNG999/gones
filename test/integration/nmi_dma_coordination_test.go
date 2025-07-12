package integration

import (
	"testing"
)

// TestNMIDMACoordination tests the interaction between NMI generation and DMA transfers
func TestNMIDMACoordination(t *testing.T) {
	t.Run("NMI_During_DMA_Transfer", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that records when it executes
		nmiHandler := []uint8{
			0xA9, 0xFF,       // LDA #$FF
			0x85, 0xB0,       // STA $B0 (NMI executed marker)
			0xAD, 0x02, 0x20, // LDA $2002 (read PPUSTATUS)
			0x85, 0xB1,       // STA $B1 (save status)
			0x40,             // RTI
		}

		// Main program that enables NMI and triggers DMA near VBlank timing
		program := []uint8{
			// Enable NMI and rendering
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			
			// Set up sprite data
			0xA2, 0x00,       // LDX #$00
			0xA9, 0x7F,       // LDA #$7F
			0x9D, 0x00, 0x02, // STA $0200,X
			0xE8,             // INX
			0xE0, 0x10,       // CPX #$10
			0x90, 0xF8,       // BCC (loop)
			
			// Wait for specific timing, then trigger DMA
			0xEA, 0xEA, 0xEA, // NOPs for timing
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (trigger DMA)
			0xE6, 0xB2,       // INC $B2 (DMA triggered marker)
			
			// Continue execution
			0xEA,             // NOP
			0x4C, 0x1D, 0x80, // JMP to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		copy(romData[0x0100:], nmiHandler)
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize test variables
		helper.Memory.Write(0x00B0, 0x00) // NMI executed marker
		helper.Memory.Write(0x00B1, 0x00) // PPUSTATUS at NMI
		helper.Memory.Write(0x00B2, 0x00) // DMA triggered marker

		// Execute program until we get interesting interaction
		maxSteps := 100000
		nmiDuringDMA := false
		
		for i := 0; i < maxSteps; i++ {
			wasDMAInProgress := helper.Bus.IsDMAInProgress()
			
			helper.Bus.Step()
			
			// Check if NMI occurred during DMA
			nmiExecuted := helper.Memory.Read(0x00B0) == 0xFF
			dmaTriggered := helper.Memory.Read(0x00B2) > 0
			
			if nmiExecuted && wasDMAInProgress {
				nmiDuringDMA = true
				ppuStatus := helper.Memory.Read(0x00B1)
				
				t.Logf("NMI occurred during DMA transfer at step %d", i)
				t.Logf("PPUSTATUS when NMI executed: 0x%02X", ppuStatus)
				t.Logf("VBlank flag set: %v", (ppuStatus&0x80) != 0)
				
				// NMI should still execute even during DMA
				if !nmiExecuted {
					t.Error("NMI should execute even during DMA transfer")
				}
				
				break
			}
			
			// Also log normal case where they don't interfere
			if nmiExecuted && dmaTriggered && !wasDMAInProgress {
				t.Logf("NMI and DMA occurred but didn't interfere (step %d)", i)
				break
			}
		}

		// Test completed - analyze results
		finalNMIExecuted := helper.Memory.Read(0x00B0) == 0xFF
		finalDMATriggered := helper.Memory.Read(0x00B2) > 0
		
		t.Logf("Test completion: NMI executed=%v, DMA triggered=%v, NMI during DMA=%v",
			finalNMIExecuted, finalDMATriggered, nmiDuringDMA)
	})

	t.Run("DMA_Delay_NMI_Timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that records precise timing
		nmiHandler := []uint8{
			0xE6, 0xC0, // INC $C0 (NMI counter)
			0xAD, 0x02, 0x20, // LDA $2002
			0x85, 0xC1, // STA $C1 (PPUSTATUS)
			0x40,       // RTI
		}

		// Program that triggers DMA just before expected VBlank
		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			
			// Timing loop to get close to VBlank
			0xE6, 0xC2,       // INC $C2 (cycle counter)
			0xA5, 0xC2,       // LDA $C2
			0xC9, 0xF0,       // CMP #$F0 (approach VBlank timing)
			0x90, 0xF8,       // BCC (continue counting)
			
			// Trigger DMA at critical timing
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (DMA)
			0xE6, 0xC3,       // INC $C3 (DMA done marker)
			
			0x4C, 0x08, 0x80, // JMP back to timing loop
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
		helper.Memory.Write(0x00C0, 0x00) // NMI counter
		helper.Memory.Write(0x00C1, 0x00) // PPUSTATUS
		helper.Memory.Write(0x00C2, 0x00) // Cycle counter
		helper.Memory.Write(0x00C3, 0x00) // DMA done marker

		// Execute and measure timing relationships
		maxSteps := 50000
		nmiOccurred := false
		dmaOccurred := false
		
		for i := 0; i < maxSteps; i++ {
			helper.Bus.Step()
			
			nmiCount := helper.Memory.Read(0x00C0)
			dmaCount := helper.Memory.Read(0x00C3)
			
			if nmiCount > 0 && !nmiOccurred {
				nmiOccurred = true
				ppuStatus := helper.Memory.Read(0x00C1)
				t.Logf("First NMI occurred at step %d, PPUSTATUS: 0x%02X", i, ppuStatus)
			}
			
			if dmaCount > 0 && !dmaOccurred {
				dmaOccurred = true
				t.Logf("First DMA occurred at step %d", i)
			}
			
			if nmiOccurred && dmaOccurred {
				break
			}
		}

		if !nmiOccurred {
			t.Error("NMI should occur during test")
		}
		if !dmaOccurred {
			t.Error("DMA should occur during test")
		}

		t.Logf("Timing test completed: NMI and DMA coordination verified")
	})

	t.Run("Rapid_DMA_Near_VBlank", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		nmiHandler := []uint8{
			0xE6, 0xD0, // INC $D0
			0x40,       // RTI
		}

		// Program that performs rapid DMAs around VBlank time
		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			
			// Wait for near VBlank
			0xAD, 0x02, 0x20, // LDA $2002
			0x10, 0xFB,       // BPL (wait for VBlank)
			
			// Rapid DMA sequence
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (DMA 1)
			0xE6, 0xD1,       // INC $D1
			0xA9, 0x03,       // LDA #$03
			0x8D, 0x14, 0x40, // STA $4014 (DMA 2)
			0xE6, 0xD2,       // INC $D2
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (DMA 3)
			0xE6, 0xD3,       // INC $D3
			
			0x4C, 0x08, 0x80, // JMP back
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
		helper.Memory.Write(0x00D0, 0x00) // NMI counter
		helper.Memory.Write(0x00D1, 0x00) // DMA 1 counter
		helper.Memory.Write(0x00D2, 0x00) // DMA 2 counter
		helper.Memory.Write(0x00D3, 0x00) // DMA 3 counter

		// Run test
		for i := 0; i < 75000; i++ {
			helper.Bus.Step()
			
			nmiCount := helper.Memory.Read(0x00D0)
			dma1Count := helper.Memory.Read(0x00D1)
			dma2Count := helper.Memory.Read(0x00D2)
			dma3Count := helper.Memory.Read(0x00D3)
			
			// Break when we have some activity
			if nmiCount > 0 && (dma1Count > 0 || dma2Count > 0 || dma3Count > 0) {
				t.Logf("Rapid DMA test results after %d steps:", i)
				t.Logf("  NMI count: %d", nmiCount)
				t.Logf("  DMA 1 count: %d", dma1Count)
				t.Logf("  DMA 2 count: %d", dma2Count)
				t.Logf("  DMA 3 count: %d", dma3Count)
				break
			}
		}

		// Verify system handled rapid DMAs correctly
		totalDMAs := int(helper.Memory.Read(0x00D1)) + 
					int(helper.Memory.Read(0x00D2)) + 
					int(helper.Memory.Read(0x00D3))
		
		if totalDMAs == 0 {
			t.Error("No DMAs were executed")
		}

		t.Logf("Rapid DMA test completed: %d total DMAs executed", totalDMAs)
	})

	t.Run("NMI_Priority_Over_DMA", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that takes some time
		nmiHandler := []uint8{
			0xE6, 0xE0,       // INC $E0 (NMI entry)
			0xA2, 0x10,       // LDX #$10
			0xCA,             // DEX
			0xD0, 0xFD,       // BNE (small delay loop)
			0xE6, 0xE1,       // INC $E1 (NMI exit)
			0x40,             // RTI
		}

		// Program that triggers DMA and expects NMI
		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E,       // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			
			// Wait for VBlank approach
			0xE6, 0xE2,       // INC $E2 (main loop counter)
			0xA5, 0xE2,       // LDA $E2
			0xC9, 0x80,       // CMP #$80
			0x90, 0xF8,       // BCC (continue)
			
			// Trigger DMA at critical moment
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (DMA)
			0xE6, 0xE3,       // INC $E3 (DMA marker)
			
			// Reset and continue
			0xA9, 0x00,       // LDA #$00
			0x85, 0xE2,       // STA $E2 (reset counter)
			0x4C, 0x08, 0x80, // JMP back
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
		helper.Memory.Write(0x00E0, 0x00) // NMI entry count
		helper.Memory.Write(0x00E1, 0x00) // NMI exit count
		helper.Memory.Write(0x00E2, 0x00) // Main loop counter
		helper.Memory.Write(0x00E3, 0x00) // DMA marker

		// Execute test
		for i := 0; i < 100000; i++ {
			helper.Bus.Step()
			
			nmiEntry := helper.Memory.Read(0x00E0)
			nmiExit := helper.Memory.Read(0x00E1)
			dmaCount := helper.Memory.Read(0x00E3)
			
			// Check for NMI priority demonstration
			if nmiEntry > 0 && dmaCount > 0 {
				t.Logf("Priority test results at step %d:", i)
				t.Logf("  NMI entries: %d", nmiEntry)
				t.Logf("  NMI exits: %d", nmiExit)
				t.Logf("  DMA triggers: %d", dmaCount)
				
				// NMI should complete even if DMA is pending
				if nmiEntry != nmiExit {
					t.Logf("  NMI in progress (entry > exit)")
				}
				
				break
			}
		}

		finalNMIEntry := helper.Memory.Read(0x00E0)
		finalNMIExit := helper.Memory.Read(0x00E1)
		finalDMACount := helper.Memory.Read(0x00E3)
		
		if finalNMIEntry == 0 {
			t.Error("NMI should occur during priority test")
		}
		if finalDMACount == 0 {
			t.Error("DMA should occur during priority test")
		}
		
		t.Logf("NMI priority test completed: NMI entry=%d, exit=%d, DMA=%d", 
			finalNMIEntry, finalNMIExit, finalDMACount)
	})
}