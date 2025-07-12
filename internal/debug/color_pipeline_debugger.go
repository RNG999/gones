// Package debug provides comprehensive color pipeline debugging tools
package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// ColorStage represents a stage in the color transformation pipeline
type ColorStage string

const (
	StageColorIndexLookup ColorStage = "color_index_lookup"
	StagePaletteRAMLookup ColorStage = "palette_ram_lookup"
	StageNESColorToRGB    ColorStage = "nes_color_to_rgb"
	StageColorEmphasis    ColorStage = "color_emphasis"
	StageFrameBuffer      ColorStage = "frame_buffer"
	StageSDLTextureUpdate ColorStage = "sdl_texture_update"
	StageSDLRender        ColorStage = "sdl_render"
)

// ColorTransformationEvent represents a single color transformation step
type ColorTransformationEvent struct {
	Timestamp     time.Time
	Frame         uint64
	Scanline      int
	Cycle         int
	PixelX        int
	PixelY        int
	Stage         ColorStage
	InputValue    uint32
	OutputValue   uint32
	Description   string
	ExtraData     map[string]interface{}
}

// ColorPipelineDebugger traces color values through the rendering pipeline
type ColorPipelineDebugger struct {
	enabled        bool
	targetColor    uint8    // Color index to trace (e.g., 0x22 for sky blue)
	targetPixelX   int      // Specific pixel to trace
	targetPixelY   int      // Specific pixel to trace
	traceAllPixels bool     // If true, trace all pixels
	maxEvents      int      // Maximum events to store
	events         []ColorTransformationEvent
	outputDir      string   // Directory for debug output files
}

// NewColorPipelineDebugger creates a new color pipeline debugger
func NewColorPipelineDebugger(outputDir string) *ColorPipelineDebugger {
	return &ColorPipelineDebugger{
		enabled:        false,
		targetColor:    0x22, // Default to sky blue color index
		targetPixelX:   -1,   // -1 means track any pixel
		targetPixelY:   -1,
		traceAllPixels: false,
		maxEvents:      10000,
		events:         make([]ColorTransformationEvent, 0),
		outputDir:      outputDir,
	}
}

// Enable activates color pipeline debugging
func (cpd *ColorPipelineDebugger) Enable() {
	cpd.enabled = true
	// Ensure output directory exists
	os.MkdirAll(cpd.outputDir, 0755)
}

// Disable deactivates color pipeline debugging
func (cpd *ColorPipelineDebugger) Disable() {
	cpd.enabled = false
}

// SetTargetColor sets the specific color index to trace
func (cpd *ColorPipelineDebugger) SetTargetColor(colorIndex uint8) {
	cpd.targetColor = colorIndex
}

// SetTargetPixel sets the specific pixel coordinates to trace
func (cpd *ColorPipelineDebugger) SetTargetPixel(x, y int) {
	cpd.targetPixelX = x
	cpd.targetPixelY = y
}

// SetTraceAllPixels enables/disables tracing of all pixels
func (cpd *ColorPipelineDebugger) SetTraceAllPixels(enabled bool) {
	cpd.traceAllPixels = enabled
}

// IsTracking determines if we should trace this particular event
func (cpd *ColorPipelineDebugger) IsTracking(frame uint64, x, y int, colorIndex uint8) bool {
	if !cpd.enabled {
		return false
	}

	// If tracing all pixels, always track
	if cpd.traceAllPixels {
		return true
	}

	// Check if we're targeting a specific color
	if cpd.targetColor != 0xFF && colorIndex != cpd.targetColor {
		return false
	}

	// Check if we're targeting a specific pixel
	if cpd.targetPixelX >= 0 && cpd.targetPixelY >= 0 {
		return x == cpd.targetPixelX && y == cpd.targetPixelY
	}

	return true
}

// TraceColorTransformation records a color transformation event
func (cpd *ColorPipelineDebugger) TraceColorTransformation(
	frame uint64, scanline, cycle, x, y int,
	stage ColorStage, inputValue, outputValue uint32,
	description string, extraData map[string]interface{}) {

	if !cpd.enabled {
		return
	}

	// Check if we've exceeded max events
	if len(cpd.events) >= cpd.maxEvents {
		// Remove oldest events
		copy(cpd.events, cpd.events[1000:])
		cpd.events = cpd.events[:len(cpd.events)-1000]
	}

	event := ColorTransformationEvent{
		Timestamp:   time.Now(),
		Frame:       frame,
		Scanline:    scanline,
		Cycle:       cycle,
		PixelX:      x,
		PixelY:      y,
		Stage:       stage,
		InputValue:  inputValue,
		OutputValue: outputValue,
		Description: description,
		ExtraData:   extraData,
	}

	cpd.events = append(cpd.events, event)
}

// GetEvents returns all recorded events
func (cpd *ColorPipelineDebugger) GetEvents() []ColorTransformationEvent {
	return cpd.events
}

// ClearEvents clears all recorded events
func (cpd *ColorPipelineDebugger) ClearEvents() {
	cpd.events = cpd.events[:0]
}

