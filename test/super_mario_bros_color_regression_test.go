package test

import (
	"testing"
	"gones/internal/ppu"
	"gones/internal/memory"
)

// TestSuperMarioBrosColorRegression ensures specific Super Mario Bros colors render correctly
// This test prevents regressions in color handling that would make the game look wrong
func TestSuperMarioBrosColorRegression(t *testing.T) {
	ppu := ppu.New()
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppu.SetMemory(ppuMem)

	// Super Mario Bros known color issues and their correct values
	tests := []struct {
		name           string
		paletteAddr    uint16
		colorIndex     uint8
		expectedRGB    uint32
		gameElement    string
		issueDesc      string
	}{
		{
			"Mario Red Hat",
			0x3F11,
			0x16,
			0xB40000,
			"Mario's red cap and shirt",
			"Red should not appear blue due to channel swapping",
		},
		{
			"Mario Blue Overalls",
			0x3F12,
			0x12,
			0x0000F0,
			"Mario's blue overalls",
			"Blue should not appear red due to channel swapping",
		},
		{
			"Sky Background",
			0x3F00,
			0x22,
			0x5C94FC,
			"Sky blue background",
			"Sky should be blue, not red or other color",
		},
		{
			"Ground Brown",
			0x3F01,
			0x17,
			0xE40058,
			"Ground and brick blocks",
			"Ground should be brownish, not completely different color",
		},
		{
			"Pipe Green",
			0x3F02,
			0x29,
			0x00A800,
			"Warp pipes",
			"Pipes should be green, not red or blue",
		},
		{
			"Question Block Yellow",
			0x3F05,
			0x28,
			0xF0BC3C,
			"Question mark blocks",
			"Question blocks should be yellow/golden",
		},
		{
			"Coin Gold",
			0x3F06,
			0x38,
			0xFCD8A8,
			"Coins",
			"Coins should be golden yellow",
		},
		{
			"Goomba Brown",
			0x3F15,
			0x18,
			0xD82800,
			"Goomba enemies",
			"Goombas should be brown/orange",
		},
		{
			"Koopa Shell Green",
			0x3F16,
			0x2A,
			0x4CDC48,
			"Koopa Troopa shells",
			"Koopa shells should be green",
		},
		{
			"Mario Skin Tone",
			0x3F13,
			0x27,
			0xFC7460,
			"Mario's skin color",
			"Mario's skin should be peachy/orange, not other colors",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up the palette
			ppuMem.Write(tt.paletteAddr, tt.colorIndex)
			
			// Get the RGB color
			rgb := ppu.NESColorToRGB(tt.colorIndex)
			
			if rgb != tt.expectedRGB {
				r1 := (rgb >> 16) & 0xFF
				g1 := (rgb >> 8) & 0xFF
				b1 := rgb & 0xFF
				
				r2 := (tt.expectedRGB >> 16) & 0xFF
				g2 := (tt.expectedRGB >> 8) & 0xFF
				b2 := tt.expectedRGB & 0xFF
				
				t.Errorf("Super Mario Bros %s REGRESSION: %s\n"+
					"  Game Element: %s\n"+
					"  Color Index: $%02X at palette address $%04X\n"+
					"  Expected RGB: #%06X (%d,%d,%d)\n"+
					"  Actual RGB:   #%06X (%d,%d,%d)\n"+
					"  Issue: %s",
					tt.name, tt.issueDesc, tt.gameElement, 
					tt.colorIndex, tt.paletteAddr,
					tt.expectedRGB, r2, g2, b2,
					rgb, r1, g1, b1,
					tt.issueDesc)
			}
		})
	}
}

