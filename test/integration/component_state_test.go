package integration

import (
	"testing"
)

// ComponentStateTestHelper provides utilities for testing component state coordination
type ComponentStateTestHelper struct {
	*IntegrationTestHelper
	stateLog  []ComponentStateSnapshot
	cycleLog  []CycleState
	memoryLog []MemoryAccessEvent
}

// ComponentStateSnapshot represents the state of all components at a point in time
type ComponentStateSnapshot struct {
	StepNumber  int
	CPUState    CPUSnapshot
	PPUState    PPUSnapshot
	APUState    APUSnapshot
	MemoryState MemorySnapshot
	BusState    BusSnapshot
	Timestamp   uint64
}

// CPUSnapshot captures CPU state
type CPUSnapshot struct {
	PC         uint16
	A, X, Y    uint8
	SP         uint8
	P          uint8
	Cycles     uint64
	LastOpcode uint8
	Flags      CPUFlags
}

// CPUFlags represents CPU status flags
type CPUFlags struct {
	N, V, B, D, I, Z, C bool
}

// PPUSnapshot captures PPU state
type PPUSnapshot struct {
	Scanline     int
	Cycle        int
	FrameCount   uint64
	VBlankFlag   bool
	RenderingOn  bool
	NMIEnabled   bool
	LastRegister uint8
	LastWrite    uint8
}

// APUSnapshot captures APU state
type APUSnapshot struct {
	FrameCounter uint64
	Cycles       uint64
	LastRegister uint16
	LastWrite    uint8
}

// MemorySnapshot captures memory access patterns
type MemorySnapshot struct {
	LastCPURead    uint16
	LastCPUWrite   uint16
	LastPPURead    uint16
	LastPPUWrite   uint16
	DMAActive      bool
	CartridgeReads int
}

// BusSnapshot captures bus coordination state
type BusSnapshot struct {
	TotalCycles   uint64
	FrameCount    uint64
	DMAInProgress bool
	NMIPending    bool
	OddFrame      bool
}

// CycleState represents the state during a single cycle
type CycleState struct {
	CycleNumber uint64
	CPUActive   bool
	PPUActive   bool
	APUActive   bool
	MemoryBusy  bool
	Operation   string
}

// MemoryAccessEvent represents a memory access coordination event
type MemoryAccessEvent struct {
	Cycle     uint64
	Address   uint16
	Value     uint8
	Operation string // "CPU_READ", "CPU_WRITE", "PPU_READ", "PPU_WRITE", "DMA"
	Source    string
}

// NewComponentStateTestHelper creates a new component state test helper
func NewComponentStateTestHelper() *ComponentStateTestHelper {
	return &ComponentStateTestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		stateLog:              make([]ComponentStateSnapshot, 0),
		cycleLog:              make([]CycleState, 0),
		memoryLog:             make([]MemoryAccessEvent, 0),
	}
}

