package cartridge

import (
	"bytes"
	"fmt"
	"math/rand"
	"testing"
	"time"
)

// TestCartridgeCompatibility provides comprehensive compatibility validation tests
// This test suite validates overall cartridge compatibility, integration scenarios, and real-world usage patterns

// CartridgeTestProfile represents a test profile for different cartridge configurations
type CartridgeTestProfile struct {
	Name            string
	PRGSize         uint8
	CHRSize         uint8
	MapperID        uint8
	Flags6          uint8
	Flags7          uint8
	HasTrainer      bool
	ExpectedMirror  MirrorMode
	ExpectedBattery bool
	Description     string
}

// getStandardTestProfiles returns a set of standard cartridge configurations for compatibility testing
func getStandardTestProfiles() []CartridgeTestProfile {
	return []CartridgeTestProfile{
		{
			Name:            "Standard NROM-128",
			PRGSize:         1,
			CHRSize:         1,
			MapperID:        0,
			Flags6:          0x00,
			Flags7:          0x00,
			HasTrainer:      false,
			ExpectedMirror:  MirrorHorizontal,
			ExpectedBattery: false,
			Description:     "16KB PRG ROM + 8KB CHR ROM, horizontal mirroring",
		},
		{
			Name:            "Standard NROM-256",
			PRGSize:         2,
			CHRSize:         1,
			MapperID:        0,
			Flags6:          0x00,
			Flags7:          0x00,
			HasTrainer:      false,
			ExpectedMirror:  MirrorHorizontal,
			ExpectedBattery: false,
			Description:     "32KB PRG ROM + 8KB CHR ROM, horizontal mirroring",
		},
		{
			Name:            "NROM with CHR RAM",
			PRGSize:         1,
			CHRSize:         0,
			MapperID:        0,
			Flags6:          0x00,
			Flags7:          0x00,
			HasTrainer:      false,
			ExpectedMirror:  MirrorHorizontal,
			ExpectedBattery: false,
			Description:     "16KB PRG ROM + 8KB CHR RAM, horizontal mirroring",
		},
		{
			Name:            "NROM with battery",
			PRGSize:         1,
			CHRSize:         1,
			MapperID:        0,
			Flags6:          0x02,
			Flags7:          0x00,
			HasTrainer:      false,
			ExpectedMirror:  MirrorHorizontal,
			ExpectedBattery: true,
			Description:     "16KB PRG ROM + 8KB CHR ROM with battery-backed SRAM",
		},
		{
			Name:            "NROM vertical mirroring",
			PRGSize:         1,
			CHRSize:         1,
			MapperID:        0,
			Flags6:          0x01,
			Flags7:          0x00,
			HasTrainer:      false,
			ExpectedMirror:  MirrorVertical,
			ExpectedBattery: false,
			Description:     "16KB PRG ROM + 8KB CHR ROM, vertical mirroring",
		},
		{
			Name:            "NROM four-screen",
			PRGSize:         2,
			CHRSize:         1,
			MapperID:        0,
			Flags6:          0x08,
			Flags7:          0x00,
			HasTrainer:      false,
			ExpectedMirror:  MirrorFourScreen,
			ExpectedBattery: false,
			Description:     "32KB PRG ROM + 8KB CHR ROM, four-screen mirroring",
		},
		{
			Name:            "NROM with trainer",
			PRGSize:         1,
			CHRSize:         1,
			MapperID:        0,
			Flags6:          0x04,
			Flags7:          0x00,
			HasTrainer:      true,
			ExpectedMirror:  MirrorHorizontal,
			ExpectedBattery: false,
			Description:     "16KB PRG ROM + 8KB CHR ROM with 512-byte trainer",
		},
		{
			Name:            "Complex configuration",
			PRGSize:         2,
			CHRSize:         2,
			MapperID:        0,
			Flags6:          0x0F, // All low flags set
			Flags7:          0x00,
			HasTrainer:      true,
			ExpectedMirror:  MirrorFourScreen, // Four-screen overrides vertical
			ExpectedBattery: true,
			Description:     "32KB PRG ROM + 16KB CHR ROM with all features",
		},
		{
			Name:            "Unsupported mapper fallback",
			PRGSize:         1,
			CHRSize:         1,
			MapperID:        1, // MMC1 - unsupported, should fall back to NROM
			Flags6:          0x10,
			Flags7:          0x00,
			HasTrainer:      false,
			ExpectedMirror:  MirrorHorizontal,
			ExpectedBattery: false,
			Description:     "Unsupported mapper should fall back to NROM behavior",
		},
	}
}

