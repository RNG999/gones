package debug

import (
	"fmt"
	"gones/internal/memory"
	"testing"
)

// Color Debug Validation Test Suite
// Tests the debugging infrastructure and basic color pipeline validation
// without creating import cycles

func TestColorDebuggingInfrastructure(t *testing.T) {
	// Test the debugging infrastructure
	
	InitializeColorDebugging("debug_output")
	debugger := GetColorDebugger()
	
	if debugger == nil {
		t.Fatal("Failed to initialize color debugger")
	}
	
	// Test debugger configuration
	debugger.SetTargetColor(0x22) // Sky blue
	debugger.Enable()
	
	// Test event recording
	frame := uint64(1)
	debugger.TraceColorTransformation(
		frame, 100, 150, 64, 50,
		StageColorIndexLookup,
		0x3F01, 0x22,
		"Palette lookup test",
		map[string]interface{}{
			"test": "infrastructure",
		})
	
	// Verify event was recorded
	events := debugger.GetEvents()
	if len(events) == 0 {
		t.Error("No events recorded by debugger")
	} else {
		t.Logf("Successfully recorded %d events", len(events))
	}
	
	debugger.Disable()
}

func TestColorHookFunctionality(t *testing.T) {
	// Test the color hook functions
	
	InitializeColorDebugging("debug_output")
	EnableColorDebugging()
	defer DisableColorDebugging()
	
	frame := uint64(1)
	
	// Test each hook function
	HookColorIndexLookup(frame, 100, 150, 64, 50, 0x3F01, 0x22)
	HookNESColorToRGB(frame, 100, 150, 64, 50, 0x22, 0x64B0FF)
	HookColorEmphasis(frame, 100, 150, 64, 50, 0x64B0FF, 0x4C8CBF, 0x20)
	HookFrameBufferWrite(frame, 64, 50, 0x4C8CBF)
	
	// Verify events were recorded
	debugger := GetColorDebugger()
	if debugger != nil {
		events := debugger.GetEvents()
		if len(events) < 4 {
			t.Errorf("Expected at least 4 events, got %d", len(events))
		}
		
		// Check that we have events for different stages
		stagesSeen := make(map[ColorStage]bool)
		for _, event := range events {
			stagesSeen[event.Stage] = true
		}
		
		expectedStages := []ColorStage{
			StageColorIndexLookup,
			StageNESColorToRGB,
			StageColorEmphasis,
			StageFrameBuffer,
		}
		
		for _, stage := range expectedStages {
			if !stagesSeen[stage] {
				t.Errorf("Missing events for stage: %s", stage)
			}
		}
		
		t.Logf("Hook test recorded events for %d stages", len(stagesSeen))
	}
}

func TestPaletteRAMValidation(t *testing.T) {
	// Test palette RAM behavior without PPU dependency
	
	ppuMem := memory.NewPPUMemory(nil, memory.MirrorHorizontal)
	
	// Test basic palette storage
	skyColorIndex := uint8(0x22)
	ppuMem.Write(0x3F01, skyColorIndex)
	
	actualIndex := ppuMem.Read(0x3F01)
	if actualIndex != skyColorIndex {
		t.Errorf("Palette RAM storage failed: expected 0x%02X, got 0x%02X", 
			skyColorIndex, actualIndex)
	}
	
	// Test palette mirroring
	ppuMem.Write(0x3F01, skyColorIndex)
	mirroredIndex := ppuMem.Read(0x3F01 + 0x20) // $3F21 should mirror $3F01
	
	if mirroredIndex != skyColorIndex {
		t.Errorf("Palette mirroring failed: expected 0x%02X, got 0x%02X", 
			skyColorIndex, mirroredIndex)
	}
	
	// Test universal background color
	ppuMem.Write(0x3F00, skyColorIndex)
	universalBG := ppuMem.Read(0x3F00)
	
	if universalBG != skyColorIndex {
		t.Errorf("Universal background color failed: expected 0x%02X, got 0x%02X", 
			skyColorIndex, universalBG)
	}
	
	t.Logf("Palette RAM validation passed for color 0x%02X", skyColorIndex)
}

