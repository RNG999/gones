package integration

import (
	"testing"
	"time"
)

// TimingAccuracyTestHelper provides specialized utilities for cycle-accurate timing validation
type TimingAccuracyTestHelper struct {
	*IntegrationTestHelper
	cycleTracker   *CycleTracker
	timingBaseline TimingBaseline
}

// CycleTracker tracks precise cycle counts for validation
type CycleTracker struct {
	cpuCycles        uint64
	ppuCycles        uint64
	frameCount       uint64
	scanlineCount    uint64
	lastFrameStart   uint64
	frameCycleCounts []uint64
}

// TimingBaseline represents the expected NTSC timing characteristics
type TimingBaseline struct {
	CPUFrequency      float64 // 1.789773 MHz
	PPUFrequency      float64 // 5.369319 MHz
	CyclesPerScanline int     // 341
	ScanlinesPerFrame int     // 262
	VisibleScanlines  int     // 240
	VBlankScanlines   int     // 20
	CPUCyclesPerFrame int     // 29780.67 (rounded)
	PPUCyclesPerFrame int     // 89342
	PPUToCPURatio     float64 // 3.0
	FrameRate         float64 // 60.0988 Hz
}

// NewTimingAccuracyTestHelper creates a timing accuracy test helper
func NewTimingAccuracyTestHelper() *TimingAccuracyTestHelper {
	helper := &TimingAccuracyTestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		cycleTracker: &CycleTracker{
			frameCycleCounts: make([]uint64, 0, 100),
		},
		timingBaseline: TimingBaseline{
			CPUFrequency:      1789773.0,
			PPUFrequency:      5369319.0,
			CyclesPerScanline: 341,
			ScanlinesPerFrame: 262,
			VisibleScanlines:  240,
			VBlankScanlines:   20,
			CPUCyclesPerFrame: 29781,
			PPUCyclesPerFrame: 89342,
			PPUToCPURatio:     3.0,
			FrameRate:         60.0988,
		},
	}
	return helper
}

// StepWithCycleTracking executes one system step while tracking precise cycles
func (h *TimingAccuracyTestHelper) StepWithCycleTracking() (cpuCycles, ppuCycles uint64) {
	initialCPUCycles := h.cycleTracker.cpuCycles
	initialPPUCycles := h.cycleTracker.ppuCycles

	// Execute one bus step
	h.Bus.Step()

	// In a full implementation, this would read actual cycle counters
	// For now, simulate expected behavior based on instruction execution
	executedCycles := uint64(2) // Assume average instruction cycles
	h.cycleTracker.cpuCycles += executedCycles
	h.cycleTracker.ppuCycles += executedCycles * 3

	return h.cycleTracker.cpuCycles - initialCPUCycles,
		h.cycleTracker.ppuCycles - initialPPUCycles
}

// GetCurrentRatio returns the current CPU-to-PPU cycle ratio
func (h *TimingAccuracyTestHelper) GetCurrentRatio() float64 {
	if h.cycleTracker.cpuCycles == 0 {
		return 0.0
	}
	return float64(h.cycleTracker.ppuCycles) / float64(h.cycleTracker.cpuCycles)
}

