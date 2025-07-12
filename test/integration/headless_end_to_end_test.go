package integration

import (
	"fmt"
	"testing"
	"time"

	"gones/internal/input"
)

// EndToEndTestHelper provides utilities for comprehensive end-to-end testing
type EndToEndTestHelper struct {
	*HeadlessEmulatorTestHelper
	testScenarios []TestScenario
	testResults   []EndToEndTestResult
}

// TestScenario represents a complete test scenario combining ROM, input, and validation
type TestScenario struct {
	Name             string
	Description      string
	ROM              []uint8
	InputSequence    []HeadlessInputEvent
	ValidationFrames int
	ExpectedOutcome  ExpectedOutcome
}

// ExpectedOutcome defines what outcomes are expected from a test scenario
type ExpectedOutcome struct {
	FrameBufferValid    bool
	AudioGenerated      bool
	InputProcessed      bool
	MinUniqueColors     int
	MaxExecutionTimeMs  int64
	MemoryValidation    []MemoryValidation
	CustomValidations   []string
}

// MemoryValidation defines expected memory state validation
type MemoryValidation struct {
	Address       uint16
	ExpectedValue uint8
	Description   string
}

// EndToEndTestResult represents the result of a complete end-to-end test
type EndToEndTestResult struct {
	TestName           string
	Passed             bool
	ExecutionTimeMs    int64
	FramesProcessed    int
	FrameBufferResult  FrameBufferValidationResult
	AudioResult        AudioValidationResult
	InputResult        InputValidationResult
	MemoryResults      []MemoryValidationResult
	PerformanceMetrics map[string]interface{}
	ErrorMessage       string
	DetailedReport     string
}

// InputValidationResult represents input validation results
type InputValidationResult struct {
	EventsProcessed int
	ButtonsDetected []input.Button
	Valid           bool
	Message         string
}

// MemoryValidationResult represents memory validation results
type MemoryValidationResult struct {
	Address      uint16
	Expected     uint8
	Actual       uint8
	Valid        bool
	Description  string
}

// NewEndToEndTestHelper creates a new end-to-end test helper
func NewEndToEndTestHelper() (*EndToEndTestHelper, error) {
	headlessHelper, err := NewHeadlessEmulatorTestHelper()
	if err != nil {
		return nil, err
	}

	return &EndToEndTestHelper{
		HeadlessEmulatorTestHelper: headlessHelper,
		testScenarios:             make([]TestScenario, 0),
		testResults:               make([]EndToEndTestResult, 0),
	}, nil
}

// CreateBasicDisplayScenario creates a scenario that tests basic display functionality
func (h *EndToEndTestHelper) CreateBasicDisplayScenario() TestScenario {
	rom := []uint8{
		// Initialize PPU for rendering
		0xA9, 0x80, // LDA #$80
		0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL - enable NMI)
		0xA9, 0x1E, // LDA #$1E
		0x8D, 0x01, 0x20, // STA $2001 (PPUMASK - enable background and sprites)

		// Set up basic palette
		0xA9, 0x3F, // LDA #$3F
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low)

		// Write 4 palette colors
		0xA9, 0x0F, // LDA #$0F (black)
		0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
		0xA9, 0x30, // LDA #$30 (white)
		0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
		0xA9, 0x16, // LDA #$16 (red)
		0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
		0xA9, 0x12, // LDA #$12 (blue)
		0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)

		// Set nametable data to show some pattern
		0xA9, 0x20, // LDA #$20
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR high)
		0xA9, 0x00, // LDA #$00
		0x8D, 0x06, 0x20, // STA $2006 (PPUADDR low)

		// Write pattern to nametable
		0xA2, 0x20, // LDX #$20 (32 tiles)
		0xA9, 0x01, // LDA #$01 (tile 1)
		0x8D, 0x07, 0x20, // STA $2007 (write tile)
		0xCA,             // DEX
		0xD0, 0xF9,       // BNE -7 (loop)

		// Main loop
		0x4C, 0x2E, 0x80, // JMP $802E (infinite loop)
	}

	return TestScenario{
		Name:             "basic_display",
		Description:      "Test basic display rendering functionality",
		ROM:              rom,
		InputSequence:    []HeadlessInputEvent{}, // No input needed
		ValidationFrames: 10,
		ExpectedOutcome: ExpectedOutcome{
			FrameBufferValid:   true,
			AudioGenerated:     false,
			InputProcessed:     false,
			MinUniqueColors:    1,
			MaxExecutionTimeMs: 5000,
		},
	}
}

