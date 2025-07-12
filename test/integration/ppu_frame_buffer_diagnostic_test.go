package integration

import (
	"bytes"
	"io"
	"os"
	"testing"

	"gones/internal/cartridge"
)

// PPUFrameBufferDiagnostic performs comprehensive analysis of PPU frame buffer during actual ROM execution
type PPUFrameBufferDiagnostic struct {
	helper   *IntegrationTestHelper
	cartData []byte
}

// LoadSampleROM loads the sample.nes ROM and initializes the test helper
func (d *PPUFrameBufferDiagnostic) LoadSampleROM(t *testing.T) {
	file, err := os.Open("/home/claude/work/gones/roms/sample.nes")
	if err != nil {
		t.Fatalf("Failed to open sample.nes: %v", err)
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		t.Fatalf("Failed to read sample.nes: %v", err)
	}

	d.cartData = data

	reader := bytes.NewReader(data)
	cart, err := cartridge.LoadFromReader(reader)
	if err != nil {
		t.Fatalf("Failed to load sample.nes cartridge: %v", err)
	}

	d.helper = NewIntegrationTestHelper()
	d.helper.Cartridge = cart
	d.helper.Bus.LoadCartridge(cart)
	d.helper.UpdateReferences()
}

// ExecuteROMAndCaptureFrames runs the ROM and captures multiple frames for analysis
func (d *PPUFrameBufferDiagnostic) ExecuteROMAndCaptureFrames(t *testing.T, numFrames int) []FrameCapture {
	d.helper.Bus.Reset()
	
	frames := make([]FrameCapture, 0, numFrames)
	
	for frame := 0; frame < numFrames; frame++ {
		// Run a complete frame
		d.helper.RunFrame()
		
		// Capture frame buffer and PPU state
		frameBuffer := d.helper.PPU.GetFrameBuffer()
		
		capture := FrameCapture{
			FrameNumber: frame,
			FrameBuffer: frameBuffer,
			PPUState: PPUStateSnapshot{
				Scanline: d.helper.PPU.GetScanline(),
				Cycle:    d.helper.PPU.GetCycle(),
				Control:  d.helper.PPU.ReadRegister(0x2000),
				Mask:     d.helper.PPU.ReadRegister(0x2001),
				Status:   d.helper.PPU.ReadRegister(0x2002),
			},
		}
		
		// Capture palette state
		capture.PaletteState = d.capturePaletteState()
		
		frames = append(frames, capture)
		
		t.Logf("Frame %d captured: PPUCTRL=0x%02X, PPUMASK=0x%02X, PPUSTATUS=0x%02X", 
			frame, capture.PPUState.Control, capture.PPUState.Mask, capture.PPUState.Status)
	}
	
	return frames
}

// capturePaletteState reads all palette memory for analysis
func (d *PPUFrameBufferDiagnostic) capturePaletteState() PaletteState {
	palette := PaletteState{}
	
	// Save current PPUADDR state
	originalPPUCTRL := d.helper.PPU.ReadRegister(0x2000)
	
	// Read background palettes
	for i := 0; i < 16; i++ {
		d.helper.PPU.WriteRegister(0x2006, 0x3F) // PPUADDR high
		d.helper.PPU.WriteRegister(0x2006, uint8(i)) // PPUADDR low
		palette.BackgroundPalette[i] = d.helper.PPU.ReadRegister(0x2007)
	}
	
	// Read sprite palettes
	for i := 0; i < 16; i++ {
		d.helper.PPU.WriteRegister(0x2006, 0x3F) // PPUADDR high
		d.helper.PPU.WriteRegister(0x2006, uint8(0x10+i)) // PPUADDR low
		palette.SpritePalette[i] = d.helper.PPU.ReadRegister(0x2007)
	}
	
	// Restore PPUCTRL
	d.helper.PPU.WriteRegister(0x2000, originalPPUCTRL)
	
	return palette
}

// FrameCapture represents a captured frame and its associated state
type FrameCapture struct {
	FrameNumber  int
	FrameBuffer  [256 * 240]uint32
	PPUState     PPUStateSnapshot
	PaletteState PaletteState
}


