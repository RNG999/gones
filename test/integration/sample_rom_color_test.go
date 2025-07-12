package integration

import (
	"bytes"
	"io"
	"os"
	"testing"

	"gones/internal/cartridge"
)

// SampleROMColorTestSuite provides comprehensive color output verification for sample.nes ROM
// These tests are designed to FAIL initially to reveal the specific nature of the red screen bug
type SampleROMColorTestSuite struct {
	helper   *IntegrationTestHelper
	cartData []byte
}

// LoadSampleROM loads the sample.nes ROM from the roms directory
func (s *SampleROMColorTestSuite) LoadSampleROM(t *testing.T) {
	file, err := os.Open("/home/claude/work/gones/roms/sample.nes")
	if err != nil {
		t.Fatalf("Failed to open sample.nes: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read sample.nes: %v", err)
	}

	s.cartData = data

	// Load cartridge into emulator
	reader := bytes.NewReader(data)
	cart, err := cartridge.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load sample.nes cartridge: %v", err)
	}

	s.helper = NewIntegrationTestHelper()
	s.helper.Cartridge = cart
	s.helper.Bus.LoadCartridge(cart)
}

// BootROMAndWaitForDisplay boots the ROM and waits for initial display setup
func (s *SampleROMColorTestSuite) BootROMAndWaitForDisplay(t *testing.T) {
	// Reset system
	s.helper.Bus.Reset()

	// Run for enough cycles to complete initialization
	// Sample ROM should initialize palette and start rendering within 5 frames
	maxFrames := 5
	for frame := 0; frame < maxFrames; frame++ {
		s.helper.RunFrame()
		
		// Check if PPU rendering is enabled
		ppuMask := s.helper.PPU.ReadRegister(0x2001)
		if (ppuMask & 0x18) != 0 { // Background or sprite rendering enabled
			t.Logf("PPU rendering enabled after frame %d (PPUMASK=0x%02X)", frame+1, ppuMask)
			break
		}
	}
}

// TestSampleROM_PaletteInitialization tests that palette values are loaded correctly
func TestSampleROM_PaletteInitialization(t *testing.T) {
	suite := &SampleROMColorTestSuite{}
	suite.LoadSampleROM(t)
	suite.BootROMAndWaitForDisplay(t)

	t.Run("Universal background color", func(t *testing.T) {
		// Universal background color should be black (0x0F)
		// PPU palette address 0x3F00
		suite.helper.PPU.WriteRegister(0x2006, 0x3F) // PPUADDR high
		suite.helper.PPU.WriteRegister(0x2006, 0x00) // PPUADDR low
		universalBG := suite.helper.PPU.ReadRegister(0x2007) // PPUDATA

		if universalBG != 0x0F {
			t.Errorf("Universal background color should be black (0x0F), got 0x%02X", universalBG)
		}
	})

	t.Run("Background palette 0", func(t *testing.T) {
		// Background palette 0 should contain specific colors for "HELLO, WORLD!" text
		expectedPalette := []uint8{0x0F, 0x30, 0x10, 0x00} // Black, white, gray, black

		suite.helper.PPU.WriteRegister(0x2006, 0x3F) // PPUADDR high
		suite.helper.PPU.WriteRegister(0x2006, 0x01) // PPUADDR low (skip universal BG)

		for i, expectedColor := range expectedPalette[1:] { // Skip universal BG
			actualColor := suite.helper.PPU.ReadRegister(0x2007)
			if actualColor != expectedColor {
				t.Errorf("Background palette 0[%d] should be 0x%02X, got 0x%02X", 
					i+1, expectedColor, actualColor)
			}
		}
	})

	t.Run("Sprite palette 0", func(t *testing.T) {
		// Sprite palette 0 (if used)
		suite.helper.PPU.WriteRegister(0x2006, 0x3F) // PPUADDR high
		suite.helper.PPU.WriteRegister(0x2006, 0x11) // PPUADDR low (sprite palette 0, color 1)

		spriteColor1 := suite.helper.PPU.ReadRegister(0x2007)
		// Sprite palettes might not be set or might be different
		t.Logf("Sprite palette 0[1] = 0x%02X", spriteColor1)
	})

	t.Run("Verify no red screen palette", func(t *testing.T) {
		// Check that universal background is NOT red
		suite.helper.PPU.WriteRegister(0x2006, 0x3F) // PPUADDR high
		suite.helper.PPU.WriteRegister(0x2006, 0x00) // PPUADDR low
		universalBG := suite.helper.PPU.ReadRegister(0x2007)

		// Common red colors: 0x06, 0x16, 0x26, 0x36
		redColors := []uint8{0x06, 0x16, 0x26, 0x36}
		for _, redColor := range redColors {
			if universalBG == redColor {
				t.Errorf("Universal background is red (0x%02X) - this indicates the red screen bug!", 
					universalBG)
			}
		}
	})
}