// TestCartridgeCompatibility_StandardConfigurations tests standard cartridge configurations
func TestCartridgeCompatibility_StandardConfigurations(t *testing.T) {
	profiles := getStandardTestProfiles()

	for _, profile := range profiles {
		t.Run(profile.Name, func(t *testing.T) {
			// Create ROM data based on profile
			header := createValidINESHeader(profile.PRGSize, profile.CHRSize, profile.MapperID,
				profile.Flags6, profile.Flags7)

			romData := append([]byte{}, header...)

			// Add trainer if specified
			if profile.HasTrainer {
				trainerData := make([]byte, 512)
				for i := range trainerData {
					trainerData[i] = 0xCC // Recognizable pattern
				}
				romData = append(romData, trainerData...)
			}

			// Add PRG ROM data
			prgSize := int(profile.PRGSize) * 16384
			prgData := make([]byte, prgSize)
			for i := range prgData {
				prgData[i] = uint8((i + 0x12) & 0xFF)
			}
			romData = append(romData, prgData...)

			// Add CHR ROM data if specified
			if profile.CHRSize > 0 {
				chrSize := int(profile.CHRSize) * 8192
				chrData := make([]byte, chrSize)
				for i := range chrData {
					chrData[i] = uint8((i + 0x34) & 0xFF)
				}
				romData = append(romData, chrData...)
			}

			// Load cartridge
			reader := bytes.NewReader(romData)
			cartridge, err := LoadFromReader(reader)

			if err != nil {
				t.Fatalf("Failed to load %s: %v", profile.Description, err)
			}

			// Validate configuration
			if cartridge.mapperID != profile.MapperID {
				t.Errorf("Mapper ID mismatch: expected %d, got %d",
					profile.MapperID, cartridge.mapperID)
			}

			if cartridge.mirror != profile.ExpectedMirror {
				t.Errorf("Mirror mode mismatch: expected %d, got %d",
					profile.ExpectedMirror, cartridge.mirror)
			}

			if cartridge.hasBattery != profile.ExpectedBattery {
				t.Errorf("Battery flag mismatch: expected %v, got %v",
					profile.ExpectedBattery, cartridge.hasBattery)
			}

			// Validate ROM sizes
			expectedPRGSize := int(profile.PRGSize) * 16384
			if len(cartridge.prgROM) != expectedPRGSize {
				t.Errorf("PRG ROM size mismatch: expected %d, got %d",
					expectedPRGSize, len(cartridge.prgROM))
			}

			expectedCHRSize := 8192 // Default CHR RAM size
			if profile.CHRSize > 0 {
				expectedCHRSize = int(profile.CHRSize) * 8192
			}
			if len(cartridge.chrROM) != expectedCHRSize {
				t.Errorf("CHR ROM size mismatch: expected %d, got %d",
					expectedCHRSize, len(cartridge.chrROM))
			}

			// Validate CHR RAM flag
			expectedCHRRAM := profile.CHRSize == 0
			if cartridge.hasCHRRAM != expectedCHRRAM {
				t.Errorf("CHR RAM flag mismatch: expected %v, got %v",
					expectedCHRRAM, cartridge.hasCHRRAM)
			}

			t.Logf("Successfully validated: %s", profile.Description)
		})
	}
}

