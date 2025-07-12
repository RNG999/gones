package cartridge

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Test data constants for iNES header construction
const (
	validINESMagic = "NES\x1A"
	invalidMagic   = "ROM\x1A"
)

// createValidINESHeader creates a valid 16-byte iNES header for testing
func createValidINESHeader(prgSize, chrSize, mapper, flags6, flags7 uint8) []byte {
	header := make([]byte, 16)
	copy(header[0:4], validINESMagic)
	header[4] = prgSize // PRG ROM size in 16KB units
	header[5] = chrSize // CHR ROM size in 8KB units

	// If mapper is non-zero, encode it into flags6/flags7, otherwise use provided flags
	if mapper != 0 {
		header[6] = (mapper << 4) | (flags6 & 0x0F)   // Mapper lower nibble + other flags
		header[7] = (mapper & 0xF0) | (flags7 & 0x0F) // Mapper upper nibble + other flags
	} else {
		header[6] = flags6 // Mapper lower nibble, mirroring, battery, trainer
		header[7] = flags7 // Mapper upper nibble, format
	}
	// Remaining bytes 8-15 are padding (zeros)
	return header
}

// createMinimalValidROM creates a minimal valid iNES ROM with specified sizes
func createMinimalValidROM(prgSize, chrSize uint8) []byte {
	header := createValidINESHeader(prgSize, chrSize, 0, 0, 0)

	// Add PRG ROM data (filled with pattern for verification)
	prgData := make([]byte, int(prgSize)*16384)
	for i := range prgData {
		prgData[i] = uint8(i % 256)
	}

	// Add CHR ROM data if specified
	chrData := make([]byte, int(chrSize)*8192)
	for i := range chrData {
		chrData[i] = uint8((i + 128) % 256)
	}

	// Combine all data
	rom := append(header, prgData...)
	if chrSize > 0 {
		rom = append(rom, chrData...)
	}

	return rom
}

func TestLoadFromReader_ValidiNESFormat_ShouldSucceed(t *testing.T) {
	tests := []struct {
		name        string
		prgSize     uint8
		chrSize     uint8
		expectedPRG int
		expectedCHR int
	}{
		{"16KB PRG, 8KB CHR", 1, 1, 16384, 8192},
		{"32KB PRG, 8KB CHR", 2, 1, 32768, 8192},
		{"16KB PRG, CHR RAM", 1, 0, 16384, 8192}, // CHR RAM defaults to 8KB
		{"32KB PRG, 16KB CHR", 2, 2, 32768, 16384},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			romData := createMinimalValidROM(tt.prgSize, tt.chrSize)
			reader := bytes.NewReader(romData)

			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Fatalf("Expected successful load, got error: %v", err)
			}
			if cartridge == nil {
				t.Fatal("Expected cartridge, got nil")
			}
			if len(cartridge.prgROM) != tt.expectedPRG {
				t.Errorf("Expected PRG ROM size %d, got %d", tt.expectedPRG, len(cartridge.prgROM))
			}
			if len(cartridge.chrROM) != tt.expectedCHR {
				t.Errorf("Expected CHR ROM size %d, got %d", tt.expectedCHR, len(cartridge.chrROM))
			}
		})
	}
}

func TestLoadFromReader_InvalidMagicNumber_ShouldFail(t *testing.T) {
	header := make([]byte, 16)
	copy(header[0:4], invalidMagic)
	header[4] = 1 // 16KB PRG ROM
	header[5] = 1 // 8KB CHR ROM

	// Add minimal ROM data
	prgData := make([]byte, 16384)
	chrData := make([]byte, 8192)
	romData := append(header, prgData...)
	romData = append(romData, chrData...)

	reader := bytes.NewReader(romData)

	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for invalid magic number, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for invalid magic, got cartridge")
	}
	if !strings.Contains(err.Error(), "invalid iNES file") {
		t.Errorf("Expected 'invalid iNES file' error, got: %v", err)
	}
}

func TestLoadFromReader_MapperIdentification_ShouldExtractCorrectly(t *testing.T) {
	tests := []struct {
		name           string
		flags6         uint8
		flags7         uint8
		expectedMapper uint8
	}{
		{"Mapper 0 (NROM)", 0x00, 0x00, 0},
		{"Mapper 1 (MMC1)", 0x10, 0x00, 1},
		{"Mapper 4 (MMC3)", 0x40, 0x00, 4},
		{"Mapper 2 from flags7", 0x00, 0x20, 2},
		{"Mapper 15 combined", 0xF0, 0x00, 15},
		{"Mapper 240 combined", 0x00, 0xF0, 240},
		{"Mapper 255 max", 0xF0, 0xF0, 255},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createValidINESHeader(1, 1, 0, tt.flags6, tt.flags7)
			prgData := make([]byte, 16384)
			chrData := make([]byte, 8192)
			romData := append(header, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Fatalf("Expected success, got error: %v", err)
			}
			if cartridge.mapperID != tt.expectedMapper {
				t.Errorf("Expected mapper ID %d, got %d", tt.expectedMapper, cartridge.mapperID)
			}
		})
	}
}

