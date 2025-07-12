// Package bus implements the system bus for communication between NES components.
package bus

import (
	"fmt"
	
	"gones/internal/apu"
	"gones/internal/cartridge"
	"gones/internal/cpu"
	"gones/internal/input"
	"gones/internal/memory"
	"gones/internal/ppu"
)

// Bus connects all NES components together
type Bus struct {
	// Core components
	CPU    *cpu.CPU
	PPU    *ppu.PPU
	APU    *apu.APU
	Memory *memory.Memory
	Input  *input.InputState

	// System state
	totalCycles uint64
	cpuCycles   uint64
	ppuCycles   uint64
	frameCount  uint64

	// Timing coordination
	dmaSuspendCycles uint64
	dmaInProgress    bool
	nmiPending       bool

	// Frame timing (NTSC: 262 scanlines, 341 PPU cycles/scanline)
	cyclesPerFrame uint64 // 89342 PPU cycles = 29780.67 CPU cycles
	oddFrame       bool

	// Execution logging for testing
	executionLog   []BusExecutionEvent
	loggingEnabled bool

	// Memory monitoring for debugging
	memoryWatchpoints map[uint16]uint8 // Address -> previous value
	watchpointLogging bool
}

// New creates a new system bus with all components
func New() *Bus {
	bus := &Bus{
		PPU:   ppu.New(),
		APU:   apu.New(),
		Input: input.NewInputState(),

		// NTSC timing: 89342 PPU cycles per frame
		cyclesPerFrame: 89342,

		// Initialize memory monitoring
		memoryWatchpoints: make(map[uint16]uint8),
		watchpointLogging: false,
	}

	// Memory needs references to PPU and APU
	bus.Memory = memory.New(bus.PPU, bus.APU, nil) // Cartridge will be set later

	// Set up input system in memory
	bus.Memory.SetInputSystem(bus.Input)

	// CPU needs memory interface
	bus.CPU = cpu.New(bus.Memory)

	// Set up callbacks
	bus.PPU.SetNMICallback(bus.triggerNMI)
	bus.PPU.SetFrameCompleteCallback(bus.handleFrameComplete)
	bus.Memory.SetDMACallback(bus.TriggerOAMDMA)

	// Reset all components to proper initial state
	bus.Reset()

	return bus
}

// Reset resets all components to their initial state
func (b *Bus) Reset() {
	b.CPU.Reset()
	b.PPU.Reset()
	b.APU.Reset()
	b.Input.Reset()

	// Reset timing state
	b.totalCycles = 0
	b.cpuCycles = 0
	b.ppuCycles = 0
	b.frameCount = 0
	b.dmaSuspendCycles = 0
	b.dmaInProgress = false
	b.nmiPending = false
	b.oddFrame = false

	// Synchronize PPU frame count with bus
	b.PPU.SetFrameCount(0)

	// Clear execution log
	b.executionLog = make([]BusExecutionEvent, 0)
	b.loggingEnabled = false

	// Initialize memory monitoring
	b.memoryWatchpoints = make(map[uint16]uint8)
	b.watchpointLogging = false
}

// triggerNMI is called by the PPU when an NMI should be triggered
func (b *Bus) triggerNMI() {
	b.nmiPending = true
}

// handleFrameComplete is called by the PPU when a frame is naturally completed
func (b *Bus) handleFrameComplete() {
	// Synchronize bus frame counter with PPU's frame counter
	b.frameCount = b.PPU.GetFrameCount()
	
	// Frame-synchronized input update (like ChibiNES/Fogleman NES)
	// This ensures input states are refreshed every frame for proper game sync
	if b.Input != nil {
		// The input states are maintained but this gives games a consistent
		// point to poll controller states, similar to real NES VBlank timing
		b.synchronizeInputStates()
	}
	
	// The PPU manages its own timing internally, we just track frame completion
	// Do NOT reset any cycle counters - they should be cumulative for timing accuracy
	// The PPU handles odd/even frame timing internally with proper cycle skipping
}

