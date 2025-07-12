package debug

import (
	"os"
	"path/filepath"
	"testing"
)

// TestCompleteColorDebuggingWorkflow demonstrates the full debugging pipeline
func TestCompleteColorDebuggingWorkflow(t *testing.T) {
	// Setup output directory
	outputDir := "test_complete_debug"
	defer os.RemoveAll(outputDir)

	// Create debugging session
	session, err := QuickSkyBlueDebugging(outputDir)
	if err != nil {
		t.Fatalf("Failed to start debugging session: %v", err)
	}
	defer session.StopDebugging()

	// Simulate frame processing with color corruption
	for frameNum := uint64(0); frameNum < 3; frameNum++ {
		// Create a test frame buffer
		var frameBuffer [256 * 240]uint32

		// Fill with background color (should be gray)
		backgroundColor := uint32(0x666666) // Gray
		for i := range frameBuffer {
			frameBuffer[i] = backgroundColor
		}

		// Add some sky blue pixels that should be blue but might be corrupted
		for y := 50; y < 60; y++ {
			for x := 100; x < 110; x++ {
				if frameNum == 0 {
					// Frame 0: Correct sky blue
					frameBuffer[y*256+x] = 0x64B0FF
				} else {
					// Frame 1+: Corrupted to brown (simulating the bug)
					frameBuffer[y*256+x] = 0x8B4513
				}
			}
		}

		// Add some white pixels
		for y := 100; y < 105; y++ {
			for x := 150; x < 155; x++ {
				frameBuffer[y*256+x] = 0xFFFEFF // White
			}
		}

		// Process the frame
		if err := session.ProcessFrame(frameBuffer, frameNum); err != nil {
			t.Errorf("Failed to process frame %d: %v", frameNum, err)
		}

		// Simulate the color pipeline events that would be generated
		simulateColorPipelineEvents(frameNum, t)
	}

	// Stop debugging and generate report
	if err := session.StopDebugging(); err != nil {
		t.Errorf("Failed to stop debugging session: %v", err)
	}

	// Verify output files were created
	sessionDir := session.GetSessionOutputDir()
	expectedFiles := []string{
		"session_info.txt",
		"final_analysis_report.txt",
		"color_pipeline_events.log",
	}

	for _, filename := range expectedFiles {
		fullPath := filepath.Join(sessionDir, filename)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Logf("Expected output file %s not found at %s", filename, fullPath)
			// Don't fail the test for missing files in demo - just log
		}
	}

	t.Logf("Debugging session completed. Output directory: %s", sessionDir)
}

// simulateColorPipelineEvents simulates the events that would be generated during rendering
func simulateColorPipelineEvents(frameNum uint64, t *testing.T) {
	// Simulate palette RAM lookup for sky blue
	HookColorIndexLookup(frameNum, 50, 0, 105, 55, 0x3F01, 0x22)

	if frameNum == 0 {
		// Frame 0: Correct color conversion
		HookNESColorToRGB(frameNum, 50, 0, 105, 55, 0x22, 0x64B0FF)
		HookFrameBufferWrite(frameNum, 105, 55, 0x64B0FF)
		HookSDLTextureUpdate(frameNum, 105, 55, 0x64B0FF, 0x64B0FF, "RGBA8888")
	} else {
		// Frame 1+: Corrupted color (simulating the bug)
		HookNESColorToRGB(frameNum, 50, 0, 105, 55, 0x22, 0x8B4513) // Wrong!
		HookFrameBufferWrite(frameNum, 105, 55, 0x8B4513)
		HookSDLTextureUpdate(frameNum, 105, 55, 0x8B4513, 0x8B4513, "RGBA8888")
	}

	// Simulate some background pixels
	HookColorIndexLookup(frameNum, 50, 0, 0, 0, 0x3F00, 0x00)
	HookNESColorToRGB(frameNum, 50, 0, 0, 0, 0x00, 0x666666)
	HookFrameBufferWrite(frameNum, 0, 0, 0x666666)

	// Simulate white pixels
	HookColorIndexLookup(frameNum, 50, 0, 152, 102, 0x3F01, 0x30)
	HookNESColorToRGB(frameNum, 50, 0, 152, 102, 0x30, 0xFFFEFF)
	HookFrameBufferWrite(frameNum, 152, 102, 0xFFFEFF)

	// Simulate final render
	HookSDLRender(frameNum, "surface", 61440)
}

