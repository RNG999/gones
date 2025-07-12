package integration

import (
	"fmt"
	"testing"

	"gones/internal/input"
)

// InputTestHelper provides utilities for testing input simulation without SDL2
type InputTestHelper struct {
	*HeadlessEmulatorTestHelper
	inputSequences    []InputSequence
	inputTestResults  []InputTestResult
	controllerStates  map[int]*ControllerState
}

// InputSequence represents a sequence of input events over time
type InputSequence struct {
	Name        string
	Events      []ScheduledInputEvent
	Description string
}

// ScheduledInputEvent represents an input event scheduled for a specific frame
type ScheduledInputEvent struct {
	Frame      int
	Controller int
	Button     input.Button
	Pressed    bool
}

// InputTestResult represents the result of an input test
type InputTestResult struct {
	TestName        string
	Sequence        string
	ExpectedReads   int
	ActualReads     int
	ButtonsDetected []input.Button
	Valid           bool
	Message         string
}

// ControllerState tracks the state of a controller for validation
type ControllerState struct {
	ButtonStates map[input.Button]bool
	LastRead     map[input.Button]bool
	ReadCount    int
}

// NewInputTestHelper creates a new input test helper
func NewInputTestHelper() (*InputTestHelper, error) {
	headlessHelper, err := NewHeadlessEmulatorTestHelper()
	if err != nil {
		return nil, err
	}

	return &InputTestHelper{
		HeadlessEmulatorTestHelper: headlessHelper,
		inputSequences:            make([]InputSequence, 0),
		inputTestResults:          make([]InputTestResult, 0),
		controllerStates:          make(map[int]*ControllerState),
	}, nil
}

// CreateInputSequence creates a new input sequence for testing
func (h *InputTestHelper) CreateInputSequence(name, description string) *InputSequence {
	sequence := InputSequence{
		Name:        name,
		Events:      make([]ScheduledInputEvent, 0),
		Description: description,
	}

	h.inputSequences = append(h.inputSequences, sequence)
	return &h.inputSequences[len(h.inputSequences)-1]
}

// AddInputEvent adds an input event to a sequence
func (seq *InputSequence) AddInputEvent(frame int, controller int, button input.Button, pressed bool) {
	event := ScheduledInputEvent{
		Frame:      frame,
		Controller: controller,
		Button:     button,
		Pressed:    pressed,
	}
	seq.Events = append(seq.Events, event)
}

// ExecuteInputSequence executes an input sequence and validates the results
func (h *InputTestHelper) ExecuteInputSequence(sequence InputSequence, maxFrames int) InputTestResult {
	result := InputTestResult{
		TestName:        sequence.Name,
		Sequence:        sequence.Description,
		ButtonsDetected: make([]input.Button, 0),
		Valid:          false,
	}

	// Schedule all events from the sequence
	for _, event := range sequence.Events {
		h.ScheduleInputEvent(event.Controller, event.Button, event.Pressed, event.Frame)
	}

	// Create ROM that reads controller input
	program := h.createControllerReadROM()

	err := h.LoadMockROM(program)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to load test ROM: %v", err)
		h.inputTestResults = append(h.inputTestResults, result)
		return result
	}

	// Run the test
	err = h.RunHeadlessFrames(maxFrames)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to run test frames: %v", err)
		h.inputTestResults = append(h.inputTestResults, result)
		return result
	}

	// Validate the results by checking memory locations where button states were stored
	result = h.validateInputResults(sequence, result)
	h.inputTestResults = append(h.inputTestResults, result)
	return result
}

