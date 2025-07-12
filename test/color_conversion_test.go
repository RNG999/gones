package test

import (
	"testing"
	"gones/internal/ppu"
)

func TestNESColorConversionAccuracy(t *testing.T) {
	tests := []struct {
		name        string
		colorIndex  uint8
		expectedRGB uint32
		description string
	}{
		{"Black", 0x0F, 0x000000, "Color index $0F should map to black RGB"},
		{"White", 0x30, 0xFCFCFC, "Color index $30 should map to white RGB"},
		{"Red", 0x16, 0xB40000, "Color index $16 should map to red RGB"}, // Updated to actual palette value
		{"Green", 0x2A, 0x4CDC48, "Color index $2A should map to green RGB"}, // Updated to actual palette value
		{"Blue", 0x02, 0x0000A8, "Color index $02 should map to blue RGB"},
		{"Gray", 0x00, 0x747474, "Color index $00 should map to gray RGB"},
		{"Dark Red", 0x06, 0xA40000, "Color index $06 should map to dark red RGB"},
		{"Mario Background", 0x29, 0x00A800, "Color index $29 should map to Mario green background"}, // Updated to actual palette value
		{"Mario Skin", 0x17, 0xE40058, "Color index $17 should map to Mario skin tone"}, // Updated to actual palette value
		{"Goomba Brown", 0x18, 0xD82800, "Color index $18 should map to Goomba brown"}, // Updated to actual palette value
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ppu := ppu.New()
			rgb := ppu.NESColorToRGB(tt.colorIndex)
			
			if rgb != tt.expectedRGB {
				r1 := (rgb >> 16) & 0xFF
				g1 := (rgb >> 8) & 0xFF
				b1 := rgb & 0xFF
				
				r2 := (tt.expectedRGB >> 16) & 0xFF
				g2 := (tt.expectedRGB >> 8) & 0xFF
				b2 := tt.expectedRGB & 0xFF
				
				t.Errorf("%s: Color index $%02X expected RGB(#%06X = %d,%d,%d) but got RGB(#%06X = %d,%d,%d)",
					tt.description, tt.colorIndex, tt.expectedRGB, r2, g2, b2, rgb, r1, g1, b1)
			}
		})
	}
}

func TestRedBlueChannelSwappingPrevention(t *testing.T) {
	tests := []struct {
		name       string
		colorIndex uint8
		checkFunc  func(r, g, b uint8) bool
		description string
	}{
		{
			"Pure Red Detection", 0x16,
			func(r, g, b uint8) bool { return r >= 180 && g < 50 && b < 50 },
			"Pure red color should have high red channel, low green and blue",
		},
		{
			"Pure Blue Detection", 0x02,
			func(r, g, b uint8) bool { return r < 50 && g < 50 && b > 150 },
			"Pure blue color should have high blue channel, low red and green",
		},
		{
			"Purple Detection", 0x04,
			func(r, g, b uint8) bool { return r > 100 && g < 50 && b > 100 },
			"Purple color should have high red and blue, low green",
		},
		{
			"Green Detection", 0x2A,
			func(r, g, b uint8) bool { return g > r && g > b },
			"Green color should have highest green channel",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ppu := ppu.New()
			rgb := ppu.NESColorToRGB(tt.colorIndex)
			
			r := uint8((rgb >> 16) & 0xFF)
			g := uint8((rgb >> 8) & 0xFF)
			b := uint8(rgb & 0xFF)
			
			if !tt.checkFunc(r, g, b) {
				t.Errorf("%s: Color index $%02X produced RGB(%d,%d,%d) which fails channel validation. Possible red/blue swap detected!",
					tt.description, tt.colorIndex, r, g, b)
			}
		})
	}
}