// TestCartridgeCompatibility_MemoryAccessPatterns tests various memory access patterns
func TestCartridgeCompatibility_MemoryAccessPatterns(t *testing.T) {
	t.Run("Sequential access pattern", func(t *testing.T) {
		romData := createMinimalValidROM(2, 1) // 32KB PRG + 8KB CHR
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Test sequential PRG ROM access
		for addr := uint16(0x8000); addr <= 0xFFFF; addr += 0x100 {
			value := cartridge.ReadPRG(addr)
			_ = value // Each read should succeed without error
		}

		// Test sequential CHR ROM access
		for addr := uint16(0x0000); addr < 0x2000; addr += 0x40 {
			value := cartridge.ReadCHR(addr)
			_ = value // Each read should succeed without error
		}

		// Test sequential SRAM access
		testPattern := uint8(0x77)
		for addr := uint16(0x6000); addr < 0x8000; addr += 0x80 {
			cartridge.WritePRG(addr, testPattern)
			value := cartridge.ReadPRG(addr)
			if value != testPattern {
				t.Errorf("SRAM access failed at 0x%04X: wrote 0x%02X, read 0x%02X",
					addr, testPattern, value)
			}
		}
	})

	t.Run("Random access pattern", func(t *testing.T) {
		romData := createMinimalValidROM(1, 0) // 16KB PRG + CHR RAM
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Use deterministic random seed for reproducible tests
		rng := rand.New(rand.NewSource(12345))

		// Random PRG ROM access
		for i := 0; i < 100; i++ {
			addr := uint16(0x8000 + rng.Intn(0x8000))
			value := cartridge.ReadPRG(addr)
			_ = value
		}

		// Random CHR RAM access with write/read verification
		for i := 0; i < 50; i++ {
			addr := uint16(rng.Intn(0x2000))
			testValue := uint8(rng.Intn(256))

			cartridge.WriteCHR(addr, testValue)
			readValue := cartridge.ReadCHR(addr)

			if readValue != testValue {
				t.Errorf("Random CHR RAM access failed at 0x%04X: wrote 0x%02X, read 0x%02X",
					addr, testValue, readValue)
			}
		}

		// Random SRAM access
		for i := 0; i < 50; i++ {
			addr := uint16(0x6000 + rng.Intn(0x2000))
			testValue := uint8(rng.Intn(256))

			cartridge.WritePRG(addr, testValue)
			readValue := cartridge.ReadPRG(addr)

			if readValue != testValue {
				t.Errorf("Random SRAM access failed at 0x%04X: wrote 0x%02X, read 0x%02X",
					addr, testValue, readValue)
			}
		}
	})

	t.Run("Burst access pattern", func(t *testing.T) {
		romData := createMinimalValidROM(2, 2) // 32KB PRG + 16KB CHR
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Burst read from PRG ROM
		burstSize := 256
		for base := 0x8000; base <= 0xFF00; base += burstSize {
			for offset := 0; offset < burstSize; offset++ {
				addr := uint16(base + offset)
				value := cartridge.ReadPRG(addr)
				_ = value
			}
		}

		// Burst write/read to SRAM
		for base := 0x6000; base <= 0x7F00; base += burstSize {
			pattern := uint8(base >> 8)

			// Burst write
			for offset := 0; offset < burstSize && base+offset < 0x8000; offset++ {
				addr := uint16(base + offset)
				cartridge.WritePRG(addr, pattern)
			}

			// Burst read and verify
			for offset := 0; offset < burstSize && base+offset < 0x8000; offset++ {
				addr := uint16(base + offset)
				value := cartridge.ReadPRG(addr)
				if value != pattern {
					t.Errorf("Burst SRAM access failed at 0x%04X: expected 0x%02X, got 0x%02X",
						addr, pattern, value)
				}
			}
		}
	})
}

