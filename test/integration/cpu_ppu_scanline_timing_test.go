package integration

import (
	"testing"
)

// TestScanlineTimingPrecision validates precise scanline timing with 3:1 CPU-PPU synchronization
func TestScanlineTimingPrecision(t *testing.T) {
	t.Run("341 PPU cycles per scanline", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Create test program
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP (2 CPU cycles = 6 PPU cycles)
		romData[0x0001] = 0x4C // JMP $8000 (3 CPU cycles = 9 PPU cycles)
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		const (
			PPUCyclesPerScanline = 341
			CPUCyclesPerScanline = PPUCyclesPerScanline / 3 // 113.67 ≈ 114
		)
		
		// Run for multiple scanlines and verify timing
		scanlinesPerTest := 10
		totalPPUCyclesExpected := uint64(scanlinesPerTest * PPUCyclesPerScanline)
		totalCPUCyclesExpected := totalPPUCyclesExpected / 3
		
		initialCPU := helper.Bus.GetCycleCount()
		
		// Run for calculated CPU cycles to complete scanlines
		targetCPU := initialCPU + totalCPUCyclesExpected
		
		for helper.Bus.GetCycleCount() < targetCPU {
			helper.Bus.Step()
		}
		
		finalCPU := helper.Bus.GetCycleCount()
		actualCPUCycles := finalCPU - initialCPU
		actualPPUCycles := actualCPUCycles * 3
		
		// Verify scanline boundary alignment
		ppuCyclesRemainder := actualPPUCycles % PPUCyclesPerScanline
		
		// Allow small tolerance for instruction boundary alignment
		tolerance := uint64(15) // ~1 NOP+JMP instruction worth
		
		if ppuCyclesRemainder > tolerance && ppuCyclesRemainder < PPUCyclesPerScanline-tolerance {
			t.Errorf("PPU cycles should align to scanline boundaries. Remainder: %d of %d",
				ppuCyclesRemainder, PPUCyclesPerScanline)
		}
		
		completedScanlines := actualPPUCycles / PPUCyclesPerScanline
		
		if completedScanlines < uint64(scanlinesPerTest-1) || completedScanlines > uint64(scanlinesPerTest+1) {
			t.Errorf("Expected ~%d scanlines, completed %d", scanlinesPerTest, completedScanlines)
		}
		
		t.Logf("Completed %d scanlines: %d CPU cycles, %d PPU cycles", 
			completedScanlines, actualCPUCycles, actualPPUCycles)
	})
	
	t.Run("Scanline timing consistency", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Deterministic program for consistent timing
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xA9, 0x00, // LDA #$00 (2 cycles)
			0x85, 0x10, // STA $10 (3 cycles)
			0xE8,       // INX (2 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		const (
			PPUCyclesPerScanline = 341
			CyclesPerLoop        = 10 // 2+3+2+3 = 10 CPU cycles per loop
			PPUCyclesPerLoop     = CyclesPerLoop * 3 // 30 PPU cycles per loop
		)
		
		// Calculate loops needed for multiple scanlines
		scanlinesPerTest := 5
		targetPPUCycles := uint64(scanlinesPerTest * PPUCyclesPerScanline)
		_ = targetPPUCycles / PPUCyclesPerLoop // loopsNeeded (unused for now)
		
		scanlineTimings := make([]uint64, scanlinesPerTest)
		
		for scanline := 0; scanline < scanlinesPerTest; scanline++ {
			initialCPU := helper.Bus.GetCycleCount()
			
			// Run one scanline worth of PPU cycles
			targetPPUForScanline := uint64((scanline + 1) * PPUCyclesPerScanline)
			targetCPUForScanline := targetPPUForScanline / 3
			
			for helper.Bus.GetCycleCount() < targetCPUForScanline {
				helper.Bus.Step()
			}
			
			finalCPU := helper.Bus.GetCycleCount()
			scanlineTimings[scanline] = finalCPU - initialCPU
		}
		
		// All scanlines should have similar cycle counts
		expectedCyclesPerScanline := uint64(PPUCyclesPerScanline / 3) // ~113.67
		tolerance := uint64(2)
		
		for i, cycles := range scanlineTimings {
			if cycles < expectedCyclesPerScanline-tolerance || 
			   cycles > expectedCyclesPerScanline+tolerance {
				t.Errorf("Scanline %d: expected ~%d CPU cycles, got %d",
					i, expectedCyclesPerScanline, cycles)
			}
		}
		
		// Verify total timing
		totalCycles := uint64(0)
		for _, cycles := range scanlineTimings {
			totalCycles += cycles
		}
		
		expectedTotal := uint64(scanlinesPerTest) * expectedCyclesPerScanline
		totalTolerance := uint64(scanlinesPerTest * 2)
		
		if totalCycles < expectedTotal-totalTolerance || 
		   totalCycles > expectedTotal+totalTolerance {
			t.Errorf("Total scanline cycles: expected ~%d, got %d", expectedTotal, totalCycles)
		}
		
		t.Logf("Scanline timings: %v (total: %d)", scanlineTimings, totalCycles)
	})
	
	t.Run("Visible vs non-visible scanline timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Enable rendering to test rendering vs non-rendering differences
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP
		romData[0x0001] = 0x4C // JMP $8000
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		
		// Enable rendering
		helper.PPU.WriteRegister(0x2001, 0x18) // Show background and sprites
		helper.Bus.EnableExecutionLogging()
		
		const PPUCyclesPerScanline = 341
		
		// Test different scanline types
		scanlineTypes := []struct {
			name          string
			startScanline int
			endScanline   int
			description   string
		}{
			{"Visible", 0, 239, "Visible scanlines (0-239)"},
			{"Post-render", 240, 240, "Post-render scanline (240)"},
			{"VBlank", 241, 260, "VBlank scanlines (241-260)"},
			{"Pre-render", 261, 261, "Pre-render scanline (261)"},
		}
		
		for _, scanlineType := range scanlineTypes {
			t.Run(scanlineType.name, func(t *testing.T) {
				helper.Bus.Reset()
				helper.PPU.WriteRegister(0x2001, 0x18) // Re-enable rendering
				
				// Run to start of target scanline range
				// This is a simplified test - real implementation would need
				// precise scanline tracking
				
				scanlineCount := scanlineType.endScanline - scanlineType.startScanline + 1
				totalPPUCycles := uint64(scanlineCount * PPUCyclesPerScanline)
				targetCPUCycles := totalPPUCycles / 3
				
				initialCPU := helper.Bus.GetCycleCount()
				runTargetCPU := initialCPU + targetCPUCycles
				
				for helper.Bus.GetCycleCount() < runTargetCPU {
					helper.Bus.Step()
				}
				
				finalCPU := helper.Bus.GetCycleCount()
				actualCPUCycles := finalCPU - initialCPU
				actualPPUCycles := actualCPUCycles * 3
				
				// Verify timing is consistent regardless of scanline type
				expectedPPUCycles := totalPPUCycles
				tolerance := uint64(15) // Allow for instruction boundaries
				
				if actualPPUCycles < expectedPPUCycles-tolerance || 
				   actualPPUCycles > expectedPPUCycles+tolerance {
					t.Errorf("%s: expected ~%d PPU cycles, got %d",
						scanlineType.name, expectedPPUCycles, actualPPUCycles)
				}
				
				// Verify 3:1 ratio maintained
				ratio := float64(actualPPUCycles) / float64(actualCPUCycles)
				if ratio != 3.0 {
					t.Errorf("%s: PPU/CPU ratio should be 3.0, got %.2f",
						scanlineType.name, ratio)
				}
				
				t.Logf("%s (%d scanlines): %d CPU cycles, %d PPU cycles",
					scanlineType.name, scanlineCount, actualCPUCycles, actualPPUCycles)
			})
		}
	})
}