// synchronizeInputStates provides frame-synchronized input refreshing
func (b *Bus) synchronizeInputStates() {
	// This method can be used for frame-based input synchronization
	// Currently, our simplified approach doesn't require frame buffering,
	// but this provides a hook for future enhancements if needed
	
	// For debugging: log frame sync events occasionally
	if b.frameCount%60 == 0 { // Once per second at 60fps
		fmt.Printf("[FRAME_SYNC] Frame %d: Input synchronized\n", b.frameCount)
	}
}

// Step executes one CPU instruction and advances other components accordingly
func (b *Bus) Step() {
	var cpuCycles uint64

	// Capture pre-step state for logging
	preFrameCount := b.frameCount
	prePC := b.CPU.PC
	var preOpcode uint8
	if b.Memory != nil {
		preOpcode = b.Memory.Read(prePC)
	}

	// Check if CPU is suspended for DMA
	if b.dmaSuspendCycles > 0 {
		// CPU is suspended, consume DMA cycles
		cpuCycles = 1
		b.dmaSuspendCycles--
		if b.dmaSuspendCycles == 0 {
			b.dmaInProgress = false
		}
	} else {
		// Handle pending NMI before executing instruction
		if b.nmiPending {
			b.CPU.TriggerNMI()
			b.nmiPending = false
		}

		// Execute one CPU instruction
		cpuCycles = b.CPU.Step()
	}

	// PPU runs at exactly 3x CPU speed (cycle-accurate)
	ppuCyclesToRun := cpuCycles * 3
	for i := uint64(0); i < ppuCyclesToRun; i++ {
		b.PPU.Step()
		b.ppuCycles++
	}

	// APU runs at CPU speed
	for i := uint64(0); i < cpuCycles; i++ {
		b.APU.Step()
	}

	// Update counters
	b.cpuCycles += cpuCycles
	b.totalCycles += cpuCycles

	// Frame completion is now handled by PPU callback for precise timing

	// Check memory watchpoints for changes (reduced frequency for better performance)
	if b.watchpointLogging && b.frameCount%300 == 0 { // Check every 5 seconds at 60fps
		b.CheckMemoryWatchpoints()
	}

	// Log execution if enabled
	if b.loggingEnabled {
		event := BusExecutionEvent{
			StepNumber:    len(b.executionLog) + 1,
			CPUCycles:     b.cpuCycles,
			PPUCycles:     b.cpuCycles * 3, // PPU runs at 3x CPU speed
			FrameCount:    b.frameCount,
			DMAActive:     b.dmaInProgress,
			NMIProcessed:  b.frameCount > preFrameCount, // Frame count increased
			PCValue:       prePC,
			InstructionOp: preOpcode,
		}
		b.executionLog = append(b.executionLog, event)
	}
}

// TriggerOAMDMA initiates an OAM DMA transfer
func (b *Bus) TriggerOAMDMA(sourcePage uint8) {
	if b.dmaInProgress {
		return // DMA already in progress
	}

	// Calculate DMA duration: 513 cycles if starting on even CPU cycle, 514 if odd
	dmaCycles := uint64(513)
	if b.cpuCycles%2 == 1 {
		dmaCycles = 514
	}

	b.dmaInProgress = true
	b.dmaSuspendCycles = dmaCycles

	// Perform the actual OAM transfer
	sourceAddress := uint16(sourcePage) << 8
	for i := 0; i < 256; i++ {
		data := b.Memory.Read(sourceAddress + uint16(i))
		b.PPU.WriteOAM(uint8(i), data)
	}
}

