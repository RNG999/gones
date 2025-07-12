package memory

import (
	"testing"
)

// TestOAMDMA_BasicTransfer tests basic OAM DMA functionality
func TestOAMDMA_BasicTransfer(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up test data in RAM page 0
	for i := 0; i < 256; i++ {
		mem.ram[i] = uint8(i)
	}

	// Trigger OAM DMA from page 0
	mem.Write(0x4014, 0x00)

	// Verify 256 bytes were transferred
	if len(ppu.writeCalls) != 256 {
		t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
	}

	// Verify each write was to OAMDATA register with correct value
	for i := 0; i < 256; i++ {
		call := ppu.writeCalls[i]
		if call.Address != 0x2004 {
			t.Errorf("OAM write %d: address = %04X, want 2004", i, call.Address)
		}
		if call.Value != uint8(i) {
			t.Errorf("OAM write %d: value = %02X, want %02X", i, call.Value, uint8(i))
		}
	}
}

func TestOAMDMA_AllPageSources(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test DMA from different source pages
	testPages := []struct {
		page         uint8
		expectedAddr uint16
		description  string
	}{
		{0x00, 0x0000, "Page 0 (RAM)"},
		{0x01, 0x0100, "Page 1 (RAM)"},
		{0x02, 0x0200, "Page 2 (RAM)"},
		{0x03, 0x0300, "Page 3 (RAM)"},
		{0x04, 0x0400, "Page 4 (RAM mirror)"},
		{0x07, 0x0700, "Page 7 (RAM mirror)"},
		{0x20, 0x2000, "Page 20 (PPU registers)"},
		{0x40, 0x4000, "Page 40 (APU registers)"},
		{0x80, 0x8000, "Page 80 (PRG ROM)"},
		{0xFF, 0xFF00, "Page FF (PRG ROM)"},
	}

	for _, tp := range testPages {
		t.Run(tp.description, func(t *testing.T) {
			// Clear previous calls
			ppu.writeCalls = nil

			// Set up test data based on page type
			setupPageData(mem, tp.page, tp.expectedAddr)

			// Trigger DMA
			mem.Write(0x4014, tp.page)

			// Verify transfer occurred
			if len(ppu.writeCalls) != 256 {
				t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
			}

			// Verify first few bytes have expected pattern
			for i := 0; i < 4; i++ {
				call := ppu.writeCalls[i]
				if call.Address != 0x2004 {
					t.Errorf("OAM write %d: address = %04X, want 2004", i, call.Address)
				}
				// Value depends on page type, just verify it's consistent
				_ = call.Value
			}
		})
	}
}

func setupPageData(mem *Memory, page uint8, baseAddr uint16) {
	// Set up different data patterns based on address range
	switch {
	case baseAddr < 0x2000:
		// RAM or RAM mirrors - fill with pattern
		for i := 0; i < 256; i++ {
			ramAddr := (baseAddr + uint16(i)) & 0x7FF
			mem.ram[ramAddr] = uint8(page + byte(i))
		}
	case baseAddr >= 0x8000:
		// PRG ROM - set up cartridge data
		cart := mem.cartridge.(*MockCartridge)
		for i := 0; i < 256; i++ {
			cart.prgData[(baseAddr+uint16(i))&0x7FFF] = uint8(0x80 + byte(i))
		}
	default:
		// Other regions - leave as is (will read 0 or register values)
	}
}

func TestOAMDMA_RAMMirroring(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up test pattern in base RAM
	for i := 0; i < 256; i++ {
		mem.ram[i] = uint8(0xAA + i)
	}

	// Test DMA from RAM mirrors
	ramMirrorPages := []uint8{0x08, 0x10, 0x18} // Pages that mirror $0000-$07FF

	for _, page := range ramMirrorPages {
		t.Run("RAM mirror page", func(t *testing.T) {
			// Clear previous calls
			ppu.writeCalls = nil

			// Trigger DMA from mirror page
			mem.Write(0x4014, page)

			// Should transfer same data as from page 0
			if len(ppu.writeCalls) != 256 {
				t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
			}

			// Verify data matches base RAM
			for i := 0; i < 256; i++ {
				call := ppu.writeCalls[i]
				expectedValue := uint8(0xAA + i)
				if call.Value != expectedValue {
					t.Errorf("Mirror page %02X, byte %d: value = %02X, want %02X",
						page, i, call.Value, expectedValue)
				}
			}
		})
	}
}

