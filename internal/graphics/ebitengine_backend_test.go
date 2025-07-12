//go:build !headless
// +build !headless

package graphics

import (
	"testing"
)

// Test-specific mock types for isolated testing
type MockEbitengineGame struct {
	frameBuffer     [256 * 240]uint32
	updateCalled    bool
	drawCalled      bool
	updateFunc      func() error
	updateCallCount int
	drawCallCount   int
}

func (m *MockEbitengineGame) Update() error {
	m.updateCalled = true
	m.updateCallCount++
	if m.updateFunc != nil {
		return m.updateFunc()
	}
	return nil
}

func (m *MockEbitengineGame) Draw(screen interface{}) {
	m.drawCalled = true
	m.drawCallCount++
}

func (m *MockEbitengineGame) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return outsideWidth, outsideHeight
}

// TestEbitengineBackend_Initialize tests backend initialization
func TestEbitengineBackend_Initialize(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle:  "Test Window",
		WindowWidth:  800,
		WindowHeight: 600,
		Fullscreen:   false,
		VSync:        true,
		Filter:       "nearest",
		AspectRatio:  "4:3",
		Headless:     false,
		Debug:        false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Expected successful initialization, got error: %v", err)
	}
	
	// Verify backend is initialized
	if !backend.(*EbitengineBackend).initialized {
		t.Error("Backend should be marked as initialized")
	}
	
	// Verify config is stored
	if backend.(*EbitengineBackend).config.WindowTitle != "Test Window" {
		t.Error("Config not properly stored during initialization")
	}
}

// TestEbitengineBackend_DoubleInitialize tests that double initialization fails
func TestEbitengineBackend_DoubleInitialize(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle: "Test Window",
		Headless:    false,
	}
	
	// First initialization should succeed
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("First initialization failed: %v", err)
	}
	
	// Second initialization should fail
	err = backend.Initialize(config)
	if err == nil {
		t.Fatal("Expected error on double initialization, got nil")
	}
	
	expectedError := "Ebitengine backend already initialized"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestEbitengineBackend_CreateWindow tests window creation
func TestEbitengineBackend_CreateWindow(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle:  "Test Window",
		WindowWidth:  800,
		WindowHeight: 600,
		Headless:     false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Test Game", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	if window == nil {
		t.Fatal("Window should not be nil")
	}
	
	// Verify window properties
	width, height := window.GetSize()
	if width != 800 || height != 600 {
		t.Errorf("Expected window size 800x600, got %dx%d", width, height)
	}
	
	// Verify backend has game instance
	ebitengineBackend := backend.(*EbitengineBackend)
	if ebitengineBackend.game == nil {
		t.Error("Backend should have game instance after window creation")
	}
}

