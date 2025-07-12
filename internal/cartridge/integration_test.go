package cartridge

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// Integration tests for complete ROM-to-cartridge-to-memory workflows

func TestIntegration_ROMLoadToMemoryAccess_CompleteWorkflow(t *testing.T) {
	// Create a test ROM with known data patterns
	generator := &TestROMGenerator{}
	romData := generator.CreateValidROMData(2, 1, 0, 0x00, 0x00) // 32KB PRG, 8KB CHR

	// Load ROM into cartridge
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Verify cartridge properties
	if cartridge.mapperID != 0 {
		t.Errorf("Expected mapper 0, got %d", cartridge.mapperID)
	}
	if cartridge.mirror != MirrorHorizontal {
		t.Errorf("Expected horizontal mirroring, got %d", cartridge.mirror)
	}
	if cartridge.hasBattery {
		t.Error("Expected no battery, but hasBattery is true")
	}

	// Test PRG ROM access across the full range
	testAddresses := []struct {
		address  uint16
		expected uint8
	}{
		{0x8000, 0x00}, // Start of first bank (offset 0)
		{0x8100, 0x01}, // Offset 0x100 / 256 = 1
		{0x9000, 0x04}, // Offset 0x1000 / 256 = 4
		{0xC000, 0x10}, // Start of second bank (offset 0x4000 / 256 = 16)
		{0xD000, 0x14}, // Offset 0x5000 / 256 = 20
	}

	for _, test := range testAddresses {
		value := cartridge.ReadPRG(test.address)
		if value != test.expected {
			t.Errorf("PRG address 0x%04X: expected 0x%02X, got 0x%02X",
				test.address, test.expected, value)
		}
	}

	// Test CHR ROM access
	chrTestAddresses := []struct {
		address  uint16
		expected uint8
	}{
		{0x0000, 0xFF}, // 255 - (0 % 256) = 255
		{0x0100, 0x9F}, // 255 - (256 % 256) = 255 - 0 = 255, but 255 - (0x100 % 256) = 255 - 0 = 255
		{0x0800, 0xF7}, // 255 - (0x800 % 256) = 255 - 0 = 255, wait: 0x800 % 256 = 0, so 255 - 0 = 255
		{0x1000, 0xEF}, // Let me recalculate: (0x1000 + 128) % 256 = (4096 + 128) % 256 = 4224 % 256 = 128, inverted = 255 - 128 = 127
	}

	// Recalculate expected values based on the generator logic
	for _, test := range chrTestAddresses {
		value := cartridge.ReadCHR(test.address)
		expectedValue := uint8(255 - (int(test.address) % 256)) // Matches generator pattern
		if value != expectedValue {
			t.Errorf("CHR address 0x%04X: expected 0x%02X, got 0x%02X",
				test.address, expectedValue, value)
		}
	}
}

func TestIntegration_MultipleROMFormats_ShouldLoadCorrectly(t *testing.T) {
	generator := &TestROMGenerator{}
	testROMs := generator.CreateMapperTestROMs()

	// Test each ROM format
	for romName, romData := range testROMs {
		t.Run(romName, func(t *testing.T) {
			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Fatalf("Failed to load ROM %s: %v", romName, err)
			}

			// Verify basic functionality
			switch romName {
			case "vertical_mirror":
				if cartridge.mirror != MirrorVertical {
					t.Error("Expected vertical mirroring")
				}
			case "four_screen_mirror":
				if cartridge.mirror != MirrorFourScreen {
					t.Error("Expected four-screen mirroring")
				}
			case "battery_backup":
				if !cartridge.hasBattery {
					t.Error("Expected battery backup")
				}
			case "nrom_16k_chr_ram":
				// Verify CHR RAM functionality
				cartridge.WriteCHR(0x0000, 0x42)
				value := cartridge.ReadCHR(0x0000)
				if value != 0x42 {
					t.Error("CHR RAM should be writable")
				}
			}

			// Test basic memory access
			cartridge.ReadPRG(0x8000)
			cartridge.WritePRG(0x6000, 0x55)
			sramValue := cartridge.ReadPRG(0x6000)
			if sramValue != 0x55 {
				t.Error("SRAM should be functional")
			}
		})
	}
}

