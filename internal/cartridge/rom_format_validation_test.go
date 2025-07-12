package cartridge

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

// TestROMFormatValidation provides comprehensive ROM format validation tests
// This test suite validates iNES format parsing, NES 2.0 detection, and various ROM configurations

// createCorruptedHeader creates headers with specific corruption patterns
func createCorruptedHeader(corruptionType string) []byte {
	header := make([]byte, 16)
	copy(header[0:4], "NES\x1A")

	switch corruptionType {
	case "short_magic":
		return []byte("NES")
	case "wrong_magic":
		copy(header[0:4], "ROM\x1A")
	case "null_magic":
		copy(header[0:4], "\x00\x00\x00\x00")
	case "partial_magic":
		copy(header[0:4], "NE\x00\x1A")
	case "zero_prg":
		header[4] = 0 // Zero PRG ROM size
		header[5] = 1 // Valid CHR ROM size
	case "excessive_prg":
		header[4] = 255 // Maximum PRG ROM size
		header[5] = 255 // Maximum CHR ROM size
	case "invalid_flags":
		header[4] = 1
		header[5] = 1
		header[6] = 0xFF // All flags set
		header[7] = 0xFF // All flags set
	case "mixed_format":
		header[4] = 1
		header[5] = 1
		header[7] = 0x08 // NES 2.0 identifier without proper format
	default:
		// Standard valid header
		header[4] = 1
		header[5] = 1
	}

	return header
}

// TestROMFormatValidation_iNESHeaderParsing tests comprehensive iNES header parsing
func TestROMFormatValidation_iNESHeaderParsing(t *testing.T) {
	tests := []struct {
		name           string
		headerData     []byte
		expectError    bool
		errorSubstring string
		validateFunc   func(*testing.T, *Cartridge)
	}{
		{
			name:        "Valid minimal iNES header",
			headerData:  createValidINESHeader(1, 1, 0, 0, 0),
			expectError: false,
			validateFunc: func(t *testing.T, cart *Cartridge) {
				if cart.mapperID != 0 {
					t.Errorf("Expected mapper 0, got %d", cart.mapperID)
				}
				if len(cart.prgROM) != 16384 {
					t.Errorf("Expected 16KB PRG ROM, got %d bytes", len(cart.prgROM))
				}
			},
		},
		{
			name:           "Corrupted magic number",
			headerData:     createCorruptedHeader("wrong_magic"),
			expectError:    true,
			errorSubstring: "invalid iNES file",
		},
		{
			name:           "Short magic number",
			headerData:     createCorruptedHeader("short_magic"),
			expectError:    true,
			errorSubstring: "",
		},
		{
			name:           "Null magic number",
			headerData:     createCorruptedHeader("null_magic"),
			expectError:    true,
			errorSubstring: "invalid iNES file",
		},
		{
			name:           "Partial magic number",
			headerData:     createCorruptedHeader("partial_magic"),
			expectError:    true,
			errorSubstring: "invalid iNES file",
		},
		{
			name:           "Zero PRG ROM size",
			headerData:     createCorruptedHeader("zero_prg"),
			expectError:    true,
			errorSubstring: "PRG ROM size cannot be zero",
		},
		{
			name:           "Maximum ROM sizes",
			headerData:     createCorruptedHeader("excessive_prg"),
			expectError:    true, // Will fail due to insufficient data, not size validation
			errorSubstring: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create complete ROM data
			var romData []byte
			if len(tt.headerData) >= 16 {
				romData = append(romData, tt.headerData...)

				// Add PRG ROM data if header indicates non-zero size
				if len(tt.headerData) >= 5 && tt.headerData[4] > 0 {
					prgSize := int(tt.headerData[4]) * 16384
					// Limit size for test to avoid memory issues
					if prgSize > 1024*1024 {
						prgSize = 32768      // Use 32KB for oversized tests
						tt.headerData[4] = 2 // Adjust header
					}
					prgData := make([]byte, prgSize)
					romData = append(romData, prgData...)
				}

				// Add CHR ROM data if header indicates non-zero size
				if len(tt.headerData) >= 6 && tt.headerData[5] > 0 {
					chrSize := int(tt.headerData[5]) * 8192
					// Limit size for test to avoid memory issues
					if chrSize > 512*1024 {
						chrSize = 16384      // Use 16KB for oversized tests
						tt.headerData[5] = 2 // Adjust header
					}
					chrData := make([]byte, chrSize)
					romData = append(romData, chrData...)
				}
			} else {
				romData = tt.headerData
			}

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, but got success", tt.name)
				} else if tt.errorSubstring != "" && !strings.Contains(err.Error(), tt.errorSubstring) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorSubstring, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected success for %s, got error: %v", tt.name, err)
				}
				if cartridge == nil {
					t.Errorf("Expected cartridge for %s, got nil", tt.name)
				}
				if tt.validateFunc != nil {
					tt.validateFunc(t, cartridge)
				}
			}
		})
	}
}

