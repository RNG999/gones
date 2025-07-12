package cartridge

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// TestROMGenerator provides utilities to create test ROM data for various scenarios
type TestROMGenerator struct{}

// CreateValidROMData creates a valid iNES ROM with specified parameters
func (g *TestROMGenerator) CreateValidROMData(prgSize, chrSize, mapper uint8, flags6, flags7 uint8) []byte {
	header := make([]byte, 16)
	copy(header[0:4], "NES\x1A")
	header[4] = prgSize
	header[5] = chrSize
	header[6] = flags6 | (mapper << 4)   // Lower 4 bits of mapper
	header[7] = flags7 | (mapper & 0xF0) // Upper 4 bits of mapper
	// Bytes 8-15 remain zero (padding)

	// Create PRG ROM data with identifiable pattern
	prgData := make([]byte, int(prgSize)*16384)
	for i := range prgData {
		prgData[i] = uint8((i / 256) % 256) // Creates a pattern that changes every 256 bytes
	}

	// Create CHR ROM data with different pattern
	chrData := make([]byte, int(chrSize)*8192)
	for i := range chrData {
		chrData[i] = uint8(255 - (i % 256)) // Inverted pattern for CHR
	}

	// Combine all data
	rom := append(header, prgData...)
	if chrSize > 0 {
		rom = append(rom, chrData...)
	}

	return rom
}

// CreateROMWithTrainer creates a ROM with trainer data
func (g *TestROMGenerator) CreateROMWithTrainer(prgSize, chrSize uint8) []byte {
	header := make([]byte, 16)
	copy(header[0:4], "NES\x1A")
	header[4] = prgSize
	header[5] = chrSize
	header[6] = 0x04 // Trainer flag set
	header[7] = 0x00

	// Create trainer data (512 bytes)
	trainer := make([]byte, 512)
	for i := range trainer {
		trainer[i] = 0xAA // Distinctive pattern
	}

	// Create PRG ROM data
	prgData := make([]byte, int(prgSize)*16384)
	for i := range prgData {
		prgData[i] = uint8(i % 256) // Different from trainer pattern
	}

	// Create CHR ROM data
	chrData := make([]byte, int(chrSize)*8192)
	for i := range chrData {
		chrData[i] = uint8((i + 128) % 256)
	}

	// Combine: header + trainer + PRG + CHR
	rom := append(header, trainer...)
	rom = append(rom, prgData...)
	if chrSize > 0 {
		rom = append(rom, chrData...)
	}

	return rom
}

// CreateCorruptedHeader creates a ROM with corrupted header
func (g *TestROMGenerator) CreateCorruptedHeader(corruptionType string) []byte {
	header := make([]byte, 16)

	switch corruptionType {
	case "invalid_magic":
		copy(header[0:4], "ROM\x1A") // Wrong magic
		header[4] = 1                // 16KB PRG
		header[5] = 1                // 8KB CHR

	case "truncated_header":
		copy(header[0:4], "NES\x1A")
		return header[:8] // Only 8 bytes instead of 16

	case "zero_prg":
		copy(header[0:4], "NES\x1A")
		header[4] = 0 // Zero PRG ROM size
		header[5] = 1 // 8KB CHR

	case "excessive_size":
		copy(header[0:4], "NES\x1A")
		header[4] = 255 // Maximum PRG size
		header[5] = 255 // Maximum CHR size

	default:
		copy(header[0:4], "NES\x1A")
		header[4] = 1
		header[5] = 1
	}

	// Add minimal ROM data for non-truncated cases
	if corruptionType != "truncated_header" {
		prgSize := int(header[4]) * 16384
		chrSize := int(header[5]) * 8192

		if prgSize > 0 {
			prgData := make([]byte, prgSize)
			header = append(header, prgData...)
		}

		if chrSize > 0 {
			chrData := make([]byte, chrSize)
			header = append(header, chrData...)
		}
	}

	return header
}