// createControllerReadROM creates a ROM that reads controller input and stores results
func (h *InputTestHelper) createControllerReadROM() []uint8 {
	return []uint8{
		// Main loop - continuously read controller
		// Strobe controller 1
		0xA9, 0x01,       // LDA #$01
		0x8D, 0x16, 0x40, // STA $4016 (strobe high)
		0xA9, 0x00,       // LDA #$00
		0x8D, 0x16, 0x40, // STA $4016 (strobe low)

		// Read 8 buttons from controller 1
		0xAD, 0x16, 0x40, // LDA $4016 (A button)
		0x29, 0x01,       // AND #$01 (mask bit 0)
		0x85, 0x10,       // STA $10   (store A button state)

		0xAD, 0x16, 0x40, // LDA $4016 (B button)
		0x29, 0x01,       // AND #$01
		0x85, 0x11,       // STA $11   (store B button state)

		0xAD, 0x16, 0x40, // LDA $4016 (Select button)
		0x29, 0x01,       // AND #$01
		0x85, 0x12,       // STA $12   (store Select button state)

		0xAD, 0x16, 0x40, // LDA $4016 (Start button)
		0x29, 0x01,       // AND #$01
		0x85, 0x13,       // STA $13   (store Start button state)

		0xAD, 0x16, 0x40, // LDA $4016 (Up button)
		0x29, 0x01,       // AND #$01
		0x85, 0x14,       // STA $14   (store Up button state)

		0xAD, 0x16, 0x40, // LDA $4016 (Down button)
		0x29, 0x01,       // AND #$01
		0x85, 0x15,       // STA $15   (store Down button state)

		0xAD, 0x16, 0x40, // LDA $4016 (Left button)
		0x29, 0x01,       // AND #$01
		0x85, 0x16,       // STA $16   (store Left button state)

		0xAD, 0x16, 0x40, // LDA $4016 (Right button)
		0x29, 0x01,       // AND #$01
		0x85, 0x17,       // STA $17   (store Right button state)

		// Increment read counter
		0xE6, 0x20,       // INC $20

		// Short delay then loop
		0xA2, 0xFF,       // LDX #$FF
		0xCA,             // DEX (delay loop)
		0xD0, 0xFD,       // BNE -3

		0x4C, 0x00, 0x80, // JMP $8000 (repeat)
	}
}

// validateInputResults validates the input test results by examining memory
func (h *InputTestHelper) validateInputResults(sequence InputSequence, result InputTestResult) InputTestResult {
	bus := h.app.GetBus()
	if bus == nil {
		result.Message = "Bus not available for validation"
		return result
	}

	// Check the read counter to see how many times input was read
	memory := bus.Memory
	if memory != nil {
		result.ActualReads = int(memory.Read(0x0020))
	}

	// Check for button presses by examining memory locations
	buttonMap := map[uint16]input.Button{
		0x0010: input.A,
		0x0011: input.B,
		0x0012: input.Select,
		0x0013: input.Start,
		0x0014: input.Up,
		0x0015: input.Down,
		0x0016: input.Left,
		0x0017: input.Right,
	}

	detectedButtons := make(map[input.Button]bool)

	if memory != nil {
		for addr, button := range buttonMap {
			buttonState := memory.Read(addr)
			if buttonState == 1 {
				detectedButtons[button] = true
				result.ButtonsDetected = append(result.ButtonsDetected, button)
			}
		}
	}

	// Validate against expected inputs from sequence
	expectedButtons := make(map[input.Button]bool)
	for _, event := range sequence.Events {
		if event.Pressed {
			expectedButtons[event.Button] = true
		}
	}

	// Count matches
	matches := 0
	for button := range expectedButtons {
		if detectedButtons[button] {
			matches++
		}
	}

	// Determine if test passed
	result.ExpectedReads = len(expectedButtons)
	totalExpected := len(expectedButtons)
	
	if totalExpected == 0 {
		result.Valid = result.ActualReads > 0 // At least some input reading occurred
		result.Message = fmt.Sprintf("Input reading test: %d reads performed", result.ActualReads)
	} else {
		result.Valid = matches == totalExpected
		result.Message = fmt.Sprintf("Input validation: %d/%d buttons detected correctly, %d reads",
			matches, totalExpected, result.ActualReads)
	}

	return result
}

// GetInputTestResults returns all input test results
func (h *InputTestHelper) GetInputTestResults() []InputTestResult {
	return h.inputTestResults
}