func TestSuperMarioBrosRedBackgroundBugPrevention(t *testing.T) {
	ppu := ppu.New()
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppu.SetMemory(ppuMem)

	// Test the specific red background bug scenario
	t.Run("Prevent red background bug", func(t *testing.T) {
		// Set up typical Super Mario Bros sky background
		ppuMem.Write(0x3F00, 0x22) // Sky blue
		
		// Enable normal rendering (no emphasis)
		ppu.WriteRegister(0x2001, 0x18) // Background and sprite rendering on
		
		// Get the background color
		skyRGB := ppu.NESColorToRGB(0x22)
		
		// Extract RGB components
		r := (skyRGB >> 16) & 0xFF
		g := (skyRGB >> 8) & 0xFF
		b := skyRGB & 0xFF
		
		// Sky should be predominantly blue, not red
		if r > b {
			t.Errorf("RED BACKGROUND BUG DETECTED: Sky color $22 has more red (%d) than blue (%d). RGB: (%d,%d,%d)",
				r, b, r, g, b)
		}
		
		// Blue component should be the highest for sky color
		if b < g || b < r {
			t.Errorf("Sky color $22 should be predominantly blue. RGB: (%d,%d,%d)", r, g, b)
		}
		
		// Verify the expected sky blue color
		expectedSkyRGB := uint32(0x5C94FC)
		if skyRGB != expectedSkyRGB {
			t.Errorf("Sky color mismatch: expected #%06X, got #%06X", expectedSkyRGB, skyRGB)
		}
	})

	t.Run("Test all backgrounds are not erroneously red", func(t *testing.T) {
		// Common Super Mario Bros background colors that should NOT be red
		nonRedBackgrounds := []struct {
			colorIndex uint8
			name       string
			maxRedRatio float64 // Maximum allowed red component ratio
		}{
			{0x22, "Sky Blue", 0.4},      // Sky should be mostly blue
			{0x0F, "Black", 0.1},          // Black should have minimal color
			{0x30, "White", 0.35},         // White should be balanced
			{0x29, "Green", 0.3},          // Green should be mostly green
			{0x2A, "Bright Green", 0.3},   // Bright green should be mostly green
		}
		
		for _, bg := range nonRedBackgrounds {
			ppuMem.Write(0x3F00, bg.colorIndex)
			rgb := ppu.NESColorToRGB(bg.colorIndex)
			
			r := float64((rgb >> 16) & 0xFF)
			g := float64((rgb >> 8) & 0xFF)
			b := float64(rgb & 0xFF)
			total := r + g + b
			
			if total > 0 {
				redRatio := r / total
				if redRatio > bg.maxRedRatio {
					t.Errorf("%s background (color $%02X) is too red: %.2f%% red (max allowed: %.1f%%). RGB: (%.0f,%.0f,%.0f)",
						bg.name, bg.colorIndex, redRatio*100, bg.maxRedRatio*100, r, g, b)
				}
			}
		}
	})
}

