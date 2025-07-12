package cartridge

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// NROM Compatibility Tests
// Tests compatibility with various NROM cartridge configurations and popular games

// createiNESROM creates a test iNES ROM with specified parameters
func createiNESROM(prgSize, chrSize uint8, flags6, flags7 uint8) []byte {
	var buffer bytes.Buffer

	// iNES header
	header := iNESHeader{
		Magic:      [4]uint8{'N', 'E', 'S', 0x1A},
		PRGROMSize: prgSize,
		CHRROMSize: chrSize,
		Flags6:     flags6,
		Flags7:     flags7,
		PRGRAMSize: 0,
		TVSystem1:  0,
		TVSystem2:  0,
		Padding:    [5]uint8{0, 0, 0, 0, 0},
	}

	binary.Write(&buffer, binary.LittleEndian, header)

	// PRG ROM data
	prgData := make([]byte, int(prgSize)*16384)
	for i := range prgData {
		prgData[i] = byte((i + 0x10) & 0xFF) // Distinct pattern
	}
	buffer.Write(prgData)

	// CHR ROM data
	if chrSize > 0 {
		chrData := make([]byte, int(chrSize)*8192)
		for i := range chrData {
			chrData[i] = byte((i + 0x80) & 0xFF) // Distinct pattern
		}
		buffer.Write(chrData)
	}

	return buffer.Bytes()
}

// TestNROM_SuperMarioBros_Configuration tests Super Mario Bros. ROM configuration
func TestNROM_SuperMarioBros_Configuration(t *testing.T) {
	// Super Mario Bros: 32KB PRG ROM, 8KB CHR ROM, Horizontal mirroring
	romData := createiNESROM(2, 1, 0x00, 0x00) // 32KB PRG, 8KB CHR, horizontal mirroring

	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load Super Mario Bros ROM: %v", err)
	}

	// Verify configuration
	if cart.mapperID != 0 {
		t.Errorf("Expected mapper 0, got %d", cart.mapperID)
	}
	if cart.mirror != MirrorHorizontal {
		t.Errorf("Expected horizontal mirroring, got %v", cart.mirror)
	}
	if len(cart.prgROM) != 0x8000 {
		t.Errorf("Expected 32KB PRG ROM, got %d bytes", len(cart.prgROM))
	}
	if len(cart.chrROM) != 0x2000 {
		t.Errorf("Expected 8KB CHR ROM, got %d bytes", len(cart.chrROM))
	}
	if cart.hasCHRRAM {
		t.Error("Expected CHR ROM, got CHR RAM")
	}

	// Test reset vector area (typical for SMB)
	resetVectorLow := cart.ReadPRG(0xFFFC)
	resetVectorHigh := cart.ReadPRG(0xFFFD)

	// Should read actual data, not zero
	if resetVectorLow == 0 && resetVectorHigh == 0 {
		t.Error("Reset vector should not be zero in typical ROM")
	}

	// Test typical game code area
	codeValue := cart.ReadPRG(0x8000)
	expectedValue := uint8(0x10) // Based on our pattern
	if codeValue != expectedValue {
		t.Errorf("Expected code value 0x%02X at 0x8000, got 0x%02X", expectedValue, codeValue)
	}
}

// TestNROM_Donkey_Kong_Configuration tests Donkey Kong ROM configuration
func TestNROM_Donkey_Kong_Configuration(t *testing.T) {
	// Donkey Kong: 16KB PRG ROM, 8KB CHR ROM, Vertical mirroring
	romData := createiNESROM(1, 1, 0x01, 0x00) // 16KB PRG, 8KB CHR, vertical mirroring

	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load Donkey Kong ROM: %v", err)
	}

	// Verify configuration
	if cart.mapperID != 0 {
		t.Errorf("Expected mapper 0, got %d", cart.mapperID)
	}
	if cart.mirror != MirrorVertical {
		t.Errorf("Expected vertical mirroring, got %v", cart.mirror)
	}
	if len(cart.prgROM) != 0x4000 {
		t.Errorf("Expected 16KB PRG ROM, got %d bytes", len(cart.prgROM))
	}

	// Test 16KB mirroring behavior
	value1 := cart.ReadPRG(0x8000)
	value2 := cart.ReadPRG(0xC000) // Should mirror to same location

	if value1 != value2 {
		t.Errorf("16KB ROM mirroring failed: 0x8000=0x%02X, 0xC000=0x%02X", value1, value2)
	}

	// Test end of ROM mirroring
	value3 := cart.ReadPRG(0xBFFF)
	value4 := cart.ReadPRG(0xFFFF) // Should mirror

	if value3 != value4 {
		t.Errorf("16KB ROM end mirroring failed: 0xBFFF=0x%02X, 0xFFFF=0x%02X", value3, value4)
	}
}

