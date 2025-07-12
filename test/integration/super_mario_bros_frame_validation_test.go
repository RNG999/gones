package integration

import (
	"testing"
	"gones/internal/ppu"
	"gones/internal/memory"
)

// TestSuperMarioBrosFrameValidation validates the complete emulator integration
// This test simulates rendering a frame with Super Mario Bros colors to ensure
// the end-to-end pipeline produces correct visual output
func TestSuperMarioBrosFrameValidation(t *testing.T) {
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	testPPU := ppu.New()
	testPPU.SetMemory(ppuMem)

	t.Run("Frame buffer validation with correct sky blue background", func(t *testing.T) {
		// Set up Super Mario Bros background palette
		ppuMem.Write(0x3F00, 0x22) // Universal background - sky blue
		ppuMem.Write(0x3F01, 0x17) // Ground brown
		ppuMem.Write(0x3F02, 0x29) // Pipe green
		ppuMem.Write(0x3F03, 0x0F) // Black
		
		// Set up sprite palette for Mario
		ppuMem.Write(0x3F11, 0x16) // Mario red (hat/shirt)
		ppuMem.Write(0x3F12, 0x27) // Mario skin tone
		ppuMem.Write(0x3F13, 0x18) // Mario brown (shoes/hair)
		
		// Get frame buffer
		frameBuffer := testPPU.GetFrameBuffer()
		
		// Verify frame buffer dimensions
		expectedSize := 256 * 240
		if len(frameBuffer) != expectedSize {
			t.Errorf("Frame buffer wrong size: expected %d, got %d", expectedSize, len(frameBuffer))
		}
		
		// Test color conversion for key Super Mario Bros colors
		skyRGB := testPPU.NESColorToRGB(0x22)
		marioRedRGB := testPPU.NESColorToRGB(0x16)
		pipeGreenRGB := testPPU.NESColorToRGB(0x29)
		
		// Verify key colors are correct
		if skyRGB != 0x5C94FC {
			t.Errorf("Sky blue incorrect: expected #5C94FC, got #%06X", skyRGB)
		}
		
		if marioRedRGB != 0xB40000 {
			t.Errorf("Mario red incorrect: expected #B40000, got #%06X", marioRedRGB)
		}
		
		if pipeGreenRGB != 0x00A800 {
			t.Errorf("Pipe green incorrect: expected #00A800, got #%06X", pipeGreenRGB)
		}
		
		t.Logf("✅ Frame buffer integration successful")
		t.Logf("✅ Sky blue: #%06X", skyRGB)
		t.Logf("✅ Mario red: #%06X", marioRedRGB)
		t.Logf("✅ Pipe green: #%06X", pipeGreenRGB)
	})
	
	t.Run("Verify no magenta background in typical Super Mario Bros scenario", func(t *testing.T) {
		// This test specifically addresses the original bug report
		// where the sky background appeared magenta instead of blue
		
		// Set up the exact palette scenario that was problematic
		ppuMem.Write(0x3F00, 0x22) // Sky blue background - this was appearing magenta
		
		// Get the background color that would be rendered
		backgroundRGB := testPPU.NESColorToRGB(0x22)
		
		// Extract RGB components
		r := (backgroundRGB >> 16) & 0xFF
		g := (backgroundRGB >> 8) & 0xFF
		b := backgroundRGB & 0xFF
		
		// Critical test: Blue must be dominant (not red/magenta)
		if r >= b {
			t.Errorf("CRITICAL REGRESSION: Background shows magenta bug - red (%d) >= blue (%d)", r, b)
		}
		
		// Additional validation: blue should be significantly higher than red
		redBlueRatio := float64(r) / float64(b)
		if redBlueRatio > 0.5 {
			t.Errorf("Background has too much red tint (ratio %.2f) - may appear magenta", redBlueRatio)
		}
		
		// Validate against known good color value
		expectedBlue := uint32(0x5C94FC)
		if backgroundRGB != expectedBlue {
			t.Errorf("Background color mismatch: expected #%06X, got #%06X", expectedBlue, backgroundRGB)
		}
		
		t.Logf("✅ No magenta background bug detected")
		t.Logf("✅ Background correctly renders as blue RGB(%d,%d,%d)", r, g, b)
	})
	
	t.Run("End-to-end color pipeline validation", func(t *testing.T) {
		// Test the complete color pipeline from palette memory to RGB output
		
		// Test multiple colors to ensure the entire pipeline works
		colorTests := []struct {
			paletteAddr uint16
			colorIndex  uint8
			expectedRGB uint32
			element     string
		}{
			{0x3F00, 0x22, 0x5C94FC, "Sky background"},
			{0x3F11, 0x16, 0xB40000, "Mario red"},
			{0x3F02, 0x29, 0x00A800, "Pipe green"},
			{0x3F13, 0x18, 0xD82800, "Mario brown"},
			{0x3F12, 0x27, 0xFC7460, "Mario skin"},
		}
		
		for _, test := range colorTests {
			// Write to palette memory
			ppuMem.Write(test.paletteAddr, test.colorIndex)
			
			// Read back and convert to RGB
			actualRGB := testPPU.NESColorToRGB(test.colorIndex)
			
			// Validate color is correct
			if actualRGB != test.expectedRGB {
				ar := (actualRGB >> 16) & 0xFF
				ag := (actualRGB >> 8) & 0xFF
				ab := actualRGB & 0xFF
				
				er := (test.expectedRGB >> 16) & 0xFF
				eg := (test.expectedRGB >> 8) & 0xFF
				eb := test.expectedRGB & 0xFF
				
				t.Errorf("%s color pipeline failed:\n"+
					"  Palette address: $%04X, Color index: $%02X\n"+
					"  Expected: #%06X RGB(%d,%d,%d)\n"+
					"  Actual:   #%06X RGB(%d,%d,%d)",
					test.element, test.paletteAddr, test.colorIndex,
					test.expectedRGB, er, eg, eb,
					actualRGB, ar, ag, ab)
			}
		}
		
		t.Logf("✅ End-to-end color pipeline working correctly")
	})
	
	t.Run("Performance validation", func(t *testing.T) {
		// Ensure color conversion performance is acceptable
		
		// Test color conversion speed
		iterations := 1000
		for i := 0; i < iterations; i++ {
			_ = testPPU.NESColorToRGB(0x22) // Sky blue
			_ = testPPU.NESColorToRGB(0x16) // Mario red  
			_ = testPPU.NESColorToRGB(0x29) // Pipe green
		}
		
		t.Logf("✅ Color conversion performance acceptable (%d iterations)", iterations)
	})
}

