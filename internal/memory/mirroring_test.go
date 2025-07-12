package memory

import (
	"testing"
)

// TestCPUMemoryMirroring tests all CPU memory mirroring behaviors
func TestCPUMemoryMirroring_InternalRAM(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test internal RAM mirroring every 2KB
	// The 2KB RAM is mirrored 4 times in the $0000-$1FFF range

	testPatterns := []struct {
		name        string
		baseAddr    uint16
		mirrorAddrs []uint16
	}{
		{
			name:        "RAM address 0x0000",
			baseAddr:    0x0000,
			mirrorAddrs: []uint16{0x0800, 0x1000, 0x1800},
		},
		{
			name:        "RAM address 0x0001",
			baseAddr:    0x0001,
			mirrorAddrs: []uint16{0x0801, 0x1001, 0x1801},
		},
		{
			name:        "RAM address 0x0100",
			baseAddr:    0x0100,
			mirrorAddrs: []uint16{0x0900, 0x1100, 0x1900},
		},
		{
			name:        "RAM address 0x07FF",
			baseAddr:    0x07FF,
			mirrorAddrs: []uint16{0x0FFF, 0x17FF, 0x1FFF},
		},
	}

	for _, tp := range testPatterns {
		t.Run(tp.name, func(t *testing.T) {
			value := uint8(tp.baseAddr & 0xFF)

			// Write to base address
			mem.Write(tp.baseAddr, value)

			// Verify all mirrors read the same value
			for _, mirrorAddr := range tp.mirrorAddrs {
				result := mem.Read(mirrorAddr)
				if result != value {
					t.Errorf("Mirror read: Read(%04X) = %02X, want %02X", mirrorAddr, result, value)
				}
			}

			// Write to first mirror
			newValue := uint8(value + 1)
			mem.Write(tp.mirrorAddrs[0], newValue)

			// Base address should now have new value
			result := mem.Read(tp.baseAddr)
			if result != newValue {
				t.Errorf("After mirror write: Read(%04X) = %02X, want %02X", tp.baseAddr, result, newValue)
			}

			// All other mirrors should also have new value
			for i, mirrorAddr := range tp.mirrorAddrs {
				result := mem.Read(mirrorAddr)
				if result != newValue {
					t.Errorf("Mirror %d after write: Read(%04X) = %02X, want %02X", i, mirrorAddr, result, newValue)
				}
			}
		})
	}
}

func TestCPUMemoryMirroring_PPURegisters(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test PPU register mirroring every 8 bytes in $2000-$3FFF range
	baseRegisters := []uint16{0x2000, 0x2001, 0x2002, 0x2003, 0x2004, 0x2005, 0x2006, 0x2007}

	for _, baseReg := range baseRegisters {
		t.Run("PPU register mirroring", func(t *testing.T) {
			value := uint8(baseReg & 0xFF)

			// Test mirrors throughout the PPU register space
			for mirrorAddr := baseReg + 8; mirrorAddr <= 0x3FFF; mirrorAddr += 8 {
				// Clear previous calls
				ppu.writeCalls = nil
				ppu.readCalls = nil

				// Write to mirror address
				mem.Write(mirrorAddr, value)

				// Verify PPU was called with base register address
				if len(ppu.writeCalls) == 0 {
					t.Errorf("PPU WriteRegister not called for mirror %04X", mirrorAddr)
					continue
				}

				call := ppu.writeCalls[0]
				if call.Address != baseReg {
					t.Errorf("Mirror %04X: PPU called with %04X, want %04X",
						mirrorAddr, call.Address, baseReg)
				}
				if call.Value != value {
					t.Errorf("Mirror %04X: PPU called with value %02X, want %02X",
						mirrorAddr, call.Value, value)
				}

				// Test read from mirror
				ppu.registers[baseReg&0x7] = value
				result := mem.Read(mirrorAddr)

				// Verify PPU was called with base register address
				if len(ppu.readCalls) == 0 {
					t.Errorf("PPU ReadRegister not called for mirror %04X", mirrorAddr)
					continue
				}

				readCall := ppu.readCalls[len(ppu.readCalls)-1]
				if readCall != baseReg {
					t.Errorf("Mirror read %04X: PPU called with %04X, want %04X",
						mirrorAddr, readCall, baseReg)
				}

				if result != value {
					t.Errorf("Mirror read %04X: got %02X, want %02X", mirrorAddr, result, value)
				}
			}
		})
	}
}

