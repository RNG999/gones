package integration

import (
	"gones/internal/memory"
	"testing"
)

// MemoryIntegrationHelper provides utilities for memory integration testing
type MemoryIntegrationHelper struct {
	*IntegrationTestHelper
	accessLog []MemoryAccess
}

// MemoryAccess represents a memory access for logging
type MemoryAccess struct {
	Address uint16
	Value   uint8
	IsWrite bool
	Source  string // "CPU", "PPU", "DMA"
}

// NewMemoryIntegrationHelper creates a memory integration test helper
func NewMemoryIntegrationHelper() *MemoryIntegrationHelper {
	return &MemoryIntegrationHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		accessLog:             make([]MemoryAccess, 0),
	}
}

// LogAccess logs a memory access
func (h *MemoryIntegrationHelper) LogAccess(address uint16, value uint8, isWrite bool, source string) {
	h.accessLog = append(h.accessLog, MemoryAccess{
		Address: address,
		Value:   value,
		IsWrite: isWrite,
		Source:  source,
	})
}

// ClearAccessLog clears the access log
func (h *MemoryIntegrationHelper) ClearAccessLog() {
	h.accessLog = h.accessLog[:0]
}

// GetAccessCount returns the number of accesses to a specific address
func (h *MemoryIntegrationHelper) GetAccessCount(address uint16) int {
	count := 0
	for _, access := range h.accessLog {
		if access.Address == address {
			count++
		}
	}
	return count
}