// TestROMFormatValidation_NES20Detection tests NES 2.0 format detection
func TestROMFormatValidation_NES20Detection(t *testing.T) {
	tests := []struct {
		name        string
		flags7      uint8
		description string
		isNES20     bool
	}{
		{
			name:        "Standard iNES format",
			flags7:      0x00,
			description: "Standard iNES with no NES 2.0 identifier",
			isNES20:     false,
		},
		{
			name:        "NES 2.0 format identifier",
			flags7:      0x08,
			description: "NES 2.0 format with proper identifier bits",
			isNES20:     true,
		},
		{
			name:        "Mixed format flags",
			flags7:      0x0C,
			description: "Mixed format flags that might indicate NES 2.0",
			isNES20:     true,
		},
		{
			name:        "Legacy format with high bits",
			flags7:      0x04,
			description: "Legacy format with some high bits set",
			isNES20:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createValidINESHeader(1, 1, 0, 0, tt.flags7)
			prgData := make([]byte, 16384)
			chrData := make([]byte, 8192)
			romData := append(header, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Fatalf("Failed to load ROM: %v", err)
			}

			// Note: Current implementation doesn't expose NES 2.0 detection
			// This test validates the ROM loads successfully despite format variations
			if cartridge == nil {
				t.Error("Expected cartridge to load successfully")
			}

			t.Logf("Successfully loaded %s ROM format", tt.description)
		})
	}
}

// TestROMFormatValidation_TrainerDataHandling tests trainer data scenarios
func TestROMFormatValidation_TrainerDataHandling(t *testing.T) {
	tests := []struct {
		name        string
		hasTrainer  bool
		trainerSize int
		expectError bool
		description string
	}{
		{
			name:        "No trainer data",
			hasTrainer:  false,
			trainerSize: 0,
			expectError: false,
			description: "Standard ROM without trainer",
		},
		{
			name:        "Valid trainer data",
			hasTrainer:  true,
			trainerSize: 512,
			expectError: false,
			description: "ROM with standard 512-byte trainer",
		},
		{
			name:        "Incomplete trainer data",
			hasTrainer:  true,
			trainerSize: 256,
			expectError: true,
			description: "ROM with incomplete trainer data",
		},
		{
			name:        "Oversized trainer data",
			hasTrainer:  true,
			trainerSize: 1024,
			expectError: false,
			description: "ROM with oversized trainer (should read only 512 bytes)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create header with trainer flag
			flags6 := uint8(0)
			if tt.hasTrainer {
				flags6 |= 0x04 // Set trainer flag
			}

			header := createValidINESHeader(1, 1, 0, flags6, 0)
			romData := append([]byte{}, header...)

			// Add trainer data if specified
			if tt.hasTrainer {
				trainerData := make([]byte, tt.trainerSize)
				for i := range trainerData {
					trainerData[i] = 0xFF // Fill with recognizable pattern
				}
				romData = append(romData, trainerData...)
			}

			// Add PRG and CHR ROM data
			prgData := make([]byte, 16384)
			for i := range prgData {
				prgData[i] = uint8(i % 256) // Recognizable pattern
			}
			chrData := make([]byte, 8192)

			romData = append(romData, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got success", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success for %s, got error: %v", tt.description, err)
				}

				// Verify PRG ROM data starts with expected pattern (not trainer data)
				if cartridge != nil {
					prgValue := cartridge.ReadPRG(0x8000)
					expectedValue := uint8(0) // First byte of PRG pattern
					if prgValue != expectedValue {
						t.Errorf("PRG ROM contaminated by trainer: expected 0x%02X, got 0x%02X",
							expectedValue, prgValue)
					}
				}
			}
		})
	}
}