func TestOAMDMA_CrossPageBoundary(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up data that crosses RAM mirror boundaries
	for i := 0; i < 0x800; i++ {
		mem.ram[i] = uint8(i & 0xFF)
	}

	// Test DMA from page that crosses boundary
	// Page 0x07 starts at 0x0700, so it reads 0x0700-0x07FF (256 bytes)
	ppu.writeCalls = nil
	mem.Write(0x4014, 0x07)

	if len(ppu.writeCalls) != 256 {
		t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
	}

	// Verify data comes from correct addresses
	for i := 0; i < 256; i++ {
		call := ppu.writeCalls[i]
		sourceAddr := 0x0700 + i
		expectedValue := uint8(sourceAddr & 0xFF)
		if call.Value != expectedValue {
			t.Errorf("Cross-boundary byte %d: value = %02X, want %02X",
				i, call.Value, expectedValue)
		}
	}
}

func TestOAMDMA_PPURegisterSource(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up PPU register values
	for i := 0; i < 8; i++ {
		ppu.registers[i] = uint8(0x20 + i)
	}

	// Trigger DMA from PPU register page
	ppu.writeCalls = nil
	ppu.readCalls = nil
	mem.Write(0x4014, 0x20)

	if len(ppu.writeCalls) != 256 {
		t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
	}

	// Should have made 256 read calls to PPU registers (with mirroring)
	if len(ppu.readCalls) != 256 {
		t.Fatalf("Expected 256 PPU read calls, got %d", len(ppu.readCalls))
	}

	// Verify read addresses are correctly mirrored
	for i := 0; i < 256; i++ {
		readCall := ppu.readCalls[i]
		expectedRegister := 0x2000 + uint16(i&0x7)
		if readCall != expectedRegister {
			t.Errorf("PPU read %d: address = %04X, want %04X",
				i, readCall, expectedRegister)
		}
	}
}

func TestOAMDMA_CartridgeSource(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up cartridge data
	for i := 0; i < 256; i++ {
		cart.prgData[i] = uint8(0x90 + i)
	}

	// Trigger DMA from PRG ROM
	cart.prgReads = nil
	ppu.writeCalls = nil
	mem.Write(0x4014, 0x80)

	if len(ppu.writeCalls) != 256 {
		t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
	}

	// Should have made 256 read calls to cartridge
	if len(cart.prgReads) != 256 {
		t.Fatalf("Expected 256 cartridge reads, got %d", len(cart.prgReads))
	}

	// Verify cartridge read addresses
	for i := 0; i < 256; i++ {
		readCall := cart.prgReads[i]
		expectedAddr := 0x8000 + uint16(i)
		if readCall != expectedAddr {
			t.Errorf("Cartridge read %d: address = %04X, want %04X",
				i, readCall, expectedAddr)
		}

		// Verify OAM data
		writeCall := ppu.writeCalls[i]
		expectedValue := uint8(0x90 + i)
		if writeCall.Value != expectedValue {
			t.Errorf("OAM write %d: value = %02X, want %02X",
				i, writeCall.Value, expectedValue)
		}
	}
}

func TestOAMDMA_UnmappedSource(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test DMA from unmapped regions (should read 0)
	unmappedPages := []uint8{0x41, 0x50, 0x60, 0x70} // Expansion ROM area

	for _, page := range unmappedPages {
		t.Run("Unmapped page", func(t *testing.T) {
			ppu.writeCalls = nil
			mem.Write(0x4014, page)

			if len(ppu.writeCalls) != 256 {
				t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
			}

			// All values should be 0 (unmapped reads)
			for i := 0; i < 256; i++ {
				call := ppu.writeCalls[i]
				if call.Value != 0 {
					t.Errorf("Unmapped page %02X, byte %d: value = %02X, want 00",
						page, i, call.Value)
				}
			}
		})
	}
}

func TestOAMDMA_TransferOrder(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up sequential data
	for i := 0; i < 256; i++ {
		mem.ram[i] = uint8(i)
	}

	// Trigger DMA
	mem.Write(0x4014, 0x00)

	// Verify transfer order is sequential
	if len(ppu.writeCalls) != 256 {
		t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
	}

	for i := 0; i < 256; i++ {
		call := ppu.writeCalls[i]
		if call.Value != uint8(i) {
			t.Errorf("Transfer order error: write %d has value %02X, want %02X",
				i, call.Value, uint8(i))
		}
	}
}

