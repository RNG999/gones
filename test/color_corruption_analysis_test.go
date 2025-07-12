package test

import (
	"gones/internal/debug"
	"testing"
)

// Color Corruption Analysis Test
// This test analyzes the actual NES color palette to understand
// the sky color corruption issue

func TestNESPaletteAnalysis(t *testing.T) {
	// Firebrandx NES palette (accurate representation)
	palette := []uint32{
		0x666666, 0x002A88, 0x1412A7, 0x3B00A4, 0x5C007E, 0x6E0040, 0x6C0700, 0x561D00,
		0x333500, 0x0B4800, 0x005200, 0x004C18, 0x003E5B, 0x000000, 0x000000, 0x000000,
		0xADADAD, 0x155FD9, 0x4240FF, 0x7527FE, 0xA01ACC, 0xB71E7B, 0xB53120, 0x994E00,
		0x6B6D00, 0x388700, 0x0D9300, 0x008C47, 0x007AB8, 0x000000, 0x000000, 0x000000,
		0xFFFEFF, 0x64B0FF, 0x9290FF, 0xC676FF, 0xF36AFF, 0xFF6ECC, 0xFF8170, 0xFF9C12,
		0xDAB700, 0x88D300, 0x5AC554, 0x3CC98C, 0x3EC7F4, 0x474747, 0x000000, 0x000000,
		0xFFFEFF, 0xC0DFFF, 0xD3D2FF, 0xE8C8FF, 0xFAC2FF, 0xFFC4EA, 0xFFCCC5, 0xFFD7AA,
		0xE4E594, 0xCFEF96, 0xBDF4AB, 0xB3F3CC, 0xB5EBF2, 0xB8B8B8, 0x000000, 0x000000,
	}
	
	// Analyze blue colors in the palette
	t.Logf("Blue colors in NES palette:")
	for i, color := range palette {
		r, g, b := extractRGB(color)
		
		// Look for colors that are predominantly blue or could be sky colors
		if b > r && b > g && b > 150 {
			t.Logf("  Index 0x%02X (%2d): RGB(%3d,%3d,%3d) = #%06X", 
				i, i, r, g, b, color)
		}
	}
	
	// Check the specific colors mentioned in Super Mario Bros
	t.Logf("\nSuper Mario Bros relevant colors:")
	relevantColors := []int{0x21, 0x22, 0x31, 0x32}
	
	for _, index := range relevantColors {
		if index < len(palette) {
			color := palette[index]
			r, g, b := extractRGB(color)
			t.Logf("  Index 0x%02X (%2d): RGB(%3d,%3d,%3d) = #%06X", 
				index, index, r, g, b, color)
		}
	}
	
	// Key finding analysis
	color0x21 := palette[0x21] // 33 decimal - this is the bright blue
	color0x22 := palette[0x22] // 34 decimal - this is the purple-blue
	
	r21, g21, b21 := extractRGB(color0x21)
	r22, g22, b22 := extractRGB(color0x22)
	
	t.Logf("\nKey Analysis:")
	t.Logf("  Color 0x21: RGB(%d,%d,%d) = #%06X (bright blue - likely sky)", r21, g21, b21, color0x21)
	t.Logf("  Color 0x22: RGB(%d,%d,%d) = #%06X (purple-blue)", r22, g22, b22, color0x22)
	
	// Check if Super Mario Bros actually uses 0x21 instead of 0x22 for sky
	if color0x21 == 0x64B0FF {
		t.Logf("  FINDING: 0x21 is the expected sky blue (#64B0FF)")
	} else if color0x22 == 0x64B0FF {
		t.Logf("  FINDING: 0x22 is the expected sky blue (#64B0FF)")
	} else {
		t.Logf("  FINDING: Neither 0x21 nor 0x22 matches expected sky blue #64B0FF")
		t.Logf("           Closest is 0x21 = #%06X", color0x21)
	}
}