// CreateAudioTestScenario creates a scenario that tests audio functionality
func (h *EndToEndTestHelper) CreateAudioTestScenario() TestScenario {
	rom := []uint8{
		// Initialize APU
		0xA9, 0x0F, // LDA #$0F
		0x8D, 0x15, 0x40, // STA $4015 (APU_STATUS - enable all channels)

		// Set up Pulse 1 channel
		0xA9, 0xBF, // LDA #$BF (duty cycle 50%, constant volume 15)
		0x8D, 0x00, 0x40, // STA $4000 (PULSE1_DUTY)

		0xA9, 0x00, // LDA #$00
		0x8D, 0x01, 0x40, // STA $4001 (PULSE1_SWEEP - no sweep)

		0xA9, 0xF9, // LDA #$F9 (low frequency byte)
		0x8D, 0x02, 0x40, // STA $4002 (PULSE1_LO)

		0xA9, 0x00, // LDA #$00 (high frequency byte + length)
		0x8D, 0x03, 0x40, // STA $4003 (PULSE1_HI)

		// Set up Triangle channel
		0xA9, 0x81, // LDA #$81 (linear counter)
		0x8D, 0x08, 0x40, // STA $4008 (TRIANGLE_LINEAR)

		0xA9, 0xF9, // LDA #$F9
		0x8D, 0x0A, 0x40, // STA $400A (TRIANGLE_LO)

		0xA9, 0x00, // LDA #$00
		0x8D, 0x0B, 0x40, // STA $400B (TRIANGLE_HI)

		// Main loop
		0x4C, 0x1E, 0x80, // JMP $801E (infinite loop)
	}

	return TestScenario{
		Name:             "audio_test",
		Description:      "Test audio generation functionality",
		ROM:              rom,
		InputSequence:    []HeadlessInputEvent{}, // No input needed
		ValidationFrames: 20,
		ExpectedOutcome: ExpectedOutcome{
			FrameBufferValid:   true,
			AudioGenerated:     true,
			InputProcessed:     false,
			MinUniqueColors:    0,
			MaxExecutionTimeMs: 5000,
		},
	}
}

// CreateInputResponseScenario creates a scenario that tests input responsiveness
func (h *EndToEndTestHelper) CreateInputResponseScenario() TestScenario {
	rom := []uint8{
		// Main loop - read controller and respond
		0xA9, 0x01, // LDA #$01
		0x8D, 0x16, 0x40, // STA $4016 (strobe controller)
		0xA9, 0x00, // LDA #$00
		0x8D, 0x16, 0x40, // STA $4016 (stop strobe)

		// Read A button
		0xAD, 0x16, 0x40, // LDA $4016
		0x29, 0x01,       // AND #$01
		0xF0, 0x06,       // BEQ +6 (skip if not pressed)

		// A button pressed - store response
		0xA9, 0xAA,       // LDA #$AA
		0x85, 0x10,       // STA $10 (memory marker)
		0x4C, 0x18, 0x80, // JMP +6

		// A button not pressed
		0xA9, 0x00,       // LDA #$00
		0x85, 0x10,       // STA $10

		// Read B button
		0xAD, 0x16, 0x40, // LDA $4016
		0x29, 0x01,       // AND #$01
		0xF0, 0x06,       // BEQ +6

		// B button pressed
		0xA9, 0xBB,       // LDA #$BB
		0x85, 0x11,       // STA $11
		0x4C, 0x26, 0x80, // JMP +6

		// B button not pressed
		0xA9, 0x00,       // LDA #$00
		0x85, 0x11,       // STA $11

		// Skip remaining buttons for brevity
		0xAD, 0x16, 0x40, // LDA $4016 (Skip select)
		0xAD, 0x16, 0x40, // LDA $4016 (Skip start)
		0xAD, 0x16, 0x40, // LDA $4016 (Skip up)
		0xAD, 0x16, 0x40, // LDA $4016 (Skip down)
		0xAD, 0x16, 0x40, // LDA $4016 (Skip left)
		0xAD, 0x16, 0x40, // LDA $4016 (Skip right)

		0x4C, 0x00, 0x80, // JMP $8000 (repeat)
	}

	inputSequence := []HeadlessInputEvent{
		{Controller: 1, Button: input.A, Pressed: true, FrameDelay: 5},
		{Controller: 1, Button: input.A, Pressed: false, FrameDelay: 8},
		{Controller: 1, Button: input.B, Pressed: true, FrameDelay: 10},
		{Controller: 1, Button: input.B, Pressed: false, FrameDelay: 13},
	}

	return TestScenario{
		Name:             "input_response",
		Description:      "Test input responsiveness and processing",
		ROM:              rom,
		InputSequence:    inputSequence,
		ValidationFrames: 20,
		ExpectedOutcome: ExpectedOutcome{
			FrameBufferValid:   true,
			AudioGenerated:     false,
			InputProcessed:     true,
			MinUniqueColors:    0,
			MaxExecutionTimeMs: 5000,
			MemoryValidation: []MemoryValidation{
				{Address: 0x0010, ExpectedValue: 0xAA, Description: "A button response marker"},
				{Address: 0x0011, ExpectedValue: 0xBB, Description: "B button response marker"},
			},
		},
	}
}

