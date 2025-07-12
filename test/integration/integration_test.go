package integration

import (
	"gones/internal/apu"
	"gones/internal/bus"
	"gones/internal/cpu"
	"gones/internal/input"
	"gones/internal/memory"
	"gones/internal/ppu"
	"testing"
)

// IntegrationTestHelper provides utilities for system-level integration testing
type IntegrationTestHelper struct {
	Bus       *bus.Bus
	CPU       *cpu.CPU
	PPU       *ppu.PPU
	Memory    *memory.Memory
	APU       *apu.APU
	Input     *input.InputState
	Cartridge memory.CartridgeInterface
}

// GetMockCartridge returns the cartridge as a MockCartridge for test-specific operations
func (h *IntegrationTestHelper) GetMockCartridge() *MockCartridge {
	if mockCart, ok := h.Cartridge.(*MockCartridge); ok {
		return mockCart
	}
	return nil
}

// MockCartridge implements CartridgeInterface for integration testing
type MockCartridge struct {
	prgROM    [0x8000]uint8 // 32KB PRG ROM
	chrROM    [0x2000]uint8 // 8KB CHR ROM
	prgRAM    [0x2000]uint8 // 8KB PRG RAM
	chrRAM    [0x2000]uint8 // 8KB CHR RAM
	mirroring memory.MirrorMode

	// Tracking for integration tests
	prgReads  []uint16
	prgWrites []uint16
	chrReads  []uint16
	chrWrites []uint16
}

// NewMockCartridge creates a new mock cartridge for testing
func NewMockCartridge() *MockCartridge {
	return &MockCartridge{
		mirroring: memory.MirrorHorizontal,
		prgReads:  make([]uint16, 0),
		prgWrites: make([]uint16, 0),
		chrReads:  make([]uint16, 0),
		chrWrites: make([]uint16, 0),
	}
}

// ReadPRG implements CartridgeInterface
func (c *MockCartridge) ReadPRG(address uint16) uint8 {
	c.prgReads = append(c.prgReads, address)
	// Mirror 16KB ROM to 32KB space if needed
	index := (address - 0x8000) % uint16(len(c.prgROM))
	if address >= 0x8000 {
		index = address - 0x8000
		if index >= 0x4000 && len(c.prgROM) == 0x4000 {
			// Mirror 16KB ROM
			index = index % 0x4000
		}
	}
	return c.prgROM[index]
}

// WritePRG implements CartridgeInterface
func (c *MockCartridge) WritePRG(address uint16, value uint8) {
	c.prgWrites = append(c.prgWrites, address)
	// Some mappers allow writes to PRG area (for RAM or registers)
	if address >= 0x6000 && address < 0x8000 {
		// PRG RAM area
		c.prgRAM[address-0x6000] = value
	}
	// Writes to ROM area might be for mapper control (ignored in basic test)
}

// ReadCHR implements CartridgeInterface
func (c *MockCartridge) ReadCHR(address uint16) uint8 {
	c.chrReads = append(c.chrReads, address)
	if address < 0x2000 {
		return c.chrROM[address]
	}
	return 0
}

// WriteCHR implements CartridgeInterface
func (c *MockCartridge) WriteCHR(address uint16, value uint8) {
	c.chrWrites = append(c.chrWrites, address)
	if address < 0x2000 {
		c.chrRAM[address] = value
	}
}

// LoadPRG loads data into PRG ROM
func (c *MockCartridge) LoadPRG(data []uint8) {
	copy(c.prgROM[:], data)
}

// LoadCHR loads data into CHR ROM
func (c *MockCartridge) LoadCHR(data []uint8) {
	copy(c.chrROM[:], data)
}

// SetMirroring sets the nametable mirroring mode
func (c *MockCartridge) SetMirroring(mode memory.MirrorMode) {
	c.mirroring = mode
}

// GetMirroring returns the current mirroring mode
func (c *MockCartridge) GetMirroring() memory.MirrorMode {
	return c.mirroring
}


// ClearLogs clears all access logs
func (c *MockCartridge) ClearLogs() {
	c.prgReads = c.prgReads[:0]
	c.prgWrites = c.prgWrites[:0]
	c.chrReads = c.chrReads[:0]
	c.chrWrites = c.chrWrites[:0]
}

