// Package debug provides frame buffer dumping utilities
package debug

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FrameDumper provides utilities for dumping frame buffer contents
type FrameDumper struct {
	outputDir      string
	dumpEnabled    bool
	frameCount     uint64
	maxDumps       int
	dumpInterval   int // Dump every N frames
	pixelFilter    func(x, y int, rgb uint32) bool
}

// NewFrameDumper creates a new frame dumper
func NewFrameDumper(outputDir string) *FrameDumper {
	return &FrameDumper{
		outputDir:    outputDir,
		dumpEnabled:  false,
		maxDumps:     10,
		dumpInterval: 1,
		pixelFilter:  nil,
	}
}

// Enable activates frame dumping
func (fd *FrameDumper) Enable() {
	fd.dumpEnabled = true
	os.MkdirAll(fd.outputDir, 0755)
}

// Disable deactivates frame dumping
func (fd *FrameDumper) Disable() {
	fd.dumpEnabled = false
}

// SetMaxDumps sets the maximum number of frames to dump
func (fd *FrameDumper) SetMaxDumps(max int) {
	fd.maxDumps = max
}

// SetDumpInterval sets the interval between frame dumps
func (fd *FrameDumper) SetDumpInterval(interval int) {
	fd.dumpInterval = interval
}

// SetPixelFilter sets a filter function for which pixels to include in dumps
func (fd *FrameDumper) SetPixelFilter(filter func(x, y int, rgb uint32) bool) {
	fd.pixelFilter = filter
}

// DumpFrameBuffer dumps a complete frame buffer to a text file
func (fd *FrameDumper) DumpFrameBuffer(frameBuffer [256 * 240]uint32, frameNum uint64) error {
	if !fd.dumpEnabled {
		return nil
	}

	// Check if we should dump this frame
	if frameNum%uint64(fd.dumpInterval) != 0 {
		return nil
	}

	// Check if we've exceeded max dumps
	if fd.frameCount >= uint64(fd.maxDumps) {
		return nil
	}

	filename := fmt.Sprintf("frame_%06d_%s.txt", frameNum, time.Now().Format("150405"))
	filePath := filepath.Join(fd.outputDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create frame dump file: %v", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "Frame Buffer Dump\n")
	fmt.Fprintf(file, "Frame Number: %d\n", frameNum)
	fmt.Fprintf(file, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "Dimensions: 256x240\n")
	fmt.Fprintf(file, "===================\n\n")

	// Write pixel data
	for y := 0; y < 240; y++ {
		fmt.Fprintf(file, "Line %03d: ", y)
		for x := 0; x < 256; x++ {
			pixel := frameBuffer[y*256+x]
			
			// Apply pixel filter if set
			if fd.pixelFilter != nil && !fd.pixelFilter(x, y, pixel) {
				continue
			}
			
			// Write pixel in hex format
			if x%16 == 0 && x > 0 {
				fmt.Fprintf(file, "\n          ")
			}
			fmt.Fprintf(file, "%06X ", pixel)
		}
		fmt.Fprintf(file, "\n")
	}

	fd.frameCount++
	return nil
}

// DumpFrameBufferRGB dumps a frame buffer with RGB component breakdown
func (fd *FrameDumper) DumpFrameBufferRGB(frameBuffer [256 * 240]uint32, frameNum uint64) error {
	if !fd.dumpEnabled {
		return nil
	}

	filename := fmt.Sprintf("frame_rgb_%06d_%s.txt", frameNum, time.Now().Format("150405"))
	filePath := filepath.Join(fd.outputDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create RGB frame dump file: %v", err)
	}
	defer file.Close()

	// Write header
	fmt.Fprintf(file, "Frame Buffer RGB Dump\n")
	fmt.Fprintf(file, "Frame Number: %d\n", frameNum)
	fmt.Fprintf(file, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "Format: X,Y: RGB(r,g,b) #RRGGBB\n")
	fmt.Fprintf(file, "========================\n\n")

	// Color frequency analysis
	colorFreq := make(map[uint32]int)
	
	// Write pixel data with RGB breakdown
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			pixel := frameBuffer[y*256+x]
			
			// Apply pixel filter if set
			if fd.pixelFilter != nil && !fd.pixelFilter(x, y, pixel) {
				continue
			}
			
			r := (pixel >> 16) & 0xFF
			g := (pixel >> 8) & 0xFF
			b := pixel & 0xFF
			
			colorFreq[pixel]++
			
			// Only log interesting pixels (non-black or specifically filtered)
			if pixel != 0 || (fd.pixelFilter != nil) {
				fmt.Fprintf(file, "%3d,%3d: RGB(%3d,%3d,%3d) #%06X\n", x, y, r, g, b, pixel)
			}
		}
	}

	// Add color frequency analysis
	fmt.Fprintf(file, "\nColor Frequency Analysis:\n")
	fmt.Fprintf(file, "Color      | Count | Percentage\n")
	fmt.Fprintf(file, "-----------|-------|----------\n")
	
	totalPixels := 256 * 240
	for color, count := range colorFreq {
		percentage := float64(count) / float64(totalPixels) * 100
		r := (color >> 16) & 0xFF
		g := (color >> 8) & 0xFF
		b := color & 0xFF
		fmt.Fprintf(file, "#%06X | %5d | %6.2f%%  RGB(%3d,%3d,%3d)\n", 
			color, count, percentage, r, g, b)
	}

	return nil
}