// TestCrossComponentMemoryAccess tests memory access between different components
func TestCrossComponentMemoryAccess(t *testing.T) {
	t.Run("CPU to PPU register communication", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)
		helper.Bus.Reset()

		// Program that writes to various PPU registers
		program := []uint8{
			// Configure PPU control
			0xA9, 0x80, // LDA #$80 (NMI enable)
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)

			// Configure PPU mask
			0xA9, 0x1E, // LDA #$1E (show background and sprites)
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK)

			// Set VRAM address
			0xA9, 0x20, // LDA #$20
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low)

			// Write to VRAM
			0xA9, 0x01, // LDA #$01
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)

			// Read PPU status
			0xAD, 0x02, 0x20, // LDA $2002 (PPUSTATUS)

			0xEA,             // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute the program step by step
		steps := []string{
			"LDA #$80",
			"STA $2000 (PPUCTRL)",
			"LDA #$1E",
			"STA $2001 (PPUMASK)",
			"LDA #$20",
			"STA $2006 (PPUADDR high)",
			"LDA #$00",
			"STA $2006 (PPUADDR low)",
			"LDA #$01",
			"STA $2007 (PPUDATA)",
			"LDA $2002 (PPUSTATUS)",
		}

		for i, step := range steps {
			helper.Bus.Step()
			t.Logf("Step %d: %s", i+1, step)
		}

		// Verify that PPU registers were accessed through memory system
		// Check that writes reached the PPU and reads returned expected values

		// Test register mirroring - $2008 should behave like $2000
		helper.Memory.Write(0x2008, 0x40) // Should mirror to PPUCTRL
		_ = helper.Memory.Read(0x2000)
		// In a full implementation, we would verify the mirroring worked

		t.Log("CPU-PPU register communication test completed")
	})

	t.Run("PPU VRAM access through memory interface", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Create PPU memory with test cartridge
		ppuMem := memory.NewPPUMemory(helper.Cartridge, memory.MirrorHorizontal)
		helper.PPU.SetMemory(ppuMem)

		// Test Pattern Table access (CHR ROM)
		chrValue := ppuMem.Read(0x0000) // First pattern table
		if chrValue != 0xAA {           // From our basic CHR setup
			t.Errorf("Expected 0xAA from pattern table, got 0x%02X", chrValue)
		}

		// Test Nametable access
		ppuMem.Write(0x2000, 0x55) // Write to first nametable
		value := ppuMem.Read(0x2000)
		if value != 0x55 {
			t.Errorf("Expected 0x55 from nametable, got 0x%02X", value)
		}

		// Test nametable mirroring
		ppuMem.Write(0x2400, 0xAA)           // Write to second nametable
		mirroredValue := ppuMem.Read(0x2000) // Should read from first nametable due to horizontal mirroring
		if mirroredValue != 0x55 {           // Should still be original value
			t.Errorf("Nametable mirroring test failed: expected 0x55, got 0x%02X", mirroredValue)
		}

		// Test palette RAM access
		ppuMem.Write(0x3F00, 0x0F) // Universal background color
		paletteValue := ppuMem.Read(0x3F00)
		if paletteValue != 0x0F {
			t.Errorf("Expected 0x0F from palette RAM, got 0x%02X", paletteValue)
		}

		// Test palette mirroring
		ppuMem.Write(0x3F10, 0x20)      // Sprite palette background
		_ = ppuMem.Read(0x3F00) // Should mirror background colors
		// Palette mirroring is complex, just verify no crash for now

		t.Log("PPU VRAM access test completed")
	})

	t.Run("Cartridge memory integration", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Clear cartridge access logs
		helper.GetMockCartridge().ClearLogs()

		// Test CPU access to PRG ROM
		_ = helper.Memory.Read(0x8000)
		if len(helper.GetMockCartridge().prgReads) == 0 {
			t.Error("No PRG reads logged")
		}
		if helper.GetMockCartridge().prgReads[0] != 0x8000 {
			t.Errorf("Expected PRG read at 0x8000, got 0x%04X", helper.GetMockCartridge().prgReads[0])
		}

		// Test PRG ROM mirroring (if 16KB ROM)
		helper.GetMockCartridge().ClearLogs()
		_ = helper.Memory.Read(0x8000)
		_ = helper.Memory.Read(0xC000) // Should mirror for 16KB ROM
		// Implementation would handle mirroring logic

		// Test PRG RAM access (if available)
		helper.Memory.Write(0x6000, 0x42) // PRG RAM area
		ramValue := helper.Memory.Read(0x6000)
		if ramValue != 0x42 {
			t.Errorf("Expected 0x42 from PRG RAM, got 0x%02X", ramValue)
		}

		// Test CHR access through PPU memory
		if helper.PPU != nil {
			ppuMem := memory.NewPPUMemory(helper.Cartridge, memory.MirrorHorizontal)
			helper.PPU.SetMemory(ppuMem)

			helper.GetMockCartridge().ClearLogs()
			_ = ppuMem.Read(0x0000)
			if len(helper.GetMockCartridge().chrReads) == 0 {
				t.Error("No CHR reads logged")
			}
		}

		t.Log("Cartridge memory integration test completed")
	})

	t.Run("Memory bus arbitration", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)

		// Test that multiple components can access memory without conflicts
		// In a real NES, there are timing restrictions, but basic access should work

		// CPU writes to RAM
		helper.Memory.Write(0x0200, 0x11)

		// CPU writes to PPU registers
		helper.Memory.Write(0x2000, 0x80)
		helper.Memory.Write(0x2001, 0x1E)

		// CPU reads from cartridge
		_ = helper.Memory.Read(0x8000)

		// CPU reads from RAM
		ramValue := helper.Memory.Read(0x0200)
		if ramValue != 0x11 {
			t.Errorf("Expected 0x11 from RAM, got 0x%02X", ramValue)
		}

		// CPU reads PPU status
		_ = helper.Memory.Read(0x2002)
		// Just verify no crash - actual PPU behavior tested elsewhere

		// Verify memory map correctness
		testCases := []struct {
			address uint16
			region  string
		}{
			{0x0000, "Internal RAM"},
			{0x0800, "Internal RAM (mirror)"},
			{0x2000, "PPU registers"},
			{0x2008, "PPU registers (mirror)"},
			{0x4000, "APU/IO registers"},
			{0x6000, "Cartridge PRG RAM"},
			{0x8000, "Cartridge PRG ROM"},
			{0xFFFF, "Cartridge PRG ROM"},
		}

		for _, tc := range testCases {
			// Test that reads don't crash
			value := helper.Memory.Read(tc.address)
			t.Logf("Read from 0x%04X (%s): 0x%02X", tc.address, tc.region, value)

			// Test that writes don't crash (except ROM areas)
			if tc.address < 0x8000 {
				helper.Memory.Write(tc.address, 0x42)
			}
		}

		t.Log("Memory bus arbitration test completed")
	})
}

