package cartridge

import (
	"bytes"
	"io"
	"strings"
	"testing"
)

// errorReader simulates read errors for testing
type errorReader struct {
	data    []byte
	pos     int
	failAt  int
	errType string
}

func (r *errorReader) Read(p []byte) (n int, err error) {
	if r.pos >= r.failAt {
		switch r.errType {
		case "unexpected_eof":
			return 0, io.ErrUnexpectedEOF
		case "generic_error":
			return 0, io.ErrNoProgress
		default:
			return 0, io.EOF
		}
	}

	remaining := len(r.data) - r.pos
	if remaining == 0 {
		return 0, io.EOF
	}

	toCopy := len(p)
	if toCopy > remaining {
		toCopy = remaining
	}
	if r.pos+toCopy > r.failAt {
		toCopy = r.failAt - r.pos
	}

	copy(p, r.data[r.pos:r.pos+toCopy])
	r.pos += toCopy
	return toCopy, nil
}

func TestLoadFromReader_ReadErrorDuringHeader_ShouldFail(t *testing.T) {
	// Create a reader that fails while reading header
	data := []byte("NES\x1A\x01\x01") // Partial header
	reader := &errorReader{
		data:    data,
		failAt:  6,
		errType: "unexpected_eof",
	}

	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for header read failure, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for read error")
	}
}

func TestLoadFromReader_ReadErrorDuringTrainer_ShouldFail(t *testing.T) {
	// Create valid header with trainer flag
	header := createValidINESHeader(1, 1, 0, 0x04, 0) // Trainer flag set
	partialTrainer := make([]byte, 256)               // Only half trainer data

	data := append(header, partialTrainer...)
	reader := &errorReader{
		data:    data,
		failAt:  len(header) + 256,
		errType: "unexpected_eof",
	}

	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for trainer read failure, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for trainer read error")
	}
}

func TestLoadFromReader_ReadErrorDuringPRG_ShouldFail(t *testing.T) {
	header := createValidINESHeader(2, 1, 0, 0, 0) // 32KB PRG, 8KB CHR
	partialPRG := make([]byte, 16384)              // Only half PRG data

	data := append(header, partialPRG...)
	reader := &errorReader{
		data:    data,
		failAt:  len(header) + 16384,
		errType: "unexpected_eof",
	}

	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for PRG read failure, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for PRG read error")
	}
}

func TestLoadFromReader_ReadErrorDuringCHR_ShouldFail(t *testing.T) {
	header := createValidINESHeader(1, 1, 0, 0, 0) // 16KB PRG, 8KB CHR
	prgData := make([]byte, 16384)
	partialCHR := make([]byte, 4096) // Only half CHR data

	data := append(header, prgData...)
	data = append(data, partialCHR...)

	reader := &errorReader{
		data:    data,
		failAt:  len(header) + len(prgData) + 4096,
		errType: "unexpected_eof",
	}

	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for CHR read failure, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for CHR read error")
	}
}

func TestLoadFromReader_EmptyReader_ShouldFail(t *testing.T) {
	reader := bytes.NewReader([]byte{})

	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for empty reader, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for empty reader")
	}
}

func TestLoadFromReader_OnlyMagicBytes_ShouldFail(t *testing.T) {
	reader := bytes.NewReader([]byte("NES\x1A"))

	cartridge, err := LoadFromReader(reader)

	if err == nil {
		t.Fatal("Expected error for incomplete header, got success")
	}
	if cartridge != nil {
		t.Fatal("Expected nil cartridge for incomplete header")
	}
}

func TestLoadFromReader_InvalidFlagsCombination_ShouldHandleGracefully(t *testing.T) {
	// Test unusual but technically valid flag combinations
	testCases := []struct {
		name   string
		flags6 uint8
		flags7 uint8
	}{
		{"All flags set", 0xFF, 0xFF},
		{"High bits in flags6", 0xF0, 0x00},
		{"High bits in flags7", 0x00, 0xFF},
		{"Reserved bits set", 0x20, 0x08}, // Unused bits set
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			header := createValidINESHeader(1, 1, 0, tc.flags6, tc.flags7)
			prgData := make([]byte, 16384)
			chrData := make([]byte, 8192)
			romData := append(header, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			// Should still load successfully despite unusual flags
			if err != nil {
				t.Errorf("ROM with flags6=0x%02X, flags7=0x%02X should load: %v", tc.flags6, tc.flags7, err)
			}
			if cartridge == nil {
				t.Error("Expected cartridge despite unusual flags")
			}
		})
	}
}

