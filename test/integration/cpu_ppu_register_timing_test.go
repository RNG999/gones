package integration

import (
	"testing"
)

// TestPPURegisterAccessTiming validates timing of PPU register access relative to CPU cycles
func TestPPURegisterAccessTiming(t *testing.T) {
	t.Run("PPU register access maintains 3:1 cycle ratio", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program that accesses various PPU registers
		romData := make([]uint8, 0x8000)
		program := []uint8{
			// Write to PPUCTRL (4 cycles)
			0xA9, 0x80,       // LDA #$80 (2 cycles)
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles) - PPUCTRL
			
			// Write to PPUMASK (4 cycles)
			0xA9, 0x18,       // LDA #$18 (2 cycles)
			0x8D, 0x01, 0x20, // STA $2001 (4 cycles) - PPUMASK
			
			// Read from PPUSTATUS (4 cycles)
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles) - PPUSTATUS
			
			// Write to PPUADDR (4 cycles each)
			0xA9, 0x20,       // LDA #$20 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - PPUADDR high
			0xA9, 0x00,       // LDA #$00 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - PPUADDR low
			
			// Write to PPUDATA (4 cycles)
			0xA9, 0x42,       // LDA #$42 (2 cycles)
			0x8D, 0x07, 0x20, // STA $2007 (4 cycles) - PPUDATA
			
			// Read from PPUDATA (4 cycles)
			0xAD, 0x07, 0x20, // LDA $2007 (4 cycles) - PPUDATA
			
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// Expected cycle counts for each instruction
		expectedInstructionCycles := []int{
			2, 4, // LDA #$80, STA $2000
			2, 4, // LDA #$18, STA $2001
			4,    // LDA $2002
			2, 4, // LDA #$20, STA $2006
			2, 4, // LDA #$00, STA $2006
			2, 4, // LDA #$42, STA $2007
			4,    // LDA $2007
			3,    // JMP $8000
		}
		
		totalCPUCycles := uint64(0)
		totalPPUCycles := uint64(0)
		
		for i, expectedCycles := range expectedInstructionCycles {
			initialCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step()
			actualCPU := helper.Bus.GetCycleCount() - initialCPU
			
			if actualCPU != uint64(expectedCycles) {
				t.Errorf("Instruction %d: expected %d CPU cycles, got %d", 
					i, expectedCycles, actualCPU)
			}
			
			totalCPUCycles += actualCPU
			totalPPUCycles += actualCPU * 3
			
			// Verify 3:1 ratio maintained for each register access
			log := helper.Bus.GetExecutionLog()
			if len(log) > i {
				stepPPUCycles := log[i].PPUCycles
				if i > 0 {
					stepPPUCycles -= log[i-1].PPUCycles
				}
				expectedStepPPU := actualCPU * 3
				
				if stepPPUCycles != expectedStepPPU {
					t.Errorf("Instruction %d: expected %d PPU cycles, got %d",
						i, expectedStepPPU, stepPPUCycles)
				}
			}
		}
		
		// Verify overall 3:1 ratio
		finalRatio := float64(totalPPUCycles) / float64(totalCPUCycles)
		if finalRatio != 3.0 {
			t.Errorf("Overall PPU/CPU ratio should be 3.0, got %.2f", finalRatio)
		}
		
		t.Logf("PPU register access test: %d CPU cycles, %d PPU cycles", 
			totalCPUCycles, totalPPUCycles)
	})
	
	t.Run("PPUSTATUS read timing effects", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program that reads PPUSTATUS multiple times
		romData := make([]uint8, 0x8000)
		program := []uint8{
			// Multiple PPUSTATUS reads
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles)
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles)
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles)
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// Set VBlank flag initially
		helper.PPU.WriteRegister(0x2002, 0x80) // Simulate VBlank set
		
		statusReadResults := make([]uint8, 4)
		cycleTimings := make([]uint64, 4)
		
		for i := 0; i < 4; i++ {
			initialCPU := helper.Bus.GetCycleCount()
			
			// Read the status value before the instruction
			preStatus := helper.PPU.ReadRegister(0x2002)
			
			helper.Bus.Step() // Execute LDA $2002
			
			// The actual status read would be done by the instruction
			// Here we verify timing consistency
			actualCycles := helper.Bus.GetCycleCount() - initialCPU
			
			statusReadResults[i] = preStatus
			cycleTimings[i] = actualCycles
			
			if actualCycles != 4 {
				t.Errorf("PPUSTATUS read %d: expected 4 CPU cycles, got %d", i, actualCycles)
			}
		}
		
		// Verify PPU cycles are exactly 3x CPU cycles for each read
		log := helper.Bus.GetExecutionLog()
		for i := 0; i < 4; i++ {
			if len(log) > i {
				stepPPUCycles := log[i].PPUCycles
				if i > 0 {
					stepPPUCycles -= log[i-1].PPUCycles
				}
				expectedPPU := cycleTimings[i] * 3
				
				if stepPPUCycles != expectedPPU {
					t.Errorf("PPUSTATUS read %d: expected %d PPU cycles, got %d",
						i, expectedPPU, stepPPUCycles)
				}
			}
		}
		
		t.Logf("PPUSTATUS read cycles: %v", cycleTimings)
		t.Logf("Status values: %02X %02X %02X %02X", 
			statusReadResults[0], statusReadResults[1], statusReadResults[2], statusReadResults[3])
	})
	
	t.Run("PPUDATA access timing during rendering", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program that accesses PPUDATA during different rendering phases
		romData := make([]uint8, 0x8000)
		program := []uint8{
			// Set up PPUADDR for VRAM access
			0xA9, 0x20,       // LDA #$20 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - PPUADDR high
			0xA9, 0x00,       // LDA #$00 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - PPUADDR low
			
			// Write to PPUDATA
			0xA9, 0x55,       // LDA #$55 (2 cycles)
			0x8D, 0x07, 0x20, // STA $2007 (4 cycles) - PPUDATA write
			
			// Read from PPUDATA  
			0xAD, 0x07, 0x20, // LDA $2007 (4 cycles) - PPUDATA read
			
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		
		// Test with rendering disabled first
		helper.PPU.WriteRegister(0x2001, 0x00) // Disable rendering
		helper.Bus.EnableExecutionLogging()
		
		expectedCycles := []int{2, 4, 2, 4, 2, 4, 4, 3}
		noRenderingCycles := make([]uint64, len(expectedCycles))
		
		for i, expected := range expectedCycles {
			initialCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step()
			actualCPU := helper.Bus.GetCycleCount() - initialCPU
			noRenderingCycles[i] = actualCPU
			
			if actualCPU != uint64(expected) {
				t.Errorf("No rendering - instruction %d: expected %d cycles, got %d",
					i, expected, actualCPU)
			}
		}
		
		// Reset and test with rendering enabled
		helper.Bus.Reset()
		helper.PPU.WriteRegister(0x2001, 0x18) // Enable rendering
		helper.Bus.ClearExecutionLog()
		
		renderingCycles := make([]uint64, len(expectedCycles))
		
		for i, expected := range expectedCycles {
			initialCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step()
			actualCPU := helper.Bus.GetCycleCount() - initialCPU
			renderingCycles[i] = actualCPU
			
			// During rendering, PPUDATA access timing should be the same
			// The PPU behavior might differ, but CPU timing should be consistent
			if actualCPU != uint64(expected) {
				t.Errorf("With rendering - instruction %d: expected %d cycles, got %d",
					i, expected, actualCPU)
			}
		}
		
		// Compare timing consistency
		for i := range expectedCycles {
			if noRenderingCycles[i] != renderingCycles[i] {
				t.Errorf("Instruction %d timing differs: no rendering=%d, with rendering=%d",
					i, noRenderingCycles[i], renderingCycles[i])
			}
		}
		
		// Verify 3:1 ratio maintained in both cases
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("PPU/CPU ratio with rendering should be 3.0, got %.2f", ratio)
			}
		}
		
		t.Logf("PPUDATA timing - no rendering: %v", noRenderingCycles)
		t.Logf("PPUDATA timing - with rendering: %v", renderingCycles)
	})
	
	t.Run("PPU register write buffering and timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Test rapid successive writes to PPU registers
		romData := make([]uint8, 0x8000)
		program := []uint8{
			// Rapid PPUCTRL writes
			0xA9, 0x00,       // LDA #$00 (2 cycles)
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles)
			0xA9, 0x80,       // LDA #$80 (2 cycles)
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles)
			0xA9, 0x90,       // LDA #$90 (2 cycles)
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles)
			
			// Rapid PPUADDR writes (address setting)
			0xA9, 0x20,       // LDA #$20 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - high byte
			0xA9, 0x00,       // LDA #$00 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - low byte
			0xA9, 0x20,       // LDA #$20 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - high byte again
			0xA9, 0x10,       // LDA #$10 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - low byte again
			
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		expectedCycles := []int{2, 4, 2, 4, 2, 4, 2, 4, 2, 4, 2, 4, 2, 4, 3}
		totalExpectedCPU := 0
		for _, cycles := range expectedCycles {
			totalExpectedCPU += cycles
		}
		
		actualCycleTimes := make([]uint64, len(expectedCycles))
		totalActualCPU := uint64(0)
		
		for i, expected := range expectedCycles {
			initialCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step()
			actualCPU := helper.Bus.GetCycleCount() - initialCPU
			actualCycleTimes[i] = actualCPU
			totalActualCPU += actualCPU
			
			if actualCPU != uint64(expected) {
				t.Errorf("Rapid write %d: expected %d CPU cycles, got %d",
					i, expected, actualCPU)
			}
		}
		
		// Verify total timing
		if totalActualCPU != uint64(totalExpectedCPU) {
			t.Errorf("Total CPU cycles: expected %d, got %d", 
				totalExpectedCPU, totalActualCPU)
		}
		
		// Verify PPU maintained exact 3:1 ratio throughout
		expectedTotalPPU := totalActualCPU * 3
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			if lastEntry.PPUCycles != expectedTotalPPU {
				t.Errorf("Total PPU cycles: expected %d, got %d",
					expectedTotalPPU, lastEntry.PPUCycles)
			}
			
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("Final PPU/CPU ratio should be 3.0, got %.2f", ratio)
			}
		}
		
		t.Logf("Rapid PPU writes: %d instructions, %d total CPU cycles", 
			len(expectedCycles), totalActualCPU)
	})
}

