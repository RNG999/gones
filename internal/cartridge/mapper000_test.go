package cartridge

import (
	"testing"
)

// Test Mapper 0 (NROM) specific behavior and configurations
// This file focuses on mapper-specific functionality, memory layouts, and hardware behavior

// TestMapper000_Configuration_16KB_PRG_ROM tests 16KB PRG ROM configuration
func TestMapper000_Configuration_16KB_PRG_ROM(t *testing.T) {
	// Create cartridge with 16KB PRG ROM, 8KB CHR ROM
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000), // 16KB
		chrROM:    make([]uint8, 0x2000), // 8KB
		mapperID:  0,
		mirror:    MirrorHorizontal,
		hasCHRRAM: false,
	}

	// Fill with test pattern
	for i := range cart.prgROM {
		cart.prgROM[i] = uint8(i & 0xFF)
	}

	mapper := NewMapper000(cart)

	// Verify configuration
	if mapper.prgBanks != 1 {
		t.Errorf("Expected 1 PRG bank for 16KB ROM, got %d", mapper.prgBanks)
	}

	// Test mirroring behavior - 16KB ROM should mirror to fill 32KB space
	// Address 0x8000 should equal 0xC000 (mirrored)
	value1 := mapper.ReadPRG(0x8000)
	value2 := mapper.ReadPRG(0xC000)
	if value1 != value2 {
		t.Errorf("16KB ROM mirroring failed: 0x8000=0x%02X, 0xC000=0x%02X", value1, value2)
	}

	// Test specific mirroring pattern
	value3 := mapper.ReadPRG(0x8123)
	value4 := mapper.ReadPRG(0xC123)
	if value3 != value4 {
		t.Errorf("16KB ROM mirroring failed at offset: 0x8123=0x%02X, 0xC123=0x%02X", value3, value4)
	}

	// Verify actual values match expected pattern
	expectedValue := uint8(0x123 & 0xFF)
	if value3 != expectedValue {
		t.Errorf("Expected pattern value 0x%02X at offset 0x123, got 0x%02X", expectedValue, value3)
	}
}

// TestMapper000_Configuration_32KB_PRG_ROM tests 32KB PRG ROM configuration
func TestMapper000_Configuration_32KB_PRG_ROM(t *testing.T) {
	// Create cartridge with 32KB PRG ROM, 8KB CHR ROM
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x8000), // 32KB
		chrROM:    make([]uint8, 0x2000), // 8KB
		mapperID:  0,
		mirror:    MirrorVertical,
		hasCHRRAM: false,
	}

	// Fill with test pattern using high byte to differentiate 16KB banks
	for i := range cart.prgROM {
		cart.prgROM[i] = uint8((i >> 8) & 0xFF)
	}

	mapper := NewMapper000(cart)

	// Verify configuration
	if mapper.prgBanks != 2 {
		t.Errorf("Expected 2 PRG banks for 32KB ROM, got %d", mapper.prgBanks)
	}

	// Test no mirroring - different addresses should return different values
	value1 := mapper.ReadPRG(0x8000) // Start of ROM
	value2 := mapper.ReadPRG(0xC000) // Middle of ROM

	expectedValue1 := uint8(0x00)  // cart.prgROM[0] = (0 >> 8) & 0xFF = 0x00
	expectedValue2 := uint8(0x40)  // cart.prgROM[0x4000] = (0x4000 >> 8) & 0xFF = 0x40

	if value1 != expectedValue1 {
		t.Errorf("Expected 0x%02X at 0x8000, got 0x%02X", expectedValue1, value1)
	}
	if value2 != expectedValue2 {
		t.Errorf("Expected 0x%02X at 0xC000, got 0x%02X", expectedValue2, value2)
	}

	// Values should be different (no mirroring)
	if value1 == value2 {
		t.Errorf("32KB ROM should not mirror - values should be different: 0x8000=0x%02X, 0xC000=0x%02X", value1, value2)
	}
}