// DumpColorCorruption specifically dumps pixels that appear to be corrupted
func (fd *FrameDumper) DumpColorCorruption(frameBuffer [256 * 240]uint32, frameNum uint64) error {
	if !fd.dumpEnabled {
		return nil
	}

	filename := fmt.Sprintf("color_corruption_%06d_%s.txt", frameNum, time.Now().Format("150405"))
	filePath := filepath.Join(fd.outputDir, filename)

	file, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create corruption dump file: %v", err)
	}
	defer file.Close()

	// Expected colors for common NES palette indices
	expectedColors := map[uint32]string{
		0x64B0FF: "Sky Blue (0x22)",
		0xFFFEFF: "White (0x30)",
		0x666666: "Gray (0x00)",
		0x000000: "Black (0x0F)",
	}

	// Common corruption patterns
	corruptionPatterns := map[uint32]string{
		0x8B4513: "Brown (corrupted blue?)",
		0xFF0000: "Red (emphasis bug?)",
		0x654321: "Dark Brown",
	}

	fmt.Fprintf(file, "Color Corruption Analysis\n")
	fmt.Fprintf(file, "Frame Number: %d\n", frameNum)
	fmt.Fprintf(file, "Timestamp: %s\n", time.Now().Format(time.RFC3339))
	fmt.Fprintf(file, "=========================\n\n")

	suspiciousPixels := 0
	
	// Scan for suspicious colors
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			pixel := frameBuffer[y*256+x]
			
			// Check if this is a known corruption pattern
			if description, isCorrupted := corruptionPatterns[pixel]; isCorrupted {
				r := (pixel >> 16) & 0xFF
				g := (pixel >> 8) & 0xFF
				b := pixel & 0xFF
				
				fmt.Fprintf(file, "CORRUPTION at %3d,%3d: #%06X RGB(%3d,%3d,%3d) - %s\n", 
					x, y, pixel, r, g, b, description)
				suspiciousPixels++
			}
		}
	}

	// Add summary
	fmt.Fprintf(file, "\nSummary:\n")
	fmt.Fprintf(file, "Suspicious pixels found: %d\n", suspiciousPixels)
	fmt.Fprintf(file, "Corruption rate: %.2f%%\n", 
		float64(suspiciousPixels)/float64(256*240)*100)

	// Add expected vs actual color analysis
	fmt.Fprintf(file, "\nExpected Color Analysis:\n")
	for expectedColor, description := range expectedColors {
		count := 0
		for _, pixel := range frameBuffer {
			if pixel == expectedColor {
				count++
			}
		}
		fmt.Fprintf(file, "%s: %d pixels (%.2f%%)\n", 
			description, count, float64(count)/float64(256*240)*100)
	}

	return nil
}

// CreateSkyBluePixelFilter creates a filter for sky blue pixels (0x22 color index)
func CreateSkyBluePixelFilter() func(x, y int, rgb uint32) bool {
	return func(x, y int, rgb uint32) bool {
		return rgb == 0x64B0FF // Expected sky blue RGB
	}
}

// CreateRegionFilter creates a filter for a specific rectangular region
func CreateRegionFilter(x1, y1, x2, y2 int) func(x, y int, rgb uint32) bool {
	return func(x, y int, rgb uint32) bool {
		return x >= x1 && x <= x2 && y >= y1 && y <= y2
	}
}

// CreateColorRangeFilter creates a filter for colors within a certain range
func CreateColorRangeFilter(minRGB, maxRGB uint32) func(x, y int, rgb uint32) bool {
	return func(x, y int, rgb uint32) bool {
		return rgb >= minRGB && rgb <= maxRGB
	}
}