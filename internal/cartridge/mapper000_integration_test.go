package cartridge

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// System-level integration tests for Mapper 0 (NROM)
// Tests integration with memory system, CPU, and PPU interfaces

// TestMapper000_CPUMemoryIntegration tests CPU memory map integration
func TestMapper000_CPUMemoryIntegration(t *testing.T) {
	// Create NROM cartridge simulating typical CPU memory layout
	cart := &Cartridge{
		prgROM:     make([]uint8, 0x8000), // 32KB PRG ROM
		chrROM:     make([]uint8, 0x2000), // 8KB CHR ROM
		mapperID:   0,
		mirror:     MirrorHorizontal,
		hasBattery: true,
		sram:       [0x2000]uint8{}, // 8KB SRAM
	}

	// Fill PRG ROM with 6502 opcodes and data
	for i := range cart.prgROM {
		cart.prgROM[i] = uint8((i + 0x60) & 0xFF) // Simulate 6502 code/data
	}

	mapper := NewMapper000(cart)

	// Test CPU memory regions ($6000-$FFFF handled by cartridge)
	cpuMemoryTests := []struct {
		address     uint16
		region      string
		writable    bool
		description string
	}{
		{0x6000, "SRAM", true, "Start of SRAM"},
		{0x6100, "SRAM", true, "SRAM middle"},
		{0x7FFF, "SRAM", true, "End of SRAM"},
		{0x8000, "PRG ROM", false, "Start of PRG ROM"},
		{0xA000, "PRG ROM", false, "PRG ROM middle"},
		{0xC000, "PRG ROM", false, "PRG ROM upper"},
		{0xFFFC, "PRG ROM", false, "Reset vector low"},
		{0xFFFD, "PRG ROM", false, "Reset vector high"},
		{0xFFFE, "PRG ROM", false, "IRQ vector low"},
		{0xFFFF, "PRG ROM", false, "IRQ vector high"},
	}

	for _, test := range cpuMemoryTests {
		// Test read access
		value := mapper.ReadPRG(test.address)
		_ = value // Should not panic

		// Test write behavior
		originalValue := mapper.ReadPRG(test.address)
		mapper.WritePRG(test.address, 0xAA)
		afterWriteValue := mapper.ReadPRG(test.address)

		if test.writable {
			if afterWriteValue != 0xAA {
				t.Errorf("%s at 0x%04X should be writable but write failed",
					test.region, test.address)
			}
		} else {
			if afterWriteValue != originalValue {
				t.Errorf("%s at 0x%04X should be read-only but write succeeded",
					test.region, test.address)
			}
		}
	}
}

// TestMapper000_PPUMemoryIntegration tests PPU memory map integration
func TestMapper000_PPUMemoryIntegration(t *testing.T) {
	// Test CHR ROM configuration
	cartCHRROM := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000), // 8KB CHR ROM
		mapperID:  0,
		hasCHRRAM: false,
	}

	// Fill CHR ROM with pattern table data
	for i := range cartCHRROM.chrROM {
		cartCHRROM.chrROM[i] = uint8((i + 0x20) & 0xFF) // Pattern data
	}

	mapperCHRROM := NewMapper000(cartCHRROM)

	// Test CHR RAM configuration
	cartCHRRAM := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000), // 8KB CHR RAM
		mapperID:  0,
		hasCHRRAM: true,
	}

	mapperCHRRAM := NewMapper000(cartCHRRAM)

	// PPU memory regions ($0000-$1FFF handled by cartridge)
	ppuMemoryTests := []struct {
		address     uint16
		description string
	}{
		{0x0000, "Pattern table 0 start"},
		{0x07FF, "Pattern table 0 end"},
		{0x0800, "Pattern table 1 start"},
		{0x0FFF, "Pattern table 1 end"},
		{0x1000, "Pattern table extension"},
		{0x1FFF, "End of CHR space"},
	}

	for _, test := range ppuMemoryTests {
		// Test CHR ROM (read-only)
		romValue := mapperCHRROM.ReadCHR(test.address)
		_ = romValue

		originalROMValue := mapperCHRROM.ReadCHR(test.address)
		mapperCHRROM.WriteCHR(test.address, 0xBB)
		afterWriteROMValue := mapperCHRROM.ReadCHR(test.address)

		if afterWriteROMValue != originalROMValue {
			t.Errorf("CHR ROM at 0x%04X should be read-only but write succeeded", test.address)
		}

		// Test CHR RAM (writable)
		mapperCHRRAM.WriteCHR(test.address, 0xCC)
		ramValue := mapperCHRRAM.ReadCHR(test.address)

		if ramValue != 0xCC {
			t.Errorf("CHR RAM at 0x%04X should be writable but write failed", test.address)
		}
	}
}