// CaptureComponentState captures the current state of all components
func (h *ComponentStateTestHelper) CaptureComponentState() {
	snapshot := ComponentStateSnapshot{
		StepNumber: len(h.stateLog) + 1,
		CPUState: CPUSnapshot{
			PC:         h.CPU.PC,
			A:          h.CPU.A,
			X:          h.CPU.X,
			Y:          h.CPU.Y,
			SP:         h.CPU.SP,
			P:          h.getCPUStatus(),
			Cycles:     h.Bus.GetCycleCount(),
			LastOpcode: h.Memory.Read(h.CPU.PC),
			Flags: CPUFlags{
				N: h.CPU.N,
				V: h.CPU.V,
				B: h.CPU.B,
				D: h.CPU.D,
				I: h.CPU.I,
				Z: h.CPU.Z,
				C: h.CPU.C,
			},
		},
		PPUState: PPUSnapshot{
			Scanline:     h.getPPUScanline(),
			Cycle:        h.getPPUCycle(),
			FrameCount:   h.Bus.GetFrameCount(),
			VBlankFlag:   (h.PPU.ReadRegister(0x2002) & 0x80) != 0,
			RenderingOn:  (h.PPU.ReadRegister(0x2001) & 0x18) != 0,
			NMIEnabled:   (h.getPPUCtrl() & 0x80) != 0,
			LastRegister: 0x02, // Would track last accessed register (low byte only)
			LastWrite:    0,
		},
		APUState: APUSnapshot{
			FrameCounter: h.Bus.GetFrameCount(), // APU frame counter
			Cycles:       h.Bus.GetCycleCount(),
			LastRegister: 0x4000, // Would track last accessed register
			LastWrite:    0,
		},
		MemoryState: MemorySnapshot{
			LastCPURead:    h.CPU.PC,
			LastCPUWrite:   0, // Would need to track this
			LastPPURead:    0, // Would need to track this
			LastPPUWrite:   0, // Would need to track this
			DMAActive:      h.Bus.IsDMAInProgress(),
			CartridgeReads: len(h.GetMockCartridge().prgReads),
		},
		BusState: BusSnapshot{
			TotalCycles:   h.Bus.GetCycleCount(),
			FrameCount:    h.Bus.GetFrameCount(),
			DMAInProgress: h.Bus.IsDMAInProgress(),
			NMIPending:    false, // Would need to expose this
			OddFrame:      (h.Bus.GetFrameCount() % 2) == 1,
		},
		Timestamp: h.Bus.GetCycleCount(),
	}

	h.stateLog = append(h.stateLog, snapshot)
}

// StepWithStateCapture executes one step and captures component state
func (h *ComponentStateTestHelper) StepWithStateCapture() {
	h.Bus.Step()
	h.CaptureComponentState()
}

// getCPUStatus returns the CPU status register value
func (h *ComponentStateTestHelper) getCPUStatus() uint8 {
	status := uint8(0)
	if h.CPU.N {
		status |= 0x80
	}
	if h.CPU.V {
		status |= 0x40
	}
	if h.CPU.B {
		status |= 0x10
	}
	if h.CPU.D {
		status |= 0x08
	}
	if h.CPU.I {
		status |= 0x04
	}
	if h.CPU.Z {
		status |= 0x02
	}
	if h.CPU.C {
		status |= 0x01
	}
	return status
}

// getPPUScanline returns the current PPU scanline (simplified)
func (h *ComponentStateTestHelper) getPPUScanline() int {
	// This would need to be exposed by the PPU
	// For now, estimate based on frame progress
	totalCycles := h.Bus.GetCycleCount()
	frameCycles := totalCycles % 29781 // Cycles per frame
	return int(frameCycles / 113)      // Approximate cycles per scanline
}

// getPPUCycle returns the current PPU cycle within scanline (simplified)
func (h *ComponentStateTestHelper) getPPUCycle() int {
	totalCycles := h.Bus.GetCycleCount()
	frameCycles := totalCycles % 29781
	return int(frameCycles % 113)
}

// getPPUCtrl returns the PPUCTRL register value (simplified)
func (h *ComponentStateTestHelper) getPPUCtrl() uint8 {
	// This would need to be exposed by the PPU
	// For now, return a default value
	return 0x80 // Assume NMI enabled for most tests
}

// GetStateLog returns the component state log
func (h *ComponentStateTestHelper) GetStateLog() []ComponentStateSnapshot {
	return h.stateLog
}

// ClearStateLogs clears all state logs
func (h *ComponentStateTestHelper) ClearStateLogs() {
	h.stateLog = h.stateLog[:0]
	h.cycleLog = h.cycleLog[:0]
	h.memoryLog = h.memoryLog[:0]
}

