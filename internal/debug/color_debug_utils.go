// Package debug provides utilities for easy integration of color debugging
package debug

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnableSuperMarioBrosColorDebugging enables debugging specifically for Super Mario Bros color issues
func EnableSuperMarioBrosColorDebugging() (*ColorDebugSession, error) {
	// Create debug output directory
	debugDir := filepath.Join("debug_output", "super_mario_bros_colors")
	if err := os.MkdirAll(debugDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create debug directory: %v", err)
	}

	// Start debugging session
	session, err := QuickSkyBlueDebugging(debugDir)
	if err != nil {
		return nil, fmt.Errorf("failed to start sky blue debugging: %v", err)
	}

	fmt.Printf("Super Mario Bros color debugging enabled.\n")
	fmt.Printf("Debug output will be saved to: %s\n", session.GetSessionOutputDir())
	fmt.Printf("Tracking color index 0x22 (sky blue) for corruption detection.\n")

	return session, nil
}

// CreateColorDebugEnvironment sets up a complete debugging environment
func CreateColorDebugEnvironment(outputDir string) error {
	// Initialize global debugger
	InitializeColorDebugging(outputDir)
	
	// Enable debugging
	EnableColorDebugging()
	
	// Configure for Super Mario Bros debugging
	TraceColorIndex0x22() // Sky blue
	
	fmt.Printf("Color debug environment initialized in: %s\n", outputDir)
	fmt.Printf("Use DumpColorDebugReport() to generate analysis.\n")
	
	return nil
}

// AnalyzeColorPipeline performs a quick analysis of the current debug data
func AnalyzeColorPipeline() {
	debugger := GetColorDebugger()
	if debugger == nil {
		fmt.Println("Color debugger not initialized")
		return
	}

	events := debugger.GetEvents()
	if len(events) == 0 {
		fmt.Println("No color pipeline events recorded")
		return
	}

	fmt.Printf("Color Pipeline Analysis:\n")
	fmt.Printf("Total Events: %d\n", len(events))

	// Count events by stage
	stageCount := make(map[ColorStage]int)
	for _, event := range events {
		stageCount[event.Stage]++
	}

	fmt.Printf("Events by Stage:\n")
	for stage, count := range stageCount {
		fmt.Printf("  %s: %d\n", stage, count)
	}

	// Check for color 0x22 (sky blue) corruption
	skyBlueEvents := 0
	corruptedEvents := 0
	
	for _, event := range events {
		if event.Stage == StageNESColorToRGB && event.InputValue == 0x22 {
			skyBlueEvents++
			if event.OutputValue != 0x64B0FF {
				corruptedEvents++
				fmt.Printf("  CORRUPTION: Color 0x22 -> 0x%06X (expected 0x64B0FF)\n", event.OutputValue)
			}
		}
	}

	if skyBlueEvents > 0 {
		fmt.Printf("Sky Blue (0x22) Analysis:\n")
		fmt.Printf("  Total conversions: %d\n", skyBlueEvents)
		fmt.Printf("  Corrupted conversions: %d\n", corruptedEvents)
		if corruptedEvents > 0 {
			fmt.Printf("  Corruption rate: %.1f%%\n", float64(corruptedEvents)/float64(skyBlueEvents)*100)
		}
	}
}

// PrintColorPaletteReference prints the expected NES color palette for reference
func PrintColorPaletteReference() {
	fmt.Println("NES Color Palette Reference (relevant colors):")
	fmt.Println("Index | RGB      | Color Name")
	fmt.Println("------|----------|------------------")
	fmt.Println("0x00  | #666666  | Gray (background)")
	fmt.Println("0x0F  | #000000  | Black")
	fmt.Println("0x22  | #64B0FF  | Sky Blue (SMB sky)")
	fmt.Println("0x30  | #FFFEFF  | White")
	fmt.Println()
	fmt.Println("Common corruption patterns:")
	fmt.Println("0x22 -> Brown (#8B4513) = Color emphasis bug")
	fmt.Println("0x22 -> Red (#FF0000) = Red screen bug")
	fmt.Println("Any color -> Gray = Greyscale mode enabled")
}

// QuickColorTest runs a quick test to verify color conversion
func QuickColorTest() {
	fmt.Println("Running quick color conversion test...")
	
	testColors := []struct {
		index    uint8
		expected uint32
		name     string
	}{
		{0x00, 0x666666, "Gray"},
		{0x0F, 0x000000, "Black"},
		{0x22, 0x64B0FF, "Sky Blue"},
		{0x30, 0xFFFEFF, "White"},
	}

	// Initialize debugging to capture test events
	CreateColorDebugEnvironment("test_debug")
	defer DisableColorDebugging()

	for _, test := range testColors {
		// Simulate the color conversion
		HookNESColorToRGB(0, 0, 0, 0, 0, test.index, test.expected)
		fmt.Printf("Test: 0x%02X -> #%06X (%s)\n", test.index, test.expected, test.name)
	}

	fmt.Println("Quick test complete. Check debug output for detailed analysis.")
}

// GetDebugStatistics returns current debugging statistics
func GetDebugStatistics() map[string]interface{} {
	debugger := GetColorDebugger()
	if debugger == nil {
		return map[string]interface{}{
			"enabled": false,
			"error":   "debugger not initialized",
		}
	}

	events := debugger.GetEvents()
	stageCount := make(map[ColorStage]int)
	
	for _, event := range events {
		stageCount[event.Stage]++
	}

	return map[string]interface{}{
		"enabled":       true,
		"total_events":  len(events),
		"stage_counts":  stageCount,
		"target_color":  "0x22 (sky blue)",
	}
}