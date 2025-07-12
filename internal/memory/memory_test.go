package memory

import (
	"testing"
)

// MockPPU implements PPUInterface for testing
type MockPPU struct {
	registers  [8]uint8
	readCalls  []uint16
	writeCalls []RegisterWrite
}

type RegisterWrite struct {
	Address uint16
	Value   uint8
}

func (m *MockPPU) ReadRegister(address uint16) uint8 {
	m.readCalls = append(m.readCalls, address)
	return m.registers[address&0x7]
}

func (m *MockPPU) WriteRegister(address uint16, value uint8) {
	m.writeCalls = append(m.writeCalls, RegisterWrite{Address: address, Value: value})
	m.registers[address&0x7] = value
}

// MockAPU implements APUInterface for testing
type MockAPU struct {
	registers  [0x18]uint8
	writeCalls []RegisterWrite
}

func (m *MockAPU) WriteRegister(address uint16, value uint8) {
	m.writeCalls = append(m.writeCalls, RegisterWrite{Address: address, Value: value})
	if address >= 0x4000 && address <= 0x4017 {
		m.registers[address-0x4000] = value
	}
}

func (m *MockAPU) ReadStatus() uint8 {
	return 0x00 // Mock implementation
}

// MockCartridge implements CartridgeInterface for testing
type MockCartridge struct {
	prgData   [0x8000]uint8
	chrData   [0x2000]uint8
	prgReads  []uint16
	prgWrites []RegisterWrite
	chrReads  []uint16
	chrWrites []RegisterWrite
}

func (m *MockCartridge) ReadPRG(address uint16) uint8 {
	m.prgReads = append(m.prgReads, address)
	if address >= 0x6000 && address < 0x8000 {
		// MockCartridge doesn't have SRAM, return 0 for SRAM range
		return 0
	}
	return m.prgData[address&0x7FFF]
}

func (m *MockCartridge) WritePRG(address uint16, value uint8) {
	m.prgWrites = append(m.prgWrites, RegisterWrite{Address: address, Value: value})
	if address >= 0x6000 && address < 0x8000 {
		// MockCartridge doesn't have SRAM, ignore writes to SRAM range
		return
	}
	m.prgData[address&0x7FFF] = value
}

func (m *MockCartridge) ReadCHR(address uint16) uint8 {
	m.chrReads = append(m.chrReads, address)
	return m.chrData[address&0x1FFF]
}

func (m *MockCartridge) WriteCHR(address uint16, value uint8) {
	m.chrWrites = append(m.chrWrites, RegisterWrite{Address: address, Value: value})
	m.chrData[address&0x1FFF] = value
}

func TestMemory_New(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}

	mem := New(ppu, apu, cart)

	if mem == nil {
		t.Fatal("New() returned nil")
	}

	// Verify all RAM is initialized to zero
	for i := 0; i < len(mem.ram); i++ {
		if mem.ram[i] != 0 {
			t.Errorf("RAM[%d] = %d, want 0", i, mem.ram[i])
		}
	}
}

func TestMemory_ReadWriteRAM(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	testCases := []struct {
		name             string
		address          uint16
		expectedRAMIndex uint16
	}{
		{"Direct RAM access", 0x0000, 0x0000},
		{"Direct RAM access end", 0x07FF, 0x07FF},
		{"First mirror", 0x0800, 0x0000},
		{"First mirror end", 0x0FFF, 0x07FF},
		{"Second mirror", 0x1000, 0x0000},
		{"Second mirror end", 0x17FF, 0x07FF},
		{"Third mirror", 0x1800, 0x0000},
		{"Third mirror end", 0x1FFF, 0x07FF},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Write a unique value
			value := uint8(tc.address & 0xFF)
			mem.Write(tc.address, value)

			// Read back and verify
			result := mem.Read(tc.address)
			if result != value {
				t.Errorf("Read(%04X) = %02X, want %02X", tc.address, result, value)
			}

			// Verify the actual RAM location was updated
			if mem.ram[tc.expectedRAMIndex] != value {
				t.Errorf("RAM[%04X] = %02X, want %02X", tc.expectedRAMIndex, mem.ram[tc.expectedRAMIndex], value)
			}
		})
	}
}

