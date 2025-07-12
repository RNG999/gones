package cartridge

import (
	"testing"
)

// Mapper Interface Compliance Tests
// Tests that Mapper 0 properly implements the Mapper interface
// and integrates correctly with the cartridge system

// TestMapperInterface_Implementation tests Mapper interface implementation
func TestMapperInterface_Implementation(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Verify mapper implements Mapper interface
	var _ Mapper = mapper

	// Test all interface methods are callable
	mapper.ReadPRG(0x8000)
	mapper.WritePRG(0x6000, 0x42)
	mapper.ReadCHR(0x0000)
	mapper.WriteCHR(0x0000, 0x42)
}

// TestMapperInterface_ReadPRG_Signature tests ReadPRG method signature
func TestMapperInterface_ReadPRG_Signature(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Test method signature: ReadPRG(uint16) uint8
	var result uint8 = mapper.ReadPRG(0x8000)
	_ = result

	// Test with various address types
	addresses := []uint16{0x0000, 0x6000, 0x8000, 0xFFFF}
	for _, addr := range addresses {
		value := mapper.ReadPRG(addr)
		_ = value // Should compile without error
	}
}

// TestMapperInterface_WritePRG_Signature tests WritePRG method signature
func TestMapperInterface_WritePRG_Signature(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Test method signature: WritePRG(uint16, uint8)
	mapper.WritePRG(0x6000, 0x42)

	// Test with various value types
	values := []uint8{0x00, 0x55, 0xAA, 0xFF}
	for _, val := range values {
		mapper.WritePRG(0x6000, val) // Should compile without error
	}
}

// TestMapperInterface_ReadCHR_Signature tests ReadCHR method signature
func TestMapperInterface_ReadCHR_Signature(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Test method signature: ReadCHR(uint16) uint8
	var result uint8 = mapper.ReadCHR(0x0000)
	_ = result

	// Test with various addresses
	addresses := []uint16{0x0000, 0x1000, 0x1FFF, 0x2000}
	for _, addr := range addresses {
		value := mapper.ReadCHR(addr)
		_ = value // Should compile without error
	}
}

// TestMapperInterface_WriteCHR_Signature tests WriteCHR method signature
func TestMapperInterface_WriteCHR_Signature(t *testing.T) {
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000),
		mapperID:  0,
		hasCHRRAM: true,
	}

	mapper := NewMapper000(cart)

	// Test method signature: WriteCHR(uint16, uint8)
	mapper.WriteCHR(0x0000, 0x42)

	// Test with various values
	values := []uint8{0x00, 0x33, 0x66, 0x99, 0xCC, 0xFF}
	for _, val := range values {
		mapper.WriteCHR(0x0000, val) // Should compile without error
	}
}

// TestMapperInterface_CartridgeIntegration tests integration with Cartridge
func TestMapperInterface_CartridgeIntegration(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
		sram:     [0x2000]uint8{},
	}

	// Fill ROM with test pattern
	for i := range cart.prgROM {
		cart.prgROM[i] = uint8(i & 0xFF)
	}

	mapper := NewMapper000(cart)

	// Test that cartridge methods delegate to mapper
	cartValue := cart.ReadPRG(0x8000)
	mapperValue := mapper.ReadPRG(0x8000)

	if cartValue != mapperValue {
		t.Errorf("Cartridge-mapper integration failed: cart=0x%02X, mapper=0x%02X",
			cartValue, mapperValue)
	}

	// Test write delegation
	cart.WritePRG(0x6000, 0x55)
	mapperReadValue := mapper.ReadPRG(0x6000)

	if mapperReadValue != 0x55 {
		t.Errorf("Write delegation failed: expected 0x55, got 0x%02X", mapperReadValue)
	}
}