func TestCPUMemoryMirroring_EdgeCases(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test edge cases in mirroring
	edgeCases := []struct {
		name             string
		address          uint16
		expectedBehavior string
	}{
		{"RAM end to mirror start", 0x07FF, "RAM mirrors at 0x0FFF, 0x17FF, 0x1FFF"},
		{"RAM mirror end", 0x1FFF, "Maps to RAM 0x07FF"},
		{"PPU reg end", 0x2007, "Mirrors at 0x200F, 0x2017, etc."},
		{"PPU space end", 0x3FFF, "Maps to PPUDATA (0x2007)"},
	}

	for _, ec := range edgeCases {
		t.Run(ec.name, func(t *testing.T) {
			value := uint8(0x42)

			// Should not panic
			mem.Write(ec.address, value)
			result := mem.Read(ec.address)
			_ = result
		})
	}
}

func TestPPUMemoryMirroring_Nametables(t *testing.T) {
	cart := &MockCartridge{}

	mirrorModes := []struct {
		mode  MirrorMode
		name  string
		tests []nametableMirrorTest
	}{
		{
			mode: MirrorHorizontal,
			name: "Horizontal Mirroring",
			tests: []nametableMirrorTest{
				{0x2000, 0x2400, "NT0 and NT1"},
				{0x2800, 0x2C00, "NT2 and NT3"},
				{0x2100, 0x2500, "NT0 and NT1 offset"},
				{0x2900, 0x2D00, "NT2 and NT3 offset"},
			},
		},
		{
			mode: MirrorVertical,
			name: "Vertical Mirroring",
			tests: []nametableMirrorTest{
				{0x2000, 0x2800, "NT0 and NT2"},
				{0x2400, 0x2C00, "NT1 and NT3"},
				{0x2100, 0x2900, "NT0 and NT2 offset"},
				{0x2500, 0x2D00, "NT1 and NT3 offset"},
			},
		},
		{
			mode: MirrorSingleScreen0,
			name: "Single Screen 0",
			tests: []nametableMirrorTest{
				{0x2000, 0x2400, "All to screen 0"},
				{0x2000, 0x2800, "All to screen 0"},
				{0x2000, 0x2C00, "All to screen 0"},
				{0x2400, 0x2800, "All to screen 0"},
			},
		},
		{
			mode: MirrorSingleScreen1,
			name: "Single Screen 1",
			tests: []nametableMirrorTest{
				{0x2000, 0x2400, "All to screen 1"},
				{0x2000, 0x2800, "All to screen 1"},
				{0x2000, 0x2C00, "All to screen 1"},
				{0x2400, 0x2800, "All to screen 1"},
			},
		},
	}

	for _, mm := range mirrorModes {
		t.Run(mm.name, func(t *testing.T) {
			ppu := NewPPUMemory(cart, mm.mode)

			for _, test := range mm.tests {
				t.Run(test.name, func(t *testing.T) {
					value := uint8(0x55)

					// Write to first address
					ppu.Write(test.addr1, value)

					// Read from both addresses
					result1 := ppu.Read(test.addr1)
					result2 := ppu.Read(test.addr2)

					if result1 != value {
						t.Errorf("Read(%04X) = %02X, want %02X", test.addr1, result1, value)
					}

					if mm.mode == MirrorFourScreen {
						// In four-screen mode, addresses should be independent
						if result2 == value {
							t.Errorf("Four-screen mode: Read(%04X) = %02X, should be independent", test.addr2, result2)
						}
					} else {
						// In other modes, addresses should mirror
						if result2 != value {
							t.Errorf("Mirrored Read(%04X) = %02X, want %02X", test.addr2, result2, value)
						}
					}

					// Write to second address
					newValue := uint8(0x77)
					ppu.Write(test.addr2, newValue)

					result1 = ppu.Read(test.addr1)
					result2 = ppu.Read(test.addr2)

					if mm.mode == MirrorFourScreen {
						// In four-screen mode, first address should be unchanged
						if result1 != value {
							t.Errorf("Four-screen mode: Read(%04X) = %02X, want %02X (unchanged)", test.addr1, result1, value)
						}
						if result2 != newValue {
							t.Errorf("Four-screen mode: Read(%04X) = %02X, want %02X", test.addr2, result2, newValue)
						}
					} else {
						// In other modes, both should have new value
						if result1 != newValue {
							t.Errorf("After mirror write: Read(%04X) = %02X, want %02X", test.addr1, result1, newValue)
						}
						if result2 != newValue {
							t.Errorf("After mirror write: Read(%04X) = %02X, want %02X", test.addr2, result2, newValue)
						}
					}
				})
			}
		})
	}
}

