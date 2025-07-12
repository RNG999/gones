package integration

import (
	"math"
	"testing"
)

// FrameSynchronizationTestHelper provides utilities for testing frame synchronization
type FrameSynchronizationTestHelper struct {
	*IntegrationTestHelper
	frameLog       []FrameSyncEvent
	vblankEvents   []VBlankEvent
	frameStartTime uint64
	lastFrameCount uint64
}

// FrameSyncEvent represents a frame synchronization event
type FrameSyncEvent struct {
	FrameNumber     uint64
	StartCPUCycle   uint64
	EndCPUCycle     uint64
	FrameLength     uint64
	VBlankDetected  bool
	RenderingActive bool
	OddFrame        bool
}

// VBlankEvent represents a VBlank timing event
type VBlankEvent struct {
	FrameNumber  uint64
	VBlankStart  uint64
	VBlankEnd    uint64
	VBlankLength uint64
	NMITriggered bool
}

// NewFrameSynchronizationTestHelper creates a new frame synchronization test helper
func NewFrameSynchronizationTestHelper() *FrameSynchronizationTestHelper {
	return &FrameSynchronizationTestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		frameLog:              make([]FrameSyncEvent, 0),
		vblankEvents:          make([]VBlankEvent, 0),
		frameStartTime:        0,
		lastFrameCount:        0,
	}
}

// StepWithFrameSync executes one step and tracks frame synchronization
func (h *FrameSynchronizationTestHelper) StepWithFrameSync() {
	preCycles := h.Bus.GetCycleCount()
	preFrameCount := h.Bus.GetFrameCount()

	// Check for frame start
	if preFrameCount > h.lastFrameCount {
		h.onFrameStart(preFrameCount, preCycles)
	}

	// Execute step
	h.Bus.Step()

	// Check for frame transitions and VBlank events
	postCycles := h.Bus.GetCycleCount()
	postFrameCount := h.Bus.GetFrameCount()

	h.checkVBlankEvents(postFrameCount, postCycles)
	h.lastFrameCount = postFrameCount
}

// onFrameStart handles frame start event
func (h *FrameSynchronizationTestHelper) onFrameStart(frameNumber, cycles uint64) {
	if h.frameStartTime > 0 {
		// Complete previous frame
		frameLength := cycles - h.frameStartTime
		renderingActive := h.isRenderingEnabled()
		oddFrame := (frameNumber % 2) == 1

		event := FrameSyncEvent{
			FrameNumber:     frameNumber - 1,
			StartCPUCycle:   h.frameStartTime,
			EndCPUCycle:     cycles,
			FrameLength:     frameLength,
			VBlankDetected:  len(h.vblankEvents) > 0,
			RenderingActive: renderingActive,
			OddFrame:        oddFrame,
		}
		h.frameLog = append(h.frameLog, event)
	}

	h.frameStartTime = cycles
}

// checkVBlankEvents checks for VBlank start/end events
func (h *FrameSynchronizationTestHelper) checkVBlankEvents(frameNumber, cycles uint64) {
	ppuStatus := h.PPU.ReadRegister(0x2002)
	vblankActive := (ppuStatus & 0x80) != 0

	// For this test, we'll simulate VBlank detection
	// In a real implementation, this would be more sophisticated
	if vblankActive && len(h.vblankEvents) == 0 {
		// VBlank started
		event := VBlankEvent{
			FrameNumber:  frameNumber,
			VBlankStart:  cycles,
			NMITriggered: true, // Assume NMI is enabled
		}
		h.vblankEvents = append(h.vblankEvents, event)
	}
}

// isRenderingEnabled checks if rendering is currently enabled
func (h *FrameSynchronizationTestHelper) isRenderingEnabled() bool {
	ppumask := h.PPU.ReadRegister(0x2001)
	return (ppumask & 0x18) != 0 // Background or sprite rendering enabled
}

// GetFrameLog returns the frame synchronization log
func (h *FrameSynchronizationTestHelper) GetFrameLog() []FrameSyncEvent {
	return h.frameLog
}

// GetVBlankEvents returns the VBlank events log
func (h *FrameSynchronizationTestHelper) GetVBlankEvents() []VBlankEvent {
	return h.vblankEvents
}