func TestSuperMarioBrosColorEmphasisRegression(t *testing.T) {
	ppu := ppu.New()
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppu.SetMemory(ppuMem)

	// Set up Mario's red color
	ppuMem.Write(0x3F11, 0x16) // Mario red

	t.Run("Red emphasis should not cause red screen bug", func(t *testing.T) {
		// Test normal color first
		ppu.WriteRegister(0x2001, 0x00) // No emphasis
		normalRGB := ppu.NESColorToRGB(0x16)
		
		// Test red emphasis
		ppu.WriteRegister(0x2001, 0x20) // Red emphasis
		redEmphasisRGB := ppu.NESColorToRGB(0x16)
		
		// Red emphasis should modify the color but not create completely wrong colors
		normalR := uint8((normalRGB >> 16) & 0xFF)
		normalG := uint8((normalRGB >> 8) & 0xFF)
		normalB := uint8(normalRGB & 0xFF)
		
		emphasizedR := uint8((redEmphasisRGB >> 16) & 0xFF)
		emphasizedG := uint8((redEmphasisRGB >> 8) & 0xFF)
		emphasizedB := uint8(redEmphasisRGB & 0xFF)
		
		// Red channel should be preserved or enhanced
		if emphasizedR < uint8(float64(normalR)*0.8) {
			t.Errorf("Red emphasis reduced red channel too much: %d -> %d", normalR, emphasizedR)
		}
		
		// Green and blue should be reduced
		if emphasizedG > normalG {
			t.Errorf("Red emphasis should reduce green channel: %d -> %d", normalG, emphasizedG)
		}
		if emphasizedB > normalB {
			t.Errorf("Red emphasis should reduce blue channel: %d -> %d", normalB, emphasizedB)
		}
		
		// Reset for next test
		ppu.WriteRegister(0x2001, 0x00)
	})

	t.Run("Green emphasis on pipes", func(t *testing.T) {
		// Test green emphasis on pipe color
		pipeColor := uint8(0x29) // Green
		ppuMem.Write(0x3F02, pipeColor)
		
		ppu.WriteRegister(0x2001, 0x00) // No emphasis
		normalRGB := ppu.NESColorToRGB(pipeColor)
		
		ppu.WriteRegister(0x2001, 0x40) // Green emphasis
		greenEmphasisRGB := ppu.NESColorToRGB(pipeColor)
		
		normalG := uint8((normalRGB >> 8) & 0xFF)
		emphasizedG := uint8((greenEmphasisRGB >> 8) & 0xFF)
		
		// Green channel should be preserved
		if emphasizedG < uint8(float64(normalG)*0.8) {
			t.Errorf("Green emphasis reduced green channel too much: %d -> %d", normalG, emphasizedG)
		}
		
		// Reset for next test
		ppu.WriteRegister(0x2001, 0x00)
	})

	t.Run("Blue emphasis on sky", func(t *testing.T) {
		// Test blue emphasis on sky color
		skyColor := uint8(0x22) // Sky blue
		ppuMem.Write(0x3F00, skyColor)
		
		ppu.WriteRegister(0x2001, 0x00) // No emphasis
		normalRGB := ppu.NESColorToRGB(skyColor)
		
		ppu.WriteRegister(0x2001, 0x80) // Blue emphasis
		blueEmphasisRGB := ppu.NESColorToRGB(skyColor)
		
		normalB := uint8(normalRGB & 0xFF)
		emphasizedB := uint8(blueEmphasisRGB & 0xFF)
		
		// Blue channel should be preserved
		if emphasizedB < uint8(float64(normalB)*0.8) {
			t.Errorf("Blue emphasis reduced blue channel too much: %d -> %d", normalB, emphasizedB)
		}
		
		// Reset for next test
		ppu.WriteRegister(0x2001, 0x00)
	})
}

