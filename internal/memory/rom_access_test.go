package memory

import (
	"testing"
	"gones/internal/cartridge"
)

// TestCPUROMAccess validates CPU ability to read ROM data in the $8000-$FFFF range
func TestCPUROMAccess(t *testing.T) {
	// Create test ROM with known data patterns
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1). // 16KB ROM
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0xAA, 0xBB, 0xCC, 0xDD}). // First 4 bytes
		WithData(0x3000, []uint8{0x11, 0x22, 0x33, 0x44})  // Mid-range bytes (avoid vector area)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}

	// Create memory with cartridge
	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	testCases := []struct {
		name          string
		address       uint16
		expectedValue uint8
		description   string
	}{
		{"ROM Start", 0x8000, 0xAA, "First byte of ROM"},
		{"ROM Second", 0x8001, 0xBB, "Second byte of ROM"},
		{"ROM Third", 0x8002, 0xCC, "Third byte of ROM"},
		{"ROM Fourth", 0x8003, 0xDD, "Fourth byte of ROM"},
		{"ROM Mid-range", 0x9000, 0x00, "Middle of ROM (uninitialized)"},
		{"ROM Test Area", 0xB000, 0x11, "Test data area"},
		{"ROM Test+1", 0xB001, 0x22, "Test data area + 1"},
		{"ROM Test+2", 0xB002, 0x33, "Test data area + 2"},
		{"ROM Test+3", 0xB003, 0x44, "Test data area + 3"},
		// NROM-128 mirroring: $C000-$FFFF mirrors $8000-$BFFF
		{"Mirror Start", 0xC000, 0xAA, "First byte of mirrored ROM"},
		{"Mirror Second", 0xC001, 0xBB, "Second byte of mirrored ROM"},
		{"Mirror Test Area", 0xF000, 0x11, "Test data area mirrored"},
		{"Mirror Test+1", 0xF001, 0x22, "Test data area + 1 mirrored"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mem.Read(tc.address)
			if result != tc.expectedValue {
				t.Errorf("Read(0x%04X) = 0x%02X, want 0x%02X (%s)",
					tc.address, result, tc.expectedValue, tc.description)
			}
		})
	}
}

// TestCPUROMAccess32KB validates CPU ROM access for 32KB (NROM-256) cartridges
func TestCPUROMAccess32KB(t *testing.T) {
	// Create 32KB test ROM
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(2). // 32KB ROM
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0xAA, 0xBB}). // First bank start
		WithData(0x4000, []uint8{0xCC, 0xDD}). // Second bank start
		WithData(0x7000, []uint8{0xEE, 0xFF})  // End area (avoid vectors)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create 32KB test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	testCases := []struct {
		name          string
		address       uint16
		expectedValue uint8
		description   string
	}{
		{"First Bank Start", 0x8000, 0xAA, "First byte of first bank"},
		{"First Bank Second", 0x8001, 0xBB, "Second byte of first bank"},
		{"Second Bank Start", 0xC000, 0xCC, "First byte of second bank"},
		{"Second Bank Second", 0xC001, 0xDD, "Second byte of second bank"},
		{"ROM Test Area", 0xF000, 0xEE, "Test area in second bank"},
		{"ROM Test+1", 0xF001, 0xFF, "Test area + 1 in second bank"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := mem.Read(tc.address)
			if result != tc.expectedValue {
				t.Errorf("Read(0x%04X) = 0x%02X, want 0x%02X (%s)",
					tc.address, result, tc.expectedValue, tc.description)
			}
		})
	}
}