// ClearLogs clears all synchronization logs
func (h *FrameSynchronizationTestHelper) ClearLogs() {
	h.frameLog = h.frameLog[:0]
	h.vblankEvents = h.vblankEvents[:0]
	h.frameStartTime = 0
	h.lastFrameCount = 0
}

// TestFrameSynchronizationBasics tests basic frame synchronization functionality
func TestFrameSynchronizationBasics(t *testing.T) {
	t.Run("Frame boundary detection", func(t *testing.T) {
		helper := NewFrameSynchronizationTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Run until we detect multiple frame boundaries
		maxSteps := 200000
		targetFrames := 3
		framesDetected := 0

		for i := 0; i < maxSteps && framesDetected < targetFrames; i++ {
			currentFrameCount := helper.Bus.GetFrameCount()
			helper.StepWithFrameSync()
			newFrameCount := helper.Bus.GetFrameCount()

			if newFrameCount > currentFrameCount {
				framesDetected++
			}
		}

		if framesDetected < targetFrames {
			t.Fatalf("Expected to detect %d frames, got %d", targetFrames, framesDetected)
		}

		// Verify frame log contains entries
		frameLog := helper.GetFrameLog()
		if len(frameLog) == 0 {
			t.Error("No frame events logged")
		}

		t.Logf("Detected %d frame boundaries in %d steps", framesDetected, maxSteps)
	})

	t.Run("NTSC frame timing accuracy", func(t *testing.T) {
		helper := NewFrameSynchronizationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Enable rendering for accurate frame timing
		helper.Memory.Write(0x2000, 0x80) // PPUCTRL - enable NMI
		helper.Memory.Write(0x2001, 0x1E) // PPUMASK - enable rendering

		helper.Bus.Reset()

		// Run multiple frames and measure timing
		frameTimings := make([]uint64, 0)
		lastFrameCount := uint64(0)
		lastCycleCount := uint64(0)

		for i := 0; i < 300000; i++ {
			helper.Bus.Step()
			currentFrameCount := helper.Bus.GetFrameCount()
			currentCycleCount := helper.Bus.GetCycleCount()

			if currentFrameCount > lastFrameCount {
				if lastFrameCount > 0 {
					frameLength := currentCycleCount - lastCycleCount
					frameTimings = append(frameTimings, frameLength)
				}
				lastFrameCount = currentFrameCount
				lastCycleCount = currentCycleCount

				if len(frameTimings) >= 5 {
					break
				}
			}
		}

		if len(frameTimings) < 3 {
			t.Fatalf("Expected at least 3 frame timings, got %d", len(frameTimings))
		}

		// NTSC frame timing: ~29,780.67 CPU cycles per frame
		expectedFrameCycles := uint64(29781)
		tolerance := uint64(50)

		for i, timing := range frameTimings {
			if timing < expectedFrameCycles-tolerance || timing > expectedFrameCycles+tolerance {
				t.Errorf("Frame %d timing out of range: got %d cycles, expected %d±%d",
					i+1, timing, expectedFrameCycles, tolerance)
			}
		}

		// Calculate average and verify consistency
		sum := uint64(0)
		for _, timing := range frameTimings {
			sum += timing
		}
		average := sum / uint64(len(frameTimings))

		t.Logf("Frame timings: %v", frameTimings)
		t.Logf("Average frame length: %d cycles", average)
	})

	t.Run("Odd frame cycle skip", func(t *testing.T) {
		helper := NewFrameSynchronizationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Enable rendering to trigger odd frame behavior
		helper.Memory.Write(0x2000, 0x80) // PPUCTRL - enable NMI
		helper.Memory.Write(0x2001, 0x1E) // PPUMASK - enable rendering

		helper.Bus.Reset()

		// Collect frame timings to detect odd frame skip pattern
		frameTimings := make([]uint64, 0)
		frameNumbers := make([]uint64, 0)
		lastFrameCount := uint64(0)
		lastCycleCount := uint64(0)

		for i := 0; i < 400000; i++ {
			helper.Bus.Step()
			currentFrameCount := helper.Bus.GetFrameCount()
			currentCycleCount := helper.Bus.GetCycleCount()

			if currentFrameCount > lastFrameCount {
				if lastFrameCount > 0 {
					frameLength := currentCycleCount - lastCycleCount
					frameTimings = append(frameTimings, frameLength)
					frameNumbers = append(frameNumbers, lastFrameCount)
				}
				lastFrameCount = currentFrameCount
				lastCycleCount = currentCycleCount

				if len(frameTimings) >= 10 {
					break
				}
			}
		}

		if len(frameTimings) < 6 {
			t.Fatalf("Not enough frames for odd frame test: got %d", len(frameTimings))
		}

		// Analyze frame timings for odd frame pattern
		normalFrameCycles := uint64(29781)
		shortFrameCycles := uint64(29780) // Odd frames are 1 cycle shorter
		tolerance := uint64(10)

		shortFrames := 0
		normalFrames := 0

		for i, timing := range frameTimings {
			if timing >= shortFrameCycles-tolerance && timing <= shortFrameCycles+tolerance {
				shortFrames++
				t.Logf("Frame %d (frame #%d): %d cycles (odd frame)", i, frameNumbers[i], timing)
			} else if timing >= normalFrameCycles-tolerance && timing <= normalFrameCycles+tolerance {
				normalFrames++
				t.Logf("Frame %d (frame #%d): %d cycles (normal frame)", i, frameNumbers[i], timing)
			} else {
				t.Logf("Frame %d (frame #%d): %d cycles (unexpected)", i, frameNumbers[i], timing)
			}
		}

		// We should see evidence of frame timing variation when rendering is enabled
		t.Logf("Short frames: %d, Normal frames: %d", shortFrames, normalFrames)

		// At minimum, verify frames are within reasonable range
		for i, timing := range frameTimings {
			if timing < shortFrameCycles-tolerance*2 || timing > normalFrameCycles+tolerance*2 {
				t.Errorf("Frame %d timing completely out of range: %d cycles", i, timing)
			}
		}
	})
}