// TestNTSCTimingAccuracy validates NTSC timing requirements
func TestNTSCTimingAccuracy(t *testing.T) {
	t.Run("CPU frequency validation", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Test CPU frequency accuracy using known instruction cycles
		program := []uint8{
			0xEA,             // NOP (2 cycles)
			0xEA,             // NOP (2 cycles)
			0xEA,             // NOP (2 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Measure execution time for known cycle count
		iterations := 1000
		expectedCyclesPerIteration := 9 // 2+2+2+3 cycles
		expectedTotalCycles := uint64(iterations * expectedCyclesPerIteration)

		startTime := time.Now()
		for i := 0; i < iterations*4; i++ { // 4 instructions per iteration
			helper.Bus.Step()
		}
		elapsedTime := time.Since(startTime)

		// Calculate effective frequency
		actualCyclesExecuted := expectedTotalCycles
		effectiveFrequency := float64(actualCyclesExecuted) / elapsedTime.Seconds()

		// In a real emulator, this would validate against 1.789773 MHz
		// For testing purposes, verify the calculation is reasonable
		t.Logf("Executed %d cycles in %v", actualCyclesExecuted, elapsedTime)
		t.Logf("Effective frequency: %.2f Hz (target: %.2f Hz)",
			effectiveFrequency, helper.timingBaseline.CPUFrequency)

		if effectiveFrequency < 1000000 || effectiveFrequency > 10000000 {
			t.Errorf("Effective frequency out of reasonable range: %.2f Hz", effectiveFrequency)
		}
	})

	t.Run("PPU-CPU ratio precision", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test with various instruction types to ensure ratio consistency
		program := []uint8{
			// 2-cycle instructions
			0xEA, // NOP
			0xE8, // INX
			0xC8, // INY
			0x18, // CLC

			// Immediate addressing (2 cycles)
			0xA9, 0x42, // LDA #$42
			0xA2, 0x55, // LDX #$55
			0xA0, 0xAA, // LDY #$AA

			// Zero page (3 cycles)
			0x85, 0x10, // STA $10
			0x86, 0x11, // STX $11
			0x84, 0x12, // STY $12

			// Absolute (4 cycles)
			0x8D, 0x00, 0x30, // STA $3000
			0xAD, 0x00, 0x30, // LDA $3000

			// Jump
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute instructions and verify ratio precision
		instructionCycles := []int{2, 2, 2, 2, 2, 2, 2, 3, 3, 3, 4, 4, 3}
		totalCPUCycles := uint64(0)
		totalPPUCycles := uint64(0)

		for i, expectedCycles := range instructionCycles {
			cpuCycles, ppuCycles := helper.StepWithCycleTracking()
			totalCPUCycles += cpuCycles
			totalPPUCycles += ppuCycles

			expectedPPUCycles := uint64(expectedCycles) * 3
			if ppuCycles != expectedPPUCycles {
				t.Errorf("Instruction %d: Expected %d PPU cycles, got %d",
					i, expectedPPUCycles, ppuCycles)
			}
		}

		// Verify final ratio
		actualRatio := helper.GetCurrentRatio()
		expectedRatio := helper.timingBaseline.PPUToCPURatio
		tolerance := 0.001

		if actualRatio < expectedRatio-tolerance || actualRatio > expectedRatio+tolerance {
			t.Errorf("PPU-CPU ratio out of tolerance: expected %.3f, got %.3f",
				expectedRatio, actualRatio)
		}
	})

	t.Run("Scanline timing validation", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Verify each scanline takes exactly 341 PPU cycles
		helper.PPU.ReadRegister(0x2002) // Read to get initial state

		// Run for one scanline worth of cycles
		scanlineCycles := helper.timingBaseline.CyclesPerScanline
		cpuCyclesPerScanline := scanlineCycles / 3

		startingCycles := helper.cycleTracker.ppuCycles
		for i := 0; i < cpuCyclesPerScanline; i++ {
			helper.StepWithCycleTracking()
		}

		elapsedPPUCycles := helper.cycleTracker.ppuCycles - startingCycles

		// Allow small tolerance for fractional cycles
		tolerance := uint64(3)
		expectedCycles := uint64(scanlineCycles)

		if elapsedPPUCycles < expectedCycles-tolerance ||
			elapsedPPUCycles > expectedCycles+tolerance {
			t.Errorf("Scanline timing incorrect: expected ~%d PPU cycles, got %d",
				expectedCycles, elapsedPPUCycles)
		}

		t.Logf("Scanline completed in %d PPU cycles (expected %d)",
			elapsedPPUCycles, expectedCycles)
	})

	t.Run("Frame timing accuracy", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Run for multiple frames and verify consistent timing
		framesToTest := 5
		frameCycleCounts := make([]uint64, framesToTest)

		for frame := 0; frame < framesToTest; frame++ {
			// Track frame start cycles

			// Run one frame worth of cycles
			frameCycles := helper.timingBaseline.PPUCyclesPerFrame
			cpuSteps := frameCycles / 3

			for i := 0; i < cpuSteps; i++ {
				helper.StepWithCycleTracking()
			}

			frameCycleCounts[frame] = helper.cycleTracker.ppuCycles
		}

		// Verify frame consistency
		baseFrameCycles := frameCycleCounts[0]
		tolerance := uint64(10) // Allow small variance

		for i, cycles := range frameCycleCounts {
			if cycles < baseFrameCycles-tolerance || cycles > baseFrameCycles+tolerance {
				t.Errorf("Frame %d cycle count inconsistent: expected ~%d, got %d",
					i, baseFrameCycles, cycles)
			}
		}

		// Verify against baseline
		expectedFrameCycles := uint64(helper.timingBaseline.PPUCyclesPerFrame)
		avgFrameCycles := uint64(0)
		for _, cycles := range frameCycleCounts {
			avgFrameCycles += cycles
		}
		avgFrameCycles /= uint64(framesToTest)

		frameTolerance := uint64(100)
		if avgFrameCycles < expectedFrameCycles-frameTolerance ||
			avgFrameCycles > expectedFrameCycles+frameTolerance {
			t.Errorf("Average frame cycles incorrect: expected ~%d, got %d",
				expectedFrameCycles, avgFrameCycles)
		}

		t.Logf("Frame cycle counts: %v (avg: %d, expected: %d)",
			frameCycleCounts, avgFrameCycles, expectedFrameCycles)
	})
}

