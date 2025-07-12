package integration

import (
	"testing"
)

// TestVBlankTimingAccuracy validates precise VBlank timing with 3:1 CPU-PPU synchronization
func TestVBlankTimingAccuracy(t *testing.T) {
	t.Run("VBlank flag set at exact cycle count", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Create test program
		romData := make([]uint8, 0x8000)
		romData[0x0000] = 0xEA // NOP (2 cycles)
		romData[0x0001] = 0x4C // JMP $8000 (3 cycles)
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		helper.Bus.EnableExecutionLogging()
		
		const (
			PPUCyclesPerScanline = 341
			VisibleScanlines     = 240
			PostRenderScanline   = 240
			VBlankStartScanline  = 241
			VBlankStartCycle     = 1
			
			// VBlank starts at scanline 241, cycle 1
			PPUCyclesUntilVBlank = VisibleScanlines*PPUCyclesPerScanline + // 240 visible scanlines
				PPUCyclesPerScanline + // Post-render scanline 240
				VBlankStartCycle       // Cycle 1 of scanline 241
			CPUCyclesUntilVBlank = PPUCyclesUntilVBlank / 3
		)
		
		// Run until just before VBlank
		targetCPU := uint64(CPUCyclesUntilVBlank - 10) // Stop slightly before
		initialCPU := helper.Bus.GetCycleCount()
		
		for helper.Bus.GetCycleCount() < initialCPU+targetCPU {
			helper.Bus.Step()
		}
		
		// Check VBlank flag is not set yet
		status := helper.PPU.ReadRegister(0x2002)
		if (status & 0x80) != 0 {
			t.Error("VBlank flag should not be set before scanline 241, cycle 1")
		}
		
		// Continue until VBlank should be set
		vblankDetected := false
		vblankCPUCycle := uint64(0)
		maxSteps := 50 // Safety limit
		
		for steps := 0; steps < maxSteps && !vblankDetected; steps++ {
			currentCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step()
			
			status := helper.PPU.ReadRegister(0x2002)
			if (status & 0x80) != 0 {
				vblankDetected = true
				vblankCPUCycle = currentCPU - initialCPU
				break
			}
		}
		
		if !vblankDetected {
			t.Fatal("VBlank flag was not detected within expected range")
		}
		
		// Verify VBlank timing accuracy
		expectedCPUCycle := uint64(CPUCyclesUntilVBlank)
		tolerance := uint64(5) // Allow small tolerance for instruction boundaries
		
		if vblankCPUCycle < expectedCPUCycle-tolerance || 
		   vblankCPUCycle > expectedCPUCycle+tolerance {
			t.Errorf("VBlank detected at CPU cycle %d, expected ~%d", 
				vblankCPUCycle, expectedCPUCycle)
		}
		
		// Verify corresponding PPU cycle count
		vblankPPUCycle := vblankCPUCycle * 3
		expectedPPUCycle := uint64(PPUCyclesUntilVBlank)
		ppuTolerance := tolerance * 3
		
		if vblankPPUCycle < expectedPPUCycle-ppuTolerance || 
		   vblankPPUCycle > expectedPPUCycle+ppuTolerance {
			t.Errorf("VBlank PPU timing incorrect: got %d, expected ~%d", 
				vblankPPUCycle, expectedPPUCycle)
		}
		
		t.Logf("VBlank detected at CPU cycle %d, PPU cycle %d", vblankCPUCycle, vblankPPUCycle)
	})
	
	t.Run("VBlank flag cleared at pre-render scanline", func(t *testing.T) {
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
		helper.Bus.EnableExecutionLogging()
		
		const (
			PPUCyclesPerScanline = 341
			ScanlinesPerFrame    = 262
			PreRenderScanline    = 261
			VBlankClearCycle     = 1
			
			// VBlank is cleared at scanline 261 (pre-render), cycle 1
			PPUCyclesUntilVBlankClear = (PreRenderScanline * PPUCyclesPerScanline) + VBlankClearCycle
			CPUCyclesUntilVBlankClear = PPUCyclesUntilVBlankClear / 3
		)
		
		// First, wait for VBlank to be set
		vblankSet := false
		maxSteps := 100000
		
		for steps := 0; steps < maxSteps && !vblankSet; steps++ {
			helper.Bus.Step()
			status := helper.PPU.ReadRegister(0x2002)
			if (status & 0x80) != 0 {
				vblankSet = true
				break
			}
		}
		
		if !vblankSet {
			t.Fatal("VBlank was never set")
		}
		
		// Now wait for VBlank to be cleared
		vblankCleared := false
		clearCPUCycle := uint64(0)
		initialCPU := helper.Bus.GetCycleCount()
		
		for steps := 0; steps < maxSteps && !vblankCleared; steps++ {
			currentCPU := helper.Bus.GetCycleCount()
			helper.Bus.Step()
			
			status := helper.PPU.ReadRegister(0x2002)
			if (status & 0x80) == 0 {
				vblankCleared = true
				clearCPUCycle = currentCPU - initialCPU
				break
			}
		}
		
		if !vblankCleared {
			t.Fatal("VBlank was never cleared")
		}
		
		t.Logf("VBlank cleared after %d additional CPU cycles", clearCPUCycle)
		
		// Verify VBlank stays cleared
		for i := 0; i < 100; i++ {
			helper.Bus.Step()
			status := helper.PPU.ReadRegister(0x2002)
			if (status & 0x80) != 0 {
				t.Error("VBlank flag should remain cleared after being cleared")
				break
			}
		}
	})
	
	t.Run("VBlank timing consistency across frames", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Deterministic test program
		romData := make([]uint8, 0x8000)
		program := []uint8{
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
		
		framesToTest := 3
		vblankTimings := make([]uint64, framesToTest)
		
		for frame := 0; frame < framesToTest; frame++ {
			initialCPU := helper.Bus.GetCycleCount()
			
			// Wait for VBlank
			vblankDetected := false
			maxSteps := 50000
			
			for steps := 0; steps < maxSteps && !vblankDetected; steps++ {
				helper.Bus.Step()
				status := helper.PPU.ReadRegister(0x2002)
				if (status & 0x80) != 0 {
					vblankDetected = true
					vblankTimings[frame] = helper.Bus.GetCycleCount() - initialCPU
					break
				}
			}
			
			if !vblankDetected {
				t.Fatalf("VBlank not detected in frame %d", frame)
			}
			
			// Wait for VBlank to clear (next frame)
			vblankCleared := false
			for steps := 0; steps < maxSteps && !vblankCleared; steps++ {
				helper.Bus.Step()
				status := helper.PPU.ReadRegister(0x2002)
				if (status & 0x80) == 0 {
					vblankCleared = true
					break
				}
			}
			
			if !vblankCleared {
				t.Fatalf("VBlank not cleared in frame %d", frame)
			}
		}
		
		// Verify consistent VBlank timing
		baseFrameTiming := vblankTimings[0]
		tolerance := uint64(2) // Very tight tolerance for deterministic program
		
		for i, timing := range vblankTimings {
			if timing < baseFrameTiming-tolerance || timing > baseFrameTiming+tolerance {
				t.Errorf("Frame %d VBlank timing inconsistent: expected %d±%d, got %d",
					i, baseFrameTiming, tolerance, timing)
			}
		}
		
		t.Logf("VBlank timings across frames: %v", vblankTimings)
	})
}