// TestNROM_IceClimber_Configuration tests Ice Climber ROM configuration
func TestNROM_IceClimber_Configuration(t *testing.T) {
	// Ice Climber: 32KB PRG ROM, 8KB CHR ROM, Vertical mirroring
	romData := createiNESROM(2, 1, 0x01, 0x00) // 32KB PRG, 8KB CHR, vertical mirroring

	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load Ice Climber ROM: %v", err)
	}

	// Verify no mirroring in 32KB ROM
	value1 := cart.ReadPRG(0x8000)
	value2 := cart.ReadPRG(0xC000)

	// Should be different values (no mirroring)
	if value1 == value2 {
		t.Error("32KB ROM should not mirror - found identical values")
	}

	// Test full address range
	testAddresses := []uint16{0x8000, 0x9000, 0xA000, 0xB000, 0xC000, 0xD000, 0xE000, 0xF000}
	values := make([]uint8, len(testAddresses))

	for i, addr := range testAddresses {
		values[i] = cart.ReadPRG(addr)
	}

	// All values should be different (based on our pattern)
	for i := 1; i < len(values); i++ {
		if values[i] == values[0] {
			t.Errorf("Address 0x%04X returned same value as 0x8000: 0x%02X",
				testAddresses[i], values[i])
		}
	}
}

// TestNROM_HomeBrew_CHR_RAM tests homebrew ROM with CHR RAM
func TestNROM_HomeBrew_CHR_RAM(t *testing.T) {
	// Homebrew: 16KB PRG ROM, CHR RAM, Horizontal mirroring
	romData := createiNESROM(1, 0, 0x00, 0x00) // 16KB PRG, no CHR ROM (CHR RAM)

	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load homebrew ROM: %v", err)
	}

	// Verify CHR RAM configuration
	if !cart.hasCHRRAM {
		t.Error("Expected CHR RAM, got CHR ROM")
	}
	if len(cart.chrROM) != 0x2000 {
		t.Errorf("Expected 8KB CHR RAM, got %d bytes", len(cart.chrROM))
	}

	// Test CHR RAM is writable
	cart.WriteCHR(0x0000, 0xAA)
	value := cart.ReadCHR(0x0000)
	if value != 0xAA {
		t.Errorf("CHR RAM write failed: expected 0xAA, got 0x%02X", value)
	}

	// Test CHR RAM pattern tiles (common use case)
	tileData := []uint8{0x3C, 0x42, 0x81, 0xA5, 0x81, 0x42, 0x3C, 0x00}
	for i, data := range tileData {
		cart.WriteCHR(uint16(i), data)
	}

	for i, expected := range tileData {
		value := cart.ReadCHR(uint16(i))
		if value != expected {
			t.Errorf("Tile data at offset %d: expected 0x%02X, got 0x%02X", i, expected, value)
		}
	}
}