// PaletteState captures all palette memory
type PaletteState struct {
	BackgroundPalette [16]uint8
	SpritePalette     [16]uint8
}

// AnalyzeFrameBuffer performs detailed pixel analysis on a frame buffer
func (d *PPUFrameBufferDiagnostic) AnalyzeFrameBuffer(t *testing.T, frame FrameCapture) FrameAnalysis {
	analysis := FrameAnalysis{
		FrameNumber:    frame.FrameNumber,
		TotalPixels:    256 * 240,
		ColorHistogram: make(map[uint32]int),
	}
	
	// Count pixel colors
	for _, pixel := range frame.FrameBuffer {
		analysis.ColorHistogram[pixel]++
	}
	
	// Find dominant colors
	maxCount := 0
	for color, count := range analysis.ColorHistogram {
		if count > maxCount {
			maxCount = count
			analysis.DominantColor = color
		}
	}
	analysis.DominantColorCount = maxCount
	
	// Check for red screen (common bug)
	redPixels := 0
	for color, count := range analysis.ColorHistogram {
		if d.isRedColor(color) {
			redPixels += count
		}
	}
	analysis.RedPixelCount = redPixels
	analysis.RedScreenPercentage = float64(redPixels) / float64(analysis.TotalPixels) * 100
	
	// Check for black screen
	blackPixels := analysis.ColorHistogram[0x000000] // Pure black
	analysis.BlackPixelCount = blackPixels
	analysis.BlackScreenPercentage = float64(blackPixels) / float64(analysis.TotalPixels) * 100
	
	// Count unique colors
	analysis.UniqueColors = len(analysis.ColorHistogram)
	
	// Analyze text region (approximate center area where "HELLO, WORLD!" should be)
	analysis.TextRegionAnalysis = d.analyzeTextRegion(frame.FrameBuffer)
	
	return analysis
}

// FrameAnalysis contains comprehensive frame buffer analysis results
type FrameAnalysis struct {
	FrameNumber           int
	TotalPixels          int
	ColorHistogram       map[uint32]int
	DominantColor        uint32
	DominantColorCount   int
	RedPixelCount        int
	RedScreenPercentage  float64
	BlackPixelCount      int
	BlackScreenPercentage float64
	UniqueColors         int
	TextRegionAnalysis   TextRegionAnalysis
}

// TextRegionAnalysis analyzes the region where text should appear
type TextRegionAnalysis struct {
	RegionPixels    int
	ForegroundPixels int
	BackgroundPixels int
	HasContrast     bool
	AverageColor    uint32
}

// analyzeTextRegion analyzes the center region where "HELLO, WORLD!" should appear
func (d *PPUFrameBufferDiagnostic) analyzeTextRegion(frameBuffer [256 * 240]uint32) TextRegionAnalysis {
	// Text region: approximately center of screen
	// "HELLO, WORLD!" is typically 13 characters wide, 8 pixels tall
	// Center it around scanline 112-120, pixels 80-176
	startX, endX := 80, 176
	startY, endY := 112, 120
	
	analysis := TextRegionAnalysis{}
	colorCounts := make(map[uint32]int)
	
	for y := startY; y < endY && y < 240; y++ {
		for x := startX; x < endX && x < 256; x++ {
			pixel := frameBuffer[y*256+x]
			colorCounts[pixel]++
			analysis.RegionPixels++
		}
	}
	
	// Determine if there's contrast in the text region
	if len(colorCounts) >= 2 {
		analysis.HasContrast = true
		
		// Find the two most common colors (should be background and foreground)
		maxCount1, maxCount2 := 0, 0
		var color1 uint32
		
		for color, count := range colorCounts {
			if count > maxCount1 {
				maxCount2 = maxCount1
				maxCount1 = count
				color1 = color
			} else if count > maxCount2 {
				maxCount2 = count
			}
		}
		
		analysis.BackgroundPixels = maxCount1
		analysis.ForegroundPixels = maxCount2
		analysis.AverageColor = color1 // Use most common as "average"
	} else if len(colorCounts) == 1 {
		// Single color - no contrast
		for color, count := range colorCounts {
			analysis.AverageColor = color
			analysis.BackgroundPixels = count
		}
	}
	
	return analysis
}

