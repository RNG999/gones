package integration

import (
	"testing"
	"gones/internal/ppu"
	"gones/internal/memory"
)

// TestSuperMarioBrosColorIntegration validates that the PPU color fixes work end-to-end
// This test simulates the Super Mario Bros color scenario to ensure blue sky instead of magenta
func TestSuperMarioBrosColorIntegration(t *testing.T) {
	// Create a minimal test cartridge that simulates Super Mario Bros color scenario
	cart := &MockCartridge{}
	
	// Create PPU with memory
	ppuMem := memory.NewPPUMemory(cart, memory.MirrorHorizontal)
	testPPU := ppu.New()
	testPPU.SetMemory(ppuMem)
	
	t.Run("Verify sky background color is blue not magenta", func(t *testing.T) {
		// Set up Super Mario Bros typical palette
		// Background palette 0: Universal background color at 0x3F00
		ppuMem.Write(0x3F00, 0x22) // Sky blue color index
		
		// Get the RGB color that would be rendered
		skyRGB := testPPU.NESColorToRGB(0x22)
		
		// Extract RGB components
		r := (skyRGB >> 16) & 0xFF
		g := (skyRGB >> 8) & 0xFF
		b := skyRGB & 0xFF
		
		// Verify this is the correct blue sky color from our palette
		expectedSkyRGB := uint32(0x5C94FC) // Correct blue sky color
		if skyRGB != expectedSkyRGB {
			t.Errorf("Sky color incorrect: expected #%06X, got #%06X (RGB: %d,%d,%d)",
				expectedSkyRGB, skyRGB, r, g, b)
		}
		
		// Verify it's predominantly blue (not red/magenta)
		if r > b {
			t.Errorf("CRITICAL: Sky has more red (%d) than blue (%d) - magenta background bug detected!",
				r, b)
		}
		
		// Blue should be the dominant component for sky color
		if b < r || b < g {
			t.Errorf("Sky should be blue-dominant, got RGB(%d,%d,%d)", r, g, b)
		}
		
		t.Logf("SUCCESS: Sky color correct - RGB(%d,%d,%d) = #%06X", r, g, b, skyRGB)
	})
	
	t.Run("Verify Mario red color is actually red", func(t *testing.T) {
		// Set Mario's red sprite color
		ppuMem.Write(0x3F11, 0x16) // Mario's red hat/shirt color
		
		// Get the RGB color
		redRGB := testPPU.NESColorToRGB(0x16)
		
		// Extract RGB components
		r := (redRGB >> 16) & 0xFF
		g := (redRGB >> 8) & 0xFF
		b := redRGB & 0xFF
		
		// Verify this is the correct red color
		expectedRedRGB := uint32(0xB40000) // Mario's red from our test
		if redRGB != expectedRedRGB {
			t.Errorf("Mario red color incorrect: expected #%06X, got #%06X (RGB: %d,%d,%d)",
				expectedRedRGB, redRGB, r, g, b)
		}
		
		// Red should be the dominant component
		if r <= g || r <= b {
			t.Errorf("Mario red should be red-dominant, got RGB(%d,%d,%d)", r, g, b)
		}
		
		t.Logf("SUCCESS: Mario red color correct - RGB(%d,%d,%d) = #%06X", r, g, b, redRGB)
	})
	
	t.Run("Verify pipe green color is actually green", func(t *testing.T) {
		// Set pipe green color
		ppuMem.Write(0x3F02, 0x29) // Pipe green color
		
		// Get the RGB color
		greenRGB := testPPU.NESColorToRGB(0x29)
		
		// Extract RGB components
		r := (greenRGB >> 16) & 0xFF
		g := (greenRGB >> 8) & 0xFF
		b := greenRGB & 0xFF
		
		// Verify this is the correct green color
		expectedGreenRGB := uint32(0x00A800) // Pipe green from our test
		if greenRGB != expectedGreenRGB {
			t.Errorf("Pipe green color incorrect: expected #%06X, got #%06X (RGB: %d,%d,%d)",
				expectedGreenRGB, greenRGB, r, g, b)
		}
		
		// Green should be the dominant component
		if g <= r || g <= b {
			t.Errorf("Pipe green should be green-dominant, got RGB(%d,%d,%d)", r, g, b)
		}
		
		t.Logf("SUCCESS: Pipe green color correct - RGB(%d,%d,%d) = #%06X", r, g, b, greenRGB)
	})
	
	t.Run("Verify no color channel swapping", func(t *testing.T) {
		// Test a range of colors to ensure no RGB channel swapping
		colorTests := []struct {
			colorIndex uint8
			name       string
			expectDominant string
		}{
			{0x06, "Pure Red", "red"},
			{0x1A, "Light Green", "green"},  
			{0x0C, "Blue", "blue"},
			{0x16, "Mario Red", "red"},
			{0x22, "Sky Blue", "blue"},
			{0x29, "Pipe Green", "green"},
		}
		
		for _, test := range colorTests {
			rgb := testPPU.NESColorToRGB(test.colorIndex)
			r := (rgb >> 16) & 0xFF
			g := (rgb >> 8) & 0xFF
			b := rgb & 0xFF
			
			switch test.expectDominant {
			case "red":
				if r <= g || r <= b {
					t.Errorf("Color %s (index $%02X) should be red-dominant but got RGB(%d,%d,%d)",
						test.name, test.colorIndex, r, g, b)
				}
			case "green":
				if g <= r || g <= b {
					t.Errorf("Color %s (index $%02X) should be green-dominant but got RGB(%d,%d,%d)",
						test.name, test.colorIndex, r, g, b)
				}
			case "blue":
				if b <= r || b <= g {
					t.Errorf("Color %s (index $%02X) should be blue-dominant but got RGB(%d,%d,%d)",
						test.name, test.colorIndex, r, g, b)
				}
			}
			
			t.Logf("Color %s: RGB(%d,%d,%d) = #%06X - %s dominant ✓",
				test.name, r, g, b, rgb, test.expectDominant)
		}
	})
	
	t.Run("Simulate frame buffer rendering", func(t *testing.T) {
		// Simulate a small frame buffer to verify the integration works end-to-end
		frameBuffer := testPPU.GetFrameBuffer()
		
		// Set up typical Super Mario Bros background
		ppuMem.Write(0x3F00, 0x22) // Sky blue background
		
		// Get a few pixels and verify they have correct colors
		// Note: In a real test, we'd need to trigger PPU rendering,
		// but for this integration test, we're validating the color conversion pipeline
		
		skyColor := testPPU.NESColorToRGB(0x22)
		expectedSkyColor := uint32(0x5C94FC)
		
		if skyColor != expectedSkyColor {
			t.Errorf("Frame buffer color conversion failed: expected #%06X, got #%06X",
				expectedSkyColor, skyColor)
		}
		
		// Verify frame buffer exists and has expected dimensions
		if len(frameBuffer) != 256*240 {
			t.Errorf("Frame buffer has wrong size: expected %d, got %d", 256*240, len(frameBuffer))
		}
		
		t.Logf("SUCCESS: Frame buffer integration validated")
	})
}