// CreateIncompleteROM creates a ROM with incomplete data
func (g *TestROMGenerator) CreateIncompleteROM(missingPart string) []byte {
	header := make([]byte, 16)
	copy(header[0:4], "NES\x1A")
	header[4] = 2 // 32KB PRG
	header[5] = 1 // 8KB CHR

	switch missingPart {
	case "partial_prg":
		// Only half the expected PRG data
		prgData := make([]byte, 16384) // Should be 32KB (32768)
		return append(header, prgData...)

	case "missing_chr":
		// Complete PRG but no CHR data
		prgData := make([]byte, 32768)
		return append(header, prgData...)

	case "partial_chr":
		// Complete PRG but partial CHR data
		prgData := make([]byte, 32768)
		chrData := make([]byte, 4096) // Should be 8KB (8192)
		rom := append(header, prgData...)
		return append(rom, chrData...)

	default:
		return header
	}
}

// CreateMapperTestROMs creates ROMs for testing different mapper configurations
func (g *TestROMGenerator) CreateMapperTestROMs() map[string][]byte {
	roms := make(map[string][]byte)

	// Mapper 0 variants
	roms["nrom_16k_chr_rom"] = g.CreateValidROMData(1, 1, 0, 0x00, 0x00)
	roms["nrom_32k_chr_rom"] = g.CreateValidROMData(2, 1, 0, 0x00, 0x00)
	roms["nrom_16k_chr_ram"] = g.CreateValidROMData(1, 0, 0, 0x00, 0x00)
	roms["nrom_32k_chr_ram"] = g.CreateValidROMData(2, 0, 0, 0x00, 0x00)

	// Different mirroring modes
	roms["horizontal_mirror"] = g.CreateValidROMData(1, 1, 0, 0x00, 0x00)
	roms["vertical_mirror"] = g.CreateValidROMData(1, 1, 0, 0x01, 0x00)
	roms["four_screen_mirror"] = g.CreateValidROMData(1, 1, 0, 0x08, 0x00)

	// Battery-backed RAM
	roms["battery_backup"] = g.CreateValidROMData(1, 1, 0, 0x02, 0x00)
	roms["battery_vertical"] = g.CreateValidROMData(1, 1, 0, 0x03, 0x00)

	// Different mapper IDs (will default to mapper 0 in implementation)
	roms["mapper_1"] = g.CreateValidROMData(1, 1, 1, 0x00, 0x00)
	roms["mapper_4"] = g.CreateValidROMData(1, 1, 4, 0x00, 0x00)
	roms["mapper_255"] = g.CreateValidROMData(1, 1, 255, 0x00, 0x00)

	return roms
}

// CreateEdgeCaseROMs creates ROMs for testing edge cases
func (g *TestROMGenerator) CreateEdgeCaseROMs() map[string][]byte {
	roms := make(map[string][]byte)

	// Size edge cases
	roms["minimal_16k"] = g.CreateValidROMData(1, 1, 0, 0x00, 0x00)
	roms["large_512k_prg"] = g.CreateValidROMData(32, 1, 0, 0x00, 0x00)
	roms["large_256k_chr"] = g.CreateValidROMData(1, 32, 0, 0x00, 0x00)
	roms["maximum_size"] = g.CreateValidROMData(255, 255, 0, 0x00, 0x00)

	// Corruption cases
	roms["invalid_magic"] = g.CreateCorruptedHeader("invalid_magic")
	roms["truncated_header"] = g.CreateCorruptedHeader("truncated_header")
	roms["zero_prg_size"] = g.CreateCorruptedHeader("zero_prg")

	// Incomplete data cases
	roms["partial_prg"] = g.CreateIncompleteROM("partial_prg")
	roms["missing_chr"] = g.CreateIncompleteROM("missing_chr")
	roms["partial_chr"] = g.CreateIncompleteROM("partial_chr")

	// Special cases
	roms["with_trainer"] = g.CreateROMWithTrainer(1, 1)
	roms["trainer_no_chr"] = g.CreateROMWithTrainer(1, 0)

	return roms
}

// Test data integration tests
func TestTestROMGenerator_CreateValidROMData_ShouldProduceLoadableROM(t *testing.T) {
	generator := &TestROMGenerator{}

	romData := generator.CreateValidROMData(1, 1, 0, 0x00, 0x00)
	reader := bytes.NewReader(romData)

	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Generated ROM should be loadable, got error: %v", err)
	}
	if cartridge == nil {
		t.Fatal("Expected cartridge, got nil")
	}
	if len(cartridge.prgROM) != 16384 {
		t.Errorf("Expected 16KB PRG ROM, got %d bytes", len(cartridge.prgROM))
	}
	if len(cartridge.chrROM) != 8192 {
		t.Errorf("Expected 8KB CHR ROM, got %d bytes", len(cartridge.chrROM))
	}
}