// TestROMFormatValidation_HeaderFlags tests comprehensive flag interpretation
func TestROMFormatValidation_HeaderFlags(t *testing.T) {
	tests := []struct {
		name            string
		flags6          uint8
		flags7          uint8
		expectedMapper  uint8
		expectedMirror  MirrorMode
		expectedBattery bool
		description     string
	}{
		{
			name:            "NROM horizontal mirroring",
			flags6:          0x00,
			flags7:          0x00,
			expectedMapper:  0,
			expectedMirror:  MirrorHorizontal,
			expectedBattery: false,
			description:     "Standard NROM with horizontal mirroring",
		},
		{
			name:            "NROM vertical mirroring",
			flags6:          0x01,
			flags7:          0x00,
			expectedMapper:  0,
			expectedMirror:  MirrorVertical,
			expectedBattery: false,
			description:     "NROM with vertical mirroring",
		},
		{
			name:            "NROM with battery",
			flags6:          0x02,
			flags7:          0x00,
			expectedMapper:  0,
			expectedMirror:  MirrorHorizontal,
			expectedBattery: true,
			description:     "NROM with battery-backed SRAM",
		},
		{
			name:            "NROM four-screen mirroring",
			flags6:          0x08,
			flags7:          0x00,
			expectedMapper:  0,
			expectedMirror:  MirrorFourScreen,
			expectedBattery: false,
			description:     "NROM with four-screen mirroring",
		},
		{
			name:            "Four-screen overrides vertical",
			flags6:          0x09, // Four-screen + vertical
			flags7:          0x00,
			expectedMapper:  0,
			expectedMirror:  MirrorFourScreen,
			expectedBattery: false,
			description:     "Four-screen mirroring overrides vertical flag",
		},
		{
			name:            "Mapper 1 MMC1",
			flags6:          0x10, // Mapper 1 lower nibble
			flags7:          0x00,
			expectedMapper:  1,
			expectedMirror:  MirrorHorizontal,
			expectedBattery: false,
			description:     "MMC1 mapper identification",
		},
		{
			name:            "High mapper number",
			flags6:          0xF0, // Mapper 15 lower nibble
			flags7:          0xF0, // Mapper 15 upper nibble (total 255)
			expectedMapper:  255,
			expectedMirror:  MirrorHorizontal,
			expectedBattery: false,
			description:     "High mapper number from combined flags",
		},
		{
			name:            "Complex flag combination",
			flags6:          0x17, // Mapper 1 + vertical + battery + trainer
			flags7:          0x00,
			expectedMapper:  1,
			expectedMirror:  MirrorVertical,
			expectedBattery: true,
			description:     "Complex combination of flags",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := createValidINESHeader(1, 1, 0, tt.flags6, tt.flags7)

			// Add trainer data if trainer flag is set
			romData := append([]byte{}, header...)
			if (tt.flags6 & 0x04) != 0 {
				trainerData := make([]byte, 512)
				romData = append(romData, trainerData...)
			}

			// Add PRG and CHR ROM data
			prgData := make([]byte, 16384)
			chrData := make([]byte, 8192)
			romData = append(romData, prgData...)
			romData = append(romData, chrData...)

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Fatalf("Failed to load ROM for %s: %v", tt.description, err)
			}

			// Validate mapper ID
			if cartridge.mapperID != tt.expectedMapper {
				t.Errorf("Mapper ID mismatch for %s: expected %d, got %d",
					tt.description, tt.expectedMapper, cartridge.mapperID)
			}

			// Validate mirroring mode
			if cartridge.mirror != tt.expectedMirror {
				t.Errorf("Mirror mode mismatch for %s: expected %d, got %d",
					tt.description, tt.expectedMirror, cartridge.mirror)
			}

			// Validate battery flag
			if cartridge.hasBattery != tt.expectedBattery {
				t.Errorf("Battery flag mismatch for %s: expected %v, got %v",
					tt.description, tt.expectedBattery, cartridge.hasBattery)
			}
		})
	}
}

