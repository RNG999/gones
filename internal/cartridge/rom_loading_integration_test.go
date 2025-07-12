package cartridge

import (
	"testing"
	"bytes"
	"gones/internal/memory"
)

// MockPPU implements memory.PPUInterface for testing
type MockPPU struct {
	registers  [8]uint8
	readCalls  []uint16
	writeCalls []RegisterWrite
}

type RegisterWrite struct {
	Address uint16
	Value   uint8
}

func (m *MockPPU) ReadRegister(address uint16) uint8 {
	m.readCalls = append(m.readCalls, address)
	return m.registers[address&0x7]
}

func (m *MockPPU) WriteRegister(address uint16, value uint8) {
	m.writeCalls = append(m.writeCalls, RegisterWrite{Address: address, Value: value})
	m.registers[address&0x7] = value
}

// MockAPU implements memory.APUInterface for testing
type MockAPU struct {
	registers  [0x18]uint8
	writeCalls []RegisterWrite
}

func (m *MockAPU) WriteRegister(address uint16, value uint8) {
	m.writeCalls = append(m.writeCalls, RegisterWrite{Address: address, Value: value})
	if address >= 0x4000 && address <= 0x4017 {
		m.registers[address-0x4000] = value
	}
}

func (m *MockAPU) ReadStatus() uint8 {
	return 0x00 // Mock implementation
}

// TestROMLoadingIntegration validates complete ROM loading and startup sequence
func TestROMLoadingIntegration(t *testing.T) {
	// Create comprehensive test ROM
	romBuilder := NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithResetVector(0x8000).
		WithNMIVector(0x8100).
		WithIRQVector(0x8200).
		WithData(0x0000, []uint8{
			0xA9, 0x42, // LDA #$42
			0x85, 0x00, // STA $00
			0x4C, 0x00, 0x80, // JMP $8000 (infinite loop)
		}).
		WithDescription("Integration test ROM")

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create integration test cartridge: %v", err)
	}

	// Test cartridge properties
	t.Run("Cartridge Properties", func(t *testing.T) {
		if cart.mapperID != 0 {
			t.Errorf("Mapper ID = %d, want 0", cart.mapperID)
		}

		if len(cart.prgROM) != 16384 {
			t.Errorf("PRG ROM size = %d, want 16384", len(cart.prgROM))
		}

		if len(cart.chrROM) != 8192 {
			t.Errorf("CHR ROM size = %d, want 8192", len(cart.chrROM))
		}
	})

	// Test ROM data integrity
	t.Run("ROM Data Integrity", func(t *testing.T) {
		// Verify instructions were loaded correctly
		expectedInstructions := []uint8{0xA9, 0x42, 0x85, 0x00, 0x4C, 0x00, 0x80}
		for i, expected := range expectedInstructions {
			actual := cart.ReadPRG(0x8000 + uint16(i))
			if actual != expected {
				t.Errorf("ROM[0x%04X] = 0x%02X, want 0x%02X", 
					0x8000+uint16(i), actual, expected)
			}
		}
	})

	// Test vector setup
	t.Run("Interrupt Vectors", func(t *testing.T) {
		// Reset vector
		resetLow := cart.ReadPRG(0xFFFC)
		resetHigh := cart.ReadPRG(0xFFFD)
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		if resetVector != 0x8000 {
			t.Errorf("Reset vector = 0x%04X, want 0x8000", resetVector)
		}

		// NMI vector
		nmiLow := cart.ReadPRG(0xFFFA)
		nmiHigh := cart.ReadPRG(0xFFFB)
		nmiVector := uint16(nmiLow) | (uint16(nmiHigh) << 8)
		if nmiVector != 0x8100 {
			t.Errorf("NMI vector = 0x%04X, want 0x8100", nmiVector)
		}

		// IRQ vector
		irqLow := cart.ReadPRG(0xFFFE)
		irqHigh := cart.ReadPRG(0xFFFF)
		irqVector := uint16(irqLow) | (uint16(irqHigh) << 8)
		if irqVector != 0x8200 {
			t.Errorf("IRQ vector = 0x%04X, want 0x8200", irqVector)
		}
	})

	// Test memory system integration
	t.Run("Memory System Integration", func(t *testing.T) {
		mockPPU := &MockPPU{}
		mockAPU := &MockAPU{}
		mem := memory.New(mockPPU, mockAPU, cart)

		// Test CPU can read ROM through memory system
		instruction := mem.Read(0x8000)
		if instruction != 0xA9 {
			t.Errorf("CPU read of first instruction = 0x%02X, want 0xA9", instruction)
		}

		// Test reset vector access through memory system
		resetLow := mem.Read(0xFFFC)
		resetHigh := mem.Read(0xFFFD)
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		if resetVector != 0x8000 {
			t.Errorf("Reset vector through memory = 0x%04X, want 0x8000", resetVector)
		}
	})

	// Test PPU memory integration
	t.Run("PPU Memory Integration", func(t *testing.T) {
		ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)

		// Test CHR ROM access
		chrValue := ppuMem.Read(0x0000)
		// CHR should be accessible (even if initialized to 0)
		_ = chrValue // Just verify no panic

		// Test CHR write (should work if CHR RAM, ignored if CHR ROM)
		ppuMem.Write(0x0000, 0x55)
		newValue := ppuMem.Read(0x0000)
		// Value behavior depends on CHR type, just verify no panic
		_ = newValue
	})
}