// TestMapper000_Configuration_CHR_ROM tests CHR ROM configuration
func TestMapper000_Configuration_CHR_ROM(t *testing.T) {
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000), // 8KB CHR ROM
		mapperID:  0,
		hasCHRRAM: false,
	}

	// Fill CHR ROM with pattern
	for i := range cart.chrROM {
		cart.chrROM[i] = uint8((i + 0x40) & 0xFF)
	}

	mapper := NewMapper000(cart)

	// Test reading CHR ROM
	value := mapper.ReadCHR(0x0000)
	expectedValue := uint8(0x40)
	if value != expectedValue {
		t.Errorf("Expected CHR ROM value 0x%02X, got 0x%02X", expectedValue, value)
	}

	// Test CHR ROM is read-only
	originalValue := mapper.ReadCHR(0x0100)
	mapper.WriteCHR(0x0100, 0xFF)
	afterWriteValue := mapper.ReadCHR(0x0100)

	if afterWriteValue != originalValue {
		t.Error("CHR ROM should be read-only - write was not ignored")
	}
}

// TestMapper000_Configuration_CHR_RAM tests CHR RAM configuration
func TestMapper000_Configuration_CHR_RAM(t *testing.T) {
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000), // 8KB CHR RAM
		mapperID:  0,
		hasCHRRAM: true,
	}

	mapper := NewMapper000(cart)

	// CHR RAM should initially be zero
	value := mapper.ReadCHR(0x0000)
	if value != 0 {
		t.Errorf("Expected CHR RAM initial value 0, got 0x%02X", value)
	}

	// Test CHR RAM is writable
	mapper.WriteCHR(0x0100, 0xAB)
	value = mapper.ReadCHR(0x0100)
	if value != 0xAB {
		t.Errorf("Expected CHR RAM value 0xAB after write, got 0x%02X", value)
	}

	// Test full range of CHR RAM
	testPattern := []struct {
		address uint16
		value   uint8
	}{
		{0x0000, 0x11},
		{0x0800, 0x22},
		{0x1000, 0x33},
		{0x1800, 0x44},
		{0x1FFF, 0x55},
	}

	for _, test := range testPattern {
		mapper.WriteCHR(test.address, test.value)
		readValue := mapper.ReadCHR(test.address)
		if readValue != test.value {
			t.Errorf("CHR RAM at 0x%04X: expected 0x%02X, got 0x%02X",
				test.address, test.value, readValue)
		}
	}
}

// TestMapper000_SRAM_BatteryBacked tests battery-backed SRAM functionality
func TestMapper000_SRAM_BatteryBacked(t *testing.T) {
	cart := &Cartridge{
		prgROM:     make([]uint8, 0x4000),
		chrROM:     make([]uint8, 0x2000),
		mapperID:   0,
		hasBattery: true,            // Battery-backed SRAM
		sram:       [0x2000]uint8{}, // 8KB SRAM
	}

	mapper := NewMapper000(cart)

	// Test SRAM address range 0x6000-0x7FFF
	testData := []struct {
		address uint16
		value   uint8
	}{
		{0x6000, 0xDE},
		{0x6001, 0xAD},
		{0x6100, 0xBE},
		{0x7000, 0xEF},
		{0x7FFE, 0xCA},
		{0x7FFF, 0xFE},
	}

	// Write test pattern
	for _, test := range testData {
		mapper.WritePRG(test.address, test.value)
	}

	// Verify pattern persists
	for _, test := range testData {
		value := mapper.ReadPRG(test.address)
		if value != test.value {
			t.Errorf("SRAM at 0x%04X: expected 0x%02X, got 0x%02X",
				test.address, test.value, value)
		}
	}

	// Test address masking - SRAM is 8KB, so addresses should wrap
	mapper.WritePRG(0x6000, 0x11)
	mapper.WritePRG(0x8000, 0x22) // This should NOT affect SRAM

	sramValue := mapper.ReadPRG(0x6000)
	if sramValue != 0x11 {
		t.Errorf("SRAM value changed when ROM area written: expected 0x11, got 0x%02X", sramValue)
	}
}