func TestLoadFromReader_LargeROMSizes_ShouldHandleCorrectly(t *testing.T) {
	// Test edge cases for ROM sizes
	testCases := []struct {
		name    string
		prgSize uint8
		chrSize uint8
		valid   bool
	}{
		{"Single bank minimum", 1, 1, true},
		{"Large PRG", 64, 1, true},
		{"Large CHR", 1, 64, true},
		{"Both large", 32, 32, true},
		{"Maximum theoretical", 255, 255, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Only create ROM if it would be reasonable size
			if int(tc.prgSize)*16384+int(tc.chrSize)*8192 > 16*1024*1024 { // Skip > 16MB
				t.Skip("Skipping very large ROM test to avoid memory issues")
			}

			header := createValidINESHeader(tc.prgSize, tc.chrSize, 0, 0, 0)
			prgData := make([]byte, int(tc.prgSize)*16384)
			chrData := make([]byte, int(tc.chrSize)*8192)
			romData := append(header, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if tc.valid {
				if err != nil {
					t.Errorf("Valid large ROM should load: %v", err)
				}
				if cartridge == nil {
					t.Error("Expected cartridge for valid large ROM")
				}
			}
		})
	}
}

func TestLoadFromReader_UnsupportedMappers_ShouldDefaultToMapper0(t *testing.T) {
	// Test various unsupported mapper IDs
	mapperIDs := []uint8{1, 2, 3, 4, 5, 10, 50, 100, 200, 255}

	for _, mapperID := range mapperIDs {
		t.Run(string(rune(mapperID+'0')), func(t *testing.T) {
			// Create ROM with specific mapper ID
			header := createValidINESHeader(1, 1, mapperID, 0, 0)
			prgData := make([]byte, 16384)
			chrData := make([]byte, 8192)
			romData := append(header, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Errorf("ROM with mapper %d should load (default to mapper 0): %v", mapperID, err)
			}
			if cartridge == nil {
				t.Error("Expected cartridge for unsupported mapper (should default)")
			}
			if cartridge != nil && cartridge.mapperID != mapperID {
				t.Errorf("Mapper ID should be preserved: expected %d, got %d", mapperID, cartridge.mapperID)
			}
		})
	}
}

func TestLoadFromReader_ConcurrentAccess_ShouldBeSafe(t *testing.T) {
	// Test concurrent loading of the same ROM data
	romData := createMinimalValidROM(1, 1)

	results := make(chan error, 10)

	// Launch multiple goroutines
	for i := 0; i < 10; i++ {
		go func() {
			reader := bytes.NewReader(romData)
			_, err := LoadFromReader(reader)
			results <- err
		}()
	}

	// Check all results
	for i := 0; i < 10; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent load %d failed: %v", i, err)
		}
	}
}

func TestCartridge_MemoryAccess_BoundaryConditions(t *testing.T) {
	romData := createMinimalValidROM(1, 1)
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load test ROM: %v", err)
	}

	// Test boundary addresses for PRG access
	boundaryTests := []struct {
		address uint16
		name    string
	}{
		{0x5FFF, "Just before SRAM"},
		{0x6000, "SRAM start"},
		{0x7FFF, "SRAM end"},
		{0x8000, "ROM start"},
		{0xFFFF, "ROM end"},
	}

	for _, test := range boundaryTests {
		// Read shouldn't crash
		value := cartridge.ReadPRG(test.address)
		_ = value // Use value to avoid unused variable warning

		// Write shouldn't crash
		cartridge.WritePRG(test.address, 0x42)
	}

	// Test CHR boundary conditions
	chrBoundaryTests := []uint16{0x0000, 0x1FFF, 0x2000, 0xFFFF}

	for _, address := range chrBoundaryTests {
		value := cartridge.ReadCHR(address)
		_ = value

		cartridge.WriteCHR(address, 0x55)
	}
}

