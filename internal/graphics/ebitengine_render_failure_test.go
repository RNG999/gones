//go:build !headless
// +build !headless

package graphics

import (
	"testing"
)

// TestRenderingPipeline_FailsWithoutRenderCalls tests that rendering fails when render() is not called
func TestRenderingPipeline_FailsWithoutRenderCalls(t *testing.T) {
	// Initialize backend
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Failure Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	// Create window but don't call RenderFrame
	window, err := backend.CreateWindow("Failure Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Verify that no frame has been rendered yet
	ebitengineWindow := window.(*EbitengineWindow)
	if ebitengineWindow.game == nil {
		t.Fatal("Game should be initialized after window creation")
	}
	
	// Check that frame buffer is empty/default initialized
	frameBuffer := ebitengineWindow.GetFrameBufferForTesting()
	
	// All pixels should be zero (default initialization)
	nonZeroPixels := 0
	for i := 0; i < len(frameBuffer); i++ {
		if frameBuffer[i] != 0 {
			nonZeroPixels++
		}
	}
	
	if nonZeroPixels > 0 {
		t.Errorf("Expected all pixels to be zero before rendering, found %d non-zero pixels", nonZeroPixels)
	}
}

// TestRenderingPipeline_FailsWithoutEmulatorUpdate tests that emulator updates are not called without setup
func TestRenderingPipeline_FailsWithoutEmulatorUpdate(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Update Failure Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Update Failure Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	ebitengineWindow := window.(*EbitengineWindow)
	
	// Verify no emulator update function is set initially
	updateFunc := ebitengineWindow.GetEmulatorUpdateFuncForTesting()
	if updateFunc != nil {
		t.Error("Emulator update function should be nil initially")
	}
	
	// Call game update without setting emulator update function
	err = ebitengineWindow.game.Update()
	if err != nil {
		t.Fatalf("Game update should not fail even without emulator update function: %v", err)
	}
	
	// This should pass because the game update is designed to handle nil emulator update function
}

// TestRenderingPipeline_FailsWithoutFrameBuffer tests rendering without proper frame buffer
func TestRenderingPipeline_FailsWithoutFrameBuffer(t *testing.T) {
	// Create a window with nil game to simulate broken state
	window := &EbitengineWindow{
		game: nil,
	}
	
	// Attempt to render frame with nil game
	var frameBuffer [256 * 240]uint32
	for i := 0; i < len(frameBuffer); i++ {
		frameBuffer[i] = 0xFF0000FF // Red
	}
	
	err := window.RenderFrame(frameBuffer)
	if err == nil {
		t.Fatal("Expected error when rendering with nil game, got nil")
	}
	
	expectedError := "game not initialized"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestRenderingPipeline_FrameBufferNotTransferred tests detection of frame buffer transfer issues
func TestRenderingPipeline_FrameBufferNotTransferred(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Transfer Failure Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Transfer Failure Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Create a unique frame buffer pattern
	var originalFrameBuffer [256 * 240]uint32
	for i := 0; i < len(originalFrameBuffer); i++ {
		originalFrameBuffer[i] = 0x12345678 // Unique pattern
	}
	
	// Render the frame
	err = window.RenderFrame(originalFrameBuffer)
	if err != nil {
		t.Fatalf("Frame render failed: %v", err)
	}
	
	// Verify frame buffer was properly transferred
	ebitengineWindow := window.(*EbitengineWindow)
	actualFrameBuffer := ebitengineWindow.GetFrameBufferForTesting()
	
	// Check that the transfer actually occurred
	for i := 0; i < 10; i++ {
		expected := originalFrameBuffer[i]
		actual := actualFrameBuffer[i]
		if actual != expected {
			t.Errorf("Frame buffer transfer failed at pixel %d: expected 0x%08X, got 0x%08X", i, expected, actual)
		}
	}
	
	// This test should PASS if the implementation is correct,
	// but it demonstrates the verification that would fail if RenderFrame wasn't working
}

// TestEbitengineGame_UpdateWithoutRenderLoop tests game update without proper rendering loop
func TestEbitengineGame_UpdateWithoutRenderLoop(t *testing.T) {
	// Create isolated game instance
	game := &EbitengineGame{
		window:      nil, // No window connection
		nesWidth:    256,
		nesHeight:   240,
		windowWidth: 800,
		windowHeight: 600,
	}
	
	// Update should handle missing window gracefully
	err := game.Update()
	if err != nil {
		t.Fatalf("Game update with nil window should not fail: %v", err)
	}
	
	// Create window but don't set emulator update function
	window := &EbitengineWindow{}
	game.window = window
	
	// Update should still work without emulator update function
	err = game.Update()
	if err != nil {
		t.Fatalf("Game update without emulator function should not fail: %v", err)
	}
	
	// Set emulator update function that returns error
	window.emulatorUpdateFunc = func() error {
		return &MockRenderError{message: "emulator failed"}
	}
	
	// Update should handle emulator errors gracefully (log but continue)
	err = game.Update()
	if err != nil {
		t.Fatalf("Game update should handle emulator errors gracefully: %v", err)
	}
}

// MockRenderError simulates rendering errors
type MockRenderError struct {
	message string
}

func (e *MockRenderError) Error() string {
	return e.message
}

// TestRenderingPipeline_DetectsFrameBufferCorruption tests detection of frame buffer corruption
func TestRenderingPipeline_DetectsFrameBufferCorruption(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Corruption Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Corruption Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Render first frame with known pattern
	var frameBuffer1 [256 * 240]uint32
	for i := 0; i < len(frameBuffer1); i++ {
		frameBuffer1[i] = 0xAABBCCDD
	}
	
	err = window.RenderFrame(frameBuffer1)
	if err != nil {
		t.Fatalf("First frame render failed: %v", err)
	}
	
	// Render second frame with different pattern
	var frameBuffer2 [256 * 240]uint32
	for i := 0; i < len(frameBuffer2); i++ {
		frameBuffer2[i] = 0x11223344
	}
	
	err = window.RenderFrame(frameBuffer2)
	if err != nil {
		t.Fatalf("Second frame render failed: %v", err)
	}
	
	// Verify the latest frame is correctly stored
	ebitengineWindow := window.(*EbitengineWindow)
	actualFrameBuffer := ebitengineWindow.GetFrameBufferForTesting()
	
	// Should contain the second frame buffer, not the first
	for i := 0; i < 10; i++ {
		expected := frameBuffer2[i]
		actual := actualFrameBuffer[i]
		if actual != expected {
			t.Errorf("Frame buffer corruption detected at pixel %d: expected 0x%08X, got 0x%08X", i, expected, actual)
		}
		
		// Should NOT contain the first frame buffer
		if actual == frameBuffer1[i] {
			t.Errorf("Frame buffer contains old data at pixel %d: got 0x%08X", i, actual)
		}
	}
}