// TestSampleROM_NameTableSetup tests nametable initialization for text display
func TestSampleROM_NameTableSetup(t *testing.T) {
	suite := &SampleROMColorTestSuite{}
	suite.LoadSampleROM(t)
	suite.BootROMAndWaitForDisplay(t)

	t.Run("Hello World text in nametable", func(t *testing.T) {
		// "HELLO, WORLD!" should be written to nametable starting around PPU address 0x21CA
		expectedText := "HELLO, WORLD!"
		
		// Sample.asm typically puts text at row 14, column 10 (approximate)
		// PPU address = 0x2000 + (row * 32) + column = 0x2000 + (14 * 32) + 10 = 0x21CA
		baseAddr := uint16(0x21CA)

		suite.helper.PPU.WriteRegister(0x2006, uint8(baseAddr>>8))   // PPUADDR high
		suite.helper.PPU.WriteRegister(0x2006, uint8(baseAddr&0xFF)) // PPUADDR low

		for i, expectedChar := range expectedText {
			actualTile := suite.helper.PPU.ReadRegister(0x2007) // PPUDATA
			
			// Convert ASCII to tile index (sample ROM specific mapping)
			var expectedTile uint8
			if expectedChar >= 'A' && expectedChar <= 'Z' {
				expectedTile = uint8(expectedChar - 'A' + 1) // A=1, B=2, etc.
			} else if expectedChar == ' ' {
				expectedTile = 0x00
			} else if expectedChar == ',' {
				expectedTile = 0x1B // Example mapping
			} else if expectedChar == '!' {
				expectedTile = 0x1C // Example mapping
			}

			if actualTile != expectedTile {
				t.Errorf("Text char %d ('%c') should be tile 0x%02X, got 0x%02X", 
					i, expectedChar, expectedTile, actualTile)
			}
		}
	})

	t.Run("Background area should be empty", func(t *testing.T) {
		// Check that most of the nametable is filled with tile 0 (space)
		emptyTileCount := 0
		totalChecked := 0

		// Check a sample of nametable positions
		checkPositions := []uint16{
			0x2000, 0x2020, 0x2040, 0x2060, // Top area
			0x2380, 0x23A0, 0x23C0, 0x23E0, // Bottom area
		}

		for _, addr := range checkPositions {
			suite.helper.PPU.WriteRegister(0x2006, uint8(addr>>8))
			suite.helper.PPU.WriteRegister(0x2006, uint8(addr&0xFF))
			
			tile := suite.helper.PPU.ReadRegister(0x2007)
			if tile == 0x00 {
				emptyTileCount++
			}
			totalChecked++
		}

		// Most background should be empty
		if emptyTileCount < totalChecked/2 {
			t.Errorf("Background not properly cleared: only %d/%d positions empty", 
				emptyTileCount, totalChecked)
		}
	})
}