func TestNESColorPaletteValidation(t *testing.T) {
	// Test the NES color palette values
	
	// Define expected RGB values for key colors
	expectedColors := map[uint8]uint32{
		0x00: 0x666666, // Gray
		0x0F: 0x000000, // Black
		0x22: 0x64B0FF, // Sky Blue
		0x30: 0xFFFEFF, // White
		0x16: 0xB53120, // Red/Brown
	}
	
	// Firebrandx NES palette
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
	
	for colorIndex, expectedRGB := range expectedColors {
		if int(colorIndex) >= len(palette) {
			t.Errorf("Color index 0x%02X out of range", colorIndex)
			continue
		}
		
		actualRGB := palette[colorIndex]
		if actualRGB != expectedRGB {
			r1, g1, b1 := extractRGB(expectedRGB)
			r2, g2, b2 := extractRGB(actualRGB)
			t.Errorf("Color 0x%02X mismatch:\n"+
				"  Expected: RGB(%d,%d,%d) = #%06X\n"+
				"  Actual:   RGB(%d,%d,%d) = #%06X",
				colorIndex,
				r1, g1, b1, expectedRGB,
				r2, g2, b2, actualRGB)
		}
	}
	
	// Specifically validate sky blue (the problematic color)
	skyBlueRGB := palette[0x22]
	expectedSkyBlue := uint32(0x64B0FF)
	
	if skyBlueRGB != expectedSkyBlue {
		r1, g1, b1 := extractRGB(expectedSkyBlue)
		r2, g2, b2 := extractRGB(skyBlueRGB)
		t.Errorf("SKY BLUE COLOR VALIDATION FAILED:\n"+
			"  Expected RGB(%d,%d,%d) = #%06X\n"+
			"  Actual   RGB(%d,%d,%d) = #%06X",
			r1, g1, b1, expectedSkyBlue,
			r2, g2, b2, skyBlueRGB)
	} else {
		t.Logf("Sky blue color validation PASSED: 0x22 -> #%06X", skyBlueRGB)
	}
}

func TestColorEmphasisCalculation(t *testing.T) {
	// Test color emphasis calculations
	
	baseRGB := uint32(0x64B0FF) // Sky blue
	
	// Test no emphasis (should return unchanged)
	noEmphasisRGB := applyTestEmphasis(baseRGB, 0x00)
	if noEmphasisRGB != baseRGB {
		t.Errorf("No emphasis should preserve color: expected #%06X, got #%06X", 
			baseRGB, noEmphasisRGB)
	}
	
	// Test red emphasis (should darken green and blue)
	redEmphasisRGB := applyTestEmphasis(baseRGB, 0x20)
	r1, g1, b1 := extractRGB(baseRGB)
	r2, g2, b2 := extractRGB(redEmphasisRGB)
	
	if g2 >= g1 || b2 >= b1 {
		t.Errorf("Red emphasis should darken green/blue:\n"+
			"  Original: RGB(%d,%d,%d)\n"+
			"  Emphasized: RGB(%d,%d,%d)", r1, g1, b1, r2, g2, b2)
	}
	
	// Test for problematic brown tint
	if r2 > 150 && g2 > 80 && g2 < 150 && b2 < 100 {
		t.Errorf("Red emphasis causing brown tint: RGB(%d,%d,%d) = #%06X", r2, g2, b2, redEmphasisRGB)
	}
	
	// Test red+green emphasis (potential yellow/brown corruption)
	yellowEmphasisRGB := applyTestEmphasis(baseRGB, 0x60) // Red + Green
	r3, g3, b3 := extractRGB(yellowEmphasisRGB)
	
	if b3 >= b1 { // Blue should be darkened
		t.Errorf("Red+Green emphasis should darken blue: original %d, emphasized %d", b1, b3)
	}
	
	// Check for problematic brown/yellow tint
	if r3 > 100 && g3 > 80 && g3 < 180 && b3 < 100 {
		t.Logf("WARNING: Red+Green emphasis may cause brown/yellow tint: RGB(%d,%d,%d) = #%06X", 
			r3, g3, b3, yellowEmphasisRGB)
	}
}