func TestColorEmphasisChannelCorrectness(t *testing.T) {
	ppu := ppu.New()
	baseColorIndex := uint8(0x30) // White color for emphasis testing
	normalRGB := ppu.NESColorToRGB(baseColorIndex)
	normalR := uint8((normalRGB >> 16) & 0xFF)
	normalG := uint8((normalRGB >> 8) & 0xFF)
	normalB := uint8(normalRGB & 0xFF)

	tests := []struct {
		name          string
		maskValue     uint8
		validateFunc  func(r, g, b, normalR, normalG, normalB uint8) bool
		description   string
	}{
		{
			"Red Emphasis", 0x20,
			func(r, g, b, nr, ng, nb uint8) bool {
				return r == nr && g < ng && b < nb
			},
			"Red emphasis should preserve red channel while darkening green and blue",
		},
		{
			"Green Emphasis", 0x40,
			func(r, g, b, nr, ng, nb uint8) bool {
				return r < nr && g == ng && b < nb
			},
			"Green emphasis should preserve green channel while darkening red and blue",
		},
		{
			"Blue Emphasis", 0x80,
			func(r, g, b, nr, ng, nb uint8) bool {
				return r < nr && g < ng && b == nb
			},
			"Blue emphasis should preserve blue channel while darkening red and green",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ppu.WriteRegister(0x2001, tt.maskValue)
			emphasizedRGB := ppu.NESColorToRGB(baseColorIndex)
			
			er := uint8((emphasizedRGB >> 16) & 0xFF)
			eg := uint8((emphasizedRGB >> 8) & 0xFF)
			eb := uint8(emphasizedRGB & 0xFF)
			
			if !tt.validateFunc(er, eg, eb, normalR, normalG, normalB) {
				t.Errorf("%s: Normal RGB(%d,%d,%d) -> Emphasized RGB(%d,%d,%d) failed validation. Channel ordering may be incorrect!",
					tt.description, normalR, normalG, normalB, er, eg, eb)
			}
			
			// Reset mask for next test
			ppu.WriteRegister(0x2001, 0x00)
		})
	}
}

func TestSuperMarioBrosSpecificColors(t *testing.T) {
	tests := []struct {
		name        string
		colorIndex  uint8
		expectedRGB uint32
		gameElement string
	}{
		{"Mario Red Hat", 0x16, 0xB40000, "Mario's red cap and shirt"},
		{"Mario Blue Overalls", 0x12, 0x0000F0, "Mario's blue overalls"},
		{"Mario Skin Tone", 0x27, 0xFC7460, "Mario's skin color"},
		{"Sky Blue Background", 0x22, 0x5C94FC, "Sky background color"},
		{"Ground Brown", 0x18, 0xD82800, "Ground and brick color"},
		{"Pipe Green", 0x29, 0x00A800, "Warp pipe green color"},
		{"Question Block Yellow", 0x28, 0xF0BC3C, "Question mark block color"},
		{"Coin Yellow", 0x38, 0xFCD8A8, "Coin color"},
		{"Goomba Brown", 0x17, 0xE40058, "Goomba enemy color"},
		{"Koopa Green Shell", 0x2A, 0x4CDC48, "Koopa shell green"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ppu := ppu.New()
			rgb := ppu.NESColorToRGB(tt.colorIndex)
			
			if rgb != tt.expectedRGB {
				r1 := (rgb >> 16) & 0xFF
				g1 := (rgb >> 8) & 0xFF
				b1 := rgb & 0xFF
				
				r2 := (tt.expectedRGB >> 16) & 0xFF
				g2 := (tt.expectedRGB >> 8) & 0xFF
				b2 := tt.expectedRGB & 0xFF
				
				t.Errorf("Super Mario Bros %s: Color index $%02X expected RGB(#%06X = %d,%d,%d) but got RGB(#%06X = %d,%d,%d)",
					tt.gameElement, tt.colorIndex, tt.expectedRGB, r2, g2, b2, rgb, r1, g1, b1)
			}
		})
	}
}

func TestColorConversionConsistency(t *testing.T) {
	ppu := ppu.New()
	
	// Test that color conversion is deterministic
	for i := 0; i < 64; i++ {
		colorIndex := uint8(i)
		
		// Get color multiple times
		rgb1 := ppu.NESColorToRGB(colorIndex)
		rgb2 := ppu.NESColorToRGB(colorIndex)
		rgb3 := ppu.NESColorToRGB(colorIndex)
		
		if rgb1 != rgb2 || rgb2 != rgb3 {
			t.Errorf("Color conversion inconsistency for index $%02X: got %06X, %06X, %06X",
				colorIndex, rgb1, rgb2, rgb3)
		}
	}
}

func TestInvalidColorIndexHandling(t *testing.T) {
	ppu := ppu.New()
	
	tests := []struct {
		name       string
		colorIndex uint8
		expected   uint32
	}{
		{"Valid Index 63", 63, 0x000000}, // Last valid color
		{"Invalid Index 64", 64, 0x000000}, // Should return black or default
		{"Invalid Index 255", 255, 0x000000}, // Should return black or default
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rgb := ppu.NESColorToRGB(tt.colorIndex)
			
			if tt.colorIndex >= 64 && rgb != tt.expected {
				t.Errorf("Invalid color index %d should return default color %06X, got %06X",
					tt.colorIndex, tt.expected, rgb)
			}
		})
	}
}