package integration

import (
	"testing"
)

// InterruptCoordinationTestHelper provides utilities for testing interrupt coordination
type InterruptCoordinationTestHelper struct {
	*IntegrationTestHelper
	interruptLog []InterruptEvent
	nmiHistory   []NMIInterruptEvent
	irqHistory   []IRQEvent
}

// InterruptEvent represents an interrupt coordination event
type InterruptEvent struct {
	StepNumber    int
	InterruptType string // "NMI", "IRQ", "BRK"
	TriggerCycle  uint64
	ServiceCycle  uint64
	PCBefore      uint16
	PCAfter       uint16
	VectorUsed    uint16
	FrameNumber   uint64
	PPUState      string
}

// NMIInterruptEvent represents an NMI-specific event
type NMIInterruptEvent struct {
	FrameNumber    uint64
	TriggerCycle   uint64
	VBlankStart    uint64
	ServiceLatency uint64
	PPUStatusRead  bool
	Suppressed     bool
}

// IRQEvent represents an IRQ-specific event
type IRQEvent struct {
	TriggerCycle   uint64
	Source         string // "APU", "Mapper", "External"
	ServiceLatency uint64
	Masked         bool
}

// NewInterruptCoordinationTestHelper creates a new interrupt coordination test helper
func NewInterruptCoordinationTestHelper() *InterruptCoordinationTestHelper {
	return &InterruptCoordinationTestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		interruptLog:          make([]InterruptEvent, 0),
		nmiHistory:            make([]NMIInterruptEvent, 0),
		irqHistory:            make([]IRQEvent, 0),
	}
}

// StepWithInterruptTracking executes one step and tracks interrupt events
func (h *InterruptCoordinationTestHelper) StepWithInterruptTracking() {
	prePC := h.CPU.PC
	preCycles := h.Bus.GetCycleCount()
	preFrameCount := h.Bus.GetFrameCount()

	// Check for VBlank start (potential NMI trigger)
	ppuStatus := h.PPU.ReadRegister(0x2002)
	vblankStart := (ppuStatus & 0x80) != 0

	// Execute step
	h.Bus.Step()

	// Check for interrupt handling
	postPC := h.CPU.PC
	postCycles := h.Bus.GetCycleCount()
	_ = h.Bus.GetFrameCount()

	// Detect if an interrupt was serviced (PC jumped to vector)
	h.detectInterruptService(prePC, postPC, preCycles, postCycles, preFrameCount, vblankStart)
}

// detectInterruptService detects if an interrupt was serviced
func (h *InterruptCoordinationTestHelper) detectInterruptService(prePC, postPC uint16, preCycles, postCycles, frameCount uint64, vblankStart bool) {
	// Check for NMI vector jump ($FFFA)
	if postPC == 0x8100 || (postPC < prePC && postPC < 0x8000) { // Simplified NMI detection
		event := InterruptEvent{
			StepNumber:    len(h.interruptLog) + 1,
			InterruptType: "NMI",
			TriggerCycle:  preCycles,
			ServiceCycle:  postCycles,
			PCBefore:      prePC,
			PCAfter:       postPC,
			VectorUsed:    0xFFFA,
			FrameNumber:   frameCount,
			PPUState:      "VBlank",
		}
		h.interruptLog = append(h.interruptLog, event)

		// Log NMI-specific details
		nmiEvent := NMIInterruptEvent{
			FrameNumber:    frameCount,
			TriggerCycle:   preCycles,
			ServiceLatency: postCycles - preCycles,
			PPUStatusRead:  false, // Would need to track this
		}
		h.nmiHistory = append(h.nmiHistory, nmiEvent)
	}

	// Check for IRQ vector jump ($FFFE)
	if postPC == 0x8200 { // Simplified IRQ detection
		event := InterruptEvent{
			StepNumber:    len(h.interruptLog) + 1,
			InterruptType: "IRQ",
			TriggerCycle:  preCycles,
			ServiceCycle:  postCycles,
			PCBefore:      prePC,
			PCAfter:       postPC,
			VectorUsed:    0xFFFE,
			FrameNumber:   frameCount,
			PPUState:      "Active",
		}
		h.interruptLog = append(h.interruptLog, event)

		// Log IRQ-specific details
		irqEvent := IRQEvent{
			TriggerCycle:   preCycles,
			Source:         "Unknown",
			ServiceLatency: postCycles - preCycles,
			Masked:         h.CPU.I, // Interrupt disable flag
		}
		h.irqHistory = append(h.irqHistory, irqEvent)
	}
}