func TestTestROMGenerator_CreateROMWithTrainer_ShouldSkipTrainerCorrectly(t *testing.T) {
	generator := &TestROMGenerator{}

	romData := generator.CreateROMWithTrainer(1, 1)
	reader := bytes.NewReader(romData)

	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("ROM with trainer should be loadable, got error: %v", err)
	}

	// Verify PRG ROM starts with expected pattern (not trainer pattern 0xAA)
	if cartridge.prgROM[0] != 0 {
		t.Errorf("Expected PRG ROM to start with 0, got 0x%02X (trainer may not have been skipped)", cartridge.prgROM[0])
	}
}

func TestTestROMGenerator_CreateCorruptedHeader_ShouldFailAppropriately(t *testing.T) {
	generator := &TestROMGenerator{}

	testCases := []struct {
		corruptionType string
		shouldFail     bool
		errorContains  string
	}{
		{"invalid_magic", true, "invalid iNES file"},
		{"truncated_header", true, ""},
		{"zero_prg", true, ""},
	}

	for _, tc := range testCases {
		t.Run(tc.corruptionType, func(t *testing.T) {
			romData := generator.CreateCorruptedHeader(tc.corruptionType)
			reader := bytes.NewReader(romData)

			cartridge, err := LoadFromReader(reader)

			if tc.shouldFail {
				if err == nil {
					t.Errorf("Expected error for %s, got success", tc.corruptionType)
				}
				if cartridge != nil {
					t.Error("Expected nil cartridge for corrupted ROM")
				}
			} else {
				if err != nil {
					t.Errorf("Expected success for %s, got error: %v", tc.corruptionType, err)
				}
			}
		})
	}
}

func TestTestROMGenerator_CreateMapperTestROMs_ShouldCoverAllVariants(t *testing.T) {
	generator := &TestROMGenerator{}

	roms := generator.CreateMapperTestROMs()

	expectedROMs := []string{
		"nrom_16k_chr_rom", "nrom_32k_chr_rom", "nrom_16k_chr_ram", "nrom_32k_chr_ram",
		"horizontal_mirror", "vertical_mirror", "four_screen_mirror",
		"battery_backup", "battery_vertical",
		"mapper_1", "mapper_4", "mapper_255",
	}

	for _, romName := range expectedROMs {
		if _, exists := roms[romName]; !exists {
			t.Errorf("Expected ROM variant %s not found", romName)
		}
	}

	// Test a few specific variants
	reader := bytes.NewReader(roms["vertical_mirror"])
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Vertical mirror ROM should load: %v", err)
	}
	if cartridge.mirror != MirrorVertical {
		t.Error("Expected vertical mirroring")
	}

	reader = bytes.NewReader(roms["battery_backup"])
	cartridge, err = LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Battery backup ROM should load: %v", err)
	}
	if !cartridge.hasBattery {
		t.Error("Expected battery backup flag to be set")
	}
}

func TestTestROMGenerator_CreateEdgeCaseROMs_ShouldHandleExtremes(t *testing.T) {
	generator := &TestROMGenerator{}

	roms := generator.CreateEdgeCaseROMs()

	// Test minimal ROM loads successfully
	reader := bytes.NewReader(roms["minimal_16k"])
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Minimal ROM should load: %v", err)
	}
	if len(cartridge.prgROM) != 16384 {
		t.Error("Minimal ROM should have 16KB PRG")
	}

	// Test large ROM loads successfully
	reader = bytes.NewReader(roms["large_512k_prg"])
	cartridge, err = LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Large PRG ROM should load: %v", err)
	}
	if len(cartridge.prgROM) != 32*16384 {
		t.Error("Large ROM should have correct PRG size")
	}

	// Test corrupted ROMs fail appropriately
	corruptedROMs := []string{"invalid_magic", "truncated_header", "partial_prg"}

	for _, romName := range corruptedROMs {
		reader := bytes.NewReader(roms[romName])
		cartridge, err := LoadFromReader(reader)

		if err == nil {
			t.Errorf("Corrupted ROM %s should fail to load", romName)
		}
		if cartridge != nil {
			t.Errorf("Corrupted ROM %s should return nil cartridge", romName)
		}
	}
}