func TestMemory_PPURegisterAccess(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	testCases := []struct {
		name               string
		address            uint16
		expectedPPUAddress uint16
	}{
		{"PPUCTRL", 0x2000, 0x2000},
		{"PPUMASK", 0x2001, 0x2001},
		{"PPUSTATUS", 0x2002, 0x2002},
		{"OAMADDR", 0x2003, 0x2003},
		{"OAMDATA", 0x2004, 0x2004},
		{"PPUSCROLL", 0x2005, 0x2005},
		{"PPUADDR", 0x2006, 0x2006},
		{"PPUDATA", 0x2007, 0x2007},
		{"First mirror PPUCTRL", 0x2008, 0x2000},
		{"Mirror PPUMASK", 0x2009, 0x2001},
		{"Second mirror PPUCTRL", 0x2010, 0x2000},
		{"High mirror PPUSTATUS", 0x3002, 0x2002},
		{"End of PPU space", 0x3FFF, 0x2007},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" Write", func(t *testing.T) {
			value := uint8(0x42)
			mem.Write(tc.address, value)

			// Verify PPU WriteRegister was called with correct address
			if len(ppu.writeCalls) == 0 {
				t.Fatal("PPU WriteRegister not called")
			}

			lastCall := ppu.writeCalls[len(ppu.writeCalls)-1]
			if lastCall.Address != tc.expectedPPUAddress {
				t.Errorf("PPU WriteRegister called with address %04X, want %04X",
					lastCall.Address, tc.expectedPPUAddress)
			}
			if lastCall.Value != value {
				t.Errorf("PPU WriteRegister called with value %02X, want %02X",
					lastCall.Value, value)
			}
		})

		t.Run(tc.name+" Read", func(t *testing.T) {
			// Set expected value in mock PPU
			ppu.registers[tc.expectedPPUAddress&0x7] = 0x84

			result := mem.Read(tc.address)

			// Verify PPU ReadRegister was called with correct address
			if len(ppu.readCalls) == 0 {
				t.Fatal("PPU ReadRegister not called")
			}

			lastCall := ppu.readCalls[len(ppu.readCalls)-1]
			if lastCall != tc.expectedPPUAddress {
				t.Errorf("PPU ReadRegister called with address %04X, want %04X",
					lastCall, tc.expectedPPUAddress)
			}

			if result != 0x84 {
				t.Errorf("Read(%04X) = %02X, want %02X", tc.address, result, 0x84)
			}
		})
	}
}

func TestMemory_APURegisterAccess(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	testCases := []struct {
		name    string
		address uint16
	}{
		{"APU Pulse 1 Vol", 0x4000},
		{"APU Pulse 1 Sweep", 0x4001},
		{"APU Pulse 1 Timer Low", 0x4002},
		{"APU Pulse 1 Timer High", 0x4003},
		{"APU Pulse 2 Vol", 0x4004},
		{"APU Triangle Linear", 0x4008},
		{"APU Noise Vol", 0x400C},
		{"APU DMC", 0x4010},
		{"APU Status", 0x4015},
		{"APU Frame Counter", 0x4017},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			value := uint8(0x55)
			mem.Write(tc.address, value)

			// Verify APU WriteRegister was called
			if len(apu.writeCalls) == 0 {
				t.Fatal("APU WriteRegister not called")
			}

			lastCall := apu.writeCalls[len(apu.writeCalls)-1]
			if lastCall.Address != tc.address {
				t.Errorf("APU WriteRegister called with address %04X, want %04X",
					lastCall.Address, tc.address)
			}
			if lastCall.Value != value {
				t.Errorf("APU WriteRegister called with value %02X, want %02X",
					lastCall.Value, value)
			}
		})
	}
}