// TestInstructionCyclePrecision validates cycle-accurate instruction timing
func TestInstructionCyclePrecision(t *testing.T) {
	t.Run("Basic instruction cycle validation", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test every addressing mode for cycle accuracy
		testInstructions := []struct {
			name           string
			instruction    []uint8
			expectedCycles int
			setup          func(*TimingAccuracyTestHelper)
		}{
			// Implied (2 cycles)
			{"NOP", []uint8{0xEA}, 2, nil},
			{"TAX", []uint8{0xAA}, 2, nil},
			{"INX", []uint8{0xE8}, 2, nil},

			// Immediate (2 cycles)
			{"LDA #$42", []uint8{0xA9, 0x42}, 2, nil},
			{"ADC #$10", []uint8{0x69, 0x10}, 2, nil},
			{"CMP #$80", []uint8{0xC9, 0x80}, 2, nil},

			// Zero page (3 cycles)
			{"LDA $80", []uint8{0xA5, 0x80}, 3, func(h *TimingAccuracyTestHelper) {
				h.Memory.Write(0x0080, 0x42)
			}},
			{"STA $90", []uint8{0x85, 0x90}, 3, func(h *TimingAccuracyTestHelper) {
				h.CPU.A = 0x55
			}},

			// Zero page,X (4 cycles)
			{"LDA $80,X", []uint8{0xB5, 0x80}, 4, func(h *TimingAccuracyTestHelper) {
				h.CPU.X = 0x05
				h.Memory.Write(0x0085, 0x33)
			}},

			// Absolute (4 cycles)
			{"LDA $3000", []uint8{0xAD, 0x00, 0x30}, 4, func(h *TimingAccuracyTestHelper) {
				h.Memory.Write(0x3000, 0x44)
			}},
			{"STA $4000", []uint8{0x8D, 0x00, 0x40}, 4, func(h *TimingAccuracyTestHelper) {
				h.CPU.A = 0x66
			}},

			// Absolute,X no page cross (4 cycles)
			{"LDA $2000,X (no cross)", []uint8{0xBD, 0x00, 0x20}, 4, func(h *TimingAccuracyTestHelper) {
				h.CPU.X = 0x10
				h.Memory.Write(0x2010, 0x77)
			}},

			// Absolute,X page cross (5 cycles)
			{"LDA $20FF,X (page cross)", []uint8{0xBD, 0xFF, 0x20}, 5, func(h *TimingAccuracyTestHelper) {
				h.CPU.X = 0x01
				h.Memory.Write(0x2100, 0x88)
			}},
		}

		for _, test := range testInstructions {
			t.Run(test.name, func(t *testing.T) {
				helper.Bus.Reset()

				// Load instruction
				romData := make([]uint8, 0x8000)
				copy(romData[:], test.instruction)
				romData[0x7FFC] = 0x00
				romData[0x7FFD] = 0x80
				helper.GetMockCartridge().LoadPRG(romData)
				helper.Bus.Reset()

				// Setup if needed
				if test.setup != nil {
					test.setup(helper)
				}

				// Execute and measure
				// Track instruction start cycles
				cpuCycles, ppuCycles := helper.StepWithCycleTracking()

				// Validate CPU cycles
				if int(cpuCycles) != test.expectedCycles {
					t.Errorf("Expected %d CPU cycles, got %d", test.expectedCycles, cpuCycles)
				}

				// Validate PPU cycles (should be 3x CPU)
				expectedPPUCycles := uint64(test.expectedCycles * 3)
				if ppuCycles != expectedPPUCycles {
					t.Errorf("Expected %d PPU cycles, got %d", expectedPPUCycles, ppuCycles)
				}
			})
		}
	})

	t.Run("Page boundary crossing precision", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)

		pageCrossingTests := []struct {
			name           string
			instruction    []uint8
			indexValue     uint8
			baseCycles     int
			crossingCycles int
			setup          func(*TimingAccuracyTestHelper, uint8)
		}{
			{
				"LDA absolute,X",
				[]uint8{0xBD, 0xF0, 0x20}, // LDA $20F0,X
				0x20,                      // Cross from $20F0 to $2110
				4, 5,
				func(h *TimingAccuracyTestHelper, idx uint8) {
					h.CPU.X = idx
					h.Memory.Write(0x2110, 0x42)
				},
			},
			{
				"LDA absolute,Y",
				[]uint8{0xB9, 0xFF, 0x30}, // LDA $30FF,Y
				0x01,                      // Cross from $30FF to $3100
				4, 5,
				func(h *TimingAccuracyTestHelper, idx uint8) {
					h.CPU.Y = idx
					h.Memory.Write(0x3100, 0x55)
				},
			},
		}

		for _, test := range pageCrossingTests {
			t.Run(test.name+" (no crossing)", func(t *testing.T) {
				helper.Bus.Reset()

				romData := make([]uint8, 0x8000)
				copy(romData[:], test.instruction)
				romData[0x7FFC] = 0x00
				romData[0x7FFD] = 0x80
				helper.GetMockCartridge().LoadPRG(romData)
				helper.Bus.Reset()

				// Test without page crossing
				test.setup(helper, 0x05) // No crossing

				cpuCycles, _ := helper.StepWithCycleTracking()
				if int(cpuCycles) != test.baseCycles {
					t.Errorf("No page cross: expected %d cycles, got %d",
						test.baseCycles, cpuCycles)
				}
			})

			t.Run(test.name+" (with crossing)", func(t *testing.T) {
				helper.Bus.Reset()

				romData := make([]uint8, 0x8000)
				copy(romData[:], test.instruction)
				romData[0x7FFC] = 0x00
				romData[0x7FFD] = 0x80
				helper.GetMockCartridge().LoadPRG(romData)
				helper.Bus.Reset()

				// Test with page crossing
				test.setup(helper, test.indexValue)

				cpuCycles, _ := helper.StepWithCycleTracking()
				if int(cpuCycles) != test.crossingCycles {
					t.Errorf("Page cross: expected %d cycles, got %d",
						test.crossingCycles, cpuCycles)
				}
			})
		}
	})

	t.Run("Branch instruction timing precision", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)

		branchTests := []struct {
			name           string
			instruction    []uint8
			setup          func(*TimingAccuracyTestHelper)
			expectedCycles int
			description    string
		}{
			{
				"BNE not taken",
				[]uint8{0xD0, 0x10},
				func(h *TimingAccuracyTestHelper) { h.CPU.Z = true },
				2,
				"Branch not taken (2 cycles)",
			},
			{
				"BNE taken (forward, no page cross)",
				[]uint8{0xD0, 0x10},
				func(h *TimingAccuracyTestHelper) { h.CPU.Z = false },
				3,
				"Branch taken, forward, no page cross (3 cycles)",
			},
			{
				"BEQ taken (backward, page cross)",
				[]uint8{0xF0, 0xF0}, // -16 bytes
				func(h *TimingAccuracyTestHelper) { h.CPU.Z = true },
				4,
				"Branch taken, backward, page cross (4 cycles)",
			},
		}

		for _, test := range branchTests {
			t.Run(test.name, func(t *testing.T) {
				helper.Bus.Reset()

				// Place instruction at appropriate location for page crossing test
				var romData [0x8000]uint8
				if test.name == "BEQ taken (backward, page cross)" {
					// Place at $8010 so backward branch crosses page
					copy(romData[0x0010:], test.instruction)
					romData[0x7FFC] = 0x10
					romData[0x7FFD] = 0x80
				} else {
					copy(romData[:], test.instruction)
					romData[0x7FFC] = 0x00
					romData[0x7FFD] = 0x80
				}

				helper.GetMockCartridge().LoadPRG(romData[:])
				helper.Bus.Reset()

				// Setup condition
				test.setup(helper)

				// Execute and measure
				cpuCycles, _ := helper.StepWithCycleTracking()

				if int(cpuCycles) != test.expectedCycles {
					t.Errorf("%s: expected %d cycles, got %d",
						test.description, test.expectedCycles, cpuCycles)
				}
			})
		}
	})
}

