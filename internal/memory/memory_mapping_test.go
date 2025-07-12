package memory

import (
	"testing"
	"gones/internal/cartridge"
)

// TestMemoryMappingNROM128 validates NROM-128 (16KB) memory mapping behavior
func TestMemoryMappingNROM128(t *testing.T) {
	// Create 16KB ROM with distinct patterns in different regions
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1). // 16KB
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0x10, 0x20, 0x30, 0x40}). // Start pattern
		WithData(0x1000, []uint8{0x11, 0x21, 0x31, 0x41}). // 4KB offset
		WithData(0x2000, []uint8{0x12, 0x22, 0x32, 0x42}). // 8KB offset
		WithData(0x3000, []uint8{0x13, 0x23, 0x33, 0x43})  // 12KB offset

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create NROM-128 test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	// Test mapping: $8000-$BFFF contains the ROM, $C000-$FFFF mirrors it
	testCases := []struct {
		name      string
		addr1     uint16 // Address in first bank
		addr2     uint16 // Address in mirrored bank
		expected  uint8  // Expected value
		description string
	}{
		{"Start Mirror", 0x8000, 0xC000, 0x10, "ROM start mirrors correctly"},
		{"Start+1 Mirror", 0x8001, 0xC001, 0x20, "ROM start+1 mirrors correctly"},
		{"4KB Mirror", 0x9000, 0xD000, 0x11, "4KB offset mirrors correctly"},
		{"4KB+1 Mirror", 0x9001, 0xD001, 0x21, "4KB+1 offset mirrors correctly"},
		{"8KB Mirror", 0xA000, 0xE000, 0x12, "8KB offset mirrors correctly"},
		{"8KB+1 Mirror", 0xA001, 0xE001, 0x22, "8KB+1 offset mirrors correctly"},
		{"12KB Mirror", 0xB000, 0xF000, 0x13, "12KB offset mirrors correctly"},
		{"12KB+1 Mirror", 0xB001, 0xF001, 0x23, "12KB+1 offset mirrors correctly"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Read from both addresses
			val1 := mem.Read(tc.addr1)
			val2 := mem.Read(tc.addr2)

			// Both should match expected value
			if val1 != tc.expected {
				t.Errorf("Read(0x%04X) = 0x%02X, want 0x%02X", tc.addr1, val1, tc.expected)
			}
			if val2 != tc.expected {
				t.Errorf("Read(0x%04X) = 0x%02X, want 0x%02X", tc.addr2, val2, tc.expected)
			}

			// Values should be identical (mirrored)
			if val1 != val2 {
				t.Errorf("Mirror mismatch: 0x%04X=0x%02X, 0x%04X=0x%02X (%s)",
					tc.addr1, val1, tc.addr2, val2, tc.description)
			}
		})
	}
}