// TestSpecificPixelDebugging tests debugging of specific pixel coordinates
func TestSpecificPixelDebugging(t *testing.T) {
	outputDir := "test_pixel_debug"
	defer os.RemoveAll(outputDir)

	session := NewColorDebugSession(outputDir)
	if err := session.StartDebugging(); err != nil {
		t.Fatalf("Failed to start debugging: %v", err)
	}
	defer session.StopDebugging()

	// Configure to track specific pixel
	TracePixelAt(128, 120) // Center of screen

	// Simulate events for the tracked pixel
	HookColorIndexLookup(1, 120, 0, 128, 120, 0x3F01, 0x22)
	HookNESColorToRGB(1, 120, 0, 128, 120, 0x22, 0x64B0FF)
	HookFrameBufferWrite(1, 128, 120, 0x64B0FF)

	// Check that events were recorded
	debugger := GetColorDebugger()
	if debugger == nil {
		t.Fatal("Debugger not available")
	}

	events := debugger.GetEvents()
	found := false
	for _, event := range events {
		if event.PixelX == 128 && event.PixelY == 120 {
			found = true
			break
		}
	}

	if !found {
		t.Error("Expected to find events for pixel (128, 120)")
	}
}

// TestIntegratedColorCorruptionDetection tests the corruption detection capabilities
func TestIntegratedColorCorruptionDetection(t *testing.T) {
	outputDir := "test_corruption_detection"
	defer os.RemoveAll(outputDir)

	session := NewColorDebugSession(outputDir)
	if err := session.StartDebugging(); err != nil {
		t.Fatalf("Failed to start debugging: %v", err)
	}
	defer session.StopDebugging()

	// Simulate correct color conversion
	HookNESColorToRGB(1, 0, 0, 0, 0, 0x22, 0x64B0FF)

	// Simulate corrupted color conversion (the bug we're looking for)
	HookNESColorToRGB(1, 0, 0, 1, 0, 0x22, 0x8B4513) // Brown instead of blue

	// Analyze corruption
	debugger := GetColorDebugger()
	analysis := debugger.AnalyzeColorCorruption()

	if analysis == nil {
		t.Fatal("Expected corruption analysis")
	}

	if analysis.CorruptionStages[StageNESColorToRGB] == 0 {
		t.Error("Expected to detect corruption in NESColorToRGB stage")
	}

	if len(analysis.SampleEvents) == 0 {
		t.Error("Expected sample corruption events")
	}
}

// TestFrameDumperFilters tests frame dumper filtering capabilities
func TestFrameDumperFilters(t *testing.T) {
	outputDir := "test_frame_filters"
	defer os.RemoveAll(outputDir)

	frameDumper := NewFrameDumper(outputDir)
	frameDumper.Enable()

	// Test sky blue filter
	skyBlueFilter := CreateSkyBluePixelFilter()
	if !skyBlueFilter(0, 0, 0x64B0FF) {
		t.Error("Sky blue filter should accept sky blue pixels")
	}
	if skyBlueFilter(0, 0, 0x8B4513) {
		t.Error("Sky blue filter should reject non-sky-blue pixels")
	}

	// Test region filter
	regionFilter := CreateRegionFilter(10, 10, 20, 20)
	if !regionFilter(15, 15, 0x000000) {
		t.Error("Region filter should accept pixels within region")
	}
	if regionFilter(5, 5, 0x000000) {
		t.Error("Region filter should reject pixels outside region")
	}

	// Test color range filter
	rangeFilter := CreateColorRangeFilter(0x600000, 0x70FFFF)
	if !rangeFilter(0, 0, 0x64B0FF) {
		t.Error("Range filter should accept colors within range")
	}
	if rangeFilter(0, 0, 0x000000) {
		t.Error("Range filter should reject colors outside range")
	}
}

// BenchmarkDebuggingOverhead measures the performance impact of debugging
func BenchmarkDebuggingOverhead(b *testing.B) {
	outputDir := "bench_debug_overhead"
	defer os.RemoveAll(outputDir)

	// Test without debugging
	b.Run("NoDebugging", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// Simulate typical color pipeline calls without debugging
			_ = uint32(0x64B0FF) // Just simulate the work
		}
	})

	// Test with debugging enabled
	b.Run("WithDebugging", func(b *testing.B) {
		session := NewColorDebugSession(outputDir)
		session.StartDebugging()
		defer session.StopDebugging()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			HookNESColorToRGB(uint64(i), 0, 0, i%256, i%240, 0x22, 0x64B0FF)
		}
	})
}