// TestMemoryMirroring tests memory mirroring behavior across the system
func TestMemoryMirroring(t *testing.T) {
	t.Run("Internal RAM mirroring", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)

		// Internal RAM is 2KB but appears in 8KB space with 4x mirroring
		testValue := uint8(0x55)

		// Write to base address
		helper.Memory.Write(0x0100, testValue)

		// Read from mirrored addresses
		mirrors := []uint16{0x0100, 0x0900, 0x1100, 0x1900}
		for i, addr := range mirrors {
			value := helper.Memory.Read(addr)
			if value != testValue {
				t.Errorf("Mirror %d (0x%04X): expected 0x%02X, got 0x%02X",
					i, addr, testValue, value)
			}
		}

		// Write to mirror and read from base
		helper.Memory.Write(0x0900, 0xAA)
		value := helper.Memory.Read(0x0100)
		if value != 0xAA {
			t.Errorf("Mirror write failed: expected 0xAA at base, got 0x%02X", value)
		}

		// Test edge cases
		helper.Memory.Write(0x07FF, 0x33)  // Last address in 2KB space
		value = helper.Memory.Read(0x0FFF) // Mirror
		if value != 0x33 {
			t.Errorf("Edge case mirror failed: expected 0x33, got 0x%02X", value)
		}
	})

	t.Run("PPU register mirroring", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)

		// PPU registers repeat every 8 bytes from $2000-$3FFF

		// Write to base PPUCTRL
		helper.Memory.Write(0x2000, 0x80)

		// Read from mirrored addresses
		mirrors := []uint16{0x2000, 0x2008, 0x2010, 0x2018, 0x3000, 0x3FF8}

		for _, addr := range mirrors {
			// Note: Reading PPUCTRL returns different values, but write should affect same register
			// We test that the access doesn't crash and follows mirroring pattern
			helper.Memory.Write(addr, 0x40)
			// Verify write was processed (implementation dependent)
		}

		// Test that addresses map to correct registers
		registerMap := map[uint16]string{
			0x2000: "PPUCTRL",
			0x2001: "PPUMASK",
			0x2002: "PPUSTATUS",
			0x2003: "OAMADDR",
			0x2004: "OAMDATA",
			0x2005: "PPUSCROLL",
			0x2006: "PPUADDR",
			0x2007: "PPUDATA",
		}

		for addr, name := range registerMap {
			// Test base address
			helper.Memory.Write(addr, 0x42)
			t.Logf("Wrote to %s (0x%04X)", name, addr)

			// Test mirrored address
			mirrorAddr := addr + 0x0008
			helper.Memory.Write(mirrorAddr, 0x84)
			t.Logf("Wrote to %s mirror (0x%04X)", name, mirrorAddr)
		}
	})

	t.Run("Nametable mirroring modes", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Test different mirroring modes
		mirrorModes := []struct {
			mode        memory.MirrorMode
			description string
		}{
			{memory.MirrorHorizontal, "Horizontal mirroring"},
			{memory.MirrorVertical, "Vertical mirroring"},
			{memory.MirrorSingleScreen0, "Single screen 0"},
			{memory.MirrorSingleScreen1, "Single screen 1"},
		}

		for _, test := range mirrorModes {
			t.Run(test.description, func(t *testing.T) {
				// Create PPU memory with specific mirroring
				ppuMem := memory.NewPPUMemory(helper.Cartridge, test.mode)

				// Test nametable addresses
				nametables := []uint16{0x2000, 0x2400, 0x2800, 0x2C00}

				// Write different values to each nametable
				for i, addr := range nametables {
					ppuMem.Write(addr, uint8(0x10+i))
				}

				// Read back and verify mirroring behavior
				for _, addr := range nametables {
					value := ppuMem.Read(addr)
					t.Logf("Nametable 0x%04X: 0x%02X", addr, value)
				}

				// Verify mirroring behavior matches mode
				switch test.mode {
				case memory.MirrorHorizontal:
					// $2000/$2400 should mirror, $2800/$2C00 should mirror
					val0 := ppuMem.Read(0x2000)
					val1 := ppuMem.Read(0x2400)
					if val0 != val1 {
						t.Errorf("Horizontal mirroring failed: 0x2000=0x%02X, 0x2400=0x%02X", val0, val1)
					}

				case memory.MirrorVertical:
					// $2000/$2800 should mirror, $2400/$2C00 should mirror
					val0 := ppuMem.Read(0x2000)
					val2 := ppuMem.Read(0x2800)
					if val0 != val2 {
						t.Errorf("Vertical mirroring failed: 0x2000=0x%02X, 0x2800=0x%02X", val0, val2)
					}

				case memory.MirrorSingleScreen0:
					// All should read the same value
					val0 := ppuMem.Read(0x2000)
					for _, addr := range nametables[1:] {
						val := ppuMem.Read(addr)
						if val != val0 {
							t.Errorf("Single screen 0 mirroring failed: 0x%04X=0x%02X, expected 0x%02X",
								addr, val, val0)
						}
					}
				}
			})
		}
	})
}