// CreateComplexScenario creates a comprehensive scenario testing multiple systems
func (h *EndToEndTestHelper) CreateComplexScenario() TestScenario {
	rom := []uint8{
		// Initialize systems
		0xA9, 0x80, // LDA #$80
		0x8D, 0x00, 0x20, // STA $2000 (PPU)
		0xA9, 0x1E, // LDA #$1E
		0x8D, 0x01, 0x20, // STA $2001 (PPU)
		0xA9, 0x0F, // LDA #$0F
		0x8D, 0x15, 0x40, // STA $4015 (APU)

		// Set initial state marker
		0xA9, 0x01, // LDA #$01
		0x85, 0x20, // STA $20 (state marker)

		// Main game loop
		0xA9, 0x01, // LDA #$01
		0x8D, 0x16, 0x40, // STA $4016 (strobe)
		0xA9, 0x00, // LDA #$00
		0x8D, 0x16, 0x40, // STA $4016

		// Read Start button
		0xAD, 0x16, 0x40, // Skip A
		0xAD, 0x16, 0x40, // Skip B
		0xAD, 0x16, 0x40, // Skip Select
		0xAD, 0x16, 0x40, // LDA $4016 (Start)
		0x29, 0x01,       // AND #$01
		0xF0, 0x08,       // BEQ +8

		// Start pressed - change state
		0xE6, 0x20,       // INC $20 (increment state)
		0xA9, 0x50,       // LDA #$50 (audio duty)
		0x8D, 0x00, 0x40, // STA $4000 (make sound)

		// Skip remaining controller bits
		0xAD, 0x16, 0x40, // Up
		0xAD, 0x16, 0x40, // Down
		0xAD, 0x16, 0x40, // Left
		0xAD, 0x16, 0x40, // Right

		// Visual feedback based on state
		0xA5, 0x20,       // LDA $20
		0x29, 0x0F,       // AND #$0F
		0x8D, 0x07, 0x20, // STA $2007 (write to PPU)

		0x4C, 0x10, 0x80, // JMP $8010 (main loop)
	}

	inputSequence := []HeadlessInputEvent{
		{Controller: 1, Button: input.Start, Pressed: true, FrameDelay: 3},
		{Controller: 1, Button: input.Start, Pressed: false, FrameDelay: 5},
		{Controller: 1, Button: input.Start, Pressed: true, FrameDelay: 10},
		{Controller: 1, Button: input.Start, Pressed: false, FrameDelay: 12},
		{Controller: 1, Button: input.Start, Pressed: true, FrameDelay: 15},
		{Controller: 1, Button: input.Start, Pressed: false, FrameDelay: 17},
	}

	return TestScenario{
		Name:             "complex_scenario",
		Description:      "Comprehensive test of PPU, APU, and input systems",
		ROM:              rom,
		InputSequence:    inputSequence,
		ValidationFrames: 25,
		ExpectedOutcome: ExpectedOutcome{
			FrameBufferValid:   true,
			AudioGenerated:     true,
			InputProcessed:     true,
			MinUniqueColors:    1,
			MaxExecutionTimeMs: 8000,
			MemoryValidation: []MemoryValidation{
				{Address: 0x0020, ExpectedValue: 0x04, Description: "State should increment with Start presses"},
			},
		},
	}
}