func TestSuperMarioBrosVisualConsistency(t *testing.T) {
	ppu := ppu.New()
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppu.SetMemory(ppuMem)

	// Set up complete Super Mario Bros World 1-1 palette
	marioWorldPalette := map[uint16]uint8{
		// Background palettes
		0x3F00: 0x22, // Sky blue (universal background)
		0x3F01: 0x17, // Brown (ground/bricks)
		0x3F02: 0x29, // Green (pipes)
		0x3F03: 0x0F, // Black
		0x3F04: 0x22, // Sky blue (palette 1)
		0x3F05: 0x28, // Yellow (question blocks)
		0x3F06: 0x38, // Light yellow (coins)
		0x3F07: 0x0F, // Black
		0x3F08: 0x22, // Sky blue (palette 2)
		0x3F09: 0x1A, // Light green
		0x3F0A: 0x29, // Green
		0x3F0B: 0x0F, // Black
		0x3F0C: 0x22, // Sky blue (palette 3)
		0x3F0D: 0x30, // White (clouds)
		0x3F0E: 0x27, // Orange
		0x3F0F: 0x0F, // Black
		
		// Sprite palettes
		0x3F11: 0x16, // Mario red (hat/shirt)
		0x3F12: 0x27, // Mario skin
		0x3F13: 0x18, // Mario brown (shoes/hair)
		0x3F15: 0x2A, // Sprite green
		0x3F16: 0x16, // Sprite red
		0x3F17: 0x0F, // Black
		0x3F19: 0x29, // Koopa green
		0x3F1A: 0x38, // Light yellow
		0x3F1B: 0x0F, // Black
		0x3F1D: 0x1A, // Goomba brown
		0x3F1E: 0x17, // Dark brown
		0x3F1F: 0x0F, // Black
	}

	// Set up the palette
	for addr, colorIndex := range marioWorldPalette {
		ppuMem.Write(addr, colorIndex)
	}

	t.Run("Visual consistency check", func(t *testing.T) {
		// Check that key visual elements have correct color relationships
		
		// Sky should be blue
		skyRGB := ppu.NESColorToRGB(0x22)
		skyB := skyRGB & 0xFF
		skyG := (skyRGB >> 8) & 0xFF
		skyR := (skyRGB >> 16) & 0xFF
		if skyB <= skyR {
			t.Errorf("Sky should be blue-dominant: RGB(%d,%d,%d)", skyR, skyG, skyB)
		}
		
		// Mario's hat should be red
		hatRGB := ppu.NESColorToRGB(0x16)
		hatR := (hatRGB >> 16) & 0xFF
		hatG := (hatRGB >> 8) & 0xFF
		hatB := hatRGB & 0xFF
		if hatR <= hatG || hatR <= hatB {
			t.Errorf("Mario's hat should be red-dominant: RGB(%d,%d,%d)", hatR, hatG, hatB)
		}
		
		// Pipes should be green
		pipeRGB := ppu.NESColorToRGB(0x29)
		pipeR := (pipeRGB >> 16) & 0xFF
		pipeG := (pipeRGB >> 8) & 0xFF
		pipeB := pipeRGB & 0xFF
		if pipeG <= pipeR || pipeG <= pipeB {
			t.Errorf("Pipes should be green-dominant: RGB(%d,%d,%d)", pipeR, pipeG, pipeB)
		}
		
		// Question blocks should be yellow (high red and green, low blue)
		blockRGB := ppu.NESColorToRGB(0x28)
		blockR := (blockRGB >> 16) & 0xFF
		blockG := (blockRGB >> 8) & 0xFF
		blockB := blockRGB & 0xFF
		if blockB > blockR || blockB > blockG {
			t.Errorf("Question blocks should be yellow (low blue): RGB(%d,%d,%d)", blockR, blockG, blockB)
		}
	})

	t.Run("Color distinction check", func(t *testing.T) {
		// Ensure key colors are visually distinct
		colorTests := []struct {
			color1, color2 uint8
			name1, name2   string
		}{
			{0x22, 0x16, "Sky Blue", "Mario Red"},
			{0x29, 0x16, "Pipe Green", "Mario Red"},
			{0x28, 0x0F, "Question Block Yellow", "Black"},
			{0x27, 0x16, "Mario Skin", "Mario Red"},
			{0x30, 0x0F, "White", "Black"},
		}
		
		for _, ct := range colorTests {
			rgb1 := ppu.NESColorToRGB(ct.color1)
			rgb2 := ppu.NESColorToRGB(ct.color2)
			
			if rgb1 == rgb2 {
				t.Errorf("%s (color $%02X) and %s (color $%02X) should be visually distinct but both map to #%06X",
					ct.name1, ct.color1, ct.name2, ct.color2, rgb1)
			}
			
			// Calculate color distance for additional validation
			r1, g1, b1 := (rgb1>>16)&0xFF, (rgb1>>8)&0xFF, rgb1&0xFF
			r2, g2, b2 := (rgb2>>16)&0xFF, (rgb2>>8)&0xFF, rgb2&0xFF
			
			// Simple color distance calculation
			dist := int((r1-r2)*(r1-r2) + (g1-g2)*(g1-g2) + (b1-b2)*(b1-b2))
			minDist := 1000 // Minimum visual distinction threshold
			
			if dist < minDist {
				t.Errorf("%s and %s are too similar (distance: %d, minimum: %d). RGB1: (%d,%d,%d), RGB2: (%d,%d,%d)",
					ct.name1, ct.name2, dist, minDist, r1, g1, b1, r2, g2, b2)
			}
		}
	})
}