// TestMemoryMappingNROM256 validates NROM-256 (32KB) memory mapping behavior
func TestMemoryMappingNROM256(t *testing.T) {
	// Create 32KB ROM with distinct patterns in each 16KB bank
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(2). // 32KB
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0xA0, 0xA1, 0xA2, 0xA3}). // First bank start
		WithData(0x3000, []uint8{0xAF, 0xAE, 0xAD, 0xAC}). // First bank test area
		WithData(0x4000, []uint8{0xB0, 0xB1, 0xB2, 0xB3}). // Second bank start
		WithData(0x7000, []uint8{0xBF, 0xBE, 0xBD, 0xBC})  // Second bank test area

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create NROM-256 test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	// Test that banks are distinct (no mirroring in 32KB ROM)
	testCases := []struct {
		name         string
		firstBank    uint16
		secondBank   uint16
		expectedFirst uint8
		expectedSecond uint8
		description  string
	}{
		{"Bank Start", 0x8000, 0xC000, 0xA0, 0xB0, "First bytes differ between banks"},
		{"Bank Start+1", 0x8001, 0xC001, 0xA1, 0xB1, "Second bytes differ between banks"},
		{"Bank Start+2", 0x8002, 0xC002, 0xA2, 0xB2, "Third bytes differ between banks"},
		{"Bank Start+3", 0x8003, 0xC003, 0xA3, 0xB3, "Fourth bytes differ between banks"},
		{"Bank Test-3", 0xB000, 0xF000, 0xAF, 0xBF, "Test area bytes differ between banks"},
		{"Bank Test-2", 0xB001, 0xF001, 0xAE, 0xBE, "Test area+1 bytes differ between banks"},
		{"Bank Test-1", 0xB002, 0xF002, 0xAD, 0xBD, "Test area+2 bytes differ between banks"},
		{"Bank Test", 0xB003, 0xF003, 0xAC, 0xBC, "Test area+3 bytes differ between banks"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			val1 := mem.Read(tc.firstBank)
			val2 := mem.Read(tc.secondBank)

			if val1 != tc.expectedFirst {
				t.Errorf("First bank Read(0x%04X) = 0x%02X, want 0x%02X",
					tc.firstBank, val1, tc.expectedFirst)
			}
			if val2 != tc.expectedSecond {
				t.Errorf("Second bank Read(0x%04X) = 0x%02X, want 0x%02X",
					tc.secondBank, val2, tc.expectedSecond)
			}

			// Values should be different (no mirroring)
			if val1 == val2 {
				t.Errorf("Banks should differ but both = 0x%02X (%s)",
					val1, tc.description)
			}
		})
	}
}

// TestCHRMirroringModes validates different CHR mirroring configurations
func TestCHRMirroringModes(t *testing.T) {
	// Create CHR ROM with distinct patterns
	chrData := make([]uint8, 8192)
	for i := 0; i < len(chrData); i++ {
		chrData[i] = uint8((i / 1024) + 1) // 1, 2, 3, 4, 5, 6, 7, 8 for each 1KB
	}

	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithCHRData(chrData).
		WithResetVector(0x8000)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create CHR test cartridge: %v", err)
	}

	// Test that CHR ROM data is accessible
	ppuMem := NewPPUMemory(cart, MirrorHorizontal)

	testCases := []struct {
		name     string
		address  uint16
		expected uint8
	}{
		{"CHR Block 0", 0x0000, 1}, // First 1KB block
		{"CHR Block 1", 0x0400, 2}, // Second 1KB block
		{"CHR Block 2", 0x0800, 3}, // Third 1KB block
		{"CHR Block 3", 0x0C00, 4}, // Fourth 1KB block
		{"CHR Block 4", 0x1000, 5}, // Fifth 1KB block
		{"CHR Block 5", 0x1400, 6}, // Sixth 1KB block
		{"CHR Block 6", 0x1800, 7}, // Seventh 1KB block
		{"CHR Block 7", 0x1C00, 8}, // Eighth 1KB block
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ppuMem.Read(tc.address)
			if result != tc.expected {
				t.Errorf("CHR Read(0x%04X) = %d, want %d", tc.address, result, tc.expected)
			}
		})
	}
}