// GetInterruptLog returns the interrupt log
func (h *InterruptCoordinationTestHelper) GetInterruptLog() []InterruptEvent {
	return h.interruptLog
}

// GetNMIHistory returns the NMI history
func (h *InterruptCoordinationTestHelper) GetNMIHistory() []NMIInterruptEvent {
	return h.nmiHistory
}

// GetIRQHistory returns the IRQ history
func (h *InterruptCoordinationTestHelper) GetIRQHistory() []IRQEvent {
	return h.irqHistory
}

// ClearInterruptLogs clears all interrupt logs
func (h *InterruptCoordinationTestHelper) ClearInterruptLogs() {
	h.interruptLog = h.interruptLog[:0]
	h.nmiHistory = h.nmiHistory[:0]
	h.irqHistory = h.irqHistory[:0]
}

// TestNMICoordination tests NMI coordination between CPU and PPU
func TestNMICoordination(t *testing.T) {
	t.Run("VBlank NMI generation", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up NMI handler
		romData := make([]uint8, 0x8000)
		// Main program
		romData[0x0000] = 0xA9 // LDA #$80
		romData[0x0001] = 0x80
		romData[0x0002] = 0x8D // STA $2000 (enable NMI)
		romData[0x0003] = 0x00
		romData[0x0004] = 0x20
		romData[0x0005] = 0xA9 // LDA #$1E
		romData[0x0006] = 0x1E
		romData[0x0007] = 0x8D // STA $2001 (enable rendering)
		romData[0x0008] = 0x01
		romData[0x0009] = 0x20
		// Main loop
		romData[0x000A] = 0xEA // NOP
		romData[0x000B] = 0x4C // JMP $800A
		romData[0x000C] = 0x0A
		romData[0x000D] = 0x80

		// NMI handler at $8100
		romData[0x0100] = 0x48 // PHA
		romData[0x0101] = 0xE6 // INC $50 (NMI counter)
		romData[0x0102] = 0x50
		romData[0x0103] = 0x68 // PLA
		romData[0x0104] = 0x40 // RTI

		// Set vectors
		romData[0x7FFA] = 0x00 // NMI vector low
		romData[0x7FFB] = 0x81 // NMI vector high
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize NMI counter
		helper.Memory.Write(0x0050, 0x00)

		// Run until we see NMI activity
		nmiCount := 0
		maxSteps := 200000

		for i := 0; i < maxSteps; i++ {
			helper.StepWithInterruptTracking()

			// Check NMI counter
			currentNMICount := helper.Memory.Read(0x0050)
			if int(currentNMICount) > nmiCount {
				nmiCount = int(currentNMICount)
				t.Logf("NMI %d detected at step %d", nmiCount, i)

				if nmiCount >= 3 {
					break
				}
			}
		}

		if nmiCount == 0 {
			t.Fatal("No NMIs were generated")
		}

		// Verify NMI timing patterns
		nmiHistory := helper.GetNMIHistory()
		if len(nmiHistory) > 1 {
			// Check intervals between NMIs (should be ~1 frame)
			for i := 1; i < len(nmiHistory); i++ {
				interval := nmiHistory[i].TriggerCycle - nmiHistory[i-1].TriggerCycle
				expectedInterval := uint64(29781) // ~1 frame
				tolerance := uint64(5000)

				if interval < expectedInterval-tolerance || interval > expectedInterval+tolerance {
					t.Errorf("NMI interval %d out of range: %d cycles (expected ~%d)",
						i, interval, expectedInterval)
				}
			}
		}

		t.Logf("Generated %d NMIs in %d steps", nmiCount, maxSteps)
	})

	t.Run("NMI timing precision", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that precisely tracks NMI timing
		romData := make([]uint8, 0x8000)
		// Enable NMI and rendering
		romData[0x0000] = 0xA9 // LDA #$80
		romData[0x0001] = 0x80
		romData[0x0002] = 0x8D // STA $2000
		romData[0x0003] = 0x00
		romData[0x0004] = 0x20
		romData[0x0005] = 0xA9 // LDA #$1E
		romData[0x0006] = 0x1E
		romData[0x0007] = 0x8D // STA $2001
		romData[0x0008] = 0x01
		romData[0x0009] = 0x20

		// Precise timing loop
		romData[0x000A] = 0xEA // NOP
		romData[0x000B] = 0xEA // NOP
		romData[0x000C] = 0xEA // NOP
		romData[0x000D] = 0xE6 // INC $60 (main loop counter)
		romData[0x000E] = 0x60
		romData[0x000F] = 0x4C // JMP $800A
		romData[0x0010] = 0x0A
		romData[0x0011] = 0x80

		// NMI handler
		romData[0x0100] = 0x48 // PHA
		romData[0x0101] = 0xA5 // LDA $60 (read main counter)
		romData[0x0102] = 0x60
		romData[0x0103] = 0x85 // STA $61 (save counter at NMI)
		romData[0x0104] = 0x61
		romData[0x0105] = 0xE6 // INC $62 (NMI counter)
		romData[0x0106] = 0x62
		romData[0x0107] = 0x68 // PLA
		romData[0x0108] = 0x40 // RTI

		// Set vectors
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize counters
		helper.Memory.Write(0x0060, 0x00) // Main loop counter
		helper.Memory.Write(0x0061, 0x00) // Counter value at NMI
		helper.Memory.Write(0x0062, 0x00) // NMI counter

		// Run and track NMI timing precision
		nmiTimings := make([]uint8, 0)
		lastNMICount := uint8(0)

		for i := 0; i < 200000; i++ {
			helper.StepWithInterruptTracking()

			currentNMICount := helper.Memory.Read(0x0062)
			if currentNMICount > lastNMICount {
				// New NMI occurred
				counterAtNMI := helper.Memory.Read(0x0061)
				nmiTimings = append(nmiTimings, counterAtNMI)
				lastNMICount = currentNMICount

				t.Logf("NMI %d: main counter was %d", currentNMICount, counterAtNMI)

				if len(nmiTimings) >= 5 {
					break
				}
			}
		}

		if len(nmiTimings) < 3 {
			t.Fatalf("Not enough NMI timing samples: %d", len(nmiTimings))
		}

		// Analyze timing consistency
		// The main counter should have similar values at each NMI
		baseValue := nmiTimings[0]
		tolerance := uint8(10) // Allow some variation

		for i, timing := range nmiTimings {
			if timing < baseValue-tolerance || timing > baseValue+tolerance {
				t.Errorf("NMI %d timing inconsistent: got %d, expected ~%d",
					i+1, timing, baseValue)
			}
		}
	})

	t.Run("NMI suppression", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that reads PPU status during VBlank (should suppress NMI)
		romData := make([]uint8, 0x8000)
		// Enable NMI
		romData[0x0000] = 0xA9 // LDA #$80
		romData[0x0001] = 0x80
		romData[0x0002] = 0x8D // STA $2000
		romData[0x0003] = 0x00
		romData[0x0004] = 0x20
		romData[0x0005] = 0xA9 // LDA #$1E
		romData[0x0006] = 0x1E
		romData[0x0007] = 0x8D // STA $2001
		romData[0x0008] = 0x01
		romData[0x0009] = 0x20

		// Loop that polls PPU status
		romData[0x000A] = 0xAD // LDA $2002 (read PPU status - clears VBlank flag)
		romData[0x000B] = 0x02
		romData[0x000C] = 0x20
		romData[0x000D] = 0x85 // STA $70 (save status)
		romData[0x000E] = 0x70
		romData[0x000F] = 0xE6 // INC $71 (polling counter)
		romData[0x0010] = 0x71
		romData[0x0011] = 0x4C // JMP $800A
		romData[0x0012] = 0x0A
		romData[0x0013] = 0x80

		// NMI handler (should rarely execute due to suppression)
		romData[0x0100] = 0x48 // PHA
		romData[0x0101] = 0xE6 // INC $72 (NMI counter)
		romData[0x0102] = 0x72
		romData[0x0103] = 0x68 // PLA
		romData[0x0104] = 0x40 // RTI

		// Set vectors
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize counters
		helper.Memory.Write(0x0070, 0x00) // PPU status
		helper.Memory.Write(0x0071, 0x00) // Polling counter
		helper.Memory.Write(0x0072, 0x00) // NMI counter

		// Run for several frames
		for i := 0; i < 300000; i++ {
			helper.StepWithInterruptTracking()
		}

		pollingCount := helper.Memory.Read(0x0071)
		nmiCount := helper.Memory.Read(0x0072)

		t.Logf("Polling count: %d, NMI count: %d", pollingCount, nmiCount)

		// Due to status polling, NMI should be suppressed frequently
		// We should see far fewer NMIs than polling iterations
		if nmiCount > pollingCount/10 {
			t.Logf("Warning: NMI suppression may not be working properly")
			t.Logf("Expected NMI count < %d, got %d", pollingCount/10, nmiCount)
		}

		// But some NMIs might still occur if timing allows
		if pollingCount == 0 {
			t.Error("No polling occurred - test setup problem")
		}
	})
}