func TestCartridge_StateConsistency_AfterOperations(t *testing.T) {
	romData := createMinimalValidROM(2, 0) // 32KB PRG, CHR RAM
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load test ROM: %v", err)
	}

	// Test SRAM persistence
	testPattern := []struct {
		address uint16
		value   uint8
	}{
		{0x6000, 0xAA},
		{0x6800, 0xBB},
		{0x7000, 0xCC},
		{0x7FFF, 0xDD},
	}

	// Write pattern
	for _, p := range testPattern {
		cartridge.WritePRG(p.address, p.value)
	}

	// Verify persistence across multiple reads
	for i := 0; i < 3; i++ {
		for _, p := range testPattern {
			value := cartridge.ReadPRG(p.address)
			if value != p.value {
				t.Errorf("SRAM consistency check %d failed at 0x%04X: expected 0x%02X, got 0x%02X",
					i, p.address, p.value, value)
			}
		}
	}

	// Test CHR RAM persistence
	chrTestPattern := []struct {
		address uint16
		value   uint8
	}{
		{0x0000, 0x11},
		{0x0800, 0x22},
		{0x1000, 0x33},
		{0x1FFF, 0x44},
	}

	for _, p := range chrTestPattern {
		cartridge.WriteCHR(p.address, p.value)
	}

	for i := 0; i < 3; i++ {
		for _, p := range chrTestPattern {
			value := cartridge.ReadCHR(p.address)
			if value != p.value {
				t.Errorf("CHR RAM consistency check %d failed at 0x%04X: expected 0x%02X, got 0x%02X",
					i, p.address, p.value, value)
			}
		}
	}
}

func TestCartridge_ROMIntegrity_ShouldRemainUnmodified(t *testing.T) {
	romData := createMinimalValidROM(1, 1) // CHR ROM
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load test ROM: %v", err)
	}

	// Read initial ROM values
	prgInitial := cartridge.ReadPRG(0x8000)
	chrInitial := cartridge.ReadCHR(0x0000)

	// Attempt to write to ROM areas
	cartridge.WritePRG(0x8000, ^prgInitial) // Write inverted value
	cartridge.WriteCHR(0x0000, ^chrInitial)

	// Verify ROM values unchanged
	prgAfter := cartridge.ReadPRG(0x8000)
	chrAfter := cartridge.ReadCHR(0x0000)

	if prgAfter != prgInitial {
		t.Errorf("PRG ROM was modified: initial=0x%02X, after=0x%02X", prgInitial, prgAfter)
	}
	if chrAfter != chrInitial {
		t.Errorf("CHR ROM was modified: initial=0x%02X, after=0x%02X", chrInitial, chrAfter)
	}
}

func TestLoadFromReader_MaliciousInput_ShouldHandleSafely(t *testing.T) {
	// Test various potentially malicious inputs
	maliciousInputs := []struct {
		name string
		data []byte
	}{
		{"Very long magic", bytes.Repeat([]byte("N"), 1000)},
		{"Null bytes", make([]byte, 100)},
		{"Random bytes", []byte{0xFF, 0xFE, 0xFD, 0xFC, 0xFB, 0xFA, 0xF9, 0xF8}},
		{"Alternating pattern", []byte{0xAA, 0x55, 0xAA, 0x55, 0xAA, 0x55}},
	}

	for _, input := range maliciousInputs {
		t.Run(input.name, func(t *testing.T) {
			reader := bytes.NewReader(input.data)
			cartridge, err := LoadFromReader(reader)

			// Should fail gracefully, not crash
			if err == nil && cartridge != nil {
				t.Errorf("Malicious input %s unexpectedly succeeded", input.name)
			}

			// If it did succeed, verify basic functionality
			if err == nil && cartridge != nil {
				cartridge.ReadPRG(0x8000)
				cartridge.WritePRG(0x6000, 0x42)
				cartridge.ReadCHR(0x0000)
				cartridge.WriteCHR(0x0000, 0x55)
			}
		})
	}
}

