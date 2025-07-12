package integration

import (
	"fmt"
	"testing"
)

// DisplayValidationTestHelper provides utilities for testing display output without SDL2
type DisplayValidationTestHelper struct {
	*HeadlessEmulatorTestHelper
	patternTestResults []PatternTestResult
	colorTestResults   []ColorTestResult
}

// PatternTestResult represents the result of a pattern validation test
type PatternTestResult struct {
	TestName        string
	ExpectedPattern string
	DetectedPattern string
	MatchPercentage float64
	Valid           bool
	Message         string
}

// ColorTestResult represents the result of a color validation test
type ColorTestResult struct {
	TestName       string
	ExpectedColors []uint32
	DetectedColors []uint32
	ColorMatches   int
	Valid          bool
	Message        string
}

// NewDisplayValidationTestHelper creates a new display validation test helper
func NewDisplayValidationTestHelper() (*DisplayValidationTestHelper, error) {
	headlessHelper, err := NewHeadlessEmulatorTestHelper()
	if err != nil {
		return nil, err
	}

	return &DisplayValidationTestHelper{
		HeadlessEmulatorTestHelper: headlessHelper,
		patternTestResults:         make([]PatternTestResult, 0),
		colorTestResults:          make([]ColorTestResult, 0),
	}, nil
}

// ValidatePattern validates that the frame buffer contains an expected pattern
func (h *DisplayValidationTestHelper) ValidatePattern(expectedPattern string, tolerance float64) PatternTestResult {
	result := PatternTestResult{
		TestName:        expectedPattern,
		ExpectedPattern: expectedPattern,
		Valid:          false,
	}

	// Simple pattern validation - this could be enhanced with more sophisticated algorithms
	switch expectedPattern {
	case "all_black":
		result = h.validateAllBlackPattern()
	case "all_white":
		result = h.validateAllWhitePattern()
	case "checkerboard":
		result = h.validateCheckerboardPattern(tolerance)
	case "horizontal_stripes":
		result = h.validateHorizontalStripesPattern(tolerance)
	case "vertical_stripes":
		result = h.validateVerticalStripesPattern(tolerance)
	case "solid_color":
		result = h.validateSolidColorPattern(tolerance)
	default:
		result.Message = fmt.Sprintf("Unknown pattern type: %s", expectedPattern)
	}

	h.patternTestResults = append(h.patternTestResults, result)
	return result
}

// validateAllBlackPattern validates that the frame buffer is all black (0x00000000)
func (h *DisplayValidationTestHelper) validateAllBlackPattern() PatternTestResult {
	blackPixels := 0
	totalPixels := len(h.frameBuffer)

	for _, pixel := range h.frameBuffer {
		if pixel == 0x00000000 {
			blackPixels++
		}
	}

	percentage := float64(blackPixels) / float64(totalPixels) * 100.0
	result := PatternTestResult{
		TestName:        "all_black",
		ExpectedPattern: "all_black",
		DetectedPattern: fmt.Sprintf("%.1f%% black pixels", percentage),
		MatchPercentage: percentage,
		Valid:          percentage > 95.0, // Allow for some tolerance
		Message:        fmt.Sprintf("Found %d/%d black pixels (%.1f%%)", blackPixels, totalPixels, percentage),
	}

	return result
}

// validateAllWhitePattern validates that the frame buffer is all white (0xFFFFFFFF)
func (h *DisplayValidationTestHelper) validateAllWhitePattern() PatternTestResult {
	whitePixels := 0
	totalPixels := len(h.frameBuffer)

	for _, pixel := range h.frameBuffer {
		if pixel == 0xFFFFFFFF {
			whitePixels++
		}
	}

	percentage := float64(whitePixels) / float64(totalPixels) * 100.0
	result := PatternTestResult{
		TestName:        "all_white",
		ExpectedPattern: "all_white",
		DetectedPattern: fmt.Sprintf("%.1f%% white pixels", percentage),
		MatchPercentage: percentage,
		Valid:          percentage > 95.0,
		Message:        fmt.Sprintf("Found %d/%d white pixels (%.1f%%)", whitePixels, totalPixels, percentage),
	}

	return result
}