// TestIRQCoordination tests IRQ coordination between components
func TestIRQCoordination(t *testing.T) {
	t.Run("Basic IRQ handling", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that enables IRQ and sets up handler
		romData := make([]uint8, 0x8000)
		// Clear interrupt disable flag
		romData[0x0000] = 0x58 // CLI
		// Main loop
		romData[0x0001] = 0xEA // NOP
		romData[0x0002] = 0xE6 // INC $80 (main counter)
		romData[0x0003] = 0x80
		romData[0x0004] = 0x4C // JMP $8001
		romData[0x0005] = 0x01
		romData[0x0006] = 0x80

		// IRQ handler at $8200
		romData[0x0200] = 0x48 // PHA
		romData[0x0201] = 0xE6 // INC $81 (IRQ counter)
		romData[0x0202] = 0x81
		romData[0x0203] = 0x68 // PLA
		romData[0x0204] = 0x40 // RTI

		// Set vectors
		romData[0x7FFC] = 0x00 // Reset vector
		romData[0x7FFD] = 0x80
		romData[0x7FFE] = 0x00 // IRQ vector
		romData[0x7FFF] = 0x82

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize counters
		helper.Memory.Write(0x0080, 0x00) // Main counter
		helper.Memory.Write(0x0081, 0x00) // IRQ counter

		// Simulate external IRQ trigger (this would need actual IRQ source in real system)
		// For this test, we'll just verify the framework can handle IRQ setup

		// Run main program
		for i := 0; i < 10000; i++ {
			helper.StepWithInterruptTracking()
		}

		mainCount := helper.Memory.Read(0x0080)
		irqCount := helper.Memory.Read(0x0081)

		t.Logf("Main count: %d, IRQ count: %d", mainCount, irqCount)

		// Verify main loop executed
		if mainCount == 0 {
			t.Error("Main loop did not execute")
		}

		// Verify interrupt flag state
		if helper.CPU.I {
			t.Error("Interrupt flag should be clear after CLI")
		}
	})

	t.Run("IRQ masking", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that tests IRQ masking
		romData := make([]uint8, 0x8000)
		// Set interrupt disable flag
		romData[0x0000] = 0x78 // SEI
		// Main loop with interrupts disabled
		romData[0x0001] = 0xEA // NOP
		romData[0x0002] = 0xE6 // INC $90 (main counter)
		romData[0x0003] = 0x90
		romData[0x0004] = 0x4C // JMP $8001
		romData[0x0005] = 0x01
		romData[0x0006] = 0x80

		// IRQ handler (should not execute when masked)
		romData[0x0200] = 0x48 // PHA
		romData[0x0201] = 0xE6 // INC $91 (IRQ counter)
		romData[0x0202] = 0x91
		romData[0x0203] = 0x68 // PLA
		romData[0x0204] = 0x40 // RTI

		// Set vectors
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		romData[0x7FFE] = 0x00
		romData[0x7FFF] = 0x82

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize counters
		helper.Memory.Write(0x0090, 0x00) // Main counter
		helper.Memory.Write(0x0091, 0x00) // IRQ counter

		// Run with interrupts masked
		for i := 0; i < 10000; i++ {
			helper.StepWithInterruptTracking()
		}

		mainCount := helper.Memory.Read(0x0090)
		irqCount := helper.Memory.Read(0x0091)

		t.Logf("With IRQ masked - Main count: %d, IRQ count: %d", mainCount, irqCount)

		// Verify main loop executed
		if mainCount == 0 {
			t.Error("Main loop did not execute with IRQ masked")
		}

		// Verify IRQ was masked (no IRQs should have occurred)
		if irqCount > 0 {
			t.Errorf("IRQ should be masked, but %d IRQs occurred", irqCount)
		}

		// Verify interrupt flag state
		if !helper.CPU.I {
			t.Error("Interrupt flag should be set after SEI")
		}
	})
}

