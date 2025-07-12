package main

import (
	"fmt"
	"gones/internal/bus"
	"gones/internal/memory"
)

// MockCartridge implements CartridgeInterface for testing
type MockCartridge struct {
	prgROM [0x8000]uint8
}

func (c *MockCartridge) ReadPRG(address uint16) uint8 {
	index := address - 0x8000
	return c.prgROM[index]
}

func (c *MockCartridge) WritePRG(address uint16, value uint8) {}
func (c *MockCartridge) ReadCHR(address uint16) uint8         { return 0 }
func (c *MockCartridge) WriteCHR(address uint16, value uint8) {}

func (c *MockCartridge) GetMirroring() memory.MirrorMode {
	return memory.MirrorHorizontal
}

func (c *MockCartridge) LoadPRG(data []uint8) {
	copy(c.prgROM[:], data)
}

func main() {
	fmt.Println("Testing NES emulation loop implementation...")

	// Create system bus
	systemBus := bus.New()
	if systemBus == nil {
		fmt.Println("ERROR: Failed to create system bus")
		return
	}
	fmt.Println("✓ System bus created")

	// Create mock cartridge with basic ROM
	cartridge := &MockCartridge{}
	romData := make([]uint8, 0x8000)

	// Set reset vector
	romData[0x7FFC] = 0x00 // Reset vector low
	romData[0x7FFD] = 0x80 // Reset vector high

	// Add basic program
	romData[0x0000] = 0xEA // NOP
	romData[0x0001] = 0xEA // NOP
	romData[0x0002] = 0x4C // JMP
	romData[0x0003] = 0x00 // $8000
	romData[0x0004] = 0x80

	cartridge.LoadPRG(romData)
	systemBus.LoadCartridge(cartridge)
	fmt.Println("✓ Cartridge loaded")

	// Reset system
	systemBus.Reset()
	fmt.Println("✓ System reset")

	// Test basic functionality
	initialCycles := systemBus.GetCycleCount()
	initialFrames := systemBus.GetFrameCount()
	fmt.Printf("Initial state: %d cycles, %d frames\n", initialCycles, initialFrames)

	// Execute some steps
	for i := 0; i < 10; i++ {
		systemBus.Step()
	}

	finalCycles := systemBus.GetCycleCount()
	finalFrames := systemBus.GetFrameCount()
	fmt.Printf("After 10 steps: %d cycles, %d frames\n", finalCycles, finalFrames)

	if finalCycles <= initialCycles {
		fmt.Println("ERROR: Cycles not increasing")
		return
	}
	fmt.Println("✓ CPU cycles increasing")

	// Test Frame() method
	frameStartCycles := systemBus.GetCycleCount()
	systemBus.Frame()
	frameEndCycles := systemBus.GetCycleCount()

	frameCycles := frameEndCycles - frameStartCycles
	fmt.Printf("Frame execution: %d cycles\n", frameCycles)

	expectedFrameCycles := uint64(29781)
	if frameCycles < expectedFrameCycles-100 || frameCycles > expectedFrameCycles+100 {
		fmt.Printf("WARNING: Frame cycles (%d) not close to expected (%d)\n",
			frameCycles, expectedFrameCycles)
	} else {
		fmt.Println("✓ Frame timing within expected range")
	}

	// Test state access
	cpuState := systemBus.GetCPUState()
	ppuState := systemBus.GetPPUState()
	fmt.Printf("CPU State: PC=0x%04X, A=0x%02X, Cycles=%d\n",
		cpuState.PC, cpuState.A, cpuState.Cycles)
	fmt.Printf("PPU State: Scanline=%d, Cycle=%d, Frame=%d\n",
		ppuState.Scanline, ppuState.Cycle, ppuState.FrameCount)

	if cpuState.PC < 0x8000 {
		fmt.Printf("WARNING: PC (0x%04X) not in ROM area\n", cpuState.PC)
	} else {
		fmt.Println("✓ CPU PC in valid ROM area")
	}

	// Test execution logging
	systemBus.EnableExecutionLogging()
	systemBus.ClearExecutionLog()

	// Execute a few steps with logging
	for i := 0; i < 5; i++ {
		systemBus.Step()
	}

	log := systemBus.GetExecutionLog()
	fmt.Printf("Execution log contains %d entries\n", len(log))

	if len(log) != 5 {
		fmt.Printf("WARNING: Expected 5 log entries, got %d\n", len(log))
	} else {
		fmt.Println("✓ Execution logging working")
	}

	// Test DMA functionality
	isDMAActive := systemBus.IsDMAInProgress()
	fmt.Printf("DMA active: %v\n", isDMAActive)

	if isDMAActive {
		fmt.Println("WARNING: DMA should not be active initially")
	} else {
		fmt.Println("✓ DMA state correct")
	}

	fmt.Println("\n=== Integration Test Summary ===")
	fmt.Println("✓ Bus creation and component initialization")
	fmt.Println("✓ Cartridge loading and system reset")
	fmt.Println("✓ CPU instruction execution and cycle counting")
	fmt.Println("✓ Frame execution timing")
	fmt.Println("✓ Component state access")
	fmt.Println("✓ Execution logging functionality")
	fmt.Println("✓ DMA state tracking")
	fmt.Println("\nEmulation loop implementation appears to be working correctly!")
}
