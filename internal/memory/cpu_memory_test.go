package memory

import (
	"testing"
)

// TestCPUMemoryMap tests the complete CPU memory mapping ($0000-$FFFF)
func TestCPUMemoryMap_InternalRAM(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test internal RAM region ($0000-$07FF)
	t.Run("Internal RAM region", func(t *testing.T) {
		for addr := uint16(0x0000); addr <= 0x07FF; addr++ {
			value := uint8(addr & 0xFF)
			mem.Write(addr, value)

			result := mem.Read(addr)
			if result != value {
				t.Errorf("RAM[%04X] = %02X, want %02X", addr, result, value)
			}
		}
	})

	// Test all three mirrors of internal RAM
	mirrorTests := []struct {
		name      string
		startAddr uint16
		endAddr   uint16
	}{
		{"First mirror", 0x0800, 0x0FFF},
		{"Second mirror", 0x1000, 0x17FF},
		{"Third mirror", 0x1800, 0x1FFF},
	}

	for _, mt := range mirrorTests {
		t.Run(mt.name, func(t *testing.T) {
			// Write a pattern to base RAM
			for i := uint16(0); i < 0x800; i++ {
				mem.ram[i] = uint8(i & 0xFF)
			}

			// Read through mirror and verify
			for addr := mt.startAddr; addr <= mt.endAddr; addr++ {
				expected := uint8((addr & 0x7FF) & 0xFF)
				result := mem.Read(addr)
				if result != expected {
					t.Errorf("Mirror read[%04X] = %02X, want %02X", addr, result, expected)
				}
			}

			// Test write through mirror
			for addr := mt.startAddr; addr <= mt.endAddr && addr < mt.startAddr+0x100; addr++ {
				value := uint8(0xAA)
				mem.Write(addr, value)

				// Verify base RAM was modified
				baseAddr := addr & 0x7FF
				if mem.ram[baseAddr] != value {
					t.Errorf("Mirror write[%04X] did not update RAM[%04X]", addr, baseAddr)
				}
			}
		})
	}
}

func TestCPUMemoryMap_PPURegisters(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test base PPU registers ($2000-$2007)
	baseRegisters := []struct {
		addr uint16
		name string
	}{
		{0x2000, "PPUCTRL"},
		{0x2001, "PPUMASK"},
		{0x2002, "PPUSTATUS"},
		{0x2003, "OAMADDR"},
		{0x2004, "OAMDATA"},
		{0x2005, "PPUSCROLL"},
		{0x2006, "PPUADDR"},
		{0x2007, "PPUDATA"},
	}

	for _, reg := range baseRegisters {
		t.Run(reg.name, func(t *testing.T) {
			value := uint8(0x42)
			mem.Write(reg.addr, value)

			// Verify PPU was called with correct address
			if len(ppu.writeCalls) == 0 {
				t.Fatal("PPU WriteRegister not called")
			}

			lastCall := ppu.writeCalls[len(ppu.writeCalls)-1]
			if lastCall.Address != reg.addr {
				t.Errorf("PPU called with %04X, want %04X", lastCall.Address, reg.addr)
			}
		})
	}

	// Test PPU register mirroring ($2008-$3FFF)
	t.Run("PPU register mirroring", func(t *testing.T) {
		// Test every 8th address mirrors the base registers
		for baseAddr := uint16(0x2000); baseAddr <= 0x2007; baseAddr++ {
			expectedRegister := baseAddr

			// Test mirrors every 8 bytes up to $3FFF
			for mirrorAddr := baseAddr + 8; mirrorAddr <= 0x3FFF; mirrorAddr += 8 {
				// Clear previous calls
				ppu.writeCalls = nil

				value := uint8(mirrorAddr & 0xFF)
				mem.Write(mirrorAddr, value)

				if len(ppu.writeCalls) == 0 {
					t.Errorf("No PPU call for mirror address %04X", mirrorAddr)
					continue
				}

				call := ppu.writeCalls[0]
				if call.Address != expectedRegister {
					t.Errorf("Mirror %04X called PPU with %04X, want %04X",
						mirrorAddr, call.Address, expectedRegister)
				}
				if call.Value != value {
					t.Errorf("Mirror %04X called PPU with value %02X, want %02X",
						mirrorAddr, call.Value, value)
				}
			}
		}
	})
}