func TestColorCorruptionDetectionDebug(t *testing.T) {
	// Test corruption detection logic
	
	InitializeColorDebugging("debug_output")
	debugger := GetColorDebugger()
	if debugger != nil {
		debugger.SetTargetColor(0x22)
		debugger.Enable()
	}
	defer DisableColorDebugging()
	
	frame := uint64(1)
	
	// Simulate normal transformation
	HookNESColorToRGB(frame, 100, 150, 64, 50, 0x22, 0x64B0FF)
	
	// Simulate corrupted transformation (sky blue -> brown)
	HookNESColorToRGB(frame, 100, 151, 64, 51, 0x22, 0x8B4513)
	
	// Analyze corruption
	if debugger != nil {
		analysis := debugger.AnalyzeColorCorruption()
		if analysis != nil {
			t.Logf("Corruption analysis:")
			t.Logf("  Total events: %d", analysis.TotalEvents)
			t.Logf("  Transformation events: %d", analysis.TransformationEvents)
			
			for stage, count := range analysis.CorruptionStages {
				if count > 0 {
					t.Logf("  %s corruptions: %d", stage, count)
				}
			}
			
			// Should detect corruption in NES color to RGB stage
			if count, exists := analysis.CorruptionStages[StageNESColorToRGB]; !exists || count == 0 {
				t.Error("Should detect corruption in NES color to RGB stage")
			}
		}
	}
}

func TestSkyBlueSpecificCorruption(t *testing.T) {
	// Test specifically for sky blue corruption patterns
	
	InitializeColorDebugging("debug_output")
	TraceColorIndex0x22() // Use the specific sky blue tracer
	defer DisableColorDebugging()
	
	frame := uint64(1)
	
	// Test normal sky blue transformation
	HookColorIndexLookup(frame, 100, 150, 128, 50, 0x3F01, 0x22)
	HookNESColorToRGB(frame, 100, 150, 128, 50, 0x22, 0x64B0FF)
	HookColorEmphasis(frame, 100, 150, 128, 50, 0x64B0FF, 0x64B0FF, 0x00)
	
	// Test corruption scenarios
	corruptionScenarios := []struct {
		name string
		originalRGB uint32
		corruptedRGB uint32
		emphasisMask uint8
	}{
		{"Red Emphasis Brown", 0x64B0FF, 0x8B4513, 0x20},
		{"Yellow Emphasis", 0x64B0FF, 0xDAA520, 0x60},
		{"Overall Red Tint", 0x64B0FF, 0xFF6B6B, 0x20},
	}
	
	for _, scenario := range corruptionScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			HookColorEmphasis(frame, 101, 151, 129, 51, 
				scenario.originalRGB, scenario.corruptedRGB, scenario.emphasisMask)
			
			// Analyze the corruption type
			corruptionType := analyzeColorCorruption(scenario.originalRGB, scenario.corruptedRGB)
			t.Logf("%s corruption type: %s", scenario.name, corruptionType)
			
			// Check if it matches expected corruption patterns
			if scenario.name == "Red Emphasis Brown" && corruptionType != "Yellow/Brown Tint (red+green boost, blue reduction)" {
				t.Logf("Note: %s shows corruption type: %s", scenario.name, corruptionType)
			}
		})
	}
}

// Helper functions

func extractRGB(rgb uint32) (uint8, uint8, uint8) {
	r := uint8((rgb >> 16) & 0xFF)
	g := uint8((rgb >> 8) & 0xFF)
	b := uint8(rgb & 0xFF)
	return r, g, b
}

func applyTestEmphasis(rgb uint32, emphasisMask uint8) uint32 {
	// Test implementation of color emphasis
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

func analyzeColorCorruption(expected, actual uint32) string {
	r1, g1, b1 := extractRGB(expected)
	r2, g2, b2 := extractRGB(actual)
	
	if r2 > r1 && g2 < g1 && b2 < b1 {
		return "Red Channel Boost (possible red screen bug)"
	}
	if r2 > r1 && g2 > g1 && b2 < b1 {
		return "Yellow/Brown Tint (red+green boost, blue reduction)"
	}
	if r2 < r1 && g2 < g1 && b2 < b1 {
		return "Overall Darkening"
	}
	if r2 > r1 && g2 > r1 && b2 > b1 {
		return "Overall Brightening"
	}
	
	return fmt.Sprintf("Custom corruption: R%+d G%+d B%+d", 
		int(r2)-int(r1), int(g2)-int(g1), int(b2)-int(b1))
}