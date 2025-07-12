package cartridge

import (
	"testing"
)

// TestMapper000_Verification tests the critical fixes for Mapper 0
func TestMapper000_Verification(t *testing.T) {
	t.Run("16KB PRG ROM mirroring verification", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x4000), // 16KB
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
		}

		// Fill with identifiable pattern
		for i := range cart.prgROM {
			cart.prgROM[i] = uint8(i & 0xFF)
		}

		mapper := NewMapper000(cart)

		// Critical test: 16KB ROM should mirror
		// Address 0x8000 maps to ROM offset 0x0000
		// Address 0xC000 should also map to ROM offset 0x0000 (mirrored)
		val1 := mapper.ReadPRG(0x8000)
		val2 := mapper.ReadPRG(0xC000)

		if val1 != 0x00 {
			t.Errorf("Expected 0x00 at 0x8000, got 0x%02X", val1)
		}
		if val2 != 0x00 {
			t.Errorf("Expected 0x00 at 0xC000 (mirrored), got 0x%02X", val2)
		}
		if val1 != val2 {
			t.Error("16KB mirroring failed: 0x8000 and 0xC000 should read same value")
		}

		// Test offset within the bank
		val3 := mapper.ReadPRG(0x8123)
		val4 := mapper.ReadPRG(0xC123)
		expectedVal := uint8(0x23) // Pattern at offset 0x123

		if val3 != expectedVal {
			t.Errorf("Expected 0x%02X at 0x8123, got 0x%02X", expectedVal, val3)
		}
		if val4 != expectedVal {
			t.Errorf("Expected 0x%02X at 0xC123 (mirrored), got 0x%02X", expectedVal, val4)
		}
	})

	t.Run("32KB PRG ROM direct mapping verification", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x8000), // 32KB
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
		}

		// Fill with pattern that helps identify offset
		for i := range cart.prgROM {
			cart.prgROM[i] = uint8((i >> 8) & 0xFF) // High byte of offset
		}

		mapper := NewMapper000(cart)

		// Test direct mapping (no mirroring)
		testCases := []struct {
			address  uint16
			expected uint8
		}{
			{0x8000, 0x00}, // Offset 0x0000
			{0x8100, 0x01}, // Offset 0x0100
			{0x9000, 0x10}, // Offset 0x1000
			{0xC000, 0x40}, // Offset 0x4000
			{0xE000, 0x60}, // Offset 0x6000
			{0xFFFF, 0x7F}, // Offset 0x7FFF
		}

		for _, tc := range testCases {
			val := mapper.ReadPRG(tc.address)
			if val != tc.expected {
				t.Errorf("32KB mapping at 0x%04X: expected 0x%02X, got 0x%02X",
					tc.address, tc.expected, val)
			}
		}
	})

	t.Run("SRAM addressing verification", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x4000),
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
			sram:     [0x2000]uint8{},
		}

		mapper := NewMapper000(cart)

		// Test SRAM write and read
		testData := []struct {
			address uint16
			value   uint8
		}{
			{0x6000, 0xAA}, // Start of SRAM
			{0x6001, 0xBB},
			{0x7000, 0xCC}, // Middle of SRAM
			{0x7FFE, 0xDD},
			{0x7FFF, 0xEE}, // End of SRAM
		}

		// Write test data
		for _, td := range testData {
			mapper.WritePRG(td.address, td.value)
		}

		// Read and verify
		for _, td := range testData {
			val := mapper.ReadPRG(td.address)
			if val != td.value {
				t.Errorf("SRAM at 0x%04X: expected 0x%02X, got 0x%02X",
					td.address, td.value, val)
			}
		}

		// Verify SRAM boundaries
		mapper.WritePRG(0x5FFF, 0xFF) // Below SRAM
		mapper.WritePRG(0x8000, 0xFF) // Above SRAM (ROM area)

		// These writes should be ignored, original SRAM values preserved
		val := mapper.ReadPRG(0x6000)
		if val != 0xAA {
			t.Error("SRAM value corrupted by out-of-bounds write")
		}
	})

	t.Run("CHR memory verification", func(t *testing.T) {
		// Test CHR ROM (read-only)
		cartROM := &Cartridge{
			prgROM:    make([]uint8, 0x4000),
			chrROM:    make([]uint8, 0x2000),
			mapperID:  0,
			hasCHRRAM: false,
		}

		// Fill CHR ROM with pattern
		for i := range cartROM.chrROM {
			cartROM.chrROM[i] = uint8((i + 0x40) & 0xFF)
		}

		mapperROM := NewMapper000(cartROM)

		// Test CHR ROM read
		val := mapperROM.ReadCHR(0x0000)
		if val != 0x40 {
			t.Errorf("CHR ROM read: expected 0x40, got 0x%02X", val)
		}

		// Test CHR ROM is read-only
		mapperROM.WriteCHR(0x0000, 0xFF)
		val = mapperROM.ReadCHR(0x0000)
		if val != 0x40 {
			t.Error("CHR ROM should be read-only")
		}

		// Test CHR RAM (read-write)
		cartRAM := &Cartridge{
			prgROM:    make([]uint8, 0x4000),
			chrROM:    make([]uint8, 0x2000),
			mapperID:  0,
			hasCHRRAM: true,
		}

		mapperRAM := NewMapper000(cartRAM)

		// CHR RAM should be writable
		mapperRAM.WriteCHR(0x0100, 0x77)
		val = mapperRAM.ReadCHR(0x0100)
		if val != 0x77 {
			t.Errorf("CHR RAM write/read: expected 0x77, got 0x%02X", val)
		}

		// Test CHR boundary
		val = mapperRAM.ReadCHR(0x2000) // Beyond CHR range
		if val != 0 {
			t.Errorf("CHR read beyond range should return 0, got 0x%02X", val)
		}
	})
}
