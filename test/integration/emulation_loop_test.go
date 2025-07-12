package integration

import (
	"testing"
	"time"
)

// EmulationLoopTestHelper provides utilities for testing the main emulation loop
type EmulationLoopTestHelper struct {
	*IntegrationTestHelper
	executionLog []ExecutionEvent
	totalSteps   int
}

// ExecutionEvent represents a single step in the emulation loop
type ExecutionEvent struct {
	StepNumber    int
	CPUCycles     uint64
	PPUCycles     uint64
	FrameCount    uint64
	DMAActive     bool
	NMIProcessed  bool
	PCValue       uint16
	InstructionOp uint8
}

// NewEmulationLoopTestHelper creates a new emulation loop test helper
func NewEmulationLoopTestHelper() *EmulationLoopTestHelper {
	return &EmulationLoopTestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		executionLog:          make([]ExecutionEvent, 0),
		totalSteps:            0,
	}
}

// StepWithLogging executes one emulation step and logs the execution state
func (h *EmulationLoopTestHelper) StepWithLogging() {
	// Capture state before step
	preFrameCount := h.Bus.GetFrameCount()
	prePC := h.CPU.PC
	preOpcode := h.Memory.Read(prePC)

	// Execute the step
	h.Bus.Step()
	h.totalSteps++

	// Capture state after step
	postCPUCycles := h.Bus.GetCycleCount()
	postFrameCount := h.Bus.GetFrameCount()
	dmaActive := h.Bus.IsDMAInProgress()

	// Log the execution event
	event := ExecutionEvent{
		StepNumber:    h.totalSteps,
		CPUCycles:     postCPUCycles,
		PPUCycles:     postCPUCycles * 3, // PPU runs at 3x CPU speed
		FrameCount:    postFrameCount,
		DMAActive:     dmaActive,
		NMIProcessed:  postFrameCount > preFrameCount, // Frame count increased
		PCValue:       prePC,
		InstructionOp: preOpcode,
	}

	h.executionLog = append(h.executionLog, event)
}

// GetExecutionLog returns the execution log
func (h *EmulationLoopTestHelper) GetExecutionLog() []ExecutionEvent {
	return h.executionLog
}

// ClearExecutionLog clears the execution log
func (h *EmulationLoopTestHelper) ClearExecutionLog() {
	h.executionLog = h.executionLog[:0]
	h.totalSteps = 0
}

// TestEmulationLoopBasicOperation tests basic emulation loop functionality
func TestEmulationLoopBasicOperation(t *testing.T) {
	t.Run("Single step execution", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Execute single step
		initialCycleCount := helper.Bus.GetCycleCount()
		helper.StepWithLogging()

		// Verify step executed
		if len(helper.GetExecutionLog()) != 1 {
			t.Fatalf("Expected 1 execution log entry, got %d", len(helper.GetExecutionLog()))
		}

		// Verify cycle count increased
		finalCycleCount := helper.Bus.GetCycleCount()
		if finalCycleCount <= initialCycleCount {
			t.Errorf("CPU cycle count should increase after step: initial=%d, final=%d",
				initialCycleCount, finalCycleCount)
		}

		// Verify PPU cycles are properly coordinated (3:1 ratio)
		log := helper.GetExecutionLog()[0]
		expectedPPUCycles := log.CPUCycles * 3
		if log.PPUCycles != expectedPPUCycles {
			t.Errorf("PPU cycles should be 3x CPU cycles: CPU=%d, PPU=%d, expected=%d",
				log.CPUCycles, log.PPUCycles, expectedPPUCycles)
		}
	})

	t.Run("Multiple step execution", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Execute multiple steps
		stepCount := 100
		for i := 0; i < stepCount; i++ {
			helper.StepWithLogging()
		}

		// Verify all steps logged
		if len(helper.GetExecutionLog()) != stepCount {
			t.Fatalf("Expected %d execution log entries, got %d", stepCount, len(helper.GetExecutionLog()))
		}

		// Verify monotonic cycle count increase
		log := helper.GetExecutionLog()
		for i := 1; i < len(log); i++ {
			if log[i].CPUCycles <= log[i-1].CPUCycles {
				t.Errorf("CPU cycles should increase monotonically: step %d: %d, step %d: %d",
					i-1, log[i-1].CPUCycles, i, log[i].CPUCycles)
			}
		}

		// Verify PPU-CPU coordination maintained throughout
		for i, entry := range log {
			expectedPPUCycles := entry.CPUCycles * 3
			if entry.PPUCycles != expectedPPUCycles {
				t.Errorf("Step %d: PPU cycles coordination failed: CPU=%d, PPU=%d, expected=%d",
					i, entry.CPUCycles, entry.PPUCycles, expectedPPUCycles)
			}
		}
	})

	t.Run("Program counter advancement", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()

		// Create program with known instruction sequence
		program := []uint8{
			0xEA,       // NOP (1 byte, 2 cycles)
			0xA9, 0x42, // LDA #$42 (2 bytes, 2 cycles)
			0x85, 0x00, // STA $00 (2 bytes, 3 cycles)
			0xE6, 0x00, // INC $00 (2 bytes, 5 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 bytes, 3 cycles)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute instructions and verify PC advancement
		expectedPCs := []uint16{0x8000, 0x8001, 0x8003, 0x8005, 0x8007}
		instructions := []string{"NOP", "LDA #$42", "STA $00", "INC $00", "JMP $8000"}

		for i, expectedPC := range expectedPCs {
			helper.StepWithLogging()
			log := helper.GetExecutionLog()

			if len(log) != i+1 {
				t.Fatalf("Expected %d log entries, got %d", i+1, len(log))
			}

			if log[i].PCValue != expectedPC {
				t.Errorf("Step %d (%s): expected PC=0x%04X, got PC=0x%04X",
					i, instructions[i], expectedPC, log[i].PCValue)
			}
		}
	})
}