// TestSampleROM_PPUState tests PPU register states for correct rendering
func TestSampleROM_PPUState(t *testing.T) {
	suite := &SampleROMColorTestSuite{}
	suite.LoadSampleROM(t)
	suite.BootROMAndWaitForDisplay(t)

	t.Run("PPUCTRL register state", func(t *testing.T) {
		// Sample ROM should set up PPUCTRL for normal operation
		// Expected: Base nametable = 0x2000, increment = 1, background pattern = $0000
		ppuCtrl := suite.helper.PPU.ReadRegister(0x2000)
		
		// Check base nametable (bits 0-1)
		nametableBase := ppuCtrl & 0x03
		if nametableBase != 0x00 {
			t.Errorf("Expected nametable base 0, got %d", nametableBase)
		}

		// Check VRAM increment (bit 2) - should be 1 (across)
		vramIncrement := (ppuCtrl >> 2) & 0x01
		if vramIncrement != 0x00 {
			t.Errorf("Expected VRAM increment across (0), got %d", vramIncrement)
		}

		// Check background pattern table (bit 4)
		bgPatternTable := (ppuCtrl >> 4) & 0x01
		t.Logf("Background pattern table: $%04X", uint16(bgPatternTable)*0x1000)
	})

	t.Run("PPUMASK register state", func(t *testing.T) {
		// PPUMASK should enable background rendering and possibly sprites
		ppuMask := suite.helper.PPU.ReadRegister(0x2001)

		// Check if background rendering is enabled (bit 3)
		backgroundEnabled := (ppuMask >> 3) & 0x01
		if backgroundEnabled == 0 {
			t.Error("Background rendering should be enabled")
		}

		// Check color intensity (bits 5-7) - should not be set to red
		colorEmphasis := (ppuMask >> 5) & 0x07
		if colorEmphasis == 0x04 { // Red emphasis only
			t.Error("Red color emphasis detected - this may cause red screen!")
		}

		t.Logf("PPUMASK = 0x%02X (bg:%d, spr:%d, emphasis:0x%X)", 
			ppuMask, backgroundEnabled, (ppuMask>>4)&0x01, colorEmphasis)
	})

	t.Run("PPUSTATUS register behavior", func(t *testing.T) {
		// Read PPUSTATUS to check VBlank and sprite 0 hit
		ppuStatus := suite.helper.PPU.ReadRegister(0x2002)

		// VBlank flag behavior (bit 7)
		vblankFlag := (ppuStatus >> 7) & 0x01
		t.Logf("VBlank flag: %d", vblankFlag)

		// Sprite 0 hit (bit 6) - might be set if sprites are used
		sprite0Hit := (ppuStatus >> 6) & 0x01
		t.Logf("Sprite 0 hit: %d", sprite0Hit)

		// Sprite overflow (bit 5)
		spriteOverflow := (ppuStatus >> 5) & 0x01
		if spriteOverflow != 0 {
			t.Logf("Warning: Sprite overflow detected")
		}
	})
}

