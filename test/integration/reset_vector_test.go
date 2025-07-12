package integration

import (
	"bytes"
	"io"
	"os"
	"testing"

	"gones/internal/cartridge"
)

// TestSampleROM_ResetVectorHandling tests the reset vector handling specifically
func TestSampleROM_ResetVectorHandling(t *testing.T) {
	// Load sample.nes ROM
	file, err := os.Open("/home/claude/work/gones/roms/sample.nes")
	if err != nil {
		t.Fatalf("Failed to open sample.nes: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read sample.nes: %v", err)
	}

	t.Run("Reset vector location analysis", func(t *testing.T) {
		// iNES header analysis
		headerSize := 16
		prgSize := int(data[4]) * 16384 // 16KB units
		chrSize := int(data[5]) * 8192  // 8KB units
		
		t.Logf("Header size: %d bytes", headerSize)
		t.Logf("PRG ROM size: %d bytes", prgSize)
		t.Logf("CHR ROM size: %d bytes", chrSize)
		
		// Reset vector should be at the end of PRG ROM
		resetVectorOffset := headerSize + prgSize - 4
		nmiVectorOffset := headerSize + prgSize - 6
		irqVectorOffset := headerSize + prgSize - 2
		
		t.Logf("Reset vector offset in file: 0x%X", resetVectorOffset)
		
		if resetVectorOffset >= len(data) {
			t.Fatalf("Reset vector offset beyond file size")
		}
		
		// Read the vectors
		resetLow := data[resetVectorOffset]
		resetHigh := data[resetVectorOffset+1]
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		
		nmiLow := data[nmiVectorOffset]
		nmiHigh := data[nmiVectorOffset+1]
		nmiVector := uint16(nmiLow) | (uint16(nmiHigh) << 8)
		
		irqLow := data[irqVectorOffset]
		irqHigh := data[irqVectorOffset+1]
		irqVector := uint16(irqLow) | (uint16(irqHigh) << 8)
		
		t.Logf("NMI vector: 0x%04X (bytes: 0x%02X 0x%02X)", nmiVector, nmiLow, nmiHigh)
		t.Logf("Reset vector: 0x%04X (bytes: 0x%02X 0x%02X)", resetVector, resetLow, resetHigh)
		t.Logf("IRQ vector: 0x%04X (bytes: 0x%02X 0x%02X)", irqVector, irqLow, irqHigh)
		
		// Verify reset vector is in ROM range
		if resetVector < 0x8000 {
			t.Errorf("Reset vector 0x%04X is outside ROM range (0x8000-0xFFFF)", resetVector)
		}
	})

	t.Run("Cartridge reset vector access", func(t *testing.T) {
		// Load cartridge and test reset vector reading
		reader := bytes.NewReader(data)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		// Test reading reset vector through cartridge interface
		resetLow := cart.ReadPRG(0xFFFC)
		resetHigh := cart.ReadPRG(0xFFFD)
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		
		t.Logf("Reset vector via cartridge: 0x%04X (0x%02X 0x%02X)", resetVector, resetLow, resetHigh)
		
		if resetVector == 0x0000 {
			t.Error("Reset vector reads as 0x0000 - cartridge vector reading broken!")
		}
		
		// Test reading the code at the reset vector
		if resetVector >= 0x8000 {
			firstOpcode := cart.ReadPRG(resetVector)
			secondByte := cart.ReadPRG(resetVector + 1)
			thirdByte := cart.ReadPRG(resetVector + 2)
			
			t.Logf("Code at reset vector 0x%04X: 0x%02X 0x%02X 0x%02X", 
				resetVector, firstOpcode, secondByte, thirdByte)
			
			// Should not be all zeros
			if firstOpcode == 0x00 && secondByte == 0x00 && thirdByte == 0x00 {
				t.Error("Code at reset vector is all zeros - PRG ROM not loaded correctly!")
			}
		}
	})

	t.Run("Memory system reset vector access", func(t *testing.T) {
		// Test reset vector reading through the memory system
		reader := bytes.NewReader(data)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.LoadCartridge(cart)

		// Read reset vector through memory system (as CPU would during reset)
		resetLow := helper.Memory.Read(0xFFFC)
		resetHigh := helper.Memory.Read(0xFFFD)
		resetVector := uint16(resetLow) | (uint16(resetHigh) << 8)
		
		t.Logf("Reset vector via memory system: 0x%04X (0x%02X 0x%02X)", resetVector, resetLow, resetHigh)
		
		if resetVector == 0x0000 {
			t.Error("Memory system returns 0x0000 for reset vector - memory mapping broken!")
		}
		
		// Test that we can read ROM code through memory system
		if resetVector >= 0x8000 {
			codeAtReset := helper.Memory.Read(resetVector)
			t.Logf("First opcode at reset vector via memory: 0x%02X", codeAtReset)
			
			if codeAtReset == 0x00 {
				t.Error("Memory system returns 0x00 for ROM code - ROM mapping broken!")
			}
		}
	})

	t.Run("CPU reset behavior", func(t *testing.T) {
		// Test actual CPU reset behavior
		reader := bytes.NewReader(data)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.LoadCartridge(cart)

		// Check initial state before reset
		t.Logf("Before reset - CPU PC: 0x%04X", helper.CPU.PC)
		
		// Perform reset
		helper.Bus.Reset()
		
		// Check CPU PC after reset
		t.Logf("After reset - CPU PC: 0x%04X", helper.CPU.PC)
		
		if helper.CPU.PC == 0x0000 {
			t.Error("CPU PC is 0x0000 after reset - reset vector not loaded correctly!")
			
			// Manual verification - read reset vector ourselves
			resetLow := helper.Memory.Read(0xFFFC)
			resetHigh := helper.Memory.Read(0xFFFD)
			expectedPC := uint16(resetLow) | (uint16(resetHigh) << 8)
			
			t.Logf("Expected PC from reset vector: 0x%04X", expectedPC)
			t.Logf("This indicates the CPU reset logic is not reading the reset vector properly")
		}
		
		// Verify that the memory at the PC location contains valid code
		currentPC := helper.CPU.PC
		if currentPC != 0x0000 {
			opcodeAtPC := helper.Memory.Read(currentPC)
			t.Logf("Opcode at PC 0x%04X: 0x%02X", currentPC, opcodeAtPC)
		}
	})

	t.Run("Step-by-step memory verification", func(t *testing.T) {
		// Comprehensive verification of memory mapping
		reader := bytes.NewReader(data)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		// Test key memory addresses
		testAddresses := []uint16{
			0x8000, 0x8001, 0x8002, // Start of ROM
			0xFFFC, 0xFFFD,         // Reset vector
			0xFFFE, 0xFFFF,         // IRQ vector
		}

		t.Log("Direct cartridge reads:")
		for _, addr := range testAddresses {
			value := cart.ReadPRG(addr)
			t.Logf("  Cart[0x%04X] = 0x%02X", addr, value)
		}

		// Create integrated system
		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.LoadCartridge(cart)

		t.Log("Memory system reads:")
		for _, addr := range testAddresses {
			value := helper.Memory.Read(addr)
			t.Logf("  Mem[0x%04X] = 0x%02X", addr, value)
		}

		// Compare with actual ROM data
		headerSize := 16
		t.Log("Raw ROM file data (vectors):")
		prgSize := int(data[4]) * 16384
		vectorOffset := headerSize + prgSize - 4
		
		for i := 0; i < 4; i++ {
			romAddr := 0xFFFC + i
			fileOffset := vectorOffset + i
			if fileOffset < len(data) {
				t.Logf("  ROM[0x%04X] = file[0x%X] = 0x%02X", 
					romAddr, fileOffset, data[fileOffset])
			}
		}
	})
}