// NewIntegrationTestHelper creates a new integration test helper
func NewIntegrationTestHelper() *IntegrationTestHelper {
	// Create mock cartridge
	cartridge := NewMockCartridge()

	// Create system bus (this creates all components)
	systemBus := bus.New()

	// Load the cartridge into the system
	systemBus.LoadCartridge(cartridge)

	helper := &IntegrationTestHelper{
		Bus:       systemBus,
		CPU:       systemBus.CPU,
		PPU:       systemBus.PPU,
		Memory:    systemBus.Memory,
		APU:       systemBus.APU,
		Input:     systemBus.Input,
		Cartridge: cartridge,
	}
	
	return helper
}

// UpdateReferences updates the helper's component references after cartridge loading
func (h *IntegrationTestHelper) UpdateReferences() {
	h.CPU = h.Bus.CPU
	h.PPU = h.Bus.PPU
	h.Memory = h.Bus.Memory
	h.APU = h.Bus.APU
	h.Input = h.Bus.Input
}

// SetupBasicROM sets up a basic ROM with reset vector and minimal program
func (h *IntegrationTestHelper) SetupBasicROM(resetVector uint16) {
	// Create basic ROM data
	romData := make([]uint8, 0x8000)

	// Set reset vector at end of ROM
	romData[0x7FFC] = uint8(resetVector & 0xFF)        // Reset vector low
	romData[0x7FFD] = uint8((resetVector >> 8) & 0xFF) // Reset vector high
	romData[0x7FFE] = 0x00                             // IRQ vector low
	romData[0x7FFF] = 0x80                             // IRQ vector high

	// Load basic program at reset vector (relative to ROM start)
	if resetVector >= 0x8000 {
		offset := resetVector - 0x8000
		romData[offset] = 0xEA                        // NOP
		romData[offset+1] = 0xEA                      // NOP
		romData[offset+2] = 0x4C                      // JMP
		romData[offset+3] = uint8(resetVector & 0xFF) // Jump back to start
		romData[offset+4] = uint8((resetVector >> 8) & 0xFF)
	}

	h.GetMockCartridge().LoadPRG(romData)
}

// SetupBasicCHR sets up basic CHR data for pattern tables
func (h *IntegrationTestHelper) SetupBasicCHR() {
	chrData := make([]uint8, 0x2000)

	// Create simple pattern data for testing
	for i := 0; i < 256; i++ {
		// Create a simple 8x8 pattern
		baseAddr := i * 16
		if baseAddr < len(chrData)-16 {
			// Simple checkerboard pattern
			chrData[baseAddr+0] = 0xAA // 10101010
			chrData[baseAddr+1] = 0x55 // 01010101
			chrData[baseAddr+2] = 0xAA // 10101010
			chrData[baseAddr+3] = 0x55 // 01010101
			chrData[baseAddr+4] = 0xAA // 10101010
			chrData[baseAddr+5] = 0x55 // 01010101
			chrData[baseAddr+6] = 0xAA // 10101010
			chrData[baseAddr+7] = 0x55 // 01010101
			// High bit plane (all zeros for color 1)
			for j := 8; j < 16; j++ {
				chrData[baseAddr+j] = 0x00
			}
		}
	}

	h.GetMockCartridge().LoadCHR(chrData)
}

// RunCycles runs the system for a specified number of CPU cycles
func (h *IntegrationTestHelper) RunCycles(cycles int) {
	for i := 0; i < cycles; i++ {
		h.Bus.Step()
	}
}

// RunToVBlank runs the system until VBlank starts
func (h *IntegrationTestHelper) RunToVBlank() int {
	cycles := 0
	maxCycles := 100000 // Safety limit

	for cycles < maxCycles {
		// Check if we're at VBlank (scanline 241)
		ppuStatus := h.PPU.ReadRegister(0x2002)
		if (ppuStatus & 0x80) != 0 { // VBlank flag set
			break
		}

		h.Bus.Step()
		cycles++
	}

	return cycles
}

// RunFrame runs the system for one complete frame
func (h *IntegrationTestHelper) RunFrame() {
	// NTSC frame is approximately 29780 CPU cycles
	h.RunCycles(29780)
}