// TestROMFormatValidation_ROMSizeVariations tests various ROM size configurations
func TestROMFormatValidation_ROMSizeVariations(t *testing.T) {
	tests := []struct {
		name          string
		prgSize       uint8
		chrSize       uint8
		expectError   bool
		validateFunc  func(*testing.T, *Cartridge)
		skipLargeTest bool
	}{
		{
			name:        "Minimum configuration",
			prgSize:     1,
			chrSize:     0, // CHR RAM
			expectError: false,
			validateFunc: func(t *testing.T, cart *Cartridge) {
				if len(cart.prgROM) != 16384 {
					t.Errorf("Expected 16KB PRG ROM, got %d bytes", len(cart.prgROM))
				}
				if len(cart.chrROM) != 8192 {
					t.Errorf("Expected 8KB CHR RAM, got %d bytes", len(cart.chrROM))
				}
				if !cart.hasCHRRAM {
					t.Error("Expected CHR RAM flag to be set")
				}
			},
		},
		{
			name:        "Standard configuration",
			prgSize:     2,
			chrSize:     1,
			expectError: false,
			validateFunc: func(t *testing.T, cart *Cartridge) {
				if len(cart.prgROM) != 32768 {
					t.Errorf("Expected 32KB PRG ROM, got %d bytes", len(cart.prgROM))
				}
				if len(cart.chrROM) != 8192 {
					t.Errorf("Expected 8KB CHR ROM, got %d bytes", len(cart.chrROM))
				}
				if cart.hasCHRRAM {
					t.Error("Expected CHR RAM flag to be clear")
				}
			},
		},
		{
			name:        "Large CHR ROM",
			prgSize:     1,
			chrSize:     4, // 32KB CHR ROM
			expectError: false,
			validateFunc: func(t *testing.T, cart *Cartridge) {
				if len(cart.chrROM) != 32768 {
					t.Errorf("Expected 32KB CHR ROM, got %d bytes", len(cart.chrROM))
				}
			},
		},
		{
			name:          "Very large ROM",
			prgSize:       64, // 1MB PRG ROM
			chrSize:       32, // 256KB CHR ROM
			expectError:   false,
			skipLargeTest: true, // Skip if memory constrained
		},
		{
			name:          "Maximum theoretical sizes",
			prgSize:       255,
			chrSize:       255,
			expectError:   false, // Should succeed if we have enough memory
			skipLargeTest: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip large tests if they would consume too much memory
			if tt.skipLargeTest {
				totalSize := int(tt.prgSize)*16384 + int(tt.chrSize)*8192
				if totalSize > 16*1024*1024 { // Skip if > 16MB
					t.Skip("Skipping large ROM test to avoid memory issues")
				}
			}

			header := createValidINESHeader(tt.prgSize, tt.chrSize, 0, 0, 0)

			// Create PRG ROM data
			prgData := make([]byte, int(tt.prgSize)*16384)
			for i := range prgData {
				prgData[i] = uint8((i >> 8) & 0xFF) // Pattern for verification
			}

			// Create CHR ROM data if specified
			var chrData []byte
			if tt.chrSize > 0 {
				chrData = make([]byte, int(tt.chrSize)*8192)
				for i := range chrData {
					chrData[i] = uint8((i + 0x80) & 0xFF) // Different pattern
				}
			}

			romData := append(header, prgData...)
			if len(chrData) > 0 {
				romData = append(romData, chrData...)
			}

			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got success", tt.name)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success for %s, got error: %v", tt.name, err)
				}
				if cartridge != nil && tt.validateFunc != nil {
					tt.validateFunc(t, cartridge)
				}
			}
		})
	}
}