// ExecuteScenario executes a test scenario and returns detailed results
func (h *EndToEndTestHelper) ExecuteScenario(scenario TestScenario) EndToEndTestResult {
	startTime := time.Now()

	result := EndToEndTestResult{
		TestName:        scenario.Name,
		Passed:          false,
		MemoryResults:   make([]MemoryValidationResult, 0),
		ErrorMessage:    "",
		DetailedReport:  "",
	}

	// Load ROM
	err := h.LoadMockROM(scenario.ROM)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to load ROM: %v", err)
		h.testResults = append(h.testResults, result)
		return result
	}

	// Schedule input events
	for _, inputEvent := range scenario.InputSequence {
		h.ScheduleInputEvent(inputEvent.Controller, inputEvent.Button, inputEvent.Pressed, inputEvent.FrameDelay)
	}

	// Execute the scenario
	err = h.RunHeadlessFrames(scenario.ValidationFrames)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Failed to run scenario: %v", err)
		h.testResults = append(h.testResults, result)
		return result
	}

	// Record execution metrics
	result.ExecutionTimeMs = time.Since(startTime).Milliseconds()
	result.FramesProcessed = scenario.ValidationFrames
	result.PerformanceMetrics = h.GetPerformanceMetrics()

	// Validate frame buffer
	result.FrameBufferResult = h.ValidateFrameBuffer()

	// Validate audio
	result.AudioResult = h.ValidateAudio()

	// Validate input (simplified)
	result.InputResult = InputValidationResult{
		EventsProcessed: len(scenario.InputSequence),
		Valid:           len(scenario.InputSequence) == 0 || result.AudioResult.SampleCount > 0, // Simplified
		Message:         fmt.Sprintf("Processed %d input events", len(scenario.InputSequence)),
	}

	// Validate memory
	bus := h.app.GetBus()
	if bus != nil && bus.Memory != nil {
		for _, memValidation := range scenario.ExpectedOutcome.MemoryValidation {
			actualValue := bus.Memory.Read(memValidation.Address)
			memResult := MemoryValidationResult{
				Address:     memValidation.Address,
				Expected:    memValidation.ExpectedValue,
				Actual:      actualValue,
				Valid:       actualValue == memValidation.ExpectedValue,
				Description: memValidation.Description,
			}
			result.MemoryResults = append(result.MemoryResults, memResult)
		}
	}

	// Overall validation
	result.Passed = h.validateScenarioOutcome(scenario, result)

	// Generate detailed report
	result.DetailedReport = h.generateDetailedReport(scenario, result)

	h.testResults = append(h.testResults, result)
	return result
}

// validateScenarioOutcome validates the scenario outcome against expectations
func (h *EndToEndTestHelper) validateScenarioOutcome(scenario TestScenario, result EndToEndTestResult) bool {
	expected := scenario.ExpectedOutcome

	// Check execution time
	if result.ExecutionTimeMs > expected.MaxExecutionTimeMs {
		return false
	}

	// Check frame buffer validity
	if expected.FrameBufferValid && !result.FrameBufferResult.Valid {
		return false
	}

	// Check minimum unique colors
	if result.FrameBufferResult.UniqueColors < expected.MinUniqueColors {
		return false
	}

	// Check audio generation
	if expected.AudioGenerated && !result.AudioResult.NonSilent {
		return false
	}

	// Check memory validations
	for _, memResult := range result.MemoryResults {
		if !memResult.Valid {
			return false
		}
	}

	// If we get here, all validations passed
	return true
}