func TestIntegration_FileToCartridgeToMemory_CompleteChain(t *testing.T) {
	// Create temporary ROM file
	generator := &TestROMGenerator{}
	romData := generator.CreateROMWithTrainer(1, 1)

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test_with_trainer.nes")

	err := os.WriteFile(filename, romData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Load from file
	cartridge, err := LoadFromFile(filename)
	if err != nil {
		t.Fatalf("Failed to load from file: %v", err)
	}

	// Verify trainer was skipped and PRG data is correct
	// The trainer pattern is 0xAA, PRG pattern starts at 0x00
	prgValue := cartridge.ReadPRG(0x8000)
	if prgValue != 0x00 {
		t.Errorf("Expected PRG to start with 0x00 (trainer skipped), got 0x%02X", prgValue)
	}

	// Verify full memory access chain
	testPattern := []struct {
		address uint16
		value   uint8
	}{
		{0x6000, 0xAA},
		{0x6500, 0xBB},
		{0x7000, 0xCC},
		{0x7FFF, 0xDD},
	}

	// Write through cartridge interface
	for _, p := range testPattern {
		cartridge.WritePRG(p.address, p.value)
	}

	// Read back through cartridge interface
	for _, p := range testPattern {
		value := cartridge.ReadPRG(p.address)
		if value != p.value {
			t.Errorf("Memory chain test failed at 0x%04X: expected 0x%02X, got 0x%02X",
				p.address, p.value, value)
		}
	}
}

func TestIntegration_ErrorHandlingChain_ShouldPropagateCorrectly(t *testing.T) {
	// Test error propagation from file to cartridge to memory

	// Test 1: File not found
	cartridge, err := LoadFromFile("/nonexistent/file.nes")
	if err == nil {
		t.Error("Expected error for nonexistent file")
	}
	if cartridge != nil {
		t.Error("Expected nil cartridge for file error")
	}

	// Test 2: Invalid file content
	tmpDir := t.TempDir()
	invalidFile := filepath.Join(tmpDir, "invalid.nes")

	err = os.WriteFile(invalidFile, []byte("invalid data"), 0644)
	if err != nil {
		t.Fatalf("Failed to create invalid file: %v", err)
	}

	cartridge, err = LoadFromFile(invalidFile)
	if err == nil {
		t.Error("Expected error for invalid file content")
	}
	if cartridge != nil {
		t.Error("Expected nil cartridge for invalid content")
	}

	// Test 3: Reader error propagation
	generator := &TestROMGenerator{}
	corruptedData := generator.CreateCorruptedHeader("partial_prg")

	reader := bytes.NewReader(corruptedData)
	cartridge, err = LoadFromReader(reader)
	if err == nil {
		t.Error("Expected error for corrupted ROM")
	}
	if cartridge != nil {
		t.Error("Expected nil cartridge for corrupted ROM")
	}
}

func TestIntegration_CartridgeToMapper_InterfaceCompliance(t *testing.T) {
	// Test that cartridge properly delegates to mapper
	generator := &TestROMGenerator{}
	romData := generator.CreateValidROMData(2, 0, 0, 0x00, 0x00) // 32KB PRG, CHR RAM

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Verify mapper was created and is accessible
	if cartridge.mapper == nil {
		t.Fatal("Mapper should be created")
	}

	// Test that cartridge methods delegate to mapper
	// This is tested indirectly through functional tests

	// Test PRG access delegation
	testValue := uint8(0x42)
	cartridge.WritePRG(0x6000, testValue)
	retrievedValue := cartridge.ReadPRG(0x6000)
	if retrievedValue != testValue {
		t.Errorf("PRG delegation failed: expected 0x%02X, got 0x%02X", testValue, retrievedValue)
	}

	// Test CHR access delegation
	cartridge.WriteCHR(0x0000, testValue)
	retrievedValue = cartridge.ReadCHR(0x0000)
	if retrievedValue != testValue {
		t.Errorf("CHR delegation failed: expected 0x%02X, got 0x%02X", testValue, retrievedValue)
	}

	// Test ROM access delegation
	romValue := cartridge.ReadPRG(0x8000)
	expectedROMValue := uint8(0x00) // Pattern from generator
	if romValue != expectedROMValue {
		t.Errorf("ROM delegation failed: expected 0x%02X, got 0x%02X", expectedROMValue, romValue)
	}
}

func TestIntegration_MemoryLayout_ShouldMatchNESSpecification(t *testing.T) {
	// Test that memory layout matches NES specifications
	generator := &TestROMGenerator{}
	romData := generator.CreateValidROMData(1, 1, 0, 0x00, 0x00) // 16KB PRG, 8KB CHR

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Test SRAM range (0x6000-0x7FFF)
	sramTests := []uint16{0x6000, 0x6001, 0x7FFE, 0x7FFF}
	for _, addr := range sramTests {
		// Should be writable
		cartridge.WritePRG(addr, 0x55)
		value := cartridge.ReadPRG(addr)
		if value != 0x55 {
			t.Errorf("SRAM at 0x%04X should be writable", addr)
		}
	}

	// Test ROM range (0x8000-0xFFFF)
	romTests := []uint16{0x8000, 0x8001, 0xFFFE, 0xFFFF}
	for _, addr := range romTests {
		// Should return data, not be writable
		initialValue := cartridge.ReadPRG(addr)
		cartridge.WritePRG(addr, ^initialValue) // Write inverted value
		afterWrite := cartridge.ReadPRG(addr)
		if afterWrite != initialValue {
			t.Errorf("ROM at 0x%04X should not be writable", addr)
		}
	}

	// Test CHR range (0x0000-0x1FFF)
	chrTests := []uint16{0x0000, 0x0001, 0x1FFE, 0x1FFF}
	for _, addr := range chrTests {
		// Should return data
		value := cartridge.ReadCHR(addr)
		_ = value // Just verify it doesn't crash
	}

	// Test invalid ranges
	invalidPRGTests := []uint16{0x0000, 0x4000, 0x5FFF}
	for _, addr := range invalidPRGTests {
		value := cartridge.ReadPRG(addr)
		if value != 0 {
			t.Errorf("Invalid PRG address 0x%04X should return 0, got 0x%02X", addr, value)
		}
	}

	invalidCHRTests := []uint16{0x2000, 0x4000, 0x8000}
	for _, addr := range invalidCHRTests {
		value := cartridge.ReadCHR(addr)
		if value != 0 {
			t.Errorf("Invalid CHR address 0x%04X should return 0, got 0x%02X", addr, value)
		}
	}
}

func TestIntegration_BankSwitching_16KBMirroring(t *testing.T) {
	// Test 16KB ROM mirroring behavior
	generator := &TestROMGenerator{}
	romData := generator.CreateValidROMData(1, 1, 0, 0x00, 0x00) // 16KB PRG

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Test mirroring pairs
	mirrorPairs := []struct {
		addr1 uint16
		addr2 uint16
	}{
		{0x8000, 0xC000},
		{0x8100, 0xC100},
		{0x9000, 0xD000},
		{0xBFFF, 0xFFFF},
	}

	for _, pair := range mirrorPairs {
		value1 := cartridge.ReadPRG(pair.addr1)
		value2 := cartridge.ReadPRG(pair.addr2)

		if value1 != value2 {
			t.Errorf("16KB mirroring failed: 0x%04X=0x%02X, 0x%04X=0x%02X",
				pair.addr1, value1, pair.addr2, value2)
		}
	}
}

func TestIntegration_BankSwitching_32KBNoMirroring(t *testing.T) {
	// Test 32KB ROM no mirroring behavior
	generator := &TestROMGenerator{}
	romData := generator.CreateValidROMData(2, 1, 0, 0x00, 0x00) // 32KB PRG

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Test different banks return different values
	bankTests := []struct {
		addr1 uint16
		addr2 uint16
	}{
		{0x8000, 0xC000}, // First bank vs second bank
		{0x8100, 0xC100}, // Offset within banks
		{0x9000, 0xD000}, // Different offsets
	}

	for _, pair := range bankTests {
		value1 := cartridge.ReadPRG(pair.addr1)
		value2 := cartridge.ReadPRG(pair.addr2)

		// Values should be different (no mirroring)
		if value1 == value2 {
			t.Errorf("32KB ROM should not mirror: 0x%04X=0x%02X, 0x%04X=0x%02X",
				pair.addr1, value1, pair.addr2, value2)
		}
	}
}

func TestIntegration_FullSystemSimulation_BasicOperations(t *testing.T) {
	// Simulate basic operations that a CPU would perform
	generator := &TestROMGenerator{}
	romData := generator.CreateValidROMData(2, 0, 0, 0x02, 0x00) // Battery-backed SRAM

	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load ROM: %v", err)
	}

	// Simulate CPU operations

	// 1. Reset vector read (typically at 0xFFFC-0xFFFD)
	resetLow := cartridge.ReadPRG(0xFFFC)
	resetHigh := cartridge.ReadPRG(0xFFFD)
	resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
	_ = resetVector // Use the reset vector

	// 2. Code execution simulation - read instruction bytes
	instructionAddrs := []uint16{0x8000, 0x8001, 0x8002, 0x8003, 0x8004}
	for _, addr := range instructionAddrs {
		instruction := cartridge.ReadPRG(addr)
		_ = instruction // Process instruction
	}

	// 3. SRAM operations (save game data)
	saveData := []struct {
		addr uint16
		data uint8
	}{
		{0x6000, 0x12}, // Player progress
		{0x6100, 0x34}, // Inventory
		{0x6200, 0x56}, // Settings
		{0x6300, 0x78}, // High score
	}

	// Write save data
	for _, save := range saveData {
		cartridge.WritePRG(save.addr, save.data)
	}

	// Read back save data (simulate game loading)
	for _, save := range saveData {
		value := cartridge.ReadPRG(save.addr)
		if value != save.data {
			t.Errorf("Save data corrupted at 0x%04X: expected 0x%02X, got 0x%02X",
				save.addr, save.data, value)
		}
	}

	// 4. CHR RAM operations (pattern table updates)
	patternData := []struct {
		addr uint16
		data uint8
	}{
		{0x0000, 0xFF}, // Sprite pattern
		{0x0010, 0x00}, // Background pattern
		{0x1000, 0xAA}, // Second pattern table
		{0x1010, 0x55}, // More patterns
	}

	for _, pattern := range patternData {
		cartridge.WriteCHR(pattern.addr, pattern.data)
		value := cartridge.ReadCHR(pattern.addr)
		if value != pattern.data {
			t.Errorf("CHR data corrupted at 0x%04X: expected 0x%02X, got 0x%02X",
				pattern.addr, pattern.data, value)
		}
	}
}