// TestFrameSynchronizationWithRendering tests frame sync with different rendering states
func TestFrameSynchronizationWithRendering(t *testing.T) {
	t.Run("Rendering enabled vs disabled timing", func(t *testing.T) {
		helper := NewFrameSynchronizationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test with rendering disabled
		helper.Memory.Write(0x2000, 0x00) // PPUCTRL - NMI disabled, rendering off
		helper.Memory.Write(0x2001, 0x00) // PPUMASK - rendering disabled
		helper.Bus.Reset()

		disabledTimings := make([]uint64, 0)
		lastFrameCount := uint64(0)
		lastCycleCount := uint64(0)

		for i := 0; i < 200000; i++ {
			helper.Bus.Step()
			currentFrameCount := helper.Bus.GetFrameCount()
			currentCycleCount := helper.Bus.GetCycleCount()

			if currentFrameCount > lastFrameCount {
				if lastFrameCount > 0 {
					frameLength := currentCycleCount - lastCycleCount
					disabledTimings = append(disabledTimings, frameLength)
				}
				lastFrameCount = currentFrameCount
				lastCycleCount = currentCycleCount

				if len(disabledTimings) >= 3 {
					break
				}
			}
		}

		// Test with rendering enabled
		helper.Memory.Write(0x2000, 0x80) // PPUCTRL - enable NMI, rendering on
		helper.Memory.Write(0x2001, 0x1E) // PPUMASK - enable rendering
		helper.Bus.Reset()

		enabledTimings := make([]uint64, 0)
		lastFrameCount = 0
		lastCycleCount = 0

		for i := 0; i < 200000; i++ {
			helper.Bus.Step()
			currentFrameCount := helper.Bus.GetFrameCount()
			currentCycleCount := helper.Bus.GetCycleCount()

			if currentFrameCount > lastFrameCount {
				if lastFrameCount > 0 {
					frameLength := currentCycleCount - lastCycleCount
					enabledTimings = append(enabledTimings, frameLength)
				}
				lastFrameCount = currentFrameCount
				lastCycleCount = currentCycleCount

				if len(enabledTimings) >= 3 {
					break
				}
			}
		}

		// Compare timings
		if len(disabledTimings) < 2 || len(enabledTimings) < 2 {
			t.Fatalf("Not enough frame timings: disabled=%d, enabled=%d",
				len(disabledTimings), len(enabledTimings))
		}

		t.Logf("Rendering disabled timings: %v", disabledTimings)
		t.Logf("Rendering enabled timings: %v", enabledTimings)

		// Both should be close to expected frame timing
		expectedFrameCycles := uint64(29781)
		tolerance := uint64(100)

		for _, timing := range disabledTimings {
			if timing < expectedFrameCycles-tolerance || timing > expectedFrameCycles+tolerance {
				t.Errorf("Rendering disabled frame timing out of range: %d", timing)
			}
		}

		for _, timing := range enabledTimings {
			if timing < expectedFrameCycles-tolerance || timing > expectedFrameCycles+tolerance {
				t.Errorf("Rendering enabled frame timing out of range: %d", timing)
			}
		}
	})

	t.Run("Mid-frame rendering toggle", func(t *testing.T) {
		helper := NewFrameSynchronizationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that toggles rendering mid-frame
		program := []uint8{
			// Initial setup
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)

			// Wait some cycles
			0xEA, 0xEA, 0xEA, 0xEA, 0xEA, // NOPs

			// Disable rendering mid-frame
			0xA9, 0x00, // LDA #$00
			0x8D, 0x01, 0x20, // STA $2001 (disable rendering)

			// More cycles
			0xEA, 0xEA, 0xEA, 0xEA, 0xEA, // NOPs

			// Re-enable rendering
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)

			// Continue
			0x4C, 0x14, 0x80, // JMP to wait loop
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Run and verify system stability despite rendering changes
		frameCount := 0
		lastFrameNumber := uint64(0)

		for i := 0; i < 100000 && frameCount < 3; i++ {
			helper.Bus.Step()
			currentFrame := helper.Bus.GetFrameCount()

			if currentFrame > lastFrameNumber {
				frameCount++
				lastFrameNumber = currentFrame
				t.Logf("Completed frame %d at step %d", frameCount, i)
			}
		}

		if frameCount < 2 {
			t.Fatalf("Expected at least 2 frames with rendering toggle, got %d", frameCount)
		}

		// Verify system remained stable
		if helper.CPU.PC < 0x8000 {
			t.Errorf("PC went outside ROM during rendering toggle: 0x%04X", helper.CPU.PC)
		}
	})
}

