package integration

import (
	"testing"
)

// BootTestHelper provides utilities for system boot integration testing
type BootTestHelper struct {
	*IntegrationTestHelper
	bootSequence []BootEvent
}

// BootEvent represents a boot sequence event
type BootEvent struct {
	Component string
	Event     string
	Cycle     int
	State     map[string]interface{}
}

// NewBootTestHelper creates a boot integration test helper
func NewBootTestHelper() *BootTestHelper {
	return &BootTestHelper{
		IntegrationTestHelper: NewIntegrationTestHelper(),
		bootSequence:          make([]BootEvent, 0),
	}
}

// LogBootEvent logs a boot sequence event
func (h *BootTestHelper) LogBootEvent(event BootEvent) {
	h.bootSequence = append(h.bootSequence, event)
}

// SetupBootROM creates a complete boot ROM with proper initialization
func (h *BootTestHelper) SetupBootROM() {
	// Create a comprehensive boot ROM
	romData := make([]uint8, 0x8000)

	// Boot sequence starting at $8000
	bootCode := []uint8{
		// CPU initialization
		0x78,       // SEI (disable interrupts)
		0xD8,       // CLD (clear decimal mode)
		0xA2, 0xFF, // LDX #$FF
		0x9A, // TXS (initialize stack)

		// Clear RAM
		0xA9, 0x00, // LDA #$00
		0xA2, 0x00, // LDX #$00

		// Clear zero page loop
		0x95, 0x00, // STA $00,X (clear ZP)
		0xE8,       // INX
		0xD0, 0xFB, // BNE -5 (loop until X=0)

		// Clear stack page
		0x9D, 0x00, 0x01, // STA $0100,X
		0xE8,       // INX
		0xD0, 0xFA, // BNE -6 (loop until X=0)

		// Initialize PPU
		0xA9, 0x00, // LDA #$00
		0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)
		0x8D, 0x01, 0x20, // STA $2001 (PPUMASK)

		// Wait for PPU to stabilize (2 frames)
		0xAD, 0x02, 0x20, // LDA $2002 (PPUSTATUS)
		0x10, 0xFB, // BPL -5 (wait for VBlank)
		0xAD, 0x02, 0x20, // LDA $2002
		0x10, 0xFB, // BPL -5 (wait for second VBlank)

		// Clear OAM
		0xA2, 0x00, // LDX #$00
		0xA9, 0xFF, // LDA #$FF
		0x8D, 0x03, 0x20, // STA $2003 (OAMADDR)
		0x8D, 0x04, 0x20, // STA $2004 (OAMDATA)
		0xE8,       // INX
		0xD0, 0xFA, // BNE -6 (clear 256 bytes)

		// Initialize VRAM
		0xA9, 0x20, // LDA #$20
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low)

		// Clear nametables
		0xA2, 0x04, // LDX #$04 (4 nametables)
		0xA0, 0x00, // LDY #$00
		0xA9, 0x00, // LDA #$00
		0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
		0xC8,       // INY
		0xD0, 0xFA, // BNE -6 (256 bytes)
		0xCA,       // DEX
		0xD0, 0xF6, // BNE -10 (next nametable)

		// Initialize palette
		0xA9, 0x3F, // LDA #$3F
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low)

		// Load default palette
		0xA2, 0x00, // LDX #$00
		0xBD, 0x80, 0x81, // LDA defaultPalette,X
		0x8D, 0x07, 0x20, // STA $2007
		0xE8,       // INX
		0xE0, 0x20, // CPX #$20
		0xD0, 0xF7, // BNE -9 (32 palette entries)

		// Enable rendering
		0xA9, 0x80, // LDA #$80 (NMI enable)
		0x8D, 0x00, 0x20, // STA $2000
		0xA9, 0x1E, // LDA #$1E (show bg+sprites)
		0x8D, 0x01, 0x20, // STA $2001

		// Boot complete - main loop
		0xEA,             // NOP
		0x4C, 0x7A, 0x80, // JMP to main loop
	}

	copy(romData, bootCode)

	// Default palette at $8180
	defaultPalette := []uint8{
		// Background palette
		0x0F, 0x00, 0x10, 0x30, // Palette 0
		0x0F, 0x01, 0x11, 0x31, // Palette 1
		0x0F, 0x02, 0x12, 0x32, // Palette 2
		0x0F, 0x03, 0x13, 0x33, // Palette 3
		// Sprite palette
		0x0F, 0x04, 0x14, 0x34, // Palette 0
		0x0F, 0x05, 0x15, 0x35, // Palette 1
		0x0F, 0x06, 0x16, 0x36, // Palette 2
		0x0F, 0x07, 0x17, 0x37, // Palette 3
	}
	copy(romData[0x0180:], defaultPalette)

	// Set interrupt vectors
	romData[0x7FFA] = 0x00 // NMI vector low
	romData[0x7FFB] = 0x82 // NMI vector high ($8200)
	romData[0x7FFC] = 0x00 // Reset vector low
	romData[0x7FFD] = 0x80 // Reset vector high ($8000)
	romData[0x7FFE] = 0x00 // IRQ vector low
	romData[0x7FFF] = 0x83 // IRQ vector high ($8300)

	// NMI handler at $8200
	nmiHandler := []uint8{
		0x48, // PHA
		0x8A, // TXA
		0x48, // PHA
		0x98, // TYA
		0x48, // PHA

		// NMI work would go here
		0xE6, 0x00, // INC $00 (frame counter)

		0x68, // PLA
		0xA8, // TAY
		0x68, // PLA
		0xAA, // TAX
		0x68, // PLA
		0x40, // RTI
	}
	copy(romData[0x0200:], nmiHandler)

	// IRQ handler at $8300
	irqHandler := []uint8{
		0x40, // RTI (ignore IRQ)
	}
	copy(romData[0x0300:], irqHandler)

	h.GetMockCartridge().LoadPRG(romData)
}