func TestSuperMarioBrosActualColorUsage(t *testing.T) {
	// Test what colors Super Mario Bros actually uses for sky
	
	debug.InitializeColorDebugging("debug_output")
	debug.EnableColorDebugging()
	defer debug.DisableColorDebugging()
	
	// Typical Super Mario Bros palette setup
	// According to various sources, SMB uses color 0x22 for the universal background
	// But let's verify what that actually looks like
	
	palette := []uint32{
		0x666666, 0x002A88, 0x1412A7, 0x3B00A4, 0x5C007E, 0x6E0040, 0x6C0700, 0x561D00,
		0x333500, 0x0B4800, 0x005200, 0x004C18, 0x003E5B, 0x000000, 0x000000, 0x000000,
		0xADADAD, 0x155FD9, 0x4240FF, 0x7527FE, 0xA01ACC, 0xB71E7B, 0xB53120, 0x994E00,
		0x6B6D00, 0x388700, 0x0D9300, 0x008C47, 0x007AB8, 0x000000, 0x000000, 0x000000,
		0xFFFEFF, 0x64B0FF, 0x9290FF, 0xC676FF, 0xF36AFF, 0xFF6ECC, 0xFF8170, 0xFF9C12,
		0xDAB700, 0x88D300, 0x5AC554, 0x3CC98C, 0x3EC7F4, 0x474747, 0x000000, 0x000000,
		0xFFFEFF, 0xC0DFFF, 0xD3D2FF, 0xE8C8FF, 0xFAC2FF, 0xFFC4EA, 0xFFCCC5, 0xFFD7AA,
		0xE4E594, 0xCFEF96, 0xBDF4AB, 0xB3F3CC, 0xB5EBF2, 0xB8B8B8, 0x000000, 0x000000,
	}
	
	// Test both potential sky colors
	testColors := []struct {
		index uint8
		name  string
	}{
		{0x21, "Bright Blue (potential correct sky)"},
		{0x22, "Purple-Blue (what we tested)"},
	}
	
	for _, test := range testColors {
		rgb := palette[test.index]
		r, g, b := extractRGB(rgb)
		
		t.Logf("Testing %s - Index 0x%02X:", test.name, test.index)
		t.Logf("  RGB(%d,%d,%d) = #%06X", r, g, b, rgb)
		
		// Test emphasis effects on this color
		testEmphasisEffects(t, test.index, rgb)
	}
	
	// Key insight: if SMB uses 0x22 and it's purple-blue, then the "brown corruption"
	// might actually be correct emphasis behavior, not corruption!
}

func TestColorEmphasisOnActualSkyColor(t *testing.T) {
	// Test emphasis effects on the actual sky color used by Super Mario Bros
	
	debug.InitializeColorDebugging("debug_output")
	debug.EnableColorDebugging()
	defer debug.DisableColorDebugging()
	
	// Color 0x22 (what SMB actually uses for sky) = RGB(146,144,255) = #9290FF
	actualSkyRGB := uint32(0x9290FF)
	r, g, b := extractRGB(actualSkyRGB)
	
	t.Logf("Testing emphasis on actual SMB sky color:")
	t.Logf("  Base color: RGB(%d,%d,%d) = #%06X", r, g, b, actualSkyRGB)
	
	// Test various emphasis modes
	emphasisTests := []struct {
		mask uint8
		name string
	}{
		{0x00, "No Emphasis"},
		{0x20, "Red Emphasis"},
		{0x40, "Green Emphasis"},
		{0x80, "Blue Emphasis"},
		{0x60, "Red+Green (Yellow) Emphasis"},
		{0xE0, "All Emphasis"},
	}
	
	for _, test := range emphasisTests {
		emphasizedRGB := applyEmphasis(actualSkyRGB, test.mask)
		er, eg, eb := extractRGB(emphasizedRGB)
		
		t.Logf("  %s: RGB(%d,%d,%d) = #%06X", test.name, er, eg, eb, emphasizedRGB)
		
		// Check if this creates a brownish color
		if isBrownish(emphasizedRGB) {
			t.Logf("    -> BROWNISH COLOR DETECTED!")
		}
	}
}

func TestSkyColorCorruptionHypothesis(t *testing.T) {
	// Test the hypothesis: the "corruption" might be normal emphasis behavior
	// on the actual sky color used by Super Mario Bros
	
	// Hypothesis: SMB uses color 0x22 (purple-blue), and when red emphasis
	// is applied, it creates a brownish tint that looks like corruption
	
	actualSkyColor := uint32(0x9290FF) // Color 0x22 in NES palette
	
	// Apply red emphasis (common in some games/situations)
	redEmphasisResult := applyEmphasis(actualSkyColor, 0x20)
	
	r, g, b := extractRGB(redEmphasisResult)
	t.Logf("Sky color 0x22 with red emphasis: RGB(%d,%d,%d) = #%06X", r, g, b, redEmphasisResult)
	
	// Apply red+green emphasis (creates yellow/brown tint)
	yellowEmphasisResult := applyEmphasis(actualSkyColor, 0x60)
	yr, yg, yb := extractRGB(yellowEmphasisResult)
	t.Logf("Sky color 0x22 with red+green emphasis: RGB(%d,%d,%d) = #%06X", yr, yg, yb, yellowEmphasisResult)
	
	// Check if either of these creates the "brown corruption" appearance
	if isBrownish(redEmphasisResult) {
		t.Logf("HYPOTHESIS CONFIRMED: Red emphasis on actual sky color creates brownish appearance")
		t.Logf("This may explain the 'corruption' reports - it could be normal emphasis behavior")
	}
	
	if isBrownish(yellowEmphasisResult) {
		t.Logf("HYPOTHESIS CONFIRMED: Red+Green emphasis on actual sky color creates brownish appearance")
		t.Logf("This may explain the 'corruption' reports - it could be normal emphasis behavior")
	}
}