// TestMapper000_AddressDecoding_PRG tests PRG address decoding
func TestMapper000_AddressDecoding_PRG(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x8000), // 32KB
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	// Create identifiable pattern
	for i := range cart.prgROM {
		cart.prgROM[i] = uint8((i >> 8) & 0xFF) // High byte of address
	}

	mapper := NewMapper000(cart)

	// Test address decoding for 32KB ROM
	testCases := []struct {
		address  uint16
		expected uint8
	}{
		{0x8000, 0x00}, // ROM offset 0x0000
		{0x8100, 0x01}, // ROM offset 0x0100
		{0x9000, 0x10}, // ROM offset 0x1000
		{0xA000, 0x20}, // ROM offset 0x2000
		{0xC000, 0x40}, // ROM offset 0x4000
		{0xE000, 0x60}, // ROM offset 0x6000
		{0xFFFF, 0x7F}, // ROM offset 0x7FFF
	}

	for _, test := range testCases {
		value := mapper.ReadPRG(test.address)
		if value != test.expected {
			t.Errorf("Address 0x%04X: expected 0x%02X, got 0x%02X",
				test.address, test.expected, value)
		}
	}
}

// TestMapper000_AddressDecoding_CHR tests CHR address decoding
func TestMapper000_AddressDecoding_CHR(t *testing.T) {
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000), // 8KB
		mapperID:  0,
		hasCHRRAM: false,
	}

	// Create identifiable pattern
	for i := range cart.chrROM {
		cart.chrROM[i] = uint8((i >> 4) & 0xFF) // Address bits 4-11
	}

	mapper := NewMapper000(cart)

	// Test CHR address decoding
	testCases := []struct {
		address  uint16
		expected uint8
	}{
		{0x0000, 0x00},
		{0x0010, 0x01},
		{0x0100, 0x10},
		{0x0800, 0x80},
		{0x1000, 0x00}, // Wraps at 0x100 due to pattern
		{0x1800, 0x80},
		{0x1FF0, 0xFF},
	}

	for _, test := range testCases {
		value := mapper.ReadCHR(test.address)
		if value != test.expected {
			t.Errorf("CHR address 0x%04X: expected 0x%02X, got 0x%02X",
				test.address, test.expected, value)
		}
	}
}

// TestMapper000_MemoryBoundaries tests memory boundary handling
func TestMapper000_MemoryBoundaries(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
		sram:     [0x2000]uint8{},
	}

	mapper := NewMapper000(cart)

	// Test boundaries between memory regions
	testCases := []struct {
		address      uint16
		shouldReturn uint8
		description  string
	}{
		{0x5FFF, 0, "Below SRAM range"},
		{0x6000, 0, "Start of SRAM range"},
		{0x7FFF, 0, "End of SRAM range"},
		{0x8000, 0, "Start of ROM range"},
		{0xFFFF, 0, "End of ROM range"},
	}

	for _, test := range testCases {
		value := mapper.ReadPRG(test.address)
		// We don't test specific values, just that reads don't crash
		_ = value

		// Test writes don't crash
		mapper.WritePRG(test.address, 0x42)
	}

	// Test CHR boundaries
	chrTestCases := []uint16{0x0000, 0x1FFF, 0x2000, 0xFFFF}
	for _, address := range chrTestCases {
		value := mapper.ReadCHR(address)
		_ = value
		mapper.WriteCHR(address, 0x42)
	}
}

// TestMapper000_ResetBehavior tests power-on and reset behavior
func TestMapper000_ResetBehavior(t *testing.T) {
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000),
		mapperID:  0,
		sram:      [0x2000]uint8{},
		hasCHRRAM: true,
	}

	// Fill SRAM with non-zero pattern
	for i := range cart.sram {
		cart.sram[i] = uint8(i & 0xFF)
	}

	mapper := NewMapper000(cart)

	// SRAM should retain values after mapper creation
	for i := 0; i < 0x100; i++ {
		address := uint16(0x6000 + i)
		value := mapper.ReadPRG(address)
		expected := uint8(i & 0xFF)
		if value != expected {
			t.Errorf("SRAM not preserved at 0x%04X: expected 0x%02X, got 0x%02X",
				address, expected, value)
		}
	}

	// CHR RAM should be zero-initialized
	for i := 0; i < 0x100; i++ {
		value := mapper.ReadCHR(uint16(i))
		if value != 0 {
			t.Errorf("CHR RAM not zero-initialized at 0x%04X: got 0x%02X", i, value)
		}
	}
}