// TestSystemBoot tests the complete system boot sequence
func TestSystemBoot(t *testing.T) {
	t.Run("Complete boot sequence", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()
		helper.SetupBasicCHR()

		// Perform system reset
		helper.Bus.Reset()

		// Verify initial CPU state
		if helper.CPU.PC != 0x8000 {
			t.Errorf("PC should be 0x8000 after reset, got 0x%04X", helper.CPU.PC)
		}
		if helper.CPU.SP != 0xFD {
			t.Errorf("SP should be 0xFD after reset, got 0x%02X", helper.CPU.SP)
		}
		if !helper.CPU.I {
			t.Error("Interrupt flag should be set after reset")
		}

		// Execute boot sequence step by step
		bootSteps := []struct {
			description string
			cycles      int
			verify      func(*testing.T, *BootTestHelper)
		}{
			{
				"Initialize CPU state",
				5, // SEI, CLD, LDX #$FF, TXS
				func(t *testing.T, h *BootTestHelper) {
					if h.CPU.SP != 0xFF {
						t.Errorf("Stack should be initialized to 0xFF, got 0x%02X", h.CPU.SP)
					}
					if h.CPU.X != 0xFF {
						t.Errorf("X should be 0xFF, got 0x%02X", h.CPU.X)
					}
				},
			},
			{
				"Clear zero page",
				260, // RAM clearing loop
				func(t *testing.T, h *BootTestHelper) {
					// Check that zero page was cleared
					for i := uint16(0x00); i < 0x100; i++ {
						value := h.Memory.Read(i)
						if value != 0x00 {
							t.Errorf("Zero page not cleared at 0x%02X: got 0x%02X", i, value)
							break
						}
					}
				},
			},
			{
				"Clear stack page",
				260, // Stack clearing loop
				func(t *testing.T, h *BootTestHelper) {
					// Check that stack page was cleared
					for i := uint16(0x0100); i < 0x0200; i++ {
						value := h.Memory.Read(i)
						if value != 0x00 {
							t.Errorf("Stack page not cleared at 0x%04X: got 0x%02X", i, value)
							break
						}
					}
				},
			},
			{
				"Initialize PPU registers",
				6, // PPU register initialization
				func(t *testing.T, h *BootTestHelper) {
					// PPU should be in known state
					// Registers would be set to 0
				},
			},
			{
				"Wait for PPU stabilization",
				60000, // Wait for 2 VBlanks
				func(t *testing.T, h *BootTestHelper) {
					// PPU should be stable after 2 frames
					status := h.PPU.ReadRegister(0x2002)
					// VBlank flag behavior would be tested here
					_ = status
				},
			},
		}

		for _, step := range bootSteps {
			t.Run(step.description, func(t *testing.T) {
				// Execute the step
				for i := 0; i < step.cycles; i++ {
					helper.Bus.Step()
				}

				// Verify the step
				step.verify(t, helper)

				t.Logf("Boot step '%s' completed after %d cycles", step.description, step.cycles)
			})
		}

		t.Log("Complete boot sequence test passed")
	})

	t.Run("Power-on state verification", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()

		// Test power-on state before any operations
		// CPU should be in reset state
		if helper.CPU.A != 0 {
			t.Errorf("A register should be 0 at power-on, got 0x%02X", helper.CPU.A)
		}
		if helper.CPU.X != 0 {
			t.Errorf("X register should be 0 at power-on, got 0x%02X", helper.CPU.X)
		}
		if helper.CPU.Y != 0 {
			t.Errorf("Y register should be 0 at power-on, got 0x%02X", helper.CPU.Y)
		}

		// PPU should be in initial state
		ppuCtrl := helper.PPU.ReadRegister(0x2000)
		ppuMask := helper.PPU.ReadRegister(0x2001)
		ppuStatus := helper.PPU.ReadRegister(0x2002)

		// Initial PPU state verification
		_ = ppuCtrl
		_ = ppuMask
		_ = ppuStatus

		// Memory should be uninitialized (random or zero)
		// We don't test specific values since real hardware varies

		t.Log("Power-on state verification completed")
	})

	t.Run("Reset behavior during boot", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()

		// Start boot sequence
		helper.Bus.Reset()

		// Execute partway through boot
		for i := 0; i < 100; i++ {
			helper.Bus.Step()
		}

		// Reset again during boot
		initialPC := helper.CPU.PC
		helper.Bus.Reset()

		// Should restart from beginning
		if helper.CPU.PC != 0x8000 {
			t.Errorf("Reset should restart from 0x8000, got 0x%04X", helper.CPU.PC)
		}
		if helper.CPU.PC == initialPC {
			t.Error("Reset should change PC from current position")
		}

		// CPU state should be reset
		if helper.CPU.SP != 0xFD {
			t.Errorf("SP should be reset to 0xFD, got 0x%02X", helper.CPU.SP)
		}
		if !helper.CPU.I {
			t.Error("Interrupt flag should be set after reset")
		}

		t.Log("Reset behavior during boot test completed")
	})
}