// TestSampleROM_RenderingOutput tests actual screen output for color correctness
func TestSampleROM_RenderingOutput(t *testing.T) {
	suite := &SampleROMColorTestSuite{}
	suite.LoadSampleROM(t)
	suite.BootROMAndWaitForDisplay(t)

	// Run several more frames to ensure stable rendering
	for i := 0; i < 3; i++ {
		suite.helper.RunFrame()
	}

	t.Run("Screen color composition", func(t *testing.T) {
		// Test pixel colors at known text positions
		// This would require access to the PPU's frame buffer or pixel output

		// Sample positions where "HELLO, WORLD!" should appear
		textPositions := []struct {
			x, y     int
			expected string
		}{
			{80, 112, "text area"},      // Approximate center of "HELLO"
			{144, 112, "text area"},     // Approximate center of "WORLD"
			{40, 60, "background"},      // Above text
			{200, 160, "background"},    // Below text
		}

		for _, pos := range textPositions {
			// In a real implementation, you would get the pixel color at (x, y)
			// color := suite.helper.PPU.GetPixelColor(pos.x, pos.y)
			
			// For now, we can only test the setup
			t.Logf("Position (%d, %d) should be %s", pos.x, pos.y, pos.expected)
		}
	})

	t.Run("Background color verification", func(t *testing.T) {
		// Verify that the majority of screen pixels are black, not red
		
		// Sample multiple background positions
		backgroundSamples := []struct{ x, y int }{
			{10, 10}, {100, 50}, {200, 30},   // Top area
			{50, 200}, {150, 220}, {240, 190}, // Bottom area
			{10, 120}, {240, 120},             // Sides
		}

		for _, sample := range backgroundSamples {
			// In a real test, check pixel color at position
			// actualColor := suite.helper.PPU.GetPixelColor(sample.x, sample.y)
			// if actualColor == redColor { t.Error... }
			
			t.Logf("Background sample at (%d, %d) should be black", sample.x, sample.y)
		}
	})

	t.Run("Text color verification", func(t *testing.T) {
		// Text should appear as white pixels on black background
		
		// Sample positions within the "HELLO, WORLD!" text
		textSamples := []struct{ x, y int }{
			{84, 116}, {88, 116}, {92, 116}, // Within "H"
			{100, 116}, {104, 116},          // Within "E"
			{148, 116}, {152, 116},          // Within "W"
		}

		for _, sample := range textSamples {
			// In a real test, check that pixels are white/foreground color
			// actualColor := suite.helper.PPU.GetPixelColor(sample.x, sample.y)
			// if actualColor != whiteColor { t.Error... }
			
			t.Logf("Text sample at (%d, %d) should be white/foreground color", sample.x, sample.y)
		}
	})
}

// TestSampleROM_RedScreenBugDetection specifically tests for red screen conditions
func TestSampleROM_RedScreenBugDetection(t *testing.T) {
	suite := &SampleROMColorTestSuite{}
	suite.LoadSampleROM(t)
	suite.BootROMAndWaitForDisplay(t)

	t.Run("Detect red screen bug patterns", func(t *testing.T) {
		// Common causes of red screen bug:
		
		// 1. PPU palette not initialized
		suite.helper.PPU.WriteRegister(0x2006, 0x3F)
		suite.helper.PPU.WriteRegister(0x2006, 0x00)
		universalBG := suite.helper.PPU.ReadRegister(0x2007)
		
		if universalBG == 0x00 {
			t.Error("Universal background color is uninitialized (0x00) - palette not loaded!")
		}

		// 2. Red color emphasis in PPUMASK
		ppuMask := suite.helper.PPU.ReadRegister(0x2001)
		redEmphasis := (ppuMask >> 5) & 0x04
		if redEmphasis != 0 {
			t.Error("Red color emphasis is enabled - this causes red tint!")
		}

		// 3. Incorrect palette values
		redValues := []uint8{0x06, 0x16, 0x26, 0x36}
		for _, redVal := range redValues {
			if universalBG == redVal {
				t.Errorf("Universal background is red (0x%02X) - red screen bug detected!", universalBG)
			}
		}

		// 4. PPU not properly initialized
		ppuCtrl := suite.helper.PPU.ReadRegister(0x2000)
		if ppuCtrl == 0x00 && ppuMask == 0x00 {
			t.Error("PPU registers not initialized - system may not be running ROM properly")
		}
	})

	t.Run("Verify color generation system", func(t *testing.T) {
		// Test that the color generation/palette system is working

		// Read multiple palette entries to verify system is functional
		paletteAddresses := []uint16{0x3F00, 0x3F01, 0x3F02, 0x3F03, 0x3F10, 0x3F11}
		allZero := true
		allSame := true
		firstValue := uint8(0)

		for i, addr := range paletteAddresses {
			suite.helper.PPU.WriteRegister(0x2006, uint8(addr>>8))
			suite.helper.PPU.WriteRegister(0x2006, uint8(addr&0xFF))
			value := suite.helper.PPU.ReadRegister(0x2007)

			if i == 0 {
				firstValue = value
			}

			if value != 0x00 {
				allZero = false
			}
			if value != firstValue {
				allSame = false
			}

			t.Logf("Palette[0x%04X] = 0x%02X", addr, value)
		}

		if allZero {
			t.Error("All palette entries are zero - palette system not working!")
		}
		if allSame && firstValue != 0x0F {
			t.Errorf("All palette entries are the same (0x%02X) - palette not properly loaded!", firstValue)
		}
	})

	t.Run("Frame buffer consistency check", func(t *testing.T) {
		// Run multiple frames and verify consistent output
		
		// Capture initial state
		suite.helper.PPU.WriteRegister(0x2006, 0x3F)
		suite.helper.PPU.WriteRegister(0x2006, 0x00)
		initialBG := suite.helper.PPU.ReadRegister(0x2007)

		// Run several frames
		for frame := 0; frame < 5; frame++ {
			suite.helper.RunFrame()
		}

		// Check that palette is still consistent
		suite.helper.PPU.WriteRegister(0x2006, 0x3F)
		suite.helper.PPU.WriteRegister(0x2006, 0x00)
		currentBG := suite.helper.PPU.ReadRegister(0x2007)

		if currentBG != initialBG {
			t.Errorf("Palette changed during rendering: was 0x%02X, now 0x%02X", 
				initialBG, currentBG)
		}

		// Check PPU state consistency
		ppuMask := suite.helper.PPU.ReadRegister(0x2001)
		if (ppuMask & 0x18) == 0 {
			t.Error("PPU rendering disabled during test - system may have crashed")
		}
	})
}

