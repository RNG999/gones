package integration

import (
	"testing"
	"gones/internal/ppu"
	"gones/internal/memory"
)

// TestVisualColorValidation validates that the color fixes produce the expected visual output
// This test ensures we've fixed the red/magenta background bug from the original screenshot
func TestVisualColorValidation(t *testing.T) {
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	testPPU := ppu.New()
	testPPU.SetMemory(ppuMem)

	t.Run("Validate Super Mario Bros color scenario from original bug report", func(t *testing.T) {
		// The original problematic screenshot showed a magenta/pink background instead of blue
		// This test validates that color index 0x22 (sky blue) produces correct blue RGB values
		
		// Set up the exact scenario from Super Mario Bros that was problematic
		ppuMem.Write(0x3F00, 0x22) // Universal background - Sky blue
		
		// Get the actual RGB color that would be rendered
		actualRGB := testPPU.NESColorToRGB(0x22)
		
		// Expected blue sky color (correct)
		expectedBlueRGB := uint32(0x5C94FC)
		
		// Colors that would indicate the bug is present (magenta/red tints)
		problematicMagentaRGB := uint32(0xFC5C94) // RGB channels swapped
		problematicRedRGB := uint32(0xFF0000)     // Pure red
		
		// Verify we get the correct blue color
		if actualRGB != expectedBlueRGB {
			t.Errorf("Sky blue color incorrect: expected #%06X, got #%06X", expectedBlueRGB, actualRGB)
		}
		
		// Verify we don't get problematic colors
		if actualRGB == problematicMagentaRGB {
			t.Errorf("CRITICAL BUG: Got magenta color #%06X instead of blue #%06X - RGB channel swapping detected!",
				actualRGB, expectedBlueRGB)
		}
		
		if actualRGB == problematicRedRGB {
			t.Errorf("CRITICAL BUG: Got red color #%06X instead of blue #%06X - color channel error detected!",
				actualRGB, expectedBlueRGB)
		}
		
		// Extract and validate RGB components
		r := (actualRGB >> 16) & 0xFF
		g := (actualRGB >> 8) & 0xFF
		b := actualRGB & 0xFF
		
		// For sky blue, blue component should be highest
		if b <= r {
			t.Errorf("Sky blue should have blue > red, but got RGB(%d,%d,%d)", r, g, b)
		}
		
		if b <= g {
			t.Errorf("Sky blue should have blue > green, but got RGB(%d,%d,%d)", r, g, b)
		}
		
		// Blue should be significantly higher than red (to avoid magenta tint)
		if float64(r)/float64(b) > 0.5 {
			t.Errorf("Sky blue has too much red component (ratio %.2f) - may appear magenta", float64(r)/float64(b))
		}
		
		t.Logf("SUCCESS: Sky blue correctly rendered as RGB(%d,%d,%d) = #%06X", r, g, b, actualRGB)
	})
	
	t.Run("Validate no color channel swapping across palette", func(t *testing.T) {
		// Test that fundamental colors maintain their channel identity
		colorTests := []struct {
			colorIndex    uint8
			name          string
			expectHighest string // Which RGB channel should be highest
		}{
			{0x06, "NES Red", "red"},
			{0x0C, "NES Blue", "blue"},
			{0x1A, "NES Green", "green"},
			{0x16, "Mario Red", "red"},
			{0x22, "Sky Blue", "blue"},
			{0x29, "Pipe Green", "green"},
		}
		
		for _, test := range colorTests {
			rgb := testPPU.NESColorToRGB(test.colorIndex)
			r := (rgb >> 16) & 0xFF
			g := (rgb >> 8) & 0xFF
			b := rgb & 0xFF
			
			switch test.expectHighest {
			case "red":
				if r < g || r < b {
					t.Errorf("Color %s (index $%02X) should be red-dominant but got RGB(%d,%d,%d)",
						test.name, test.colorIndex, r, g, b)
				}
			case "green":
				if g < r || g < b {
					t.Errorf("Color %s (index $%02X) should be green-dominant but got RGB(%d,%d,%d)",
						test.name, test.colorIndex, r, g, b)
				}
			case "blue":
				if b < r || b < g {
					t.Errorf("Color %s (index $%02X) should be blue-dominant but got RGB(%d,%d,%d)",
						test.name, test.colorIndex, r, g, b)
				}
			}
		}
		
		t.Logf("SUCCESS: All color channels maintain correct dominance")
	})
	
	t.Run("Validate specific problematic color indices from bug report", func(t *testing.T) {
		// Test specific colors that were reported as problematic in the original issue
		problematicColors := []struct {
			colorIndex  uint8
			name        string
			expectedRGB uint32
			issue       string
		}{
			{0x22, "Sky Blue", 0x5C94FC, "Should be blue sky, not magenta background"},
			{0x16, "Mario Red", 0xB40000, "Should be red hat/shirt, not blue"},
			{0x29, "Pipe Green", 0x00A800, "Should be green pipes, not red/blue"},
			{0x0F, "Black", 0x000000, "Should be black, not other color"},
		}
		
		for _, pc := range problematicColors {
			actualRGB := testPPU.NESColorToRGB(pc.colorIndex)
			
			if actualRGB != pc.expectedRGB {
				r := (actualRGB >> 16) & 0xFF
				g := (actualRGB >> 8) & 0xFF
				b := actualRGB & 0xFF
				
				er := (pc.expectedRGB >> 16) & 0xFF
				eg := (pc.expectedRGB >> 8) & 0xFF
				eb := pc.expectedRGB & 0xFF
				
				t.Errorf("Color %s (index $%02X) REGRESSION: %s\n"+
					"  Expected: #%06X RGB(%d,%d,%d)\n"+
					"  Actual:   #%06X RGB(%d,%d,%d)",
					pc.name, pc.colorIndex, pc.issue,
					pc.expectedRGB, er, eg, eb,
					actualRGB, r, g, b)
			}
		}
		
		t.Logf("SUCCESS: All problematic colors now render correctly")
	})
	
	t.Run("Validate color emphasis does not break channel integrity", func(t *testing.T) {
		// Ensure color emphasis (PPU mask bits 5-7) doesn't break color channel integrity
		testPPU.WriteRegister(0x2001, 0x00) // No emphasis
		normalSkyRGB := testPPU.NESColorToRGB(0x22)
		
		testPPU.WriteRegister(0x2001, 0x20) // Red emphasis
		redEmphasisRGB := testPPU.NESColorToRGB(0x22)
		
		testPPU.WriteRegister(0x2001, 0x40) // Green emphasis  
		greenEmphasisRGB := testPPU.NESColorToRGB(0x22)
		
		testPPU.WriteRegister(0x2001, 0x80) // Blue emphasis
		blueEmphasisRGB := testPPU.NESColorToRGB(0x22)
		
		// All emphasis modes should still result in blue-dominant colors for sky blue
		emphasisTests := []struct {
			rgb  uint32
			name string
		}{
			{normalSkyRGB, "Normal"},
			{redEmphasisRGB, "Red Emphasis"},
			{greenEmphasisRGB, "Green Emphasis"},
			{blueEmphasisRGB, "Blue Emphasis"},
		}
		
		for _, et := range emphasisTests {
			r := (et.rgb >> 16) & 0xFF
			g := (et.rgb >> 8) & 0xFF
			b := et.rgb & 0xFF
			
			// Sky should remain blue-dominant even with emphasis
			if b < r {
				t.Errorf("Sky with %s should remain blue-dominant but got RGB(%d,%d,%d)",
					et.name, r, g, b)
			}
		}
		
		// Reset to normal
		testPPU.WriteRegister(0x2001, 0x00)
		
		t.Logf("SUCCESS: Color emphasis preserves channel integrity")
	})
}