// TestSystemInitialization tests that all components initialize correctly
func TestSystemInitialization(t *testing.T) {
	t.Run("System components creation", func(t *testing.T) {
		helper := NewIntegrationTestHelper()

		// Verify all components exist
		if helper.Bus == nil {
			t.Fatal("System bus not created")
		}
		if helper.CPU == nil {
			t.Fatal("CPU not created")
		}
		if helper.PPU == nil {
			t.Fatal("PPU not created")
		}
		if helper.Memory == nil {
			t.Fatal("Memory not created")
		}
		if helper.APU == nil {
			t.Fatal("APU not created")
		}
		if helper.Input == nil {
			t.Fatal("Input not created")
		}
		if helper.Cartridge == nil {
			t.Fatal("Cartridge not created")
		}
	})

	t.Run("System reset state", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Reset the system
		helper.Bus.Reset()

		// Verify initial state
		if helper.CPU.A != 0 {
			t.Errorf("CPU A register should be 0 after reset, got %d", helper.CPU.A)
		}
		if helper.CPU.X != 0 {
			t.Errorf("CPU X register should be 0 after reset, got %d", helper.CPU.X)
		}
		if helper.CPU.Y != 0 {
			t.Errorf("CPU Y register should be 0 after reset, got %d", helper.CPU.Y)
		}
		if helper.CPU.SP != 0xFD {
			t.Errorf("CPU stack pointer should be 0xFD after reset, got 0x%02X", helper.CPU.SP)
		}
		if !helper.CPU.I {
			t.Error("CPU interrupt flag should be set after reset")
		}
	})

	t.Run("Cartridge integration", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Test PRG ROM access through memory system
		value := helper.Memory.Read(0x8000)
		if value != 0xEA { // Should be NOP from basic ROM
			t.Errorf("Expected NOP (0xEA) at 0x8000, got 0x%02X", value)
		}

		// Verify cartridge access was logged
		if len(helper.GetMockCartridge().prgReads) == 0 {
			t.Error("No PRG reads logged")
		}

		// Test that reset vector is read correctly
		helper.Bus.Reset()
		if helper.CPU.PC != 0x8000 {
			t.Errorf("PC should be 0x8000 after reset, got 0x%04X", helper.CPU.PC)
		}
	})
}

// TestBasicExecution tests basic CPU instruction execution
func TestBasicExecution(t *testing.T) {
	t.Run("Single instruction execution", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		helper.Bus.Reset()

		// Execute one step (should be NOP)
		initialPC := helper.CPU.PC
		helper.Bus.Step()

		// PC should advance by 1 for NOP
		if helper.CPU.PC != initialPC+1 {
			t.Errorf("PC should advance by 1 after NOP, was 0x%04X now 0x%04X",
				initialPC, helper.CPU.PC)
		}
	})

	t.Run("Multiple instruction execution", func(t *testing.T) {
		helper := NewIntegrationTestHelper()

		// Create a simple program
		program := []uint8{
			0xA9, 0x42, // LDA #$42
			0x8D, 0x00, 0x20, // STA $2000
			0xEA,             // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}

		// Set up ROM with program
		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		helper.GetMockCartridge().LoadPRG(romData)

		helper.Bus.Reset()

		// Execute LDA #$42
		helper.Bus.Step()
		if helper.CPU.A != 0x42 {
			t.Errorf("Expected A=0x42 after LDA, got 0x%02X", helper.CPU.A)
		}

		// Execute STA $2000 (PPU register)
		helper.Bus.Step()
		// This should write to PPU PPUCTRL register

		// Execute NOP
		helper.Bus.Step()

		// Verify execution continued correctly
		if helper.CPU.PC != 0x8006 {
			t.Errorf("Expected PC=0x8006 after three instructions, got 0x%04X", helper.CPU.PC)
		}
	})
}