func TestLoadFromReader_MirroringModes_ShouldDetectCorrectly(t *testing.T) {
	tests := []struct {
		name           string
		flags6         uint8
		expectedMirror MirrorMode
	}{
		{"Horizontal mirroring", 0x00, MirrorHorizontal},
		{"Vertical mirroring", 0x01, MirrorVertical},
		{"Four-screen mirroring", 0x08, MirrorFourScreen},
		{"Four-screen overrides vertical", 0x09, MirrorFourScreen},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createValidINESHeader(1, 1, 0, tt.flags6, 0)
			prgData := make([]byte, 16384)
			chrData := make([]byte, 8192)
			romData := append(header, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Fatalf("Expected success, got error: %v", err)
			}
			if cartridge.mirror != tt.expectedMirror {
				t.Errorf("Expected mirror mode %d, got %d", tt.expectedMirror, cartridge.mirror)
			}
		})
	}
}

func TestLoadFromReader_BatteryDetection_ShouldIdentifyCorrectly(t *testing.T) {
	tests := []struct {
		name       string
		flags6     uint8
		hasBattery bool
	}{
		{"No battery", 0x00, false},
		{"Has battery", 0x02, true},
		{"Battery with other flags", 0x03, true}, // Battery + vertical mirroring
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createValidINESHeader(1, 1, 0, tt.flags6, 0)
			prgData := make([]byte, 16384)
			chrData := make([]byte, 8192)
			romData := append(header, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Fatalf("Expected success, got error: %v", err)
			}
			if cartridge.hasBattery != tt.hasBattery {
				t.Errorf("Expected battery %v, got %v", tt.hasBattery, cartridge.hasBattery)
			}
		})
	}
}

func TestLoadFromReader_TrainerHandling_ShouldSkipCorrectly(t *testing.T) {
	// Create ROM with trainer flag set
	header := createValidINESHeader(1, 1, 0, 0x04, 0) // Trainer flag set
	trainerData := make([]byte, 512)
	for i := range trainerData {
		trainerData[i] = 0xFF // Fill trainer with pattern
	}
	prgData := make([]byte, 16384)
	for i := range prgData {
		prgData[i] = uint8(i % 256) // Different pattern for PRG
	}
	chrData := make([]byte, 8192)

	romData := append(header, trainerData...)
	romData = append(romData, prgData...)
	romData = append(romData, chrData...)

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Expected success, got error: %v", err)
	}

	// Verify that PRG data starts with expected pattern, not trainer pattern
	if cartridge.prgROM[0] != 0 || cartridge.prgROM[1] != 1 {
		t.Error("PRG ROM data doesn't match expected pattern, trainer may not have been skipped")
	}
}

func TestLoadFromReader_IncompleteHeader_ShouldFail(t *testing.T) {
	incompleteHeader := []byte("NES\x1A\x01\x01") // Only 6 bytes
	reader := bytes.NewReader(incompleteHeader)

	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for incomplete header, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for incomplete header")
	}
}

func TestLoadFromReader_IncompletePRGData_ShouldFail(t *testing.T) {
	header := createValidINESHeader(1, 1, 0, 0, 0)
	incompletePRG := make([]byte, 8192) // Only half the expected PRG data
	romData := append(header, incompletePRG...)

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for incomplete PRG data, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for incomplete PRG data")
	}
}

func TestLoadFromReader_IncompleteCHRData_ShouldFail(t *testing.T) {
	header := createValidINESHeader(1, 1, 0, 0, 0)
	prgData := make([]byte, 16384)
	incompleteCHR := make([]byte, 4096) // Only half the expected CHR data
	romData := append(header, prgData...)
	romData = append(romData, incompleteCHR...)

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for incomplete CHR data, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for incomplete CHR data")
	}
}

func TestLoadFromReader_ZeroPRGSize_ShouldFail(t *testing.T) {
	header := createValidINESHeader(0, 1, 0, 0, 0) // Zero PRG ROM size
	chrData := make([]byte, 8192)
	romData := append(header, chrData...)

	reader := bytes.NewReader(romData)
	_, err := LoadFromReader(reader)

	// This will create empty PRG ROM slice and fail when trying to read 0 bytes
	if err == nil {
		t.Fatal("Expected error for zero PRG size, got success")
	}
}

func TestLoadFromReader_MaximumSizes_ShouldHandleCorrectly(t *testing.T) {
	// Test with maximum typical sizes
	header := createValidINESHeader(255, 255, 0, 0, 0) // Max sizes
	prgData := make([]byte, 255*16384)                 // Very large PRG ROM
	chrData := make([]byte, 255*8192)                  // Very large CHR ROM

	romData := append(header, prgData...)
	romData = append(romData, chrData...)

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Expected success for maximum sizes, got error: %v", err)
	}
	if len(cartridge.prgROM) != 255*16384 {
		t.Errorf("Expected PRG ROM size %d, got %d", 255*16384, len(cartridge.prgROM))
	}
	if len(cartridge.chrROM) != 255*8192 {
		t.Errorf("Expected CHR ROM size %d, got %d", 255*8192, len(cartridge.chrROM))
	}
}

