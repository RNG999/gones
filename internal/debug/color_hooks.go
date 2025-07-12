// Package debug provides color pipeline debugging hooks
package debug

import (
	"fmt"
	"sync"
)

// Global color pipeline debugger instance
var (
	globalDebugger *ColorPipelineDebugger
	debuggerMutex  sync.RWMutex
)

// InitializeColorDebugging initializes the global color debugger
func InitializeColorDebugging(outputDir string) {
	debuggerMutex.Lock()
	defer debuggerMutex.Unlock()
	
	globalDebugger = NewColorPipelineDebugger(outputDir)
}

// GetColorDebugger returns the global color debugger (thread-safe)
func GetColorDebugger() *ColorPipelineDebugger {
	debuggerMutex.RLock()
	defer debuggerMutex.RUnlock()
	return globalDebugger
}

// EnableColorDebugging enables color pipeline debugging
func EnableColorDebugging() {
	debuggerMutex.Lock()
	defer debuggerMutex.Unlock()
	
	if globalDebugger != nil {
		globalDebugger.Enable()
	}
}

// DisableColorDebugging disables color pipeline debugging
func DisableColorDebugging() {
	debuggerMutex.Lock()
	defer debuggerMutex.Unlock()
	
	if globalDebugger != nil {
		globalDebugger.Disable()
	}
}

// Hook functions for different stages of the color pipeline

// HookColorIndexLookup traces color index lookups in palette RAM
func HookColorIndexLookup(frame uint64, scanline, cycle, x, y int, paletteAddress uint16, colorIndex uint8) {
	debuggerMutex.RLock()
	debugger := globalDebugger
	debuggerMutex.RUnlock()
	
	if debugger == nil || !debugger.IsTracking(frame, x, y, colorIndex) {
		return
	}
	
	extraData := map[string]interface{}{
		"palette_address": fmt.Sprintf("0x%04X", paletteAddress),
		"color_index":     fmt.Sprintf("0x%02X", colorIndex),
	}
	
	debugger.TraceColorTransformation(
		frame, scanline, cycle, x, y,
		StageColorIndexLookup,
		uint32(paletteAddress), uint32(colorIndex),
		fmt.Sprintf("Palette RAM lookup at 0x%04X -> 0x%02X", paletteAddress, colorIndex),
		extraData)
}

// HookNESColorToRGB traces NES color index to RGB conversion
func HookNESColorToRGB(frame uint64, scanline, cycle, x, y int, colorIndex uint8, rgb uint32) {
	debuggerMutex.RLock()
	debugger := globalDebugger
	debuggerMutex.RUnlock()
	
	if debugger == nil || !debugger.IsTracking(frame, x, y, colorIndex) {
		return
	}
	
	r := (rgb >> 16) & 0xFF
	g := (rgb >> 8) & 0xFF
	b := rgb & 0xFF
	
	extraData := map[string]interface{}{
		"color_index": fmt.Sprintf("0x%02X", colorIndex),
		"red":         r,
		"green":       g,
		"blue":        b,
	}
	
	debugger.TraceColorTransformation(
		frame, scanline, cycle, x, y,
		StageNESColorToRGB,
		uint32(colorIndex), rgb,
		fmt.Sprintf("NES color 0x%02X -> RGB(%d,%d,%d)", colorIndex, r, g, b),
		extraData)
}

// HookColorEmphasis traces color emphasis application
func HookColorEmphasis(frame uint64, scanline, cycle, x, y int, originalRGB, emphasizedRGB uint32, maskBits uint8) {
	debuggerMutex.RLock()
	debugger := globalDebugger
	debuggerMutex.RUnlock()
	
	if debugger == nil {
		return
	}
	
	// Extract color components for both original and emphasized
	origR := (originalRGB >> 16) & 0xFF
	origG := (originalRGB >> 8) & 0xFF
	origB := originalRGB & 0xFF
	
	emphR := (emphasizedRGB >> 16) & 0xFF
	emphG := (emphasizedRGB >> 8) & 0xFF
	emphB := emphasizedRGB & 0xFF
	
	extraData := map[string]interface{}{
		"mask_bits":       fmt.Sprintf("0x%02X", maskBits),
		"red_emphasis":    (maskBits & 0x20) != 0,
		"green_emphasis":  (maskBits & 0x40) != 0,
		"blue_emphasis":   (maskBits & 0x80) != 0,
		"original_rgb":    fmt.Sprintf("RGB(%d,%d,%d)", origR, origG, origB),
		"emphasized_rgb":  fmt.Sprintf("RGB(%d,%d,%d)", emphR, emphG, emphB),
	}
	
	debugger.TraceColorTransformation(
		frame, scanline, cycle, x, y,
		StageColorEmphasis,
		originalRGB, emphasizedRGB,
		fmt.Sprintf("Color emphasis: RGB(%d,%d,%d) -> RGB(%d,%d,%d)", origR, origG, origB, emphR, emphG, emphB),
		extraData)
}