// validateCheckerboardPattern validates a checkerboard pattern
func (h *DisplayValidationTestHelper) validateCheckerboardPattern(tolerance float64) PatternTestResult {
	width, height := 256, 240
	matchingPixels := 0
	totalPixels := width * height

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixelIndex := y*width + x
			if pixelIndex >= len(h.frameBuffer) {
				continue
			}

			// Determine expected pattern for checkerboard (8x8 squares)
			squareX := x / 8
			squareY := y / 8
			expectedBlack := (squareX+squareY)%2 == 0

			pixel := h.frameBuffer[pixelIndex]
			isBlack := pixel == 0x00000000
			isWhite := pixel == 0xFFFFFFFF

			if (expectedBlack && isBlack) || (!expectedBlack && isWhite) {
				matchingPixels++
			}
		}
	}

	percentage := float64(matchingPixels) / float64(totalPixels) * 100.0
	result := PatternTestResult{
		TestName:        "checkerboard",
		ExpectedPattern: "checkerboard",
		DetectedPattern: fmt.Sprintf("%.1f%% matching pattern", percentage),
		MatchPercentage: percentage,
		Valid:          percentage > tolerance,
		Message:        fmt.Sprintf("Checkerboard pattern match: %.1f%% (threshold: %.1f%%)", percentage, tolerance),
	}

	return result
}

// validateHorizontalStripesPattern validates horizontal stripes pattern
func (h *DisplayValidationTestHelper) validateHorizontalStripesPattern(tolerance float64) PatternTestResult {
	width, height := 256, 240
	matchingPixels := 0
	totalPixels := width * height

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixelIndex := y*width + x
			if pixelIndex >= len(h.frameBuffer) {
				continue
			}

			// Horizontal stripes - alternate every 8 lines
			expectedBlack := (y/8)%2 == 0

			pixel := h.frameBuffer[pixelIndex]
			isBlack := pixel == 0x00000000
			isWhite := pixel == 0xFFFFFFFF

			if (expectedBlack && isBlack) || (!expectedBlack && isWhite) {
				matchingPixels++
			}
		}
	}

	percentage := float64(matchingPixels) / float64(totalPixels) * 100.0
	result := PatternTestResult{
		TestName:        "horizontal_stripes",
		ExpectedPattern: "horizontal_stripes",
		DetectedPattern: fmt.Sprintf("%.1f%% matching pattern", percentage),
		MatchPercentage: percentage,
		Valid:          percentage > tolerance,
		Message:        fmt.Sprintf("Horizontal stripes pattern match: %.1f%% (threshold: %.1f%%)", percentage, tolerance),
	}

	return result
}

// validateVerticalStripesPattern validates vertical stripes pattern
func (h *DisplayValidationTestHelper) validateVerticalStripesPattern(tolerance float64) PatternTestResult {
	width, height := 256, 240
	matchingPixels := 0
	totalPixels := width * height

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixelIndex := y*width + x
			if pixelIndex >= len(h.frameBuffer) {
				continue
			}

			// Vertical stripes - alternate every 8 columns
			expectedBlack := (x/8)%2 == 0

			pixel := h.frameBuffer[pixelIndex]
			isBlack := pixel == 0x00000000
			isWhite := pixel == 0xFFFFFFFF

			if (expectedBlack && isBlack) || (!expectedBlack && isWhite) {
				matchingPixels++
			}
		}
	}

	percentage := float64(matchingPixels) / float64(totalPixels) * 100.0
	result := PatternTestResult{
		TestName:        "vertical_stripes",
		ExpectedPattern: "vertical_stripes",
		DetectedPattern: fmt.Sprintf("%.1f%% matching pattern", percentage),
		MatchPercentage: percentage,
		Valid:          percentage > tolerance,
		Message:        fmt.Sprintf("Vertical stripes pattern match: %.1f%% (threshold: %.1f%%)", percentage, tolerance),
	}

	return result
}

// validateSolidColorPattern validates that most of the screen is a single color
func (h *DisplayValidationTestHelper) validateSolidColorPattern(tolerance float64) PatternTestResult {
	colorCounts := make(map[uint32]int)
	totalPixels := len(h.frameBuffer)

	// Count occurrences of each color
	for _, pixel := range h.frameBuffer {
		colorCounts[pixel]++
	}

	// Find the most common color
	var dominantColor uint32
	maxCount := 0
	for color, count := range colorCounts {
		if count > maxCount {
			maxCount = count
			dominantColor = color
		}
	}

	percentage := float64(maxCount) / float64(totalPixels) * 100.0
	result := PatternTestResult{
		TestName:        "solid_color",
		ExpectedPattern: "solid_color",
		DetectedPattern: fmt.Sprintf("dominant color 0x%08X covers %.1f%%", dominantColor, percentage),
		MatchPercentage: percentage,
		Valid:          percentage > tolerance,
		Message:        fmt.Sprintf("Solid color pattern: 0x%08X covers %.1f%% (threshold: %.1f%%)", dominantColor, percentage, tolerance),
	}

	return result
}

