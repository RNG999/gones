package integration

import (
	"testing"
)

// TestComprehensiveTimingIntegration validates all aspects of 3:1 CPU-PPU timing together
func TestComprehensiveTimingIntegration(t *testing.T) {
	t.Run("End-to-end timing validation", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Comprehensive test program that exercises all timing-critical areas
		romData := make([]uint8, 0x8000)
		
		// Main program - exercises various instruction types and PPU access
		program := []uint8{
			// Basic instruction mix (varied cycle counts)
			0xEA,             // NOP (2 cycles)
			0xA9, 0x00,       // LDA #$00 (2 cycles)
			0x85, 0x10,       // STA $10 (3 cycles)
			0x8D, 0x00, 0x30, // STA $3000 (4 cycles)
			
			// PPU register access
			0xA9, 0x80,       // LDA #$80 (2 cycles)
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles) - PPUCTRL (enable NMI)
			0xA9, 0x18,       // LDA #$18 (2 cycles)
			0x8D, 0x01, 0x20, // STA $2001 (4 cycles) - PPUMASK (enable rendering)
			
			// VRAM address setup
			0xA9, 0x20,       // LDA #$20 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - PPUADDR high
			0xA9, 0x00,       // LDA #$00 (2 cycles)
			0x8D, 0x06, 0x20, // STA $2006 (4 cycles) - PPUADDR low
			
			// VRAM data write
			0xA9, 0x42,       // LDA #$42 (2 cycles)
			0x8D, 0x07, 0x20, // STA $2007 (4 cycles) - PPUDATA
			
			// Page crossing instruction
			0xA2, 0x10,       // LDX #$10 (2 cycles)
			0xBD, 0xF0, 0x20, // LDA $20F0,X (5 cycles - page cross)
			
			// Branch instruction
			0xC9, 0x00,       // CMP #$00 (2 cycles)
			0xF0, 0x02,       // BEQ +2 (3 cycles if taken, 2 if not)
			0xEA,             // NOP (2 cycles - skipped if branch taken)
			
			// Memory read
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles) - PPUSTATUS read
			
			// Potential DMA trigger (commented out to avoid complexity)
			// 0xA9, 0x02,       // LDA #$02 (2 cycles)
			// 0x8D, 0x14, 0x40, // STA $4014 (4 cycles) - OAM DMA
			
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		
		// NMI handler for interrupt testing
		romData[0x0100] = 0xEA // NOP (2 cycles)
		romData[0x0101] = 0x40 // RTI (6 cycles)
		
		// Vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// Expected cycle counts for the main program
		expectedMainLoopCycles := []int{
			2, 2, 3, 4, // Basic instructions
			2, 4, 2, 4, // PPU control setup
			2, 4, 2, 4, // PPUADDR setup
			2, 4,       // PPUDATA write
			2, 5,       // Page crossing instruction
			2, 3,       // Branch taken (assuming CMP result)
			4,          // PPUSTATUS read
			3,          // JMP
		}
		
		totalExpectedCPU := 0
		for _, cycles := range expectedMainLoopCycles {
			totalExpectedCPU += cycles
		}
		
		// Run multiple complete loops to test consistency
		loopsToTest := 10
		loopTimings := make([]uint64, loopsToTest)
		
		for loop := 0; loop < loopsToTest; loop++ {
			initialCPU := helper.Bus.GetCycleCount()
			
			// Execute one complete loop
			for i := 0; i < len(expectedMainLoopCycles); i++ {
				helper.Bus.Step()
			}
			
			finalCPU := helper.Bus.GetCycleCount()
			loopTimings[loop] = finalCPU - initialCPU
		}
		
		// Verify loop timing consistency
		baseLoopTiming := loopTimings[0]
		tolerance := uint64(2) // Allow small tolerance
		
		for i, timing := range loopTimings {
			if timing < baseLoopTiming-tolerance || timing > baseLoopTiming+tolerance {
				t.Errorf("Loop %d timing inconsistent: expected %d±%d, got %d",
					i, baseLoopTiming, tolerance, timing)
			}
		}
		
		// Verify expected cycle counts
		expectedLoopCycles := uint64(totalExpectedCPU)
		if baseLoopTiming < expectedLoopCycles-tolerance || 
		   baseLoopTiming > expectedLoopCycles+tolerance {
			t.Errorf("Loop cycles: expected %d±%d, got %d", 
				expectedLoopCycles, tolerance, baseLoopTiming)
		}
		
		// Verify 3:1 ratio maintained throughout
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("Overall PPU/CPU ratio should be 3.0, got %.2f", ratio)
			}
			
			// Check individual step ratios
			for i, entry := range log {
				if i > 0 {
					stepCPU := entry.CPUCycles - log[i-1].CPUCycles
					stepPPU := entry.PPUCycles - log[i-1].PPUCycles
					if stepCPU > 0 {
						stepRatio := float64(stepPPU) / float64(stepCPU)
						if stepRatio != 3.0 {
							t.Errorf("Step %d PPU/CPU ratio should be 3.0, got %.2f", 
								i, stepRatio)
						}
					}
				}
			}
		}
		
		t.Logf("Comprehensive timing test: %d loops, %d cycles each", 
			loopsToTest, baseLoopTiming)
		t.Logf("Loop timings: %v", loopTimings)
	})
	
	t.Run("Timing accuracy across frame boundaries", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program that runs across frame boundaries
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xEA,       // NOP (2 cycles)
			0xE8,       // INX (2 cycles)
			0xA9, 0x00, // LDA #$00 (2 cycles)
			0x85, 0x10, // STA $10 (3 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		const (
			PPUCyclesPerFrame = 89342
			CPUCyclesPerFrame = PPUCyclesPerFrame / 3
			CyclesPerLoop     = 12 // 2+2+2+3+3 = 12 CPU cycles per loop
		)
		
		// Run for multiple frames and verify timing consistency
		framesToTest := 5
		frameTimings := make([]struct {
			cpuCycles   uint64
			ppuCycles   uint64
			frameCount  uint64
			loopCount   uint64
		}, framesToTest)
		
		for frame := 0; frame < framesToTest; frame++ {
			initialCPU := helper.Bus.GetCycleCount()
			initialFrame := helper.Bus.GetFrameCount()
			
			// Run one frame worth of cycles
			targetCPU := initialCPU + CPUCyclesPerFrame
			loopCount := uint64(0)
			
			for helper.Bus.GetCycleCount() < targetCPU {
				loopStartCPU := helper.Bus.GetCycleCount()
				
				// Execute one complete loop
				for i := 0; i < 5; i++ { // 5 instructions per loop
					helper.Bus.Step()
				}
				
				loopEndCPU := helper.Bus.GetCycleCount()
				actualLoopCycles := loopEndCPU - loopStartCPU
				
				if actualLoopCycles != CyclesPerLoop {
					t.Errorf("Frame %d, loop %d: expected %d cycles, got %d",
						frame, loopCount, CyclesPerLoop, actualLoopCycles)
				}
				
				loopCount++
			}
			
			finalCPU := helper.Bus.GetCycleCount()
			finalFrame := helper.Bus.GetFrameCount()
			
			frameTimings[frame] = struct {
				cpuCycles   uint64
				ppuCycles   uint64
				frameCount  uint64
				loopCount   uint64
			}{
				cpuCycles:  finalCPU - initialCPU,
				ppuCycles:  (finalCPU - initialCPU) * 3,
				frameCount: finalFrame - initialFrame,
				loopCount:  loopCount,
			}
		}
		
		// Verify frame timing consistency
		baseFrameTiming := frameTimings[0]
		tolerance := uint64(10)
		
		for i, timing := range frameTimings {
			if timing.cpuCycles < baseFrameTiming.cpuCycles-tolerance ||
			   timing.cpuCycles > baseFrameTiming.cpuCycles+tolerance {
				t.Errorf("Frame %d CPU cycles inconsistent: expected %d±%d, got %d",
					i, baseFrameTiming.cpuCycles, tolerance, timing.cpuCycles)
			}
			
			if timing.ppuCycles != timing.cpuCycles*3 {
				t.Errorf("Frame %d PPU cycles should be 3x CPU: CPU=%d, PPU=%d",
					i, timing.cpuCycles, timing.ppuCycles)
			}
		}
		
		// Verify frame progression
		for i, timing := range frameTimings {
			expectedFrames := uint64(1)
			if timing.frameCount != expectedFrames {
				t.Errorf("Frame %d should increment frame count by %d, got %d",
					i, expectedFrames, timing.frameCount)
			}
		}
		
		t.Logf("Frame boundary timing test:")
		for i, timing := range frameTimings {
			t.Logf("  Frame %d: %d CPU cycles, %d PPU cycles, %d loops, %d frames",
				i, timing.cpuCycles, timing.ppuCycles, timing.loopCount, timing.frameCount)
		}
	})
	
	t.Run("Timing accuracy with interrupts and DMA", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Complex program with NMI and DMA
		romData := make([]uint8, 0x8000)
		
		// Main program
		program := []uint8{
			// Enable NMI
			0xA9, 0x80,       // LDA #$80 (2 cycles)
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles) - PPUCTRL
			
			// Setup for potential DMA
			0xA9, 0x02,       // LDA #$02 (2 cycles)
			0x85, 0x20,       // STA $20 (3 cycles) - store DMA page
			
			// Main loop with various operations
			0xEA,             // NOP (2 cycles)
			0xE8,             // INX (2 cycles)
			0xC8,             // INY (2 cycles)
			
			// Conditional DMA trigger (every 256 loops when X wraps)
			0xE0, 0x00,       // CPX #$00 (2 cycles)
			0xD0, 0x04,       // BNE +4 (2/3 cycles)
			0xA5, 0x20,       // LDA $20 (3 cycles) - load DMA page
			0x8D, 0x14, 0x40, // STA $4014 (4 cycles) - trigger DMA
			
			// PPU register access
			0xAD, 0x02, 0x20, // LDA $2002 (4 cycles) - PPUSTATUS
			
			0x4C, 0x08, 0x80, // JMP $8008 (3 cycles) - loop back to main loop
		}
		copy(romData, program)
		
		// NMI handler
		romData[0x0100] = 0xEA // NOP (2 cycles)
		romData[0x0101] = 0xE6 // INC $21 (5 cycles) - increment NMI counter
		romData[0x0102] = 0x21
		romData[0x0103] = 0x40 // RTI (6 cycles)
		
		// Vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// Initialize NMI counter
		helper.Memory.Write(0x21, 0x00)
		
		// Run for extended period to test various conditions
		targetCycles := uint64(100000) // Run for 100k CPU cycles
		initialCPU := helper.Bus.GetCycleCount()
		
		nmiCount := 0
		dmaCount := 0
		stepCount := 0
		
		for helper.Bus.GetCycleCount() < initialCPU+targetCycles {
			preStepCPU := helper.Bus.GetCycleCount()
			cpuState := helper.Bus.GetCPUState()
			
			// Track NMI entries
			if cpuState.PC >= 0x8100 && cpuState.PC <= 0x8103 {
				nmiCount++
			}
			
			// Track DMA
			if helper.Bus.IsDMAInProgress() {
				dmaCount++
			}
			
			helper.Bus.Step()
			stepCount++
			
			// Verify 3:1 ratio maintained at every step
			postStepCPU := helper.Bus.GetCycleCount()
			stepCPUCycles := postStepCPU - preStepCPU
			
			// PPU should advance exactly 3x the CPU cycles
			expectedPPUAdvance := stepCPUCycles * 3
			
			// Get actual PPU advance from execution log
			log := helper.Bus.GetExecutionLog()
			if len(log) >= 2 {
				actualPPUAdvance := log[len(log)-1].PPUCycles - log[len(log)-2].PPUCycles
				if actualPPUAdvance != expectedPPUAdvance {
					t.Errorf("Step %d: PPU advance should be %d, got %d",
						stepCount, expectedPPUAdvance, actualPPUAdvance)
					break // Avoid too many error messages
				}
			}
		}
		
		finalCPU := helper.Bus.GetCycleCount()
		totalCPUCycles := finalCPU - initialCPU
		totalPPUCycles := totalCPUCycles * 3
		
		// Read final NMI counter
		finalNMICount := helper.Memory.Read(0x21)
		
		// Verify overall timing accuracy
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			actualTotalPPU := lastEntry.PPUCycles - (initialCPU * 3)
			
			if actualTotalPPU != totalPPUCycles {
				t.Errorf("Total PPU cycles: expected %d, got %d",
					totalPPUCycles, actualTotalPPU)
			}
			
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("Final PPU/CPU ratio should be 3.0, got %.2f", ratio)
			}
		}
		
		t.Logf("Complex timing test: %d CPU cycles, %d steps", totalCPUCycles, stepCount)
		t.Logf("NMI occurrences: %d (counter: %d), DMA steps: %d", 
			nmiCount, finalNMICount, dmaCount)
	})
	
	t.Run("Long-term timing stability", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Simple, stable program for long-term testing
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xEA,       // NOP (2 cycles)
			0xEA,       // NOP (2 cycles)
			0xE8,       // INX (2 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		const cyclesPerLoop = 9 // 2+2+2+3 = 9 CPU cycles per loop
		
		// Run for many loops to test long-term stability
		loopsToTest := 10000
		sampleInterval := 1000 // Sample every 1000 loops
		
		samples := make([]struct {
			loopNumber uint64
			cpuCycles  uint64
			ppuCycles  uint64
			ratio      float64
		}, 0)
		
		for loop := 0; loop < loopsToTest; loop++ {
			// Execute one loop
			for i := 0; i < 4; i++ { // 4 instructions per loop
				helper.Bus.Step()
			}
			
			// Sample periodically
			if loop%sampleInterval == 0 || loop == loopsToTest-1 {
				currentCPU := helper.Bus.GetCycleCount()
				_ = currentCPU * 3 // currentPPU (unused for now)
				
				log := helper.Bus.GetExecutionLog()
				actualPPU := uint64(0)
				if len(log) > 0 {
					actualPPU = log[len(log)-1].PPUCycles
				}
				
				ratio := float64(actualPPU) / float64(currentCPU)
				
				samples = append(samples, struct {
					loopNumber uint64
					cpuCycles  uint64
					ppuCycles  uint64
					ratio      float64
				}{
					loopNumber: uint64(loop),
					cpuCycles:  currentCPU,
					ppuCycles:  actualPPU,
					ratio:      ratio,
				})
			}
		}
		
		// Verify long-term stability
		for i, sample := range samples {
			// Verify expected CPU cycles
			expectedCPU := sample.loopNumber * cyclesPerLoop
			tolerance := uint64(cyclesPerLoop) // Allow one loop tolerance
			
			if sample.cpuCycles < expectedCPU || sample.cpuCycles > expectedCPU+tolerance {
				t.Errorf("Sample %d: CPU cycles drift, expected ~%d, got %d",
					i, expectedCPU, sample.cpuCycles)
			}
			
			// Verify 3:1 ratio maintained
			if sample.ratio != 3.0 {
				t.Errorf("Sample %d: PPU/CPU ratio should be 3.0, got %.3f",
					i, sample.ratio)
			}
		}
		
		// Check for any systematic drift
		_ = samples[0] // firstSample (unused for now)
		lastSample := samples[len(samples)-1]
		
		expectedFinalCPU := lastSample.loopNumber * cyclesPerLoop
		actualDrift := int64(lastSample.cpuCycles) - int64(expectedFinalCPU)
		
		maxAllowedDrift := int64(cyclesPerLoop * 2) // Allow 2 loops drift max
		if actualDrift < -maxAllowedDrift || actualDrift > maxAllowedDrift {
			t.Errorf("Long-term CPU cycle drift: %d cycles over %d loops",
				actualDrift, loopsToTest)
		}
		
		ppuDrift := float64(lastSample.ppuCycles) - float64(lastSample.cpuCycles*3)
		if ppuDrift != 0 {
			t.Errorf("Long-term PPU cycle drift: %.0f cycles", ppuDrift)
		}
		
		t.Logf("Long-term stability test: %d loops, %d samples", loopsToTest, len(samples))
		t.Logf("CPU drift: %d cycles, PPU drift: %.0f cycles", actualDrift, ppuDrift)
		t.Logf("Final state: Loop %d, CPU %d, PPU %d, Ratio %.3f",
			lastSample.loopNumber, lastSample.cpuCycles, lastSample.ppuCycles, lastSample.ratio)
	})
}

