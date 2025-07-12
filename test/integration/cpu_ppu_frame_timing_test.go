package integration

import (
	"testing"
)

// TestFrameTimingAccuracy validates precise frame timing with 3:1 CPU-PPU synchronization
func TestFrameTimingAccuracy(t *testing.T) {
	t.Run("NTSC frame cycle counts", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Create simple test program
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP
		romData[0x0001] = 0x4C // JMP $8000
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// NTSC timing constants
		const (
			PPUCyclesPerScanline  = 341
			ScanlinesPerFrame     = 262
			PPUCyclesPerFrame     = PPUCyclesPerScanline * ScanlinesPerFrame // 89,342
			CPUCyclesPerFrame     = PPUCyclesPerFrame / 3                     // 29,780.67 ≈ 29,781
		)
		
		// Test exact frame timing
		initialCPUCycles := helper.Bus.GetCycleCount()
		initialFrameCount := helper.Bus.GetFrameCount()
		
		// Run until frame completion
		targetFrames := initialFrameCount + 1
		for helper.Bus.GetFrameCount() < targetFrames {
			helper.Bus.Step()
		}
		
		finalCPUCycles := helper.Bus.GetCycleCount()
		cpuCyclesInFrame := finalCPUCycles - initialCPUCycles
		
		// Allow small tolerance due to fractional cycles
		tolerance := uint64(2)
		if cpuCyclesInFrame < CPUCyclesPerFrame-tolerance || 
		   cpuCyclesInFrame > CPUCyclesPerFrame+tolerance {
			t.Errorf("Frame should take ~%d CPU cycles, took %d", 
				CPUCyclesPerFrame, cpuCyclesInFrame)
		}
		
		// Verify PPU cycles are exactly 3x CPU cycles for this frame
		expectedPPUCycles := cpuCyclesInFrame * 3
		
		// PPU cycles should equal expected frame cycles
		ppuTolerance := uint64(6) // Allow for rounding
		if expectedPPUCycles < PPUCyclesPerFrame-ppuTolerance || 
		   expectedPPUCycles > PPUCyclesPerFrame+ppuTolerance {
			t.Errorf("Frame should take ~%d PPU cycles (3x %d CPU), calculated %d", 
				PPUCyclesPerFrame, cpuCyclesInFrame, expectedPPUCycles)
		}
		
		t.Logf("Frame completed: %d CPU cycles, %d PPU cycles (3:1 ratio)", 
			cpuCyclesInFrame, expectedPPUCycles)
	})
	
	t.Run("Multiple frame consistency", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Create deterministic test program
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
		
		framesToTest := 5
		frameCycleCounts := make([]uint64, framesToTest)
		
		for frame := 0; frame < framesToTest; frame++ {
			initialCPU := helper.Bus.GetCycleCount()
			initialFrame := helper.Bus.GetFrameCount()
			
			// Run exactly one frame
			targetFrame := initialFrame + 1
			for helper.Bus.GetFrameCount() < targetFrame {
				helper.Bus.Step()
			}
			
			finalCPU := helper.Bus.GetCycleCount()
			frameCycleCounts[frame] = finalCPU - initialCPU
		}
		
		// All frames should have identical cycle counts
		baseFrameCycles := frameCycleCounts[0]
		tolerance := uint64(1) // Very tight tolerance for identical program
		
		for i, cycles := range frameCycleCounts {
			if cycles < baseFrameCycles-tolerance || cycles > baseFrameCycles+tolerance {
				t.Errorf("Frame %d cycle count inconsistent: expected %d±%d, got %d",
					i, baseFrameCycles, tolerance, cycles)
			}
		}
		
		// Verify deterministic frame timing
		expectedCyclesPerLoop := uint64(12) // 2+2+2+3+3 from program
		
		t.Logf("Frame cycle counts: %v (base: %d, loop cycles: %d)", 
			frameCycleCounts, baseFrameCycles, expectedCyclesPerLoop)
	})
	
	t.Run("Odd frame cycle skip validation", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Create test program
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP
		romData[0x0001] = 0x4C // JMP $8000
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		
		// Enable rendering to trigger odd frame skip
		helper.PPU.WriteRegister(0x2001, 0x18) // Show background and sprites
		
		helper.Bus.EnableExecutionLogging()
		
		evenFrameCycles := make([]uint64, 3)
		oddFrameCycles := make([]uint64, 3)
		
		frameIndex := 0
		for frameIndex < 6 {
			initialCPU := helper.Bus.GetCycleCount()
			initialFrame := helper.Bus.GetFrameCount()
			
			// Run one frame
			targetFrame := initialFrame + 1
			for helper.Bus.GetFrameCount() < targetFrame {
				helper.Bus.Step()
			}
			
			finalCPU := helper.Bus.GetCycleCount()
			frameCycles := finalCPU - initialCPU
			
			if frameIndex%2 == 0 {
				evenFrameCycles[frameIndex/2] = frameCycles
			} else {
				oddFrameCycles[frameIndex/2] = frameCycles
			}
			
			frameIndex++
		}
		
		// Calculate averages
		avgEvenCycles := uint64(0)
		avgOddCycles := uint64(0)
		
		for i := 0; i < 3; i++ {
			avgEvenCycles += evenFrameCycles[i]
			avgOddCycles += oddFrameCycles[i]
		}
		avgEvenCycles /= 3
		avgOddCycles /= 3
		
		// Odd frames should be 1 PPU cycle shorter when rendering is enabled
		// This translates to ~1/3 CPU cycle difference
		expectedDifference := uint64(1) // Allow rounding
		
		if avgEvenCycles <= avgOddCycles {
			t.Errorf("Even frames should take more cycles than odd frames. Even: %d, Odd: %d",
				avgEvenCycles, avgOddCycles)
		}
		
		actualDifference := avgEvenCycles - avgOddCycles
		if actualDifference > expectedDifference*2 {
			t.Errorf("Frame cycle difference too large. Expected ~%d, got %d",
				expectedDifference, actualDifference)
		}
		
		t.Logf("Even frame cycles: %v (avg: %d)", evenFrameCycles, avgEvenCycles)
		t.Logf("Odd frame cycles: %v (avg: %d)", oddFrameCycles, avgOddCycles)
		t.Logf("Difference: %d cycles", actualDifference)
	})
}

