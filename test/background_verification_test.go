package test

import (
	"os"
	"testing"

	"gones/internal/bus"
	"gones/internal/cartridge"
)

func TestBackgroundRenderingVerification(t *testing.T) {
	t.Run("Sample ROM background rendering", func(t *testing.T) {
		romPath := "../roms/sample.nes"
		if _, err := os.Stat(romPath); os.IsNotExist(err) {
			t.Skip("sample.nes ROM not found, skipping test")
		}

		file, err := os.Open(romPath)
		if err != nil {
			t.Fatalf("Failed to open ROM file: %v", err)
		}
		defer file.Close()

		cart, err := cartridge.LoadFromReader(file)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		emulator := bus.New()
		emulator.LoadCartridge(cart)
		emulator.Reset()

		for i := 0; i < 50000; i++ {
			emulator.Step()
		}

		frameBuffer := emulator.GetFrameBuffer()
		
		blackPixelCount := 0
		whitePixelCount := 0
		redPixelCount := 0
		otherPixelCount := 0
		
		for _, pixel := range frameBuffer {
			r := (pixel >> 16) & 0xFF
			g := (pixel >> 8) & 0xFF
			b := pixel & 0xFF
			
			if r < 50 && g < 50 && b < 50 {
				blackPixelCount++
			} else if r > 200 && g > 200 && b > 200 {
				whitePixelCount++
			} else if r > 200 && g < 100 && b < 100 {
				redPixelCount++
			} else {
				otherPixelCount++
			}
		}
		
		totalPixels := len(frameBuffer)
		t.Logf("Background rendering analysis:")
		t.Logf("  Total pixels: %d", totalPixels)
		t.Logf("  Black pixels: %d (%.1f%%)", blackPixelCount, float64(blackPixelCount)*100/float64(totalPixels))
		t.Logf("  White pixels: %d (%.1f%%)", whitePixelCount, float64(whitePixelCount)*100/float64(totalPixels))
		t.Logf("  Red pixels: %d (%.1f%%)", redPixelCount, float64(redPixelCount)*100/float64(totalPixels))
		t.Logf("  Other pixels: %d (%.1f%%)", otherPixelCount, float64(otherPixelCount)*100/float64(totalPixels))
		
		if whitePixelCount == 0 {
			t.Error("No white pixels found - expected white text from sample ROM")
		}
		
		if blackPixelCount < totalPixels/2 {
			t.Logf("Warning: Less than 50%% black pixels, expected mostly black background")
		}
		
		if redPixelCount > totalPixels/4 {
			t.Error("Too many red pixels - red screen bug may still exist")
		}
		
		// Note: PPUCTRL and PPUMASK are write-only registers
		// Reading them returns open bus data, not the register values
		// Instead, check the actual PPU rendering state
		renderingEnabled := emulator.PPU.IsRenderingEnabled()
		t.Logf("PPU rendering enabled: %v", renderingEnabled)
		
		if !renderingEnabled {
			t.Error("Background rendering not enabled")
		}
	})
}