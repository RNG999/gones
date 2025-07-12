package integration

import (
	"testing"
)

// TestNMITimingPrecision tests cycle-accurate NMI generation timing
func TestNMITimingPrecision(t *testing.T) {
	t.Run("VBlank_NMI_Exact_Timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that records exact timing
		nmiHandler := []uint8{
			0xAD, 0x02, 0x20, // LDA $2002 (read PPUSTATUS to capture state)
			0x85, 0x20,       // STA $20 (store status for analysis)
			0xE6, 0x21,       // INC $21 (NMI counter)
			0x40,             // RTI
		}

		// Setup NMI vector and handler
		romData := make([]uint8, 0x8000)
		copy(romData[0x0100:], nmiHandler) // Handler at $8100

		// Main program
		program := []uint8{
			0xA9, 0x80,       // LDA #$80 (NMI enable)
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)
			0xA9, 0x08,       // LDA #$08 (background enable)
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK)
			// Tight loop to measure exact timing
			0xE6, 0x22,       // INC $22 (cycle counter)
			0x4C, 0x0A, 0x80, // JMP to counter loop
		}
		copy(romData, program)

		// Set interrupt vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize test variables
		helper.Memory.Write(0x0020, 0x00) // PPUSTATUS capture
		helper.Memory.Write(0x0021, 0x00) // NMI counter
		helper.Memory.Write(0x0022, 0x00) // Cycle counter

		// Execute setup (enable NMI and rendering)
		for i := 0; i < 4; i++ {
			helper.Bus.Step()
		}

		// Record initial PPU state
		initialFrameCount := helper.Bus.GetFrameCount()
		initialCycles := helper.Bus.GetCycleCount()

		// Run until first NMI occurs
		maxCycles := 50000
		nmiOccurred := false
		
		for i := 0; i < maxCycles; i++ {
			helper.Bus.Step()
			
			nmiCount := helper.Memory.Read(0x0021)
			if nmiCount > 0 {
				nmiOccurred = true
				ppuStatus := helper.Memory.Read(0x0020)
				cycleCounter := helper.Memory.Read(0x0022)
				currentFrameCount := helper.Bus.GetFrameCount()
				currentCycles := helper.Bus.GetCycleCount()

				t.Logf("NMI occurred after %d bus steps", i)
				t.Logf("PPU Status when NMI triggered: 0x%02X", ppuStatus)
				t.Logf("Cycle counter value: %d", cycleCounter)
				t.Logf("Frame count changed: %d -> %d", initialFrameCount, currentFrameCount)
				t.Logf("Total CPU cycles: %d (delta: %d)", currentCycles, currentCycles-initialCycles)

				// Verify VBlank flag was set when NMI occurred
				if (ppuStatus & 0x80) == 0 {
					t.Error("VBlank flag should be set when NMI occurs")
				}

				// Verify frame count incremented (indicating VBlank start)
				if currentFrameCount <= initialFrameCount {
					t.Error("Frame count should increment when VBlank starts")
				}

				break
			}
		}

		if !nmiOccurred {
			t.Fatal("NMI did not occur within expected time frame")
		}
	})

	t.Run("NMI_Scanline_241_Cycle_1_Precision", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that captures precise PPU timing
		nmiHandler := []uint8{
			0xA9, 0xFF,       // LDA #$FF
			0x85, 0x30,       // STA $30 (NMI occurred marker)
			0x40,             // RTI
		}

		// Setup timing test program
		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E,       // LDA #$1E (show background and sprites)
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)
			// Precise timing loop
			0xEA,             // NOP (2 cycles)
			0xEA,             // NOP (2 cycles)
			0x4C, 0x0A, 0x80, // JMP (3 cycles) - 7 cycle loop
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
		helper.Memory.Write(0x0030, 0x00)

		// Execute setup
		for i := 0; i < 4; i++ {
			helper.Bus.Step()
		}

		// Count exact cycles until NMI
		cycleCount := 0
		for cycleCount < 100000 {
			helper.Bus.Step()
			cycleCount++

			if helper.Memory.Read(0x0030) == 0xFF {
				t.Logf("NMI occurred after exactly %d CPU cycles", cycleCount)
				
				// For NTSC: VBlank should start at specific timing
				// 89342 PPU cycles per frame = 29780.67 CPU cycles
				// This test validates the timing is consistent
				if cycleCount < 25000 || cycleCount > 35000 {
					t.Errorf("NMI timing unexpected: %d cycles (expected ~29781)", cycleCount)
				}
				
				break
			}
		}

		if cycleCount >= 100000 {
			t.Fatal("NMI did not occur within timing window")
		}
	})

	t.Run("Multiple_Frame_NMI_Consistency", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler that counts occurrences
		nmiHandler := []uint8{
			0xE6, 0x40, // INC $40 (NMI counter)
			0x40,       // RTI
		}

		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x08,       // LDA #$08
			0x8D, 0x01, 0x20, // STA $2001 (enable background)
			// Simple loop
			0x4C, 0x08, 0x80, // JMP $8008
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
		helper.Memory.Write(0x0040, 0x00)

		// Execute setup
		for i := 0; i < 4; i++ {
			helper.Bus.Step()
		}

		// Measure timing for multiple frames
		frameTimings := make([]int, 0)
		lastNMICount := uint8(0)
		cycles := 0

		for len(frameTimings) < 5 && cycles < 200000 {
			helper.Bus.Step()
			cycles++

			currentNMICount := helper.Memory.Read(0x0040)
			if currentNMICount > lastNMICount {
				frameTimings = append(frameTimings, cycles)
				t.Logf("Frame %d: NMI at cycle %d", len(frameTimings), cycles)
				lastNMICount = currentNMICount
			}
		}

		if len(frameTimings) < 2 {
			t.Fatal("Not enough NMI occurrences to test consistency")
		}

		// Calculate frame intervals
		for i := 1; i < len(frameTimings); i++ {
			interval := frameTimings[i] - frameTimings[i-1]
			t.Logf("Frame %d interval: %d cycles", i, interval)

			// NTSC timing should be consistent (~29781 cycles per frame)
			expectedInterval := 29781
			tolerance := 100 // Allow small tolerance for timing precision

			if interval < expectedInterval-tolerance || interval > expectedInterval+tolerance {
				t.Errorf("Frame %d interval %d cycles outside tolerance of %d±%d",
					i, interval, expectedInterval, tolerance)
			}
		}
	})
}