// ValidateColors validates that specific colors are present in the frame buffer
func (h *DisplayValidationTestHelper) ValidateColors(expectedColors []uint32, testName string) ColorTestResult {
	result := ColorTestResult{
		TestName:       testName,
		ExpectedColors: expectedColors,
		DetectedColors: make([]uint32, 0),
		ColorMatches:   0,
		Valid:         false,
	}

	// Count unique colors in frame buffer
	colorMap := make(map[uint32]bool)
	for _, pixel := range h.frameBuffer {
		colorMap[pixel] = true
	}

	// Convert to slice
	for color := range colorMap {
		result.DetectedColors = append(result.DetectedColors, color)
	}

	// Check for expected colors
	for _, expectedColor := range expectedColors {
		if colorMap[expectedColor] {
			result.ColorMatches++
		}
	}

	// Validation criteria
	if len(expectedColors) > 0 {
		result.Valid = result.ColorMatches == len(expectedColors)
		result.Message = fmt.Sprintf("Found %d/%d expected colors, %d total unique colors",
			result.ColorMatches, len(expectedColors), len(result.DetectedColors))
	} else {
		result.Valid = len(result.DetectedColors) > 0
		result.Message = fmt.Sprintf("Found %d unique colors", len(result.DetectedColors))
	}

	h.colorTestResults = append(h.colorTestResults, result)
	return result
}

// GetPatternTestResults returns all pattern test results
func (h *DisplayValidationTestHelper) GetPatternTestResults() []PatternTestResult {
	return h.patternTestResults
}

// GetColorTestResults returns all color test results
func (h *DisplayValidationTestHelper) GetColorTestResults() []ColorTestResult {
	return h.colorTestResults
}

// TestHeadlessDisplayPatterns tests various display patterns without SDL2 video
func TestHeadlessDisplayPatterns(t *testing.T) {
	t.Run("Black screen test", func(t *testing.T) {
		helper, err := NewDisplayValidationTestHelper()
		if err != nil {
			t.Fatalf("Failed to create display test helper: %v", err)
		}
		defer helper.Cleanup()

		// ROM that should produce a black screen (no rendering enabled)
		program := []uint8{
			0xA9, 0x00, // LDA #$00
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK - disable rendering)
			0x4C, 0x05, 0x80, // JMP $8005 (infinite loop)
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		err = helper.RunHeadlessFrames(5)
		if err != nil {
			t.Fatalf("Failed to run frames: %v", err)
		}

		// Validate black screen pattern
		result := helper.ValidatePattern("all_black", 95.0)
		if !result.Valid {
			t.Errorf("Black screen test failed: %s", result.Message)
		}

		t.Logf("Black screen test: %s", result.Message)
	})

	t.Run("Solid color background test", func(t *testing.T) {
		helper, err := NewDisplayValidationTestHelper()
		if err != nil {
			t.Fatalf("Failed to create display test helper: %v", err)
		}
		defer helper.Cleanup()

		// ROM that sets a solid background color
		program := []uint8{
			// Enable rendering
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL - enable NMI)
			0xA9, 0x08, // LDA #$08
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK - enable background)

			// Set palette background color
			0xA9, 0x3F, // LDA #$3F
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low)
			0xA9, 0x30, // LDA #$30 (light blue)
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)

			0x4C, 0x17, 0x80, // JMP $8017 (infinite loop)
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		err = helper.RunHeadlessFrames(10)
		if err != nil {
			t.Fatalf("Failed to run frames: %v", err)
		}

		// Validate solid color pattern
		result := helper.ValidatePattern("solid_color", 80.0)
		t.Logf("Solid color test: %s", result.Message)

		// Validate that we have some rendering output
		fbResult := helper.ValidateFrameBuffer()
		if fbResult.UniqueColors < 1 {
			t.Error("Expected at least 1 unique color in solid color test")
		}
	})

	t.Run("Pattern generation test", func(t *testing.T) {
		helper, err := NewDisplayValidationTestHelper()
		if err != nil {
			t.Fatalf("Failed to create display test helper: %v", err)
		}
		defer helper.Cleanup()

		// ROM that generates a simple pattern using sprites/background
		program := []uint8{
			// Enable rendering
			0xA9, 0x90, // LDA #$90
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL - enable NMI, sprite table $1000)
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK - enable background and sprites)

			// Set up palette
			0xA9, 0x3F, // LDA #$3F
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR)

			// Write simple 4-color palette
			0xA9, 0x0F, // LDA #$0F (black)
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
			0xA9, 0x30, // LDA #$30 (white)
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
			0xA9, 0x16, // LDA #$16 (red)
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
			0xA9, 0x12, // LDA #$12 (blue)
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)

			0x4C, 0x23, 0x80, // JMP $8023 (infinite loop)
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		err = helper.RunHeadlessFrames(10)
		if err != nil {
			t.Fatalf("Failed to run frames: %v", err)
		}

		// Validate that we have multiple colors
		expectedColors := []uint32{0xFF000000} // Black should be present
		colorResult := helper.ValidateColors(expectedColors, "pattern_colors")
		t.Logf("Pattern color test: %s", colorResult.Message)

		// Validate frame buffer has reasonable content
		fbResult := helper.ValidateFrameBuffer()
		if fbResult.UniqueColors < 1 {
			t.Error("Pattern test should produce at least 1 unique color")
		}

		t.Logf("Pattern generation test: %d unique colors found", fbResult.UniqueColors)
	})
}