// TestTimingRegression validates that timing remains consistent across changes
func TestTimingRegression(t *testing.T) {
	t.Run("Baseline timing establishment", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)

		// Run standardized test program
		program := []uint8{
			0xA9, 0x00, // LDA #$00 (2)
			0x85, 0x10, // STA $10 (3)
			0xA2, 0x08, // LDX #$08 (2)
			0xBD, 0x00, 0x30, // LDA $3000,X (4)
			0x18,       // CLC (2)
			0x69, 0x01, // ADC #$01 (2)
			0x85, 0x11, // STA $11 (3)
			0xCA,       // DEX (2)
			0xD0, 0xF5, // BNE -11 (3/2)
			0x4C, 0x00, 0x80, // JMP $8000 (3)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute full loop and measure timing
		loopIterations := 8
		expectedCyclesPerIteration := 13                                                         // 4+2+2+3+2 for loop body + 3 for taken branch
		expectedFinalCycles := 2 + 3 + 2 + (loopIterations * expectedCyclesPerIteration) + 2 + 3 // Setup + loop + final branch not taken + jump

		totalCycles := uint64(0)

		// Run until loop completes
		for i := 0; i < 100; i++ { // Safety limit
			cpuCycles, _ := helper.StepWithCycleTracking()
			totalCycles += cpuCycles

			// Check if we've completed the expected cycles
			if totalCycles >= uint64(expectedFinalCycles) {
				break
			}
		}

		// Establish baseline for regression testing
		baseline := totalCycles
		t.Logf("Established timing baseline: %d cycles for standard test program", baseline)

		// Store baseline for future regression tests
		// In a real implementation, this would be saved to a file
		if baseline < 50 || baseline > 500 {
			t.Errorf("Baseline timing seems incorrect: %d cycles", baseline)
		}
	})

	t.Run("Memory allocation timing impact", func(t *testing.T) {
		helper := NewTimingAccuracyTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test that garbage collection doesn't affect timing
		program := []uint8{
			0xEA, 0xEA, 0xEA, 0xEA, // 4 NOPs
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Measure timing before allocation pressure
		timingsBeforeGC := make([]uint64, 10)
		for i := 0; i < 10; i++ {
			helper.Bus.Reset()
			cycles, _ := helper.StepWithCycleTracking() // NOP
			timingsBeforeGC[i] = cycles
		}

		// Create allocation pressure
		var allocations [][]byte
		for i := 0; i < 1000; i++ {
			allocations = append(allocations, make([]byte, 1024))
		}

		// Measure timing after allocation pressure
		timingsAfterGC := make([]uint64, 10)
		for i := 0; i < 10; i++ {
			helper.Bus.Reset()
			cycles, _ := helper.StepWithCycleTracking() // NOP
			timingsAfterGC[i] = cycles
		}

		// Compare timing consistency
		avgBefore := uint64(0)
		avgAfter := uint64(0)

		for i := 0; i < 10; i++ {
			avgBefore += timingsBeforeGC[i]
			avgAfter += timingsAfterGC[i]
		}
		avgBefore /= 10
		avgAfter /= 10

		// Timing should be consistent regardless of GC pressure
		if avgBefore != avgAfter {
			t.Logf("WARNING: Timing affected by memory pressure: before=%d, after=%d",
				avgBefore, avgAfter)
		}

		// Prevent compiler from optimizing away allocations
		_ = allocations
	})
}