// LoadCartridge loads a cartridge into the system
func (b *Bus) LoadCartridge(cart memory.CartridgeInterface) {
	// Update memory with cartridge
	b.Memory = memory.New(b.PPU, b.APU, cart)
	
	// Re-establish input system connection
	b.Memory.SetInputSystem(b.Input)
	
	b.CPU = cpu.New(b.Memory)

	// Create PPU memory with proper mirroring mode
	// We need to cast to check if the cartridge has mirroring info
	var mirrorMode memory.MirrorMode
	if cartridge, ok := cart.(*cartridge.Cartridge); ok {
		// Convert cartridge mirror mode to memory mirror mode
		switch cartridge.GetMirrorMode() {
		case 0: // MirrorHorizontal
			mirrorMode = memory.MirrorHorizontal
		case 1: // MirrorVertical
			mirrorMode = memory.MirrorVertical
		case 2: // MirrorSingleScreen0
			mirrorMode = memory.MirrorSingleScreen0
		case 3: // MirrorSingleScreen1
			mirrorMode = memory.MirrorSingleScreen1
		case 4: // MirrorFourScreen
			mirrorMode = memory.MirrorFourScreen
		default:
			mirrorMode = memory.MirrorHorizontal // Default to horizontal
		}
	} else {
		mirrorMode = memory.MirrorHorizontal // Default to horizontal
	}

	// Create and set PPU memory
	ppuMemory := memory.NewPPUMemory(cart, mirrorMode)
	b.PPU.SetMemory(ppuMemory)

	// Re-establish callbacks after recreating memory and CPU
	b.PPU.SetNMICallback(b.triggerNMI)
	b.Memory.SetDMACallback(b.TriggerOAMDMA)

	// Reset the CPU to properly initialize PC from reset vector
	b.CPU.Reset()
}

// Run runs the emulator for a specified number of frames
func (b *Bus) Run(frames int) {
	targetFrames := b.frameCount + uint64(frames)

	// Run until we complete the target number of frames
	for b.frameCount < targetFrames {
		b.Step()
	}
}

// RunCycles runs the emulator for a specified number of CPU cycles
func (b *Bus) RunCycles(cycles uint64) {
	targetCycles := b.cpuCycles + cycles

	for b.cpuCycles < targetCycles {
		b.Step()
	}
}

// GetFrameRate returns the current frame rate based on NTSC timing
func (b *Bus) GetFrameRate() float64 {
	// NTSC: CPU frequency ~1.789773 MHz, 29780.67 CPU cycles per frame
	cpuFrequency := 1789773.0
	cpuCyclesPerFrame := cpuFrequency / 60.098803 // NTSC frame rate
	return cpuFrequency / cpuCyclesPerFrame
}

// GetFrameBuffer returns the current PPU frame buffer
func (b *Bus) GetFrameBuffer() []uint32 {
	frameBuffer := b.PPU.GetFrameBuffer()
	return frameBuffer[:]
}

// GetAudioSamples returns the current audio samples from the APU
func (b *Bus) GetAudioSamples() []float32 {
	return b.APU.GetSamples()
}

// SetAudioSampleRate sets the target audio sample rate for the APU
func (b *Bus) SetAudioSampleRate(rate int) {
	b.APU.SetSampleRate(rate)
}

// GetCycleCount returns the current CPU cycle count
func (b *Bus) GetCycleCount() uint64 {
	return b.cpuCycles
}

// GetFrameCount returns the current frame count
func (b *Bus) GetFrameCount() uint64 {
	return b.frameCount
}

// IsDMAInProgress returns whether DMA is currently in progress
func (b *Bus) IsDMAInProgress() bool {
	return b.dmaInProgress
}

// isRenderingEnabled checks if PPU rendering is enabled
func (b *Bus) isRenderingEnabled() bool {
	// Read PPUMASK register to check if background or sprites are enabled
	mask := b.PPU.ReadRegister(0x2001)
	return (mask & 0x18) != 0 // Check bits 3 and 4 (show background/sprites)
}

// SetControllerButton sets the state of a controller button
func (b *Bus) SetControllerButton(controller int, button input.Button, pressed bool) {
	switch controller {
	case 0, 1: // Support both 0-based and 1-based indexing
		fmt.Printf("[BUS_DEBUG] SetControllerButton: controller=%d, button=%d, pressed=%t\n", 
			controller, uint8(button), pressed)
		b.Input.Controller1.SetButton(button, pressed)
	case 2:
		fmt.Printf("[BUS_DEBUG] SetControllerButton: controller=%d, button=%d, pressed=%t\n", 
			controller, uint8(button), pressed)
		b.Input.Controller2.SetButton(button, pressed)
	}
}