func TestMemory_OAMDMA(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up test data in RAM
	for i := 0; i < 256; i++ {
		mem.ram[i] = uint8(i)
	}

	// Trigger OAM DMA from page 0
	mem.Write(0x4014, 0x00)

	// Verify 256 OAMDATA writes occurred
	if len(ppu.writeCalls) != 256 {
		t.Fatalf("Expected 256 PPU writes, got %d", len(ppu.writeCalls))
	}

	// Verify each write was to OAMDATA with correct value
	for i := 0; i < 256; i++ {
		call := ppu.writeCalls[i]
		if call.Address != 0x2004 {
			t.Errorf("Write %d: address = %04X, want 2004", i, call.Address)
		}
		if call.Value != uint8(i) {
			t.Errorf("Write %d: value = %02X, want %02X", i, call.Value, uint8(i))
		}
	}
}

func TestMemory_OAMDMA_HighPage(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up test data in high RAM page
	for i := 0; i < 256; i++ {
		mem.ram[0x300+i] = uint8(0xFF - i)
	}

	// Trigger OAM DMA from page 3 (mirrored to 0x0300 in RAM)
	mem.Write(0x4014, 0x03)

	// Verify 256 OAMDATA writes occurred with correct values
	if len(ppu.writeCalls) != 256 {
		t.Fatalf("Expected 256 PPU writes, got %d", len(ppu.writeCalls))
	}

	for i := 0; i < 256; i++ {
		call := ppu.writeCalls[i]
		expectedValue := uint8(0xFF - i)
		if call.Value != expectedValue {
			t.Errorf("Write %d: value = %02X, want %02X", i, call.Value, expectedValue)
		}
	}
}

func TestMemory_CartridgeAccess(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	// Set up test data in cartridge
	cart.prgData[0x0000] = 0xAA
	cart.prgData[0x7FFF] = 0x55

	testCases := []struct {
		name          string
		address       uint16
		expectedValue uint8
	}{
		{"PRG ROM start", 0x8000, 0xAA},
		{"PRG ROM end", 0xFFFF, 0x55},
		{"PRG ROM middle", 0xC000, 0x00},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" Read", func(t *testing.T) {
			result := mem.Read(tc.address)

			if result != tc.expectedValue {
				t.Errorf("Read(%04X) = %02X, want %02X", tc.address, result, tc.expectedValue)
			}

			// Verify cartridge was called
			if len(cart.prgReads) == 0 {
				t.Fatal("Cartridge ReadPRG not called")
			}

			lastCall := cart.prgReads[len(cart.prgReads)-1]
			if lastCall != tc.address {
				t.Errorf("Cartridge ReadPRG called with %04X, want %04X", lastCall, tc.address)
			}
		})

		t.Run(tc.name+" Write", func(t *testing.T) {
			value := uint8(0x42)
			mem.Write(tc.address, value)

			// Verify cartridge write was called
			if len(cart.prgWrites) == 0 {
				t.Fatal("Cartridge WritePRG not called")
			}

			lastCall := cart.prgWrites[len(cart.prgWrites)-1]
			if lastCall.Address != tc.address {
				t.Errorf("Cartridge WritePRG called with address %04X, want %04X",
					lastCall.Address, tc.address)
			}
			if lastCall.Value != value {
				t.Errorf("Cartridge WritePRG called with value %02X, want %02X",
					lastCall.Value, value)
			}
		})
	}
}

func TestMemory_UnmappedRegions(t *testing.T) {
	ppu := &MockPPU{}
	apu := &MockAPU{}
	cart := &MockCartridge{}
	mem := New(ppu, apu, cart)

	unmappedAddresses := []uint16{
		0x4018, // Above APU registers
		0x4019,
		0x401F, // End of test mode
		0x4020, // Start of cartridge expansion
		0x5000, // Middle of expansion
		0x7FFF, // End of expansion before PRG ROM
	}

	for _, addr := range unmappedAddresses {
		t.Run("Unmapped read", func(t *testing.T) {
			result := mem.Read(addr)
			if result != 0 {
				t.Errorf("Read(%04X) = %02X, want 0 (unmapped)", addr, result)
			}
		})

		t.Run("Unmapped write", func(t *testing.T) {
			// These should not panic and should be ignored
			mem.Write(addr, 0xFF)
		})
	}
}