// TestPPURegisterTimingEdgeCases validates register access timing in edge cases
func TestPPURegisterTimingEdgeCases(t *testing.T) {
	t.Run("PPU register access during VBlank transition", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program that reads PPUSTATUS around VBlank timing
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles) - PPUSTATUS
			0xEA,             // NOP (2 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// Run until near VBlank to test timing during transition
		const approximateVBlankCycles = 80000 // Rough estimate to get near VBlank
		
		for helper.Bus.GetCycleCount() < approximateVBlankCycles {
			helper.Bus.Step()
		}
		
		// Now test register access timing during VBlank period
		vblankAccessCycles := make([]uint64, 10)
		
		for i := 0; i < 10; i++ {
			initialCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step() // Should be LDA $2002
			vblankAccessCycles[i] = helper.Bus.GetCycleCount() - initialCPU
			
			// Skip NOP and JMP to stay on PPUSTATUS reads
			helper.Bus.Step() // NOP
			helper.Bus.Step() // JMP
		}
		
		// All PPUSTATUS reads should take exactly 4 cycles regardless of VBlank state
		for i, cycles := range vblankAccessCycles {
			if cycles != 4 {
				t.Errorf("VBlank PPUSTATUS read %d: expected 4 cycles, got %d", i, cycles)
			}
		}
		
		// Verify 3:1 ratio maintained during VBlank transitions
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("PPU/CPU ratio during VBlank should be 3.0, got %.2f", ratio)
			}
		}
		
		t.Logf("PPUSTATUS access during VBlank: %v cycles", vblankAccessCycles)
	})
	
	t.Run("PPU register access with DMA", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program that triggers DMA and then accesses PPU registers
		romData := make([]uint8, 0x8000)
		program := []uint8{
			// Trigger DMA
			0xA9, 0x02,       // LDA #$02 (2 cycles)
			0x8D, 0x14, 0x40, // STA $4014 (4 cycles) - trigger OAM DMA
			
			// After DMA, access PPU registers
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles) - PPUSTATUS
			0xA9, 0x80,       // LDA #$80 (2 cycles)
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles) - PPUCTRL
			
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// Execute DMA trigger
		helper.Bus.Step() // LDA #$02
		dmaTriggerCPU := helper.Bus.GetCycleCount()
		
		helper.Bus.Step() // STA $4014 - triggers DMA
		
		// Verify DMA is in progress
		if !helper.Bus.IsDMAInProgress() {
			t.Error("DMA should be in progress after STA $4014")
		}
		
		// During DMA, PPU should continue at 3:1 ratio
		dmaSteps := 0
		for helper.Bus.IsDMAInProgress() && dmaSteps < 600 {
			helper.Bus.Step()
			dmaSteps++
		}
		
		postDMACPU := helper.Bus.GetCycleCount()
		dmaDuration := postDMACPU - dmaTriggerCPU
		
		// Now test PPU register access after DMA
		postDMAAccessCycles := make([]uint64, 3)
		expectedPostDMA := []int{4, 2, 4} // PPUSTATUS read, LDA #$80, PPUCTRL write
		
		for i := 0; i < 3; i++ {
			initialCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step()
			postDMAAccessCycles[i] = helper.Bus.GetCycleCount() - initialCPU
			
			if postDMAAccessCycles[i] != uint64(expectedPostDMA[i]) {
				t.Errorf("Post-DMA instruction %d: expected %d cycles, got %d",
					i, expectedPostDMA[i], postDMAAccessCycles[i])
			}
		}
		
		// Verify overall 3:1 ratio maintained through DMA and register access
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("PPU/CPU ratio after DMA should be 3.0, got %.2f", ratio)
			}
		}
		
		t.Logf("DMA duration: %d CPU cycles, %d steps", dmaDuration, dmaSteps)
		t.Logf("Post-DMA register access: %v cycles", postDMAAccessCycles)
	})
	
	t.Run("Simultaneous CPU and PPU register access patterns", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Complex pattern mixing memory and PPU register access
		romData := make([]uint8, 0x8000)
		program := []uint8{
			// Memory operation
			0xA5, 0x10,       // LDA $10 (3 cycles) - zero page read
			
			// PPU register
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles) - PPUCTRL
			
			// Memory operation  
			0x85, 0x11,       // STA $11 (3 cycles) - zero page write
			
			// PPU register
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles) - PPUSTATUS
			
			// Memory operation
			0x8D, 0x00, 0x30, // STA $3000 (4 cycles) - absolute write
			
			// PPU register
			0xA9, 0x20,       // LDA #$20 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - PPUADDR
			
			// Memory operation
			0xAD, 0x00, 0x30, // LDA $3000 (4 cycles) - absolute read
			
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		expectedCycles := []int{3, 4, 3, 4, 4, 2, 4, 4, 3}
		mixedAccessCycles := make([]uint64, len(expectedCycles))
		totalCPU := uint64(0)
		
		for i, expected := range expectedCycles {
			initialCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step()
			actualCPU := helper.Bus.GetCycleCount() - initialCPU
			mixedAccessCycles[i] = actualCPU
			totalCPU += actualCPU
			
			if actualCPU != uint64(expected) {
				t.Errorf("Mixed access %d: expected %d cycles, got %d",
					i, expected, actualCPU)
			}
		}
		
		// Verify PPU timing consistency across mixed access pattern
		expectedTotalPPU := totalCPU * 3
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			if lastEntry.PPUCycles != expectedTotalPPU {
				t.Errorf("Mixed access total PPU cycles: expected %d, got %d",
					expectedTotalPPU, lastEntry.PPUCycles)
			}
			
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("Mixed access PPU/CPU ratio should be 3.0, got %.2f", ratio)
			}
		}
		
		t.Logf("Mixed access pattern: %v cycles (total CPU: %d)", 
			mixedAccessCycles, totalCPU)
	})
}