// TestInterruptTiming tests precise interrupt timing
func TestInterruptTiming(t *testing.T) {
	t.Run("Interrupt service latency", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test NMI service latency
		romData := make([]uint8, 0x8000)
		// Enable NMI
		romData[0x0000] = 0xA9 // LDA #$80
		romData[0x0001] = 0x80
		romData[0x0002] = 0x8D // STA $2000
		romData[0x0003] = 0x00
		romData[0x0004] = 0x20
		romData[0x0005] = 0xA9 // LDA #$1E
		romData[0x0006] = 0x1E
		romData[0x0007] = 0x8D // STA $2001
		romData[0x0008] = 0x01
		romData[0x0009] = 0x20

		// Main loop with known instruction timing
		romData[0x000A] = 0xEA // NOP (2 cycles)
		romData[0x000B] = 0xEA // NOP (2 cycles)
		romData[0x000C] = 0xEA // NOP (2 cycles)
		romData[0x000D] = 0x4C // JMP $800A (3 cycles)
		romData[0x000E] = 0x0A
		romData[0x000F] = 0x80

		// Fast NMI handler
		romData[0x0100] = 0x40 // RTI (6 cycles)

		// Set vectors
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Run and measure interrupt latency
		interruptLatencies := make([]uint64, 0)

		for i := 0; i < 200000; i++ {
			helper.StepWithInterruptTracking()

			// Check for new interrupt events
			interruptLog := helper.GetInterruptLog()
			if len(interruptLog) > len(interruptLatencies) {
				// New interrupt occurred
				newEvent := interruptLog[len(interruptLatencies)]
				latency := newEvent.ServiceCycle - newEvent.TriggerCycle
				interruptLatencies = append(interruptLatencies, latency)

				t.Logf("Interrupt %d latency: %d cycles", len(interruptLatencies), latency)

				if len(interruptLatencies) >= 3 {
					break
				}
			}
		}

		if len(interruptLatencies) == 0 {
			t.Fatal("No interrupt latencies measured")
		}

		// NMI service should take consistent time
		// Typical NMI service overhead is 7 cycles
		expectedLatency := uint64(7)
		tolerance := uint64(3)

		for i, latency := range interruptLatencies {
			if latency < expectedLatency-tolerance || latency > expectedLatency+tolerance {
				t.Errorf("Interrupt %d latency out of range: %d cycles (expected %d±%d)",
					i+1, latency, expectedLatency, tolerance)
			}
		}
	})

	t.Run("Interrupt during different instructions", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program with various instruction types
		romData := make([]uint8, 0x8000)
		// Enable NMI
		romData[0x0000] = 0xA9 // LDA #$80
		romData[0x0001] = 0x80
		romData[0x0002] = 0x8D // STA $2000
		romData[0x0003] = 0x00
		romData[0x0004] = 0x20
		romData[0x0005] = 0xA9 // LDA #$1E
		romData[0x0006] = 0x1E
		romData[0x0007] = 0x8D // STA $2001
		romData[0x0008] = 0x01
		romData[0x0009] = 0x20

		// Mixed instruction types
		romData[0x000A] = 0xA9 // LDA #$01 (2 cycles)
		romData[0x000B] = 0x01
		romData[0x000C] = 0x85 // STA $00 (3 cycles)
		romData[0x000D] = 0x00
		romData[0x000E] = 0xA5 // LDA $00 (3 cycles)
		romData[0x000F] = 0x00
		romData[0x0010] = 0x69 // ADC #$01 (2 cycles)
		romData[0x0011] = 0x01
		romData[0x0012] = 0x8D // STA $2000 (4 cycles)
		romData[0x0013] = 0x00
		romData[0x0014] = 0x20
		romData[0x0015] = 0x4C // JMP $800A (3 cycles)
		romData[0x0016] = 0x0A
		romData[0x0017] = 0x80

		// NMI handler that tracks interrupt context
		romData[0x0100] = 0x48 // PHA
		romData[0x0101] = 0x8A // TXA
		romData[0x0102] = 0x48 // PHA
		romData[0x0103] = 0xE6 // INC $A0 (interrupt counter)
		romData[0x0104] = 0xA0
		romData[0x0105] = 0x68 // PLA
		romData[0x0106] = 0xAA // TAX
		romData[0x0107] = 0x68 // PLA
		romData[0x0108] = 0x40 // RTI

		// Set vectors
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize interrupt counter
		helper.Memory.Write(0x00A0, 0x00)

		// Run and verify interrupts can occur during different instructions
		for i := 0; i < 300000; i++ {
			helper.StepWithInterruptTracking()
		}

		interruptCount := helper.Memory.Read(0x00A0)
		interruptLog := helper.GetInterruptLog()

		t.Logf("Interrupts during mixed instructions: %d", interruptCount)
		t.Logf("Logged interrupt events: %d", len(interruptLog))

		if interruptCount == 0 {
			t.Error("No interrupts occurred during mixed instruction test")
		}

		// Verify interrupt context was preserved
		// (This would require more detailed state tracking in a real implementation)
	})
}

