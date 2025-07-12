package debug

import (
	"testing"
	"time"
)

// Color Pipeline Debugging Tests
// Tests that use the debugging infrastructure to capture and analyze
// color transformations in real-time

func TestColorPipelineDebuggingInfrastructure(t *testing.T) {
	// Test the debugging infrastructure itself
	
	// Initialize debugger
	InitializeColorDebugging("debug_output")
	debugger := GetColorDebugger()
	
	if debugger == nil {
		t.Fatal("Failed to initialize color debugger")
	}
	
	// Test debugger configuration
	debugger.SetTargetColor(0x22) // Sky blue
	debugger.Enable()
	
	// Test event recording
	testEvents := []struct {
		stage       ColorStage
		inputValue  uint32
		outputValue uint32
		description string
	}{
		{StageColorIndexLookup, 0x3F01, 0x22, "Palette lookup: $3F01 -> 0x22"},
		{StageNESColorToRGB, 0x22, 0x64B0FF, "NES color: 0x22 -> sky blue RGB"},
		{StageColorEmphasis, 0x64B0FF, 0x4C8CBF, "Red emphasis applied"},
		{StageFrameBuffer, 0x4C8CBF, 0x4C8CBF, "Written to frame buffer"},
	}
	
	for i, event := range testEvents {
		debugger.TraceColorTransformation(
			uint64(i/10), // frame
			i%10,         // scanline
			i*10,         // cycle
			100+i*2,      // x
			50+i,         // y
			event.stage,
			event.inputValue,
			event.outputValue,
			event.description,
			nil)
	}
	
	// Verify events were recorded
	events := debugger.GetEvents()
	if len(events) != len(testEvents) {
		t.Errorf("Expected %d events, got %d", len(testEvents), len(events))
	}
	
	// Test event export
	err := debugger.ExportEventsToFile("test_color_events.log")
	if err != nil {
		t.Errorf("Failed to export events: %v", err)
	}
	
	// Test corruption analysis
	analysis := debugger.AnalyzeColorCorruption()
	if analysis == nil {
		t.Error("Analysis should not be nil with recorded events")
	} else {
		t.Logf("Analysis results: %d total events, %d transformations",
			analysis.TotalEvents, analysis.TransformationEvents)
	}
	
	debugger.Disable()
}

func TestColorPipelineHookTesting(t *testing.T) {
	// Test the color pipeline hooks in isolation
	
	InitializeColorDebugging("debug_output")
	EnableColorDebugging()
	defer DisableColorDebugging()
	
	// Test each hook function
	frame := uint64(1)
	scanline, cycle := 120, 200
	x, y := 128, 100
	
	// Test color index lookup hook
	HookColorIndexLookup(frame, scanline, cycle, x, y, 0x3F01, 0x22)
	
	// Test NES color to RGB hook
	HookNESColorToRGB(frame, scanline, cycle, x, y, 0x22, 0x64B0FF)
	
	// Test color emphasis hook
	HookColorEmphasis(frame, scanline, cycle, x, y, 0x64B0FF, 0x4C8CBF, 0x20)
	
	// Test frame buffer hook
	HookFrameBufferWrite(frame, x, y, 0x4C8CBF)
	
	// Test SDL hooks
	HookSDLTextureUpdate(frame, x, y, 0x4C8CBF, 0x4C8CBF, "RGBA8888")
	HookSDLRender(frame, "SDL_RenderPresent", 256*240)
	
	// Verify all hooks recorded events
	debugger := GetColorDebugger()
	if debugger != nil {
		events := debugger.GetEvents()
		expectedStages := []ColorStage{
			StageColorIndexLookup,
			StageNESColorToRGB,
			StageColorEmphasis,
			StageFrameBuffer,
			StageSDLTextureUpdate,
			StageSDLRender,
		}
		
		if len(events) < len(expectedStages) {
			t.Errorf("Expected at least %d events from hooks, got %d", len(expectedStages), len(events))
		}
		
		// Verify we got events for each stage
		stagesSeen := make(map[ColorStage]bool)
		for _, event := range events {
			stagesSeen[event.Stage] = true
		}
		
		for _, expectedStage := range expectedStages {
			if !stagesSeen[expectedStage] {
				t.Errorf("Missing events for stage: %s", expectedStage)
			}
		}
		
		t.Logf("Hook testing recorded %d events across %d stages", len(events), len(stagesSeen))
	}
}