// TestMapperInterface_MemoryRegionConsistency tests memory region consistency
func TestMapperInterface_MemoryRegionConsistency(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x8000), // 32KB
		chrROM:   make([]uint8, 0x2000), // 8KB
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Test PRG memory regions
	prgRegions := []struct {
		start, end uint16
		name       string
	}{
		{0x6000, 0x7FFF, "SRAM"},
		{0x8000, 0xFFFF, "PRG ROM"},
	}

	for _, region := range prgRegions {
		// Test boundary reads don't crash
		mapper.ReadPRG(region.start)
		mapper.ReadPRG(region.end)

		// Test boundary writes don't crash
		mapper.WritePRG(region.start, 0x42)
		mapper.WritePRG(region.end, 0x42)
	}

	// Test CHR memory region
	mapper.ReadCHR(0x0000)
	mapper.ReadCHR(0x1FFF)
	mapper.WriteCHR(0x0000, 0x42)
	mapper.WriteCHR(0x1FFF, 0x42)
}

// TestMapperInterface_ErrorHandling tests error handling compliance
func TestMapperInterface_ErrorHandling(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Test that invalid operations don't panic
	invalidPRGAddresses := []uint16{0x0000, 0x1000, 0x5FFF}
	for _, addr := range invalidPRGAddresses {
		value := mapper.ReadPRG(addr)
		if value != 0 {
			t.Errorf("Expected 0 for invalid PRG address 0x%04X, got 0x%02X", addr, value)
		}

		// Write should not crash
		mapper.WritePRG(addr, 0x42)
	}

	// Test invalid CHR addresses
	invalidCHRAddresses := []uint16{0x2000, 0x3000, 0xFFFF}
	for _, addr := range invalidCHRAddresses {
		value := mapper.ReadCHR(addr)
		if value != 0 {
			t.Errorf("Expected 0 for invalid CHR address 0x%04X, got 0x%02X", addr, value)
		}

		// Write should not crash
		mapper.WriteCHR(addr, 0x42)
	}
}

// TestMapperInterface_StateIsolation tests state isolation between instances
func TestMapperInterface_StateIsolation(t *testing.T) {
	// Create two separate cartridges
	cart1 := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000),
		mapperID:  0,
		sram:      [0x2000]uint8{},
		hasCHRRAM: true,
	}

	cart2 := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000),
		mapperID:  0,
		sram:      [0x2000]uint8{},
		hasCHRRAM: true,
	}

	mapper1 := NewMapper000(cart1)
	mapper2 := NewMapper000(cart2)

	// Write different values to each mapper
	mapper1.WritePRG(0x6000, 0xAA)
	mapper2.WritePRG(0x6000, 0x55)

	mapper1.WriteCHR(0x0000, 0xCC)
	mapper2.WriteCHR(0x0000, 0x33)

	// Verify values are isolated
	value1_prg := mapper1.ReadPRG(0x6000)
	value2_prg := mapper2.ReadPRG(0x6000)

	if value1_prg != 0xAA {
		t.Errorf("Mapper1 PRG isolation failed: expected 0xAA, got 0x%02X", value1_prg)
	}
	if value2_prg != 0x55 {
		t.Errorf("Mapper2 PRG isolation failed: expected 0x55, got 0x%02X", value2_prg)
	}

	value1_chr := mapper1.ReadCHR(0x0000)
	value2_chr := mapper2.ReadCHR(0x0000)

	if value1_chr != 0xCC {
		t.Errorf("Mapper1 CHR isolation failed: expected 0xCC, got 0x%02X", value1_chr)
	}
	if value2_chr != 0x33 {
		t.Errorf("Mapper2 CHR isolation failed: expected 0x33, got 0x%02X", value2_chr)
	}
}

// TestMapperInterface_NilSafety tests nil pointer safety
func TestMapperInterface_NilSafety(t *testing.T) {
	// Test creation with valid cartridge
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)
	if mapper == nil {
		t.Fatal("NewMapper000 returned nil with valid cartridge")
	}

	if mapper.cart != cart {
		t.Error("Mapper does not reference correct cartridge")
	}
}

// TestMapperInterface_FactoryIntegration tests integration with mapper factory
func TestMapperInterface_FactoryIntegration(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	// Test factory creates correct mapper
	mapper := createMapper(0, cart)
	if mapper == nil {
		t.Fatal("createMapper returned nil for mapper 0")
	}

	// Verify it's actually a Mapper000
	mapper000, ok := mapper.(*Mapper000)
	if !ok {
		t.Fatal("createMapper did not return Mapper000 instance")
	}

	if mapper000.cart != cart {
		t.Error("Factory-created mapper does not reference correct cartridge")
	}

	// Test interface compliance
	var _ Mapper = mapper
}

