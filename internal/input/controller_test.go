package input

import (
	"testing"
)

func TestNew_ShouldCreateControllerWithDefaultState(t *testing.T) {
	controller := New()

	if controller == nil {
		t.Fatal("Expected controller, got nil")
	}
	if controller.buttons != 0 {
		t.Errorf("Expected initial buttons state 0, got %d", controller.buttons)
	}
	if controller.shiftRegister != 0 {
		t.Errorf("Expected initial shift register 0, got %d", controller.shiftRegister)
	}
	if controller.strobe != false {
		t.Error("Expected initial strobe false, got true")
	}
}

func TestSetButton_ShouldUpdateButtonState(t *testing.T) {
	controller := New()

	// Test setting individual buttons
	buttons := []Button{
		ButtonA, ButtonB, ButtonSelect, ButtonStart,
		ButtonUp, ButtonDown, ButtonLeft, ButtonRight,
	}

	for _, button := range buttons {
		controller.SetButton(button, true)

		if !controller.IsPressed(button) {
			t.Errorf("Button %d should be pressed after SetButton(true)", button)
		}

		// Verify only this button is set
		if controller.buttons != uint8(button) {
			t.Errorf("Expected buttons state %d, got %d", uint8(button), controller.buttons)
		}

		// Clear button for next test
		controller.SetButton(button, false)

		if controller.IsPressed(button) {
			t.Errorf("Button %d should not be pressed after SetButton(false)", button)
		}
	}
}

func TestSetButton_MultipleButtons_ShouldCombineStates(t *testing.T) {
	controller := New()

	// Set multiple buttons
	controller.SetButton(ButtonA, true)
	controller.SetButton(ButtonB, true)
	controller.SetButton(ButtonStart, true)

	expectedState := uint8(ButtonA) | uint8(ButtonB) | uint8(ButtonStart)

	if controller.buttons != expectedState {
		t.Errorf("Expected combined button state %d, got %d", expectedState, controller.buttons)
	}

	// Verify individual button states
	if !controller.IsPressed(ButtonA) {
		t.Error("ButtonA should be pressed")
	}
	if !controller.IsPressed(ButtonB) {
		t.Error("ButtonB should be pressed")
	}
	if !controller.IsPressed(ButtonStart) {
		t.Error("ButtonStart should be pressed")
	}
	if controller.IsPressed(ButtonSelect) {
		t.Error("ButtonSelect should not be pressed")
	}
}

func TestSetButton_ToggleBehavior_ShouldWorkCorrectly(t *testing.T) {
	controller := New()

	// Set button A
	controller.SetButton(ButtonA, true)
	if !controller.IsPressed(ButtonA) {
		t.Error("ButtonA should be pressed after first set")
	}

	// Set button A again (should remain set)
	controller.SetButton(ButtonA, true)
	if !controller.IsPressed(ButtonA) {
		t.Error("ButtonA should still be pressed after second set")
	}

	// Clear button A
	controller.SetButton(ButtonA, false)
	if controller.IsPressed(ButtonA) {
		t.Error("ButtonA should not be pressed after clear")
	}

	// Clear button A again (should remain clear)
	controller.SetButton(ButtonA, false)
	if controller.IsPressed(ButtonA) {
		t.Error("ButtonA should still not be pressed after second clear")
	}
}

func TestIsPressed_AllButtons_ShouldReportCorrectly(t *testing.T) {
	controller := New()

	// Test all buttons individually
	buttons := []Button{
		ButtonA, ButtonB, ButtonSelect, ButtonStart,
		ButtonUp, ButtonDown, ButtonLeft, ButtonRight,
	}

	// Initially, no buttons should be pressed
	for _, button := range buttons {
		if controller.IsPressed(button) {
			t.Errorf("Button %d should not be pressed initially", button)
		}
	}

	// Set all buttons and verify
	for _, button := range buttons {
		controller.SetButton(button, true)
	}

	for _, button := range buttons {
		if !controller.IsPressed(button) {
			t.Errorf("Button %d should be pressed after setting all", button)
		}
	}
}