// TestColorFixIntegrationSummary provides a summary validation of all color fixes
func TestColorFixIntegrationSummary(t *testing.T) {
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	testPPU := ppu.New()
	testPPU.SetMemory(ppuMem)

	t.Run("Integration test summary", func(t *testing.T) {
		// Test the key fix: Sky blue (0x22) should be blue, not magenta
		ppuMem.Write(0x3F00, 0x22)
		skyRGB := testPPU.NESColorToRGB(0x22)
		
		r := (skyRGB >> 16) & 0xFF
		g := (skyRGB >> 8) & 0xFF
		b := skyRGB & 0xFF
		
		// Primary validation: Blue is the dominant channel
		blueDominant := b > r && b > g
		
		// Secondary validation: Not magenta-tinted (red << blue)
		notMagenta := float64(r)/float64(b) < 0.5
		
		// Tertiary validation: Matches expected color
		correctColor := skyRGB == 0x5C94FC
		
		if !blueDominant {
			t.Errorf("FAILED: Sky blue not blue-dominant: RGB(%d,%d,%d)", r, g, b)
		}
		
		if !notMagenta {
			t.Errorf("FAILED: Sky blue has magenta tint: red/blue ratio %.2f", float64(r)/float64(b))
		}
		
		if !correctColor {
			t.Errorf("FAILED: Sky blue wrong color: expected #5C94FC, got #%06X", skyRGB)
		}
		
		if blueDominant && notMagenta && correctColor {
			t.Logf("✅ COLOR FIX VALIDATION PASSED")
			t.Logf("✅ Sky blue renders correctly as RGB(%d,%d,%d) = #%06X", r, g, b, skyRGB)
			t.Logf("✅ No magenta background bug detected")
			t.Logf("✅ PPU color system working correctly")
		}
	})
}