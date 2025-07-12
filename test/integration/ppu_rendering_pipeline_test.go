package integration

import (
	"bytes"
	"io"
	"os"
	"testing"

	"gones/internal/cartridge"
)

// TestPPURenderingPipeline_DetailedAnalysis analyzes the PPU rendering pipeline step by step
func TestPPURenderingPipeline_DetailedAnalysis(t *testing.T) {
	// Load sample ROM
	file, err := os.Open("/home/claude/work/gones/roms/sample.nes")
	if err != nil {
		t.Fatalf("Failed to open sample.nes: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read sample.nes: %v", err)
	}

	reader := bytes.NewReader(data)
	cart, err := cartridge.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load sample.nes cartridge: %v", err)
	}

	helper := NewIntegrationTestHelper()
	helper.Cartridge = cart
	helper.Bus.LoadCartridge(cart)
	helper.UpdateReferences()

	t.Run("PPU State After ROM Boot", func(t *testing.T) {
		// Reset and run a few frames
		helper.Bus.Reset()
		for i := 0; i < 5; i++ {
			helper.RunFrame()
		}

		// Note: PPUCTRL and PPUMASK are write-only registers
		// Reading them returns open bus data, not actual register values
		// Instead, check the PPU's actual rendering state
		ppuStatus := helper.PPU.ReadRegister(0x2002)

		t.Logf("PPU State after 5 frames:")
		t.Logf("  PPUSTATUS: 0x%02X", ppuStatus)
		t.Logf("  Rendering Enabled: %v", helper.PPU.IsRenderingEnabled())

		// Check if rendering is properly enabled via internal state
		if !helper.PPU.IsRenderingEnabled() {
			t.Error("Background rendering is disabled!")
		} else {
			t.Logf("  PPU rendering properly enabled!")
		}
	})

	t.Run("Palette Memory Analysis", func(t *testing.T) {
		// Read all palette entries
		t.Logf("Palette Memory Contents:")

		// Background palettes
		t.Logf("  Background Palettes:")
		for pal := 0; pal < 4; pal++ {
			colors := make([]uint8, 4)
			for i := 0; i < 4; i++ {
				helper.PPU.WriteRegister(0x2006, 0x3F)
				helper.PPU.WriteRegister(0x2006, uint8(pal*4+i))
				colors[i] = helper.PPU.ReadRegister(0x2007)
			}
			t.Logf("    Palette %d: [0x%02X, 0x%02X, 0x%02X, 0x%02X]",
				pal, colors[0], colors[1], colors[2], colors[3])
		}

		// Sprite palettes
		t.Logf("  Sprite Palettes:")
		for pal := 0; pal < 4; pal++ {
			colors := make([]uint8, 4)
			for i := 0; i < 4; i++ {
				helper.PPU.WriteRegister(0x2006, 0x3F)
				helper.PPU.WriteRegister(0x2006, uint8(0x10+pal*4+i))
				colors[i] = helper.PPU.ReadRegister(0x2007)
			}
			t.Logf("    Palette %d: [0x%02X, 0x%02X, 0x%02X, 0x%02X]",
				pal, colors[0], colors[1], colors[2], colors[3])
		}

		// Check universal background color
		helper.PPU.WriteRegister(0x2006, 0x3F)
		helper.PPU.WriteRegister(0x2006, 0x00)
		universalBG := helper.PPU.ReadRegister(0x2007)
		t.Logf("  Universal Background: 0x%02X", universalBG)

		if universalBG == 0x00 {
			t.Error("Universal background color is 0x00 - palette not initialized!")
		}
	})

	t.Run("Nametable Content Analysis", func(t *testing.T) {
		// Check nametable content where text should be
		t.Logf("Nametable Analysis:")

		// Sample a few key locations
		locations := []struct {
			addr uint16
			desc string
		}{
			{0x2000, "Top-left corner"},
			{0x21CA, "Expected 'HELLO' location"},
			{0x21D4, "Expected 'WORLD' location"},
			{0x2100, "Row 4"},
			{0x2200, "Row 8"},
		}

		for _, loc := range locations {
			helper.PPU.WriteRegister(0x2006, uint8(loc.addr>>8))
			helper.PPU.WriteRegister(0x2006, uint8(loc.addr&0xFF))

			// Read 8 bytes at this location
			tiles := make([]uint8, 8)
			for i := 0; i < 8; i++ {
				tiles[i] = helper.PPU.ReadRegister(0x2007)
			}

			t.Logf("  0x%04X (%s): %02X %02X %02X %02X %02X %02X %02X %02X",
				loc.addr, loc.desc,
				tiles[0], tiles[1], tiles[2], tiles[3],
				tiles[4], tiles[5], tiles[6], tiles[7])
		}

		// Check if nametable has any non-zero content
		helper.PPU.WriteRegister(0x2006, 0x20)
		helper.PPU.WriteRegister(0x2006, 0x00)

		nonZeroCount := 0
		totalChecked := 100
		for i := 0; i < totalChecked; i++ {
			tile := helper.PPU.ReadRegister(0x2007)
			if tile != 0x00 {
				nonZeroCount++
			}
		}

		t.Logf("  Non-zero tiles in first %d entries: %d", totalChecked, nonZeroCount)

		if nonZeroCount == 0 {
			t.Error("Nametable appears to be empty - no text loaded!")
		}
	})

	t.Run("CHR Pattern Data Analysis", func(t *testing.T) {
		// Check CHR pattern data for expected character patterns
		t.Logf("CHR Pattern Data Analysis:")

		// Check a few character patterns
		characters := []struct {
			index uint16
			name  string
		}{
			{0, "Character 0 (space)"},
			{8, "Character 8 ('H')"},
			{5, "Character 5 ('E')"},
			{12, "Character 12 ('L')"},
			{15, "Character 15 ('O')"},
		}

		for _, char := range characters {
			t.Logf("  %s (tile %d):", char.name, char.index)

			// Read pattern data for this character (16 bytes)
			pattern := make([]uint8, 16)
			for i := 0; i < 16; i++ {
				pattern[i] = helper.Cartridge.ReadCHR(char.index*16 + uint16(i))
			}

			// Show first 8 bytes (low bit plane)
			t.Logf("    Low:  %02X %02X %02X %02X %02X %02X %02X %02X",
				pattern[0], pattern[1], pattern[2], pattern[3],
				pattern[4], pattern[5], pattern[6], pattern[7])

			// Show next 8 bytes (high bit plane)
			t.Logf("    High: %02X %02X %02X %02X %02X %02X %02X %02X",
				pattern[8], pattern[9], pattern[10], pattern[11],
				pattern[12], pattern[13], pattern[14], pattern[15])

			// Check if pattern is all zeros
			allZero := true
			for _, b := range pattern {
				if b != 0x00 {
					allZero = false
					break
				}
			}

			if allZero {
				t.Logf("    WARNING: Pattern is all zeros")
			}
		}
	})

	t.Run("Frame Buffer Update Test", func(t *testing.T) {
		// Clear frame buffer to a known pattern
		helper.PPU.ClearFrameBuffer(0xDEADBEEF)

		// Capture frame buffer before rendering
		beforeBuffer := helper.PPU.GetFrameBuffer()

		// Run one frame
		helper.RunFrame()

		// Capture frame buffer after rendering
		afterBuffer := helper.PPU.GetFrameBuffer()

		// Count changes
		changedPixels := 0
		for i := 0; i < len(beforeBuffer); i++ {
			if beforeBuffer[i] != afterBuffer[i] {
				changedPixels++
			}
		}

		t.Logf("Frame Buffer Update Test:")
		t.Logf("  Changed pixels: %d / %d", changedPixels, len(beforeBuffer))
		t.Logf("  Change percentage: %.2f%%", float64(changedPixels)/float64(len(beforeBuffer))*100)

		if changedPixels == 0 {
			t.Error("Frame buffer was not updated - rendering pipeline not working!")
		}

		// Sample some specific pixels
		samplePoints := []struct{ x, y int }{
			{0, 0}, {128, 120}, {255, 239}, {80, 116}, {144, 116},
		}

		t.Logf("  Sample pixels after rendering:")
		for _, point := range samplePoints {
			pixel := afterBuffer[point.y*256+point.x]
			t.Logf("    (%d,%d): 0x%08X", point.x, point.y, pixel)
		}
	})

	t.Run("PPU Memory Interface Test", func(t *testing.T) {
		// Test direct PPU memory access
		t.Logf("PPU Memory Interface Test:")

		// Test palette read/write
		testAddr := uint16(0x3F01)
		testValue := uint8(0x30)

		// Write via PPU registers
		helper.PPU.WriteRegister(0x2006, uint8(testAddr>>8))
		helper.PPU.WriteRegister(0x2006, uint8(testAddr&0xFF))
		helper.PPU.WriteRegister(0x2007, testValue)

		// Read back via PPU registers
		helper.PPU.WriteRegister(0x2006, uint8(testAddr>>8))
		helper.PPU.WriteRegister(0x2006, uint8(testAddr&0xFF))
		readValue := helper.PPU.ReadRegister(0x2007)

		t.Logf("  Palette write/read test: wrote 0x%02X, read 0x%02X", testValue, readValue)

		if readValue != testValue {
			t.Errorf("Palette write/read mismatch!")
		}

		// Test nametable access
		ntAddr := uint16(0x2100)
		ntValue := uint8(0x42)

		helper.PPU.WriteRegister(0x2006, uint8(ntAddr>>8))
		helper.PPU.WriteRegister(0x2006, uint8(ntAddr&0xFF))
		helper.PPU.WriteRegister(0x2007, ntValue)

		helper.PPU.WriteRegister(0x2006, uint8(ntAddr>>8))
		helper.PPU.WriteRegister(0x2006, uint8(ntAddr&0xFF))
		// PPU data reads are buffered for non-palette addresses
		// First read fills the buffer, second read returns the actual data
		helper.PPU.ReadRegister(0x2007) // Dummy read to fill buffer
		helper.PPU.WriteRegister(0x2006, uint8(ntAddr>>8))
		helper.PPU.WriteRegister(0x2006, uint8(ntAddr&0xFF))
		ntRead := helper.PPU.ReadRegister(0x2007)

		t.Logf("  Nametable write/read test: wrote 0x%02X, read 0x%02X", ntValue, ntRead)

		if ntRead != ntValue {
			t.Errorf("Nametable write/read mismatch!")
		}
	})

	t.Run("Rendering Enable/Disable Test", func(t *testing.T) {
		// Test what happens when we manually disable and enable rendering
		t.Logf("Rendering Enable/Disable Test:")

		// Disable rendering
		helper.PPU.WriteRegister(0x2001, 0x00) // Clear PPUMASK

		// Clear frame buffer
		helper.PPU.ClearFrameBuffer(0x11111111)

		// Run one frame with rendering disabled
		helper.RunFrame()

		afterDisabled := helper.PPU.GetFrameBuffer()

		// Check if frame buffer changed
		changedCount := 0
		for _, pixel := range afterDisabled {
			if pixel != 0x11111111 {
				changedCount++
			}
		}

		t.Logf("  With rendering disabled: %d pixels changed", changedCount)

		// Re-enable rendering
		helper.PPU.WriteRegister(0x2001, 0x1E) // Enable background and sprites

		// Clear frame buffer again
		helper.PPU.ClearFrameBuffer(0x22222222)

		// Run one frame with rendering enabled
		helper.RunFrame()

		afterEnabled := helper.PPU.GetFrameBuffer()

		// Check if frame buffer changed
		changedCount = 0
		for _, pixel := range afterEnabled {
			if pixel != 0x22222222 {
				changedCount++
			}
		}

		t.Logf("  With rendering enabled: %d pixels changed", changedCount)

		if changedCount == 0 {
			t.Error("Frame buffer not updated even with rendering enabled!")
		}
	})
}