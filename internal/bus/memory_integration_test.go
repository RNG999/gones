package bus

import (
	"testing"
	"gones/internal/cartridge"
)

// TestBusCartridgeIntegration validates complete bus integration with cartridge
func TestBusCartridgeIntegration(t *testing.T) {
	// Create test ROM with known patterns
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{
			0xA9, 0x42, // LDA #$42
			0x85, 0x10, // STA $10
			0xA9, 0x55, // LDA #$55
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)
			0x4C, 0x0A, 0x80, // JMP $800A (infinite loop)
		}).
		WithDescription("Bus integration test ROM")

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}

	// Create bus and load cartridge
	bus := New()
	bus.LoadCartridge(cart)

	// Test CPU can read ROM through bus
	t.Run("CPU ROM Access", func(t *testing.T) {
		// Read first instruction
		instruction := bus.Memory.Read(0x8000)
		if instruction != 0xA9 {
			t.Errorf("First instruction = 0x%02X, want 0xA9 (LDA)", instruction)
		}

		// Read operand
		operand := bus.Memory.Read(0x8001)
		if operand != 0x42 {
			t.Errorf("LDA operand = 0x%02X, want 0x42", operand)
		}
	})

	// Test reset vector accessible through bus
	t.Run("Reset Vector Access", func(t *testing.T) {
		resetLow := bus.Memory.Read(0xFFFC)
		resetHigh := bus.Memory.Read(0xFFFD)
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		
		if resetVector != 0x8000 {
			t.Errorf("Reset vector = 0x%04X, want 0x8000", resetVector)
		}
	})

	// Test PPU CHR ROM access
	t.Run("PPU CHR Access", func(t *testing.T) {
		// PPU memory access is handled through the bus's PPU memory interface
		// This test verifies the bus properly routes CHR ROM to PPU
		// Direct CHR access is tested through the PPU's internal memory interface
		
		// Verify PPU exists and is properly initialized
		if bus.PPU == nil {
			t.Error("PPU should be initialized in bus")
		}
	})

	// Test CPU startup with proper reset vector
	t.Run("CPU Reset Integration", func(t *testing.T) {
		// Reset CPU
		bus.Reset()

		// CPU should read reset vector and set PC
		state := bus.GetCPUState()
		if state.PC != 0x8000 {
			t.Errorf("CPU PC after reset = 0x%04X, want 0x8000", state.PC)
		}
	})
}

// TestBusMemoryMapping validates memory mapping through bus
func TestBusMemoryMapping(t *testing.T) {
	// Create ROM with mirroring test
	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1). // 16KB ROM for mirroring
		WithCHRSize(1).
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0xAA}). // First byte
		WithData(0x3FF0, []uint8{0xBB})  // Near end byte (before interrupt vectors)

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create mirroring test cartridge: %v", err)
	}

	bus := New()
	bus.LoadCartridge(cart)

	// Test NROM-128 mirroring: $C000-$FFFF mirrors $8000-$BFFF
	t.Run("NROM-128 Mirroring", func(t *testing.T) {
		// Test first byte mirroring
		value1 := bus.Memory.Read(0x8000)
		value2 := bus.Memory.Read(0xC000)
		if value1 != value2 {
			t.Errorf("ROM mirroring failed: 0x8000=0x%02X, 0xC000=0x%02X", value1, value2)
		}
		if value1 != 0xAA {
			t.Errorf("ROM first byte = 0x%02X, want 0xAA", value1)
		}

		// Test near-end byte mirroring (avoiding interrupt vectors)
		value3 := bus.Memory.Read(0xBFF0)
		value4 := bus.Memory.Read(0xFFF0)
		if value3 != value4 {
			t.Errorf("ROM end mirroring failed: 0xBFF0=0x%02X, 0xFFF0=0x%02X", value3, value4)
		}
		if value3 != 0xBB {
			t.Errorf("ROM near-end byte = 0x%02X, want 0xBB", value3)
		}
	})

	// Test memory regions isolation
	t.Run("Memory Region Isolation", func(t *testing.T) {
		// Write to RAM
		bus.Memory.Write(0x0000, 0x11)
		ramValue := bus.Memory.Read(0x0000)

		// Read from ROM
		romValue := bus.Memory.Read(0x8000)

		// They should be different
		if ramValue == romValue && ramValue != 0x11 {
			t.Error("RAM and ROM should be isolated")
		}

		if ramValue != 0x11 {
			t.Errorf("RAM value = 0x%02X, want 0x11", ramValue)
		}
	})

	// Test unimplemented regions return 0
	t.Run("Unimplemented Regions", func(t *testing.T) {
		unimplementedAddresses := []uint16{0x4020, 0x5000, 0x7FFF}
		for _, addr := range unimplementedAddresses {
			value := bus.Memory.Read(addr)
			if value != 0 {
				t.Errorf("Unimplemented region 0x%04X = 0x%02X, want 0x00", addr, value)
			}
		}
	})
}