func TestWrite_StrobeFalse_ShouldNotUpdateShiftRegister(t *testing.T) {
	controller := New()

	// Set some buttons
	controller.SetButton(ButtonA, true)
	controller.SetButton(ButtonB, true)

	// Write strobe = 0
	controller.Write(0x00)

	if controller.strobe != false {
		t.Error("Strobe should be false after writing 0")
	}
	if controller.shiftRegister != 0 {
		t.Errorf("Shift register should remain 0, got %d", controller.shiftRegister)
	}
}

func TestWrite_StrobeTrue_ShouldUpdateShiftRegister(t *testing.T) {
	controller := New()

	// Set some buttons
	controller.SetButton(ButtonA, true)
	controller.SetButton(ButtonB, true)

	expectedButtons := uint8(ButtonA) | uint8(ButtonB)

	// Write strobe = 1
	controller.Write(0x01)

	if controller.strobe != true {
		t.Error("Strobe should be true after writing 1")
	}
	if controller.shiftRegister != expectedButtons {
		t.Errorf("Shift register should be %d, got %d", expectedButtons, controller.shiftRegister)
	}
}

func TestWrite_StrobeWithHigherBits_ShouldIgnoreHigherBits(t *testing.T) {
	controller := New()
	controller.SetButton(ButtonA, true)

	// Write with high bits set, only bit 0 should matter
	controller.Write(0xFF) // All bits set

	if controller.strobe != true {
		t.Error("Strobe should be true (bit 0 set)")
	}

	controller.Write(0xFE) // All bits except bit 0 set

	if controller.strobe != false {
		t.Error("Strobe should be false (bit 0 clear)")
	}
}

func TestRead_StrobeActive_ShouldReturnButtonAState(t *testing.T) {
	controller := New()

	// Test with ButtonA not pressed
	controller.Write(0x01) // Enable strobe
	value := controller.Read()

	// Should return 0x40 (bit 6 set, bit 0 clear)
	expected := uint8(0x40)
	if value != expected {
		t.Errorf("Expected read value 0x%02X with ButtonA not pressed, got 0x%02X", expected, value)
	}

	// Test with ButtonA pressed
	controller.SetButton(ButtonA, true)
	controller.Write(0x01) // Refresh strobe
	value = controller.Read()

	// Should return 0x41 (bit 6 set, bit 0 set)
	expected = uint8(0x41)
	if value != expected {
		t.Errorf("Expected read value 0x%02X with ButtonA pressed, got 0x%02X", expected, value)
	}
}

func TestRead_StrobeInactive_ShouldShiftRegister(t *testing.T) {
	controller := New()

	// Set up button pattern: A and Start pressed (bits 0 and 3)
	controller.SetButton(ButtonA, true)
	controller.SetButton(ButtonStart, true)

	// Load shift register
	controller.Write(0x01) // Set strobe
	controller.Write(0x00) // Clear strobe

	// Read sequence should return buttons in order:
	// A, B, Select, Start, Up, Down, Left, Right
	expectedReadSequence := []uint8{
		0x41,                   // A pressed (bit 0) + bit 6
		0x40,                   // B not pressed + bit 6
		0x40,                   // Select not pressed + bit 6
		0x41,                   // Start pressed (bit 3 shifted to bit 0) + bit 6
		0x40, 0x40, 0x40, 0x40, // Up, Down, Left, Right not pressed + bit 6
	}

	for i, expected := range expectedReadSequence {
		value := controller.Read()
		if value != expected {
			t.Errorf("Read %d: expected 0x%02X, got 0x%02X", i, expected, value)
		}
	}
}