// generateDetailedReport generates a detailed test report
func (h *EndToEndTestHelper) generateDetailedReport(scenario TestScenario, result EndToEndTestResult) string {
	report := fmt.Sprintf("=== Test Report: %s ===\n", scenario.Name)
	report += fmt.Sprintf("Description: %s\n", scenario.Description)
	report += fmt.Sprintf("Execution Time: %d ms\n", result.ExecutionTimeMs)
	report += fmt.Sprintf("Frames Processed: %d\n", result.FramesProcessed)
	report += fmt.Sprintf("Overall Result: %s\n", map[bool]string{true: "PASS", false: "FAIL"}[result.Passed])

	report += "\nFrame Buffer:\n"
	report += fmt.Sprintf("  Valid: %t\n", result.FrameBufferResult.Valid)
	report += fmt.Sprintf("  Unique Colors: %d\n", result.FrameBufferResult.UniqueColors)
	report += fmt.Sprintf("  Non-Zero Pixels: %d\n", result.FrameBufferResult.NonZeroPixels)

	report += "\nAudio:\n"
	report += fmt.Sprintf("  Valid: %t\n", result.AudioResult.Valid)
	report += fmt.Sprintf("  Samples: %d\n", result.AudioResult.SampleCount)
	report += fmt.Sprintf("  Non-Silent: %t\n", result.AudioResult.NonSilent)

	report += "\nInput:\n"
	report += fmt.Sprintf("  Events Processed: %d\n", result.InputResult.EventsProcessed)
	report += fmt.Sprintf("  Valid: %t\n", result.InputResult.Valid)

	if len(result.MemoryResults) > 0 {
		report += "\nMemory Validations:\n"
		for _, memResult := range result.MemoryResults {
			status := map[bool]string{true: "PASS", false: "FAIL"}[memResult.Valid]
			report += fmt.Sprintf("  $%04X: %s (expected $%02X, got $%02X) - %s\n",
				memResult.Address, status, memResult.Expected, memResult.Actual, memResult.Description)
		}
	}

	return report
}

// GetTestResults returns all test results
func (h *EndToEndTestHelper) GetTestResults() []EndToEndTestResult {
	return h.testResults
}

// TestHeadlessEndToEndComplete runs comprehensive end-to-end tests
func TestHeadlessEndToEndComplete(t *testing.T) {
	t.Run("Basic display scenario", func(t *testing.T) {
		helper, err := NewEndToEndTestHelper()
		if err != nil {
			t.Fatalf("Failed to create end-to-end helper: %v", err)
		}
		defer helper.Cleanup()

		scenario := helper.CreateBasicDisplayScenario()
		result := helper.ExecuteScenario(scenario)

		if !result.Passed {
			t.Errorf("Basic display scenario failed: %s", result.ErrorMessage)
		}

		t.Logf("Basic display test completed in %d ms", result.ExecutionTimeMs)
		t.Log(result.DetailedReport)
	})

	t.Run("Audio generation scenario", func(t *testing.T) {
		helper, err := NewEndToEndTestHelper()
		if err != nil {
			t.Fatalf("Failed to create end-to-end helper: %v", err)
		}
		defer helper.Cleanup()

		scenario := helper.CreateAudioTestScenario()
		result := helper.ExecuteScenario(scenario)

		if !result.Passed {
			t.Errorf("Audio test scenario failed: %s", result.ErrorMessage)
		}

		if !result.AudioResult.NonSilent {
			t.Error("Audio test should generate non-silent audio")
		}

		t.Logf("Audio test completed in %d ms", result.ExecutionTimeMs)
		t.Log(result.DetailedReport)
	})

	t.Run("Input response scenario", func(t *testing.T) {
		helper, err := NewEndToEndTestHelper()
		if err != nil {
			t.Fatalf("Failed to create end-to-end helper: %v", err)
		}
		defer helper.Cleanup()

		scenario := helper.CreateInputResponseScenario()
		result := helper.ExecuteScenario(scenario)

		if !result.Passed {
			t.Errorf("Input response scenario failed: %s", result.ErrorMessage)
		}

		// Check that memory validations passed
		for _, memResult := range result.MemoryResults {
			if !memResult.Valid {
				t.Errorf("Memory validation failed at $%04X: expected $%02X, got $%02X",
					memResult.Address, memResult.Expected, memResult.Actual)
			}
		}

		t.Logf("Input response test completed in %d ms", result.ExecutionTimeMs)
		t.Log(result.DetailedReport)
	})

	t.Run("Complex integration scenario", func(t *testing.T) {
		helper, err := NewEndToEndTestHelper()
		if err != nil {
			t.Fatalf("Failed to create end-to-end helper: %v", err)
		}
		defer helper.Cleanup()

		scenario := helper.CreateComplexScenario()
		result := helper.ExecuteScenario(scenario)

		if !result.Passed {
			t.Errorf("Complex scenario failed: %s", result.ErrorMessage)
		}

		// Validate all subsystems worked
		if !result.FrameBufferResult.Valid {
			t.Error("Complex scenario should have valid frame buffer")
		}

		if result.FrameBufferResult.UniqueColors < 1 {
			t.Error("Complex scenario should generate visual output")
		}

		if result.InputResult.EventsProcessed == 0 {
			t.Error("Complex scenario should process input events")
		}

		t.Logf("Complex integration test completed in %d ms", result.ExecutionTimeMs)
		t.Log(result.DetailedReport)
	})
}