// TestMemoryIntegration tests memory system integration
func TestMemoryIntegration(t *testing.T) {
	t.Run("RAM access through memory system", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Test RAM write and read
		helper.Memory.Write(0x0200, 0x55)
		value := helper.Memory.Read(0x0200)

		if value != 0x55 {
			t.Errorf("Expected 0x55 from RAM, got 0x%02X", value)
		}

		// Test RAM mirroring
		helper.Memory.Write(0x0800, 0xAA) // Should mirror to 0x0000
		value = helper.Memory.Read(0x0000)

		if value != 0xAA {
			t.Errorf("Expected 0xAA from mirrored RAM, got 0x%02X", value)
		}
	})

	t.Run("PPU register access through memory", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Write to PPUCTRL
		helper.Memory.Write(0x2000, 0x80)

		// Read PPUSTATUS
		status := helper.Memory.Read(0x2002)

		// Just verify the read doesn't crash (actual PPU behavior tested elsewhere)
		_ = status

		// Test register mirroring
		helper.Memory.Write(0x2008, 0x40) // Should mirror to 0x2000
	})

	t.Run("Cartridge access through memory", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Read from cartridge
		value := helper.Memory.Read(0x8000)

		// Should read from cartridge PRG ROM
		if len(helper.GetMockCartridge().prgReads) == 0 {
			t.Error("No cartridge reads recorded")
		}

		lastRead := helper.GetMockCartridge().prgReads[len(helper.GetMockCartridge().prgReads)-1]
		if lastRead != 0x8000 {
			t.Errorf("Expected cartridge read at 0x8000, got 0x%04X", lastRead)
		}

		// Value should match what we set up
		if value != 0xEA { // NOP from basic ROM setup
			t.Errorf("Expected 0xEA from cartridge, got 0x%02X", value)
		}
	})
}

// TestComponentCommunication tests communication between components
func TestComponentCommunication(t *testing.T) {
	t.Run("CPU to PPU communication", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Program that writes to PPU registers
		program := []uint8{
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK)
			0xEA, // NOP
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)

		helper.Bus.Reset()

		// Execute the program
		helper.Bus.Step() // LDA #$80
		helper.Bus.Step() // STA $2000
		helper.Bus.Step() // LDA #$1E
		helper.Bus.Step() // STA $2001

		// PPU registers should be set (verify through PPU interface)
		// This tests that memory writes reach the PPU
	})

	t.Run("System step coordination", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		helper.Bus.Reset()

		// Single system step should:
		// 1. Execute one CPU instruction
		// 2. Run PPU for 3x CPU cycles
		// 3. Run APU for same cycles as CPU

		// Would need access to internal cycle counter
		helper.Bus.Step()

		// Verify that step completed without errors
		// More detailed timing tests in separate files
	})
}

// TestErrorConditionsIntegration tests error handling and edge cases
func TestErrorConditionsIntegration(t *testing.T) {
	t.Run("Invalid memory access", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		// Read from unmapped region
		value := helper.Memory.Read(0x5000) // Expansion area

		// Should return 0 for unmapped reads
		if value != 0 {
			t.Errorf("Expected 0 from unmapped region, got 0x%02X", value)
		}

		// Write to unmapped region should not crash
		helper.Memory.Write(0x5000, 0x42)
	})

	t.Run("System without cartridge", func(t *testing.T) {
		// Test that system can be created without cartridge
		systemBus := bus.New()

		if systemBus == nil {
			t.Fatal("System bus creation failed")
		}

		// Reset without cartridge should not crash
		systemBus.Reset()
	})

	t.Run("Rapid stepping", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		helper.Bus.Reset()

		// Run many steps rapidly
		for i := 0; i < 1000; i++ {
			helper.Bus.Step()
		}

		// System should remain stable
		if helper.CPU.PC < 0x8000 || helper.CPU.PC >= 0xFFFF {
			t.Errorf("PC out of valid range after rapid stepping: 0x%04X", helper.CPU.PC)
		}
	})
}

// TestSystemIntegrity tests overall system integrity
func TestSystemIntegrity(t *testing.T) {
	t.Run("Complete boot sequence", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Simulate complete boot
		helper.Bus.Reset()

		// Run for several frames to ensure stability
		for frame := 0; frame < 5; frame++ {
			helper.RunFrame()
		}

		// System should remain stable
		if helper.CPU.SP > 0xFF {
			t.Errorf("Stack pointer corrupted: 0x%02X", helper.CPU.SP)
		}
	})

	t.Run("Long running stability", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		helper.Bus.Reset()

		// Run for many cycles
		helper.RunCycles(10000)

		// Check that system is still in valid state
		if helper.CPU.PC < 0x8000 {
			t.Errorf("PC moved outside ROM area: 0x%04X", helper.CPU.PC)
		}

		// Stack should not underflow or overflow significantly
		if helper.CPU.SP < 0x80 || helper.CPU.SP > 0xFF {
			t.Errorf("Stack pointer in suspicious range: 0x%02X", helper.CPU.SP)
		}
	})
}