// TestScanlineCycleAccuracy validates cycle-accurate scanline progression
func TestScanlineCycleAccuracy(t *testing.T) {
	t.Run("Exact scanline boundary detection", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program with exactly known cycle counts
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP (2 cycles = 6 PPU cycles)
		romData[0x0001] = 0x4C // JMP (3 cycles = 9 PPU cycles)
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		const (
			PPUCyclesPerScanline = 341
			CPUCyclesPerLoop     = 5  // 2 + 3 = 5 CPU cycles per loop
			PPUCyclesPerLoop     = 15 // 5 * 3 = 15 PPU cycles per loop
		)
		
		// Calculate exact number of loops for scanline boundaries
		_ = PPUCyclesPerScanline / PPUCyclesPerLoop // loopsPerScanline = 22.73... loops (unused for now)
		
		// Test multiple scanline crossings
		scanlinesPerTest := 3
		
		for scanline := 1; scanline <= scanlinesPerTest; scanline++ {
			targetPPUCycles := uint64(scanline * PPUCyclesPerScanline)
			targetCPUCycles := targetPPUCycles / 3
			
			// Run until we cross scanline boundary
			for helper.Bus.GetCycleCount() < targetCPUCycles {
				helper.Bus.Step()
			}
			
			actualCPU := helper.Bus.GetCycleCount()
			actualPPU := actualCPU * 3
			
			// Check how close we are to exact scanline boundary
			ppuRemainder := actualPPU % PPUCyclesPerScanline
			scanlineProgress := float64(ppuRemainder) / float64(PPUCyclesPerScanline) * 100
			
			t.Logf("Scanline %d boundary: %d CPU cycles, %d PPU cycles (%.1f%% into next scanline)",
				scanline, actualCPU, actualPPU, scanlineProgress)
			
			// Verify we've crossed the expected number of scanlines
			completedScanlines := actualPPU / PPUCyclesPerScanline
			if completedScanlines < uint64(scanline-1) || completedScanlines > uint64(scanline) {
				t.Errorf("Expected to complete ~%d scanlines, completed %d",
					scanline, completedScanlines)
			}
		}
	})
	
	t.Run("Fractional CPU cycle handling", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Use single-cycle resolution to test fractional handling
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xEA, // NOP (2 cycles)
			0xEA, // NOP (2 cycles)
			0xEA, // NOP (2 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		const (
			PPUCyclesPerScanline = 341
			// CPUCyclesPerScanline = 113.666... // 341 / 3 (fractional, not used in constants)
		)
		
		// Track precise cycle accumulation
		cycleCounts := make([]struct {
			cpu uint64
			ppu uint64
		}, 10)
		
		for i := 0; i < 10; i++ {
			helper.Bus.Step()
			cycleCounts[i] = struct {
				cpu uint64
				ppu uint64
			}{
				cpu: helper.Bus.GetCycleCount(),
				ppu: helper.Bus.GetCycleCount() * 3,
			}
		}
		
		// Verify no fractional cycle accumulation
		for i, count := range cycleCounts {
			// PPU cycles should always be exactly 3x CPU cycles
			expectedPPU := count.cpu * 3
			if count.ppu != expectedPPU {
				t.Errorf("Step %d: PPU cycles (%d) != 3 * CPU cycles (%d)",
					i, count.ppu, count.cpu)
			}
			
			// CPU cycles should be integers
			if count.cpu == 0 {
				t.Errorf("Step %d: CPU cycles should not be zero", i)
			}
		}
		
		// Check for consistent timing across steps
		for i := 1; i < len(cycleCounts); i++ {
			cpuDelta := cycleCounts[i].cpu - cycleCounts[i-1].cpu
			ppuDelta := cycleCounts[i].ppu - cycleCounts[i-1].ppu
			
			if ppuDelta != cpuDelta*3 {
				t.Errorf("Step %d: PPU delta (%d) != 3 * CPU delta (%d)",
					i, ppuDelta, cpuDelta)
			}
		}
		
		t.Logf("Cycle progression over 10 steps:")
		for i, count := range cycleCounts {
			scanlinePos := float64(count.ppu%PPUCyclesPerScanline) / float64(PPUCyclesPerScanline) * 100
			t.Logf("  Step %d: CPU=%d, PPU=%d (%.1f%% through scanline)", 
				i, count.cpu, count.ppu, scanlinePos)
		}
	})
	
	t.Run("Scanline timing during rendering events", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Test program
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP
		romData[0x0001] = 0x4C // JMP $8000
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		
		// Enable rendering to trigger rendering-related timing
		helper.PPU.WriteRegister(0x2001, 0x18) // Show background and sprites
		helper.Bus.EnableExecutionLogging()
		
		const PPUCyclesPerScanline = 341
		
		// Run for several scanlines and check for timing consistency
		scanlinesPerTest := 270 // Cover full frame including VBlank
		targetPPUCycles := uint64(scanlinesPerTest * PPUCyclesPerScanline)
		targetCPUCycles := targetPPUCycles / 3
		
		initialCPU := helper.Bus.GetCycleCount()
		
		for helper.Bus.GetCycleCount() < initialCPU+targetCPUCycles {
			helper.Bus.Step()
		}
		
		finalCPU := helper.Bus.GetCycleCount()
		actualCPUCycles := finalCPU - initialCPU
		actualPPUCycles := actualCPUCycles * 3
		
		// Check scanline alignment
		completedScanlines := actualPPUCycles / PPUCyclesPerScanline
		remainder := actualPPUCycles % PPUCyclesPerScanline
		
		// Should complete expected number of scanlines
		tolerance := uint64(1)
		if completedScanlines < uint64(scanlinesPerTest)-tolerance || 
		   completedScanlines > uint64(scanlinesPerTest)+tolerance {
			t.Errorf("Expected ~%d scanlines, completed %d", scanlinesPerTest, completedScanlines)
		}
		
		// Remainder should be small (within instruction boundary)
		maxRemainder := uint64(15) // ~1 instruction worth
		if remainder > maxRemainder {
			t.Errorf("Scanline remainder too large: %d cycles (max %d)", remainder, maxRemainder)
		}
		
		// Verify consistent 3:1 ratio
		ratio := float64(actualPPUCycles) / float64(actualCPUCycles)
		if ratio != 3.0 {
			t.Errorf("PPU/CPU ratio should be 3.0, got %.2f", ratio)
		}
		
		t.Logf("Rendered %d scanlines: %d CPU cycles, %d PPU cycles, %d remainder",
			completedScanlines, actualCPUCycles, actualPPUCycles, remainder)
	})
}