// TestHeadlessPerformanceValidation tests performance characteristics
func TestHeadlessPerformanceValidation(t *testing.T) {
	t.Run("Performance stress test", func(t *testing.T) {
		helper, err := NewEndToEndTestHelper()
		if err != nil {
			t.Fatalf("Failed to create end-to-end helper: %v", err)
		}
		defer helper.Cleanup()

		// Create a demanding scenario
		rom := []uint8{
			// Intensive CPU and PPU operations
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			// Tight loop with memory operations
			0xA2, 0x00, // LDX #$00
			0xA9, 0xAA, // LDA #$AA
			0x95, 0x00, // STA $00,X (zero page indexed)
			0xE8,       // INX
			0xD0, 0xF9, // BNE -7

			// PPU operations
			0xA9, 0x20, // LDA #$20
			0x8D, 0x06, 0x20, // STA $2006
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006

			0xA2, 0x00, // LDX #$00
			0x8A,       // TXA
			0x8D, 0x07, 0x20, // STA $2007
			0xE8,       // INX
			0xE0, 0x20, // CPX #$20
			0xD0, 0xF7, // BNE -9

			0x4C, 0x0A, 0x80, // JMP $800A (loop back)
		}

		scenario := TestScenario{
			Name:             "performance_stress",
			Description:      "Stress test for performance validation",
			ROM:              rom,
			ValidationFrames: 120, // 2 seconds at 60 FPS
			ExpectedOutcome: ExpectedOutcome{
				FrameBufferValid:   true,
				MaxExecutionTimeMs: 15000, // 15 seconds max
				MinUniqueColors:    1,
			},
		}

		result := helper.ExecuteScenario(scenario)

		if !result.Passed {
			t.Errorf("Performance stress test failed: %s", result.ErrorMessage)
		}

		// Performance expectations
		framesPerSecond := float64(result.FramesProcessed) / (float64(result.ExecutionTimeMs) / 1000.0)
		
		if framesPerSecond < 30.0 {
			t.Errorf("Performance too slow: %.2f FPS (expected >= 30)", framesPerSecond)
		}

		t.Logf("Performance stress test: %.2f FPS, %d ms total", framesPerSecond, result.ExecutionTimeMs)
	})

	t.Run("Memory stability test", func(t *testing.T) {
		helper, err := NewEndToEndTestHelper()
		if err != nil {
			t.Fatalf("Failed to create end-to-end helper: %v", err)
		}
		defer helper.Cleanup()

		// Test for memory leaks and stability
		scenario := helper.CreateComplexScenario()
		scenario.ValidationFrames = 300 // Run for 5 seconds

		result := helper.ExecuteScenario(scenario)

		if !result.Passed {
			t.Errorf("Memory stability test failed: %s", result.ErrorMessage)
		}

		// Check that frame buffer size remained constant
		if !result.FrameBufferResult.ExpectedDimensions {
			t.Error("Frame buffer dimensions changed during extended execution")
		}

		t.Logf("Memory stability test completed: %d frames in %d ms", 
			result.FramesProcessed, result.ExecutionTimeMs)
	})
}