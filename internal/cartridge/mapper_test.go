package cartridge

import (
	"testing"
)

// Helper function to create a test cartridge with specific configuration
func createTestCartridge(prgSize, chrSize int, hasCHRRAM bool) *Cartridge {
	cart := &Cartridge{
		prgROM:     make([]uint8, prgSize),
		mapperID:   0,
		mirror:     MirrorHorizontal,
		hasBattery: false,
		sram:       [0x2000]uint8{},
	}

	// Fill PRG ROM with test pattern
	for i := range cart.prgROM {
		cart.prgROM[i] = uint8(i % 256)
	}

	if hasCHRRAM {
		// CHR RAM - initially empty
		cart.chrROM = make([]uint8, chrSize)
		cart.hasCHRRAM = true
	} else {
		// CHR ROM with test pattern
		cart.chrROM = make([]uint8, chrSize)
		cart.hasCHRRAM = false
		for i := range cart.chrROM {
			cart.chrROM[i] = uint8((i + 128) % 256)
		}
	}

	return cart
}

func TestNewMapper000_16KBConfiguration_ShouldConfigureCorrectly(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false) // 16KB PRG, 8KB CHR ROM
	mapper := NewMapper000(cart)

	if mapper == nil {
		t.Fatal("Expected mapper, got nil")
	}
	if mapper.cart != cart {
		t.Error("Mapper cart reference not set correctly")
	}
	if mapper.prgBanks != 1 {
		t.Errorf("Expected 1 PRG bank for 16KB ROM, got %d", mapper.prgBanks)
	}
}

func TestNewMapper000_32KBConfiguration_ShouldConfigureCorrectly(t *testing.T) {
	cart := createTestCartridge(0x8000, 0x2000, false) // 32KB PRG, 8KB CHR ROM
	mapper := NewMapper000(cart)

	if mapper == nil {
		t.Fatal("Expected mapper, got nil")
	}
	if mapper.prgBanks != 2 {
		t.Errorf("Expected 2 PRG banks for 32KB ROM, got %d", mapper.prgBanks)
	}
}

func TestMapper000_ReadPRG_16KBROMShouldMirror(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false) // 16KB PRG ROM
	mapper := NewMapper000(cart)

	// Test mirroring - both addresses should return same value
	value1 := mapper.ReadPRG(0x8000) // First location
	value2 := mapper.ReadPRG(0xC000) // Mirrored location (0x8000 + 0x4000)

	if value1 != value2 {
		t.Errorf("16KB ROM mirroring failed: 0x8000=%d, 0xC000=%d", value1, value2)
	}

	// Verify the actual values match expected pattern
	expectedValue := uint8(0) // First byte of pattern
	if value1 != expectedValue {
		t.Errorf("Expected value %d at 0x8000, got %d", expectedValue, value1)
	}
}

func TestMapper000_ReadPRG_32KBROMShouldNotMirror(t *testing.T) {
	cart := createTestCartridge(0x8000, 0x2000, false) // 32KB PRG ROM
	mapper := NewMapper000(cart)

	// Different addresses should return different values (no mirroring)
	value1 := mapper.ReadPRG(0x8000) // Start of ROM
	value2 := mapper.ReadPRG(0xC001) // Middle of ROM (0x4001 offset)

	expectedValue1 := uint8(0)            // Pattern at offset 0
	expectedValue2 := uint8(0x4001 % 256) // Pattern at offset 0x4001 = 1

	if value1 != expectedValue1 {
		t.Errorf("Expected value %d at 0x8000, got %d", expectedValue1, value1)
	}
	if value2 != expectedValue2 {
		t.Errorf("Expected value %d at 0xC000, got %d", expectedValue2, value2)
	}
	if value1 == value2 {
		t.Error("32KB ROM should not mirror - values should be different")
	}
}

func TestMapper000_ReadPRG_SRAMAccess_ShouldWorkCorrectly(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	// SRAM is initially zero
	initialValue := mapper.ReadPRG(0x6000)
	if initialValue != 0 {
		t.Errorf("Expected SRAM initial value 0, got %d", initialValue)
	}

	// Write to SRAM
	mapper.WritePRG(0x6000, 0x42)

	// Read back from SRAM
	readValue := mapper.ReadPRG(0x6000)
	if readValue != 0x42 {
		t.Errorf("Expected SRAM value 0x42, got 0x%02X", readValue)
	}
}