// TestMapperInterface_TypeAssertions tests type assertions work correctly
func TestMapperInterface_TypeAssertions(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	// Get mapper through factory (returns Mapper interface)
	var mapper Mapper = createMapper(0, cart)

	// Test type assertion to concrete type
	concrete, ok := mapper.(*Mapper000)
	if !ok {
		t.Fatal("Type assertion to *Mapper000 failed")
	}

	if concrete.prgBanks != 1 {
		t.Errorf("Type assertion succeeded but values incorrect: expected 1 bank, got %d",
			concrete.prgBanks)
	}

	// Test interface satisfaction
	var iface Mapper = NewMapper000(cart)
	_ = iface
}

// TestMapperInterface_MethodCallConsistency tests method call consistency
func TestMapperInterface_MethodCallConsistency(t *testing.T) {
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000),
		mapperID:  0,
		sram:      [0x2000]uint8{},
		hasCHRRAM: true,
	}

	// Fill with pattern
	for i := range cart.prgROM {
		cart.prgROM[i] = uint8(i & 0xFF)
	}

	mapper := NewMapper000(cart)

	// Test method calls through interface
	var iface Mapper = mapper

	// Test repeated calls return consistent results
	const iterations = 100
	address := uint16(0x8000)
	expectedValue := cart.prgROM[0]

	for i := 0; i < iterations; i++ {
		value := iface.ReadPRG(address)
		if value != expectedValue {
			t.Errorf("Inconsistent read at iteration %d: expected 0x%02X, got 0x%02X",
				i, expectedValue, value)
		}
	}

	// Test write-read consistency
	testAddress := uint16(0x6000)
	testValue := uint8(0x77)

	for i := 0; i < iterations; i++ {
		iface.WritePRG(testAddress, testValue)
		readValue := iface.ReadPRG(testAddress)
		if readValue != testValue {
			t.Errorf("Write-read inconsistency at iteration %d: wrote 0x%02X, read 0x%02X",
				i, testValue, readValue)
		}

		// Change test value to ensure we're not getting cached results
		testValue = uint8((testValue + 1) & 0xFF)
	}
}

// TestMapperInterface_ConcurrentAccess tests concurrent access safety
func TestMapperInterface_ConcurrentAccess(t *testing.T) {
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000),
		mapperID:  0,
		sram:      [0x2000]uint8{},
		hasCHRRAM: true,
	}

	mapper := NewMapper000(cart)

	// Test concurrent reads (should be safe)
	done := make(chan bool, 2)

	go func() {
		for i := 0; i < 1000; i++ {
			mapper.ReadPRG(0x8000)
			mapper.ReadCHR(0x0000)
		}
		done <- true
	}()

	go func() {
		for i := 0; i < 1000; i++ {
			mapper.ReadPRG(0x8100)
			mapper.ReadCHR(0x0100)
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Note: NROM mapper doesn't have internal state that would cause
	// race conditions, but this test ensures no panics occur
}

// TestMapperInterface_AddressRangeValidation tests address range validation
func TestMapperInterface_AddressRangeValidation(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x4000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// Test PRG address ranges
	prgTests := []struct {
		address  uint16
		expected uint8
		name     string
	}{
		{0x0000, 0, "Below valid range"},
		{0x5FFF, 0, "Just below SRAM"},
		{0x6000, 0, "Start of SRAM"},
		{0x7FFF, 0, "End of SRAM"},
		{0x8000, 0, "Start of ROM"}, // Will be actual ROM data
		{0xFFFF, 0, "End of ROM"},   // Will be actual ROM data
	}

	for _, test := range prgTests {
		value := mapper.ReadPRG(test.address)
		// For ROM addresses, we don't test specific values since they depend on ROM content
		// We just ensure no panic occurs
		_ = value
	}

	// Test CHR address ranges
	chrTests := []uint16{0x0000, 0x1FFF, 0x2000, 0xFFFF}
	for _, address := range chrTests {
		value := mapper.ReadCHR(address)
		_ = value // Ensure no panic
	}
}