// TestROMLoadingFromBytes validates loading ROM from byte data
func TestROMLoadingFromBytes(t *testing.T) {
	// Generate test ROM data
	config := PrebuiltTestROMs.BasicTest
	romData, err := GenerateTestROM(config)
	if err != nil {
		t.Fatalf("Failed to generate test ROM: %v", err)
	}

	// Load cartridge from generated data
	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load cartridge from reader: %v", err)
	}

	// Validate cartridge matches expected configuration
	t.Run("Configuration Validation", func(t *testing.T) {
		if cart.mapperID != config.MapperID {
			t.Errorf("Mapper ID = %d, want %d", cart.mapperID, config.MapperID)
		}

		expectedPRGSize := int(config.PRGSize) * 16384
		if len(cart.prgROM) != expectedPRGSize {
			t.Errorf("PRG ROM size = %d, want %d", len(cart.prgROM), expectedPRGSize)
		}

		expectedCHRSize := int(config.CHRSize) * 8192
		if len(cart.chrROM) != expectedCHRSize {
			t.Errorf("CHR ROM size = %d, want %d", len(cart.chrROM), expectedCHRSize)
		}
	})

	// Test ROM content matches expected instructions
	t.Run("Instruction Validation", func(t *testing.T) {
		for i, expected := range config.Instructions {
			if i >= len(cart.prgROM) {
				break
			}
			actual := cart.prgROM[i]
			if actual != expected {
				t.Errorf("ROM[%d] = 0x%02X, want 0x%02X", i, actual, expected)
			}
		}
	})
}

// TestCompleteROMLifecycle validates the complete ROM lifecycle
func TestCompleteROMLifecycle(t *testing.T) {
	testCases := []struct {
		name   string
		config TestROMConfig
	}{
		{"Minimal ROM", PrebuiltTestROMs.MinimalNROM},
		{"Basic Test", PrebuiltTestROMs.BasicTest},
		{"Memory Test", PrebuiltTestROMs.MemoryTest},
		{"SRAM Test", PrebuiltTestROMs.SRAMTest},
		{"CHR RAM Test", PrebuiltTestROMs.CHRRAMTest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Step 1: Generate ROM
			romData, err := GenerateTestROM(tc.config)
			if err != nil {
				t.Fatalf("Failed to generate ROM: %v", err)
			}

			// Step 2: Load cartridge
			reader := bytes.NewReader(romData)
			cart, err := LoadFromReader(reader)
			if err != nil {
				t.Fatalf("Failed to load cartridge: %v", err)
			}

			// Step 3: Integrate with memory system
			mockPPU := &MockPPU{}
			mockAPU := &MockAPU{}
			mem := memory.New(mockPPU, mockAPU, cart)

			// Step 4: Test basic functionality
			// Test ROM access
			romValue := mem.Read(0x8000)
			_ = romValue // Just verify no panic

			// Test reset vector
			resetLow := mem.Read(0xFFFC)
			resetHigh := mem.Read(0xFFFD)
			resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
			if resetVector != tc.config.ResetVector {
				t.Errorf("Reset vector = 0x%04X, want 0x%04X", 
					resetVector, tc.config.ResetVector)
			}

			// Step 5: Test PPU integration
			ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
			chrValue := ppuMem.Read(0x0000)
			_ = chrValue // Just verify no panic
		})
	}
}