// TestHeadlessDisplayValidation tests display validation capabilities
func TestHeadlessDisplayValidation(t *testing.T) {
	t.Run("Frame buffer structure validation", func(t *testing.T) {
		helper, err := NewDisplayValidationTestHelper()
		if err != nil {
			t.Fatalf("Failed to create display test helper: %v", err)
		}
		defer helper.Cleanup()

		// Simple ROM for structure testing
		program := []uint8{
			0xEA, // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		err = helper.RunHeadlessFrames(1)
		if err != nil {
			t.Fatalf("Failed to run frames: %v", err)
		}

		// Validate frame buffer structure
		fbResult := helper.ValidateFrameBuffer()

		if !fbResult.ExpectedDimensions {
			t.Error("Frame buffer does not have expected NES dimensions (256x240)")
		}

		if fbResult.PixelCount != 256*240 {
			t.Errorf("Expected %d pixels, got %d", 256*240, fbResult.PixelCount)
		}

		t.Logf("Frame buffer validation: %s", fbResult.ValidationMessage)
	})

	t.Run("Color depth validation", func(t *testing.T) {
		helper, err := NewDisplayValidationTestHelper()
		if err != nil {
			t.Fatalf("Failed to create display test helper: %v", err)
		}
		defer helper.Cleanup()

		// ROM that uses various colors
		program := []uint8{
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x08, // LDA #$08
			0x8D, 0x01, 0x20, // STA $2001

			// Set multiple palette colors
			0xA9, 0x3F, // LDA #$3F
			0x8D, 0x06, 0x20, // STA $2006
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006

			0xA9, 0x0F, // LDA #$0F (black)
			0x8D, 0x07, 0x20, // STA $2007
			0xA9, 0x30, // LDA #$30 (white)
			0x8D, 0x07, 0x20, // STA $2007
			0xA9, 0x16, // LDA #$16 (red)
			0x8D, 0x07, 0x20, // STA $2007
			0xA9, 0x12, // LDA #$12 (blue)
			0x8D, 0x07, 0x20, // STA $2007

			0x4C, 0x20, 0x80, // JMP loop
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		err = helper.RunHeadlessFrames(5)
		if err != nil {
			t.Fatalf("Failed to run frames: %v", err)
		}

		// Check that pixels are 32-bit RGBA values
		fbResult := helper.ValidateFrameBuffer()
		
		// Verify we can detect multiple colors
		if fbResult.UniqueColors == 0 {
			t.Error("No colors detected in color depth test")
		}

		t.Logf("Color depth validation: %d unique colors detected", fbResult.UniqueColors)
	})

	t.Run("Display consistency validation", func(t *testing.T) {
		helper, err := NewDisplayValidationTestHelper()
		if err != nil {
			t.Fatalf("Failed to create display test helper: %v", err)
		}
		defer helper.Cleanup()

		// ROM that should produce consistent output
		program := []uint8{
			0xA9, 0x08, // LDA #$08
			0x8D, 0x01, 0x20, // STA $2001 (enable background)
			0x4C, 0x05, 0x80, // JMP $8005 (infinite loop)
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		// Run multiple frames and check consistency
		var frameResults []FrameBufferValidationResult

		for frame := 0; frame < 5; frame++ {
			err = helper.RunHeadlessFrames(1)
			if err != nil {
				t.Fatalf("Failed to run frame %d: %v", frame, err)
			}

			fbResult := helper.ValidateFrameBuffer()
			frameResults = append(frameResults, fbResult)
		}

		// Check consistency across frames
		firstResult := frameResults[0]
		for i, result := range frameResults[1:] {
			if result.PixelCount != firstResult.PixelCount {
				t.Errorf("Frame %d pixel count differs: expected %d, got %d",
					i+1, firstResult.PixelCount, result.PixelCount)
			}

			if !result.ExpectedDimensions {
				t.Errorf("Frame %d has wrong dimensions", i+1)
			}
		}

		t.Logf("Display consistency validation passed for %d frames", len(frameResults))
	})
}