// TestHeadlessInputBasics tests basic input functionality without SDL2
func TestHeadlessInputBasics(t *testing.T) {
	t.Run("Single button press test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Create sequence with single A button press
		sequence := helper.CreateInputSequence("single_a_press", "Press A button on frame 5")
		sequence.AddInputEvent(5, 1, input.A, true)  // Press A
		sequence.AddInputEvent(10, 1, input.A, false) // Release A

		result := helper.ExecuteInputSequence(*sequence, 20)

		if !result.Valid {
			t.Errorf("Single button press test failed: %s", result.Message)
		}

		// Check that A button was detected
		aButtonFound := false
		for _, button := range result.ButtonsDetected {
			if button == input.A {
				aButtonFound = true
				break
			}
		}

		if !aButtonFound {
			t.Error("A button press was not detected")
		}

		t.Logf("Single button test: %s", result.Message)
	})

	t.Run("Multiple button press test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Create sequence with multiple buttons
		sequence := helper.CreateInputSequence("multi_button", "Press A, B, Start, Select")
		sequence.AddInputEvent(2, 1, input.A, true)
		sequence.AddInputEvent(3, 1, input.B, true)
		sequence.AddInputEvent(4, 1, input.Start, true)
		sequence.AddInputEvent(5, 1, input.Select, true)
		
		// Release all buttons
		sequence.AddInputEvent(10, 1, input.A, false)
		sequence.AddInputEvent(11, 1, input.B, false)
		sequence.AddInputEvent(12, 1, input.Start, false)
		sequence.AddInputEvent(13, 1, input.Select, false)

		result := helper.ExecuteInputSequence(*sequence, 25)

		expectedButtons := []input.Button{input.A, input.B, input.Start, input.Select}
		detectedCount := 0

		for _, expectedButton := range expectedButtons {
			for _, detectedButton := range result.ButtonsDetected {
				if expectedButton == detectedButton {
					detectedCount++
					break
				}
			}
		}

		if detectedCount != len(expectedButtons) {
			t.Errorf("Expected %d buttons, detected %d", len(expectedButtons), detectedCount)
		}

		t.Logf("Multiple button test: %s", result.Message)
	})

	t.Run("D-pad input test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Create sequence with D-pad movements
		sequence := helper.CreateInputSequence("dpad_test", "Test all D-pad directions")
		sequence.AddInputEvent(2, 1, input.Up, true)
		sequence.AddInputEvent(4, 1, input.Up, false)
		sequence.AddInputEvent(5, 1, input.Right, true)
		sequence.AddInputEvent(7, 1, input.Right, false)
		sequence.AddInputEvent(8, 1, input.Down, true)
		sequence.AddInputEvent(10, 1, input.Down, false)
		sequence.AddInputEvent(11, 1, input.Left, true)
		sequence.AddInputEvent(13, 1, input.Left, false)

		result := helper.ExecuteInputSequence(*sequence, 20)

		// Check for all D-pad directions
		dpadButtons := []input.Button{input.Up, input.Down, input.Left, input.Right}
		dpadDetected := 0

		for _, dpadButton := range dpadButtons {
			for _, detectedButton := range result.ButtonsDetected {
				if dpadButton == detectedButton {
					dpadDetected++
					break
				}
			}
		}

		if dpadDetected != len(dpadButtons) {
			t.Errorf("D-pad test: expected %d directions, detected %d", len(dpadButtons), dpadDetected)
		}

		t.Logf("D-pad test: %s", result.Message)
	})
}

