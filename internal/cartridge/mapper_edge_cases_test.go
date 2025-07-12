package cartridge

import (
	"fmt"
	"testing"
)

// TestMapperEdgeCases provides comprehensive edge case testing for mapper implementations
// This test suite focuses on boundary conditions, error scenarios, and hardware-specific behaviors

// TestMapperEdgeCases_Mapper000_BoundaryConditions tests NROM mapper boundary conditions
func TestMapperEdgeCases_Mapper000_BoundaryConditions(t *testing.T) {
	t.Run("16KB ROM mirroring boundaries", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x4000), // 16KB
			chrROM:   make([]uint8, 0x2000), // 8KB
			mapperID: 0,
		}

		// Fill with identifiable pattern
		for i := range cart.prgROM {
			cart.prgROM[i] = uint8((i + 0x80) & 0xFF)
		}

		mapper := NewMapper000(cart)

		// Test exact boundary addresses for mirroring
		boundaryTests := []struct {
			addr1, addr2 uint16
			description  string
		}{
			{0x8000, 0xC000, "Start of both banks"},
			{0xBFFF, 0xFFFF, "End of both banks"},
			{0x8001, 0xC001, "Second byte of both banks"},
			{0x9000, 0xD000, "Mid-bank mirroring"},
			{0xA555, 0xE555, "Arbitrary mid-address"},
		}

		for _, test := range boundaryTests {
			value1 := mapper.ReadPRG(test.addr1)
			value2 := mapper.ReadPRG(test.addr2)

			if value1 != value2 {
				t.Errorf("%s: mirroring failed 0x%04X=0x%02X, 0x%04X=0x%02X",
					test.description, test.addr1, value1, test.addr2, value2)
			}
		}

		// Verify mirroring formula: (addr - 0x8000) & 0x3FFF
		for i := 0; i < 100; i++ {
			addr := uint16(0x8000 + i*345) // Semi-random addresses
			if addr > 0xFFFF {
				break
			}

			expected := cart.prgROM[(addr-0x8000)&0x3FFF]
			actual := mapper.ReadPRG(addr)

			if actual != expected {
				t.Errorf("Mirroring formula failed at 0x%04X: expected 0x%02X, got 0x%02X",
					addr, expected, actual)
			}
		}
	})

	t.Run("32KB ROM no mirroring verification", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x8000), // 32KB
			chrROM:   make([]uint8, 0x2000), // 8KB
			mapperID: 0,
		}

		// Fill with position-dependent pattern
		for i := range cart.prgROM {
			cart.prgROM[i] = uint8(i & 0xFF)
		}

		mapper := NewMapper000(cart)

		// Verify no mirroring occurs
		mirrorTests := []struct {
			addr1, addr2 uint16
			shouldDiffer bool
		}{
			{0x8000, 0xC000, true}, // Different banks
			{0x8100, 0xC100, true}, // Different banks, same offset
			{0xBFFF, 0xFFFF, true}, // End of each bank
			{0x8000, 0x8001, true}, // Adjacent addresses
		}

		for _, test := range mirrorTests {
			value1 := mapper.ReadPRG(test.addr1)
			value2 := mapper.ReadPRG(test.addr2)

			if test.shouldDiffer && value1 == value2 {
				t.Errorf("32KB ROM incorrectly mirroring: 0x%04X=0x%02X equals 0x%04X=0x%02X",
					test.addr1, value1, test.addr2, value2)
			}
		}

		// Verify direct mapping: ROM[addr - 0x8000] = value
		testAddresses := []uint16{0x8000, 0x8100, 0x9000, 0xA000, 0xC000, 0xE000, 0xFFFF}
		for _, addr := range testAddresses {
			romIndex := addr - 0x8000
			expected := cart.prgROM[romIndex]
			actual := mapper.ReadPRG(addr)

			if actual != expected {
				t.Errorf("32KB direct mapping failed at 0x%04X: expected 0x%02X, got 0x%02X",
					addr, expected, actual)
			}
		}
	})

	t.Run("SRAM boundary testing", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x4000),
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
			sram:     [0x2000]uint8{},
		}

		mapper := NewMapper000(cart)

		// Test SRAM boundaries 0x6000-0x7FFF
		boundaryAddresses := []uint16{
			0x5FFF, // Just before SRAM
			0x6000, // Start of SRAM
			0x6001, // Second byte of SRAM
			0x7FFE, // Second-to-last byte of SRAM
			0x7FFF, // Last byte of SRAM
			0x8000, // Just after SRAM (ROM starts)
		}

		// Write test pattern to SRAM addresses
		for _, addr := range boundaryAddresses {
			mapper.WritePRG(addr, 0x42)
		}

		// Verify SRAM range behavior
		sramPattern := uint8(0x55)
		for addr := uint16(0x6000); addr < 0x8000; addr += 0x100 {
			mapper.WritePRG(addr, sramPattern)
			value := mapper.ReadPRG(addr)

			if value != sramPattern {
				t.Errorf("SRAM write/read failed at 0x%04X: wrote 0x%02X, read 0x%02X",
					addr, sramPattern, value)
			}
		}

		// Verify non-SRAM addresses return appropriate values
		nonSRAMTests := []struct {
			addr        uint16
			description string
		}{
			{0x5FFF, "Before SRAM range"},
			{0x0000, "Zero page"},
			{0x4000, "Mid-low address"},
		}

		for _, test := range nonSRAMTests {
			value := mapper.ReadPRG(test.addr)
			if value != 0 {
				t.Logf("%s (0x%04X) returned 0x%02X", test.description, test.addr, value)
			}
		}
	})

	t.Run("CHR memory boundary testing", func(t *testing.T) {
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

		// Test CHR ROM boundaries 0x0000-0x1FFF
		chrBoundaryTests := []struct {
			addr        uint16
			expectValid bool
			description string
		}{
			{0x0000, true, "Start of CHR ROM"},
			{0x0001, true, "Second byte of CHR ROM"},
			{0x1000, true, "Mid CHR ROM"},
			{0x1FFF, true, "End of CHR ROM"},
			{0x2000, false, "Just after CHR ROM"},
			{0x3000, false, "Well beyond CHR ROM"},
			{0xFFFF, false, "High address"},
		}

		for _, test := range chrBoundaryTests {
			value := mapper.ReadCHR(test.addr)

			if test.expectValid {
				expectedValue := cart.chrROM[test.addr]
				if value != expectedValue {
					t.Errorf("%s: expected 0x%02X, got 0x%02X",
						test.description, expectedValue, value)
				}
			} else {
				if value != 0 {
					t.Errorf("%s: expected 0 for invalid address, got 0x%02X",
						test.description, value)
				}
			}
		}

		// Test CHR ROM write protection
		originalValue := mapper.ReadCHR(0x1000)
		mapper.WriteCHR(0x1000, ^originalValue)
		afterWrite := mapper.ReadCHR(0x1000)

		if afterWrite != originalValue {
			t.Errorf("CHR ROM not write-protected: original=0x%02X, after=0x%02X",
				originalValue, afterWrite)
		}
	})

	t.Run("CHR RAM boundary testing", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:    make([]uint8, 0x4000),
			chrROM:    make([]uint8, 0x2000), // 8KB CHR RAM
			mapperID:  0,
			hasCHRRAM: true,
		}

		mapper := NewMapper000(cart)

		// Test CHR RAM write/read capabilities
		chrRAMTests := []uint16{0x0000, 0x0800, 0x1000, 0x1800, 0x1FFF}

		for i, addr := range chrRAMTests {
			testValue := uint8(0x80 + i)
			mapper.WriteCHR(addr, testValue)
			readValue := mapper.ReadCHR(addr)

			if readValue != testValue {
				t.Errorf("CHR RAM at 0x%04X: wrote 0x%02X, read 0x%02X",
					addr, testValue, readValue)
			}
		}

		// Test invalid CHR addresses don't affect valid ones
		mapper.WriteCHR(0x2000, 0xFF) // Invalid address
		mapper.WriteCHR(0x3000, 0xAA) // Invalid address

		// Verify valid addresses retain their values
		for i, addr := range chrRAMTests {
			expectedValue := uint8(0x80 + i)
			actualValue := mapper.ReadCHR(addr)

			if actualValue != expectedValue {
				t.Errorf("CHR RAM corrupted at 0x%04X after invalid writes: expected 0x%02X, got 0x%02X",
					addr, expectedValue, actualValue)
			}
		}
	})
}