func TestColorCorruptionScenarios(t *testing.T) {
	// Test various color corruption scenarios using the debugging tools
	
	InitializeColorDebugging("debug_output")
	debugger := GetColorDebugger()
	if debugger != nil {
		debugger.SetTargetColor(0x22) // Focus on sky blue
		debugger.Enable()
	}
	defer DisableColorDebugging()
	
	frame := uint64(1)
	
	// Scenario 1: Normal color pipeline (no corruption)
	t.Run("NormalPipeline", func(t *testing.T) {
		debugger.ClearEvents()
		
		// Simulate normal color transformation
		HookColorIndexLookup(frame, 100, 150, 64, 50, 0x3F01, 0x22)
		HookNESColorToRGB(frame, 100, 150, 64, 50, 0x22, 0x64B0FF)
		HookColorEmphasis(frame, 100, 150, 64, 50, 0x64B0FF, 0x64B0FF, 0x00) // No emphasis
		HookFrameBufferWrite(frame, 64, 50, 0x64B0FF)
		
		// Analyze for corruption
		analysis := debugger.AnalyzeColorCorruption()
		if analysis != nil && len(analysis.CorruptionStages) > 0 {
			t.Error("Normal pipeline should not show corruption")
		}
	})
	
	// Scenario 2: Red emphasis corruption
	t.Run("RedEmphasisCorruption", func(t *testing.T) {
		debugger.ClearEvents()
		
		// Simulate red emphasis causing brown tint
		HookColorIndexLookup(frame, 100, 150, 64, 50, 0x3F01, 0x22)
		HookNESColorToRGB(frame, 100, 150, 64, 50, 0x22, 0x64B0FF)
		HookColorEmphasis(frame, 100, 150, 64, 50, 0x64B0FF, 0x8B4513, 0x20) // Red emphasis -> brown
		HookFrameBufferWrite(frame, 64, 50, 0x8B4513)
		
		// Analyze for corruption
		analysis := debugger.AnalyzeColorCorruption()
		if analysis != nil {
			// Should detect corruption at emphasis stage
			if count, exists := analysis.CorruptionStages[StageColorEmphasis]; !exists || count == 0 {
				t.Error("Should detect corruption at color emphasis stage")
			}
		}
	})
	
	// Scenario 3: Palette corruption
	t.Run("PaletteCorruption", func(t *testing.T) {
		debugger.ClearEvents()
		
		// Simulate wrong color in palette
		HookColorIndexLookup(frame, 100, 150, 64, 50, 0x3F01, 0x16) // Wrong color index
		HookNESColorToRGB(frame, 100, 150, 64, 50, 0x16, 0xB53120) // Red instead of blue
		HookColorEmphasis(frame, 100, 150, 64, 50, 0xB53120, 0xB53120, 0x00)
		HookFrameBufferWrite(frame, 64, 50, 0xB53120)
		
		// This type of corruption is harder to detect automatically,
		// but the events should be recorded for manual analysis
		events := debugger.GetEvents()
		if len(events) == 0 {
			t.Error("Events should be recorded even for palette corruption")
		}
	})
	
	// Scenario 4: SDL conversion corruption  
	t.Run("SDLConversionCorruption", func(t *testing.T) {
		debugger.ClearEvents()
		
		// Simulate SDL format conversion corruption
		HookColorIndexLookup(frame, 100, 150, 64, 50, 0x3F01, 0x22)
		HookNESColorToRGB(frame, 100, 150, 64, 50, 0x22, 0x64B0FF)
		HookColorEmphasis(frame, 100, 150, 64, 50, 0x64B0FF, 0x64B0FF, 0x00)
		HookFrameBufferWrite(frame, 64, 50, 0x64B0FF)
		HookSDLTextureUpdate(frame, 64, 50, 0x64B0FF, 0x64B0AA, "RGB565") // Slight color loss
		
		// Check events show the conversion difference
		events := debugger.GetEvents()
		sdlEvents := 0
		for _, event := range events {
			if event.Stage == StageSDLTextureUpdate {
				sdlEvents++
				if event.InputValue == event.OutputValue {
					t.Error("SDL conversion should show input != output when format conversion occurs")
				}
			}
		}
		
		if sdlEvents == 0 {
			t.Error("Should have recorded SDL texture update events")
		}
	})
}