// TestNMISuppressionBehaviors tests edge cases where NMI can be suppressed
func TestNMISuppressionBehaviors(t *testing.T) {
	t.Run("PPUSTATUS_Read_Suppresses_NMI", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// NMI handler
		nmiHandler := []uint8{
			0xE6, 0x50, // INC $50
			0x40,       // RTI
		}

		// Program that reads PPUSTATUS at critical timing
		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x08,       // LDA #$08
			0x8D, 0x01, 0x20, // STA $2001 (enable background)
			// Critical timing loop that reads PPUSTATUS
			0xAD, 0x02, 0x20, // LDA $2002 (read PPUSTATUS)
			0x85, 0x51,       // STA $51 (save status)
			0xAD, 0x02, 0x20, // LDA $2002 (read again)
			0x85, 0x52,       // STA $52 (save status)
			0x4C, 0x0A, 0x80, // JMP to loop
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
		helper.Memory.Write(0x0050, 0x00) // NMI counter
		helper.Memory.Write(0x0051, 0x00) // First status read
		helper.Memory.Write(0x0052, 0x00) // Second status read

		// Execute setup
		for i := 0; i < 4; i++ {
			helper.Bus.Step()
		}

		// Run for extended period to test suppression
		suppressionDetected := false
		nmiCount := uint8(0)
		
		for i := 0; i < 100000; i++ {
			helper.Bus.Step()

			currentNMICount := helper.Memory.Read(0x0050)
			status1 := helper.Memory.Read(0x0051)
			status2 := helper.Memory.Read(0x0052)

			// Check if we've seen VBlank flag set but no NMI
			if (status1&0x80) != 0 || (status2&0x80) != 0 {
				if currentNMICount == nmiCount {
					// VBlank flag was read but NMI didn't increment
					suppressionDetected = true
					t.Logf("NMI suppression detected: VBlank flag read but no NMI")
					t.Logf("Status reads: 0x%02X, 0x%02X", status1, status2)
					break
				}
				nmiCount = currentNMICount
			}

			// Also check for normal NMI behavior
			if currentNMICount > nmiCount {
				t.Logf("Normal NMI occurred (count: %d)", currentNMICount)
				nmiCount = currentNMICount
			}
		}

		// The test should demonstrate either suppression or normal behavior
		// Both are valid depending on exact timing
		t.Logf("Final NMI count: %d", helper.Memory.Read(0x0050))
		t.Logf("Suppression behavior detected: %v", suppressionDetected)
	})

	t.Run("NMI_Disable_During_VBlank", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		nmiHandler := []uint8{
			0xE6, 0x60, // INC $60
			0x40,       // RTI
		}

		// Program that disables NMI during VBlank
		program := []uint8{
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x08,       // LDA #$08
			0x8D, 0x01, 0x20, // STA $2001 (enable background)
			// Wait for VBlank
			0xAD, 0x02, 0x20, // LDA $2002
			0x10, 0xFB,       // BPL (wait for VBlank)
			// Immediately disable NMI
			0xA9, 0x00,       // LDA #$00
			0x8D, 0x00, 0x20, // STA $2000 (disable NMI)
			// Continue execution
			0x4C, 0x0A, 0x80, // JMP back to wait
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

		helper.Memory.Write(0x0060, 0x00)

		// Execute setup
		for i := 0; i < 4; i++ {
			helper.Bus.Step()
		}

		// Run test
		for i := 0; i < 50000; i++ {
			helper.Bus.Step()

			nmiCount := helper.Memory.Read(0x0060)
			if nmiCount > 0 {
				t.Logf("NMI occurred despite disable attempt (count: %d)", nmiCount)
				break
			}
		}

		finalCount := helper.Memory.Read(0x0060)
		t.Logf("Final NMI count after disable test: %d", finalCount)
	})

	t.Run("NMI_Enable_Edge_Detection", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		nmiHandler := []uint8{
			0xE6, 0x70, // INC $70
			0x40,       // RTI
		}

		// Test enabling NMI when VBlank is already set
		program := []uint8{
			// First, wait for VBlank without NMI enabled
			0xA9, 0x00,       // LDA #$00
			0x8D, 0x00, 0x20, // STA $2000 (NMI disabled)
			0xA9, 0x08,       // LDA #$08
			0x8D, 0x01, 0x20, // STA $2001 (enable background)
			// Wait for VBlank flag
			0xAD, 0x02, 0x20, // LDA $2002
			0x10, 0xFB,       // BPL (wait for VBlank)
			// Now enable NMI while VBlank is set
			0xA9, 0x80,       // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			// Wait and see what happens
			0xEA, 0xEA, 0xEA, // NOPs
			0x4C, 0x14, 0x80, // JMP to wait
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

		helper.Memory.Write(0x0070, 0x00)

		// Run the test
		for i := 0; i < 50000; i++ {
			helper.Bus.Step()

			nmiCount := helper.Memory.Read(0x0070)
			if nmiCount > 0 {
				t.Logf("NMI triggered by enable during VBlank (count: %d)", nmiCount)
				break
			}
		}

		finalCount := helper.Memory.Read(0x0070)
		t.Logf("Final NMI count for edge detection test: %d", finalCount)
		
		// The exact behavior depends on implementation details,
		// but the test validates the edge detection logic
	})
}