type nametableMirrorTest struct {
	addr1 uint16
	addr2 uint16
	name  string
}

func TestPPUMemoryMirroring_NametableToMirror(t *testing.T) {
	cart := &MockCartridge{}
	ppu := NewPPUMemory(cart, MirrorHorizontal)

	// Test nametable mirroring from $2000-$2FFF to $3000-$3EFF
	nametableMirrors := []struct {
		baseAddr   uint16
		mirrorAddr uint16
		name       string
	}{
		{0x2000, 0x3000, "Nametable 0 start"},
		{0x23FF, 0x33FF, "Nametable 0 end"},
		{0x2400, 0x3400, "Nametable 1 start"},
		{0x27FF, 0x37FF, "Nametable 1 end"},
		{0x2800, 0x3800, "Nametable 2 start"},
		{0x2BFF, 0x3BFF, "Nametable 2 end"},
		{0x2C00, 0x3C00, "Nametable 3 start"},
		{0x2EFF, 0x3EFF, "Nametable 3 end"},
	}

	for _, nm := range nametableMirrors {
		t.Run(nm.name, func(t *testing.T) {
			value := uint8(nm.baseAddr & 0xFF)

			// Write to base nametable
			ppu.Write(nm.baseAddr, value)

			// Read from mirror should return same value
			result := ppu.Read(nm.mirrorAddr)
			if result != value {
				t.Errorf("Mirror read: Read(%04X) = %02X, want %02X", nm.mirrorAddr, result, value)
			}

			// Write to mirror
			newValue := uint8(value + 1)
			ppu.Write(nm.mirrorAddr, newValue)

			// Base should now have new value
			result = ppu.Read(nm.baseAddr)
			if result != newValue {
				t.Errorf("After mirror write: Read(%04X) = %02X, want %02X", nm.baseAddr, result, newValue)
			}
		})
	}
}

func TestPPUMemoryMirroring_Palette(t *testing.T) {
	cart := &MockCartridge{}
	ppu := NewPPUMemory(cart, MirrorHorizontal)

	// Test palette mirroring every 32 bytes
	paletteAddresses := []uint16{
		0x3F00, 0x3F01, 0x3F02, 0x3F03, // Background palette 0
		0x3F10, 0x3F11, 0x3F12, 0x3F13, // Sprite palette 0
		0x3F1F, // Last palette address
	}

	for _, baseAddr := range paletteAddresses {
		t.Run("Palette mirror", func(t *testing.T) {
			value := uint8(baseAddr & 0xFF)

			// Write to base address
			ppu.Write(baseAddr, value)

			// Test all mirrors up to $3FFF
			for mirrorAddr := baseAddr + 0x20; mirrorAddr <= 0x3FFF; mirrorAddr += 0x20 {
				result := ppu.Read(mirrorAddr)
				if result != value {
					t.Errorf("Palette mirror: Read(%04X) = %02X, want %02X", mirrorAddr, result, value)
				}

				// Write to mirror
				newValue := uint8(value + 1)
				ppu.Write(mirrorAddr, newValue)

				// Base should have new value
				result = ppu.Read(baseAddr)
				if result != newValue {
					t.Errorf("After palette mirror write: Read(%04X) = %02X, want %02X", baseAddr, result, newValue)
				}

				// Reset for next test
				ppu.Write(baseAddr, value)
			}
		})
	}
}

