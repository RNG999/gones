package integration

import (
	"testing"
)

// TestOAMDMATransferFunctionality tests comprehensive OAM DMA transfer behavior
func TestOAMDMATransferFunctionality(t *testing.T) {
	t.Run("Basic_OAM_DMA_Transfer", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Setup test program that triggers DMA
		program := []uint8{
			// Initialize sprite data in RAM page 2 ($0200-$02FF)
			0xA2, 0x00,       // LDX #$00
			0xA9, 0x80,       // LDA #$80 (Y position)
			0x9D, 0x00, 0x02, // STA $0200,X (store Y position)
			0xA9, 0x20,       // LDA #$20 (tile index)
			0x9D, 0x01, 0x02, // STA $0201,X (store tile index)
			0xA9, 0x00,       // LDA #$00 (attributes)
			0x9D, 0x02, 0x02, // STA $0202,X (store attributes)
			0xA9, 0x88,       // LDA #$88 (X position)
			0x9D, 0x03, 0x02, // STA $0203,X (store X position)
			// Increment to next sprite (4 bytes per sprite)
			0x8A,             // TXA
			0x18,             // CLC
			0x69, 0x04,       // ADC #$04
			0xAA,             // TAX
			0xE0, 0x10,       // CPX #$10 (compare with 16 - 4 sprites)
			0x90, 0xE5,       // BCC (loop back)
			// Trigger DMA transfer
			0xA9, 0x02,       // LDA #$02 (source page $0200)
			0x8D, 0x14, 0x40, // STA $4014 (OAMDMA register)
			// Verification loop
			0xEA,             // NOP
			0x4C, 0x25, 0x80, // JMP to verification
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute sprite initialization
		for i := 0; i < 100; i++ {
			helper.Bus.Step()
			
			// Check if we've reached the DMA trigger
			if helper.CPU.PC == 0x8020 { // LDA #$02 before DMA
				break
			}
		}

		// Verify sprite data was set up
		for i := 0; i < 4; i++ {
			spriteBase := 0x0200 + i*4
			y := helper.Memory.Read(uint16(spriteBase))
			tile := helper.Memory.Read(uint16(spriteBase + 1))
			attr := helper.Memory.Read(uint16(spriteBase + 2))
			x := helper.Memory.Read(uint16(spriteBase + 3))

			if y != 0x80 || tile != 0x20 || attr != 0x00 || x != 0x88 {
				t.Errorf("Sprite %d data incorrect: Y=%02X, Tile=%02X, Attr=%02X, X=%02X",
					i, y, tile, attr, x)
			}
		}

		// Record state before DMA
		beforeDMACycles := helper.Bus.GetCycleCount()
		dmaInProgress := helper.Bus.IsDMAInProgress()

		t.Logf("Before DMA: cycles=%d, DMA in progress=%v", beforeDMACycles, dmaInProgress)

		// Execute DMA trigger
		helper.Bus.Step() // LDA #$02
		helper.Bus.Step() // STA $4014 (triggers DMA)

		// DMA should now be in progress
		afterTriggerCycles := helper.Bus.GetCycleCount()
		dmaInProgress = helper.Bus.IsDMAInProgress()

		t.Logf("After DMA trigger: cycles=%d, DMA in progress=%v", afterTriggerCycles, dmaInProgress)

		if !dmaInProgress {
			t.Error("DMA should be in progress after trigger")
		}

		// Continue execution until DMA completes
		dmaCycles := 0
		for helper.Bus.IsDMAInProgress() && dmaCycles < 1000 {
			helper.Bus.Step()
			dmaCycles++
		}

		finalCycles := helper.Bus.GetCycleCount()
		totalDMACycles := finalCycles - afterTriggerCycles

		t.Logf("DMA completed after %d steps, %d total cycles", dmaCycles, totalDMACycles)

		// Verify DMA transfer timing (should be 513/514 cycles depending on alignment)
		if totalDMACycles < 513 || totalDMACycles > 514 {
			t.Errorf("DMA cycles %d outside expected range 513-514", totalDMACycles)
		}

		// TODO: Verify OAM data was transferred correctly
		// This would require access to PPU OAM memory for verification
	})

	t.Run("DMA_CPU_Suspension_Timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that measures DMA suspension precisely
		program := []uint8{
			// Set up test pattern in RAM
			0xA2, 0x00,       // LDX #$00
			0xA9, 0xAA,       // LDA #$AA
			0x9D, 0x00, 0x03, // STA $0300,X
			0xE8,             // INX
			0xD0, 0xFA,       // BNE (loop)
			// Reset counter
			0xA9, 0x00,       // LDA #$00
			0x85, 0x80,       // STA $80 (cycle counter)
			// Trigger DMA and start counting
			0xA9, 0x03,       // LDA #$03 (source page)
			0x8D, 0x14, 0x40, // STA $4014 (trigger DMA)
			// This instruction should be delayed by DMA
			0xE6, 0x80,       // INC $80 (increment counter)
			0xE6, 0x80,       // INC $80
			0xE6, 0x80,       // INC $80
			0x4C, 0x15, 0x80, // JMP to continue counting
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute setup
		for i := 0; i < 20; i++ {
			helper.Bus.Step()
			if helper.CPU.PC == 0x800C { // At counter reset
				break
			}
		}

		// Clear counter and prepare for DMA timing test
		helper.Memory.Write(0x0080, 0x00)

		beforeDMACycles := helper.Bus.GetCycleCount()
		
		// Execute DMA trigger
		helper.Bus.Step() // LDA #$03
		helper.Bus.Step() // STA $4014

		// DMA should suspend CPU
		if !helper.Bus.IsDMAInProgress() {
			t.Error("DMA should be in progress")
		}

		// Execute steps during DMA - CPU should be suspended
		suspensionSteps := 0
		for helper.Bus.IsDMAInProgress() && suspensionSteps < 600 {
			helper.Bus.Step()
			suspensionSteps++
		}

		afterDMACycles := helper.Bus.GetCycleCount()
		dmaDuration := afterDMACycles - beforeDMACycles

		t.Logf("DMA suspension lasted %d steps, %d cycles", suspensionSteps, dmaDuration)

		// Verify CPU was suspended (counter should not have incremented during DMA)
		counterValue := helper.Memory.Read(0x0080)
		if counterValue != 0 {
			t.Errorf("CPU should be suspended during DMA, but counter = %d", counterValue)
		}

		// Continue execution briefly to see counter increment after DMA
		for i := 0; i < 10; i++ {
			helper.Bus.Step()
		}

		finalCounterValue := helper.Memory.Read(0x0080)
		t.Logf("Counter value after DMA completion: %d", finalCounterValue)

		if finalCounterValue == 0 {
			t.Error("CPU should resume execution after DMA completion")
		}
	})

	t.Run("DMA_Even_Odd_Cycle_Timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test DMA timing on both even and odd CPU cycles
		program := []uint8{
			// Setup test data
			0xA9, 0x55,       // LDA #$55
			0x85, 0x00,       // STA $00
			// Force even cycle alignment
			0xEA,             // NOP (2 cycles)
			0xA9, 0x00,       // LDA #$00 (2 cycles) - should be on even cycle
			0x8D, 0x14, 0x40, // STA $4014 (4 cycles) - trigger DMA on even
			0xE6, 0x90,       // INC $90 (mark even test)
			// Reset for odd cycle test
			0xEA,             // NOP (2 cycles)
			0xEA,             // NOP (2 cycles) 
			0xEA,             // NOP (2 cycles) - odd alignment
			0xA9, 0x00,       // LDA #$00 (2 cycles) - should be on odd cycle
			0x8D, 0x14, 0x40, // STA $4014 (4 cycles) - trigger DMA on odd
			0xE6, 0x91,       // INC $91 (mark odd test)
			0x4C, 0x1A, 0x80, // JMP to end
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		helper.Memory.Write(0x0090, 0x00) // Even cycle marker
		helper.Memory.Write(0x0091, 0x00) // Odd cycle marker

		// Execute program and measure both DMA timings
		evenDMAStart := uint64(0)
		evenDMAEnd := uint64(0)
		oddDMAStart := uint64(0)
		oddDMAEnd := uint64(0)

		for i := 0; i < 1000; i++ {
			if helper.CPU.PC == 0x8008 { // Just before even DMA trigger
				evenDMAStart = helper.Bus.GetCycleCount()
			}

			helper.Bus.Step()

			// Check for even DMA completion
			if helper.Memory.Read(0x0090) > 0 && evenDMAEnd == 0 {
				evenDMAEnd = helper.Bus.GetCycleCount()
			}

			if helper.CPU.PC == 0x8016 { // Just before odd DMA trigger
				oddDMAStart = helper.Bus.GetCycleCount()
			}

			// Check for odd DMA completion
			if helper.Memory.Read(0x0091) > 0 && oddDMAEnd == 0 {
				oddDMAEnd = helper.Bus.GetCycleCount()
				break
			}
		}

		evenDuration := evenDMAEnd - evenDMAStart
		oddDuration := oddDMAEnd - oddDMAStart

		t.Logf("Even cycle DMA duration: %d cycles", evenDuration)
		t.Logf("Odd cycle DMA duration: %d cycles", oddDuration)

		// Even cycle should be 513, odd cycle should be 514
		if evenDuration != 513 {
			t.Errorf("Even cycle DMA expected 513 cycles, got %d", evenDuration)
		}
		if oddDuration != 514 {
			t.Errorf("Odd cycle DMA expected 514 cycles, got %d", oddDuration)
		}
	})

	t.Run("DMA_Source_Page_Validation", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test DMA from different source pages
		testPages := []struct {
			page        uint8
			description string
		}{
			{0x00, "RAM page 0"},
			{0x01, "RAM page 1"},
			{0x02, "RAM page 2"},
			{0x03, "RAM page 3"},
			{0x20, "PPU registers"},
			{0x40, "APU registers"},
			{0x80, "PRG ROM low"},
			{0xFF, "PRG ROM high"},
		}

		for _, testPage := range testPages {
			t.Run(testPage.description, func(t *testing.T) {
				helper.Bus.Reset()

				// Set up test data in source page (for RAM pages)
				if testPage.page < 0x08 {
					baseAddr := uint16(testPage.page) << 8
					for i := 0; i < 256; i++ {
						// Map to actual RAM location (with mirroring)
						ramAddr := (baseAddr + uint16(i)) & 0x07FF
						helper.Memory.Write(0x0000+ramAddr, uint8(i^0xAA))
					}
				}

				// Simple DMA trigger program
				program := []uint8{
					0xA9, testPage.page, // LDA #page
					0x8D, 0x14, 0x40,    // STA $4014
					0xEA,                // NOP
					0x4C, 0x05, 0x80,    // JMP to NOP loop
				}

				romData := make([]uint8, 0x8000)
				copy(romData, program)
				romData[0x7FFC] = 0x00
				romData[0x7FFD] = 0x80

				helper.GetMockCartridge().LoadPRG(romData)
				helper.Bus.Reset()

				// Execute DMA trigger
				helper.Bus.Step() // LDA #page
				helper.Bus.Step() // STA $4014

				// Verify DMA started
				if !helper.Bus.IsDMAInProgress() {
					t.Errorf("DMA should start for page 0x%02X", testPage.page)
					return
				}

				// Wait for DMA completion
				for helper.Bus.IsDMAInProgress() {
					helper.Bus.Step()
				}

				t.Logf("DMA completed for source page 0x%02X", testPage.page)
			})
		}
	})

	t.Run("Multiple_Sequential_DMA_Transfers", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that performs multiple DMA transfers
		program := []uint8{
			// Set up different data in multiple pages
			0xA2, 0x00,       // LDX #$00
			0xA9, 0x11,       // LDA #$11
			0x9D, 0x00, 0x02, // STA $0200,X (page 2)
			0xA9, 0x22,       // LDA #$22
			0x9D, 0x00, 0x03, // STA $0300,X (page 3)
			0xE8,             // INX
			0xD0, 0xF4,       // BNE (loop)
			
			// Perform multiple DMA transfers
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (first DMA)
			0xE6, 0xA0,       // INC $A0 (mark first DMA)
			
			0xA9, 0x03,       // LDA #$03
			0x8D, 0x14, 0x40, // STA $4014 (second DMA)
			0xE6, 0xA1,       // INC $A1 (mark second DMA)
			
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (third DMA)
			0xE6, 0xA2,       // INC $A2 (mark third DMA)
			
			0x4C, 0x26, 0x80, // JMP to end
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize markers
		helper.Memory.Write(0x00A0, 0x00)
		helper.Memory.Write(0x00A1, 0x00)
		helper.Memory.Write(0x00A2, 0x00)

		// Execute entire program
		totalSteps := 0
		dmaCount := 0
		
		for totalSteps < 2000 {
			helper.Bus.Step()
			totalSteps++

			// Count completed DMAs
			newDMACount := int(helper.Memory.Read(0x00A0)) + 
							int(helper.Memory.Read(0x00A1)) + 
							int(helper.Memory.Read(0x00A2))
			
			if newDMACount > dmaCount {
				t.Logf("DMA %d completed at step %d", newDMACount, totalSteps)
				dmaCount = newDMACount
			}

			// Break when all three DMAs are done
			if dmaCount >= 3 {
				break
			}
		}

		if dmaCount < 3 {
			t.Errorf("Expected 3 DMAs, only %d completed", dmaCount)
		}

		t.Logf("All %d DMA transfers completed successfully", dmaCount)
	})
}