// TestEbitengineBackend_CreateWindow_Uninitialized tests window creation on uninitialized backend
func TestEbitengineBackend_CreateWindow_Uninitialized(t *testing.T) {
	backend := NewEbitengineBackend()
	
	_, err := backend.CreateWindow("Test Game", 800, 600)
	if err == nil {
		t.Fatal("Expected error when creating window on uninitialized backend")
	}
	
	expectedError := "backend not initialized"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestEbitengineBackend_CreateWindow_Headless tests window creation in headless mode
func TestEbitengineBackend_CreateWindow_Headless(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		Headless: true,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	_, err = backend.CreateWindow("Test Game", 800, 600)
	if err == nil {
		t.Fatal("Expected error when creating window in headless mode")
	}
	
	expectedError := "cannot create window in headless mode"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestEbitengineWindow_RenderFrame tests frame rendering functionality
func TestEbitengineWindow_RenderFrame(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle: "Test Window",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Test Game", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Create test frame buffer with specific pattern
	var frameBuffer [256 * 240]uint32
	for i := 0; i < len(frameBuffer); i++ {
		// Create a pattern: red pixels at even indices, blue at odd
		if i%2 == 0 {
			frameBuffer[i] = 0xFF0000FF // Red
		} else {
			frameBuffer[i] = 0x0000FFFF // Blue
		}
	}
	
	// Test frame rendering
	err = window.RenderFrame(frameBuffer)
	if err != nil {
		t.Fatalf("RenderFrame failed: %v", err)
	}
	
	// Verify frame buffer was copied to game
	ebitengineWindow := window.(*EbitengineWindow)
	if ebitengineWindow.game == nil {
		t.Fatal("Game instance should not be nil after rendering")
	}
	
	// Verify frame buffer content
	for i := 0; i < 10; i++ { // Check first 10 pixels
		expected := frameBuffer[i]
		actual := ebitengineWindow.game.frameBuffer[i]
		if actual != expected {
			t.Errorf("Frame buffer pixel %d: expected 0x%08X, got 0x%08X", i, expected, actual)
		}
	}
}

// TestEbitengineWindow_RenderFrame_NilGame tests rendering with nil game
func TestEbitengineWindow_RenderFrame_NilGame(t *testing.T) {
	window := &EbitengineWindow{
		game: nil,
	}
	
	var frameBuffer [256 * 240]uint32
	err := window.RenderFrame(frameBuffer)
	if err == nil {
		t.Fatal("Expected error when rendering with nil game")
	}
	
	expectedError := "game not initialized"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestEbitengineWindow_EmulatorUpdateFunc tests emulator update function integration
func TestEbitengineWindow_EmulatorUpdateFunc(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle: "Test Window",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Test Game", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	ebitengineWindow := window.(*EbitengineWindow)
	
	// Test setting emulator update function
	updateCalled := false
	updateFunc := func() error {
		updateCalled = true
		return nil
	}
	
	ebitengineWindow.SetEmulatorUpdateFunc(updateFunc)
	
	// Verify function was set
	if ebitengineWindow.emulatorUpdateFunc == nil {
		t.Fatal("Emulator update function should be set")
	}
	
	// Test calling the game's Update method
	err = ebitengineWindow.game.Update()
	if err != nil {
		t.Fatalf("Game Update failed: %v", err)
	}
	
	// Verify update function was called
	if !updateCalled {
		t.Error("Emulator update function should have been called during game update")
	}
}

// TestEbitengineWindow_EmulatorUpdateFunc_Error tests error handling in emulator update
func TestEbitengineWindow_EmulatorUpdateFunc_Error(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle: "Test Window",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Test Game", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	ebitengineWindow := window.(*EbitengineWindow)
	
	// Set update function that returns error
	updateFunc := func() error {
		return &MockError{message: "emulator error"}
	}
	
	ebitengineWindow.SetEmulatorUpdateFunc(updateFunc)
	
	// Game Update should not fail even if emulator update fails
	err = ebitengineWindow.game.Update()
	if err != nil {
		t.Fatalf("Game Update should not fail when emulator update fails: %v", err)
	}
}

// TestEbitengineGame_Update tests game update loop
func TestEbitengineGame_Update(t *testing.T) {
	window := &EbitengineWindow{}
	game := &EbitengineGame{
		window: window,
	}
	
	// Test update without emulator function
	err := game.Update()
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}
	
	// Test update with emulator function
	updateCalled := false
	window.emulatorUpdateFunc = func() error {
		updateCalled = true
		return nil
	}
	
	err = game.Update()
	if err != nil {
		t.Fatalf("Update with emulator function failed: %v", err)
	}
	
	if !updateCalled {
		t.Error("Emulator update function should have been called")
	}
}

// TestEbitengineGame_Layout tests game layout calculations
func TestEbitengineGame_Layout(t *testing.T) {
	game := &EbitengineGame{}
	
	// Test layout calculation
	screenWidth, screenHeight := game.Layout(800, 600)
	
	// Layout should return the same dimensions passed in
	if screenWidth != 800 || screenHeight != 600 {
		t.Errorf("Expected layout 800x600, got %dx%d", screenWidth, screenHeight)
	}
	
	// Verify game dimensions were updated
	if game.windowWidth != 800 || game.windowHeight != 600 {
		t.Errorf("Game window dimensions not updated correctly: %dx%d", game.windowWidth, game.windowHeight)
	}
}

// TestEbitengineWindow_WindowOperations tests basic window operations
func TestEbitengineWindow_WindowOperations(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle: "Test Window",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Initial Title", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Test title change
	window.SetTitle("New Title")
	ebitengineWindow := window.(*EbitengineWindow)
	if ebitengineWindow.title != "New Title" {
		t.Errorf("Title not updated correctly: expected 'New Title', got '%s'", ebitengineWindow.title)
	}
	
	// Test size retrieval
	width, height := window.GetSize()
	if width != 800 || height != 600 {
		t.Errorf("Size not correct: expected 800x600, got %dx%d", width, height)
	}
	
	// Test should close (initially false)
	if window.ShouldClose() {
		t.Error("Window should not initially be marked for closing")
	}
	
	// Test cleanup
	err = window.Cleanup()
	if err != nil {
		t.Fatalf("Window cleanup failed: %v", err)
	}
	
	// After cleanup, window should be marked for closing
	if !window.ShouldClose() {
		t.Error("Window should be marked for closing after cleanup")
	}
}

// TestEbitengineBackend_BackendProperties tests backend property methods
func TestEbitengineBackend_BackendProperties(t *testing.T) {
	backend := NewEbitengineBackend()
	
	// Test name
	if backend.GetName() != "Ebitengine" {
		t.Errorf("Expected backend name 'Ebitengine', got '%s'", backend.GetName())
	}
	
	// Test headless property (should be false by default)
	if backend.IsHeadless() {
		t.Error("Backend should not be headless by default")
	}
	
	// Test with headless config
	config := Config{Headless: true}
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	if !backend.IsHeadless() {
		t.Error("Backend should be headless when configured as such")
	}
}

// Mock error type for testing
type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

// TestEbitengineWindow_PollEvents tests event polling
func TestEbitengineWindow_PollEvents(t *testing.T) {
	window := &EbitengineWindow{
		events: []InputEvent{
			{Type: InputEventTypeKey, Key: KeyEscape, Pressed: true},
			{Type: InputEventTypeButton, Button: ButtonA, Pressed: true},
		},
	}
	
	// First poll should return events
	events := window.PollEvents()
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}
	
	// Second poll should return empty slice
	events = window.PollEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 events after clearing, got %d", len(events))
	}
}