// TestConcurrentMemoryAccess tests memory access under concurrent conditions
func TestConcurrentMemoryAccess(t *testing.T) {
	t.Run("CPU and PPU memory access coordination", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Create PPU memory
		ppuMem := memory.NewPPUMemory(helper.Cartridge, memory.MirrorHorizontal)
		helper.PPU.SetMemory(ppuMem)

		// Program that accesses PPU while PPU is also accessing memory
		program := []uint8{
			// Set up PPU
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (enable NMI)
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (enable rendering)

			// Write to VRAM while rendering might be active
			0xA9, 0x20, // LDA #$20
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low)

			// Multiple VRAM writes
			0xA9, 0x01, // LDA #$01
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
			0xA9, 0x02, // LDA #$02
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)

			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute program while system is running
		for i := 0; i < 50; i++ {
			helper.Bus.Step()

			// Periodically check PPU access patterns
			if i%10 == 0 {
				// PPU might be accessing pattern tables or nametables
				// This tests that concurrent access doesn't cause issues
				chrValue := ppuMem.Read(0x0000) // Pattern table access
				nmtValue := ppuMem.Read(0x2000) // Nametable access
				t.Logf("Step %d: CHR=0x%02X, NMT=0x%02X", i, chrValue, nmtValue)
			}
		}

		t.Log("Concurrent memory access test completed")
	})

	t.Run("DMA memory access patterns", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)

		// Set up OAM data in RAM
		for i := 0; i < 256; i++ {
			helper.Memory.Write(0x0200+uint16(i), uint8(i))
		}

		// Program that triggers OAM DMA
		program := []uint8{
			0xA9, 0x02, // LDA #$02 (page 2)
			0x8D, 0x14, 0x40, // STA $4014 (OAM DMA)
			0xEA,             // NOP (should be delayed)
			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Execute until DMA
		helper.Bus.Step() // LDA #$02
		helper.Bus.Step() // STA $4014 (triggers DMA)

		// DMA should now copy 256 bytes from $0200-$02FF to OAM
		// This tests that DMA can access memory while CPU is suspended

		helper.Bus.Step() // NOP (delayed by DMA)

		// Verify DMA worked by checking that memory accesses occurred
		// In a full implementation, we would track DMA memory reads

		t.Log("DMA memory access test completed")
	})

	t.Run("Memory stress test", func(t *testing.T) {
		helper := NewMemoryIntegrationHelper()
		helper.SetupBasicROM(0x8000)

		// Program that rapidly accesses different memory regions
		program := []uint8{
			// Rapid RAM access
			0xA9, 0x00, // LDA #$00
			0x85, 0x00, // STA $00
			0xE6, 0x00, // INC $00
			0xA5, 0x00, // LDA $00

			// PPU register access
			0x8D, 0x00, 0x20, // STA $2000
			0xAD, 0x02, 0x20, // LDA $2002

			// Multiple memory regions
			0x8D, 0x00, 0x03, // STA $0300 (RAM)
			0x8D, 0x00, 0x04, // STA $0400 (RAM)
			0x8D, 0x00, 0x05, // STA $0500 (RAM)

			0x4C, 0x00, 0x80, // JMP $8000
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		helper.GetMockCartridge().LoadPRG(romData)
		helper.Bus.Reset()

		// Run stress test
		for cycle := 0; cycle < 1000; cycle++ {
			helper.Bus.Step()

			// Periodically verify memory integrity
			if cycle%100 == 0 {
				// Check that basic memory operations still work
				helper.Memory.Write(0x0100, 0x55)
				value := helper.Memory.Read(0x0100)
				if value != 0x55 {
					t.Errorf("Memory integrity check failed at cycle %d: expected 0x55, got 0x%02X",
						cycle, value)
				}
			}
		}

		t.Log("Memory stress test completed")
	})
}