// TestHeadlessInputAdvanced tests advanced input scenarios
func TestHeadlessInputAdvanced(t *testing.T) {
	t.Run("Rapid button presses test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Create sequence with rapid button presses
		sequence := helper.CreateInputSequence("rapid_press", "Rapid A button presses")
		
		// Create rapid press/release sequence
		for i := 0; i < 10; i++ {
			sequence.AddInputEvent(i*2, 1, input.A, true)   // Press every 2 frames
			sequence.AddInputEvent(i*2+1, 1, input.A, false) // Release next frame
		}

		result := helper.ExecuteInputSequence(*sequence, 30)

		// For rapid presses, we expect the input system to handle it gracefully
		if result.ActualReads == 0 {
			t.Error("No input reads detected during rapid press test")
		}

		t.Logf("Rapid press test: %s", result.Message)
	})

	t.Run("Button combinations test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Test common button combinations
		sequence := helper.CreateInputSequence("combinations", "Test button combinations")
		
		// Start + Select (common reset combination)
		sequence.AddInputEvent(2, 1, input.Start, true)
		sequence.AddInputEvent(2, 1, input.Select, true)
		sequence.AddInputEvent(8, 1, input.Start, false)
		sequence.AddInputEvent(8, 1, input.Select, false)

		// A + B (common game combination)
		sequence.AddInputEvent(10, 1, input.A, true)
		sequence.AddInputEvent(10, 1, input.B, true)
		sequence.AddInputEvent(15, 1, input.A, false)
		sequence.AddInputEvent(15, 1, input.B, false)

		result := helper.ExecuteInputSequence(*sequence, 25)

		// Should detect all buttons in combinations
		expectedButtons := []input.Button{input.Start, input.Select, input.A, input.B}
		for _, expectedButton := range expectedButtons {
			found := false
			for _, detectedButton := range result.ButtonsDetected {
				if expectedButton == detectedButton {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Button %v not detected in combination test", expectedButton)
			}
		}

		t.Logf("Button combinations test: %s", result.Message)
	})

	t.Run("Controller sequence timing test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Test input timing precision
		sequence := helper.CreateInputSequence("timing_test", "Test input timing precision")

		// Precise timing sequence
		sequence.AddInputEvent(5, 1, input.A, true)
		sequence.AddInputEvent(6, 1, input.A, false)
		sequence.AddInputEvent(10, 1, input.B, true)
		sequence.AddInputEvent(11, 1, input.B, false)
		sequence.AddInputEvent(15, 1, input.Start, true)
		sequence.AddInputEvent(16, 1, input.Start, false)

		result := helper.ExecuteInputSequence(*sequence, 25)

		// Validate that input system can handle precise timing
		if result.ActualReads < 10 {
			t.Errorf("Expected frequent input reads for timing test, got %d", result.ActualReads)
		}

		expectedButtons := []input.Button{input.A, input.B, input.Start}
		detectedCount := 0
		for _, expected := range expectedButtons {
			for _, detected := range result.ButtonsDetected {
				if expected == detected {
					detectedCount++
					break
				}
			}
		}

		if detectedCount != len(expectedButtons) {
			t.Errorf("Timing test: expected %d buttons, detected %d", len(expectedButtons), detectedCount)
		}

		t.Logf("Timing test: %s", result.Message)
	})
}

// TestHeadlessInputValidation tests input validation capabilities
func TestHeadlessInputValidation(t *testing.T) {
	t.Run("Input system initialization test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Test that input system is properly initialized
		bus := helper.app.GetBus()
		if bus == nil {
			t.Fatal("Bus not available")
		}

		inputState := bus.GetInputState()
		if inputState == nil {
			t.Fatal("Input state not available")
		}

		// Test basic input functionality
		sequence := helper.CreateInputSequence("init_test", "Test input system initialization")
		sequence.AddInputEvent(1, 1, input.A, true)
		sequence.AddInputEvent(3, 1, input.A, false)

		result := helper.ExecuteInputSequence(*sequence, 10)

		if result.ActualReads == 0 {
			t.Error("Input system appears to be non-functional")
		}

		t.Logf("Input initialization test: %s", result.Message)
	})

	t.Run("Controller state persistence test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Test that controller state persists correctly
		sequence := helper.CreateInputSequence("persistence_test", "Test state persistence")
		
		// Hold button for several frames
		sequence.AddInputEvent(2, 1, input.A, true)
		// Don't release until much later
		sequence.AddInputEvent(20, 1, input.A, false)

		result := helper.ExecuteInputSequence(*sequence, 25)

		// Should detect the button since it was held
		aFound := false
		for _, button := range result.ButtonsDetected {
			if button == input.A {
				aFound = true
				break
			}
		}

		if !aFound {
			t.Error("Button state did not persist across frames")
		}

		t.Logf("State persistence test: %s", result.Message)
	})

	t.Run("Input edge cases test", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input test helper: %v", err)
		}
		defer helper.Cleanup()

		// Test edge cases
		sequence := helper.CreateInputSequence("edge_cases", "Test input edge cases")

		// Immediate press/release
		sequence.AddInputEvent(0, 1, input.A, true)
		sequence.AddInputEvent(0, 1, input.A, false)

		// Press at end of sequence
		sequence.AddInputEvent(19, 1, input.B, true)

		result := helper.ExecuteInputSequence(*sequence, 20)

		// Edge cases should be handled gracefully
		if result.ActualReads == 0 {
			t.Error("Input system failed to handle edge cases")
		}

		t.Logf("Edge cases test: %s", result.Message)
	})
}