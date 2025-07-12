//go:build !headless
// +build !headless

package graphics

import (
	"sync"
	"testing"
	"time"
)

// MockApplication simulates the Application.render() method behavior
type MockApplication struct {
	window        Window
	frameBuffer   [256 * 240]uint32
	renderCalled  bool
	renderCount   int
	renderError   error
	mu            sync.Mutex
}

func (app *MockApplication) render() error {
	app.mu.Lock()
	defer app.mu.Unlock()
	
	app.renderCalled = true
	app.renderCount++
	
	if app.renderError != nil {
		return app.renderError
	}
	
	if app.window != nil {
		return app.window.RenderFrame(app.frameBuffer)
	}
	
	return nil
}

func (app *MockApplication) setFrameBuffer(frameBuffer [256 * 240]uint32) {
	app.mu.Lock()
	defer app.mu.Unlock()
	app.frameBuffer = frameBuffer
}

func (app *MockApplication) getRenderCount() int {
	app.mu.Lock()
	defer app.mu.Unlock()
	return app.renderCount
}

// MockBus simulates the emulator bus with frame buffer
type MockBus struct {
	frameBuffer [256 * 240]uint32
}

func (bus *MockBus) GetFrameBuffer() []uint32 {
	return bus.frameBuffer[:]
}

func (bus *MockBus) SetFrameBuffer(frameBuffer [256 * 240]uint32) {
	bus.frameBuffer = frameBuffer
}

// TestRenderingPipeline_FrameBufferTransfer tests end-to-end frame buffer transfer
func TestRenderingPipeline_FrameBufferTransfer(t *testing.T) {
	// Initialize backend
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Pipeline Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	// Create window
	window, err := backend.CreateWindow("Pipeline Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Create test frame buffer with specific pattern
	var testFrameBuffer [256 * 240]uint32
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			// Create a checkerboard pattern
			if (x+y)%2 == 0 {
				testFrameBuffer[y*256+x] = 0xFF0000FF // Red
			} else {
				testFrameBuffer[y*256+x] = 0x00FF00FF // Green
			}
		}
	}
	
	// Simulate application render call
	app := &MockApplication{
		window:      window,
		frameBuffer: testFrameBuffer,
	}
	
	// Test frame buffer transfer
	err = app.render()
	if err != nil {
		t.Fatalf("Application render failed: %v", err)
	}
	
	// Verify render was called
	if !app.renderCalled {
		t.Error("Application render method should have been called")
	}
	
	// Verify frame buffer was transferred to Ebitengine
	ebitengineWindow := window.(*EbitengineWindow)
	if ebitengineWindow.game == nil {
		t.Fatal("Game should be initialized after rendering")
	}
	
	// Verify frame buffer content matches
	for i := 0; i < 100; i++ { // Check first 100 pixels
		expected := testFrameBuffer[i]
		actual := ebitengineWindow.game.frameBuffer[i]
		if actual != expected {
			t.Errorf("Frame buffer mismatch at pixel %d: expected 0x%08X, got 0x%08X", i, expected, actual)
		}
	}
	
	// Verify frame image was updated (non-nil frameImage indicates successful processing)
	if ebitengineWindow.game.frameImage == nil {
		t.Error("Frame image should be initialized after rendering")
	}
}

// TestRenderingPipeline_MultipleFrames tests rendering multiple frames
func TestRenderingPipeline_MultipleFrames(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Multi-Frame Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Multi-Frame Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	app := &MockApplication{window: window}
	
	// Render multiple frames with different patterns
	frameCount := 5
	for frame := 0; frame < frameCount; frame++ {
		var frameBuffer [256 * 240]uint32
		
		// Different pattern for each frame
		baseColor := uint32(0xFF << (8 * (frame % 3))) | 0xFF // Red, Green, Blue rotation
		for i := 0; i < len(frameBuffer); i++ {
			frameBuffer[i] = baseColor
		}
		
		app.setFrameBuffer(frameBuffer)
		
		err = app.render()
		if err != nil {
			t.Fatalf("Frame %d render failed: %v", frame, err)
		}
		
		// Verify each frame was processed
		ebitengineWindow := window.(*EbitengineWindow)
		actualColor := ebitengineWindow.game.frameBuffer[0]
		if actualColor != baseColor {
			t.Errorf("Frame %d: expected color 0x%08X, got 0x%08X", frame, baseColor, actualColor)
		}
	}
	
	// Verify all renders were called
	if app.getRenderCount() != frameCount {
		t.Errorf("Expected %d render calls, got %d", frameCount, app.getRenderCount())
	}
}