func TestMapper000_ReadPRG_SRAMAddressMasking_ShouldWrapCorrectly(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	// Test address masking within SRAM range
	mapper.WritePRG(0x6000, 0x11)
	mapper.WritePRG(0x7FFF, 0x22)

	// These addresses should wrap within 8KB SRAM
	value1 := mapper.ReadPRG(0x6000)
	value2 := mapper.ReadPRG(0x7FFF)

	if value1 != 0x11 {
		t.Errorf("Expected 0x11 at 0x6000, got 0x%02X", value1)
	}
	if value2 != 0x22 {
		t.Errorf("Expected 0x22 at 0x7FFF, got 0x%02X", value2)
	}

	// Test address wrapping - 0x8000 offset should wrap to 0x0000 in SRAM
	mapper.WritePRG(0x6000, 0x33)
	wrappedValue := mapper.ReadPRG(0x8000) // This should map to SRAM, not ROM

	// Note: This tests the current implementation which may have a bug
	// The actual hardware behavior would not map 0x8000 to SRAM
	if wrappedValue == 0x33 {
		t.Error("0x8000 should not map to SRAM - possible addressing bug")
	}
}

func TestMapper000_ReadPRG_InvalidAddressRange_ShouldReturnZero(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	// Test addresses below SRAM range
	value := mapper.ReadPRG(0x5FFF)
	if value != 0 {
		t.Errorf("Expected 0 for invalid address 0x5FFF, got %d", value)
	}

	// Test very low addresses
	value = mapper.ReadPRG(0x0000)
	if value != 0 {
		t.Errorf("Expected 0 for invalid address 0x0000, got %d", value)
	}
}

func TestMapper000_WritePRG_ROMAreaShouldBeIgnored(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	// Get initial ROM value
	initialValue := mapper.ReadPRG(0x8000)

	// Attempt to write to ROM area
	mapper.WritePRG(0x8000, 0xFF)

	// Value should remain unchanged
	afterWriteValue := mapper.ReadPRG(0x8000)
	if afterWriteValue != initialValue {
		t.Error("ROM write should be ignored, but value changed")
	}

	// Test other ROM addresses
	mapper.WritePRG(0xFFFF, 0xAA)
	// No assertion needed - just verify it doesn't crash
}

func TestMapper000_WritePRG_SRAMShouldPersist(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	// Write pattern to multiple SRAM locations
	testData := []struct {
		address uint16
		value   uint8
	}{
		{0x6000, 0x11},
		{0x6100, 0x22},
		{0x7000, 0x33},
		{0x7FFF, 0x44},
	}

	// Write all values
	for _, data := range testData {
		mapper.WritePRG(data.address, data.value)
	}

	// Verify all values persist
	for _, data := range testData {
		value := mapper.ReadPRG(data.address)
		if value != data.value {
			t.Errorf("SRAM at 0x%04X: expected 0x%02X, got 0x%02X",
				data.address, data.value, value)
		}
	}
}

func TestMapper000_ReadCHR_ROMAccess_ShouldReturnCorrectData(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false) // CHR ROM
	mapper := NewMapper000(cart)

	// Test reading CHR ROM
	value := mapper.ReadCHR(0x0000)
	expectedValue := uint8(128) // Pattern starts at 128

	if value != expectedValue {
		t.Errorf("Expected CHR ROM value %d, got %d", expectedValue, value)
	}

	// Test different address
	value = mapper.ReadCHR(0x1000)
	expectedValue = uint8((0x1000 + 128) % 256)

	if value != expectedValue {
		t.Errorf("Expected CHR ROM value %d at 0x1000, got %d", expectedValue, value)
	}
}

func TestMapper000_ReadCHR_InvalidAddress_ShouldReturnZero(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	// Address outside CHR range
	value := mapper.ReadCHR(0x2000)
	if value != 0 {
		t.Errorf("Expected 0 for invalid CHR address 0x2000, got %d", value)
	}

	value = mapper.ReadCHR(0xFFFF)
	if value != 0 {
		t.Errorf("Expected 0 for invalid CHR address 0xFFFF, got %d", value)
	}
}