func TestCorrectSkyColorIdentification(t *testing.T) {
	// Identify which color index produces the expected bright blue sky
	
	palette := []uint32{
		0x666666, 0x002A88, 0x1412A7, 0x3B00A4, 0x5C007E, 0x6E0040, 0x6C0700, 0x561D00,
		0x333500, 0x0B4800, 0x005200, 0x004C18, 0x003E5B, 0x000000, 0x000000, 0x000000,
		0xADADAD, 0x155FD9, 0x4240FF, 0x7527FE, 0xA01ACC, 0xB71E7B, 0xB53120, 0x994E00,
		0x6B6D00, 0x388700, 0x0D9300, 0x008C47, 0x007AB8, 0x000000, 0x000000, 0x000000,
		0xFFFEFF, 0x64B0FF, 0x9290FF, 0xC676FF, 0xF36AFF, 0xFF6ECC, 0xFF8170, 0xFF9C12,
		0xDAB700, 0x88D300, 0x5AC554, 0x3CC98C, 0x3EC7F4, 0x474747, 0x000000, 0x000000,
		0xFFFEFF, 0xC0DFFF, 0xD3D2FF, 0xE8C8FF, 0xFAC2FF, 0xFFC4EA, 0xFFCCC5, 0xFFD7AA,
		0xE4E594, 0xCFEF96, 0xBDF4AB, 0xB3F3CC, 0xB5EBF2, 0xB8B8B8, 0x000000, 0x000000,
	}
	
	expectedSkyBlue := uint32(0x64B0FF)
	
	t.Logf("Searching for expected sky blue #64B0FF in NES palette:")
	
	for i, color := range palette {
		if color == expectedSkyBlue {
			t.Logf("FOUND: Index 0x%02X (%d) = #%06X matches expected sky blue", i, i, color)
			return
		}
	}
	
	// If exact match not found, find closest
	t.Logf("Exact match not found. Finding closest colors:")
	
	bestMatch := 0
	bestDistance := uint32(0xFFFFFF)
	
	for i, color := range palette {
		distance := colorDistance(color, expectedSkyBlue)
		if distance < bestDistance {
			bestDistance = distance
			bestMatch = i
		}
	}
	
	bestColor := palette[bestMatch]
	r, g, b := extractRGB(bestColor)
	t.Logf("Closest match: Index 0x%02X (%d) = RGB(%d,%d,%d) = #%06X", 
		bestMatch, bestMatch, r, g, b, bestColor)
	t.Logf("Distance from expected: %d", bestDistance)
}

// Helper functions

func extractRGB(rgb uint32) (uint8, uint8, uint8) {
	r := uint8((rgb >> 16) & 0xFF)
	g := uint8((rgb >> 8) & 0xFF)
	b := uint8(rgb & 0xFF)
	return r, g, b
}

func testEmphasisEffects(t *testing.T, colorIndex uint8, baseRGB uint32) {
	emphasisModes := []struct {
		mask uint8
		name string
	}{
		{0x20, "Red"},
		{0x40, "Green"},
		{0x80, "Blue"},
		{0x60, "Red+Green"},
	}
	
	for _, mode := range emphasisModes {
		result := applyEmphasis(baseRGB, mode.mask)
		r, g, b := extractRGB(result)
		
		if isBrownish(result) {
			t.Logf("    %s emphasis -> RGB(%d,%d,%d) = #%06X (BROWNISH!)", 
				mode.name, r, g, b, result)
		}
	}
}

func applyEmphasis(rgb uint32, emphasisMask uint8) uint32 {
	emphasisBits := (emphasisMask & 0xE0) >> 5
	
	if emphasisBits == 0 {
		return rgb
	}
	
	redEmphasis := (emphasisBits & 0x01) != 0
	greenEmphasis := (emphasisBits & 0x02) != 0
	blueEmphasis := (emphasisBits & 0x04) != 0
	
	r := float64((rgb >> 16) & 0xFF)
	g := float64((rgb >> 8) & 0xFF)
	b := float64(rgb & 0xFF)
	
	emphasisFactor := 0.75
	
	if !redEmphasis {
		r *= emphasisFactor
	}
	if !greenEmphasis {
		g *= emphasisFactor
	}
	if !blueEmphasis {
		b *= emphasisFactor
	}
	
	if r > 255 { r = 255 }
	if g > 255 { g = 255 }
	if b > 255 { b = 255 }
	
	return (uint32(r) << 16) | (uint32(g) << 8) | uint32(b)
}

func isBrownish(rgb uint32) bool {
	r, g, b := extractRGB(rgb)
	
	// Define brownish as: red > 100, green 80-180, blue < 120
	return r > 100 && g > 80 && g < 180 && b < 120
}

func colorDistance(color1, color2 uint32) uint32 {
	r1, g1, b1 := extractRGB(color1)
	r2, g2, b2 := extractRGB(color2)
	
	dr := int32(r1) - int32(r2)
	dg := int32(g1) - int32(g2)
	db := int32(b1) - int32(b2)
	
	return uint32(dr*dr + dg*dg + db*db)
}