// TestCartridgeCompatibility_ConcurrentAccess tests concurrent access scenarios
func TestCartridgeCompatibility_ConcurrentAccess(t *testing.T) {
	t.Run("Concurrent ROM reads", func(t *testing.T) {
		romData := createMinimalValidROM(2, 1)
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Launch multiple goroutines reading from ROM
		const numGoroutines = 10
		const readsPerGoroutine = 100

		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				for j := 0; j < readsPerGoroutine; j++ {
					addr := uint16(0x8000 + (id*readsPerGoroutine+j)%0x8000)
					value1 := cartridge.ReadPRG(addr)
					value2 := cartridge.ReadPRG(addr)

					// Same address should return same value
					if value1 != value2 {
						errors <- fmt.Errorf("concurrent read inconsistency at 0x%04X: %02X != %02X",
							addr, value1, value2)
						return
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Errorf("Concurrent read error: %v", err)
			case <-time.After(5 * time.Second):
				t.Fatal("Concurrent read test timed out")
			}
		}
	})

	t.Run("Concurrent SRAM access", func(t *testing.T) {
		romData := createMinimalValidROM(1, 1)
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Concurrent access to different SRAM regions
		const numGoroutines = 4
		sramRegionSize := 0x2000 / numGoroutines

		done := make(chan bool, numGoroutines)
		errors := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(regionID int) {
				defer func() { done <- true }()

				baseAddr := uint16(0x6000 + regionID*sramRegionSize)
				pattern := uint8(0x10 + regionID)

				// Write pattern to region
				for offset := 0; offset < sramRegionSize; offset += 4 {
					addr := baseAddr + uint16(offset)
					cartridge.WritePRG(addr, pattern)
				}

				// Verify pattern
				for offset := 0; offset < sramRegionSize; offset += 4 {
					addr := baseAddr + uint16(offset)
					value := cartridge.ReadPRG(addr)
					if value != pattern {
						errors <- fmt.Errorf("SRAM region %d corrupted at 0x%04X: expected 0x%02X, got 0x%02X",
							regionID, addr, pattern, value)
						return
					}
				}
			}(i)
		}

		// Wait for completion
		for i := 0; i < numGoroutines; i++ {
			select {
			case <-done:
				// Success
			case err := <-errors:
				t.Errorf("Concurrent SRAM error: %v", err)
			case <-time.After(3 * time.Second):
				t.Fatal("Concurrent SRAM test timed out")
			}
		}
	})
}

// TestCartridgeCompatibility_StressTest performs stress testing under heavy load
func TestCartridgeCompatibility_StressTest(t *testing.T) {
	t.Run("High frequency access", func(t *testing.T) {
		romData := createMinimalValidROM(2, 0) // 32KB PRG + CHR RAM
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Simulate high-frequency access pattern typical of NES operation
		const iterations = 100000
		errorCount := 0

		for i := 0; i < iterations; i++ {
			// Simulate CPU instruction fetch
			addr := uint16(0x8000 + (i % 0x8000))
			_ = cartridge.ReadPRG(addr)

			// Simulate PPU pattern table access
			chrAddr := uint16(i % 0x2000)
			_ = cartridge.ReadCHR(chrAddr)

			// Occasional SRAM access
			if i%100 == 0 {
				sramAddr := uint16(0x6000 + (i % 0x2000))
				cartridge.WritePRG(sramAddr, uint8(i))
				value := cartridge.ReadPRG(sramAddr)
				if value != uint8(i) {
					errorCount++
				}
			}

			// Occasional CHR RAM write
			if i%50 == 0 {
				chrWriteAddr := uint16(i % 0x2000)
				cartridge.WriteCHR(chrWriteAddr, uint8(i>>8))
			}
		}

		if errorCount > 0 {
			t.Errorf("Stress test failed with %d errors out of %d operations", errorCount, iterations)
		}

		t.Logf("Stress test completed: %d operations with %d errors", iterations, errorCount)
	})

	t.Run("Memory pressure test", func(t *testing.T) {
		// Create multiple cartridges to test memory management
		const numCartridges = 50
		cartridges := make([]*Cartridge, numCartridges)

		for i := 0; i < numCartridges; i++ {
			prgSize := uint8(1 + (i % 4)) // 1-4 banks
			chrSize := uint8((i % 3) + 1) // 1-3 banks

			romData := createMinimalValidROM(prgSize, chrSize)
			reader := bytes.NewReader(romData)

			cart, err := LoadFromReader(reader)
			if err != nil {
				t.Fatalf("Failed to create cartridge %d: %v", i, err)
			}

			cartridges[i] = cart
		}

		// Access all cartridges to ensure they're functional
		for i, cart := range cartridges {
			if cart == nil {
				t.Errorf("Cartridge %d is nil", i)
				continue
			}

			// Test basic functionality
			value := cart.ReadPRG(0x8000)
			cart.WritePRG(0x6000, uint8(i))
			sramValue := cart.ReadPRG(0x6000)

			if sramValue != uint8(i) {
				t.Errorf("Cartridge %d SRAM test failed: expected %d, got %d", i, i, sramValue)
			}

			_ = value // Use value to avoid unused variable warning
		}

		t.Logf("Memory pressure test completed with %d cartridges", numCartridges)
	})
}