// TestResetVectorAccess validates proper reset vector reading from $FFFC-$FFFD
func TestResetVectorAccess(t *testing.T) {
	testCases := []struct {
		name        string
		resetVector uint16
		prgSize     uint8
		description string
	}{
		{"Standard Reset Vector", 0x8000, 1, "Standard reset vector at ROM start"},
		{"Custom Reset Vector", 0x8123, 1, "Custom reset vector"},
		{"High Reset Vector", 0xC456, 2, "Reset vector in second bank of 32KB ROM"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create ROM with specific reset vector
			romBuilder := cartridge.NewTestROMBuilder().
				WithPRGSize(tc.prgSize).
				WithResetVector(tc.resetVector)

			cart, err := romBuilder.BuildCartridge()
			if err != nil {
				t.Fatalf("Failed to create test cartridge: %v", err)
			}

			ppu := &MockPPU{}
			apu := &MockAPU{}
			mem := New(ppu, apu, cart)

			// Read reset vector from $FFFC-$FFFD
			vectorLow := mem.Read(0xFFFC)
			vectorHigh := mem.Read(0xFFFD)
			actualVector := uint16(vectorLow) | (uint16(vectorHigh) << 8)

			if actualVector != tc.resetVector {
				t.Errorf("Reset vector = 0x%04X, want 0x%04X (%s)",
					actualVector, tc.resetVector, tc.description)
			}

			// Verify individual bytes
			expectedLow := uint8(tc.resetVector & 0xFF)
			expectedHigh := uint8(tc.resetVector >> 8)

			if vectorLow != expectedLow {
				t.Errorf("Reset vector low byte = 0x%02X, want 0x%02X",
					vectorLow, expectedLow)
			}

			if vectorHigh != expectedHigh {
				t.Errorf("Reset vector high byte = 0x%02X, want 0x%02X",
					vectorHigh, expectedHigh)
			}
		})
	}
}

// TestInterruptVectorAccess validates IRQ and NMI vector access
func TestInterruptVectorAccess(t *testing.T) {
	nmiVector := uint16(0x8100)
	irqVector := uint16(0x8200)

	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithNMIVector(nmiVector).
		WithIRQVector(irqVector).
		WithResetVector(0x8000)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	// Test NMI vector at $FFFA-$FFFB
	nmiLow := mem.Read(0xFFFA)
	nmiHigh := mem.Read(0xFFFB)
	actualNMI := uint16(nmiLow) | (uint16(nmiHigh) << 8)

	if actualNMI != nmiVector {
		t.Errorf("NMI vector = 0x%04X, want 0x%04X", actualNMI, nmiVector)
	}

	// Test IRQ vector at $FFFE-$FFFF
	irqLow := mem.Read(0xFFFE)
	irqHigh := mem.Read(0xFFFF)
	actualIRQ := uint16(irqLow) | (uint16(irqHigh) << 8)

	if actualIRQ != irqVector {
		t.Errorf("IRQ vector = 0x%04X, want 0x%04X", actualIRQ, irqVector)
	}
}

// TestCHRROMAccess validates CHR ROM access from PPU address space $0000-$1FFF
func TestCHRROMAccess(t *testing.T) {
	// Create test CHR data pattern
	chrData := make([]uint8, 8192) // 8KB CHR ROM
	for i := 0; i < len(chrData); i++ {
		chrData[i] = uint8(i & 0xFF) // Pattern: 0x00, 0x01, ..., 0xFF, 0x00, ...
	}

	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithCHRData(chrData).
		WithResetVector(0x8000)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}

	// Create PPU memory with cartridge
	ppuMem := NewPPUMemory(cart, MirrorHorizontal)

	testCases := []struct {
		name          string
		address       uint16
		expectedValue uint8
		description   string
	}{
		{"CHR Start", 0x0000, 0x00, "First byte of CHR ROM"},
		{"CHR Pattern 1", 0x0010, 0x10, "Pattern byte at 0x10"},
		{"CHR Pattern 2", 0x00FF, 0xFF, "Pattern byte at 0xFF"},
		{"CHR Pattern 3", 0x0100, 0x00, "Pattern wraps at 0x100"},
		{"CHR Pattern 4", 0x0110, 0x10, "Pattern continues at 0x110"},
		{"CHR Mid-range", 0x1000, 0x00, "Middle of CHR ROM"},
		{"CHR Pattern 5", 0x1010, 0x10, "Pattern in second half"},
		{"CHR Near End", 0x1FF0, 0xF0, "Near end of CHR ROM"},
		{"CHR End", 0x1FFF, 0xFF, "Last byte of CHR ROM"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ppuMem.Read(tc.address)
			if result != tc.expectedValue {
				t.Errorf("CHR Read(0x%04X) = 0x%02X, want 0x%02X (%s)",
					tc.address, result, tc.expectedValue, tc.description)
			}
		})
	}
}