// TestNROM_Battery_SRAM tests battery-backed SRAM functionality
func TestNROM_Battery_SRAM(t *testing.T) {
	// ROM with battery-backed SRAM
	romData := createiNESROM(2, 1, 0x02, 0x00) // 32KB PRG, 8KB CHR, battery flag

	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load battery ROM: %v", err)
	}

	// Verify battery flag
	if !cart.hasBattery {
		t.Error("Expected battery-backed SRAM")
	}

	// Test save game data simulation
	saveData := []struct {
		address uint16
		value   uint8
	}{
		{0x6000, 0x01}, // Game progress
		{0x6001, 0x05}, // Lives
		{0x6002, 0x12}, // Score high byte
		{0x6003, 0x34}, // Score low byte
		{0x6004, 0x03}, // World
		{0x6005, 0x02}, // Level
		{0x6010, 0xFF}, // Completion flags
	}

	// Simulate saving game data
	for _, save := range saveData {
		cart.WritePRG(save.address, save.value)
	}

	// Verify data persists
	for _, save := range saveData {
		value := cart.ReadPRG(save.address)
		if value != save.value {
			t.Errorf("Save data at 0x%04X: expected 0x%02X, got 0x%02X",
				save.address, save.value, value)
		}
	}

	// Test SRAM boundary
	cart.WritePRG(0x7FFF, 0xEE) // Last SRAM address
	value := cart.ReadPRG(0x7FFF)
	if value != 0xEE {
		t.Errorf("SRAM boundary test failed: expected 0xEE, got 0x%02X", value)
	}
}

// TestNROM_FourScreen_Mirroring tests four-screen mirroring (rare)
func TestNROM_FourScreen_Mirroring(t *testing.T) {
	// ROM with four-screen mirroring
	romData := createiNESROM(2, 1, 0x08, 0x00) // 32KB PRG, 8KB CHR, four-screen

	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load four-screen ROM: %v", err)
	}

	// Verify four-screen mirroring
	if cart.mirror != MirrorFourScreen {
		t.Errorf("Expected four-screen mirroring, got %v", cart.mirror)
	}

	// Four-screen mirroring requires additional VRAM (not tested here)
	// but the cartridge should load correctly
}

// TestNROM_Trainer_Support tests trainer support (512-byte trainer)
func TestNROM_Trainer_Support(t *testing.T) {
	var buffer bytes.Buffer

	// Create ROM with trainer
	header := iNESHeader{
		Magic:      [4]uint8{'N', 'E', 'S', 0x1A},
		PRGROMSize: 1,    // 16KB
		CHRROMSize: 1,    // 8KB
		Flags6:     0x04, // Trainer present
		Flags7:     0x00,
		PRGRAMSize: 0,
		TVSystem1:  0,
		TVSystem2:  0,
		Padding:    [5]uint8{0, 0, 0, 0, 0},
	}

	binary.Write(&buffer, binary.LittleEndian, header)

	// Trainer data (512 bytes)
	trainer := make([]byte, 512)
	for i := range trainer {
		trainer[i] = byte(0xCC) // Trainer pattern
	}
	buffer.Write(trainer)

	// PRG ROM data
	prgData := make([]byte, 16384)
	for i := range prgData {
		prgData[i] = byte(0x33)
	}
	buffer.Write(prgData)

	// CHR ROM data
	chrData := make([]byte, 8192)
	for i := range chrData {
		chrData[i] = byte(0x44)
	}
	buffer.Write(chrData)

	reader := bytes.NewReader(buffer.Bytes())
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load ROM with trainer: %v", err)
	}

	// Verify PRG ROM loads correctly (trainer should be skipped)
	value := cart.ReadPRG(0x8000)
	if value != 0x33 {
		t.Errorf("Expected PRG ROM value 0x33, got 0x%02X (trainer may not have been skipped)", value)
	}

	// Verify CHR ROM loads correctly
	chrValue := cart.ReadCHR(0x0000)
	if chrValue != 0x44 {
		t.Errorf("Expected CHR ROM value 0x44, got 0x%02X", chrValue)
	}
}

// TestNROM_MemoryStress tests memory stress scenarios
func TestNROM_MemoryStress(t *testing.T) {
	// Large ROM configuration
	romData := createiNESROM(2, 1, 0x02, 0x00) // 32KB PRG, 8KB CHR, battery

	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load ROM for stress test: %v", err)
	}

	// Stress test SRAM with repetitive writes
	const iterations = 1000
	for i := 0; i < iterations; i++ {
		address := uint16(0x6000 + (i % 0x2000))
		value := uint8(i & 0xFF)

		cart.WritePRG(address, value)
		readValue := cart.ReadPRG(address)

		if readValue != value {
			t.Errorf("Stress test failed at iteration %d: address 0x%04X, expected 0x%02X, got 0x%02X",
				i, address, value, readValue)
			break
		}
	}

	// Stress test CHR ROM with repetitive reads
	for i := 0; i < iterations; i++ {
		address := uint16(i % 0x2000)
		value1 := cart.ReadCHR(address)
		value2 := cart.ReadCHR(address)

		if value1 != value2 {
			t.Errorf("CHR ROM stress test failed: inconsistent reads at 0x%04X", address)
			break
		}
	}
}