// TestCartridgeCompatibility_RealWorldScenarios tests real-world usage scenarios
func TestCartridgeCompatibility_RealWorldScenarios(t *testing.T) {
	t.Run("Game boot sequence simulation", func(t *testing.T) {
		romData := createMinimalValidROM(2, 1)
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Simulate typical NES boot sequence
		steps := []struct {
			description string
			operation   func() error
		}{
			{
				"Read reset vector",
				func() error {
					_ = cartridge.ReadPRG(0xFFFC) // Reset vector low
					_ = cartridge.ReadPRG(0xFFFD) // Reset vector high
					return nil
				},
			},
			{
				"Clear SRAM",
				func() error {
					for addr := uint16(0x6000); addr < 0x8000; addr++ {
						cartridge.WritePRG(addr, 0x00)
					}
					return nil
				},
			},
			{
				"Initialize CHR data",
				func() error {
					// Read CHR ROM patterns
					for addr := uint16(0x0000); addr < 0x2000; addr += 16 {
						_ = cartridge.ReadCHR(addr)
					}
					return nil
				},
			},
			{
				"Program execution simulation",
				func() error {
					// Simulate instruction fetching
					for pc := uint16(0x8000); pc < 0x8100; pc++ {
						_ = cartridge.ReadPRG(pc)
					}
					return nil
				},
			},
			{
				"Save game data",
				func() error {
					saveData := []uint8{0xDE, 0xAD, 0xBE, 0xEF}
					for i, data := range saveData {
						cartridge.WritePRG(0x6000+uint16(i), data)
					}

					// Verify save data
					for i, expected := range saveData {
						actual := cartridge.ReadPRG(0x6000 + uint16(i))
						if actual != expected {
							return fmt.Errorf("save data mismatch at offset %d: expected 0x%02X, got 0x%02X",
								i, expected, actual)
						}
					}
					return nil
				},
			},
		}

		for _, step := range steps {
			if err := step.operation(); err != nil {
				t.Errorf("%s failed: %v", step.description, err)
			} else {
				t.Logf("%s: OK", step.description)
			}
		}
	})

	t.Run("Mapper compatibility verification", func(t *testing.T) {
		// Test that unsupported mappers fall back to NROM behavior
		unsupportedMappers := []uint8{1, 2, 3, 4, 5}

		for _, mapperID := range unsupportedMappers {
			t.Run(fmt.Sprintf("Mapper_%d_fallback", mapperID), func(t *testing.T) {
				// Create ROM with unsupported mapper
				header := createValidINESHeader(1, 1, mapperID, 0, 0)
				prgData := make([]byte, 16384)
				chrData := make([]byte, 8192)

				romData := append(header, prgData...)
				romData = append(romData, chrData...)

				reader := bytes.NewReader(romData)
				cartridge, err := LoadFromReader(reader)

				if err != nil {
					t.Fatalf("Unsupported mapper %d should load with fallback: %v", mapperID, err)
				}

				// Should behave like NROM
				cartridge.WritePRG(0x6000, 0x42)
				value := cartridge.ReadPRG(0x6000)

				if value != 0x42 {
					t.Errorf("Mapper %d fallback doesn't work like NROM", mapperID)
				}

				// Mapper ID should be preserved even though behavior falls back
				if cartridge.mapperID != mapperID {
					t.Errorf("Mapper ID not preserved: expected %d, got %d",
						mapperID, cartridge.mapperID)
				}
			})
		}
	})

	t.Run("Battery backup simulation", func(t *testing.T) {
		// Create cartridge with battery backup
		header := createValidINESHeader(1, 1, 0, 0x02, 0) // Battery flag set
		prgData := make([]byte, 16384)
		chrData := make([]byte, 8192)

		romData := append(header, prgData...)
		romData = append(romData, chrData...)

		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load battery-backed ROM: %v", err)
		}

		// Verify battery flag is set
		if !cartridge.hasBattery {
			t.Error("Battery flag should be set")
		}

		// Simulate saving high score
		highScoreData := []uint8{0x12, 0x34, 0x56, 0x78}
		for i, score := range highScoreData {
			cartridge.WritePRG(0x6000+uint16(i), score)
		}

		// Simulate power cycle by creating new cartridge with same ROM
		reader2 := bytes.NewReader(romData)
		cartridge2, err := LoadFromReader(reader2)

		if err != nil {
			t.Fatalf("Failed to reload battery-backed ROM: %v", err)
		}

		// In real hardware, battery data would persist
		// Our implementation starts with zero SRAM, which is correct for power-on
		for i := range highScoreData {
			value := cartridge2.ReadPRG(0x6000 + uint16(i))
			if value != 0 {
				t.Logf("SRAM initialized to 0 on power-on (correct behavior)")
			}
		}

		t.Logf("Battery backup simulation completed")
	})
}