// TestCHRRAMAccess validates CHR RAM functionality
func TestCHRRAMAccess(t *testing.T) {
	// Create ROM with CHR RAM (CHR size = 0)
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRRAM(). // This sets CHR size to 0, enabling CHR RAM
		WithResetVector(0x8000)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create CHR RAM test cartridge: %v", err)
	}

	ppuMem := NewPPUMemory(cart, MirrorHorizontal)

	// Test CHR RAM read/write functionality
	testData := []struct {
		address uint16
		value   uint8
	}{
		{0x0000, 0xAA},
		{0x0001, 0xBB},
		{0x1000, 0xCC},
		{0x1FFF, 0xDD},
	}

	for _, td := range testData {
		t.Run("CHR RAM Write/Read", func(t *testing.T) {
			// Write to CHR RAM
			ppuMem.Write(td.address, td.value)

			// Read back and verify
			result := ppuMem.Read(td.address)
			if result != td.value {
				t.Errorf("CHR RAM at 0x%04X: wrote 0x%02X, read 0x%02X",
					td.address, td.value, result)
			}
		})
	}
}

// TestROMBoundaryConditions validates behavior at ROM boundaries
func TestROMBoundaryConditions(t *testing.T) {
	// Test with minimal ROM
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithResetVector(0x8000)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	// Test reads before ROM space return 0
	beforeROMAddresses := []uint16{0x7FFF, 0x6000, 0x4020}
	for _, addr := range beforeROMAddresses {
		t.Run("Before ROM Space", func(t *testing.T) {
			result := mem.Read(addr)
			if result != 0 {
				t.Errorf("Read(0x%04X) = 0x%02X, want 0x00 (before ROM)",
					addr, result)
			}
		})
	}

	// Test that ROM space responds
	romAddresses := []uint16{0x8000, 0xC000, 0xFFFF}
	for _, addr := range romAddresses {
		t.Run("ROM Space Access", func(t *testing.T) {
			// Should not panic and should return some value
			result := mem.Read(addr)
			_ = result // Just verify no panic occurs
		})
	}
}

// TestZeroSizeROMHandling validates graceful handling of edge cases
func TestZeroSizeROMHandling(t *testing.T) {
	// Create mock cartridge with zero-length ROM data
	cart := &MockCartridge{}
	// Leave prgData and chrData as zero-initialized

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	// ROM access should return 0 for empty ROM
	testAddresses := []uint16{0x8000, 0xC000, 0xFFFF}
	for _, addr := range testAddresses {
		t.Run("Zero ROM Access", func(t *testing.T) {
			result := mem.Read(addr)
			if result != 0 {
				t.Errorf("Read(0x%04X) from zero ROM = 0x%02X, want 0x00",
					addr, result)
			}
		})
	}

	// CHR access should also return 0
	ppuMem := NewPPUMemory(cart, MirrorHorizontal)
	chrAddresses := []uint16{0x0000, 0x1000, 0x1FFF}
	for _, addr := range chrAddresses {
		t.Run("Zero CHR Access", func(t *testing.T) {
			result := ppuMem.Read(addr)
			if result != 0 {
				t.Errorf("CHR Read(0x%04X) from zero CHR = 0x%02X, want 0x00",
					addr, result)
			}
		})
	}
}

// TestROMWritesBehavior validates that ROM writes are handled appropriately
func TestROMWritesBehavior(t *testing.T) {
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0xAA}) // Known value at start

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}

	ppu := &MockPPU{}
	apu := &MockAPU{}
	mem := New(ppu, apu, cart)

	// Verify initial ROM content
	originalValue := mem.Read(0x8000)
	if originalValue != 0xAA {
		t.Fatalf("Initial ROM value = 0x%02X, want 0xAA", originalValue)
	}

	// Attempt to write to ROM (should be ignored for NROM)
	mem.Write(0x8000, 0x55)

	// Verify ROM content unchanged
	currentValue := mem.Read(0x8000)
	if currentValue != originalValue {
		t.Errorf("ROM value after write = 0x%02X, want 0x%02X (unchanged)",
			currentValue, originalValue)
	}
}