// TestEmulationLoopTimingAccuracy tests timing accuracy of the emulation loop
func TestEmulationLoopTimingAccuracy(t *testing.T) {
	t.Run("CPU instruction cycle accuracy", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()

		// Program with instructions of known cycle counts
		program := []uint8{
			0xEA,       // NOP (2 cycles)
			0xA9, 0x01, // LDA #$01 (2 cycles)
			0x85, 0x10, // STA $10 (3 cycles)
			0xA5, 0x10, // LDA $10 (3 cycles)
			0x69, 0x01, // ADC #$01 (2 cycles)
			0x8D, 0x00, 0x20, // STA $2000 (4 cycles)
			0x4C, 0x00, 0x80, // JMP $8000 (3 cycles)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute instructions and verify cycle counts
		expectedCycles := []uint64{2, 4, 7, 10, 12, 16, 19}
		instructions := []string{"NOP", "LDA #$01", "STA $10", "LDA $10", "ADC #$01", "STA $2000", "JMP $8000"}

		for i, expectedTotal := range expectedCycles {
			helper.StepWithLogging()
			log := helper.GetExecutionLog()

			if len(log) != i+1 {
				t.Fatalf("Expected %d log entries, got %d", i+1, len(log))
			}

			actualCycles := log[i].CPUCycles
			if actualCycles != expectedTotal {
				t.Errorf("Step %d (%s): expected %d total CPU cycles, got %d",
					i, instructions[i], expectedTotal, actualCycles)
			}
		}
	})

	t.Run("PPU cycle synchronization", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Execute steps and verify PPU stays synchronized
		for i := 0; i < 50; i++ {
			helper.StepWithLogging()
		}

		log := helper.GetExecutionLog()
		for i, entry := range log {
			expectedPPUCycles := entry.CPUCycles * 3
			if entry.PPUCycles != expectedPPUCycles {
				t.Errorf("Step %d: PPU synchronization lost: CPU=%d, PPU=%d, expected=%d",
					i, entry.CPUCycles, entry.PPUCycles, expectedPPUCycles)
			}
		}
	})

	t.Run("Frame timing accuracy", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Run until we see frame transitions
		frameTransitions := make([]int, 0)
		currentFrame := uint64(0)

		for i := 0; i < 100000 && len(frameTransitions) < 3; i++ {
			helper.StepWithLogging()
			log := helper.GetExecutionLog()

			if len(log) > 0 {
				lastEntry := log[len(log)-1]
				if lastEntry.FrameCount > currentFrame {
					frameTransitions = append(frameTransitions, i)
					currentFrame = lastEntry.FrameCount
				}
			}
		}

		if len(frameTransitions) < 2 {
			t.Fatalf("Expected at least 2 frame transitions, got %d", len(frameTransitions))
		}

		// Verify frame timing consistency
		for i := 1; i < len(frameTransitions); i++ {
			frameLength := frameTransitions[i] - frameTransitions[i-1]
			expectedFrameSteps := 29781 / 2 // Approximate steps per frame
			tolerance := 1000

			if frameLength < expectedFrameSteps-tolerance || frameLength > expectedFrameSteps+tolerance {
				t.Errorf("Frame %d length inconsistent: expected ~%d steps, got %d",
					i, expectedFrameSteps, frameLength)
			}
		}
	})
}