// TestNametableMirroringModes validates nametable mirroring behavior
func TestNametableMirroringModes(t *testing.T) {
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithResetVector(0x8000)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create mirroring test cartridge: %v", err)
	}

	mirroringModes := []struct {
		name string
		mode MirrorMode
	}{
		{"Horizontal", MirrorHorizontal},
		{"Vertical", MirrorVertical},
		{"Single Screen 0", MirrorSingleScreen0},
		{"Single Screen 1", MirrorSingleScreen1},
	}

	for _, mm := range mirroringModes {
		t.Run(mm.name, func(t *testing.T) {
			ppuMem := NewPPUMemory(cart, mm.mode)

			// Write unique patterns to each nametable
			nametableData := []struct {
				address uint16
				value   uint8
			}{
				{0x2000, 0x10}, // Nametable 0
				{0x2400, 0x20}, // Nametable 1
				{0x2800, 0x30}, // Nametable 2
				{0x2C00, 0x40}, // Nametable 3
			}

			// Write data
			for _, data := range nametableData {
				ppuMem.Write(data.address, data.value)
			}

			// Read back and verify mirroring behavior
			for _, data := range nametableData {
				result := ppuMem.Read(data.address)
				
				switch mm.mode {
				case MirrorHorizontal:
					// $2000-$23FF and $2400-$27FF map to first 1KB
					// $2800-$2BFF and $2C00-$2FFF map to second 1KB
					if data.address < 0x2800 {
						// Should read the same value for $2000 and $2400
						if data.address == 0x2000 || data.address == 0x2400 {
							expected := uint8(0x20) // Last write to this mirror group
							if result != expected {
								t.Errorf("Horizontal mirror: Read(0x%04X) = 0x%02X, want 0x%02X",
									data.address, result, expected)
							}
						}
					} else {
						// Should read the same value for $2800 and $2C00
						if data.address == 0x2800 || data.address == 0x2C00 {
							expected := uint8(0x40) // Last write to this mirror group
							if result != expected {
								t.Errorf("Horizontal mirror: Read(0x%04X) = 0x%02X, want 0x%02X",
									data.address, result, expected)
							}
						}
					}

				case MirrorVertical:
					// $2000-$23FF and $2800-$2BFF map to first 1KB
					// $2400-$27FF and $2C00-$2FFF map to second 1KB
					if data.address == 0x2000 || data.address == 0x2800 {
						expected := uint8(0x30) // Last write to this mirror group
						if result != expected {
							t.Errorf("Vertical mirror: Read(0x%04X) = 0x%02X, want 0x%02X",
								data.address, result, expected)
						}
					} else if data.address == 0x2400 || data.address == 0x2C00 {
						expected := uint8(0x40) // Last write to this mirror group
						if result != expected {
							t.Errorf("Vertical mirror: Read(0x%04X) = 0x%02X, want 0x%02X",
								data.address, result, expected)
						}
					}

				case MirrorSingleScreen0:
					// All nametables map to first 1KB
					expected := uint8(0x40) // Last write
					if result != expected {
						t.Errorf("Single screen 0: Read(0x%04X) = 0x%02X, want 0x%02X",
							data.address, result, expected)
					}

				case MirrorSingleScreen1:
					// All nametables map to second 1KB
					expected := uint8(0x40) // Last write
					if result != expected {
						t.Errorf("Single screen 1: Read(0x%04X) = 0x%02X, want 0x%02X",
							data.address, result, expected)
					}
				}
			}
		})
	}
}

// TestMemoryMappingEdgeCases validates edge cases in memory mapping
func TestMemoryMappingEdgeCases(t *testing.T) {
	// Test with boundary addresses
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0xDE}).    // First byte
		WithData(0x3000, []uint8{0xAD})     // Test byte in ROM

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create edge case test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	// Test exact boundaries
	testCases := []struct {
		name     string
		address  uint16
		expected uint8
	}{
		{"ROM Start", 0x8000, 0xDE},
		{"ROM Test", 0xB000, 0xAD},
		{"Mirror Start", 0xC000, 0xDE},
		{"Mirror Test", 0xF000, 0xAD},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mem.Read(tc.address)
			if result != tc.expected {
				t.Errorf("Read(0x%04X) = 0x%02X, want 0x%02X",
					tc.address, result, tc.expected)
			}
		})
	}

	// Test addresses just outside ROM space
	outsideAddresses := []uint16{0x7FFF, 0x4020, 0x6000}
	for _, addr := range outsideAddresses {
		t.Run("Outside ROM", func(t *testing.T) {
			result := mem.Read(addr)
			if result != 0 {
				t.Errorf("Read(0x%04X) = 0x%02X, want 0x00 (outside ROM)",
					addr, result)
			}
		})
	}
}