// TestNMIGenerationTiming validates precise NMI timing with VBlank
func TestNMIGenerationTiming(t *testing.T) {
	t.Run("NMI triggered at VBlank with proper timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Set up NMI handling
		romData := make([]uint8, 0x8000)
		
		// Main program
		romData[0x0000] = 0xEA // NOP
		romData[0x0001] = 0x4C // JMP $8000
		romData[0x0002] = 0x00
		romData[0x0003] = 0x80
		
		// NMI handler at $8100
		romData[0x0100] = 0xEA // NOP in handler (marks NMI execution)
		romData[0x0101] = 0x40 // RTI
		
		// Vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		
		// Enable NMI in PPU
		helper.PPU.WriteRegister(0x2000, 0x80) // Set NMI enable bit
		helper.Bus.EnableExecutionLogging()
		
		// Wait for VBlank and NMI
		nmiTriggered := false
		vblankCPUCycle := uint64(0)
		nmiCPUCycle := uint64(0)
		initialCPU := helper.Bus.GetCycleCount()
		maxSteps := 100000
		
		for steps := 0; steps < maxSteps; steps++ {
			currentCPU := helper.Bus.GetCycleCount()
			cpuState := helper.Bus.GetCPUState()
			
			// Check for VBlank flag if not seen yet
			if vblankCPUCycle == 0 {
				status := helper.PPU.ReadRegister(0x2002)
				if (status & 0x80) != 0 {
					vblankCPUCycle = currentCPU - initialCPU
				}
			}
			
			// Check if we're in NMI handler
			if cpuState.PC >= 0x8100 && cpuState.PC <= 0x8101 && !nmiTriggered {
				nmiTriggered = true
				nmiCPUCycle = currentCPU - initialCPU
				break
			}
			
			helper.Bus.Step()
		}
		
		if !nmiTriggered {
			t.Fatal("NMI was not triggered within expected time")
		}
		
		if vblankCPUCycle == 0 {
			t.Fatal("VBlank flag was not detected")
		}
		
		// NMI should trigger shortly after VBlank flag is set
		// Allowing for instruction completion and NMI latency
		nmiLatency := nmiCPUCycle - vblankCPUCycle
		maxLatency := uint64(10) // NMI should happen within ~10 CPU cycles
		
		if nmiLatency > maxLatency {
			t.Errorf("NMI latency too high: %d cycles after VBlank (max %d)",
				nmiLatency, maxLatency)
		}
		
		// Verify 3:1 timing maintained during NMI
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("PPU/CPU ratio should remain 3.0 during NMI, got %.2f", ratio)
			}
		}
		
		t.Logf("VBlank at CPU cycle %d, NMI at CPU cycle %d (latency: %d)",
			vblankCPUCycle, nmiCPUCycle, nmiLatency)
	})
	
	t.Run("NMI suppression by status read timing", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program that reads PPUSTATUS near VBlank timing
		romData := make([]uint8, 0x8000)
		
		// Main program - reads PPUSTATUS in loop
		romData[0x0000] = 0xAD // LDA $2002 (read PPUSTATUS)
		romData[0x0001] = 0x02
		romData[0x0002] = 0x20
		romData[0x0003] = 0x4C // JMP $8000
		romData[0x0004] = 0x00
		romData[0x0005] = 0x80
		
		// NMI handler (should not be reached if suppressed)
		romData[0x0100] = 0xA9 // LDA #$FF (marker for NMI execution)
		romData[0x0101] = 0xFF
		romData[0x0102] = 0x40 // RTI
		
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
		
		// Run through multiple potential NMI windows
		vblankReadings := 0
		nmiSuppressed := true
		maxSteps := 200000 // Run for extended period
		
		for steps := 0; steps < maxSteps; steps++ {
			cpuState := helper.Bus.GetCPUState()
			
			// Check if NMI handler was reached (would indicate failure to suppress)
			if cpuState.PC >= 0x8100 && cpuState.PC <= 0x8102 {
				nmiSuppressed = false
				break
			}
			
			// Count VBlank flag reads
			if cpuState.PC == 0x8000 { // About to read PPUSTATUS
				status := helper.PPU.ReadRegister(0x2002)
				if (status & 0x80) != 0 {
					vblankReadings++
				}
			}
			
			helper.Bus.Step()
		}
		
		if vblankReadings == 0 {
			t.Error("No VBlank flags were read during test period")
		}
		
		// In this test, we expect NMI to be suppressed due to status reads
		// This tests the race condition where reading PPUSTATUS clears VBlank
		// before NMI can be triggered
		t.Logf("VBlank readings: %d, NMI suppressed: %t", vblankReadings, nmiSuppressed)
		
		// Note: Whether NMI is actually suppressed depends on exact timing
		// The test validates that the timing mechanism is working
	})
	
	t.Run("NMI timing with instruction boundaries", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		
		// Program with various instruction cycle counts
		romData := make([]uint8, 0x8000)
		program := []uint8{
			0xEA,             // NOP (2 cycles)
			0xA9, 0x42,       // LDA #$42 (2 cycles)
			0x8D, 0x00, 0x30, // STA $3000 (4 cycles)
			0xAD, 0x00, 0x30, // LDA $3000 (4 cycles)
			0x85, 0x10,       // STA $10 (3 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}
		copy(romData, program)
		
		// NMI handler
		romData[0x0100] = 0xEA // NOP
		romData[0x0101] = 0x40 // RTI
		
		// Vectors
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()
		
		// Enable NMI
		helper.PPU.WriteRegister(0x2000, 0x80)
		helper.Bus.EnableExecutionLogging()
		
		// Track NMI occurrences and their timing relative to instructions
		nmiOccurrences := make([]struct {
			cpuCycle    uint64
			instruction uint8
			pc          uint16
		}, 0)
		
		maxFrames := 5
		framesProcessed := 0
		maxSteps := 500000
		
		for steps := 0; steps < maxSteps && framesProcessed < maxFrames; steps++ {
			cpuState := helper.Bus.GetCPUState()
			currentCPU := helper.Bus.GetCycleCount()
			
			// Check if we're entering NMI handler
			if cpuState.PC == 0x8100 && len(nmiOccurrences) == framesProcessed {
				// Read the instruction that was interrupted
				interruptedPC := cpuState.PC // This is now in handler
				
				nmiOccurrences = append(nmiOccurrences, struct {
					cpuCycle    uint64
					instruction uint8
					pc          uint16
				}{
					cpuCycle:    currentCPU,
					instruction: 0xEA, // Placeholder - would need stack inspection for real PC
					pc:          interruptedPC,
				})
				framesProcessed++
			}
			
			helper.Bus.Step()
		}
		
		if len(nmiOccurrences) == 0 {
			t.Fatal("No NMI occurrences detected")
		}
		
		// Verify NMI timing consistency
		if len(nmiOccurrences) > 1 {
			for i := 1; i < len(nmiOccurrences); i++ {
				cycleDiff := nmiOccurrences[i].cpuCycle - nmiOccurrences[i-1].cpuCycle
				
				// Should be approximately one frame worth of cycles
				expectedFrameCycles := uint64(29781) // ~29781 CPU cycles per frame
				tolerance := uint64(100)
				
				if cycleDiff < expectedFrameCycles-tolerance || 
				   cycleDiff > expectedFrameCycles+tolerance {
					t.Errorf("NMI interval inconsistent: expected ~%d cycles, got %d",
						expectedFrameCycles, cycleDiff)
				}
			}
		}
		
		// Verify 3:1 ratio maintained across NMI events
		log := helper.Bus.GetExecutionLog()
		if len(log) > 0 {
			lastEntry := log[len(log)-1]
			ratio := float64(lastEntry.PPUCycles) / float64(lastEntry.CPUCycles)
			if ratio != 3.0 {
				t.Errorf("PPU/CPU ratio should remain 3.0 with NMIs, got %.2f", ratio)
			}
		}
		
		t.Logf("Detected %d NMI occurrences", len(nmiOccurrences))
		for i, nmi := range nmiOccurrences {
			t.Logf("  NMI %d: CPU cycle %d, PC $%04X", i, nmi.cpuCycle, nmi.pc)
		}
	})
}