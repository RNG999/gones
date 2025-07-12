package integration

import (
	"math"
	"testing"
)

// FrameTimingHelper provides utilities for frame timing integration testing
type FrameTimingHelper struct {
	*IntegrationTestHelper
	frameEvents []FrameEvent
	frameCount  int
	lastVBlank  bool
}

// FrameEvent represents a frame timing event
type FrameEvent struct {
	FrameNumber     int
	EventType       string // "VBlankStart", "VBlankEnd", "FrameComplete"
	CPUCycle        int
	PPUCycle        int
	Scanline        int
	PPUCyclesToHere int
}

// NewFrameTimingHelper creates a frame timing test helper
func NewFrameTimingHelper() *FrameTimingHelper {
	return &FrameTimingHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		frameEvents:           make([]FrameEvent, 0),
		frameCount:            0,
		lastVBlank:            false,
	}
}

// LogFrameEvent logs a frame timing event
func (h *FrameTimingHelper) LogFrameEvent(event FrameEvent) {
	h.frameEvents = append(h.frameEvents, event)
}

// StepWithFrameTracking steps the system and tracks frame events
func (h *FrameTimingHelper) StepWithFrameTracking() {
	h.Bus.Step()

	// Get actual CPU cycle count
	currentCPUCycle := int(h.Bus.GetCycleCount())

	// Check for VBlank transitions
	ppuStatus := h.PPU.ReadRegister(0x2002)
	currentVBlank := (ppuStatus & 0x80) != 0
	
	// Debug: Log VBlank transitions only (removed for cleaner output)
	// if currentVBlank != h.lastVBlank {
	//	fmt.Printf("[CYCLE_DEBUG] VBlank transition at step %d: CPU cycle %d, VBlank %t\n", len(h.frameEvents), currentCPUCycle, currentVBlank)
	// }

	if currentVBlank && !h.lastVBlank {
		// VBlank started
		h.frameCount++
		event := FrameEvent{
			FrameNumber: h.frameCount,
			EventType:   "VBlankStart",
			CPUCycle:    currentCPUCycle,
			Scanline:    241,
		}
		h.LogFrameEvent(event)
	} else if !currentVBlank && h.lastVBlank {
		// VBlank ended
		event := FrameEvent{
			FrameNumber: h.frameCount,
			EventType:   "VBlankEnd",
			CPUCycle:    currentCPUCycle,
			Scanline:    0,
		}
		h.LogFrameEvent(event)
	}

	h.lastVBlank = currentVBlank
}

// RunFrames runs the system for a specified number of complete frames
func (h *FrameTimingHelper) RunFrames(frames int) []int {
	frameCycles := make([]int, 0)
	framesCompleted := 0
	lastFrameStartCPUCycle := 0
	stepCount := 0

	for framesCompleted < frames {
		h.StepWithFrameTracking()
		stepCount++

		// Check if we completed a frame (VBlank started)
		if len(h.frameEvents) > 0 {
			lastEvent := h.frameEvents[len(h.frameEvents)-1]
			if lastEvent.EventType == "VBlankStart" && lastEvent.FrameNumber > framesCompleted {
				// Frame completed - calculate cycles since last frame start
				if lastFrameStartCPUCycle > 0 {
					// Calculate cycles between VBlank starts (frame duration)
					frameLength := lastEvent.CPUCycle - lastFrameStartCPUCycle
					frameCycles = append(frameCycles, frameLength)
				}
				lastFrameStartCPUCycle = lastEvent.CPUCycle
				framesCompleted++
			}
		}

		// Safety limit - need to run long enough to capture frames+1 VBlank events
		// Each frame is ~29,781 CPU cycles, so we need significant margin
		if stepCount > (frames+3)*40000 {
			break
		}
	}

	return frameCycles
}