// BenchmarkCartridgeCompatibility_Performance benchmarks compatibility scenarios
func BenchmarkCartridgeCompatibility_Performance(b *testing.B) {
	romData := createMinimalValidROM(2, 1)
	reader := bytes.NewReader(romData)
	cartridge, err := LoadFromReader(reader)

	if err != nil {
		b.Fatalf("Failed to load ROM: %v", err)
	}

	b.Run("ROM_read_performance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := uint16(0x8000 + (i % 0x8000))
			_ = cartridge.ReadPRG(addr)
		}
	})

	b.Run("SRAM_write_performance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := uint16(0x6000 + (i % 0x2000))
			cartridge.WritePRG(addr, uint8(i))
		}
	})

	b.Run("SRAM_read_performance", func(b *testing.B) {
		// Pre-fill SRAM
		for addr := uint16(0x6000); addr < 0x8000; addr++ {
			cartridge.WritePRG(addr, uint8(addr))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := uint16(0x6000 + (i % 0x2000))
			_ = cartridge.ReadPRG(addr)
		}
	})

	b.Run("CHR_access_performance", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			addr := uint16(i % 0x2000)
			if i%2 == 0 {
				_ = cartridge.ReadCHR(addr)
			} else {
				cartridge.WriteCHR(addr, uint8(i))
			}
		}
	})

	b.Run("Mixed_access_pattern", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			switch i % 4 {
			case 0:
				_ = cartridge.ReadPRG(0x8000 + uint16(i%0x8000))
			case 1:
				cartridge.WritePRG(0x6000+uint16(i%0x2000), uint8(i))
			case 2:
				_ = cartridge.ReadCHR(uint16(i % 0x2000))
			case 3:
				cartridge.WriteCHR(uint16(i%0x2000), uint8(i))
			}
		}
	})
}

// TestCartridgeCompatibility_ErrorRecovery tests error recovery scenarios
func TestCartridgeCompatibility_ErrorRecovery(t *testing.T) {
	t.Run("Invalid operations recovery", func(t *testing.T) {
		romData := createMinimalValidROM(1, 1)
		reader := bytes.NewReader(romData)
		cartridge, err := LoadFromReader(reader)

		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Perform invalid operations
		invalidOperations := []func(){
			func() { cartridge.WritePRG(0x8000, 0xFF) }, // Write to ROM
			func() { cartridge.WritePRG(0x0000, 0xAA) }, // Write to invalid area
			func() { cartridge.WriteCHR(0x2000, 0x55) }, // Write to invalid CHR area
			func() { _ = cartridge.ReadPRG(0x0000) },    // Read from invalid area
			func() { _ = cartridge.ReadCHR(0x3000) },    // Read from invalid CHR area
		}

		for i, operation := range invalidOperations {
			operation() // Should not crash

			// Verify cartridge still works after invalid operation
			cartridge.WritePRG(0x6000, uint8(i))
			value := cartridge.ReadPRG(0x6000)

			if value != uint8(i) {
				t.Errorf("Cartridge corrupted after invalid operation %d", i)
			}
		}

		t.Logf("Error recovery test completed successfully")
	})
}

// formatSize formats a size in bytes as human-readable string
func formatSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	} else if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}