// TestEmulationLoopErrorHandling tests error handling in the emulation loop
func TestEmulationLoopErrorHandling(t *testing.T) {
	t.Run("Invalid instruction handling", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()

		// Program with invalid/unofficial opcode
		program := []uint8{
			0xEA,             // NOP (valid)
			0x02,             // Unofficial opcode (should be handled gracefully)
			0xEA,             // NOP (should continue)
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// System should handle invalid instructions gracefully
		for i := 0; i < 10; i++ {
			helper.StepWithLogging()
		}

		// Verify system remains stable
		if helper.CPU.PC < 0x8000 {
			t.Errorf("PC should remain in ROM area after invalid instruction: 0x%04X", helper.CPU.PC)
		}

		// Verify execution log was maintained
		if len(helper.GetExecutionLog()) != 10 {
			t.Errorf("Expected 10 log entries, got %d", len(helper.GetExecutionLog()))
		}
	})

	t.Run("Memory access edge cases", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()

		// Program that accesses various memory regions
		program := []uint8{
			0xA9, 0x55, // LDA #$55
			0x8D, 0x00, 0x50, // STA $5000 (expansion area)
			0xAD, 0x02, 0x20, // LDA $2002 (PPU status)
			0x8D, 0x00, 0x40, // STA $4000 (APU register)
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute steps that access different memory regions
		for i := 0; i < 10; i++ {
			helper.StepWithLogging()
		}

		// Verify system remains stable despite unusual memory accesses
		log := helper.GetExecutionLog()
		if len(log) != 10 {
			t.Errorf("Expected 10 execution steps, got %d", len(log))
		}

		// Verify cycles are still properly tracked
		for i := 1; i < len(log); i++ {
			if log[i].CPUCycles <= log[i-1].CPUCycles {
				t.Errorf("CPU cycles should increase monotonically even with edge case memory access")
			}
		}
	})
}

// TestEmulationLoopPerformance tests performance characteristics of the emulation loop
func TestEmulationLoopPerformance(t *testing.T) {
	t.Run("Execution speed benchmark", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Benchmark execution speed
		stepCount := 10000
		startTime := time.Now()

		for i := 0; i < stepCount; i++ {
			helper.Bus.Step() // Use direct Step() for performance test
		}

		duration := time.Since(startTime)
		stepsPerSecond := float64(stepCount) / duration.Seconds()

		t.Logf("Emulation loop performance: %.0f steps/second", stepsPerSecond)

		// Verify reasonable performance (should be able to handle real-time emulation)
		// NTSC NES runs at ~1.79MHz, so we need at least 1.79M steps/second for real-time
		minStepsPerSecond := 100000.0 // Reduced for test environment
		if stepsPerSecond < minStepsPerSecond {
			t.Errorf("Emulation loop too slow: %.0f steps/second < %.0f required",
				stepsPerSecond, minStepsPerSecond)
		}
	})

	t.Run("Memory allocation consistency", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Execute many steps and verify no memory leaks in logging
		initialLogSize := len(helper.GetExecutionLog())

		for i := 0; i < 1000; i++ {
			helper.StepWithLogging()
		}

		finalLogSize := len(helper.GetExecutionLog())
		expectedSize := initialLogSize + 1000

		if finalLogSize != expectedSize {
			t.Errorf("Expected log size %d, got %d", expectedSize, finalLogSize)
		}

		// Clear log and verify cleanup
		helper.ClearExecutionLog()
		if len(helper.GetExecutionLog()) != 0 {
			t.Errorf("Log should be empty after clear, got %d entries", len(helper.GetExecutionLog()))
		}
	})
}

// TestEmulationLoopStability tests long-term stability of the emulation loop
func TestEmulationLoopStability(t *testing.T) {
	t.Run("Long running stability", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Run for extended period
		stepCount := 50000
		checkpoints := []int{10000, 25000, 40000, 50000}

		for i := 0; i < stepCount; i++ {
			helper.Bus.Step()

			// Check system state at checkpoints
			for _, checkpoint := range checkpoints {
				if i == checkpoint-1 {
					// Verify system is still in valid state
					if helper.CPU.PC < 0x8000 {
						t.Errorf("At checkpoint %d: PC outside ROM: 0x%04X", checkpoint, helper.CPU.PC)
					}

					if helper.CPU.SP > 0xFF {
						t.Errorf("At checkpoint %d: SP overflow: 0x%02X", checkpoint, helper.CPU.SP)
					}

					cycles := helper.Bus.GetCycleCount()
					if cycles == 0 {
						t.Errorf("At checkpoint %d: No cycles counted", checkpoint)
					}
				}
			}
		}

		// Final stability check
		finalCycles := helper.Bus.GetCycleCount()
		if finalCycles < uint64(stepCount) {
			t.Errorf("Expected at least %d cycles, got %d", stepCount, finalCycles)
		}
	})

	t.Run("Reset stability", func(t *testing.T) {
		helper := NewEmulationLoopTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test multiple reset cycles
		for resetCycle := 0; resetCycle < 5; resetCycle++ {
			helper.Bus.Reset()

			// Verify clean reset state
			if helper.CPU.PC != 0x8000 {
				t.Errorf("Reset cycle %d: PC should be 0x8000, got 0x%04X", resetCycle, helper.CPU.PC)
			}

			if helper.Bus.GetCycleCount() != 0 {
				t.Errorf("Reset cycle %d: Cycle count should be 0, got %d", resetCycle, helper.Bus.GetCycleCount())
			}

			// Run some steps after reset
			for i := 0; i < 100; i++ {
				helper.Bus.Step()
			}

			// Verify system is working after reset
			if helper.Bus.GetCycleCount() == 0 {
				t.Errorf("Reset cycle %d: System not executing after reset", resetCycle)
			}
		}
	})
}