func TestCPUMemoryMap_APUAndIORegisters(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test APU sound registers ($4000-$4017)
	apuRegisters := []struct {
		addr uint16
		name string
	}{
		{0x4000, "APU_PULSE1_VOL"},
		{0x4001, "APU_PULSE1_SWEEP"},
		{0x4002, "APU_PULSE1_TIMER_LOW"},
		{0x4003, "APU_PULSE1_TIMER_HIGH"},
		{0x4004, "APU_PULSE2_VOL"},
		{0x4005, "APU_PULSE2_SWEEP"},
		{0x4006, "APU_PULSE2_TIMER_LOW"},
		{0x4007, "APU_PULSE2_TIMER_HIGH"},
		{0x4008, "APU_TRIANGLE_LINEAR"},
		{0x4009, "APU_TRIANGLE_UNUSED"},
		{0x400A, "APU_TRIANGLE_TIMER_LOW"},
		{0x400B, "APU_TRIANGLE_TIMER_HIGH"},
		{0x400C, "APU_NOISE_VOL"},
		{0x400D, "APU_NOISE_UNUSED"},
		{0x400E, "APU_NOISE_PERIOD"},
		{0x400F, "APU_NOISE_LENGTH"},
		{0x4010, "APU_DMC_CONTROL"},
		{0x4011, "APU_DMC_VALUE"},
		{0x4012, "APU_DMC_ADDRESS"},
		{0x4013, "APU_DMC_LENGTH"},
		{0x4015, "APU_STATUS"},
		{0x4017, "APU_FRAME_COUNTER"},
	}

	for _, reg := range apuRegisters {
		t.Run(reg.name, func(t *testing.T) {
			value := uint8(0x33)
			mem.Write(reg.addr, value)

			// Verify APU was called
			if len(apu.writeCalls) == 0 {
				t.Fatal("APU WriteRegister not called")
			}

			lastCall := apu.writeCalls[len(apu.writeCalls)-1]
			if lastCall.Address != reg.addr {
				t.Errorf("APU called with %04X, want %04X", lastCall.Address, reg.addr)
			}
			if lastCall.Value != value {
				t.Errorf("APU called with value %02X, want %02X", lastCall.Value, value)
			}
		})
	}

	// Test OAM DMA register ($4014)
	t.Run("OAM DMA register", func(t *testing.T) {
		// Set up data in RAM page 2
		for i := 0; i < 256; i++ {
			mem.ram[0x200+i] = uint8(i + 1)
		}

		// Trigger DMA
		mem.Write(0x4014, 0x02)

		// Should have triggered 256 OAM writes
		if len(ppu.writeCalls) != 256 {
			t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
		}

		// Verify each write
		for i := 0; i < 256; i++ {
			call := ppu.writeCalls[i]
			if call.Address != 0x2004 {
				t.Errorf("OAM write %d: address %04X, want 2004", i, call.Address)
			}
			expectedValue := uint8(i + 1)
			if call.Value != expectedValue {
				t.Errorf("OAM write %d: value %02X, want %02X", i, call.Value, expectedValue)
			}
		}
	})

	// Test controller registers ($4016-$4017)
	t.Run("Controller registers read", func(t *testing.T) {
		// These should return 0 for now (TODO: implement controller reading)
		result1 := mem.Read(0x4016)
		result2 := mem.Read(0x4017)

		if result1 != 0 {
			t.Errorf("Read(4016) = %02X, want 0", result1)
		}
		if result2 != 0 {
			t.Errorf("Read(4017) = %02X, want 0", result2)
		}
	})
}

func TestCPUMemoryMap_TestMode(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test mode registers ($4018-$401F) should return 0 and ignore writes
	for addr := uint16(0x4018); addr <= 0x401F; addr++ {
		t.Run("Test mode read", func(t *testing.T) {
			result := mem.Read(addr)
			if result != 0 {
				t.Errorf("Read(%04X) = %02X, want 0", addr, result)
			}
		})

		t.Run("Test mode write", func(t *testing.T) {
			// Should not panic
			mem.Write(addr, 0xFF)
		})
	}
}