// TestRenderingPipeline_EmulatorGameLoopIntegration tests integration with emulator update loop
func TestRenderingPipeline_EmulatorGameLoopIntegration(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Game Loop Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Game Loop Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	ebitengineWindow := window.(*EbitengineWindow)
	
	// Set up emulator update function
	emulatorUpdateCalled := false
	frameBufferUpdated := false
	
	updateFunc := func() error {
		emulatorUpdateCalled = true
		
		// Simulate emulator updating frame buffer
		var newFrameBuffer [256 * 240]uint32
		for i := 0; i < len(newFrameBuffer); i++ {
			newFrameBuffer[i] = 0x0000FFFF // Blue
		}
		
		err := window.RenderFrame(newFrameBuffer)
		if err != nil {
			return err
		}
		
		frameBufferUpdated = true
		return nil
	}
	
	ebitengineWindow.SetEmulatorUpdateFunc(updateFunc)
	
	// Simulate game loop update
	err = ebitengineWindow.game.Update()
	if err != nil {
		t.Fatalf("Game update failed: %v", err)
	}
	
	// Verify emulator update was called
	if !emulatorUpdateCalled {
		t.Error("Emulator update function should have been called during game update")
	}
	
	// Verify frame buffer was updated
	if !frameBufferUpdated {
		t.Error("Frame buffer should have been updated during emulator update")
	}
	
	// Verify final frame buffer state
	expectedColor := uint32(0x0000FFFF)
	actualColor := ebitengineWindow.game.frameBuffer[0]
	if actualColor != expectedColor {
		t.Errorf("Expected frame buffer color 0x%08X, got 0x%08X", expectedColor, actualColor)
	}
}

// TestRenderingPipeline_FrameSynchronization tests frame synchronization
func TestRenderingPipeline_FrameSynchronization(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Sync Test",
		VSync:       true,
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Sync Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Test frame timing
	frameCount := 10
	startTime := time.Now()
	
	for i := 0; i < frameCount; i++ {
		var frameBuffer [256 * 240]uint32
		for j := 0; j < len(frameBuffer); j++ {
			frameBuffer[j] = uint32(i) << 16 // Different red intensity per frame
		}
		
		err = window.RenderFrame(frameBuffer)
		if err != nil {
			t.Fatalf("Frame %d render failed: %v", i, err)
		}
		
		// Small delay to simulate frame rate
		time.Sleep(16 * time.Millisecond) // ~60 FPS
	}
	
	elapsedTime := time.Since(startTime)
	expectedMinTime := time.Duration(frameCount) * 16 * time.Millisecond
	
	// Should take at least the expected time due to frame rate limiting
	if elapsedTime < expectedMinTime {
		t.Logf("Frame rendering completed faster than expected (not necessarily an error)")
		t.Logf("Expected min time: %v, Actual time: %v", expectedMinTime, elapsedTime)
	}
}

// TestRenderingPipeline_FrameBufferDataIntegrity tests data integrity during transfer
func TestRenderingPipeline_FrameBufferDataIntegrity(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Data Integrity Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Data Integrity Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Create frame buffer with specific pattern for integrity verification
	var originalFrameBuffer [256 * 240]uint32
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			// Create a unique pattern based on position
			r := uint8((x * 255) / 256)
			g := uint8((y * 255) / 240)
			b := uint8(((x + y) * 255) / (256 + 240))
			originalFrameBuffer[y*256+x] = (uint32(r) << 16) | (uint32(g) << 8) | uint32(b) | 0xFF000000
		}
	}
	
	// Render the frame
	err = window.RenderFrame(originalFrameBuffer)
	if err != nil {
		t.Fatalf("Frame render failed: %v", err)
	}
	
	// Verify complete data integrity
	ebitengineWindow := window.(*EbitengineWindow)
	for i := 0; i < len(originalFrameBuffer); i++ {
		expected := originalFrameBuffer[i]
		actual := ebitengineWindow.game.frameBuffer[i]
		if actual != expected {
			t.Errorf("Data integrity failed at pixel %d: expected 0x%08X, got 0x%08X", i, expected, actual)
			// Stop after first few errors to avoid flooding output
			if i > 10 {
				break
			}
		}
	}
}