// isRedColor checks if a color is predominantly red
func (d *PPUFrameBufferDiagnostic) isRedColor(color uint32) bool {
	// Extract RGB components (assuming RGBA format)
	r := uint8((color >> 16) & 0xFF)
	g := uint8((color >> 8) & 0xFF)
	b := uint8(color & 0xFF)
	
	// Consider it red if red component is significantly higher than others
	return r > 128 && r > g*2 && r > b*2
}

// TestPPUFrameBufferDiagnostic_SampleROMExecution is the main diagnostic test
func TestPPUFrameBufferDiagnostic_SampleROMExecution(t *testing.T) {
	diagnostic := &PPUFrameBufferDiagnostic{}
	diagnostic.LoadSampleROM(t)
	
	t.Run("Initial PPU State Verification", func(t *testing.T) {
		// Check PPU state immediately after reset
		ppuCtrl := diagnostic.helper.PPU.ReadRegister(0x2000)
		ppuMask := diagnostic.helper.PPU.ReadRegister(0x2001)
		ppuStatus := diagnostic.helper.PPU.ReadRegister(0x2002)
		
		t.Logf("Initial PPU state: CTRL=0x%02X, MASK=0x%02X, STATUS=0x%02X", 
			ppuCtrl, ppuMask, ppuStatus)
		
		if ppuCtrl != 0x00 || ppuMask != 0x00 {
			t.Errorf("PPU not properly reset: CTRL=0x%02X, MASK=0x%02X", ppuCtrl, ppuMask)
		}
	})
	
	t.Run("Frame Buffer Capture and Analysis", func(t *testing.T) {
		// Capture 10 frames to see progression
		frames := diagnostic.ExecuteROMAndCaptureFrames(t, 10)
		
		for i, frame := range frames {
			analysis := diagnostic.AnalyzeFrameBuffer(t, frame)
			
			t.Logf("Frame %d Analysis:", i)
			t.Logf("  Dominant Color: 0x%08X (%d pixels, %.1f%%)", 
				analysis.DominantColor, analysis.DominantColorCount,
				float64(analysis.DominantColorCount)/float64(analysis.TotalPixels)*100)
			t.Logf("  Unique Colors: %d", analysis.UniqueColors)
			t.Logf("  Red Screen: %.1f%% (%d pixels)", 
				analysis.RedScreenPercentage, analysis.RedPixelCount)
			t.Logf("  Black Screen: %.1f%% (%d pixels)", 
				analysis.BlackScreenPercentage, analysis.BlackPixelCount)
			t.Logf("  Text Region: %d pixels, contrast=%t, fg=%d, bg=%d", 
				analysis.TextRegionAnalysis.RegionPixels,
				analysis.TextRegionAnalysis.HasContrast,
				analysis.TextRegionAnalysis.ForegroundPixels,
				analysis.TextRegionAnalysis.BackgroundPixels)
			
			// Check for red screen bug
			if analysis.RedScreenPercentage > 80 {
				t.Errorf("Frame %d: Red screen detected! %.1f%% red pixels", 
					i, analysis.RedScreenPercentage)
			}
			
			// Check for completely black screen
			if analysis.BlackScreenPercentage > 95 {
				t.Errorf("Frame %d: Black screen detected! %.1f%% black pixels", 
					i, analysis.BlackScreenPercentage)
			}
			
			// Check for lack of color variety (indicates rendering issues)
			if analysis.UniqueColors <= 2 && i > 5 {
				t.Errorf("Frame %d: Too few colors (%d) - rendering may not be working", 
					i, analysis.UniqueColors)
			}
			
			// Log palette state for key frames
			if i == 0 || i == 5 || i == 9 {
				t.Logf("Frame %d Palette State:", i)
				t.Logf("  Universal BG: 0x%02X", frame.PaletteState.BackgroundPalette[0])
				t.Logf("  BG Palette 0: [0x%02X, 0x%02X, 0x%02X, 0x%02X]", 
					frame.PaletteState.BackgroundPalette[0],
					frame.PaletteState.BackgroundPalette[1],
					frame.PaletteState.BackgroundPalette[2],
					frame.PaletteState.BackgroundPalette[3])
			}
		}
		
		// Compare first and last frames to see progression
		if len(frames) >= 2 {
			firstFrame := diagnostic.AnalyzeFrameBuffer(t, frames[0])
			lastFrame := diagnostic.AnalyzeFrameBuffer(t, frames[len(frames)-1])
			
			t.Logf("Frame Progression Analysis:")
			t.Logf("  Colors: %d -> %d", firstFrame.UniqueColors, lastFrame.UniqueColors)
			t.Logf("  Dominant Color: 0x%08X -> 0x%08X", 
				firstFrame.DominantColor, lastFrame.DominantColor)
			
			if firstFrame.UniqueColors == lastFrame.UniqueColors && 
			   firstFrame.DominantColor == lastFrame.DominantColor {
				t.Logf("Warning: No visual changes detected between first and last frame")
			}
		}
	})
	
	t.Run("PPU Register State Progression", func(t *testing.T) {
		// Reset and trace PPU register changes over time
		diagnostic.helper.Bus.Reset()
		
		states := make([]PPUStateSnapshot, 0, 10)
		
		for frame := 0; frame < 10; frame++ {
			diagnostic.helper.RunFrame()
			
			state := PPUStateSnapshot{
				Scanline: diagnostic.helper.PPU.GetScanline(),
				Cycle:    diagnostic.helper.PPU.GetCycle(),
				Control:  diagnostic.helper.PPU.ReadRegister(0x2000),
				Mask:     diagnostic.helper.PPU.ReadRegister(0x2001),
				Status:   diagnostic.helper.PPU.ReadRegister(0x2002),
			}
			
			states = append(states, state)
			
			if frame < 5 || frame%2 == 0 {
				t.Logf("Frame %d PPU State: CTRL=0x%02X, MASK=0x%02X, STATUS=0x%02X", 
					frame, state.Control, state.Mask, state.Status)
			}
		}
		
		// Check for expected state changes
		renderingEnabled := false
		for i, state := range states {
			if (state.Mask & 0x18) != 0 {
				renderingEnabled = true
				t.Logf("PPU rendering enabled at frame %d", i)
				break
			}
		}
		
		if !renderingEnabled {
			t.Error("PPU rendering never enabled - ROM execution may have failed")
		}
		
		// Check for palette initialization
		diagnostic.helper.PPU.WriteRegister(0x2006, 0x3F)
		diagnostic.helper.PPU.WriteRegister(0x2006, 0x00)
		universalBG := diagnostic.helper.PPU.ReadRegister(0x2007)
		
		if universalBG == 0x00 {
			t.Error("Universal background color still 0x00 - palette not initialized")
		} else {
			t.Logf("Universal background color: 0x%02X", universalBG)
		}
	})
	
	t.Run("Sample Pixel Locations", func(t *testing.T) {
		// Run ROM and capture final frame
		diagnostic.helper.Bus.Reset()
		for i := 0; i < 15; i++ {
			diagnostic.helper.RunFrame()
		}
		
		frameBuffer := diagnostic.helper.PPU.GetFrameBuffer()
		
		// Sample key locations where we expect specific content
		samplePoints := []struct {
			x, y     int
			desc     string
			expected string
		}{
			{10, 10, "Top-left corner", "background"},
			{128, 112, "Center of screen (text area)", "text or background"},
			{245, 10, "Top-right corner", "background"},
			{10, 230, "Bottom-left corner", "background"},
			{245, 230, "Bottom-right corner", "background"},
			{80, 116, "Start of 'HELLO' text", "text foreground or background"},
			{144, 116, "Start of 'WORLD' text", "text foreground or background"},
		}
		
		for _, point := range samplePoints {
			if point.x < 256 && point.y < 240 {
				pixel := frameBuffer[point.y*256+point.x]
				t.Logf("Pixel at (%d,%d) [%s]: 0x%08X", 
					point.x, point.y, point.desc, pixel)
			}
		}
		
		// Check for consistent background color
		backgroundSamples := []struct{ x, y int }{
			{10, 10}, {245, 10}, {10, 230}, {245, 230}, {50, 50}, {200, 200},
		}
		
		backgroundColors := make(map[uint32]int)
		for _, sample := range backgroundSamples {
			if sample.x < 256 && sample.y < 240 {
				pixel := frameBuffer[sample.y*256+sample.x]
				backgroundColors[pixel]++
			}
		}
		
		t.Logf("Background color distribution:")
		for color, count := range backgroundColors {
			t.Logf("  0x%08X: %d samples", color, count)
		}
		
		// Most background should be the same color
		maxCount := 0
		for _, count := range backgroundColors {
			if count > maxCount {
				maxCount = count
			}
		}
		
		if maxCount < len(backgroundSamples)/2 {
			t.Logf("Warning: Background colors inconsistent (max %d/%d samples)", 
				maxCount, len(backgroundSamples))
		}
	})
}