// TestMapperEdgeCases_ErrorConditions tests error handling and edge conditions
func TestMapperEdgeCases_ErrorConditions(t *testing.T) {
	t.Run("Zero-size ROM handling", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   []uint8{}, // Zero-size PRG ROM
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
		}

		mapper := NewMapper000(cart)

		// Should handle gracefully without crashing
		value := mapper.ReadPRG(0x8000)
		if value != 0 {
			t.Errorf("Zero-size ROM should return 0, got 0x%02X", value)
		}

		// Multiple reads should be consistent
		for i := 0; i < 10; i++ {
			addr := uint16(0x8000 + i*0x1000)
			value := mapper.ReadPRG(addr)
			if value != 0 {
				t.Errorf("Zero-size ROM inconsistent at 0x%04X: got 0x%02X", addr, value)
			}
		}

		// Writes should not crash
		mapper.WritePRG(0x8000, 0x42)
		mapper.WritePRG(0xFFFF, 0x55)
	})

	t.Run("Zero-size CHR ROM handling", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:    make([]uint8, 0x4000),
			chrROM:    []uint8{}, // Zero-size CHR ROM
			mapperID:  0,
			hasCHRRAM: false,
		}

		mapper := NewMapper000(cart)

		// Should handle gracefully
		value := mapper.ReadCHR(0x0000)
		if value != 0 {
			t.Errorf("Zero-size CHR ROM should return 0, got 0x%02X", value)
		}

		// Writes should not crash
		mapper.WriteCHR(0x0000, 0x42)
		mapper.WriteCHR(0x1FFF, 0x55)
	})

	t.Run("Extreme address testing", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x8000), // 32KB
			chrROM:   make([]uint8, 0x2000), // 8KB
			mapperID: 0,
		}

		mapper := NewMapper000(cart)

		// Test extreme PRG addresses
		extremeAddresses := []uint16{
			0x0000, 0x0001, 0x00FF, // Low addresses
			0x5FFE, 0x5FFF, // Just before SRAM
			0x6000, 0x6001, // Start of SRAM
			0x7FFE, 0x7FFF, // End of SRAM
			0x8000, 0x8001, // Start of ROM
			0xFFFE, 0xFFFF, // End of address space
		}

		for _, addr := range extremeAddresses {
			// Should not crash on read
			value := mapper.ReadPRG(addr)
			_ = value

			// Should not crash on write
			mapper.WritePRG(addr, 0x42)
		}

		// Test extreme CHR addresses
		extremeCHRAddresses := []uint16{
			0x0000, 0x0001, 0x1FFE, 0x1FFF, // Valid range
			0x2000, 0x2001, 0x3FFF, // Invalid range
			0x8000, 0xFFFF, // Very high addresses
		}

		for _, addr := range extremeCHRAddresses {
			// Should not crash
			value := mapper.ReadCHR(addr)
			_ = value
			mapper.WriteCHR(addr, 0x55)
		}
	})

	t.Run("Memory aliasing verification", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x4000), // 16KB for mirroring test
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
		}

		// Create unique pattern in ROM
		for i := range cart.prgROM {
			cart.prgROM[i] = uint8((i ^ 0xAA) & 0xFF)
		}

		mapper := NewMapper000(cart)

		// Verify that different addresses map to same ROM location
		aliasingTests := []struct {
			addr1, addr2 uint16
			description  string
		}{
			{0x8000, 0xC000, "Bank mirroring start"},
			{0x8123, 0xC123, "Bank mirroring arbitrary"},
			{0xBFFF, 0xFFFF, "Bank mirroring end"},
		}

		for _, test := range aliasingTests {
			value1 := mapper.ReadPRG(test.addr1)
			value2 := mapper.ReadPRG(test.addr2)

			if value1 != value2 {
				t.Errorf("%s: aliasing failed 0x%04X=0x%02X vs 0x%04X=0x%02X",
					test.description, test.addr1, value1, test.addr2, value2)
			}

			// Verify both map to expected ROM location
			romOffset := (test.addr1 - 0x8000) & 0x3FFF
			expectedValue := cart.prgROM[romOffset]

			if value1 != expectedValue {
				t.Errorf("%s: ROM mapping incorrect at 0x%04X: expected 0x%02X, got 0x%02X",
					test.description, test.addr1, expectedValue, value1)
			}
		}
	})
}