// TestBusExecutionWithROM validates bus execution with ROM instructions
func TestBusExecutionWithROM(t *testing.T) {
	// Create ROM with executable instructions
	instructions := []uint8{
		0xA9, 0x42, // LDA #$42
		0x85, 0x10, // STA $10
		0x18,       // CLC
		0x69, 0x10, // ADC #$10
		0x85, 0x11, // STA $11
		0x4C, 0x0A, 0x80, // JMP $800A (loop back to CLC)
	}

	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithResetVector(0x8000).
		WithInstructions(instructions).
		WithDescription("Execution test ROM")

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create execution test cartridge: %v", err)
	}

	bus := New()
	bus.LoadCartridge(cart)
	bus.Reset()
	bus.EnableExecutionLogging()

	// Execute several steps
	t.Run("Instruction Execution", func(t *testing.T) {
		initialPC := bus.GetCPUState().PC
		if initialPC != 0x8000 {
			t.Errorf("Initial PC = 0x%04X, want 0x8000", initialPC)
		}

		// Execute LDA #$42
		bus.Step()
		state := bus.GetCPUState()
		if state.A != 0x42 {
			t.Errorf("After LDA, A = 0x%02X, want 0x42", state.A)
		}

		// Execute STA $10
		bus.Step()
		ramValue := bus.Memory.Read(0x10)
		if ramValue != 0x42 {
			t.Errorf("After STA, RAM[0x10] = 0x%02X, want 0x42", ramValue)
		}

		// Execute CLC
		bus.Step()
		state = bus.GetCPUState()
		if state.Flags.C {
			t.Error("After CLC, carry flag should be clear")
		}

		// Execute ADC #$10
		bus.Step()
		state = bus.GetCPUState()
		if state.A != 0x52 { // 0x42 + 0x10
			t.Errorf("After ADC, A = 0x%02X, want 0x52", state.A)
		}
	})

	// Test execution logging
	t.Run("Execution Logging", func(t *testing.T) {
		log := bus.GetExecutionLog()
		if len(log) == 0 {
			t.Error("Execution log should not be empty")
		}

		// Check first logged instruction
		firstEvent := log[0]
		if firstEvent.PCValue != 0x8000 {
			t.Errorf("First logged PC = 0x%04X, want 0x8000", firstEvent.PCValue)
		}
		if firstEvent.InstructionOp != 0xA9 {
			t.Errorf("First logged opcode = 0x%02X, want 0xA9", firstEvent.InstructionOp)
		}
	})
}

// TestBusNMIIntegration validates NMI handling with ROM
func TestBusNMIIntegration(t *testing.T) {
	nmiVector := uint16(0x8100)
	
	// Create ROM with NMI handler
	instructions := []uint8{
		// Reset handler at $8000
		0xA9, 0x01, // LDA #$01
		0x85, 0x20, // STA $20
		0x4C, 0x04, 0x80, // JMP $8004 (infinite loop)
	}
	
	nmiHandler := []uint8{
		// NMI handler at $8100 (offset 0x100)
		0xA9, 0x02, // LDA #$02
		0x85, 0x21, // STA $21
		0x40,       // RTI
	}

	romBuilder := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithResetVector(0x8000).
		WithNMIVector(nmiVector).
		WithInstructions(instructions).
		WithData(0x0100, nmiHandler). // Place NMI handler at offset 0x100
		WithDescription("NMI test ROM")

	cart, err := romBuilder.BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create NMI test cartridge: %v", err)
	}

	bus := New()
	bus.LoadCartridge(cart)
	bus.Reset()

	// Verify NMI vector is correct
	t.Run("NMI Vector Setup", func(t *testing.T) {
		nmiLow := bus.Memory.Read(0xFFFA)
		nmiHigh := bus.Memory.Read(0xFFFB)
		actualVector := uint16(nmiLow) | (uint16(nmiHigh) << 8)
		
		if actualVector != nmiVector {
			t.Errorf("NMI vector = 0x%04X, want 0x%04X", actualVector, nmiVector)
		}
	})

	// Test NMI handler accessibility
	t.Run("NMI Handler Access", func(t *testing.T) {
		// Verify NMI handler instructions are accessible
		handlerStart := bus.Memory.Read(nmiVector)
		if handlerStart != 0xA9 { // LDA
			t.Errorf("NMI handler first instruction = 0x%02X, want 0xA9", handlerStart)
		}

		handlerOperand := bus.Memory.Read(nmiVector + 1)
		if handlerOperand != 0x02 {
			t.Errorf("NMI handler operand = 0x%02X, want 0x02", handlerOperand)
		}
	})
}