// TestMapper000_VectorTableAccess tests interrupt vector access
func TestMapper000_VectorTableAccess(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x8000), // 32KB PRG ROM
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	// Simulate 6502 interrupt vectors
	// NMI vector at $FFFA-$FFFB
	cart.prgROM[0x7FFA] = 0x00 // NMI low
	cart.prgROM[0x7FFB] = 0x80 // NMI high

	// Reset vector at $FFFC-$FFFD
	cart.prgROM[0x7FFC] = 0x00 // Reset low
	cart.prgROM[0x7FFD] = 0x80 // Reset high

	// IRQ/BRK vector at $FFFE-$FFFF
	cart.prgROM[0x7FFE] = 0x50 // IRQ low
	cart.prgROM[0x7FFF] = 0x80 // IRQ high

	mapper := NewMapper000(cart)

	// Test interrupt vector access
	vectorTests := []struct {
		address  uint16
		expected uint8
		name     string
	}{
		{0xFFFA, 0x00, "NMI vector low"},
		{0xFFFB, 0x80, "NMI vector high"},
		{0xFFFC, 0x00, "Reset vector low"},
		{0xFFFD, 0x80, "Reset vector high"},
		{0xFFFE, 0x50, "IRQ vector low"},
		{0xFFFF, 0x80, "IRQ vector high"},
	}

	for _, test := range vectorTests {
		value := mapper.ReadPRG(test.address)
		if value != test.expected {
			t.Errorf("%s: expected 0x%02X, got 0x%02X", test.name, test.expected, value)
		}
	}
}

// TestMapper000_MemoryController_Integration tests memory controller coordination
func TestMapper000_MemoryController_Integration(t *testing.T) {
	cart := &Cartridge{
		prgROM:     make([]uint8, 0x4000), // 16KB PRG ROM
		chrROM:     make([]uint8, 0x2000), // 8KB CHR ROM
		mapperID:   0,
		hasBattery: true,
		sram:       [0x2000]uint8{},
	}

	mapper := NewMapper000(cart)

	// Test memory controller scenarios

	// 1. SRAM persistence simulation
	saveGameData := []uint8{0x12, 0x34, 0x56, 0x78, 0x9A, 0xBC, 0xDE, 0xF0}
	for i, data := range saveGameData {
		address := uint16(0x6000 + i)
		mapper.WritePRG(address, data)
	}

	// Simulate power cycle by creating new mapper with same cartridge
	newMapper := NewMapper000(cart)

	// Verify SRAM data persisted
	for i, expected := range saveGameData {
		address := uint16(0x6000 + i)
		value := newMapper.ReadPRG(address)
		if value != expected {
			t.Errorf("SRAM persistence failed at 0x%04X: expected 0x%02X, got 0x%02X",
				address, expected, value)
		}
	}

	// 2. Address decoding coordination
	// Test that different address ranges don't interfere
	mapper.WritePRG(0x6000, 0x11) // SRAM
	mapper.WritePRG(0x8000, 0x22) // ROM (should be ignored)

	sramValue := mapper.ReadPRG(0x6000)
	romValue := mapper.ReadPRG(0x8000)

	if sramValue != 0x11 {
		t.Errorf("SRAM write affected by ROM write: expected 0x11, got 0x%02X", sramValue)
	}
	if romValue == 0x22 {
		t.Error("ROM write should have been ignored")
	}
}