// TestMapper000_HardwareBehavior_BusConflicts tests bus conflict behavior
// NROM has no bus conflicts since it doesn't support bank switching
func TestMapper000_HardwareBehavior_BusConflicts(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	// Fill ROM with pattern
	for i := range cart.prgROM {
		cart.prgROM[i] = uint8(0xAA)
	}

	mapper := NewMapper000(cart)

	// Writes to ROM area should be ignored (no bus conflicts)
	originalValue := mapper.ReadPRG(0x8000)

	// Attempt writes with different values
	mapper.WritePRG(0x8000, 0x55)
	mapper.WritePRG(0x8000, 0xFF)
	mapper.WritePRG(0x8000, 0x00)

	// Value should remain unchanged
	afterWriteValue := mapper.ReadPRG(0x8000)
	if afterWriteValue != originalValue {
		t.Errorf("ROM write caused bus conflict: original=0x%02X, after=0x%02X",
			originalValue, afterWriteValue)
	}
}

// TestMapper000_PerformanceCharacteristics tests performance behavior
func TestMapper000_PerformanceCharacteristics(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x8000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Test that address calculations are consistent
	const iterations = 1000

	// Test PRG reads are deterministic
	for i := 0; i < iterations; i++ {
		value1 := mapper.ReadPRG(0x8000)
		value2 := mapper.ReadPRG(0x8000)
		if value1 != value2 {
			t.Error("PRG reads are not deterministic")
			break
		}
	}

	// Test CHR reads are deterministic
	for i := 0; i < iterations; i++ {
		value1 := mapper.ReadCHR(0x0000)
		value2 := mapper.ReadCHR(0x0000)
		if value1 != value2 {
			t.Error("CHR reads are not deterministic")
			break
		}
	}

	// Test SRAM writes are persistent
	for i := 0; i < 100; i++ {
		address := uint16(0x6000 + i)
		testValue := uint8(i)

		mapper.WritePRG(address, testValue)
		readValue := mapper.ReadPRG(address)

		if readValue != testValue {
			t.Errorf("SRAM write/read inconsistent at 0x%04X: wrote 0x%02X, read 0x%02X",
				address, testValue, readValue)
		}
	}
}

// TestMapper000_EdgeCase_ZeroSizeROM tests handling of zero-size ROM
func TestMapper000_EdgeCase_ZeroSizeROM(t *testing.T) {
	cart := &Cartridge{
		prgROM:   []uint8{}, // Zero-size ROM
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Should handle gracefully
	if mapper.prgBanks != 0 {
		t.Errorf("Expected 0 PRG banks for zero-size ROM, got %d", mapper.prgBanks)
	}

	// Reads should not crash
	value := mapper.ReadPRG(0x8000)
	if value != 0 {
		t.Errorf("Expected 0 for zero-size ROM read, got 0x%02X", value)
	}
}

// TestMapper000_EdgeCase_InvalidAddresses tests invalid address handling
func TestMapper000_EdgeCase_InvalidAddresses(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Test invalid PRG addresses
	invalidPRGAddresses := []uint16{
		0x0000, 0x1000, 0x2000, 0x3000,
		0x4000, 0x5000, 0x5FFF,
	}

	for _, address := range invalidPRGAddresses {
		value := mapper.ReadPRG(address)
		if value != 0 {
			t.Errorf("Expected 0 for invalid PRG address 0x%04X, got 0x%02X", address, value)
		}
	}

	// Test invalid CHR addresses
	invalidCHRAddresses := []uint16{
		0x2000, 0x3000, 0x4000, 0x8000, 0xFFFF,
	}

	for _, address := range invalidCHRAddresses {
		value := mapper.ReadCHR(address)
		if value != 0 {
			t.Errorf("Expected 0 for invalid CHR address 0x%04X, got 0x%02X", address, value)
		}
	}
}