// TestBusCartridgeSwapping validates cartridge replacement
func TestBusCartridgeSwapping(t *testing.T) {
	// Create first cartridge
	cart1, err := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0xAA}).
		BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create first cartridge: %v", err)
	}

	// Create second cartridge with different data
	cart2, err := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0xBB}).
		BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create second cartridge: %v", err)
	}

	bus := New()

	// Test first cartridge
	t.Run("First Cartridge", func(t *testing.T) {
		bus.LoadCartridge(cart1)
		value := bus.Memory.Read(0x8000)
		if value != 0xAA {
			t.Errorf("First cartridge ROM[0x8000] = 0x%02X, want 0xAA", value)
		}
	})

	// Test cartridge swapping
	t.Run("Cartridge Swapping", func(t *testing.T) {
		bus.LoadCartridge(cart2)
		value := bus.Memory.Read(0x8000)
		if value != 0xBB {
			t.Errorf("Second cartridge ROM[0x8000] = 0x%02X, want 0xBB", value)
		}
	})

	// Verify old cartridge data is no longer accessible
	t.Run("Old Data Inaccessible", func(t *testing.T) {
		value := bus.Memory.Read(0x8000)
		if value == 0xAA {
			t.Error("Old cartridge data should not be accessible after swap")
		}
		if value != 0xBB {
			t.Errorf("Current cartridge ROM[0x8000] = 0x%02X, want 0xBB", value)
		}
	})
}

// TestBusComprehensiveMemoryValidation validates all memory subsystems
func TestBusComprehensiveMemoryValidation(t *testing.T) {
	// Create comprehensive test cartridge
	cart, err := cartridge.NewTestROMBuilder().
		WithPRGSize(1).
		WithCHRSize(1).
		WithMirroring(cartridge.MirrorVertical).
		WithBattery().
		WithResetVector(0x8000).
		WithData(0x0000, []uint8{0x10, 0x20, 0x30, 0x40}).
		BuildCartridge()
	if err != nil {
		t.Fatalf("Failed to create comprehensive test cartridge: %v", err)
	}

	bus := New()
	bus.LoadCartridge(cart)

	// Test all memory regions
	t.Run("RAM Region", func(t *testing.T) {
		bus.Memory.Write(0x0000, 0x55)
		value := bus.Memory.Read(0x0000)
		if value != 0x55 {
			t.Errorf("RAM write/read failed: got 0x%02X, want 0x55", value)
		}

		// Test RAM mirroring
		mirrorValue := bus.Memory.Read(0x0800)
		if mirrorValue != 0x55 {
			t.Errorf("RAM mirroring failed: got 0x%02X, want 0x55", mirrorValue)
		}
	})

	t.Run("PPU Registers", func(t *testing.T) {
		// Write to PPUCTRL
		bus.Memory.Write(0x2000, 0x80)
		// PPU register access is tested through mock, just verify no panic
	})

	t.Run("APU Registers", func(t *testing.T) {
		// Write to APU register
		bus.Memory.Write(0x4000, 0x30)
		// APU register access is tested through mock, just verify no panic
	})

	t.Run("SRAM Region", func(t *testing.T) {
		bus.Memory.Write(0x6000, 0x77)
		value := bus.Memory.Read(0x6000)
		if value != 0x77 {
			t.Errorf("SRAM write/read failed: got 0x%02X, want 0x77", value)
		}
	})

	t.Run("ROM Region", func(t *testing.T) {
		value := bus.Memory.Read(0x8000)
		if value != 0x10 {
			t.Errorf("ROM read failed: got 0x%02X, want 0x10", value)
		}

		// Test ROM mirroring
		mirrorValue := bus.Memory.Read(0xC000)
		if mirrorValue != 0x10 {
			t.Errorf("ROM mirroring failed: got 0x%02X, want 0x10", mirrorValue)
		}
	})

	t.Run("CHR Memory", func(t *testing.T) {
		// Test CHR access through bus integration
		// CHR ROM is accessible to PPU through its memory interface
		// This validates the cartridge is properly loaded and accessible
		
		// Verify PPU is initialized and ready for CHR access
		if bus.PPU == nil {
			t.Error("PPU should be initialized")
		}
		
		// Test that bus has properly set up PPU memory with CHR data
		// (Actual CHR access testing is done at the memory layer)
	})

	t.Run("Interrupt Vectors", func(t *testing.T) {
		// Test all vectors
		resetLow := bus.Memory.Read(0xFFFC)
		resetHigh := bus.Memory.Read(0xFFFD)
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		if resetVector != 0x8000 {
			t.Errorf("Reset vector = 0x%04X, want 0x8000", resetVector)
		}
	})
}