func TestRead_ExtendedReading_ShouldReturnZeros(t *testing.T) {
	controller := New()

	// Set one button
	controller.SetButton(ButtonA, true)

	// Load and start shifting
	controller.Write(0x01)
	controller.Write(0x00)

	// Read all 8 button states
	for i := 0; i < 8; i++ {
		controller.Read()
	}

	// Additional reads should return 0x40 (just bit 6, no button data)
	for i := 0; i < 5; i++ {
		value := controller.Read()
		if value != 0x40 {
			t.Errorf("Extended read %d: expected 0x40, got 0x%02X", i, value)
		}
	}
}

func TestRead_ButtonStateChange_DuringStrobe_ShouldUseOriginalState(t *testing.T) {
	controller := New()

	// Set initial state
	controller.SetButton(ButtonA, true)

	// Enable strobe (captures current state)
	controller.Write(0x01)

	// Change button state while strobe is active
	controller.SetButton(ButtonA, false)
	controller.SetButton(ButtonB, true)

	// Read should still return original ButtonA state
	value := controller.Read()
	expected := uint8(0x41) // ButtonA was pressed when strobe was set

	if value != expected {
		t.Errorf("Expected 0x%02X (original state), got 0x%02X", expected, value)
	}
}

func TestRead_ButtonStateChange_AfterStrobeCleared_ShouldUseSnapshotState(t *testing.T) {
	controller := New()

	// Set button pattern
	controller.SetButton(ButtonA, true)
	controller.SetButton(ButtonB, true)

	// Capture state and disable strobe
	controller.Write(0x01)
	controller.Write(0x00)

	// Change button state after strobe cleared
	controller.SetButton(ButtonA, false)
	controller.SetButton(ButtonSelect, true)

	// Read sequence should use the snapshot from when strobe was last active
	value1 := controller.Read() // Should be ButtonA (was pressed at snapshot)
	value2 := controller.Read() // Should be ButtonB (was pressed at snapshot)
	value3 := controller.Read() // Should be Select (was NOT pressed at snapshot)

	if value1 != 0x41 {
		t.Errorf("First read: expected 0x41 (A pressed in snapshot), got 0x%02X", value1)
	}
	if value2 != 0x41 {
		t.Errorf("Second read: expected 0x41 (B pressed in snapshot), got 0x%02X", value2)
	}
	if value3 != 0x40 {
		t.Errorf("Third read: expected 0x40 (Select not pressed in snapshot), got 0x%02X", value3)
	}
}

func TestReset_ShouldClearAllState(t *testing.T) {
	controller := New()

	// Set up some state
	controller.SetButton(ButtonA, true)
	controller.SetButton(ButtonB, true)
	controller.Write(0x01)

	// Verify state is set
	if controller.buttons == 0 {
		t.Error("Expected buttons to be set before reset")
	}
	if controller.shiftRegister == 0 {
		t.Error("Expected shift register to be set before reset")
	}
	if controller.strobe == false {
		t.Error("Expected strobe to be true before reset")
	}

	// Reset controller
	controller.Reset()

	// Verify all state is cleared
	if controller.buttons != 0 {
		t.Errorf("Expected buttons to be 0 after reset, got %d", controller.buttons)
	}
	if controller.shiftRegister != 0 {
		t.Errorf("Expected shift register to be 0 after reset, got %d", controller.shiftRegister)
	}
	if controller.strobe != false {
		t.Error("Expected strobe to be false after reset")
	}
}

func TestNewInputState_ShouldCreateTwoControllers(t *testing.T) {
	inputState := NewInputState()

	if inputState == nil {
		t.Fatal("Expected InputState, got nil")
	}
	if inputState.Controller1 == nil {
		t.Error("Expected Controller1, got nil")
	}
	if inputState.Controller2 == nil {
		t.Error("Expected Controller2, got nil")
	}

	// Verify they are different instances
	if inputState.Controller1 == inputState.Controller2 {
		t.Error("Controller1 and Controller2 should be different instances")
	}
}