// TestMapperEdgeCases_HardwareQuirks tests hardware-specific behaviors
func TestMapperEdgeCases_HardwareQuirks(t *testing.T) {
	t.Run("Bus conflict simulation", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x4000),
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
		}

		// Fill ROM with specific pattern
		for i := range cart.prgROM {
			cart.prgROM[i] = 0xAA
		}

		mapper := NewMapper000(cart)

		// Simulate writes to ROM area that should be ignored
		busConflictTests := []struct {
			addr        uint16
			writeValue  uint8
			description string
		}{
			{0x8000, 0x55, "Write opposite pattern to ROM start"},
			{0x9000, 0x00, "Write zero to ROM middle"},
			{0xFFFF, 0xFF, "Write all ones to ROM end"},
		}

		for _, test := range busConflictTests {
			originalValue := mapper.ReadPRG(test.addr)

			// Attempt write (should be ignored)
			mapper.WritePRG(test.addr, test.writeValue)

			// Verify value unchanged
			afterWrite := mapper.ReadPRG(test.addr)
			if afterWrite != originalValue {
				t.Errorf("%s: ROM modified by write! original=0x%02X, after=0x%02X",
					test.description, originalValue, afterWrite)
			}
		}
	})

	t.Run("Timing consistency", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x8000),
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
		}

		// Fill with pattern for verification
		for i := range cart.prgROM {
			cart.prgROM[i] = uint8(i & 0xFF)
		}

		mapper := NewMapper000(cart)

		// Verify repeated reads return same values (timing consistency)
		testAddresses := []uint16{0x8000, 0x9000, 0xA000, 0xC000, 0xE000, 0xFFFF}

		for _, addr := range testAddresses {
			values := make([]uint8, 100)

			// Read same address multiple times
			for i := range values {
				values[i] = mapper.ReadPRG(addr)
			}

			// All values should be identical
			firstValue := values[0]
			for i, value := range values {
				if value != firstValue {
					t.Errorf("Timing inconsistency at 0x%04X: read %d got 0x%02X, expected 0x%02X",
						addr, i, value, firstValue)
				}
			}
		}
	})

	t.Run("Power-on state simulation", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:    make([]uint8, 0x4000),
			chrROM:    make([]uint8, 0x2000),
			mapperID:  0,
			sram:      [0x2000]uint8{}, // Should be zero-initialized
			hasCHRRAM: true,
		}

		mapper := NewMapper000(cart)

		// Verify SRAM power-on state (should be zero)
		for addr := uint16(0x6000); addr < 0x8000; addr += 0x100 {
			value := mapper.ReadPRG(addr)
			if value != 0 {
				t.Errorf("SRAM not zero-initialized at power-on: 0x%04X = 0x%02X", addr, value)
			}
		}

		// Verify CHR RAM power-on state (should be zero)
		for addr := uint16(0x0000); addr < 0x2000; addr += 0x100 {
			value := mapper.ReadCHR(addr)
			if value != 0 {
				t.Errorf("CHR RAM not zero-initialized at power-on: 0x%04X = 0x%02X", addr, value)
			}
		}

		// Verify PRG ROM is accessible immediately
		romValue := mapper.ReadPRG(0x8000)
		_ = romValue // ROM can contain any value

		t.Logf("Power-on state verification completed")
	})

	t.Run("Memory corruption resistance", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x4000),
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
			sram:     [0x2000]uint8{},
		}

		// Initialize SRAM with known pattern
		for i := range cart.sram {
			cart.sram[i] = uint8(i & 0xFF)
		}

		mapper := NewMapper000(cart)

		// Store checksum of SRAM
		originalChecksum := uint32(0)
		for addr := uint16(0x6000); addr < 0x8000; addr++ {
			originalChecksum += uint32(mapper.ReadPRG(addr))
		}

		// Perform many operations that shouldn't affect SRAM
		corruptionAttempts := []func(){
			func() { mapper.WritePRG(0x8000, 0xFF) }, // Write to ROM
			func() { mapper.WritePRG(0x5FFF, 0xAA) }, // Write before SRAM
			func() { mapper.WritePRG(0x8000, 0x55) }, // Write to ROM again
			func() { mapper.ReadPRG(0xFFFF) },        // Read from ROM end
			func() { mapper.WriteCHR(0x2000, 0x77) }, // Write to invalid CHR
		}

		for i, attempt := range corruptionAttempts {
			attempt()

			// Verify SRAM integrity
			checksum := uint32(0)
			for addr := uint16(0x6000); addr < 0x8000; addr++ {
				checksum += uint32(mapper.ReadPRG(addr))
			}

			if checksum != originalChecksum {
				t.Errorf("SRAM corrupted after operation %d: checksum changed from %d to %d",
					i, originalChecksum, checksum)
			}
		}
	})
}