func TestColorPipelinePerformanceImpact(t *testing.T) {
	// Test that the debugging system doesn't significantly impact performance
	
	// Baseline: disabled debugging
	start := time.Now()
	iterations := 10000
	
	for i := 0; i < iterations; i++ {
		// Simulate color transformations without debugging
		_ = simulateColorTransformation(0x22)
	}
	baselineDuration := time.Since(start)
	
	// With debugging enabled
	InitializeColorDebugging("debug_output")
	EnableColorDebugging()
	
	start = time.Now()
	for i := 0; i < iterations; i++ {
		// Simulate color transformations with debugging
		frame := uint64(i / 1000)
		HookNESColorToRGB(frame, i%262, i%341, i%256, i%240, 0x22, 0x64B0FF)
		_ = simulateColorTransformation(0x22)
	}
	debuggingDuration := time.Since(start)
	
	DisableColorDebugging()
	
	// Calculate overhead
	overhead := float64(debuggingDuration-baselineDuration) / float64(baselineDuration) * 100
	
	t.Logf("Performance test results:")
	t.Logf("  Baseline: %v", baselineDuration)
	t.Logf("  With debugging: %v", debuggingDuration)
	t.Logf("  Overhead: %.1f%%", overhead)
	
	// Debugging should not add more than 50% overhead
	if overhead > 50 {
		t.Errorf("Debugging overhead too high: %.1f%% (should be < 50%%)", overhead)
	}
}

func TestColorPipelineEventFiltering(t *testing.T) {
	// Test event filtering and targeting capabilities
	
	InitializeColorDebugging("debug_output")
	debugger := GetColorDebugger()
	if debugger == nil {
		t.Fatal("Failed to initialize debugger")
	}
	
	// Test target color filtering
	t.Run("TargetColorFiltering", func(t *testing.T) {
		debugger.SetTargetColor(0x22) // Only track sky blue
		debugger.Enable()
		debugger.ClearEvents()
		
		// Send events for different colors
		frame := uint64(1)
		HookNESColorToRGB(frame, 100, 150, 64, 50, 0x22, 0x64B0FF) // Should be tracked
		HookNESColorToRGB(frame, 100, 151, 65, 50, 0x16, 0xB53120) // Should be ignored
		HookNESColorToRGB(frame, 100, 152, 66, 50, 0x22, 0x64B0FF) // Should be tracked
		
		events := debugger.GetEvents()
		if len(events) != 2 {
			t.Errorf("Expected 2 events with target color filtering, got %d", len(events))
		}
		
		// Verify all events are for the target color
		for _, event := range events {
			if event.Stage == StageNESColorToRGB && event.InputValue != 0x22 {
				t.Errorf("Event should only include target color 0x22, got 0x%02X", event.InputValue)
			}
		}
	})
	
	// Test target pixel filtering
	t.Run("TargetPixelFiltering", func(t *testing.T) {
		debugger.SetTargetPixel(128, 100) // Only track specific pixel
		debugger.SetTargetColor(0xFF)     // Track any color
		debugger.ClearEvents()
		
		frame := uint64(1)
		HookFrameBufferWrite(frame, 128, 100, 0x64B0FF) // Should be tracked
		HookFrameBufferWrite(frame, 129, 100, 0x64B0FF) // Should be ignored
		HookFrameBufferWrite(frame, 128, 101, 0x64B0FF) // Should be ignored
		HookFrameBufferWrite(frame, 128, 100, 0xB53120) // Should be tracked
		
		events := debugger.GetEvents()
		if len(events) != 2 {
			t.Errorf("Expected 2 events with target pixel filtering, got %d", len(events))
		}
		
		// Verify all events are for the target pixel
		for _, event := range events {
			if event.PixelX != 128 || event.PixelY != 100 {
				t.Errorf("Event should only include target pixel (128,100), got (%d,%d)",
					event.PixelX, event.PixelY)
			}
		}
	})
	
	// Test trace all pixels mode
	t.Run("TraceAllPixelsMode", func(t *testing.T) {
		debugger.SetTraceAllPixels(true)
		debugger.SetTargetColor(0x22) // This should be ignored in trace-all mode
		debugger.SetTargetPixel(-1, -1) // This should be ignored in trace-all mode
		debugger.ClearEvents()
		
		frame := uint64(1)
		HookNESColorToRGB(frame, 100, 150, 64, 50, 0x22, 0x64B0FF)
		HookNESColorToRGB(frame, 100, 151, 65, 50, 0x16, 0xB53120)
		HookNESColorToRGB(frame, 100, 152, 66, 50, 0x30, 0xFFFEFF)
		
		events := debugger.GetEvents()
		if len(events) != 3 {
			t.Errorf("Expected 3 events in trace-all mode, got %d", len(events))
		}
	})
	
	debugger.Disable()
}