// ExportEventsToFile writes all events to a detailed log file
func (cpd *ColorPipelineDebugger) ExportEventsToFile(filename string) error {
	if !cpd.enabled || len(cpd.events) == 0 {
		return fmt.Errorf("no events to export")
	}

	filePath := filepath.Join(cpd.outputDir, filename)
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create debug file: %v", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "Color Pipeline Debug Log\n")
	fmt.Fprintf(file, "Generated: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "Target Color: 0x%02X\n", cpd.targetColor)
	fmt.Fprintf(file, "Target Pixel: (%d, %d)\n", cpd.targetPixelX, cpd.targetPixelY)
	fmt.Fprintf(file, "Total Events: %d\n\n", len(cpd.events))

	// Write column headers
	fmt.Fprintf(file, "%-20s %-8s %-4s %-4s %-4s %-4s %-20s %-10s %-10s %s\n",
		"Timestamp", "Frame", "Line", "Cyc", "X", "Y", "Stage", "Input", "Output", "Description")
	fmt.Fprintf(file, "%s\n", "=" + string(make([]byte, 120)))

	// Write events
	for _, event := range cpd.events {
		fmt.Fprintf(file, "%-20s %-8d %-4d %-4d %-4d %-4d %-20s 0x%08X 0x%08X %s\n",
			event.Timestamp.Format("15:04:05.000"),
			event.Frame,
			event.Scanline,
			event.Cycle,
			event.PixelX,
			event.PixelY,
			event.Stage,
			event.InputValue,
			event.OutputValue,
			event.Description)

		// Add extra data if present
		if event.ExtraData != nil && len(event.ExtraData) > 0 {
			for key, value := range event.ExtraData {
				fmt.Fprintf(file, "%120s %s: %v\n", "", key, value)
			}
		}
	}

	return nil
}

// AnalyzeColorCorruption analyzes events to identify where color corruption occurs
func (cpd *ColorPipelineDebugger) AnalyzeColorCorruption() *ColorCorruptionAnalysis {
	if len(cpd.events) == 0 {
		return nil
	}

	analysis := &ColorCorruptionAnalysis{
		TotalEvents:      len(cpd.events),
		CorruptionStages: make(map[ColorStage]int),
		SampleEvents:     make([]ColorTransformationEvent, 0),
	}

	expectedColors := map[uint8]uint32{
		0x22: 0x64B0FF, // Expected sky blue RGB
		0x30: 0xFFFEFF, // Expected white RGB
		0x00: 0x666666, // Expected gray RGB
	}

	for _, event := range cpd.events {
		// Check for color corruption
		if event.Stage == StageNESColorToRGB {
			colorIndex := uint8(event.InputValue)
			if expectedRGB, exists := expectedColors[colorIndex]; exists {
				if event.OutputValue != expectedRGB {
					analysis.CorruptionStages[event.Stage]++
					if len(analysis.SampleEvents) < 10 {
						analysis.SampleEvents = append(analysis.SampleEvents, event)
					}
				}
			}
		}

		// Check for unexpected value changes
		if event.InputValue != 0 && event.OutputValue != 0 && event.InputValue != event.OutputValue {
			analysis.TransformationEvents++
		}
	}

	return analysis
}

// ColorCorruptionAnalysis contains analysis results
type ColorCorruptionAnalysis struct {
	TotalEvents          int
	TransformationEvents int
	CorruptionStages     map[ColorStage]int
	SampleEvents         []ColorTransformationEvent
}

// CreateColorComparisonReport generates a detailed comparison of expected vs actual colors
func (cpd *ColorPipelineDebugger) CreateColorComparisonReport() error {
	filename := fmt.Sprintf("color_comparison_report_%s.txt", time.Now().Format("20060102_150405"))
	filePath := filepath.Join(cpd.outputDir, filename)
	
	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create comparison report: %v", err)
	}
	defer file.Close()

	fmt.Fprintf(file, "Color Comparison Report\n")
	fmt.Fprintf(file, "======================\n\n")

	// Expected vs actual color mapping
	expectedColors := map[uint8]struct {
		RGB  uint32
		Name string
	}{
		0x22: {0x64B0FF, "Sky Blue"},
		0x30: {0xFFFEFF, "White"},
		0x00: {0x666666, "Gray"},
		0x0F: {0x000000, "Black"},
	}

	fmt.Fprintf(file, "Expected Color Values:\n")
	fmt.Fprintf(file, "Index | Name      | RGB      | Hex\n")
	fmt.Fprintf(file, "------|-----------|----------|----------\n")
	for index, color := range expectedColors {
		r := (color.RGB >> 16) & 0xFF
		g := (color.RGB >> 8) & 0xFF
		b := color.RGB & 0xFF
		fmt.Fprintf(file, "0x%02X  | %-9s | %3d,%3d,%3d | #%06X\n", 
			index, color.Name, r, g, b, color.RGB)
	}

	fmt.Fprintf(file, "\nActual Color Transformations:\n")
	fmt.Fprintf(file, "Frame | Pixel | Stage                | Input    | Output   | Match\n")
	fmt.Fprintf(file, "------|-------|----------------------|----------|----------|------\n")

	for _, event := range cpd.events {
		if event.Stage == StageNESColorToRGB {
			colorIndex := uint8(event.InputValue)
			match := "✓"
			if expected, exists := expectedColors[colorIndex]; exists {
				if event.OutputValue != expected.RGB {
					match = "✗"
				}
			} else {
				match = "?"
			}
			
			fmt.Fprintf(file, "%-5d | %3d,%3d | %-20s | 0x%06X | 0x%06X | %s\n",
				event.Frame, event.PixelX, event.PixelY, event.Stage,
				event.InputValue, event.OutputValue, match)
		}
	}

	return nil
}