func TestCartridge_ErrorPropagation_ShouldPreserveContext(t *testing.T) {
	// Test that errors contain useful context information
	testCases := []struct {
		name          string
		data          []byte
		expectedError string
	}{
		{"Invalid magic", []byte("ROM\x1A\x01\x01\x00\x00"), "invalid iNES file"},
		{"Empty data", []byte{}, ""},
		{"Partial header", []byte("NES\x1A\x01"), ""},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reader := bytes.NewReader(tc.data)
			cartridge, err := LoadFromReader(reader)

			if err == nil {
				t.Errorf("Expected error for %s", tc.name)
				return
			}

			if cartridge != nil {
				t.Errorf("Expected nil cartridge for %s", tc.name)
			}

			if tc.expectedError != "" && !strings.Contains(err.Error(), tc.expectedError) {
				t.Errorf("Error for %s should contain '%s', got: %v",
					tc.name, tc.expectedError, err)
			}
		})
	}
}

// Test memory patterns that might reveal implementation details
func TestCartridge_MemoryPatterns_ShouldNotLeakInformation(t *testing.T) {
	romData := createMinimalValidROM(1, 0) // CHR RAM
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load test ROM: %v", err)
	}

	// CHR RAM should be initialized to zero
	for addr := uint16(0x0000); addr < 0x2000; addr += 256 {
		value := cartridge.ReadCHR(addr)
		if value != 0 {
			t.Errorf("CHR RAM at 0x%04X should be zero, got 0x%02X", addr, value)
		}
	}

	// SRAM should be initialized to zero
	for addr := uint16(0x6000); addr < 0x8000; addr += 256 {
		value := cartridge.ReadPRG(addr)
		if value != 0 {
			t.Errorf("SRAM at 0x%04X should be zero, got 0x%02X", addr, value)
		}
	}
}

func TestCartridge_ResourceCleanup_ShouldNotLeak(t *testing.T) {
	// Test creating and discarding many cartridges to check for leaks
	for i := 0; i < 100; i++ {
		romData := createMinimalValidROM(1, 1)
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM %d: %v", i, err)
		}

		// Use the cartridge briefly
		cartridge.ReadPRG(0x8000)
		cartridge.WritePRG(0x6000, uint8(i))

		// Let it go out of scope (should be garbage collected)
		_ = cartridge
	}
}

// Stress test with rapid operations
func TestCartridge_RapidOperations_ShouldRemainStable(t *testing.T) {
	romData := createMinimalValidROM(2, 0) // 32KB PRG, CHR RAM
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		t.Fatalf("Failed to load test ROM: %v", err)
	}

	// Perform many rapid operations
	for i := 0; i < 10000; i++ {
		addr := uint16(0x6000 + (i % 0x2000)) // Cycle through SRAM
		value := uint8(i % 256)

		cartridge.WritePRG(addr, value)
		readValue := cartridge.ReadPRG(addr)

		if readValue != value {
			t.Errorf("Rapid operation %d failed: wrote 0x%02X, read 0x%02X", i, value, readValue)
		}
	}
}

// Test for integer overflow conditions
func TestCartridge_IntegerOverflow_ShouldNotOccur(t *testing.T) {
	// Test with maximum valid sizes to check for overflow
	header := createValidINESHeader(255, 255, 0, 0, 0)

	// Don't actually create the full ROM (would be ~4GB)
	// Just test the header parsing logic
	reader := bytes.NewReader(header)

	_, err := LoadFromReader(reader)

	// Should fail due to insufficient data, but not due to overflow
	if err == nil {
		t.Error("Expected error for incomplete large ROM")
	}

	// The error should be about missing data, not about overflow
	if strings.Contains(err.Error(), "overflow") {
		t.Errorf("Unexpected overflow error: %v", err)
	}
}