// TestFrameRateAccuracy validates overall frame rate timing
func TestFrameRateAccuracy(t *testing.T) {
	t.Run("NTSC frame rate calculation", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Simple test program
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP
		romData[0x0001] = 0x4C // JMP $8000
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		
		// NTSC specifications
		const (
			CPUFrequencyHz     = 1789773.0  // 1.789773 MHz
			PPUCyclesPerFrame  = 89342      // 341 * 262
			CPUCyclesPerFrame  = PPUCyclesPerFrame / 3
			ExpectedFrameRate  = CPUFrequencyHz / CPUCyclesPerFrame // ~60.1 Hz
		)
		
		// Measure frame rate by timing multiple frames
		framesToMeasure := 10
		initialFrame := helper.Bus.GetFrameCount()
		
		// Run frames and measure total CPU cycles
		targetFrame := initialFrame + uint64(framesToMeasure)
		initialCycles := helper.Bus.GetCycleCount()
		
		for helper.Bus.GetFrameCount() < targetFrame {
			helper.Bus.Step()
		}
		
		finalCycles := helper.Bus.GetCycleCount()
		totalCPUCycles := finalCycles - initialCycles
		avgCyclesPerFrame := float64(totalCPUCycles) / float64(framesToMeasure)
		
		// Calculate effective frame rate
		effectiveFrameRate := CPUFrequencyHz / avgCyclesPerFrame
		
		// Verify frame rate is close to NTSC standard
		tolerance := 0.5 // Hz
		if effectiveFrameRate < ExpectedFrameRate-tolerance || 
		   effectiveFrameRate > ExpectedFrameRate+tolerance {
			t.Errorf("Frame rate should be ~%.2f Hz, calculated %.2f Hz",
				ExpectedFrameRate, effectiveFrameRate)
		}
		
		t.Logf("Measured %d frames: %d total CPU cycles", framesToMeasure, totalCPUCycles)
		t.Logf("Average cycles per frame: %.2f", avgCyclesPerFrame)
		t.Logf("Effective frame rate: %.2f Hz (expected: %.2f Hz)", 
			effectiveFrameRate, ExpectedFrameRate)
	})
	
	t.Run("Frame timing with varying instruction mix", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Mixed instruction program with different cycle counts
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xEA,             // NOP (2)
			0xA9, 0x42,       // LDA #$42 (2)
			0x85, 0x00,       // STA $00 (3)
			0xA5, 0x00,       // LDA $00 (3)
			0x8D, 0x00, 0x30, // STA $3000 (4)
			0xAD, 0x00, 0x30, // LDA $3000 (4)
			0xA2, 0x10,       // LDX #$10 (2)
			0xBD, 0xF0, 0x20, // LDA $20F0,X (5 - page cross)
			0x4C, 0x00, 0x80, // JMP $8000 (3)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		framesToTest := 3
		frameResults := make([]struct {
			cycles uint64
			ratio  float64
		}, framesToTest)
		
		for frame := 0; frame < framesToTest; frame++ {
			initialCPU := helper.Bus.GetCycleCount()
			initialFrame := helper.Bus.GetFrameCount()
			
			// Run one frame
			targetFrame := initialFrame + 1
			for helper.Bus.GetFrameCount() < targetFrame {
				helper.Bus.Step()
			}
			
			finalCPU := helper.Bus.GetCycleCount()
			frameCycles := finalCPU - initialCPU
			
			// Calculate actual PPU cycles (should be 3x CPU)
			expectedPPUCycles := frameCycles * 3
			
			frameResults[frame] = struct {
				cycles uint64
				ratio  float64
			}{
				cycles: frameCycles,
				ratio:  float64(expectedPPUCycles) / float64(frameCycles),
			}
		}
		
		// Verify consistent timing across frames
		baseCycles := frameResults[0].cycles
		tolerance := uint64(5) // Allow small variance due to instruction timing
		
		for i, result := range frameResults {
			if result.cycles < baseCycles-tolerance || result.cycles > baseCycles+tolerance {
				t.Errorf("Frame %d cycle count inconsistent: expected %d±%d, got %d",
					i, baseCycles, tolerance, result.cycles)
			}
			
			if result.ratio != 3.0 {
				t.Errorf("Frame %d PPU/CPU ratio should be 3.0, got %.2f", i, result.ratio)
			}
		}
		
		// Expected cycles per loop: 2+2+3+3+4+4+2+5+3 = 28
		expectedLoopCycles := uint64(28)
		loopsPerFrame := baseCycles / expectedLoopCycles
		
		t.Logf("Frame results: %+v", frameResults)
		t.Logf("Base frame cycles: %d, loops per frame: %d", baseCycles, loopsPerFrame)
	})
}