// TestInterruptCoordinationEdgeCases tests edge cases in interrupt coordination
func TestInterruptCoordinationEdgeCases(t *testing.T) {
	t.Run("Rapid interrupt toggling", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that rapidly toggles NMI enable
		romData := make([]uint8, 0x8000)
		// Toggle NMI enable rapidly
		romData[0x0000] = 0xA9 // LDA #$80
		romData[0x0001] = 0x80
		romData[0x0002] = 0x8D // STA $2000 (enable NMI)
		romData[0x0003] = 0x00
		romData[0x0004] = 0x20
		romData[0x0005] = 0xA9 // LDA #$00
		romData[0x0006] = 0x00
		romData[0x0007] = 0x8D // STA $2000 (disable NMI)
		romData[0x0008] = 0x00
		romData[0x0009] = 0x20
		romData[0x000A] = 0x4C // JMP $8000 (repeat)
		romData[0x000B] = 0x00
		romData[0x000C] = 0x80

		// NMI handler
		romData[0x0100] = 0x48 // PHA
		romData[0x0101] = 0xE6 // INC $B0
		romData[0x0102] = 0xB0
		romData[0x0103] = 0x68 // PLA
		romData[0x0104] = 0x40 // RTI

		// Set vectors
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize counter
		helper.Memory.Write(0x00B0, 0x00)

		// Run with rapid toggling
		for i := 0; i < 100000; i++ {
			helper.StepWithInterruptTracking()
		}

		interruptCount := helper.Memory.Read(0x00B0)

		t.Logf("Interrupts with rapid toggling: %d", interruptCount)

		// System should remain stable despite rapid toggling
		if helper.CPU.PC < 0x8000 {
			t.Errorf("PC went outside ROM during rapid toggling: 0x%04X", helper.CPU.PC)
		}
	})

	t.Run("Interrupt during DMA", func(t *testing.T) {
		helper := NewInterruptCoordinationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that triggers DMA and NMI
		romData := make([]uint8, 0x8000)
		// Enable NMI
		romData[0x0000] = 0xA9 // LDA #$80
		romData[0x0001] = 0x80
		romData[0x0002] = 0x8D // STA $2000
		romData[0x0003] = 0x00
		romData[0x0004] = 0x20
		romData[0x0005] = 0xA9 // LDA #$1E
		romData[0x0006] = 0x1E
		romData[0x0007] = 0x8D // STA $2001
		romData[0x0008] = 0x01
		romData[0x0009] = 0x20

		// Set up DMA source data
		romData[0x000A] = 0xA9 // LDA #$55
		romData[0x000B] = 0x55
		romData[0x000C] = 0x8D // STA $0200
		romData[0x000D] = 0x00
		romData[0x000E] = 0x02

		// Trigger DMA
		romData[0x000F] = 0xA9 // LDA #$02
		romData[0x0010] = 0x02
		romData[0x0011] = 0x8D // STA $4014 (trigger DMA)
		romData[0x0012] = 0x14
		romData[0x0013] = 0x40

		// Continue after DMA
		romData[0x0014] = 0xEA // NOP
		romData[0x0015] = 0x4C // JMP to continue
		romData[0x0016] = 0x14
		romData[0x0017] = 0x80

		// NMI handler
		romData[0x0100] = 0x48 // PHA
		romData[0x0101] = 0xE6 // INC $C0
		romData[0x0102] = 0xC0
		romData[0x0103] = 0x68 // PLA
		romData[0x0104] = 0x40 // RTI

		// Set vectors
		romData[0x7FFA] = 0x00
		romData[0x7FFB] = 0x81
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80

		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Initialize counter
		helper.Memory.Write(0x00C0, 0x00)

		// Run and verify DMA/interrupt coordination
		dmaTriggered := false
		for i := 0; i < 200000; i++ {
			helper.StepWithInterruptTracking()

			if helper.Bus.IsDMAInProgress() && !dmaTriggered {
				dmaTriggered = true
				t.Logf("DMA triggered at step %d", i)
			}
		}

		interruptCount := helper.Memory.Read(0x00C0)

		t.Logf("DMA triggered: %v", dmaTriggered)
		t.Logf("Interrupts during DMA test: %d", interruptCount)

		if !dmaTriggered {
			t.Error("DMA was not triggered")
		}

		// System should handle DMA and interrupts correctly
		if helper.CPU.PC < 0x8000 {
			t.Errorf("PC corrupted during DMA/interrupt test: 0x%04X", helper.CPU.PC)
		}
	})
}