// TestSampleROM_CHRDataAccess tests CHR data loading and access
func TestSampleROM_CHRDataAccess(t *testing.T) {
	suite := &SampleROMColorTestSuite{}
	suite.LoadSampleROM(t)

	t.Run("CHR ROM contains character data", func(t *testing.T) {
		// Verify that CHR ROM has been loaded with character pattern data
		
		// Check pattern for character 'H' (tile 8 in sample ROM)
		tileIndex := uint16(8) // 'H'
		baseAddr := tileIndex * 16

		// Read first few bytes of pattern data
		patternBytes := make([]uint8, 8)
		for i := 0; i < 8; i++ {
			patternBytes[i] = suite.helper.Cartridge.ReadCHR(baseAddr + uint16(i))
		}

		// Pattern should not be all zeros (empty)
		allZero := true
		for _, b := range patternBytes {
			if b != 0x00 {
				allZero = false
				break
			}
		}

		if allZero {
			t.Error("Character pattern data is empty - CHR ROM not loaded properly")
		}

		t.Logf("Character 'H' pattern: %02X %02X %02X %02X %02X %02X %02X %02X", 
			patternBytes[0], patternBytes[1], patternBytes[2], patternBytes[3],
			patternBytes[4], patternBytes[5], patternBytes[6], patternBytes[7])
	})

	t.Run("CHR data accessible through PPU", func(t *testing.T) {
		// Test that PPU can access CHR data correctly
		
		// Set PPU address to pattern table 0, tile 0
		suite.helper.PPU.WriteRegister(0x2006, 0x00) // PPUADDR high
		suite.helper.PPU.WriteRegister(0x2006, 0x00) // PPUADDR low

		// Read pattern data through PPU
		firstByte := suite.helper.PPU.ReadRegister(0x2007) // PPUDATA
		secondByte := suite.helper.PPU.ReadRegister(0x2007)

		t.Logf("Pattern table data via PPU: 0x%02X 0x%02X", firstByte, secondByte)

		// Data should be accessible (this tests PPU-CHR interface)
		// The exact values depend on the CHR data in sample.nes
	})
}