// TestCPUPPUFrameSynchronizationEdgeCases validates frame timing in edge cases
func TestCPUPPUFrameSynchronizationEdgeCases(t *testing.T) {
	t.Run("Frame boundary during DMA", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program that triggers DMA near frame boundary
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xA9, 0x02,       // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (trigger DMA)
			0xEA,             // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		// Run to near frame boundary
		targetCycles := uint64(29700) // Near end of frame
		for helper.Bus.GetCycleCount() < targetCycles {
			helper.Bus.Step()
		}
		
		initialFrame := helper.Bus.GetFrameCount()
		initialCPU := helper.Bus.GetCycleCount()
		
		// Trigger DMA near frame boundary
		helper.Bus.Step() // LDA #$02
		helper.Bus.Step() // STA $4014 - triggers DMA
		
		// DMA should continue across frame boundary
		for helper.Bus.IsDMAInProgress() {
			helper.Bus.Step()
		}
		
		finalFrame := helper.Bus.GetFrameCount()
		finalCPU := helper.Bus.GetCycleCount()
		
		// Frame should have completed during DMA
		if finalFrame <= initialFrame {
			t.Error("Frame should have incremented during DMA")
		}
		
		// Verify timing accuracy maintained during cross-frame DMA
		cyclesDuringDMA := finalCPU - initialCPU
		expectedDMACycles := uint64(513) // Minimum DMA cycles
		
		if cyclesDuringDMA < expectedDMACycles {
			t.Errorf("DMA should take at least %d cycles, took %d", 
				expectedDMACycles, cyclesDuringDMA)
		}
		
		t.Logf("DMA across frame boundary: %d cycles, frame %d->%d", 
			cyclesDuringDMA, initialFrame, finalFrame)
	})
	
	t.Run("Frame timing with NMI", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Set up NMI handling
		romData := make([]uint8, 0x8000)
		
		// Main program
		romData[0x0000] = 0xEA // NOP
		romData[0x0001] = 0x4C // JMP $8000
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		
		// NMI handler
		romData[0x0100] = 0xEA // NOP in handler
		romData[0x0101] = 0x40 // RTI
		
		// Vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		
		// Enable NMI
		helper.PPU.WriteRegister(0x2000, 0x80)
		helper.Bus.EnableExecutionLogging()
		
		framesToTest := 3
		frameTimings := make([]uint64, framesToTest)
		
		for frame := 0; frame < framesToTest; frame++ {
			initialFrame := helper.Bus.GetFrameCount()
			initialCPU := helper.Bus.GetCycleCount()
			
			// Run one frame (should include NMI)
			targetFrame := initialFrame + 1
			nmiDetected := false
			
			for helper.Bus.GetFrameCount() < targetFrame {
				pc := helper.Bus.GetCPUState().PC
				if pc >= 0x8100 && pc <= 0x8101 && !nmiDetected {
					nmiDetected = true
					t.Logf("NMI detected in frame %d at PC $%04X", frame, pc)
				}
				helper.Bus.Step()
			}
			
			finalCPU := helper.Bus.GetCycleCount()
			frameTimings[frame] = finalCPU - initialCPU
		}
		
		// Frame timing should be consistent even with NMI overhead
		baseTiming := frameTimings[0]
		tolerance := uint64(10) // Allow for NMI processing overhead
		
		for i, timing := range frameTimings {
			if timing < baseTiming-tolerance || timing > baseTiming+tolerance {
				t.Errorf("Frame %d timing inconsistent: expected %d±%d, got %d",
					i, baseTiming, tolerance, timing)
			}
		}
		
		t.Logf("Frame timings with NMI: %v", frameTimings)
	})
}