// Helper function to save test ROM data to temporary files for file-based testing
func createTestROMFiles(t *testing.T, roms map[string][]byte) map[string]string {
	tmpDir := t.TempDir()
	filePaths := make(map[string]string)

	for name, data := range roms {
		filename := filepath.Join(tmpDir, name+".nes")
		err := os.WriteFile(filename, data, 0644)
		if err != nil {
			t.Fatalf("Failed to create test ROM file %s: %v", name, err)
		}
		filePaths[name] = filename
	}

	return filePaths
}

func TestCreateTestROMFiles_ShouldCreateValidFiles(t *testing.T) {
	generator := &TestROMGenerator{}
	roms := generator.CreateMapperTestROMs()

	// Create temporary files
	filePaths := createTestROMFiles(t, roms)

	// Test loading from files
	for name, path := range filePaths {
		cartridge, err := LoadFromFile(path)

		// Skip expected failures
		if name == "mapper_255" || name == "mapper_1" || name == "mapper_4" {
			// These should still load (default to mapper 0)
			if err != nil {
				t.Errorf("ROM file %s should load with mapper fallback: %v", name, err)
			}
			continue
		}

		if err != nil {
			t.Errorf("Failed to load ROM file %s: %v", name, err)
			continue
		}

		if cartridge == nil {
			t.Errorf("ROM file %s returned nil cartridge", name)
		}
	}
}

// Performance test for ROM generation
func BenchmarkTestROMGenerator_CreateValidROM(b *testing.B) {
	generator := &TestROMGenerator{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.CreateValidROMData(1, 1, 0, 0x00, 0x00)
	}
}

func BenchmarkTestROMGenerator_CreateLargeROM(b *testing.B) {
	generator := &TestROMGenerator{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		generator.CreateValidROMData(32, 32, 0, 0x00, 0x00) // 512KB PRG + 256KB CHR
	}
}

// Utility function to verify ROM data integrity
func verifyROMDataIntegrity(t *testing.T, romData []byte, expectedPRGSize, expectedCHRSize int) {
	if len(romData) < 16 {
		t.Fatal("ROM data too short for header")
	}

	// Check magic
	if string(romData[0:4]) != "NES\x1A" {
		t.Error("Invalid magic in generated ROM")
	}

	// Calculate expected total size
	expectedSize := 16 + expectedPRGSize + expectedCHRSize
	if len(romData) != expectedSize {
		t.Errorf("Expected ROM size %d, got %d", expectedSize, len(romData))
	}

	// Verify PRG size in header
	prgSizeInHeader := int(romData[4]) * 16384
	if prgSizeInHeader != expectedPRGSize {
		t.Errorf("PRG size mismatch: header says %d, expected %d", prgSizeInHeader, expectedPRGSize)
	}

	// Verify CHR size in header
	chrSizeInHeader := int(romData[5]) * 8192
	if chrSizeInHeader != expectedCHRSize {
		t.Errorf("CHR size mismatch: header says %d, expected %d", chrSizeInHeader, expectedCHRSize)
	}
}

func TestVerifyROMDataIntegrity_ShouldValidateCorrectly(t *testing.T) {
	generator := &TestROMGenerator{}

	// Test various sizes
	testCases := []struct {
		prgSize uint8
		chrSize uint8
	}{
		{1, 1}, // 16KB + 8KB
		{2, 1}, // 32KB + 8KB
		{1, 0}, // 16KB + CHR RAM
		{4, 4}, // 64KB + 32KB
	}

	for _, tc := range testCases {
		romData := generator.CreateValidROMData(tc.prgSize, tc.chrSize, 0, 0x00, 0x00)
		expectedPRG := int(tc.prgSize) * 16384
		expectedCHR := int(tc.chrSize) * 8192

		verifyROMDataIntegrity(t, romData, expectedPRG, expectedCHR)
	}
}