func TestMapper000_WriteCHR_ROMShouldBeIgnored(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false) // CHR ROM
	mapper := NewMapper000(cart)

	// Get initial value
	initialValue := mapper.ReadCHR(0x0000)

	// Attempt to write to CHR ROM
	mapper.WriteCHR(0x0000, 0xFF)

	// Value should remain unchanged
	afterWriteValue := mapper.ReadCHR(0x0000)
	if afterWriteValue != initialValue {
		t.Error("CHR ROM write should be ignored, but value changed")
	}
}

func TestMapper000_WriteCHR_RAMShouldPersist(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, true) // CHR RAM
	mapper := NewMapper000(cart)

	// Initially should be zero
	initialValue := mapper.ReadCHR(0x0000)
	if initialValue != 0 {
		t.Errorf("Expected CHR RAM initial value 0, got %d", initialValue)
	}

	// Write to CHR RAM
	mapper.WriteCHR(0x0000, 0x55)
	value := mapper.ReadCHR(0x0000)

	if value != 0x55 {
		t.Errorf("Expected CHR RAM value 0x55, got 0x%02X", value)
	}

	// Test multiple locations
	testData := []struct {
		address uint16
		value   uint8
	}{
		{0x0000, 0xAA},
		{0x0800, 0xBB},
		{0x1000, 0xCC},
		{0x1FFF, 0xDD},
	}

	for _, data := range testData {
		mapper.WriteCHR(data.address, data.value)
		value := mapper.ReadCHR(data.address)
		if value != data.value {
			t.Errorf("CHR RAM at 0x%04X: expected 0x%02X, got 0x%02X",
				data.address, data.value, value)
		}
	}
}

func TestMapper000_WriteCHR_InvalidAddress_ShouldBeIgnored(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, true) // CHR RAM
	mapper := NewMapper000(cart)

	// Write to invalid address - should not crash
	mapper.WriteCHR(0x2000, 0xFF)
	mapper.WriteCHR(0xFFFF, 0xAA)

	// No assertions needed - just verify no crash
}

func TestMapper000_CHRRAMDetection_ShouldWorkCorrectly(t *testing.T) {
	// Test CHR ROM configuration (size > 0 in original ROM)
	cartROM := createTestCartridge(0x4000, 0x2000, false)
	mapperROM := NewMapper000(cartROM)

	// Write should be ignored for CHR ROM
	mapperROM.WriteCHR(0x0000, 0xFF)
	value := mapperROM.ReadCHR(0x0000)
	expectedROMValue := uint8(128) // Original pattern

	if value != expectedROMValue {
		t.Error("CHR ROM should not be writable")
	}

	// Test CHR RAM configuration
	cartRAM := createTestCartridge(0x4000, 0x2000, true)
	mapperRAM := NewMapper000(cartRAM)

	// Write should work for CHR RAM
	mapperRAM.WriteCHR(0x0000, 0xFF)
	value = mapperRAM.ReadCHR(0x0000)

	if value != 0xFF {
		t.Error("CHR RAM should be writable")
	}
}

func TestMapper000_AddressMirroring_16KBPattern(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false) // 16KB PRG ROM
	mapper := NewMapper000(cart)

	// Test specific mirroring pattern for 16KB ROM
	testAddresses := []uint16{
		0x8000, 0xC000, // Should be identical (mirrored)
		0x8100, 0xC100, // Should be identical (mirrored)
		0x9000, 0xD000, // Should be identical (mirrored)
		0xBFFF, 0xFFFF, // Should be identical (mirrored)
	}

	for i := 0; i < len(testAddresses); i += 2 {
		addr1 := testAddresses[i]
		addr2 := testAddresses[i+1]

		value1 := mapper.ReadPRG(addr1)
		value2 := mapper.ReadPRG(addr2)

		if value1 != value2 {
			t.Errorf("16KB mirroring failed: 0x%04X=%d, 0x%04X=%d",
				addr1, value1, addr2, value2)
		}
	}
}