// TestMapperEdgeCases_StateConsistency tests state consistency across operations
func TestMapperEdgeCases_StateConsistency(t *testing.T) {
	t.Run("SRAM persistence across boundary access", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x4000),
			chrROM:   make([]uint8, 0x2000),
			mapperID: 0,
			sram:     [0x2000]uint8{},
		}

		mapper := NewMapper000(cart)

		// Write pattern to SRAM
		testPattern := []struct {
			addr  uint16
			value uint8
		}{
			{0x6000, 0x11}, // First byte
			{0x6001, 0x22}, // Second byte
			{0x7000, 0x33}, // Middle
			{0x7FFE, 0x44}, // Second-to-last
			{0x7FFF, 0x55}, // Last byte
		}

		// Write pattern
		for _, p := range testPattern {
			mapper.WritePRG(p.addr, p.value)
		}

		// Perform unrelated operations
		for i := 0; i < 100; i++ {
			mapper.ReadPRG(0x8000 + uint16(i))
			mapper.WritePRG(0x8000+uint16(i), uint8(i))
			mapper.ReadCHR(uint16(i))
			mapper.WriteCHR(uint16(i), uint8(i))
		}

		// Verify pattern persists
		for _, p := range testPattern {
			value := mapper.ReadPRG(p.addr)
			if value != p.value {
				t.Errorf("SRAM pattern corrupted at 0x%04X: expected 0x%02X, got 0x%02X",
					p.addr, p.value, value)
			}
		}
	})

	t.Run("CHR RAM state consistency", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:    make([]uint8, 0x4000),
			chrROM:    make([]uint8, 0x2000),
			mapperID:  0,
			hasCHRRAM: true,
		}

		mapper := NewMapper000(cart)

		// Create checkerboard pattern in CHR RAM
		for addr := uint16(0x0000); addr < 0x2000; addr++ {
			value := uint8(0x55)
			if (addr & 1) != 0 {
				value = 0xAA
			}
			mapper.WriteCHR(addr, value)
		}

		// Perform operations that shouldn't affect CHR RAM
		for i := 0; i < 50; i++ {
			mapper.ReadPRG(0x8000 + uint16(i*100))
			mapper.WritePRG(0x6000+uint16(i), uint8(i))
		}

		// Verify checkerboard pattern intact
		for addr := uint16(0x0000); addr < 0x2000; addr += 64 { // Sample pattern
			expectedValue := uint8(0x55)
			if (addr & 1) != 0 {
				expectedValue = 0xAA
			}

			actualValue := mapper.ReadCHR(addr)
			if actualValue != expectedValue {
				t.Errorf("CHR RAM pattern corrupted at 0x%04X: expected 0x%02X, got 0x%02X",
					addr, expectedValue, actualValue)
			}
		}
	})

	t.Run("ROM data integrity", func(t *testing.T) {
		cart := &Cartridge{
			prgROM:   make([]uint8, 0x8000), // 32KB
			chrROM:   make([]uint8, 0x2000), // 8KB
			mapperID: 0,
		}

		// Fill ROM with cryptographic-like pattern
		for i := range cart.prgROM {
			cart.prgROM[i] = uint8((i*17 + 83) & 0xFF)
		}

		for i := range cart.chrROM {
			cart.chrROM[i] = uint8((i*23 + 97) & 0xFF)
		}

		mapper := NewMapper000(cart)

		// Calculate initial checksums
		prgChecksum := uint32(0)
		for addr := uint16(0x8000); addr <= 0xFFFF; addr += 73 { // Prime step
			prgChecksum += uint32(mapper.ReadPRG(addr))
		}

		chrChecksum := uint32(0)
		for addr := uint16(0x0000); addr < 0x2000; addr += 37 { // Prime step
			chrChecksum += uint32(mapper.ReadCHR(addr))
		}

		// Perform extensive operations
		for i := 0; i < 1000; i++ {
			// Random writes to writable areas
			mapper.WritePRG(0x6000+uint16(i%0x2000), uint8(i))

			// Attempted writes to ROM (should be ignored)
			mapper.WritePRG(0x8000+uint16(i%0x8000), uint8(i))
			mapper.WriteCHR(uint16(i%0x2000), uint8(i))
		}

		// Recalculate checksums
		newPRGChecksum := uint32(0)
		for addr := uint16(0x8000); addr <= 0xFFFF; addr += 73 {
			newPRGChecksum += uint32(mapper.ReadPRG(addr))
		}

		newCHRChecksum := uint32(0)
		for addr := uint16(0x0000); addr < 0x2000; addr += 37 {
			newCHRChecksum += uint32(mapper.ReadCHR(addr))
		}

		// Verify ROM integrity
		if newPRGChecksum != prgChecksum {
			t.Errorf("PRG ROM integrity compromised: checksum %d != %d", newPRGChecksum, prgChecksum)
		}

		if newCHRChecksum != chrChecksum {
			t.Errorf("CHR ROM integrity compromised: checksum %d != %d", newCHRChecksum, chrChecksum)
		}
	})
}

