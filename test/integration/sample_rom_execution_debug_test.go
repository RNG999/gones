package integration

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"testing"

	"gones/internal/cartridge"
)

// TestSampleROM_ExecutionDebug performs detailed debugging of sample.nes execution
func TestSampleROM_ExecutionDebug(t *testing.T) {
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

	// Analyze ROM structure
	t.Run("ROM structure analysis", func(t *testing.T) {
		if len(data) < 16 {
			t.Fatal("ROM too small - missing iNES header")
		}

		// Check iNES header
		if string(data[0:4]) != "NES\x1a" {
			t.Errorf("Invalid iNES signature: %v", data[0:4])
		}

		prgSize := data[4] * 16 // 16KB units
		chrSize := data[5] * 8  // 8KB units
		flags6 := data[6]
		flags7 := data[7]

		t.Logf("PRG ROM: %d KB", prgSize)
		t.Logf("CHR ROM: %d KB", chrSize)
		t.Logf("Flags 6: 0x%02X (mapper low: %d, mirroring: %d)", 
			flags6, (flags6>>4)&0xF, flags6&1)
		t.Logf("Flags 7: 0x%02X (mapper high: %d)", flags7, (flags7>>4)&0xF)

		// Verify we have CHR data
		if chrSize == 0 {
			t.Error("No CHR ROM - this might be a CHR RAM game")
		}

		// Check PRG ROM reset vector
		headerSize := 16
		prgOffset := headerSize
		resetVectorOffset := prgOffset + int(prgSize)*1024 - 4 // Last 4 bytes of PRG ROM

		if resetVectorOffset+1 < len(data) {
			resetVector := uint16(data[resetVectorOffset]) | (uint16(data[resetVectorOffset+1]) << 8)
			t.Logf("Reset vector: 0x%04X", resetVector)

			if resetVector < 0x8000 || resetVector >= 0xFFFF {
				t.Errorf("Invalid reset vector: 0x%04X (should be 0x8000-0xFFFF)", resetVector)
			}
		}
	})

	t.Run("ROM execution tracing", func(t *testing.T) {
		// Load cartridge
		reader := bytes.NewReader(data)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.LoadCartridge(cart)
		helper.UpdateReferences()

		// Reset and trace initial execution
		helper.Bus.Reset()

		t.Logf("Initial CPU state:")
		t.Logf("  PC: 0x%04X", helper.CPU.PC)
		t.Logf("  A: 0x%02X, X: 0x%02X, Y: 0x%02X", helper.CPU.A, helper.CPU.X, helper.CPU.Y)
		t.Logf("  SP: 0x%02X", helper.CPU.SP)
		t.Logf("  Flags: N:%t V:%t D:%t I:%t Z:%t C:%t", 
			helper.CPU.N, helper.CPU.V, helper.CPU.D, helper.CPU.I, helper.CPU.Z, helper.CPU.C)

		// Execute first several instructions and trace
		for step := 0; step < 50; step++ {
			pc := helper.CPU.PC
			
			// Read instruction at PC
			opcode := helper.Memory.Read(pc)
			
			// Log key instructions
			if step < 10 || opcode == 0x8D || opcode == 0xA9 || opcode == 0x85 {
				t.Logf("Step %d: PC=0x%04X, opcode=0x%02X", step, pc, opcode)
			}

			// Look for PPU register writes
			oldPC := pc
			helper.Bus.Step()
			
			// Check if PC moved significantly (could indicate issues)
			if helper.CPU.PC < 0x8000 && oldPC >= 0x8000 {
				t.Errorf("PC moved from ROM to RAM: 0x%04X -> 0x%04X", oldPC, helper.CPU.PC)
				break
			}

			// Check for infinite tight loops
			if helper.CPU.PC == oldPC {
				t.Logf("Infinite loop detected at 0x%04X after %d steps", helper.CPU.PC, step)
				break
			}

			// Stop if we've executed enough to see initial setup
			if step > 0 && step%10 == 0 {
				// Check PPU state periodically
				ppuCtrl := helper.PPU.ReadRegister(0x2000)
				ppuMask := helper.PPU.ReadRegister(0x2001)
				
				if ppuCtrl != 0 || ppuMask != 0 {
					t.Logf("PPU registers set after step %d: CTRL=0x%02X, MASK=0x%02X", 
						step, ppuCtrl, ppuMask)
				}
			}
		}

		// Final state check
		t.Logf("Final CPU state:")
		t.Logf("  PC: 0x%04X", helper.CPU.PC)
		t.Logf("  A: 0x%02X, X: 0x%02X, Y: 0x%02X", helper.CPU.A, helper.CPU.X, helper.CPU.Y)

		// Check PPU final state
		ppuCtrl := helper.PPU.ReadRegister(0x2000)
		ppuMask := helper.PPU.ReadRegister(0x2001)
		ppuStatus := helper.PPU.ReadRegister(0x2002)
		
		t.Logf("PPU registers:")
		t.Logf("  PPUCTRL: 0x%02X", ppuCtrl)
		t.Logf("  PPUMASK: 0x%02X", ppuMask)
		t.Logf("  PPUSTATUS: 0x%02X", ppuStatus)

		if ppuCtrl == 0 && ppuMask == 0 {
			t.Error("PPU registers still uninitialized - ROM execution may have failed")
		}
	})

	t.Run("Memory initialization check", func(t *testing.T) {
		reader := bytes.NewReader(data)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		helper := NewIntegrationTestHelper()
		helper.Cartridge = cart
		helper.Bus.LoadCartridge(cart)
		helper.UpdateReferences()
		helper.Bus.Reset()

		// Run for a reasonable amount of time to let initialization complete
		for i := 0; i < 10000; i++ {
			helper.Bus.Step()
			
			// Stop early if we see PPU initialization
			ppuMask := helper.PPU.ReadRegister(0x2001)
			if (ppuMask & 0x18) != 0 { // Background or sprite rendering enabled
				t.Logf("PPU rendering enabled after %d steps", i)
				break
			}
		}

		// Check if palette was loaded
		helper.PPU.WriteRegister(0x2006, 0x3F) // PPUADDR high
		helper.PPU.WriteRegister(0x2006, 0x00) // PPUADDR low
		universalBG := helper.PPU.ReadRegister(0x2007)

		if universalBG == 0x00 {
			t.Error("Universal background color still 0x00 - palette not loaded")
		} else {
			t.Logf("Universal background color: 0x%02X", universalBG)
		}

		// Check a few more palette entries
		for i := 1; i <= 3; i++ {
			helper.PPU.WriteRegister(0x2006, 0x3F)
			helper.PPU.WriteRegister(0x2006, uint8(i))
			color := helper.PPU.ReadRegister(0x2007)
			t.Logf("Palette[0x3F%02X]: 0x%02X", i, color)
		}

		// Check if nametable has any data
		helper.PPU.WriteRegister(0x2006, 0x20) // Nametable 0
		helper.PPU.WriteRegister(0x2006, 0x00)
		
		nonZeroTiles := 0
		for i := 0; i < 100; i++ { // Check first 100 tiles
			tile := helper.PPU.ReadRegister(0x2007)
			if tile != 0x00 {
				nonZeroTiles++
			}
		}
		
		t.Logf("Non-zero tiles in first 100 nametable entries: %d", nonZeroTiles)
		
		if nonZeroTiles == 0 {
			t.Error("Nametable appears empty - text not loaded")
		}
	})

	t.Run("CHR data verification", func(t *testing.T) {
		reader := bytes.NewReader(data)
		cart, err := cartridge.LoadFromReader(reader)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		// Directly check CHR data from cartridge
		t.Logf("CHR data sample (first 32 bytes):")
		for i := 0; i < 32; i += 8 {
			line := ""
			for j := 0; j < 8 && i+j < 32; j++ {
				line += fmt.Sprintf("%02X ", cart.ReadCHR(uint16(i+j)))
			}
			t.Logf("  %04X: %s", i, line)
		}

		// Check if CHR data is all zeros
		allZero := true
		for i := 0; i < 64; i++ { // Check first 4 tiles (64 bytes)
			if cart.ReadCHR(uint16(i)) != 0x00 {
				allZero = false
				break
			}
		}

		if allZero {
			t.Error("CHR ROM appears to be all zeros")
		} else {
			t.Log("CHR ROM contains pattern data")
		}
	})
}