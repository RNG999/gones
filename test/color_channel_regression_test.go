package test

import (
	"testing"
	"gones/internal/ppu"
	"gones/internal/memory"
)


// TestColorChannelSwappingRegression ensures red/blue channel swapping bugs don't return
func TestColorChannelSwappingRegression(t *testing.T) {
	ppu := ppu.New()
	
	// Test cases that specifically check for red/blue channel swapping
	tests := []struct {
		name        string
		colorIndex  uint8
		expectedRGB uint32
		redCheck    func(r, g, b uint8) bool
		blueCheck   func(r, g, b uint8) bool
		issue       string
	}{
		{
			"Pure Red Must Stay Red",
			0x16, 0xB40000,
			func(r, g, b uint8) bool { return r > 150 && g < 50 && b < 50 },
			func(r, g, b uint8) bool { return b < r/2 },
			"Red colors must not appear blue",
		},
		{
			"Pure Blue Must Stay Blue",
			0x02, 0x0000A8,
			func(r, g, b uint8) bool { return r < b/2 },
			func(r, g, b uint8) bool { return b > 150 && r < 50 && g < 50 },
			"Blue colors must not appear red",
		},
		{
			"Dark Red Detection",
			0x06, 0xA40000,
			func(r, g, b uint8) bool { return r > 100 && g < 30 && b < 30 },
			func(r, g, b uint8) bool { return b < 30 },
			"Dark red must remain predominantly red",
		},
		{
			"Purple/Magenta Verification",
			0x04, 0x8C0074,
			func(r, g, b uint8) bool { return r > 50 },
			func(r, g, b uint8) bool { return b > 50 },
			"Purple should have both red and blue components",
		},
		{
			"Mario Sky Blue",
			0x22, 0x5C94FC,
			func(r, g, b uint8) bool { return r < b },
			func(r, g, b uint8) bool { return b > r && b > g },
			"Sky blue must be predominantly blue",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rgb := ppu.NESColorToRGB(tt.colorIndex)
			
			// Extract RGB components
			r := uint8((rgb >> 16) & 0xFF)
			g := uint8((rgb >> 8) & 0xFF)
			b := uint8(rgb & 0xFF)
			
			// Check for exact match first
			if rgb != tt.expectedRGB {
				r2 := uint8((tt.expectedRGB >> 16) & 0xFF)
				g2 := uint8((tt.expectedRGB >> 8) & 0xFF)
				b2 := uint8(tt.expectedRGB & 0xFF)
				
				t.Errorf("Color index $%02X RGB mismatch: expected (%d,%d,%d) #%06X, got (%d,%d,%d) #%06X",
					tt.colorIndex, r2, g2, b2, tt.expectedRGB, r, g, b, rgb)
			}
			
			// Check red channel validation
			if !tt.redCheck(r, g, b) {
				t.Errorf("%s: Red channel validation failed for color $%02X. RGB: (%d,%d,%d). %s",
					tt.name, tt.colorIndex, r, g, b, tt.issue)
			}
			
			// Check blue channel validation
			if !tt.blueCheck(r, g, b) {
				t.Errorf("%s: Blue channel validation failed for color $%02X. RGB: (%d,%d,%d). %s",
					tt.name, tt.colorIndex, r, g, b, tt.issue)
			}
		})
	}
}

// TestRedBlueSwapDetection specifically looks for patterns that indicate channel swapping
func TestRedBlueSwapDetection(t *testing.T) {
	ppu := ppu.New()
	
	// Test pairs where red/blue swapping would be obvious
	swapTests := []struct {
		redIndex, blueIndex   uint8
		redName, blueName     string
		expectedRedRGB, expectedBlueRGB uint32
	}{
		{0x16, 0x02, "NES Red", "NES Blue", 0xB40000, 0x0000A8},
		{0x06, 0x12, "Dark Red", "Bright Blue", 0xA40000, 0x0000F0},
		{0x17, 0x01, "Orange Red", "Dark Blue", 0xE40058, 0x24188C},
	}

	for _, st := range swapTests {
		t.Run(st.redName+" vs "+st.blueName, func(t *testing.T) {
			// Get both colors
			redRGB := ppu.NESColorToRGB(st.redIndex)
			blueRGB := ppu.NESColorToRGB(st.blueIndex)
			
			// Extract components
			redR := uint8((redRGB >> 16) & 0xFF)
			redG := uint8((redRGB >> 8) & 0xFF)
			redB := uint8(redRGB & 0xFF)
			
			blueR := uint8((blueRGB >> 16) & 0xFF)
			blueG := uint8((blueRGB >> 8) & 0xFF)
			blueB := uint8(blueRGB & 0xFF)
			
			// Red color should have higher red component than blue component
			if redR <= redB {
				t.Errorf("RED/BLUE SWAP DETECTED: %s (index $%02X) has red=%d <= blue=%d. Full RGB: (%d,%d,%d)",
					st.redName, st.redIndex, redR, redB, redR, redG, redB)
			}
			
			// Blue color should have higher blue component than red component
			if blueB <= blueR {
				t.Errorf("RED/BLUE SWAP DETECTED: %s (index $%02X) has blue=%d <= red=%d. Full RGB: (%d,%d,%d)",
					st.blueName, st.blueIndex, blueB, blueR, blueR, blueG, blueB)
			}
			
			// Cross-validate: red color should have more red than the blue color has
			if redR <= blueR {
				t.Errorf("CROSS-COLOR SWAP DETECTED: %s has less red (%d) than %s (%d)",
					st.redName, redR, st.blueName, blueR)
			}
			
			// Cross-validate: blue color should have more blue than the red color has
			if blueB <= redB {
				t.Errorf("CROSS-COLOR SWAP DETECTED: %s has less blue (%d) than %s (%d)",
					st.blueName, blueB, st.redName, redB)
			}
		})
	}
}