// TestErrorConditions validates error handling in ROM loading
func TestErrorConditions(t *testing.T) {
	testCases := []struct {
		name        string
		romData     []byte
		expectError bool
		description string
	}{
		{
			name:        "Invalid Magic",
			romData:     []byte{'B', 'A', 'D', 0x1A, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expectError: true,
			description: "Invalid magic number should fail",
		},
		{
			name:        "Zero PRG Size",
			romData:     []byte{'N', 'E', 'S', 0x1A, 0, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expectError: true,
			description: "Zero PRG ROM size should fail",
		},
		{
			name:        "Truncated Header",
			romData:     []byte{'N', 'E', 'S', 0x1A, 1},
			expectError: true,
			description: "Truncated header should fail",
		},
		{
			name:        "Missing PRG Data",
			romData:     []byte{'N', 'E', 'S', 0x1A, 1, 1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0},
			expectError: true,
			description: "Missing PRG ROM data should fail",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bytes.NewReader(tc.romData)
			cart, err := LoadFromReader(reader)

			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error but got none (%s)", tc.description)
				}
				if cart != nil {
					t.Errorf("Expected nil cartridge on error but got %v", cart)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v (%s)", err, tc.description)
				}
				if cart == nil {
					t.Errorf("Expected valid cartridge but got nil")
				}
			}
		})
	}
}

// TestROMStartupSequence validates typical ROM startup behavior
func TestROMStartupSequence(t *testing.T) {
	// Create ROM with typical startup sequence
	instructions := []uint8{
		// Reset handler at $8000
		0x78,       // SEI (disable interrupts)
		0xD8,       // CLD (clear decimal mode)
		0xA2, 0xFF, // LDX #$FF
		0x9A,       // TXS (set stack pointer)
		0xA9, 0x00, // LDA #$00
		0x85, 0x00, // STA $00 (clear zero page)
		0x4C, 0x00, 0x80, // JMP $8000 (infinite loop)
	}

	romBuilder := NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithResetVector(0x8000).
		WithInstructions(instructions).
		WithDescription("Startup sequence test")

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create startup test cartridge: %v", err)
	}

	// Test memory integration
	mockPPU := &MockPPU{}
	mockAPU := &MockAPU{}
	mem := memory.New(mockPPU, mockAPU, cart)

	// Verify startup sequence is accessible
	t.Run("Startup Sequence Access", func(t *testing.T) {
		for i, expected := range instructions {
			actual := mem.Read(0x8000 + uint16(i))
			if actual != expected {
				t.Errorf("Instruction[%d] at 0x%04X = 0x%02X, want 0x%02X",
					i, 0x8000+uint16(i), actual, expected)
			}
		}
	})

	// Verify reset vector points to startup code
	t.Run("Reset Vector Validation", func(t *testing.T) {
		resetLow := mem.Read(0xFFFC)
		resetHigh := mem.Read(0xFFFD)
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		
		if resetVector != 0x8000 {
			t.Errorf("Reset vector = 0x%04X, want 0x8000", resetVector)
		}

		// Verify first instruction at reset vector
		firstInstruction := mem.Read(resetVector)
		if firstInstruction != 0x78 { // SEI
			t.Errorf("First instruction = 0x%02X, want 0x78 (SEI)", firstInstruction)
		}
	})
}

// TestCartridgeInterfaceCompliance validates cartridge interface implementation
func TestCartridgeInterfaceCompliance(t *testing.T) {
	// Create test cartridge
	cart, err := NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithResetVector(0x8000).
		BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}

	// Test that cartridge implements CartridgeInterface
	var cartInterface memory.CartridgeInterface = cart

	// Test PRG access methods
	t.Run("PRG Interface Methods", func(t *testing.T) {
		// Test read
		value := cartInterface.ReadPRG(0x8000)
		_ = value // Just verify method exists and doesn't panic

		// Test write (should be handled gracefully)
		cartInterface.WritePRG(0x8000, 0x42)
		// NROM ignores writes to ROM, should not panic
	})

	// Test CHR access methods
	t.Run("CHR Interface Methods", func(t *testing.T) {
		// Test read
		value := cartInterface.ReadCHR(0x0000)
		_ = value // Just verify method exists and doesn't panic

		// Test write
		cartInterface.WriteCHR(0x0000, 0x55)
		// Should handle CHR ROM vs CHR RAM appropriately
	})

	// Test that interface methods work through memory system
	t.Run("Interface Through Memory", func(t *testing.T) {
		mockPPU := &MockPPU{}
		mockAPU := &MockAPU{}
		mem := memory.New(mockPPU, mockAPU, cartInterface)

		// Test ROM access through memory
		romValue := mem.Read(0x8000)
		_ = romValue

		// Test PPU memory access
		ppuMem := memory.NewPPUMemory(cartInterface, memory.MirrorHorizontal)
		chrValue := ppuMem.Read(0x0000)
		_ = chrValue
	})
}