// TestEbitengineWindow_SwapBuffers tests buffer swapping
func TestEbitengineWindow_SwapBuffers(t *testing.T) {
	window := &EbitengineWindow{}
	
	// SwapBuffers should not fail (it's a no-op in Ebitengine)
	window.SwapBuffers()
}

// TestEbitengineBackend_Cleanup tests backend cleanup
func TestEbitengineBackend_Cleanup(t *testing.T) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle: "Test Window",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	// Backend should be initialized
	if !backend.(*EbitengineBackend).initialized {
		t.Error("Backend should be initialized")
	}
	
	// Cleanup should succeed
	err = backend.Cleanup()
	if err != nil {
		t.Fatalf("Backend cleanup failed: %v", err)
	}
	
	// Backend should no longer be initialized
	if backend.(*EbitengineBackend).initialized {
		t.Error("Backend should not be initialized after cleanup")
	}
}

// Benchmark tests for performance validation
func BenchmarkEbitengineWindow_RenderFrame(b *testing.B) {
	backend := NewEbitengineBackend()
	
	config := Config{
		WindowTitle: "Benchmark Window",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		b.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Benchmark Game", 800, 600)
	if err != nil {
		b.Fatalf("Window creation failed: %v", err)
	}
	
	// Create test frame buffer
	var frameBuffer [256 * 240]uint32
	for i := 0; i < len(frameBuffer); i++ {
		frameBuffer[i] = 0xFF0000FF // Red
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err = window.RenderFrame(frameBuffer)
		if err != nil {
			b.Fatalf("RenderFrame failed: %v", err)
		}
	}
}

func BenchmarkEbitengineGame_Update(b *testing.B) {
	window := &EbitengineWindow{}
	game := &EbitengineGame{
		window: window,
	}
	
	// Simple update function
	window.emulatorUpdateFunc = func() error {
		return nil
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err := game.Update()
		if err != nil {
			b.Fatalf("Update failed: %v", err)
		}
	}
}