func TestInputState_Reset_ShouldResetBothControllers(t *testing.T) {
	inputState := NewInputState()

	// Set up state on both controllers
	inputState.Controller1.SetButton(ButtonA, true)
	inputState.Controller2.SetButton(ButtonB, true)
	inputState.Controller1.Write(0x01)
	inputState.Controller2.Write(0x01)

	// Reset input state
	inputState.Reset()

	// Verify both controllers are reset
	if inputState.Controller1.buttons != 0 {
		t.Error("Controller1 should be reset")
	}
	if inputState.Controller2.buttons != 0 {
		t.Error("Controller2 should be reset")
	}
	if inputState.Controller1.strobe != false {
		t.Error("Controller1 strobe should be false after reset")
	}
	if inputState.Controller2.strobe != false {
		t.Error("Controller2 strobe should be false after reset")
	}
}

func TestInputState_Read_ShouldRouteToCorrectController(t *testing.T) {
	inputState := NewInputState()

	// Set different states for each controller
	inputState.Controller1.SetButton(ButtonA, true)
	inputState.Controller2.SetButton(ButtonB, true)

	// Enable strobe for both
	inputState.Controller1.Write(0x01)
	inputState.Controller2.Write(0x01)

	// Read from each controller port
	value1 := inputState.Read(0x4016) // Controller 1
	value2 := inputState.Read(0x4017) // Controller 2

	// Controller 1 should return ButtonA state
	expected1 := uint8(0x41) // ButtonA pressed + bit 6
	if value1 != expected1 {
		t.Errorf("Controller 1 read: expected 0x%02X, got 0x%02X", expected1, value1)
	}

	// Controller 2 should return ButtonB state (which is bit 0 when strobe is active)
	expected2 := uint8(0x40) // ButtonB is not bit 0, so not pressed + bit 6
	if value2 != expected2 {
		t.Errorf("Controller 2 read: expected 0x%02X, got 0x%02X", expected2, value2)
	}
}

func TestInputState_Read_InvalidAddress_ShouldReturnZero(t *testing.T) {
	inputState := NewInputState()

	// Test invalid addresses
	invalidAddresses := []uint16{
		0x4015, 0x4018, 0x5000, 0x0000, 0xFFFF,
	}

	for _, addr := range invalidAddresses {
		value := inputState.Read(addr)
		if value != 0 {
			t.Errorf("Invalid address 0x%04X should return 0, got %d", addr, value)
		}
	}
}

func TestInputState_Write_ShouldWriteToBothControllers(t *testing.T) {
	inputState := NewInputState()

	// Set button states
	inputState.Controller1.SetButton(ButtonA, true)
	inputState.Controller2.SetButton(ButtonB, true)

	// Write to controller port (should affect both)
	inputState.Write(0x4016, 0x01)

	// Both controllers should have strobe enabled
	if inputState.Controller1.strobe != true {
		t.Error("Controller1 strobe should be true after write")
	}
	if inputState.Controller2.strobe != true {
		t.Error("Controller2 strobe should be true after write")
	}

	// Both should have their button states captured
	if inputState.Controller1.shiftRegister != uint8(ButtonA) {
		t.Error("Controller1 shift register should contain ButtonA")
	}
	if inputState.Controller2.shiftRegister != uint8(ButtonB) {
		t.Error("Controller2 shift register should contain ButtonB")
	}
}

func TestInputState_Write_InvalidAddress_ShouldBeIgnored(t *testing.T) {
	inputState := NewInputState()

	// Set initial state
	inputState.Controller1.SetButton(ButtonA, true)
	initialState1 := inputState.Controller1.buttons
	initialStrobe1 := inputState.Controller1.strobe

	// Write to invalid address
	inputState.Write(0x4017, 0x01) // 0x4017 is read-only
	inputState.Write(0x5000, 0x01) // Invalid address

	// State should remain unchanged
	if inputState.Controller1.buttons != initialState1 {
		t.Error("Controller1 buttons should be unchanged after invalid write")
	}
	if inputState.Controller1.strobe != initialStrobe1 {
		t.Error("Controller1 strobe should be unchanged after invalid write")
	}
}