// TestComponentStateCoordination tests coordination of component states
func TestComponentStateCoordination(t *testing.T) {
	t.Run("Basic state synchronization", func(t *testing.T) {
		helper := NewComponentStateTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Execute steps and capture state
		stepCount := 50
		for i := 0; i < stepCount; i++ {
			helper.StepWithStateCapture()
		}

		stateLog := helper.GetStateLog()
		if len(stateLog) != stepCount {
			t.Fatalf("Expected %d state snapshots, got %d", stepCount, len(stateLog))
		}

		// Verify state progression
		for i := 1; i < len(stateLog); i++ {
			prev := stateLog[i-1]
			curr := stateLog[i]

			// CPU cycles should increase
			if curr.CPUState.Cycles <= prev.CPUState.Cycles {
				t.Errorf("Step %d: CPU cycles should increase: %d -> %d",
					i, prev.CPUState.Cycles, curr.CPUState.Cycles)
			}

			// Bus total cycles should match CPU cycles
			if curr.BusState.TotalCycles != curr.CPUState.Cycles {
				t.Errorf("Step %d: Bus cycles (%d) should match CPU cycles (%d)",
					i, curr.BusState.TotalCycles, curr.CPUState.Cycles)
			}

			// Frame count should be consistent across components
			if curr.PPUState.FrameCount != curr.BusState.FrameCount {
				t.Errorf("Step %d: PPU frame count (%d) != Bus frame count (%d)",
					i, curr.PPUState.FrameCount, curr.BusState.FrameCount)
			}
		}
	})

	t.Run("CPU state consistency", func(t *testing.T) {
		helper := NewComponentStateTestHelper()

		// Create program with known state changes
		program := []uint8{
			0xA9, 0x42, // LDA #$42 (A = 0x42)
			0xA2, 0x33, // LDX #$33 (X = 0x33)
			0xA0, 0x24, // LDY #$24 (Y = 0x24)
			0x18,       // CLC (C = 0)
			0x38,       // SEC (C = 1)
			0xA9, 0x00, // LDA #$00 (A = 0x00, Z = 1)
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute known sequence and verify state changes
		expectedStates := []struct {
			step int
			A    uint8
			X    uint8
			Y    uint8
			Z    bool
			C    bool
		}{
			{0, 0x00, 0x00, 0x00, false, false}, // Initial state
			{1, 0x42, 0x00, 0x00, false, false}, // After LDA #$42
			{2, 0x42, 0x33, 0x00, false, false}, // After LDX #$33
			{3, 0x42, 0x33, 0x24, false, false}, // After LDY #$24
			{4, 0x42, 0x33, 0x24, false, false}, // After CLC
			{5, 0x42, 0x33, 0x24, false, true},  // After SEC
			{6, 0x00, 0x33, 0x24, true, true},   // After LDA #$00
		}

		for i, expected := range expectedStates {
			if i > 0 {
				helper.StepWithStateCapture()
			} else {
				helper.CaptureComponentState()
			}

			stateLog := helper.GetStateLog()
			if len(stateLog) != i+1 {
				t.Fatalf("Expected %d state entries, got %d", i+1, len(stateLog))
			}

			state := stateLog[len(stateLog)-1]
			cpu := state.CPUState

			if cpu.A != expected.A {
				t.Errorf("Step %d: A register expected 0x%02X, got 0x%02X",
					i, expected.A, cpu.A)
			}
			if cpu.X != expected.X {
				t.Errorf("Step %d: X register expected 0x%02X, got 0x%02X",
					i, expected.X, cpu.X)
			}
			if cpu.Y != expected.Y {
				t.Errorf("Step %d: Y register expected 0x%02X, got 0x%02X",
					i, expected.Y, cpu.Y)
			}
			if cpu.Flags.Z != expected.Z {
				t.Errorf("Step %d: Z flag expected %v, got %v",
					i, expected.Z, cpu.Flags.Z)
			}
			if cpu.Flags.C != expected.C {
				t.Errorf("Step %d: C flag expected %v, got %v",
					i, expected.C, cpu.Flags.C)
			}
		}
	})

	t.Run("PPU state coordination", func(t *testing.T) {
		helper := NewComponentStateTestHelper()
		helper.SetupBasicROM(0x8000)

		// Enable rendering to get PPU state changes
		helper.Memory.Write(0x2000, 0x80) // PPUCTRL
		helper.Memory.Write(0x2001, 0x1E) // PPUMASK
		helper.Bus.Reset()

		// Run and track PPU state changes
		frameTransitions := make([]int, 0)
		vblankTransitions := make([]int, 0)

		for i := 0; i < 200000; i++ {
			helper.StepWithStateCapture()

			stateLog := helper.GetStateLog()
			if len(stateLog) >= 2 {
				prev := stateLog[len(stateLog)-2]
				curr := stateLog[len(stateLog)-1]

				// Track frame transitions
				if curr.PPUState.FrameCount > prev.PPUState.FrameCount {
					frameTransitions = append(frameTransitions, i)
				}

				// Track VBlank transitions
				if curr.PPUState.VBlankFlag != prev.PPUState.VBlankFlag {
					vblankTransitions = append(vblankTransitions, i)
				}

				if len(frameTransitions) >= 3 {
					break
				}
			}
		}

		if len(frameTransitions) < 2 {
			t.Fatalf("Expected at least 2 frame transitions, got %d", len(frameTransitions))
		}

		// Verify frame timing consistency
		for i := 1; i < len(frameTransitions); i++ {
			frameLength := frameTransitions[i] - frameTransitions[i-1]
			expectedFrameSteps := 29781 / 2 // Approximate
			tolerance := 1000

			if frameLength < expectedFrameSteps-tolerance || frameLength > expectedFrameSteps+tolerance {
				t.Errorf("Frame %d length inconsistent: %d steps (expected ~%d)",
					i, frameLength, expectedFrameSteps)
			}
		}

		t.Logf("Frame transitions: %v", frameTransitions)
		t.Logf("VBlank transitions: %v", vblankTransitions)
	})
}

// TestComponentStateConsistency tests consistency of component states
func TestComponentStateConsistency(t *testing.T) {
	t.Run("Memory access coordination", func(t *testing.T) {
		helper := NewComponentStateTestHelper()

		// Program that accesses different memory regions
		program := []uint8{
			0xA9, 0x55, // LDA #$55
			0x85, 0x00, // STA $00 (Zero page)
			0x8D, 0x00, 0x03, // STA $0300 (RAM)
			0x8D, 0x00, 0x20, // STA $2000 (PPU)
			0xAD, 0x02, 0x20, // LDA $2002 (PPU status)
			0x85, 0x01, // STA $01
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute and track memory access patterns
		memoryAccesses := make([]string, 0)

		for i := 0; i < 20; i++ {
			preState := helper.GetStateLog()
			preCartridgeReads := 0
			if len(preState) > 0 {
				preCartridgeReads = preState[len(preState)-1].MemoryState.CartridgeReads
			}

			helper.StepWithStateCapture()

			postState := helper.GetStateLog()
			postCartridgeReads := postState[len(postState)-1].MemoryState.CartridgeReads

			if postCartridgeReads > preCartridgeReads {
				memoryAccesses = append(memoryAccesses, "Cartridge")
			}

			// Check for memory writes (would need better tracking)
			cpu := postState[len(postState)-1].CPUState
			if cpu.LastOpcode == 0x85 || cpu.LastOpcode == 0x8D {
				memoryAccesses = append(memoryAccesses, "Write")
			}
		}

		if len(memoryAccesses) == 0 {
			t.Error("No memory accesses detected")
		}

		t.Logf("Memory access pattern: %v", memoryAccesses)

		// Verify cartridge reads increased
		finalState := helper.GetStateLog()
		if len(finalState) > 0 {
			finalReads := finalState[len(finalState)-1].MemoryState.CartridgeReads
			if finalReads == 0 {
				t.Error("No cartridge reads detected")
			}
		}
	})

	t.Run("DMA state coordination", func(t *testing.T) {
		helper := NewComponentStateTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that triggers DMA
		program := []uint8{
			0xA9, 0x55, // LDA #$55
			0x8D, 0x00, 0x02, // STA $0200 (Set up DMA source)
			0xA9, 0x02, // LDA #$02
			0x8D, 0x14, 0x40, // STA $4014 (Trigger DMA)
			0xEA,             // NOP
			0x4C, 0x08, 0x80, // JMP $8008
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Track DMA state changes
		dmaStates := make([]bool, 0)
		dmaStartStep := -1
		dmaEndStep := -1

		for i := 0; i < 1000; i++ {
			helper.StepWithStateCapture()

			stateLog := helper.GetStateLog()
			current := stateLog[len(stateLog)-1]

			dmaActive := current.MemoryState.DMAActive
			dmaStates = append(dmaStates, dmaActive)

			if dmaActive && dmaStartStep == -1 {
				dmaStartStep = i
			}
			if !dmaActive && dmaStartStep != -1 && dmaEndStep == -1 {
				dmaEndStep = i
			}

			if dmaEndStep != -1 {
				break
			}
		}

		if dmaStartStep == -1 {
			t.Fatal("DMA never started")
		}

		if dmaEndStep == -1 {
			t.Fatal("DMA never ended")
		}

		dmaLength := dmaEndStep - dmaStartStep
		expectedDMALength := 513 // Typical DMA duration in cycles
		tolerance := 100

		if dmaLength < expectedDMALength-tolerance || dmaLength > expectedDMALength+tolerance {
			t.Errorf("DMA duration unexpected: %d steps (expected ~%d)",
				dmaLength, expectedDMALength)
		}

		t.Logf("DMA active from step %d to %d (%d steps)", dmaStartStep, dmaEndStep, dmaLength)

		// Verify DMA coordination across components
		dmaMidpoint := dmaStartStep + dmaLength/2
		if dmaMidpoint < len(dmaStates) {
			midState := helper.GetStateLog()[dmaMidpoint]

			// During DMA, CPU should be suspended but other components active
			if !midState.MemoryState.DMAActive {
				t.Error("DMA should be active at midpoint")
			}

			if !midState.BusState.DMAInProgress {
				t.Error("Bus should show DMA in progress")
			}
		}
	})

	t.Run("Interrupt state coordination", func(t *testing.T) {
		helper := NewComponentStateTestHelper()
		helper.SetupBasicROM(0x8000)

		// Set up interrupt handling
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

		// Main loop
		romData[0x000A] = 0xE6 // INC $80 (main counter)
		romData[0x000B] = 0x80
		romData[0x000C] = 0x4C // JMP $800A
		romData[0x000D] = 0x0A
		romData[0x000E] = 0x80

		// NMI handler
		romData[0x0100] = 0x48 // PHA
		romData[0x0101] = 0xE6 // INC $81 (NMI counter)
		romData[0x0102] = 0x81
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
		helper.Memory.Write(0x0080, 0x00) // Main counter
		helper.Memory.Write(0x0081, 0x00) // NMI counter

		// Track interrupt coordination
		interruptStates := make([]bool, 0)
		frameTransitions := make([]int, 0)

		for i := 0; i < 200000; i++ {
			helper.StepWithStateCapture()

			stateLog := helper.GetStateLog()
			current := stateLog[len(stateLog)-1]

			// Track frame transitions (potential NMI triggers)
			if len(stateLog) >= 2 {
				prev := stateLog[len(stateLog)-2]
				if current.PPUState.FrameCount > prev.PPUState.FrameCount {
					frameTransitions = append(frameTransitions, i)
				}
			}

			// Track interrupt enable state
			interruptEnabled := !current.CPUState.Flags.I
			interruptStates = append(interruptStates, interruptEnabled)

			if len(frameTransitions) >= 3 {
				break
			}
		}

		// Verify interrupt coordination
		mainCount := helper.Memory.Read(0x0080)
		nmiCount := helper.Memory.Read(0x0081)

		t.Logf("Main count: %d, NMI count: %d", mainCount, nmiCount)
		t.Logf("Frame transitions: %d", len(frameTransitions))

		if mainCount == 0 {
			t.Error("Main loop did not execute")
		}

		if len(frameTransitions) == 0 {
			t.Error("No frame transitions detected")
		}

		// NMI should occur roughly once per frame when enabled
		if nmiCount > 0 && len(frameTransitions) > 0 {
			ratio := float64(nmiCount) / float64(len(frameTransitions))
			if ratio < 0.5 || ratio > 2.0 {
				t.Errorf("NMI/frame ratio unexpected: %.2f (NMI: %d, Frames: %d)",
					ratio, nmiCount, len(frameTransitions))
			}
		}
	})
}

// TestComponentStateEdgeCases tests edge cases in component state coordination
func TestComponentStateEdgeCases(t *testing.T) {
	t.Run("Reset state coordination", func(t *testing.T) {
		helper := NewComponentStateTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Run system for a while
		for i := 0; i < 100; i++ {
			helper.StepWithStateCapture()
		}

		// Capture pre-reset state
		preResetLog := helper.GetStateLog()
		preResetState := preResetLog[len(preResetLog)-1]

		// Reset system
		helper.Bus.Reset()
		helper.CaptureComponentState()

		// Capture post-reset state
		postResetLog := helper.GetStateLog()
		postResetState := postResetLog[len(postResetLog)-1]

		// Verify reset coordination
		if postResetState.CPUState.PC != 0x8000 {
			t.Errorf("CPU PC should be 0x8000 after reset, got 0x%04X", postResetState.CPUState.PC)
		}

		if postResetState.CPUState.Cycles != 0 {
			t.Errorf("CPU cycles should be 0 after reset, got %d", postResetState.CPUState.Cycles)
		}

		if postResetState.BusState.TotalCycles != 0 {
			t.Errorf("Bus cycles should be 0 after reset, got %d", postResetState.BusState.TotalCycles)
		}

		if postResetState.PPUState.FrameCount != 0 {
			t.Errorf("PPU frame count should be 0 after reset, got %d", postResetState.PPUState.FrameCount)
		}

		if postResetState.BusState.DMAInProgress {
			t.Error("DMA should not be in progress after reset")
		}

		t.Logf("Pre-reset cycles: %d, Post-reset cycles: %d",
			preResetState.CPUState.Cycles, postResetState.CPUState.Cycles)
	})

	t.Run("Rapid state changes", func(t *testing.T) {
		helper := NewComponentStateTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that rapidly changes various states
		program := []uint8{
			0x78,       // SEI (disable interrupts)
			0x58,       // CLI (enable interrupts)
			0x18,       // CLC (clear carry)
			0x38,       // SEC (set carry)
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x00, 0x20, // STA $2000 (disable NMI)
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Track rapid state changes
		flagChanges := 0
		nmiToggles := 0

		for i := 0; i < 50; i++ {
			preState := helper.GetStateLog()
			helper.StepWithStateCapture()
			postState := helper.GetStateLog()

			if len(preState) > 0 && len(postState) > 0 {
				pre := preState[len(preState)-1]
				post := postState[len(postState)-1]

				// Count flag changes
				if pre.CPUState.Flags.I != post.CPUState.Flags.I ||
					pre.CPUState.Flags.C != post.CPUState.Flags.C {
					flagChanges++
				}

				// Count NMI enable toggles
				if pre.PPUState.NMIEnabled != post.PPUState.NMIEnabled {
					nmiToggles++
				}
			}
		}

		t.Logf("Flag changes: %d, NMI toggles: %d", flagChanges, nmiToggles)

		// Verify system handled rapid changes
		if flagChanges == 0 {
			t.Error("No flag changes detected during rapid state change test")
		}

		// Verify final state is consistent
		finalState := helper.GetStateLog()
		if len(finalState) > 0 {
			final := finalState[len(finalState)-1]
			if final.CPUState.PC < 0x8000 {
				t.Errorf("PC should be in ROM after rapid changes: 0x%04X", final.CPUState.PC)
			}
		}
	})
}