// SetControllerButtons sets all button states for a controller (array approach like ChibiNES/Fogleman)
func (b *Bus) SetControllerButtons(controller int, buttons [8]bool) {
	switch controller {
	case 0, 1: // Controller 1
		// Debug logging disabled for performance - uncomment if needed for debugging
		// fmt.Printf("[BUS_DEBUG] SetControllerButtons: controller=%d, buttons=[A:%t B:%t Sel:%t Start:%t U:%t D:%t L:%t R:%t]\n", 
		//	controller, buttons[0], buttons[1], buttons[2], buttons[3], buttons[4], buttons[5], buttons[6], buttons[7])
		b.Input.SetButtons1(buttons)
	case 2: // Controller 2
		// Debug logging disabled for performance - uncomment if needed for debugging
		// fmt.Printf("[BUS_DEBUG] SetControllerButtons: controller=%d, buttons=[A:%t B:%t Sel:%t Start:%t U:%t D:%t L:%t R:%t]\n", 
		//	controller, buttons[0], buttons[1], buttons[2], buttons[3], buttons[4], buttons[5], buttons[6], buttons[7])
		b.Input.SetButtons2(buttons)
	}
}

// EnableInputDebug enables debug logging for input system
func (b *Bus) EnableInputDebug(enable bool) {
	b.Input.EnableDebug(enable)
}

// GetInputState returns the input state for direct access
func (b *Bus) GetInputState() *input.InputState {
	return b.Input
}

// Frame executes one complete frame worth of cycles
func (b *Bus) Frame() {
	// NTSC: 29,781 CPU cycles per frame (89,342 PPU cycles / 3)
	targetCycles := b.cpuCycles + 29781

	for b.cpuCycles < targetCycles {
		b.Step()
	}
}

// GetExecutionLog returns execution log for integration testing
func (b *Bus) GetExecutionLog() []BusExecutionEvent {
	return b.executionLog
}

// EnableExecutionLogging enables execution logging for testing
func (b *Bus) EnableExecutionLogging() {
	b.loggingEnabled = true
}

// DisableExecutionLogging disables execution logging
func (b *Bus) DisableExecutionLogging() {
	b.loggingEnabled = false
}

// ClearExecutionLog clears the execution log
func (b *Bus) ClearExecutionLog() {
	b.executionLog = make([]BusExecutionEvent, 0)
}

// BusExecutionEvent represents a single execution step for testing
type BusExecutionEvent struct {
	StepNumber    int
	CPUCycles     uint64
	PPUCycles     uint64
	FrameCount    uint64
	DMAActive     bool
	NMIProcessed  bool
	PCValue       uint16
	InstructionOp uint8
}

// GetCPUState returns the current CPU state for testing
func (b *Bus) GetCPUState() CPUState {
	return CPUState{
		PC:     b.CPU.PC,
		A:      b.CPU.A,
		X:      b.CPU.X,
		Y:      b.CPU.Y,
		SP:     b.CPU.SP,
		Cycles: b.cpuCycles,
		Flags: CPUFlags{
			N: b.CPU.N,
			V: b.CPU.V,
			B: b.CPU.B,
			D: b.CPU.D,
			I: b.CPU.I,
			Z: b.CPU.Z,
			C: b.CPU.C,
		},
	}
}

// CPUState represents CPU state snapshot for testing
type CPUState struct {
	PC      uint16
	A, X, Y uint8
	SP      uint8
	Cycles  uint64
	Flags   CPUFlags
}

// CPUFlags represents CPU status flags for testing
type CPUFlags struct {
	N, V, B, D, I, Z, C bool
}

// GetPPUState returns the current PPU state for testing
func (b *Bus) GetPPUState() PPUState {
	// Simplified PPU state for testing
	scanline := int((b.ppuCycles % b.cyclesPerFrame) / 341)
	cycle := int((b.ppuCycles % b.cyclesPerFrame) % 341)

	return PPUState{
		Scanline:    scanline,
		Cycle:       cycle,
		FrameCount:  b.frameCount,
		VBlankFlag:  (b.PPU.ReadRegister(0x2002) & 0x80) != 0,
		RenderingOn: b.isRenderingEnabled(),
		NMIEnabled:  true, // Would need to expose this from PPU
	}
}

// PPUState represents PPU state snapshot for testing
type PPUState struct {
	Scanline    int
	Cycle       int
	FrameCount  uint64
	VBlankFlag  bool
	RenderingOn bool
	NMIEnabled  bool
}