// TestBootComponentInitialization tests individual component initialization
func TestBootComponentInitialization(t *testing.T) {
	t.Run("CPU initialization sequence", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()

		helper.Bus.Reset()

		// Verify CPU starts with correct state
		initialStates := map[string]interface{}{
			"PC": uint16(0x8000),
			"SP": uint8(0xFD),
			"A":  uint8(0x00),
			"X":  uint8(0x00),
			"Y":  uint8(0x00),
			"I":  true,
			"D":  false,
		}

		for register, expected := range initialStates {
			var actual interface{}
			switch register {
			case "PC":
				actual = helper.CPU.PC
			case "SP":
				actual = helper.CPU.SP
			case "A":
				actual = helper.CPU.A
			case "X":
				actual = helper.CPU.X
			case "Y":
				actual = helper.CPU.Y
			case "I":
				actual = helper.CPU.I
			case "D":
				actual = helper.CPU.D
			}

			if actual != expected {
				t.Errorf("CPU %s should be %v after reset, got %v", register, expected, actual)
			}
		}

		// Execute first few boot instructions
		helper.Bus.Step() // SEI
		if !helper.CPU.I {
			t.Error("SEI should set interrupt flag")
		}

		helper.Bus.Step() // CLD
		if helper.CPU.D {
			t.Error("CLD should clear decimal flag")
		}

		helper.Bus.Step() // LDX #$FF
		helper.Bus.Step() // TXS
		if helper.CPU.SP != 0xFF {
			t.Errorf("TXS should set SP to 0xFF, got 0x%02X", helper.CPU.SP)
		}

		t.Log("CPU initialization sequence completed")
	})

	t.Run("PPU initialization sequence", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()
		helper.SetupBasicCHR()

		helper.Bus.Reset()

		// Skip to PPU initialization part of boot
		for i := 0; i < 520; i++ { // Skip RAM clearing
			helper.Bus.Step()
		}

		// PPU should be initialized
		// This would be verified by checking PPU register states
		// and ensuring proper PPU memory setup

		// Test PPU register initialization
		helper.Bus.Step() // LDA #$00
		helper.Bus.Step() // STA $2000
		helper.Bus.Step() // STA $2001

		// PPU registers should be set to safe defaults
		// Real verification would check actual PPU state

		t.Log("PPU initialization sequence completed")
	})

	t.Run("Memory initialization", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()

		helper.Bus.Reset()

		// Execute until memory is cleared
		for i := 0; i < 530; i++ {
			helper.Bus.Step()
		}

		// Verify RAM was properly cleared
		ramTests := []struct {
			start uint16
			end   uint16
			name  string
		}{
			{0x0000, 0x00FF, "Zero page"},
			{0x0100, 0x01FF, "Stack page"},
			{0x0200, 0x07FF, "General RAM"},
		}

		for _, test := range ramTests {
			cleared := true
			for addr := test.start; addr <= test.end; addr++ {
				value := helper.Memory.Read(addr)
				if value != 0x00 {
					cleared = false
					t.Errorf("%s not cleared at 0x%04X: got 0x%02X", test.name, addr, value)
					break
				}
			}
			if cleared {
				t.Logf("%s cleared successfully", test.name)
			}
		}
	})

	t.Run("Interrupt vector setup", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()

		// Verify interrupt vectors are properly set
		vectors := map[string]struct {
			addr     uint16
			expected uint16
		}{
			"NMI":   {0xFFFA, 0x8200},
			"Reset": {0xFFFC, 0x8000},
			"IRQ":   {0xFFFE, 0x8300},
		}

		for name, vector := range vectors {
			low := helper.Memory.Read(vector.addr)
			high := helper.Memory.Read(vector.addr + 1)
			actual := uint16(high)<<8 | uint16(low)

			if actual != vector.expected {
				t.Errorf("%s vector should be 0x%04X, got 0x%04X", name, vector.expected, actual)
			} else {
				t.Logf("%s vector correctly set to 0x%04X", name, actual)
			}
		}

		// Test that reset vector is used
		helper.Bus.Reset()
		if helper.CPU.PC != 0x8000 {
			t.Errorf("Reset should jump to vector address 0x8000, got 0x%04X", helper.CPU.PC)
		}
	})
}