// TestMapperEdgeCases_UnsupportedMappers tests handling of unsupported mapper IDs
func TestMapperEdgeCases_UnsupportedMappers(t *testing.T) {
	unsupportedMappers := []uint8{1, 2, 3, 4, 5, 10, 50, 100, 200, 255}

	for _, mapperID := range unsupportedMappers {
		t.Run(fmt.Sprintf("Mapper %d fallback", mapperID), func(t *testing.T) {
			cart := &Cartridge{
				prgROM:   make([]uint8, 0x4000),
				chrROM:   make([]uint8, 0x2000),
				mapperID: mapperID,
			}

			// Should create Mapper000 as fallback
			mapper := createMapper(mapperID, cart)

			if mapper == nil {
				t.Fatalf("createMapper returned nil for mapper %d", mapperID)
			}

			// Should behave like Mapper000
			mapper.WritePRG(0x6000, 0x42)
			value := mapper.ReadPRG(0x6000)

			if value != 0x42 {
				t.Errorf("Unsupported mapper %d fallback doesn't behave like Mapper000", mapperID)
			}

			// Type assertion to verify it's actually Mapper000
			if _, ok := mapper.(*Mapper000); !ok {
				t.Errorf("Unsupported mapper %d didn't fall back to Mapper000", mapperID)
			}
		})
	}
}

// BenchmarkMapperEdgeCases_Performance benchmarks edge case performance
func BenchmarkMapperEdgeCases_Performance(b *testing.B) {
	cart := &Cartridge{
		prgROM:   make([]uint8, 0x8000),
		chrROM:   make([]uint8, 0x2000),
		mapperID: 0,
	}

	mapper := NewMapper000(cart)

	b.Run("Boundary crossing reads", func(b *testing.B) {
		addresses := []uint16{0x7FFF, 0x8000, 0xFFFF, 0x1FFF, 0x2000}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := addresses[i%len(addresses)]
			if addr < 0x2000 {
				_ = mapper.ReadCHR(addr)
			} else {
				_ = mapper.ReadPRG(addr)
			}
		}
	})

	b.Run("Mirroring calculation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := uint16(0x8000 + (i % 0x8000))
			_ = mapper.ReadPRG(addr)
		}
	})

	b.Run("SRAM access pattern", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := uint16(0x6000 + (i % 0x2000))
			if i%2 == 0 {
				mapper.WritePRG(addr, uint8(i))
			} else {
				_ = mapper.ReadPRG(addr)
			}
		}
	})
}