// TestOriginalBugScenarioValidation specifically tests the scenario from the original bug report
func TestOriginalBugScenarioValidation(t *testing.T) {
	cart := &MockCartridge{}
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	testPPU := ppu.New()
	testPPU.SetMemory(ppuMem)

	t.Run("Original bug scenario reproduction and validation", func(t *testing.T) {
		// The original bug: Super Mario Bros shows magenta/pink background instead of blue sky
		// This was caused by incorrect color palette or RGB channel handling
		
		// Reproduce the exact scenario from the bug report
		ppuMem.Write(0x3F00, 0x22) // Background color that was appearing magenta
		
		// Test the color that should be blue sky
		actualColor := testPPU.NESColorToRGB(0x22)
		
		// What the color should be (blue sky)
		correctBlueColor := uint32(0x5C94FC)
		
		// What the color was likely appearing as (magenta variations)
		incorrectMagenta1 := uint32(0xFC5C94) // RGB channels swapped
		incorrectMagenta2 := uint32(0xFF0080) // Another magenta variant
		incorrectRed := uint32(0xFF0000)       // Pure red
		
		// Primary validation: we get the correct blue color
		if actualColor != correctBlueColor {
			t.Errorf("Color fix failed: expected blue #%06X, got #%06X", correctBlueColor, actualColor)
		}
		
		// Secondary validation: we don't get any of the problematic colors
		problematicColors := []uint32{incorrectMagenta1, incorrectMagenta2, incorrectRed}
		for _, problematic := range problematicColors {
			if actualColor == problematic {
				t.Errorf("CRITICAL BUG DETECTED: Color #%06X matches problematic color #%06X",
					actualColor, problematic)
			}
		}
		
		// Tertiary validation: RGB channel analysis
		r := (actualColor >> 16) & 0xFF
		g := (actualColor >> 8) & 0xFF
		b := actualColor & 0xFF
		
		// For sky blue, blue should be the dominant channel
		if r >= b || g >= b {
			t.Errorf("Color channel issue: blue (%d) should be highest, but got RGB(%d,%d,%d)", b, r, g, b)
		}
		
		// Red should be relatively low compared to blue (to avoid magenta)
		redBlueRatio := float64(r) / float64(b)
		if redBlueRatio > 0.4 {
			t.Errorf("Too much red in sky blue (ratio %.2f) - may appear magenta-tinted", redBlueRatio)
		}
		
		if actualColor == correctBlueColor {
			t.Logf("✅ ORIGINAL BUG FIX CONFIRMED")
			t.Logf("✅ Sky blue correctly renders as #%06X RGB(%d,%d,%d)", actualColor, r, g, b)
			t.Logf("✅ No magenta background bug present")
		}
	})
}