// Test the standard NES controller reading sequence
func TestControllerReadingSequence_StandardPattern_ShouldMatchExpected(t *testing.T) {
	controller := New()

	// Set the standard test pattern: A + Start + Right
	controller.SetButton(ButtonA, true)
	controller.SetButton(ButtonStart, true)
	controller.SetButton(ButtonRight, true)

	// Standard NES reading sequence
	controller.Write(0x01) // Set strobe
	controller.Write(0x00) // Clear strobe

	// Expected sequence: A, B, Select, Start, Up, Down, Left, Right
	expectedSequence := []struct {
		button   string
		expected uint8
	}{
		{"A", 0x41},      // Pressed
		{"B", 0x40},      // Not pressed
		{"Select", 0x40}, // Not pressed
		{"Start", 0x41},  // Pressed
		{"Up", 0x40},     // Not pressed
		{"Down", 0x40},   // Not pressed
		{"Left", 0x40},   // Not pressed
		{"Right", 0x41},  // Pressed
	}

	for i, expected := range expectedSequence {
		value := controller.Read()
		if value != expected.expected {
			t.Errorf("Button %s (position %d): expected 0x%02X, got 0x%02X",
				expected.button, i, expected.expected, value)
		}
	}
}

// Test rapid strobe cycling (edge case)
func TestController_RapidStrobeCycle_ShouldWorkCorrectly(t *testing.T) {
	controller := New()
	controller.SetButton(ButtonA, true)

	// Rapid strobe cycling
	for i := 0; i < 10; i++ {
		controller.Write(0x01)
		controller.Write(0x00)

		// First read should always be ButtonA
		value := controller.Read()
		if value != 0x41 {
			t.Errorf("Rapid cycle %d: expected 0x41, got 0x%02X", i, value)
		}
	}
}

// Test incomplete read sequence behavior
func TestController_IncompleteReadSequence_ShouldResumeCorrectly(t *testing.T) {
	controller := New()

	// Set button pattern
	controller.SetButton(ButtonA, true)
	controller.SetButton(ButtonSelect, true)

	// Start reading sequence
	controller.Write(0x01)
	controller.Write(0x00)

	// Read first two buttons
	value1 := controller.Read() // A - should be 0x41
	value2 := controller.Read() // B - should be 0x40

	if value1 != 0x41 {
		t.Errorf("First read: expected 0x41, got 0x%02X", value1)
	}
	if value2 != 0x40 {
		t.Errorf("Second read: expected 0x40, got 0x%02X", value2)
	}

	// Re-strobe (should reset sequence)
	controller.Write(0x01)
	controller.Write(0x00)

	// Should start over with ButtonA
	value3 := controller.Read()
	if value3 != 0x41 {
		t.Errorf("After re-strobe: expected 0x41, got 0x%02X", value3)
	}
}

// Benchmark tests for controller performance
func BenchmarkController_SetButton(b *testing.B) {
	controller := New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controller.SetButton(ButtonA, true)
		controller.SetButton(ButtonA, false)
	}
}

func BenchmarkController_ReadSequence(b *testing.B) {
	controller := New()
	controller.SetButton(ButtonA, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		controller.Write(0x01)
		controller.Write(0x00)
		for j := 0; j < 8; j++ {
			controller.Read()
		}
	}
}

func BenchmarkInputState_DualController(b *testing.B) {
	inputState := NewInputState()
	inputState.Controller1.SetButton(ButtonA, true)
	inputState.Controller2.SetButton(ButtonB, true)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		inputState.Write(0x4016, 0x01)
		inputState.Write(0x4016, 0x00)
		for j := 0; j < 8; j++ {
			inputState.Read(0x4016)
			inputState.Read(0x4017)
		}
	}
}