func TestMapper000_AddressDecoding_32KBPattern(t *testing.T) {
	cart := createTestCartridge(0x8000, 0x2000, false) // 32KB PRG ROM
	mapper := NewMapper000(cart)

	// Test that different regions map to different ROM areas
	testCases := []struct {
		address uint16
		offset  int
	}{
		{0x8000, 0x0000},
		{0x9000, 0x1000},
		{0xA000, 0x2000},
		{0xB000, 0x3000},
		{0xC000, 0x4000},
		{0xD000, 0x5000},
		{0xE000, 0x6000},
		{0xF000, 0x7000},
	}

	for _, tc := range testCases {
		value := mapper.ReadPRG(tc.address)
		expectedValue := uint8(tc.offset % 256) // Pattern based on offset

		if value != expectedValue {
			t.Errorf("32KB address 0x%04X: expected %d (offset 0x%04X), got %d",
				tc.address, expectedValue, tc.offset, value)
		}
	}
}

// Benchmark tests for mapper performance
func BenchmarkMapper000_ReadPRG_16KB(b *testing.B) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapper.ReadPRG(0x8000)
	}
}

func BenchmarkMapper000_ReadPRG_32KB(b *testing.B) {
	cart := createTestCartridge(0x8000, 0x2000, false)
	mapper := NewMapper000(cart)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapper.ReadPRG(0x8000)
	}
}

func BenchmarkMapper000_WritePRG_SRAM(b *testing.B) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapper.WritePRG(0x6000, uint8(i))
	}
}

func BenchmarkMapper000_ReadCHR(b *testing.B) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapper.ReadCHR(0x0000)
	}
}

func BenchmarkMapper000_WriteCHR_RAM(b *testing.B) {
	cart := createTestCartridge(0x4000, 0x2000, true) // CHR RAM
	mapper := NewMapper000(cart)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mapper.WriteCHR(0x0000, uint8(i))
	}
}

// Edge case tests
func TestMapper000_BoundaryConditions_ShouldHandleCorrectly(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	// Test boundary between SRAM and ROM
	mapper.WritePRG(0x7FFF, 0xAA)      // Last SRAM address
	romValue := mapper.ReadPRG(0x8000) // First ROM address
	sramValue := mapper.ReadPRG(0x7FFF)

	if sramValue != 0xAA {
		t.Errorf("Expected SRAM boundary value 0xAA, got 0x%02X", sramValue)
	}
	if romValue == 0xAA {
		t.Error("ROM should not have same value as SRAM at boundary")
	}

	// Test CHR boundary
	mapper.WriteCHR(0x1FFF, 0xBB)          // Last CHR address (if RAM)
	invalidValue := mapper.ReadCHR(0x2000) // Beyond CHR range

	if invalidValue != 0 {
		t.Errorf("Expected 0 beyond CHR range, got %d", invalidValue)
	}
}

func TestMapper000_ZeroSizedROM_ShouldHandleGracefully(t *testing.T) {
	// This is an edge case that shouldn't normally occur
	cart := &Cartridge{
		prgROM:   []uint8{}, // Zero-sized PRG ROM
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	if mapper.prgBanks != 0 {
		t.Errorf("Expected 0 PRG banks for empty ROM, got %d", mapper.prgBanks)
	}

	// Reading from ROM area should not crash
	value := mapper.ReadPRG(0x8000)
	// No specific assertion - just verify it doesn't panic
	_ = value
}

func TestMapper000_ExtremeAddresses_ShouldNotPanic(t *testing.T) {
	cart := createTestCartridge(0x4000, 0x2000, false)
	mapper := NewMapper000(cart)

	// Test extreme addresses - should not panic
	extremeAddresses := []uint16{
		0x0000, 0x0001, 0x00FF,
		0x5000, 0x5FFF,
		0x8000, 0xFFFF,
	}

	for _, addr := range extremeAddresses {
		// These should not panic
		mapper.ReadPRG(addr)
		mapper.WritePRG(addr, 0x42)
		mapper.ReadCHR(addr)
		mapper.WriteCHR(addr, 0x42)
	}
}