func TestCPUMemoryMap_CartridgeSpace(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up cartridge data
	cart.prgData[0x0000] = 0x12 // $8000
	cart.prgData[0x4000] = 0x34 // $C000
	cart.prgData[0x7FFF] = 0x56 // $FFFF

	cartridgeTests := []struct {
		name           string
		startAddr      uint16
		endAddr        uint16
		testAddresses  []uint16
		expectedValues []uint8
	}{
		{
			name:           "PRG ROM lower bank",
			startAddr:      0x8000,
			endAddr:        0xBFFF,
			testAddresses:  []uint16{0x8000, 0x9000, 0xBFFF},
			expectedValues: []uint8{0x12, 0x00, 0x00},
		},
		{
			name:           "PRG ROM upper bank",
			startAddr:      0xC000,
			endAddr:        0xFFFF,
			testAddresses:  []uint16{0xC000, 0xE000, 0xFFFF},
			expectedValues: []uint8{0x34, 0x00, 0x56},
		},
	}

	for _, ct := range cartridgeTests {
		t.Run(ct.name, func(t *testing.T) {
			for i, addr := range ct.testAddresses {
				result := mem.Read(addr)
				expected := ct.expectedValues[i]

				if result != expected {
					t.Errorf("Read(%04X) = %02X, want %02X", addr, result, expected)
				}

				// Verify cartridge was called
				if len(cart.prgReads) == 0 {
					t.Fatal("Cartridge ReadPRG not called")
				}

				lastCall := cart.prgReads[len(cart.prgReads)-1]
				if lastCall != addr {
					t.Errorf("Cartridge ReadPRG called with %04X, want %04X", lastCall, addr)
				}
			}
		})
	}

	// Test cartridge expansion area ($4020-$7FFF)
	t.Run("Cartridge expansion area", func(t *testing.T) {
		expansionAddresses := []uint16{0x4020, 0x5000, 0x6000, 0x7FFF}

		for _, addr := range expansionAddresses {
			// Reads should return 0 (unmapped)
			result := mem.Read(addr)
			if result != 0 {
				t.Errorf("Read(%04X) = %02X, want 0 (unmapped)", addr, result)
			}

			// Writes should not panic
			mem.Write(addr, 0xFF)
		}
	})
}

func TestCPUMemoryMap_ComprehensiveAccess(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test key addresses across the entire memory map
	testAddresses := []struct {
		addr      uint16
		region    string
		writeable bool
	}{
		{0x0000, "RAM start", true},
		{0x07FF, "RAM end", true},
		{0x0800, "RAM mirror 1", true},
		{0x1FFF, "RAM mirror 3 end", true},
		{0x2000, "PPU CTRL", true},
		{0x2007, "PPU DATA", true},
		{0x2008, "PPU mirror", true},
		{0x3FFF, "PPU mirror end", true},
		{0x4000, "APU start", true},
		{0x4014, "OAM DMA", true},
		{0x4017, "APU end", true},
		{0x4018, "Test mode", false},
		{0x401F, "Test mode end", false},
		{0x4020, "Expansion start", false},
		{0x7FFF, "Expansion end", false},
		{0x8000, "PRG ROM start", true},
		{0xFFFF, "PRG ROM end", true},
	}

	for _, ta := range testAddresses {
		t.Run(ta.region+" access", func(t *testing.T) {
			// Test read (should not panic)
			result := mem.Read(ta.addr)
			_ = result // Value depends on region

			// Test write (should not panic)
			mem.Write(ta.addr, 0x42)
		})
	}
}

func TestCPUMemoryMap_EdgeCases(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test boundary conditions
	boundaries := []struct {
		name     string
		addr     uint16
		nextAddr uint16
		region1  string
		region2  string
	}{
		{"RAM to RAM mirror", 0x07FF, 0x0800, "RAM", "RAM mirror"},
		{"RAM mirror to PPU", 0x1FFF, 0x2000, "RAM mirror", "PPU"},
		{"PPU to APU", 0x3FFF, 0x4000, "PPU", "APU"},
		{"APU to test mode", 0x4017, 0x4018, "APU", "Test mode"},
		{"Test mode to expansion", 0x401F, 0x4020, "Test mode", "Expansion"},
		{"Expansion to PRG ROM", 0x7FFF, 0x8000, "Expansion", "PRG ROM"},
	}

	for _, b := range boundaries {
		t.Run(b.name, func(t *testing.T) {
			// Access both sides of boundary
			result1 := mem.Read(b.addr)
			result2 := mem.Read(b.nextAddr)

			// Should not panic and may return different values
			_ = result1
			_ = result2

			// Test writes
			mem.Write(b.addr, 0xAA)
			mem.Write(b.nextAddr, 0x55)
		})
	}

	// Test wraparound at top of address space
	t.Run("Address wraparound", func(t *testing.T) {
		// These should all access PRG ROM space
		addresses := []uint16{0x8000, 0xC000, 0xFFFF}

		for _, addr := range addresses {
			result := mem.Read(addr)
			_ = result
			mem.Write(addr, 0x99)

			// Verify cartridge was accessed
			if len(cart.prgReads) == 0 {
				t.Errorf("Cartridge not accessed for address %04X", addr)
			}
		}
	})
}