// TestSRAMMapping validates SRAM mapping in $6000-$7FFF range
func TestSRAMMapping(t *testing.T) {
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithResetVector(0x8000).
		WithBattery() // Enable SRAM

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create SRAM test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	// Test SRAM read/write in $6000-$7FFF range
	sramTests := []struct {
		address uint16
		value   uint8
	}{
		{0x6000, 0xAA}, // SRAM start
		{0x6001, 0xBB}, // SRAM start + 1
		{0x7000, 0xCC}, // SRAM middle
		{0x7FFE, 0xDD}, // SRAM end - 1
		{0x7FFF, 0xEE}, // SRAM end
	}

	for _, test := range sramTests {
		t.Run("SRAM Access", func(t *testing.T) {
			// Write to SRAM
			mem.Write(test.address, test.value)

			// Read back
			result := mem.Read(test.address)
			if result != test.value {
				t.Errorf("SRAM at 0x%04X: wrote 0x%02X, read 0x%02X",
					test.address, test.value, result)
			}
		})
	}

	// Verify SRAM is isolated from ROM
	t.Run("SRAM ROM Isolation", func(t *testing.T) {
		// Write to SRAM
		mem.Write(0x6000, 0x55)
		
		// Read from ROM (should be different)
		romValue := mem.Read(0x8000)
		sramValue := mem.Read(0x6000)
		
		if romValue == sramValue && sramValue == 0x55 {
			t.Error("SRAM and ROM should be isolated")
		}
	})
}

// TestComplexMappingScenario validates complex mapping scenarios
func TestComplexMappingScenario(t *testing.T) {
	// Create a comprehensive test scenario
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1). // 16KB for mirroring
		WithCHRSize(1). // 8KB CHR
		WithMirroring(cartridge.MirrorVertical).
		WithBattery().
		WithResetVector(0x8000).
		WithNMIVector(0x8100).
		WithIRQVector(0x8200).
		WithData(0x0000, []uint8{0x01, 0x02, 0x03, 0x04}). // ROM start
		WithData(0x0100, []uint8{0x11, 0x12, 0x13, 0x14}). // NMI handler
		WithData(0x0200, []uint8{0x21, 0x22, 0x23, 0x24})  // IRQ handler

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create complex test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)
	ppuMem := NewPPUMemory(cart, MirrorVertical)

	// Test all memory regions
	t.Run("RAM Access", func(t *testing.T) {
		mem.Write(0x0000, 0xAA)
		if mem.Read(0x0000) != 0xAA {
			t.Error("RAM access failed")
		}
	})

	t.Run("SRAM Access", func(t *testing.T) {
		mem.Write(0x6000, 0xBB)
		if mem.Read(0x6000) != 0xBB {
			t.Error("SRAM access failed")
		}
	})

	t.Run("ROM Access", func(t *testing.T) {
		if mem.Read(0x8000) != 0x01 {
			t.Error("ROM access failed")
		}
	})

	t.Run("ROM Mirroring", func(t *testing.T) {
		if mem.Read(0x8000) != mem.Read(0xC000) {
			t.Error("ROM mirroring failed")
		}
	})

	t.Run("CHR Access", func(t *testing.T) {
		ppuMem.Write(0x0000, 0xCC)
		if ppuMem.Read(0x0000) != 0xCC {
			t.Error("CHR access failed")
		}
	})

	t.Run("Vector Access", func(t *testing.T) {
		// Test reset vector
		resetLow := mem.Read(0xFFFC)
		resetHigh := mem.Read(0xFFFD)
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		if resetVector != 0x8000 {
			t.Errorf("Reset vector = 0x%04X, want 0x8000", resetVector)
		}

		// Test NMI vector
		nmiLow := mem.Read(0xFFFA)
		nmiHigh := mem.Read(0xFFFB)
		nmiVector := uint16(nmiLow) | (uint16(nmiHigh) << 8)
		if nmiVector != 0x8100 {
			t.Errorf("NMI vector = 0x%04X, want 0x8100", nmiVector)
		}
	})
}