// TestColorEmphasisChannelIntegrity ensures color emphasis doesn't cause channel swapping
func TestColorEmphasisChannelIntegrity(t *testing.T) {
	ppu := ppu.New()
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	ppu.SetMemory(ppuMem)

	// Test colors with different emphasis modes
	testColors := []struct {
		name       string
		colorIndex uint8
		dominantChannel string
	}{
		{"Mario Red", 0x16, "red"},
		{"Sky Blue", 0x22, "blue"},
		{"Pipe Green", 0x29, "green"},
		{"White", 0x30, "balanced"},
	}

	emphasisModes := []struct {
		name string
		mask uint8
		preservedChannel string
	}{
		{"No Emphasis", 0x00, "none"},
		{"Red Emphasis", 0x20, "red"},
		{"Green Emphasis", 0x40, "green"},
		{"Blue Emphasis", 0x80, "blue"},
		{"All Emphasis", 0xE0, "all"},
	}

	for _, color := range testColors {
		for _, emphasis := range emphasisModes {
			t.Run(color.name+" with "+emphasis.name, func(t *testing.T) {
				// Set emphasis mode
				ppu.WriteRegister(0x2001, emphasis.mask)
				
				// Get emphasized color
				rgb := ppu.NESColorToRGB(color.colorIndex)
				
				r := uint8((rgb >> 16) & 0xFF)
				g := uint8((rgb >> 8) & 0xFF)
				b := uint8(rgb & 0xFF)
				
				// Check that the dominant channel relationship is preserved
				switch color.dominantChannel {
				case "red":
					if r <= g || r <= b {
						t.Errorf("%s with %s: Red dominance lost. RGB: (%d,%d,%d)",
							color.name, emphasis.name, r, g, b)
					}
				case "blue":
					if b <= r || b <= g {
						t.Errorf("%s with %s: Blue dominance lost. RGB: (%d,%d,%d)",
							color.name, emphasis.name, r, g, b)
					}
				case "green":
					if g <= r || g <= b {
						t.Errorf("%s with %s: Green dominance lost. RGB: (%d,%d,%d)",
							color.name, emphasis.name, r, g, b)
					}
				}
				
				// Reset emphasis for next test
				ppu.WriteRegister(0x2001, 0x00)
			})
		}
	}
}

// TestExtremeCaseChannelSwapping tests edge cases where channel swapping might occur
func TestExtremeCaseChannelSwapping(t *testing.T) {
	ppu := ppu.New()
	
	// Test extreme cases that might trigger bugs
	extremeTests := []struct {
		name        string
		colorIndex  uint8
		expectedRGB uint32
		validation  func(r, g, b uint8) bool
		issue       string
	}{
		{
			"Black (No Channels)",
			0x0F, 0x000000,
			func(r, g, b uint8) bool { return r == 0 && g == 0 && b == 0 },
			"Black should have all channels at zero",
		},
		{
			"White (All Channels Max)",
			0x30, 0xFCFCFC,
			func(r, g, b uint8) bool { return r > 200 && g > 200 && b > 200 },
			"White should have all channels high",
		},
		{
			"Pure Red Channel",
			0x16, 0xB40000,
			func(r, g, b uint8) bool { return r > 100 && g == 0 && b == 0 },
			"Pure red should have only red channel active",
		},
		{
			"Pure Blue Channel",
			0x02, 0x0000A8,
			func(r, g, b uint8) bool { return r == 0 && g == 0 && b > 100 },
			"Pure blue should have only blue channel active",
		},
		{
			"Mid-Range Gray",
			0x00, 0x747474,
			func(r, g, b uint8) bool { 
				diff := int(r) - int(g)
				if diff < 0 { diff = -diff }
				diff2 := int(g) - int(b)
				if diff2 < 0 { diff2 = -diff2 }
				return diff < 10 && diff2 < 10 // Channels should be close
			},
			"Gray should have balanced RGB channels",
		},
	}

	for _, tt := range extremeTests {
		t.Run(tt.name, func(t *testing.T) {
			rgb := ppu.NESColorToRGB(tt.colorIndex)
			
			r := uint8((rgb >> 16) & 0xFF)
			g := uint8((rgb >> 8) & 0xFF)
			b := uint8(rgb & 0xFF)
			
			// Check exact RGB match
			if rgb != tt.expectedRGB {
				r2 := uint8((tt.expectedRGB >> 16) & 0xFF)
				g2 := uint8((tt.expectedRGB >> 8) & 0xFF)
				b2 := uint8(tt.expectedRGB & 0xFF)
				
				t.Errorf("%s: Expected RGB (%d,%d,%d) #%06X, got RGB (%d,%d,%d) #%06X",
					tt.name, r2, g2, b2, tt.expectedRGB, r, g, b, rgb)
			}
			
			// Check validation function
			if !tt.validation(r, g, b) {
				t.Errorf("%s: Validation failed. RGB: (%d,%d,%d). %s",
					tt.name, r, g, b, tt.issue)
			}
		})
	}
}