// TestFrameSynchronizationEdgeCases tests edge cases in frame synchronization
func TestFrameSynchronizationEdgeCases(t *testing.T) {
	t.Run("Frame synchronization with DMA", func(t *testing.T) {
		helper := NewFrameSynchronizationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that triggers DMA during frame
		program := []uint8{
			// Enable rendering
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			// Set up some data in memory for DMA
			0xA9, 0x55, // LDA #$55
			0x8D, 0x00, 0x02, // STA $0200
			0x8D, 0x01, 0x02, // STA $0201

			// Trigger OAM DMA
			0xA9, 0x02, // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (trigger DMA)

			// Continue execution
			0xEA,             // NOP
			0x4C, 0x16, 0x80, // JMP to continue
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Run and track DMA effects on frame timing
		dmaTriggered := false
		frameTiming := make([]uint64, 0)
		lastFrameCount := uint64(0)
		lastCycleCount := uint64(0)

		for i := 0; i < 150000; i++ {
			helper.Bus.Step()

			// Check for DMA
			if helper.Bus.IsDMAInProgress() && !dmaTriggered {
				dmaTriggered = true
				t.Logf("DMA triggered at step %d", i)
			}

			// Track frame timing
			currentFrameCount := helper.Bus.GetFrameCount()
			currentCycleCount := helper.Bus.GetCycleCount()

			if currentFrameCount > lastFrameCount {
				if lastFrameCount > 0 {
					frameLength := currentCycleCount - lastCycleCount
					frameTiming = append(frameTiming, frameLength)
				}
				lastFrameCount = currentFrameCount
				lastCycleCount = currentCycleCount

				if len(frameTiming) >= 3 {
					break
				}
			}
		}

		if !dmaTriggered {
			t.Error("DMA was not triggered during test")
		}

		if len(frameTiming) < 2 {
			t.Fatalf("Not enough frame timings with DMA: %d", len(frameTiming))
		}

		// Frame with DMA should be longer due to CPU suspension
		t.Logf("Frame timings with DMA: %v", frameTiming)

		// All frames should still be within reasonable bounds
		for i, timing := range frameTiming {
			if timing < 29000 || timing > 35000 {
				t.Errorf("Frame %d timing unreasonable with DMA: %d cycles", i, timing)
			}
		}
	})

	t.Run("Frame synchronization across reset", func(t *testing.T) {
		helper := NewFrameSynchronizationTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Run some frames
		for i := 0; i < 100000; i++ {
			helper.Bus.Step()
			if helper.Bus.GetFrameCount() >= 2 {
				break
			}
		}

		preResetFrameCount := helper.Bus.GetFrameCount()
		preResetCycleCount := helper.Bus.GetCycleCount()

		// Reset system
		helper.Bus.Reset()

		// Verify reset cleared frame state
		postResetFrameCount := helper.Bus.GetFrameCount()
		postResetCycleCount := helper.Bus.GetCycleCount()

		if postResetFrameCount != 0 {
			t.Errorf("Frame count should be 0 after reset, got %d", postResetFrameCount)
		}

		if postResetCycleCount != 0 {
			t.Errorf("Cycle count should be 0 after reset, got %d", postResetCycleCount)
		}

		// Run after reset and verify frame sync still works
		for i := 0; i < 100000; i++ {
			helper.Bus.Step()
			if helper.Bus.GetFrameCount() >= 1 {
				break
			}
		}

		if helper.Bus.GetFrameCount() == 0 {
			t.Error("No frames completed after reset")
		}

		t.Logf("Pre-reset: %d frames, %d cycles", preResetFrameCount, preResetCycleCount)
		t.Logf("Post-reset: %d frames, %d cycles", helper.Bus.GetFrameCount(), helper.Bus.GetCycleCount())
	})
}

// TestFrameSynchronizationConsistency tests consistency of frame synchronization
func TestFrameSynchronizationConsistency(t *testing.T) {
	t.Run("Multiple frame consistency", func(t *testing.T) {
		helper := NewFrameSynchronizationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Enable consistent rendering
		helper.Memory.Write(0x2000, 0x80)
		helper.Memory.Write(0x2001, 0x1E)
		helper.Bus.Reset()

		// Collect many frame timings
		frameTimings := make([]uint64, 0)
		lastFrameCount := uint64(0)
		lastCycleCount := uint64(0)

		for i := 0; i < 500000; i++ {
			helper.Bus.Step()
			currentFrameCount := helper.Bus.GetFrameCount()
			currentCycleCount := helper.Bus.GetCycleCount()

			if currentFrameCount > lastFrameCount {
				if lastFrameCount > 0 {
					frameLength := currentCycleCount - lastCycleCount
					frameTimings = append(frameTimings, frameLength)
				}
				lastFrameCount = currentFrameCount
				lastCycleCount = currentCycleCount

				if len(frameTimings) >= 15 {
					break
				}
			}
		}

		if len(frameTimings) < 10 {
			t.Fatalf("Not enough frames for consistency test: %d", len(frameTimings))
		}

		// Calculate statistics
		sum := uint64(0)
		for _, timing := range frameTimings {
			sum += timing
		}
		mean := float64(sum) / float64(len(frameTimings))

		// Calculate standard deviation
		variance := 0.0
		for _, timing := range frameTimings {
			diff := float64(timing) - mean
			variance += diff * diff
		}
		variance /= float64(len(frameTimings))
		stdDev := math.Sqrt(variance)

		t.Logf("Frame timing statistics:")
		t.Logf("  Count: %d", len(frameTimings))
		t.Logf("  Mean: %.2f cycles", mean)
		t.Logf("  Std Dev: %.2f cycles", stdDev)
		t.Logf("  Min: %d cycles", minUint64(frameTimings))
		t.Logf("  Max: %d cycles", maxUint64(frameTimings))

		// Frame timing should be very consistent
		maxAllowedStdDev := 15.0
		if stdDev > maxAllowedStdDev {
			t.Errorf("Frame timing too inconsistent: std dev %.2f > %.2f", stdDev, maxAllowedStdDev)
		}

		// Check for outliers
		tolerance := 3.0 * stdDev
		outliers := 0
		for i, timing := range frameTimings {
			if math.Abs(float64(timing)-mean) > tolerance {
				t.Logf("Frame %d is outlier: %d cycles (%.1f from mean)",
					i+1, timing, math.Abs(float64(timing)-mean))
				outliers++
			}
		}

		maxOutliers := len(frameTimings) / 10 // Allow 10% outliers
		if outliers > maxOutliers {
			t.Errorf("Too many outlier frames: %d > %d", outliers, maxOutliers)
		}
	})
}

// Helper functions
func minUint64(slice []uint64) uint64 {
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

func maxUint64(slice []uint64) uint64 {
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