// Benchmark integration tests
func BenchmarkIntegration_ROMLoadAndAccess(b *testing.B) {
	generator := &TestROMGenerator{}
	romData := generator.CreateValidROMData(2, 1, 0, 0x00, 0x00)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)
		if err != nil {
			b.Fatalf("Failed to load ROM: %v", err)
		}

		// Simulate typical access pattern
		cartridge.ReadPRG(0x8000)        // Instruction fetch
		cartridge.WritePRG(0x6000, 0x42) // SRAM write
		cartridge.ReadPRG(0x6000)        // SRAM read
		cartridge.ReadCHR(0x0000)        // Pattern table read
	}
}

func BenchmarkIntegration_FileLoadAndAccess(b *testing.B) {
	generator := &TestROMGenerator{}
	romData := generator.CreateValidROMData(1, 1, 0, 0x00, 0x00)

	tmpDir := b.TempDir()
	filename := filepath.Join(tmpDir, "bench.nes")

	err := os.WriteFile(filename, romData, 0644)
	if err != nil {
		b.Fatalf("Failed to create test file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cartridge, err := LoadFromFile(filename)
		if err != nil {
			b.Fatalf("Failed to load ROM: %v", err)
		}

		// Basic access
		cartridge.ReadPRG(0x8000)
		cartridge.ReadCHR(0x0000)
	}
}