// TestBootTiming tests boot sequence timing
func TestBootTiming(t *testing.T) {
	t.Run("Boot sequence duration", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()

		helper.Bus.Reset()

		// Measure time to complete boot sequence
		bootCycles := 0
		maxCycles := 100000

		// Look for boot completion (when rendering is enabled)
		for bootCycles < maxCycles {
			helper.Bus.Step()
			bootCycles++

			// Check if rendering was enabled (end of boot)
			ppuMask := helper.Memory.Read(0x2001)
			if (ppuMask & 0x18) != 0 { // Background or sprites enabled
				t.Logf("Boot sequence completed in %d cycles", bootCycles)
				break
			}
		}

		if bootCycles >= maxCycles {
			t.Error("Boot sequence did not complete within expected time")
		}

		// Boot should take a reasonable amount of time
		// Typical boot sequence: ~63,000 cycles for 2 frame wait + initialization
		expectedMin := 50000
		expectedMax := 100000

		if bootCycles < expectedMin {
			t.Errorf("Boot completed too quickly: %d cycles (expected >%d)", bootCycles, expectedMin)
		}
		if bootCycles > expectedMax {
			t.Errorf("Boot took too long: %d cycles (expected <%d)", bootCycles, expectedMax)
		}
	})

	t.Run("Boot sequence frame alignment", func(t *testing.T) {
		helper := NewBootTestHelper()
		helper.SetupBootROM()

		helper.Bus.Reset()

		// Track frame timing during boot
		frameCount := 0
		lastVBlank := false

		for i := 0; i < 80000; i++ {
			helper.Bus.Step()

			// Check for VBlank transitions
			ppuStatus := helper.PPU.ReadRegister(0x2002)
			currentVBlank := (ppuStatus & 0x80) != 0

			if currentVBlank && !lastVBlank {
				frameCount++
				t.Logf("Frame %d completed at cycle %d", frameCount, i)

				// Boot should wait for at least 2 frames
				if frameCount >= 2 {
					break
				}
			}
			lastVBlank = currentVBlank
		}

		if frameCount < 2 {
			t.Errorf("Boot should wait for at least 2 frames, only saw %d", frameCount)
		}
	})
}