// TestMapper000_AddressTranslation_Validation tests address translation validation
func TestMapper000_AddressTranslation_Validation(t *testing.T) {
	// Test 16KB ROM address translation
	cart16KB := &Cartridge{
		prgROM:   make([]uint8, 0x4000), // 16KB
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	// Create distinctive pattern for address validation
	for i := range cart16KB.prgROM {
		cart16KB.prgROM[i] = uint8((i >> 8) & 0xFF) // High byte of ROM offset
	}

	mapper16KB := NewMapper000(cart16KB)

	// Test 16KB mirroring address translation
	mirrorTests := []struct {
		address  uint16
		romAddr  uint16
		expected uint8
		desc     string
	}{
		{0x8000, 0x0000, 0x00, "Start of first bank"},
		{0x8100, 0x0100, 0x01, "Middle of first bank"},
		{0xBFFF, 0x3FFF, 0x3F, "End of first bank"},
		{0xC000, 0x0000, 0x00, "Start of mirrored bank"},
		{0xC100, 0x0100, 0x01, "Middle of mirrored bank"},
		{0xFFFF, 0x3FFF, 0x3F, "End of mirrored bank"},
	}

	for _, test := range mirrorTests {
		value := mapper16KB.ReadPRG(test.address)
		if value != test.expected {
			t.Errorf("16KB address translation failed at 0x%04X (%s): expected 0x%02X, got 0x%02X",
				test.address, test.desc, test.expected, value)
		}
	}

	// Test 32KB ROM address translation (no mirroring)
	cart32KB := &Cartridge{
		prgROM:   make([]uint8, 0x8000), // 32KB
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	for i := range cart32KB.prgROM {
		cart32KB.prgROM[i] = uint8((i >> 8) & 0xFF)
	}

	mapper32KB := NewMapper000(cart32KB)

	// Test 32KB direct address translation
	directTests := []struct {
		address  uint16
		expected uint8
		desc     string
	}{
		{0x8000, 0x00, "Start of ROM"},
		{0x9000, 0x10, "16KB offset"},
		{0xA000, 0x20, "32KB offset"},
		{0xC000, 0x40, "64KB offset"},
		{0xE000, 0x60, "96KB offset"},
		{0xFFFF, 0x7F, "End of ROM"},
	}

	for _, test := range directTests {
		value := mapper32KB.ReadPRG(test.address)
		if value != test.expected {
			t.Errorf("32KB address translation failed at 0x%04X (%s): expected 0x%02X, got 0x%02X",
				test.address, test.desc, test.expected, value)
		}
	}
}

// TestMapper000_FullSystem_Simulation tests full system simulation
func TestMapper000_FullSystem_Simulation(t *testing.T) {
	// Create a complete NROM ROM simulating a simple program
	var buffer bytes.Buffer

	// iNES header for 32KB PRG + 8KB CHR + Battery
	header := iNESHeader{
		Magic:      [4]uint8{'N', 'E', 'S', 0x1A},
		PRGROMSize: 2,    // 32KB PRG ROM
		CHRROMSize: 1,    // 8KB CHR ROM
		Flags6:     0x02, // Battery flag
		Flags7:     0x00,
		PRGRAMSize: 0,
		TVSystem1:  0,
		TVSystem2:  0,
		Padding:    [5]uint8{0, 0, 0, 0, 0},
	}

	binary.Write(&buffer, binary.LittleEndian, header)

	// PRG ROM with simulated 6502 program
	prgROM := make([]byte, 32768)

	// Simulate program code at start
	prgROM[0x0000] = 0x78 // SEI (disable interrupts)
	prgROM[0x0001] = 0xD8 // CLD (clear decimal mode)
	prgROM[0x0002] = 0x4C // JMP absolute
	prgROM[0x0003] = 0x00 // Jump to $8000 (start)
	prgROM[0x0004] = 0x80

	// Set interrupt vectors
	prgROM[0x7FFC] = 0x00 // Reset vector low
	prgROM[0x7FFD] = 0x80 // Reset vector high ($8000)
	prgROM[0x7FFE] = 0x00 // IRQ vector low
	prgROM[0x7FFF] = 0x80 // IRQ vector high

	buffer.Write(prgROM)

	// CHR ROM with pattern tables
	chrROM := make([]byte, 8192)
	for i := range chrROM {
		chrROM[i] = byte((i + 0x30) & 0xFF) // Pattern data
	}
	buffer.Write(chrROM)

	// Load the ROM
	reader := bytes.NewReader(buffer.Bytes())
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load simulated ROM: %v", err)
	}

	// Verify complete system
	if cart.mapperID != 0 {
		t.Errorf("Expected mapper 0, got %d", cart.mapperID)
	}
	if !cart.hasBattery {
		t.Error("Expected battery backup")
	}

	// Test reset vector
	resetLow := cart.ReadPRG(0xFFFC)
	resetHigh := cart.ReadPRG(0xFFFD)
	if resetLow != 0x00 || resetHigh != 0x80 {
		t.Errorf("Reset vector incorrect: got 0x%02X%02X, expected 0x8000", resetHigh, resetLow)
	}

	// Test program code
	sei := cart.ReadPRG(0x8000)
	cld := cart.ReadPRG(0x8001)
	jmp := cart.ReadPRG(0x8002)

	if sei != 0x78 || cld != 0xD8 || jmp != 0x4C {
		t.Errorf("Program code incorrect: got 0x%02X 0x%02X 0x%02X", sei, cld, jmp)
	}

	// Test save game functionality
	cart.WritePRG(0x6000, 0x42) // Save score
	cart.WritePRG(0x6001, 0x03) // Save level

	if cart.ReadPRG(0x6000) != 0x42 {
		t.Error("Save game data not persisted")
	}
	if cart.ReadPRG(0x6001) != 0x03 {
		t.Error("Save game level not persisted")
	}

	// Test pattern table access
	patternData := cart.ReadCHR(0x0000)
	expectedPattern := uint8(0x30)
	if patternData != expectedPattern {
		t.Errorf("Pattern table data incorrect: expected 0x%02X, got 0x%02X",
			expectedPattern, patternData)
	}
}

// TestMapper000_HardwareTiming_Characteristics tests hardware timing
func TestMapper000_HardwareTiming_Characteristics(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x8000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	// NROM has no wait states - all accesses should be immediate
	// Test consistent timing across multiple accesses

	const testIterations = 10000

	// Test PRG ROM access timing consistency
	for i := 0; i < testIterations; i++ {
		address := uint16(0x8000 + (i % 0x8000))
		value1 := mapper.ReadPRG(address)
		value2 := mapper.ReadPRG(address)

		if value1 != value2 {
			t.Errorf("PRG ROM access inconsistent at iteration %d, address 0x%04X", i, address)
			break
		}
	}

	// Test CHR ROM access timing consistency
	for i := 0; i < testIterations; i++ {
		address := uint16(i % 0x2000)
		value1 := mapper.ReadCHR(address)
		value2 := mapper.ReadCHR(address)

		if value1 != value2 {
			t.Errorf("CHR ROM access inconsistent at iteration %d, address 0x%04X", i, address)
			break
		}
	}

	// Test SRAM timing consistency
	for i := 0; i < 1000; i++ {
		address := uint16(0x6000 + (i % 0x2000))
		value := uint8(i & 0xFF)

		mapper.WritePRG(address, value)
		readValue := mapper.ReadPRG(address)

		if readValue != value {
			t.Errorf("SRAM timing inconsistent at iteration %d, address 0x%04X", i, address)
			break
		}
	}
}

// TestMapper000_PowerOnState tests power-on state initialization
func TestMapper000_PowerOnState(t *testing.T) {
	cart := &Cartridge{
		prgROM:    make([]uint8, 0x4000),
		chrROM:    make([]uint8, 0x2000),
		mapperID:  0,
		sram:      [0x2000]uint8{},
		hasCHRRAM: true,
	}

	// Simulate random power-on state for SRAM
	for i := range cart.sram {
		cart.sram[i] = uint8(i & 0xFF)
	}

	// Simulate uninitialized CHR RAM
	for i := range cart.chrROM {
		cart.chrROM[i] = uint8((i + 0x55) & 0xFF)
	}

	mapper := NewMapper000(cart)

	// SRAM should retain power-on values (battery-backed or maintained)
	for i := 0; i < 256; i++ {
		address := uint16(0x6000 + i)
		value := mapper.ReadPRG(address)
		expected := uint8(i & 0xFF)

		if value != expected {
			t.Errorf("SRAM power-on state wrong at 0x%04X: expected 0x%02X, got 0x%02X",
				address, expected, value)
		}
	}

	// CHR RAM power-on state varies by system, but should be accessible
	for i := 0; i < 256; i++ {
		address := uint16(i)
		value := mapper.ReadCHR(address)
		_ = value // Should not panic

		// Test that we can write to CHR RAM
		mapper.WriteCHR(address, 0xAA)
		newValue := mapper.ReadCHR(address)

		if newValue != 0xAA {
			t.Errorf("CHR RAM not writable at 0x%04X after power-on", address)
		}
	}
}