// AddMemoryWatchpoint adds a memory address to monitor for changes
func (b *Bus) AddMemoryWatchpoint(address uint16) {
	if b.Memory != nil {
		b.memoryWatchpoints[address] = b.Memory.Read(address)
	}
}

// EnableWatchpointLogging enables/disables memory watchpoint logging
func (b *Bus) EnableWatchpointLogging(enabled bool) {
	b.watchpointLogging = enabled
}

// SetupSMBWatchpoints sets up memory watchpoints for Super Mario Bros debugging
func (b *Bus) SetupSMBWatchpoints() {
	// Known SMB memory locations for debugging
	addresses := []uint16{
		// Mario's coordinates and state
		0x0086, // Mario's horizontal position (low byte)
		0x0087, // Mario's horizontal position (high byte)
		0x00CE, // Mario's vertical position
		0x000E, // Mario's state (standing, jumping, etc.)
		0x001D, // Mario's power-up state

		// Coin counter
		0x07DE, // Coin count (ones)
		0x07DD, // Coin count (tens)

		// Score display
		0x07D7, // Score digit 1
		0x07D8, // Score digit 2
		0x07D9, // Score digit 3
		0x07DA, // Score digit 4
		0x07DB, // Score digit 5
		0x07DC, // Score digit 6

		// Critical game state
		0x0700, // Game state
		0x0770, // Player state
		0x075A, // Timer (hundreds)
		0x075B, // Timer (tens)
		0x075C, // Timer (ones)

		// Zero page critical variables
		0x0001, // Controller 1 input
		0x0002, // Controller 2 input
		0x00FF, // Stack pointer vicinity
		0x00FE, // Stack area
		0x00FD, // Stack area
	}

	for _, addr := range addresses {
		b.AddMemoryWatchpoint(addr)
	}

	fmt.Printf("[MEMORY_MONITOR] Set up %d watchpoints for SMB debugging\n", len(addresses))
}

// CheckMemoryWatchpoints checks all watchpoints for changes and logs them
func (b *Bus) CheckMemoryWatchpoints() {
	if !b.watchpointLogging || b.Memory == nil {
		return
	}

	for address, previousValue := range b.memoryWatchpoints {
		currentValue := b.Memory.Read(address)
		if currentValue != previousValue {
			fmt.Printf("[MEMORY_WATCH] Frame %d: $%04X changed from $%02X to $%02X (%s)\n",
				b.frameCount, address, previousValue, currentValue, b.getMemoryDescription(address))
			b.memoryWatchpoints[address] = currentValue
		}
	}
}

// getMemoryDescription returns a human-readable description of memory addresses
func (b *Bus) getMemoryDescription(address uint16) string {
	switch address {
	case 0x0086:
		return "Mario X pos (low)"
	case 0x0087:
		return "Mario X pos (high)"
	case 0x00CE:
		return "Mario Y pos"
	case 0x000E:
		return "Mario state"
	case 0x001D:
		return "Mario power-up"
	case 0x07DE:
		return "Coin count (ones)"
	case 0x07DD:
		return "Coin count (tens)"
	case 0x0700:
		return "Game state"
	case 0x0770:
		return "Player state"
	case 0x0001:
		return "Controller 1"
	case 0x0002:
		return "Controller 2"
	case 0x00FF:
		return "Stack pointer area"
	default:
		if address >= 0x07D7 && address <= 0x07DC {
			return fmt.Sprintf("Score digit %d", address-0x07D6)
		} else if address >= 0x075A && address <= 0x075C {
			return fmt.Sprintf("Timer %s", []string{"hundreds", "tens", "ones"}[address-0x075A])
		} else if address >= 0x0000 && address <= 0x00FF {
			return "Zero page"
		} else if address >= 0x0700 && address <= 0x07FF {
			return "WRAM upper"
		}
		return "Unknown"
	}
}

// CPU Debug Control Methods

// EnableCPUDebug enables/disables CPU debug logging and loop detection
func (b *Bus) EnableCPUDebug(enable bool) {
	if b.CPU != nil {
		b.CPU.EnableDebugLogging(enable)
		b.CPU.EnableLoopDetection(enable)
	}
}