// TestChannelSwappingWithDifferentPPUStates tests color consistency across PPU states
func TestChannelSwappingWithDifferentPPUStates(t *testing.T) {
	testColor := uint8(0x16) // Mario red
	expectedRGB := uint32(0xB40000)

	// Test different PPU configurations
	testCases := []struct {
		name        string
		maskValue   uint8
		description string
	}{
		{"Rendering Disabled", 0x00, "No rendering enabled"},
		{"Background Only", 0x08, "Background rendering only"},
		{"Sprites Only", 0x10, "Sprite rendering only"},
		{"Full Rendering", 0x18, "Background and sprite rendering"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ppu := ppu.New()
			cart := &MockCartridge{}
			ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
			ppu.SetMemory(ppuMem)
			
			// Set up PPU state
			ppu.WriteRegister(0x2001, tc.maskValue)
			
			// Get color
			rgb := ppu.NESColorToRGB(testColor)
			
			// Color should be consistent regardless of PPU state
			if rgb != expectedRGB {
				r := uint8((rgb >> 16) & 0xFF)
				g := uint8((rgb >> 8) & 0xFF)
				b := uint8(rgb & 0xFF)
				
				r2 := uint8((expectedRGB >> 16) & 0xFF)
				g2 := uint8((expectedRGB >> 8) & 0xFF)
				b2 := uint8(expectedRGB & 0xFF)
				
				t.Errorf("%s: Color inconsistency. Expected RGB (%d,%d,%d) #%06X, got RGB (%d,%d,%d) #%06X",
					tc.description, r2, g2, b2, expectedRGB, r, g, b, rgb)
			}
			
			// Check red dominance is preserved
			r := uint8((rgb >> 16) & 0xFF)
			g := uint8((rgb >> 8) & 0xFF)
			b := uint8(rgb & 0xFF)
			
			if r <= b {
				t.Errorf("%s: Red/blue channel swap detected. Red=%d should be > Blue=%d. RGB=(%d,%d,%d)",
					tc.description, r, b, r, g, b)
			}
		})
	}
}

// TestChannelSwappingBugRegression tests specific scenarios known to cause channel swapping
func TestChannelSwappingBugRegression(t *testing.T) {
	ppu := ppu.New()
	
	// Known problematic scenarios that have caused channel swapping in the past
	regressionTests := []struct {
		name         string
		scenario     string
		colorIndex   uint8
		expectedRGB  uint32
		bugSymptom   string
	}{
		{
			"Red Screen Bug",
			"Mario's sky background appearing red instead of blue",
			0x22, 0x5C94FC,
			"Sky appears red due to red/blue channel swap",
		},
		{
			"Mario Color Inversion",
			"Mario's red hat appearing blue",
			0x16, 0xB40000,
			"Mario's red clothing appears blue",
		},
		{
			"Pipe Color Bug",
			"Green pipes appearing magenta",
			0x29, 0x00A800,
			"Green elements appear wrong due to channel issues",
		},
		{
			"Question Block Color",
			"Yellow question blocks appearing cyan",
			0x28, 0xF0BC3C,
			"Yellow elements have wrong color due to red/blue swap",
		},
	}

	for _, rt := range regressionTests {
		t.Run(rt.name, func(t *testing.T) {
			rgb := ppu.NESColorToRGB(rt.colorIndex)
			
			if rgb != rt.expectedRGB {
				r := uint8((rgb >> 16) & 0xFF)
				g := uint8((rgb >> 8) & 0xFF)
				b := uint8(rgb & 0xFF)
				
				r2 := uint8((rt.expectedRGB >> 16) & 0xFF)
				g2 := uint8((rt.expectedRGB >> 8) & 0xFF)
				b2 := uint8(rt.expectedRGB & 0xFF)
				
				t.Errorf("REGRESSION BUG DETECTED: %s\n"+
					"  Scenario: %s\n"+
					"  Color Index: $%02X\n"+
					"  Expected RGB: (%d,%d,%d) #%06X\n"+
					"  Actual RGB:   (%d,%d,%d) #%06X\n"+
					"  Bug Symptom: %s",
					rt.name, rt.scenario, rt.colorIndex,
					r2, g2, b2, rt.expectedRGB,
					r, g, b, rgb,
					rt.bugSymptom)
			}
		})
	}
}