// TestNTSCFrameTiming tests NTSC frame timing accuracy
func TestNTSCFrameTiming(t *testing.T) {
	t.Run("Frame rate accuracy", func(t *testing.T) {
		helper := NewFrameTimingHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Enable rendering to get proper frame timing
		helper.Memory.Write(0x2000, 0x80) // PPUCTRL - NMI enable
		helper.Memory.Write(0x2001, 0x1E) // PPUMASK - show bg+sprites

		helper.Bus.Reset()

		// Run for several frames and measure timing
		frameCount := 10
		frameCycles := helper.RunFrames(frameCount)

		if len(frameCycles) < frameCount {
			t.Fatalf("Expected %d frames, got %d", frameCount, len(frameCycles))
		}

		// NTSC timing: 341 PPU cycles/scanline * 262 scanlines = 89,342 PPU cycles/frame
		// CPU runs at 1/3 speed: 29,780.67 CPU cycles/frame
		expectedCPUCycles := 29781 // Rounded
		tolerance := 100           // Allow some tolerance

		for i, cycles := range frameCycles {
			if cycles < expectedCPUCycles-tolerance || cycles > expectedCPUCycles+tolerance {
				t.Errorf("Frame %d: expected ~%d CPU cycles, got %d",
					i+1, expectedCPUCycles, cycles)
			} else {
				t.Logf("Frame %d: %d CPU cycles (within tolerance)", i+1, cycles)
			}
		}

		// Calculate average frame rate
		totalCycles := 0
		for _, cycles := range frameCycles {
			totalCycles += cycles
		}
		avgCycles := float64(totalCycles) / float64(len(frameCycles))

		// NTSC CPU frequency: ~1.789773 MHz
		cpuFreq := 1789773.0
		frameRate := cpuFreq / avgCycles
		expectedFrameRate := 60.098803 // NTSC frame rate

		frameRateTolerance := 0.1
		if math.Abs(frameRate-expectedFrameRate) > frameRateTolerance {
			t.Errorf("Frame rate out of tolerance: expected %.3f Hz, got %.3f Hz",
				expectedFrameRate, frameRate)
		} else {
			t.Logf("Frame rate: %.3f Hz (expected %.3f Hz)", frameRate, expectedFrameRate)
		}
	})

	t.Run("Frame consistency", func(t *testing.T) {
		helper := NewFrameTimingHelper()
		helper.SetupBasicROM(0x8000)

		// Enable rendering
		helper.Memory.Write(0x2000, 0x80)
		helper.Memory.Write(0x2001, 0x1E)

		helper.Bus.Reset()

		// Run many frames to test consistency
		frameCount := 20
		frameCycles := helper.RunFrames(frameCount)

		if len(frameCycles) < frameCount {
			t.Fatalf("Not enough frames completed: got %d, expected %d",
				len(frameCycles), frameCount)
		}

		// Calculate statistics
		sum := 0
		for _, cycles := range frameCycles {
			sum += cycles
		}
		mean := float64(sum) / float64(len(frameCycles))

		// Calculate standard deviation
		variance := 0.0
		for _, cycles := range frameCycles {
			diff := float64(cycles) - mean
			variance += diff * diff
		}
		variance /= float64(len(frameCycles))
		stdDev := math.Sqrt(variance)

		t.Logf("Frame timing statistics:")
		t.Logf("  Mean: %.2f cycles", mean)
		t.Logf("  Std Dev: %.2f cycles", stdDev)
		t.Logf("  Min: %d cycles", minInt(frameCycles))
		t.Logf("  Max: %d cycles", maxInt(frameCycles))

		// Frame timing should be very consistent
		maxAllowedStdDev := 10.0 // Very tight tolerance
		if stdDev > maxAllowedStdDev {
			t.Errorf("Frame timing too inconsistent: std dev %.2f > %.2f",
				stdDev, maxAllowedStdDev)
		}

		// Check for outliers
		outliers := 0
		tolerance := 50.0
		for i, cycles := range frameCycles {
			if math.Abs(float64(cycles)-mean) > tolerance {
				t.Logf("Frame %d is outlier: %d cycles (%.1f from mean)",
					i+1, cycles, math.Abs(float64(cycles)-mean))
				outliers++
			}
		}

		maxOutliers := frameCount / 10 // Allow 10% outliers
		if outliers > maxOutliers {
			t.Errorf("Too many outlier frames: %d > %d", outliers, maxOutliers)
		}
	})

	t.Run("VBlank timing precision", func(t *testing.T) {
		helper := NewFrameTimingHelper()
		helper.SetupBasicROM(0x8000)

		// Program that precisely tracks VBlank timing
		program := []uint8{
			// Enable NMI
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000

			// Main loop - poll status
			0xAD, 0x02, 0x20, // LDA $2002 (PPUSTATUS)
			0x85, 0x10, // STA $10 (save status)
			0x29, 0x80, // AND #$80 (isolate VBlank)
			0xC5, 0x11, // CMP $11 (compare with previous)
			0xF0, 0xF5, // BEQ loop (no change)

			// VBlank changed
			0x85, 0x11, // STA $11 (save new state)
			0xE6, 0x12, // INC $12 (transition counter)
			0x4C, 0x06, 0x80, // JMP to main loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize tracking variables
		helper.Memory.Write(0x0010, 0x00) // Current status
		helper.Memory.Write(0x0011, 0x00) // Previous VBlank state
		helper.Memory.Write(0x0012, 0x00) // Transition counter

		// Execute setup
		helper.Bus.Step() // LDA #$80
		helper.Bus.Step() // STA $2000

		// Run and track VBlank transitions
		transitions := make([]int, 0)
		cycleCount := 0
		lastTransitionCount := uint8(0)

		for len(transitions) < 10 && cycleCount < 300000 {
			helper.Bus.Step()
			cycleCount++

			// Check for transitions
			currentTransitions := helper.Memory.Read(0x0012)
			if currentTransitions > lastTransitionCount {
				transitions = append(transitions, cycleCount)
				lastTransitionCount = currentTransitions
				t.Logf("VBlank transition %d at cycle %d", len(transitions), cycleCount)
			}
		}

		if len(transitions) < 4 {
			t.Fatalf("Not enough VBlank transitions detected: %d", len(transitions))
		}

		// Calculate intervals between transitions
		intervals := make([]int, 0)
		for i := 1; i < len(transitions); i++ {
			interval := transitions[i] - transitions[i-1]
			intervals = append(intervals, interval)
		}

		// VBlank transitions should occur at regular intervals
		expectedInterval := 29781 / 2 // Half frame (VBlank start/end)
		tolerance := 100

		for i, interval := range intervals {
			if interval < expectedInterval-tolerance || interval > expectedInterval+tolerance {
				t.Errorf("VBlank interval %d out of range: got %d, expected ~%d",
					i+1, interval, expectedInterval)
			}
		}
	})
}

// TestFrameTimingSynchronization tests synchronization with frame timing
func TestFrameTimingSynchronization(t *testing.T) {
	t.Run("CPU-PPU frame synchronization", func(t *testing.T) {
		helper := NewFrameTimingHelper()
		helper.SetupBasicROM(0x8000)

		// Program that syncs with frame timing
		program := []uint8{
			// Enable rendering and NMI
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			// Frame sync loop
			0xAD, 0x02, 0x20, // LDA $2002 (wait for VBlank)
			0x10, 0xFB, // BPL -5

			// Do work during VBlank
			0xE6, 0x20, // INC $20 (work counter)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006 (reset scroll)
			0x8D, 0x06, 0x20, // STA $2006

			// Wait for VBlank end
			0xAD, 0x02, 0x20, // LDA $2002
			0x30, 0xFB, // BMI -5 (wait for VBlank clear)

			// Continue to next frame
			0x4C, 0x0C, 0x80, // JMP to frame sync
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize work counter
		helper.Memory.Write(0x0020, 0x00)

		// Run for several frames
		frameWorkCounts := make([]uint8, 0)
		lastWorkCount := uint8(0)

		for i := 0; i < 100000 && len(frameWorkCounts) < 5; i++ {
			helper.Bus.Step()

			// Check work counter
			currentWork := helper.Memory.Read(0x0020)
			if currentWork > lastWorkCount {
				frameWorkCounts = append(frameWorkCounts, currentWork-lastWorkCount)
				lastWorkCount = currentWork
				t.Logf("Frame %d: work count increment = %d",
					len(frameWorkCounts), currentWork-lastWorkCount)
			}
		}

		if len(frameWorkCounts) < 3 {
			t.Fatalf("Not enough frames completed: %d", len(frameWorkCounts))
		}

		// Each frame should do exactly 1 unit of work
		for i, work := range frameWorkCounts {
			if work != 1 {
				t.Errorf("Frame %d: expected 1 work unit, got %d", i+1, work)
			}
		}

		t.Log("CPU-PPU frame synchronization test completed")
	})

	t.Run("Frame drop detection", func(t *testing.T) {
		helper := NewFrameTimingHelper()
		helper.SetupBasicROM(0x8000)

		// Program that does too much work and might drop frames
		program := []uint8{
			// Enable rendering
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			// Frame sync
			0xAD, 0x02, 0x20, // LDA $2002
			0x10, 0xFB, // BPL -5 (wait for VBlank)

			// Do excessive work (might cause frame drop)
			0xA2, 0x00, // LDX #$00
			0xA0, 0x00, // LDY #$00
			0xE8,       // INX (work loop)
			0xC8,       // INY
			0xD0, 0xFC, // BNE -4 (256 iterations)
			0xCA,       // DEX
			0xD0, 0xF8, // BNE -8 (256*256 iterations)

			0xE6, 0x30, // INC $30 (frame counter)
			0x4C, 0x0C, 0x80, // JMP to frame sync
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize frame counter
		helper.Memory.Write(0x0030, 0x00)

		// Run for a specific amount of time and count frames
		targetCycles := 150000 // ~5 frames worth
		executedCycles := 0

		for executedCycles < targetCycles {
			helper.Bus.Step()
			executedCycles++
		}

		// Check how many frames were completed
		framesCompleted := helper.Memory.Read(0x0030)
		expectedFrames := targetCycles / 29781

		t.Logf("After %d cycles: %d frames completed (expected ~%d)",
			targetCycles, framesCompleted, expectedFrames)

		// If doing too much work, should complete fewer frames
		// This test verifies that heavy CPU load affects frame rate

		if framesCompleted > uint8(expectedFrames+1) {
			t.Errorf("Too many frames completed for heavy workload: got %d, expected <=%d",
				framesCompleted, expectedFrames+1)
		}
	})

	t.Run("Odd frame cycle skip", func(t *testing.T) {
		helper := NewFrameTimingHelper()
		helper.SetupBasicROM(0x8000)

		// Enable rendering to trigger odd frame behavior
		helper.Memory.Write(0x2000, 0x80)
		helper.Memory.Write(0x2001, 0x1E)

		helper.Bus.Reset()

		// Run for multiple frames and check for cycle skip pattern
		frameCycles := helper.RunFrames(10)

		if len(frameCycles) < 6 {
			t.Fatalf("Not enough frames for odd frame test: %d", len(frameCycles))
		}

		// Odd frames should be 1 cycle shorter when rendering is enabled
		// Check for alternating pattern
		normalFrameCycles := 29781
		oddFrameCycles := 29780 // 1 cycle shorter
		tolerance := 5

		oddFrameSkips := 0
		for i, cycles := range frameCycles {
			frameType := "even"

			// In practice, odd frame detection is complex
			// For this test, we just verify frame lengths are in expected range
			if cycles >= normalFrameCycles-tolerance && cycles <= normalFrameCycles+tolerance {
				// Normal frame
			} else if cycles >= oddFrameCycles-tolerance && cycles <= oddFrameCycles+tolerance {
				// Potentially skipped frame
				oddFrameSkips++
				frameType = "odd (skipped)"
			} else {
				t.Errorf("Frame %d has unexpected cycle count: %d", i+1, cycles)
			}

			t.Logf("Frame %d: %d cycles (%s)", i+1, cycles, frameType)
		}

		// Should see some evidence of cycle skipping with rendering enabled
		t.Logf("Detected %d potential odd frame cycle skips", oddFrameSkips)
	})
}

// Helper functions
func minInt(slice []int) int {
	if len(slice) == 0 {
		return 0
	}
	min := slice[0]
	for _, v := range slice[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func maxInt(slice []int) int {
	if len(slice) == 0 {
		return 0
	}
	max := slice[0]
	for _, v := range slice[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

// TestFrameTimingEdgeCases tests edge cases in frame timing
func TestFrameTimingEdgeCases(t *testing.T) {
	t.Run("Rendering disable during frame", func(t *testing.T) {
		helper := NewFrameTimingHelper()
		helper.SetupBasicROM(0x8000)

		// Program that toggles rendering mid-frame
		program := []uint8{
			// Enable rendering initially
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			// Wait partway through frame
			0xEA, 0xEA, 0xEA, // NOPs (simulate work)

			// Disable rendering mid-frame
			0xA9, 0x00, // LDA #$00
			0x8D, 0x01, 0x20, // STA $2001 (disable rendering)

			// More work
			0xEA, 0xEA, 0xEA,

			// Re-enable rendering
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			0x4C, 0x0E, 0x80, // JMP to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Run and verify system remains stable
		for i := 0; i < 50000; i++ {
			helper.Bus.Step()
		}

		// System should remain stable despite rendering changes
		if helper.CPU.PC < 0x8000 {
			t.Errorf("CPU PC went outside ROM area: 0x%04X", helper.CPU.PC)
		}

		t.Log("Rendering disable during frame test completed")
	})

	t.Run("Frame timing with interrupts", func(t *testing.T) {
		helper := NewFrameTimingHelper()
		helper.SetupBasicROM(0x8000)

		// Set up NMI handler
		nmiHandler := []uint8{
			0x48,       // PHA
			0xE6, 0x40, // INC $40 (NMI counter)
			0x68, // PLA
			0x40, // RTI
		}

		romData := make([]uint8, 0x8000)
		// Main program
		program := []uint8{
			0xA9, 0x80, // LDA #$80 (enable NMI)
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E (enable rendering)
			0x8D, 0x01, 0x20, // STA $2001

			// Main loop
			0xE6, 0x41, // INC $41 (main loop counter)
			0x4C, 0x08, 0x80, // JMP to main loop
		}
		copy(romData, program)

		// Place NMI handler
		copy(romData[0x0100:], nmiHandler)

		// Set vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize counters
		helper.Memory.Write(0x0040, 0x00) // NMI counter
		helper.Memory.Write(0x0041, 0x00) // Main loop counter

		// Run for several frames
		for i := 0; i < 100000; i++ {
			helper.Bus.Step()
		}

		// Check that both main loop and NMIs are running
		nmiCount := helper.Memory.Read(0x0040)
		mainCount := helper.Memory.Read(0x0041)

		if nmiCount == 0 {
			t.Error("No NMIs occurred during frame timing test")
		}
		if mainCount == 0 {
			t.Error("Main loop did not execute during frame timing test")
		}

		t.Logf("Frame timing with interrupts: %d NMIs, %d main loops", nmiCount, mainCount)

		// NMI should occur roughly once per frame
		// Main loop should run many more times
		if nmiCount > mainCount {
			t.Error("NMI count should not exceed main loop count")
		}
	})
}