// TestPPUFrameBufferDiagnostic_ColorSystemVerification verifies the color generation system
func TestPPUFrameBufferDiagnostic_ColorSystemVerification(t *testing.T) {
	diagnostic := &PPUFrameBufferDiagnostic{}
	diagnostic.LoadSampleROM(t)
	
	t.Run("Direct Palette Manipulation", func(t *testing.T) {
		// Test direct palette writes and reads to verify the color system
		testColors := []uint8{0x0F, 0x30, 0x10, 0x00, 0x06, 0x16, 0x26, 0x36}
		
		for i, color := range testColors {
			// Write to palette
			diagnostic.helper.PPU.WriteRegister(0x2006, 0x3F)
			diagnostic.helper.PPU.WriteRegister(0x2006, uint8(i))
			diagnostic.helper.PPU.WriteRegister(0x2007, color)
			
			// Read back
			diagnostic.helper.PPU.WriteRegister(0x2006, 0x3F)
			diagnostic.helper.PPU.WriteRegister(0x2006, uint8(i))
			readColor := diagnostic.helper.PPU.ReadRegister(0x2007)
			
			if readColor != color {
				t.Errorf("Palette write/read mismatch at 0x3F%02X: wrote 0x%02X, read 0x%02X", 
					i, color, readColor)
			} else {
				t.Logf("Palette[0x3F%02X] = 0x%02X (verified)", i, color)
			}
		}
	})
	
	t.Run("Frame Buffer Color Generation", func(t *testing.T) {
		// Set specific palette and test frame buffer generation
		diagnostic.helper.PPU.WriteRegister(0x2006, 0x3F)
		diagnostic.helper.PPU.WriteRegister(0x2006, 0x00)
		diagnostic.helper.PPU.WriteRegister(0x2007, 0x0F) // Black background
		
		// Clear frame buffer first
		diagnostic.helper.PPU.ClearFrameBuffer(0x12345678) // Distinct clear color
		
		// Run one frame to generate pixels
		diagnostic.helper.RunFrame()
		
		frameBuffer := diagnostic.helper.PPU.GetFrameBuffer()
		
		// Check if frame buffer was modified from clear color
		clearColorCount := 0
		for _, pixel := range frameBuffer {
			if pixel == 0x12345678 {
				clearColorCount++
			}
		}
		
		modifiedPixels := len(frameBuffer) - clearColorCount
		t.Logf("Frame buffer modification: %d/%d pixels changed from clear color", 
			modifiedPixels, len(frameBuffer))
		
		if modifiedPixels == 0 {
			t.Error("Frame buffer not modified - rendering system may not be working")
		}
		
		// Sample some pixels to see what colors are being generated
		samplePixels := []struct{ x, y int }{
			{0, 0}, {128, 120}, {255, 239}, {64, 64}, {192, 180},
		}
		
		for _, sample := range samplePixels {
			pixel := frameBuffer[sample.y*256+sample.x]
			t.Logf("Generated pixel at (%d,%d): 0x%08X", sample.x, sample.y, pixel)
		}
	})
}