func TestLoadFromFile_NonexistentFile_ShouldFail(t *testing.T) {
	cartridge, err := LoadFromFile("/nonexistent/path/file.nes")

	if err == nil {
		t.Fatal("Expected error for nonexistent file, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for nonexistent file")
	}
}

func TestCartridge_PRGAccess_ShouldDelegateToMapper(t *testing.T) {
	romData := createMinimalValidROM(1, 1)
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Test PRG read - should delegate to mapper
	value := cartridge.ReadPRG(0x8000)

	// Value should match pattern from createMinimalValidROM
	expectedValue := uint8(0) // First byte of pattern
	if value != expectedValue {
		t.Errorf("Expected PRG read value %d, got %d", expectedValue, value)
	}

	// Test PRG write - should delegate to mapper (for SRAM area)
	cartridge.WritePRG(0x6000, 0x42)
	readBack := cartridge.ReadPRG(0x6000)

	if readBack != 0x42 {
		t.Errorf("Expected PRG write/read value 0x42, got 0x%02X", readBack)
	}
}

func TestCartridge_CHRAccess_ShouldDelegateToMapper(t *testing.T) {
	romData := createMinimalValidROM(1, 1)
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Test CHR read - should delegate to mapper
	value := cartridge.ReadCHR(0x0000)

	// Value should match pattern from createMinimalValidROM
	expectedValue := uint8(128) // CHR pattern starts at 128
	if value != expectedValue {
		t.Errorf("Expected CHR read value %d, got %d", expectedValue, value)
	}
}

func TestCartridge_CHRRAMAccess_ShouldAllowWriteRead(t *testing.T) {
	// Create ROM with CHR RAM (chrSize = 0)
	romData := createMinimalValidROM(1, 0)
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// CHR RAM should be writable
	cartridge.WriteCHR(0x0000, 0x55)
	value := cartridge.ReadCHR(0x0000)

	if value != 0x55 {
		t.Errorf("Expected CHR RAM write/read value 0x55, got 0x%02X", value)
	}
}

// Benchmark tests for performance validation
func BenchmarkLoadFromReader_SmallROM(b *testing.B) {
	romData := createMinimalValidROM(1, 1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(romData)
		_, err := LoadFromReader(reader)
		if err != nil {
			b.Fatalf("Failed to load ROM: %v", err)
		}
	}
}

func BenchmarkLoadFromReader_LargeROM(b *testing.B) {
	romData := createMinimalValidROM(32, 32) // 512KB PRG + 256KB CHR

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(romData)
		_, err := LoadFromReader(reader)
		if err != nil {
			b.Fatalf("Failed to load ROM: %v", err)
		}
	}
}

// Helper function to create test ROM files on disk for file-based tests
func createTestROMFile(t *testing.T, data []byte) string {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.nes")

	err := os.WriteFile(filename, data, 0644)
	if err != nil {
		t.Fatalf("Failed to create test ROM file: %v", err)
	}

	return filename
}

func TestLoadFromFile_ValidFile_ShouldSucceed(t *testing.T) {
	romData := createMinimalValidROM(1, 1)
	filename := createTestROMFile(t, romData)

	cartridge, err := LoadFromFile(filename)

	if err != nil {
		t.Fatalf("Expected success loading from file, got error: %v", err)
	}
	if cartridge == nil {
		t.Fatal("Expected cartridge, got nil")
	}
}

func TestLoadFromFile_EmptyFile_ShouldFail(t *testing.T) {
	filename := createTestROMFile(t, []byte{})

	cartridge, err := LoadFromFile(filename)

	if err == nil {
		t.Fatal("Expected error for empty file, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for empty file")
	}
}

// Test createMapper function behavior
func TestCreateMapper_UnknownMapper_ShouldDefaultToMapper0(t *testing.T) {
	// Create a cartridge with unknown mapper ID
	cart := &Cartridge{
		prgROM:   make([]uint8, 16384),
		chrROM:   make([]uint8, 8192),
		mapperID: 99, // Unknown mapper
	}

	mapper := createMapper(99, cart)

	// Should create Mapper000 as default
	if mapper == nil {
		t.Fatal("Expected mapper, got nil")
	}

	// Verify it behaves like Mapper000 by testing PRG access
	mapper.WritePRG(0x6000, 0x42)
	value := mapper.ReadPRG(0x6000)
	if value != 0x42 {
		t.Error("Mapper doesn't behave like Mapper000")
	}
}

func TestCreateMapper_Mapper0_ShouldCreateCorrectType(t *testing.T) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 16384),
		chrROM:   make([]uint8, 8192),
		mapperID: 0,
	}

	mapper := createMapper(0, cart)

	// Should create Mapper000
	if mapper == nil {
		t.Fatal("Expected mapper, got nil")
	}

	// Type assertion to verify correct type
	if _, ok := mapper.(*Mapper000); !ok {
		t.Error("Expected Mapper000 type")
	}
}