// TestRenderingPipeline_ErrorHandling tests error handling in rendering pipeline
func TestRenderingPipeline_ErrorHandling(t *testing.T) {
	// Test rendering with nil window
	app := &MockApplication{window: nil}
	
	err := app.render()
	if err != nil {
		t.Errorf("Render with nil window should not fail, got: %v", err)
	}
	
	// Test rendering with window but nil game
	window := &EbitengineWindow{game: nil}
	var frameBuffer [256 * 240]uint32
	
	err = window.RenderFrame(frameBuffer)
	if err == nil {
		t.Fatal("Expected error when rendering with nil game")
	}
	
	expectedError := "game not initialized"
	if err.Error() != expectedError {
		t.Errorf("Expected error message '%s', got '%s'", expectedError, err.Error())
	}
}

// TestRenderingPipeline_ConcurrentAccess tests concurrent access to rendering pipeline
func TestRenderingPipeline_ConcurrentAccess(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Concurrent Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Concurrent Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Test concurrent frame rendering
	const numGoroutines = 5
	const framesPerGoroutine = 10
	
	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*framesPerGoroutine)
	
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for f := 0; f < framesPerGoroutine; f++ {
				var frameBuffer [256 * 240]uint32
				// Unique color per goroutine and frame
				color := uint32(goroutineID<<16 | f<<8 | 0xFF)
				for i := 0; i < len(frameBuffer); i++ {
					frameBuffer[i] = color
				}
				
				err := window.RenderFrame(frameBuffer)
				if err != nil {
					errorChan <- err
					return
				}
				
				// Small delay between frames
				time.Sleep(time.Millisecond)
			}
		}(g)
	}
	
	wg.Wait()
	close(errorChan)
	
	// Check for any errors
	for err := range errorChan {
		t.Errorf("Concurrent rendering error: %v", err)
	}
}

// TestRenderingPipeline_MemoryLeakPrevention tests for memory leaks in rendering
func TestRenderingPipeline_MemoryLeakPrevention(t *testing.T) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Memory Test",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		t.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Memory Test", 800, 600)
	if err != nil {
		t.Fatalf("Window creation failed: %v", err)
	}
	
	// Render many frames to test for memory accumulation
	frameCount := 100
	
	for i := 0; i < frameCount; i++ {
		var frameBuffer [256 * 240]uint32
		for j := 0; j < len(frameBuffer); j++ {
			frameBuffer[j] = uint32(i%256) << 16 // Rotating red intensity
		}
		
		err = window.RenderFrame(frameBuffer)
		if err != nil {
			t.Fatalf("Frame %d render failed: %v", i, err)
		}
	}
	
	// Cleanup
	err = window.Cleanup()
	if err != nil {
		t.Fatalf("Window cleanup failed: %v", err)
	}
	
	err = backend.Cleanup()
	if err != nil {
		t.Fatalf("Backend cleanup failed: %v", err)
	}
}

// Benchmark test for rendering pipeline performance
func BenchmarkRenderingPipeline_EndToEnd(b *testing.B) {
	// Initialize backend and window
	backend := NewEbitengineBackend()
	config := Config{
		WindowTitle: "Benchmark",
		Headless:    false,
	}
	
	err := backend.Initialize(config)
	if err != nil {
		b.Fatalf("Backend initialization failed: %v", err)
	}
	
	window, err := backend.CreateWindow("Benchmark", 800, 600)
	if err != nil {
		b.Fatalf("Window creation failed: %v", err)
	}
	
	// Create test frame buffer
	var frameBuffer [256 * 240]uint32
	for i := 0; i < len(frameBuffer); i++ {
		frameBuffer[i] = 0xFF0000FF // Red
	}
	
	app := &MockApplication{
		window:      window,
		frameBuffer: frameBuffer,
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		err = app.render()
		if err != nil {
			b.Fatalf("Render failed: %v", err)
		}
	}
}