func TestPPUMemoryMirroring_PaletteBackgroundColors(t *testing.T) {
	cart := &MockCartridge{}
	ppu := NewPPUMemory(cart, MirrorHorizontal)

	// Test special background color mirroring
	// $3F10, $3F14, $3F18, $3F1C mirror $3F00, $3F04, $3F08, $3F0C
	backgroundMirrors := []struct {
		bgAddr     uint16
		spriteAddr uint16
		name       string
	}{
		{0x3F00, 0x3F10, "Universal background color"},
		{0x3F04, 0x3F14, "Background palette 1 color 0"},
		{0x3F08, 0x3F18, "Background palette 2 color 0"},
		{0x3F0C, 0x3F1C, "Background palette 3 color 0"},
	}

	for _, bm := range backgroundMirrors {
		t.Run(bm.name, func(t *testing.T) {
			value := uint8(0x25)

			// Write to background palette
			ppu.Write(bm.bgAddr, value)

			// Sprite palette should mirror
			result := ppu.Read(bm.spriteAddr)
			if result != value {
				t.Errorf("Background mirror: Read(%04X) = %02X, want %02X", bm.spriteAddr, result, value)
			}

			// Write to sprite palette
			newValue := uint8(0x36)
			ppu.Write(bm.spriteAddr, newValue)

			// Background palette should have new value
			result = ppu.Read(bm.bgAddr)
			if result != newValue {
				t.Errorf("After sprite write: Read(%04X) = %02X, want %02X", bm.bgAddr, result, newValue)
			}

			// Both addresses should continue to mirror
			ppu.Write(bm.bgAddr, 0x47)
			result = ppu.Read(bm.spriteAddr)
			if result != 0x47 {
				t.Errorf("Continued mirroring: Read(%04X) = %02X, want 47", bm.spriteAddr, result)
			}
		})
	}
}

func TestMirroring_ComprehensivePatterns(t *testing.T) {
	// Test complex mirroring patterns with multiple writes
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Fill RAM with a pattern
	for i := 0; i < 0x800; i++ {
		mem.ram[i] = uint8(i & 0xFF)
	}

	// Test that the pattern is correctly mirrored
	for mirror := 1; mirror < 4; mirror++ {
		baseAddr := uint16(mirror * 0x800)

		for offset := 0; offset < 0x800; offset++ {
			addr := baseAddr + uint16(offset)
			expected := uint8(offset & 0xFF)
			result := mem.Read(addr)

			if result != expected {
				t.Errorf("Pattern mirror %d: Read(%04X) = %02X, want %02X",
					mirror, addr, result, expected)
				break // Avoid too many errors
			}
		}
	}
}

func TestMirroring_StressTest(t *testing.T) {
	// Stress test with rapid mirroring access
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Rapidly alternate between base and mirrored addresses
	for i := 0; i < 1000; i++ {
		baseAddr := uint16(i % 0x800)
		mirrorAddr := baseAddr + 0x800
		value := uint8(i & 0xFF)

		if i%2 == 0 {
			mem.Write(baseAddr, value)
			result := mem.Read(mirrorAddr)
			if result != value {
				t.Errorf("Stress test %d: mirror read failed", i)
			}
		} else {
			mem.Write(mirrorAddr, value)
			result := mem.Read(baseAddr)
			if result != value {
				t.Errorf("Stress test %d: base read failed", i)
			}
		}
	}
}