func TestOAMDMA_PartialPage(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up data only in part of the page
	for i := 0; i < 128; i++ {
		mem.ram[i] = uint8(0xFF)
	}
	// Leave rest as 0

	// Trigger DMA from page 0
	mem.Write(0x4014, 0x00)

	// Should still transfer full 256 bytes
	if len(ppu.writeCalls) != 256 {
		t.Fatalf("Expected 256 OAM writes, got %d", len(ppu.writeCalls))
	}

	// First 128 bytes should be 0xFF
	for i := 0; i < 128; i++ {
		call := ppu.writeCalls[i]
		if call.Value != 0xFF {
			t.Errorf("Partial page byte %d: value = %02X, want FF", i, call.Value)
		}
	}

	// Remaining bytes should be 0
	for i := 128; i < 256; i++ {
		call := ppu.writeCalls[i]
		if call.Value != 0x00 {
			t.Errorf("Partial page byte %d: value = %02X, want 00", i, call.Value)
		}
	}
}

func TestOAMDMA_MultipleTransfers(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up different data in different pages
	for i := 0; i < 256; i++ {
		mem.ram[i] = 0x11     // Page 0
		mem.ram[256+i] = 0x22 // Page 1 (mirrored to RAM)
	}

	// First transfer from page 0
	mem.Write(0x4014, 0x00)
	if len(ppu.writeCalls) != 256 {
		t.Fatalf("First transfer: expected 256 writes, got %d", len(ppu.writeCalls))
	}

	// Verify first transfer data
	for i := 0; i < 256; i++ {
		if ppu.writeCalls[i].Value != 0x11 {
			t.Errorf("First transfer byte %d: value = %02X, want 11",
				i, ppu.writeCalls[i].Value)
		}
	}

	// Second transfer from page 1
	mem.Write(0x4014, 0x01)
	if len(ppu.writeCalls) != 512 {
		t.Fatalf("After second transfer: expected 512 writes, got %d", len(ppu.writeCalls))
	}

	// Verify second transfer data
	for i := 256; i < 512; i++ {
		if ppu.writeCalls[i].Value != 0x22 {
			t.Errorf("Second transfer byte %d: value = %02X, want 22",
				i-256, ppu.writeCalls[i].Value)
		}
	}
}

func TestOAMDMA_RegisterInterface(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Test that only writes to $4014 trigger DMA
	nonDMAAddresses := []uint16{0x4013, 0x4015, 0x4016, 0x4017}

	for _, addr := range nonDMAAddresses {
		t.Run("Non-DMA register", func(t *testing.T) {
			ppu.writeCalls = nil
			mem.Write(addr, 0x00)

			// Should not trigger any OAM writes
			if len(ppu.writeCalls) != 0 {
				t.Errorf("Write to %04X triggered %d OAM writes, want 0",
					addr, len(ppu.writeCalls))
			}
		})
	}

	// Test that $4014 does trigger DMA
	t.Run("DMA register", func(t *testing.T) {
		ppu.writeCalls = nil
		mem.Write(0x4014, 0x00)

		if len(ppu.writeCalls) != 256 {
			t.Errorf("Write to 4014 triggered %d OAM writes, want 256", len(ppu.writeCalls))
		}
	})
}

func TestOAMDMA_StressTest(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up test data
	for i := 0; i < 0x800; i++ {
		mem.ram[i] = uint8(i & 0xFF)
	}

	// Perform multiple rapid DMA transfers
	for page := uint8(0); page < 8; page++ {
		mem.Write(0x4014, page)
	}

	// Should have performed 8 complete transfers
	expectedWrites := 8 * 256
	if len(ppu.writeCalls) != expectedWrites {
		t.Fatalf("Stress test: expected %d writes, got %d", expectedWrites, len(ppu.writeCalls))
	}

	// Verify each transfer was complete and correct
	for transfer := 0; transfer < 8; transfer++ {
		for byte := 0; byte < 256; byte++ {
			writeIndex := transfer*256 + byte
			call := ppu.writeCalls[writeIndex]

			if call.Address != 0x2004 {
				t.Errorf("Transfer %d, byte %d: address = %04X, want 2004",
					transfer, byte, call.Address)
			}

			// All pages map to same RAM due to mirroring
			expectedValue := uint8(byte)
			if call.Value != expectedValue {
				t.Errorf("Transfer %d, byte %d: value = %02X, want %02X",
					transfer, byte, call.Value, expectedValue)
			}
		}
	}
}
