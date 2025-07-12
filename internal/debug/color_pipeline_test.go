package debug

import (
	"os"
	"testing"
	"time"
)

func TestColorPipelineDebugger(t *testing.T) {
	// Create temporary directory for test output
	tmpDir := "test_debug_output"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Initialize debugger
	debugger := NewColorPipelineDebugger(tmpDir)
	debugger.Enable()
	debugger.SetTargetColor(0x22) // Track sky blue

	// Simulate some color transformations
	debugger.TraceColorTransformation(
		1, 0, 0, 100, 50,
		StageColorIndexLookup,
		0x3F01, 0x22,
		"Palette RAM lookup for sky blue",
		map[string]interface{}{
			"palette_address": "0x3F01",
			"color_index":     "0x22",
		})

	debugger.TraceColorTransformation(
		1, 0, 0, 100, 50,
		StageNESColorToRGB,
		0x22, 0x64B0FF,
		"NES color 0x22 -> RGB(100,176,255)",
		map[string]interface{}{
			"color_index": "0x22",
			"red":         100,
			"green":       176,
			"blue":        255,
		})

	// Simulate color corruption (blue -> brown)
	debugger.TraceColorTransformation(
		1, 0, 0, 100, 50,
		StageColorEmphasis,
		0x64B0FF, 0x8B4513, // Blue to brown
		"Color emphasis corruption: blue -> brown",
		map[string]interface{}{
			"mask_bits":       "0xE0",
			"original_rgb":    "RGB(100,176,255)",
			"emphasized_rgb":  "RGB(139,69,19)",
		})

	debugger.TraceColorTransformation(
		1, -1, -1, 100, 50,
		StageFrameBuffer,
		0, 0x8B4513,
		"Frame buffer write with corrupted color",
		map[string]interface{}{
			"pixel_index": 50*256 + 100,
			"red":         139,
			"green":       69,
			"blue":        19,
		})

	// Check that events were recorded
	events := debugger.GetEvents()
	if len(events) != 4 {
		t.Errorf("Expected 4 events, got %d", len(events))
	}

	// Verify the first event
	firstEvent := events[0]
	if firstEvent.Stage != StageColorIndexLookup {
		t.Errorf("Expected first event to be ColorIndexLookup, got %s", firstEvent.Stage)
	}
	if firstEvent.InputValue != 0x3F01 {
		t.Errorf("Expected input value 0x3F01, got 0x%X", firstEvent.InputValue)
	}
	if firstEvent.OutputValue != 0x22 {
		t.Errorf("Expected output value 0x22, got 0x%X", firstEvent.OutputValue)
	}

	// Export events to file
	err := debugger.ExportEventsToFile("test_events.log")
	if err != nil {
		t.Errorf("Failed to export events: %v", err)
	}

	// Create comparison report
	err = debugger.CreateColorComparisonReport()
	if err != nil {
		t.Errorf("Failed to create comparison report: %v", err)
	}

	// Analyze corruption
	analysis := debugger.AnalyzeColorCorruption()
	if analysis == nil {
		t.Error("Expected corruption analysis, got nil")
	}
	if analysis.TotalEvents != 4 {
		t.Errorf("Expected 4 total events in analysis, got %d", analysis.TotalEvents)
	}
}

func TestColorHooks(t *testing.T) {
	// Create temporary directory for test output
	tmpDir := "test_hook_output"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Initialize global debugger
	InitializeColorDebugging(tmpDir)
	EnableColorDebugging()

	// Test different hook functions
	HookColorIndexLookup(1, 0, 0, 100, 50, 0x3F01, 0x22)
	HookNESColorToRGB(1, 0, 0, 100, 50, 0x22, 0x64B0FF)
	HookColorEmphasis(1, 0, 0, 100, 50, 0x64B0FF, 0x8B4513, 0xE0)
	HookFrameBufferWrite(1, 100, 50, 0x8B4513)
	HookSDLTextureUpdate(1, 100, 50, 0x8B4513, 0x8B4513, "RGBA8888")
	HookSDLRender(1, "surface", 61440)

	// Get debugger and check events
	debugger := GetColorDebugger()
	if debugger == nil {
		t.Fatal("Global debugger not initialized")
	}

	events := debugger.GetEvents()
	if len(events) < 6 {
		t.Errorf("Expected at least 6 events, got %d", len(events))
	}

	// Test specific color tracking
	TraceColorIndex0x22()
	TracePixelAt(128, 120)

	// Generate debug report
	err := DumpColorDebugReport()
	if err != nil {
		t.Errorf("Failed to dump color debug report: %v", err)
	}

	DisableColorDebugging()
}

func TestColorCorruptionDetection(t *testing.T) {
	tmpDir := "test_corruption_output"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	debugger := NewColorPipelineDebugger(tmpDir)
	debugger.Enable()

	// Simulate proper color transformation
	debugger.TraceColorTransformation(
		1, 0, 0, 0, 0,
		StageNESColorToRGB,
		0x22, 0x64B0FF, // Correct sky blue
		"Correct color transformation",
		nil)

	// Simulate color corruption
	debugger.TraceColorTransformation(
		1, 0, 0, 1, 0,
		StageNESColorToRGB,
		0x22, 0x8B4513, // Wrong - should be blue but got brown
		"Corrupted color transformation",
		nil)

	// Analyze corruption
	analysis := debugger.AnalyzeColorCorruption()
	if analysis == nil {
		t.Fatal("Expected corruption analysis")
	}

	// Should detect one corruption event
	if analysis.CorruptionStages[StageNESColorToRGB] != 1 {
		t.Errorf("Expected 1 corruption at NESColorToRGB stage, got %d",
			analysis.CorruptionStages[StageNESColorToRGB])
	}

	if len(analysis.SampleEvents) != 1 {
		t.Errorf("Expected 1 sample event, got %d", len(analysis.SampleEvents))
	}
}

func BenchmarkColorPipelineDebugger(b *testing.B) {
	tmpDir := "bench_debug_output"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	debugger := NewColorPipelineDebugger(tmpDir)
	debugger.Enable()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		debugger.TraceColorTransformation(
			uint64(i/1000), i%240, i%341, i%256, i%240,
			StageNESColorToRGB,
			uint32(i%64), uint32(i*0x1000),
			"Benchmark color transformation",
			nil)
	}
}

func TestFrameRateImpact(t *testing.T) {
	tmpDir := "test_framerate_output"
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	debugger := NewColorPipelineDebugger(tmpDir)

	// Test with debugging disabled
	start := time.Now()
	for i := 0; i < 61440; i++ { // One full frame
		debugger.TraceColorTransformation(
			1, i/256, 0, i%256, i/256,
			StageFrameBuffer,
			0, uint32(i),
			"Test pixel",
			nil)
	}
	disabledTime := time.Since(start)

	// Test with debugging enabled
	debugger.Enable()
	start = time.Now()
	for i := 0; i < 61440; i++ { // One full frame
		debugger.TraceColorTransformation(
			1, i/256, 0, i%256, i/256,
			StageFrameBuffer,
			0, uint32(i),
			"Test pixel",
			nil)
	}
	enabledTime := time.Since(start)

	t.Logf("Debug disabled: %v, Debug enabled: %v, Overhead: %.2fx",
		disabledTime, enabledTime, float64(enabledTime)/float64(disabledTime))

	// Overhead should be reasonable (less than 10x)
	if enabledTime > disabledTime*10 {
		t.Errorf("Debug overhead too high: %.2fx", float64(enabledTime)/float64(disabledTime))
	}
}