// HookFrameBufferWrite traces writes to the frame buffer
func HookFrameBufferWrite(frame uint64, x, y int, rgb uint32) {
	debuggerMutex.RLock()
	debugger := globalDebugger
	debuggerMutex.RUnlock()
	
	if debugger == nil {
		return
	}
	
	r := (rgb >> 16) & 0xFF
	g := (rgb >> 8) & 0xFF
	b := rgb & 0xFF
	
	pixelIndex := y*256 + x
	
	extraData := map[string]interface{}{
		"pixel_index": pixelIndex,
		"red":         r,
		"green":       g,
		"blue":        b,
	}
	
	debugger.TraceColorTransformation(
		frame, -1, -1, x, y,
		StageFrameBuffer,
		0, rgb,
		fmt.Sprintf("Frame buffer[%d] = RGB(%d,%d,%d)", pixelIndex, r, g, b),
		extraData)
}

// HookSDLTextureUpdate traces SDL texture updates
func HookSDLTextureUpdate(frame uint64, x, y int, originalRGB, sdlRGB uint32, pixelFormat string) {
	debuggerMutex.RLock()
	debugger := globalDebugger
	debuggerMutex.RUnlock()
	
	if debugger == nil {
		return
	}
	
	origR := (originalRGB >> 16) & 0xFF
	origG := (originalRGB >> 8) & 0xFF
	origB := originalRGB & 0xFF
	
	sdlR := (sdlRGB >> 16) & 0xFF
	sdlG := (sdlRGB >> 8) & 0xFF
	sdlB := sdlRGB & 0xFF
	
	extraData := map[string]interface{}{
		"pixel_format":    pixelFormat,
		"original_rgb":    fmt.Sprintf("RGB(%d,%d,%d)", origR, origG, origB),
		"sdl_rgb":         fmt.Sprintf("RGB(%d,%d,%d)", sdlR, sdlG, sdlB),
		"format_changed":  originalRGB != sdlRGB,
	}
	
	debugger.TraceColorTransformation(
		frame, -1, -1, x, y,
		StageSDLTextureUpdate,
		originalRGB, sdlRGB,
		fmt.Sprintf("SDL texture update: RGB(%d,%d,%d) -> RGB(%d,%d,%d)", origR, origG, origB, sdlR, sdlG, sdlB),
		extraData)
}

// HookSDLRender traces final SDL rendering
func HookSDLRender(frame uint64, renderMethod string, totalPixels int) {
	debuggerMutex.RLock()
	debugger := globalDebugger
	debuggerMutex.RUnlock()
	
	if debugger == nil {
		return
	}
	
	extraData := map[string]interface{}{
		"render_method": renderMethod,
		"total_pixels":  totalPixels,
	}
	
	debugger.TraceColorTransformation(
		frame, -1, -1, -1, -1,
		StageSDLRender,
		0, 0,
		fmt.Sprintf("SDL render complete: %s (%d pixels)", renderMethod, totalPixels),
		extraData)
}

// Utility functions for debugging specific color issues

// TraceColorIndex0x22 specifically traces the sky blue color (0x22) that's appearing brown
func TraceColorIndex0x22() {
	debuggerMutex.Lock()
	defer debuggerMutex.Unlock()
	
	if globalDebugger != nil {
		globalDebugger.SetTargetColor(0x22) // Sky blue color index
		globalDebugger.Enable()
	}
}

// TracePixelAt specifically traces a pixel at given coordinates
func TracePixelAt(x, y int) {
	debuggerMutex.Lock()
	defer debuggerMutex.Unlock()
	
	if globalDebugger != nil {
		globalDebugger.SetTargetPixel(x, y)
		globalDebugger.Enable()
	}
}

// DumpColorDebugReport generates and saves a comprehensive debug report
func DumpColorDebugReport() error {
	debuggerMutex.RLock()
	debugger := globalDebugger
	debuggerMutex.RUnlock()
	
	if debugger == nil {
		return fmt.Errorf("color debugger not initialized")
	}
	
	// Export events to file
	if err := debugger.ExportEventsToFile("color_pipeline_events.log"); err != nil {
		return fmt.Errorf("failed to export events: %v", err)
	}
	
	// Create comparison report
	if err := debugger.CreateColorComparisonReport(); err != nil {
		return fmt.Errorf("failed to create comparison report: %v", err)
	}
	
	// Analyze corruption
	analysis := debugger.AnalyzeColorCorruption()
	if analysis != nil {
		fmt.Printf("Color Corruption Analysis:\n")
		fmt.Printf("Total Events: %d\n", analysis.TotalEvents)
		fmt.Printf("Transformation Events: %d\n", analysis.TransformationEvents)
		fmt.Printf("Corruption by Stage:\n")
		for stage, count := range analysis.CorruptionStages {
			fmt.Printf("  %s: %d\n", stage, count)
		}
	}
	
	return nil
}