func TestColorPipelineReportGeneration(t *testing.T) {
	// Test comprehensive report generation
	
	InitializeColorDebugging("debug_output")
	debugger := GetColorDebugger()
	if debugger == nil {
		t.Fatal("Failed to initialize debugger")
	}
	
	debugger.Enable()
	
	// Generate sample data
	frame := uint64(1)
	testScenarios := []struct {
		colorIndex uint8
		expectedRGB uint32
		actualRGB uint32
		description string
	}{
		{0x22, 0x64B0FF, 0x64B0FF, "Normal sky blue"},
		{0x22, 0x64B0FF, 0x8B4513, "Corrupted sky blue -> brown"},
		{0x16, 0xB53120, 0xB53120, "Normal red"},
		{0x30, 0xFFFEFF, 0xFFFEFF, "Normal white"},
	}
	
	for i, scenario := range testScenarios {
		x, y := 100+i*10, 50+i*5
		
		// Simulate complete pipeline
		HookColorIndexLookup(frame, 100+i, 150+i*2, x, y, 0x3F01, scenario.colorIndex)
		HookNESColorToRGB(frame, 100+i, 150+i*2, x, y, scenario.colorIndex, scenario.expectedRGB)
		
		// Simulate corruption if present
		finalRGB := scenario.actualRGB
		if scenario.expectedRGB != scenario.actualRGB {
			HookColorEmphasis(frame, 100+i, 150+i*2, x, y, scenario.expectedRGB, scenario.actualRGB, 0x20)
		} else {
			HookColorEmphasis(frame, 100+i, 150+i*2, x, y, scenario.expectedRGB, scenario.actualRGB, 0x00)
		}
		
		HookFrameBufferWrite(frame, x, y, finalRGB)
	}
	
	// Test event export
	err := debugger.ExportEventsToFile("comprehensive_test_events.log")
	if err != nil {
		t.Errorf("Failed to export events: %v", err)
	}
	
	// Test comparison report
	err = debugger.CreateColorComparisonReport()
	if err != nil {
		t.Errorf("Failed to create comparison report: %v", err)
	}
	
	// Test corruption analysis
	analysis := debugger.AnalyzeColorCorruption()
	if analysis == nil {
		t.Error("Analysis should not be nil")
	} else {
		t.Logf("Analysis summary:")
		t.Logf("  Total events: %d", analysis.TotalEvents)
		t.Logf("  Transformation events: %d", analysis.TransformationEvents)
		
		for stage, count := range analysis.CorruptionStages {
			t.Logf("  %s corruptions: %d", stage, count)
		}
		
		// Should detect the brown corruption we injected
		if len(analysis.SampleEvents) == 0 {
			t.Error("Should have sample events for analysis")
		}
	}
	
	// Test full debug report dump
	err = DumpColorDebugReport()
	if err != nil {
		t.Errorf("Failed to dump debug report: %v", err)
	}
	
	debugger.Disable()
}

// Helper functions

func simulateColorTransformation(colorIndex uint8) uint32 {
	// Simulate basic color transformation for performance testing
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
	
	if int(colorIndex) < len(palette) {
		return palette[colorIndex]
	}
	return 0
}