// TestROMFormatValidation_CorruptedROMData tests handling of corrupted ROM data
func TestROMFormatValidation_CorruptedROMData(t *testing.T) {
	tests := []struct {
		name           string
		corruptionFunc func([]byte) []byte
		expectError    bool
		description    string
	}{
		{
			name: "Truncated PRG ROM",
			corruptionFunc: func(data []byte) []byte {
				if len(data) > 16 {
					return data[:len(data)/2] // Cut ROM data in half
				}
				return data
			},
			expectError: true,
			description: "ROM file with incomplete PRG ROM data",
		},
		{
			name: "Truncated CHR ROM",
			corruptionFunc: func(data []byte) []byte {
				if len(data) > 16+16384 {
					// Keep header and PRG ROM, truncate CHR ROM
					return data[:16+16384+4096] // Only half CHR ROM
				}
				return data
			},
			expectError: true,
			description: "ROM file with incomplete CHR ROM data",
		},
		{
			name: "Extra data at end",
			corruptionFunc: func(data []byte) []byte {
				extraData := make([]byte, 1024)
				for i := range extraData {
					extraData[i] = 0xFF
				}
				return append(data, extraData...)
			},
			expectError: false,
			description: "ROM file with extra data at end (should be ignored)",
		},
		{
			name: "Null bytes in ROM",
			corruptionFunc: func(data []byte) []byte {
				// Zero out middle portion of ROM data
				if len(data) > 1000 {
					for i := 500; i < 1000; i++ {
						data[i] = 0
					}
				}
				return data
			},
			expectError: false,
			description: "ROM with null bytes in data (valid but unusual)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create base ROM data
			header := createValidINESHeader(1, 1, 0, 0, 0)
			prgData := make([]byte, 16384)
			chrData := make([]byte, 8192)
			for i := range prgData {
				prgData[i] = uint8(i % 256)
			}
			for i := range chrData {
				chrData[i] = uint8((i + 128) % 256)
			}

			romData := append(header, prgData...)
			romData = append(romData, chrData...)

			// Apply corruption
			corruptedData := tt.corruptionFunc(romData)

			reader := bytes.NewReader(corruptedData)
			cartridge, err := LoadFromReader(reader)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error for %s, got success", tt.description)
				}
			} else {
				if err != nil {
					t.Errorf("Expected success for %s, got error: %v", tt.description, err)
				}
				if cartridge == nil {
					t.Errorf("Expected cartridge for %s, got nil", tt.description)
				}
			}
		})
	}
}

// TestROMFormatValidation_ReadWriteOperations tests ROM vs RAM behavior
func TestROMFormatValidation_ReadWriteOperations(t *testing.T) {
	t.Run("PRG ROM write protection", func(t *testing.T) {
		romData := createMinimalValidROM(1, 1)
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Test that PRG ROM is write-protected
		originalValue := cartridge.ReadPRG(0x8000)
		cartridge.WritePRG(0x8000, ^originalValue) // Write inverted value
		afterWriteValue := cartridge.ReadPRG(0x8000)

		if afterWriteValue != originalValue {
			t.Errorf("PRG ROM not write-protected: original=0x%02X, after write=0x%02X",
				originalValue, afterWriteValue)
		}

		t.Logf("PRG ROM write protection verified: value remains 0x%02X", originalValue)
	})

	t.Run("CHR ROM write protection", func(t *testing.T) {
		romData := createMinimalValidROM(1, 1) // CHR ROM
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Test that CHR ROM is write-protected
		originalValue := cartridge.ReadCHR(0x0000)
		cartridge.WriteCHR(0x0000, ^originalValue) // Write inverted value
		afterWriteValue := cartridge.ReadCHR(0x0000)

		if afterWriteValue != originalValue {
			t.Errorf("CHR ROM not write-protected: original=0x%02X, after write=0x%02X",
				originalValue, afterWriteValue)
		}

		t.Logf("CHR ROM write protection verified: value remains 0x%02X", originalValue)
	})

	t.Run("CHR RAM write capability", func(t *testing.T) {
		romData := createMinimalValidROM(1, 0) // CHR RAM
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Test that CHR RAM is writable
		testValue := uint8(0x55)
		cartridge.WriteCHR(0x0000, testValue)
		readValue := cartridge.ReadCHR(0x0000)

		if readValue != testValue {
			t.Errorf("CHR RAM not writable: wrote 0x%02X, read 0x%02X", testValue, readValue)
		}

		t.Logf("CHR RAM write capability verified: wrote and read 0x%02X", testValue)
	})

	t.Run("SRAM write capability", func(t *testing.T) {
		romData := createMinimalValidROM(1, 1)
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Test SRAM range 0x6000-0x7FFF
		testValue := uint8(0xAA)
		cartridge.WritePRG(0x6000, testValue)
		readValue := cartridge.ReadPRG(0x6000)

		if readValue != testValue {
			t.Errorf("SRAM not writable: wrote 0x%02X, read 0x%02X", testValue, readValue)
		}

		t.Logf("SRAM write capability verified: wrote and read 0x%02X", testValue)
	})
}