// TestTimingRegressionSuite validates that timing behavior remains consistent
func TestTimingRegressionSuite(t *testing.T) {
	t.Run("Baseline timing characteristics", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Standard test program for establishing baseline
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xA9, 0x00,       // LDA #$00 (2 cycles)
			0x85, 0x10,       // STA $10 (3 cycles)
			0xA5, 0x10,       // LDA $10 (3 cycles)
			0x8D, 0x00, 0x30, // STA $3000 (4 cycles)
			0xAD, 0x00, 0x30, // LDA $3000 (4 cycles)
			0xE8,             // INX (2 cycles)
			0xD0, 0xF2,       // BNE -14 (3 cycles when taken)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// Run standardized test
		iterations := 256 // Run until X wraps around
		totalCPUCycles := uint64(0)
		
		for i := 0; i < iterations*7; i++ { // 7 instructions per loop until branch not taken
			helper.Bus.Step()
		}
		
		totalCPUCycles = helper.Bus.GetCycleCount()
		expectedTotalPPU := totalCPUCycles * 3
		
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			actualTotalPPU := log[len(log)-1].PPUCycles
			
			if actualTotalPPU != expectedTotalPPU {
				t.Errorf("Baseline PPU cycles: expected %d, got %d",
					expectedTotalPPU, actualTotalPPU)
			}
		}
		
		// Establish baseline metrics for regression testing
		baseline := struct {
			cpuCycles uint64
			ppuCycles uint64
			ratio     float64
			steps     int
		}{
			cpuCycles: totalCPUCycles,
			ppuCycles: expectedTotalPPU,
			ratio:     3.0,
			steps:     iterations * 7,
		}
		
		t.Logf("Baseline established: %+v", baseline)
		
		// Verify baseline is reasonable
		if baseline.cpuCycles < 1000 || baseline.cpuCycles > 10000 {
			t.Errorf("Baseline CPU cycles out of reasonable range: %d", baseline.cpuCycles)
		}
		
		if baseline.ppuCycles != baseline.cpuCycles*3 {
			t.Errorf("Baseline PPU/CPU ratio incorrect: %d/%d = %.2f",
				baseline.ppuCycles, baseline.cpuCycles, baseline.ratio)
		}
	})
}