// TestNROM_EdgeCase_MinimalROM tests minimal valid ROM
func TestNROM_EdgeCase_MinimalROM(t *testing.T) {
	// Minimal ROM: 16KB PRG, no CHR (CHR RAM)
	romData := createiNESROM(1, 0, 0x00, 0x00)

	reader := bytes.NewReader(romData)
	cart, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load minimal ROM: %v", err)
	}

	// Should have CHR RAM
	if !cart.hasCHRRAM {
		t.Error("Minimal ROM should have CHR RAM")
	}

	// Should support basic operations
	cart.WritePRG(0x6000, 0x11)
	cart.WriteCHR(0x0000, 0x22)

	if cart.ReadPRG(0x6000) != 0x11 {
		t.Error("SRAM write failed in minimal ROM")
	}
	if cart.ReadCHR(0x0000) != 0x22 {
		t.Error("CHR RAM write failed in minimal ROM")
	}
}

// TestNROM_Compatibility_ROMSizes tests various ROM size combinations
func TestNROM_Compatibility_ROMSizes(t *testing.T) {
	testCases := []struct {
		prgSize      uint8
		chrSize      uint8
		name         string
		shouldMirror bool
	}{
		{1, 0, "16KB PRG + CHR RAM", true},
		{1, 1, "16KB PRG + 8KB CHR", true},
		{2, 0, "32KB PRG + CHR RAM", false},
		{2, 1, "32KB PRG + 8KB CHR", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			romData := createiNESROM(tc.prgSize, tc.chrSize, 0x00, 0x00)

			reader := bytes.NewReader(romData)
			cart, err := LoadFromReader(reader)
			if err != nil {
				t.Fatalf("Failed to load %s: %v", tc.name, err)
			}

			// Test PRG ROM size
			expectedPRGSize := int(tc.prgSize) * 16384
			if len(cart.prgROM) != expectedPRGSize {
				t.Errorf("Expected %d bytes PRG ROM, got %d", expectedPRGSize, len(cart.prgROM))
			}

			// Test CHR configuration
			expectedCHRSize := 8192 // Always 8KB for NROM
			if len(cart.chrROM) != expectedCHRSize {
				t.Errorf("Expected %d bytes CHR memory, got %d", expectedCHRSize, len(cart.chrROM))
			}

			// Test mirroring behavior
			value1 := cart.ReadPRG(0x8000)
			value2 := cart.ReadPRG(0xC000)

			if tc.shouldMirror {
				if value1 != value2 {
					t.Errorf("Expected PRG mirroring for %s", tc.name)
				}
			} else {
				if value1 == value2 {
					t.Errorf("Expected no PRG mirroring for %s", tc.name)
				}
			}
		})
	}
}

// TestNROM_MirroringModes tests all mirroring modes
func TestNROM_MirroringModes(t *testing.T) {
	mirroringTests := []struct {
		flags6   uint8
		expected MirrorMode
		name     string
	}{
		{0x00, MirrorHorizontal, "Horizontal"},
		{0x01, MirrorVertical, "Vertical"},
		{0x08, MirrorFourScreen, "Four-screen"},
		{0x09, MirrorFourScreen, "Four-screen (overrides vertical)"},
	}

	for _, test := range mirroringTests {
		t.Run(test.name, func(t *testing.T) {
			romData := createiNESROM(1, 1, test.flags6, 0x00)

			reader := bytes.NewReader(romData)
			cart, err := LoadFromReader(reader)
			if err != nil {
				t.Fatalf("Failed to load ROM for %s mirroring: %v", test.name, err)
			}

			if cart.mirror != test.expected {
				t.Errorf("Expected %v mirroring, got %v", test.expected, cart.mirror)
			}
		})
	}
}