// TestROMFormatValidation_BinaryStructureValidation tests binary structure integrity
func TestROMFormatValidation_BinaryStructureValidation(t *testing.T) {
	t.Run("Header binary structure", func(t *testing.T) {
		var header iNESHeader
		headerBytes := createValidINESHeader(2, 1, 4, 0x05, 0x40)

		buf := bytes.NewBuffer(headerBytes)
		err := binary.Read(buf, binary.LittleEndian, &header)

		if err != nil {
			t.Fatalf("Failed to parse header as binary structure: %v", err)
		}

		// Validate parsed values
		if string(header.Magic[:]) != "NES\x1A" {
			t.Errorf("Magic number mismatch: expected 'NES\\x1A', got %q", string(header.Magic[:]))
		}

		if header.PRGROMSize != 2 {
			t.Errorf("PRG ROM size mismatch: expected 2, got %d", header.PRGROMSize)
		}

		if header.CHRROMSize != 1 {
			t.Errorf("CHR ROM size mismatch: expected 1, got %d", header.CHRROMSize)
		}

		if header.Flags6 != 0x45 { // Mapper 4 lower nibble + flags
			t.Errorf("Flags6 mismatch: expected 0x45, got 0x%02X", header.Flags6)
		}

		if header.Flags7 != 0x40 {
			t.Errorf("Flags7 mismatch: expected 0x40, got 0x%02X", header.Flags7)
		}

		t.Logf("Binary structure validation passed")
	})

	t.Run("Endianness verification", func(t *testing.T) {
		// Create ROM with known byte patterns to verify endianness handling
		header := createValidINESHeader(1, 1, 0, 0, 0)

		// Create PRG ROM with specific pattern
		prgData := make([]byte, 16384)
		prgData[0] = 0x12
		prgData[1] = 0x34
		prgData[256] = 0x56
		prgData[257] = 0x78

		chrData := make([]byte, 8192)
		chrData[0] = 0xAB
		chrData[1] = 0xCD

		romData := append(header, prgData...)
		romData = append(romData, chrData...)

		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Verify byte order is preserved
		if cartridge.ReadPRG(0x8000) != 0x12 {
			t.Errorf("Byte order issue: expected 0x12, got 0x%02X", cartridge.ReadPRG(0x8000))
		}

		if cartridge.ReadPRG(0x8001) != 0x34 {
			t.Errorf("Byte order issue: expected 0x34, got 0x%02X", cartridge.ReadPRG(0x8001))
		}

		if cartridge.ReadCHR(0x0000) != 0xAB {
			t.Errorf("CHR byte order issue: expected 0xAB, got 0x%02X", cartridge.ReadCHR(0x0000))
		}

		t.Logf("Endianness verification passed")
	})
}

// BenchmarkROMFormatValidation_LoadPerformance benchmarks ROM loading performance
func BenchmarkROMFormatValidation_LoadPerformance(b *testing.B) {
	// Create various ROM sizes for benchmarking
	romSizes := []struct {
		name    string
		prgSize uint8
		chrSize uint8
	}{
		{"Small (16KB+8KB)", 1, 1},
		{"Medium (32KB+16KB)", 2, 2},
		{"Large (128KB+32KB)", 8, 4},
	}

	for _, size := range romSizes {
		b.Run(size.name, func(b *testing.B) {
			// Pre-create ROM data
			romData := createMinimalValidROM(size.prgSize, size.chrSize)

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				reader := bytes.NewReader(romData)
				cartridge, err := LoadFromReader(reader)
				if err != nil {
					b.Fatalf("Failed to load ROM: %v", err)
				}

				// Basic access to ensure full initialization
				_ = cartridge.ReadPRG(0x8000)
				_ = cartridge.ReadCHR(0x0000)
			}
		})
	}
}

// formatBytes formats byte count as human-readable string
func formatBytes(bytes int) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
