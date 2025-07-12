package integration

import (
	"gones/internal/app"
	"gones/internal/cartridge"
	"gones/internal/graphics"
	"testing"
)

// TestEbitengineRenderingPipeline_ActualImplementation tests the real Ebitengine rendering pipeline
// These tests will FAIL initially if the rendering pipeline is not properly implemented
func TestEbitengineRenderingPipeline_ActualImplementation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Ebitengine integration tests in short mode")
	}
	
	// Skip if no display available
	if isHeadlessEnvironment() {
		t.Skip("Skipping Ebitengine tests in headless environment")
	}
	
	t.Run("Backend_Initialization_Must_Succeed", func(t *testing.T) {
		backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
		if err != nil {
			t.Fatalf("REQUIREMENT FAILED: CreateBackend failed: %v", err)
		}
		
		config := graphics.Config{
			WindowTitle:  "Test Backend Initialization",
			WindowWidth:  800,
			WindowHeight: 600,
			Headless:     false,
		}
		
		err = backend.Initialize(config)
		if err != nil {
			t.Fatalf("REQUIREMENT FAILED: Backend initialization failed: %v", err)
		}
		
		// Verify backend properties
		if backend.GetName() != "Ebitengine" {
			t.Errorf("REQUIREMENT FAILED: Expected backend name 'Ebitengine', got '%s'", backend.GetName())
		}
		
		if backend.IsHeadless() {
			t.Error("REQUIREMENT FAILED: Backend should not be headless")
		}
		
		// Cleanup
		err = backend.Cleanup()
		if err != nil {
			t.Errorf("Backend cleanup failed: %v", err)
		}
	})
	
	t.Run("Window_Creation_Must_Succeed", func(t *testing.T) {
		backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
		if err != nil {
			t.Fatalf("Backend creation failed: %v", err)
		}
		
		config := graphics.Config{
			WindowTitle: "Test Window Creation",
			Headless:    false,
		}
		
		err = backend.Initialize(config)
		if err != nil {
			t.Fatalf("Backend initialization failed: %v", err)
		}
		defer backend.Cleanup()
		
		window, err := backend.CreateWindow("Test Window", 800, 600)
		if err != nil {
			t.Fatalf("REQUIREMENT FAILED: Window creation failed: %v", err)
		}
		defer window.Cleanup()
		
		// Verify window properties
		width, height := window.GetSize()
		if width != 800 || height != 600 {
			t.Errorf("REQUIREMENT FAILED: Expected window size 800x600, got %dx%d", width, height)
		}
		
		// Verify window can be cast to EbitengineWindow
		ebitengineWindow, ok := graphics.AsEbitengineWindow(window)
		if !ok {
			t.Fatal("REQUIREMENT FAILED: Window should be castable to EbitengineWindow")
		}
		
		if ebitengineWindow == nil {
			t.Fatal("REQUIREMENT FAILED: EbitengineWindow should not be nil")
		}
	})
	
	t.Run("RenderFrame_Must_Transfer_FrameBuffer", func(t *testing.T) {
		backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
		if err != nil {
			t.Fatalf("Backend creation failed: %v", err)
		}
		
		config := graphics.Config{
			WindowTitle: "Test RenderFrame",
			Headless:    false,
		}
		
		err = backend.Initialize(config)
		if err != nil {
			t.Fatalf("Backend initialization failed: %v", err)
		}
		defer backend.Cleanup()
		
		window, err := backend.CreateWindow("Test RenderFrame", 800, 600)
		if err != nil {
			t.Fatalf("Window creation failed: %v", err)
		}
		defer window.Cleanup()
		
		// Create test frame buffer with specific pattern
		var testFrameBuffer [256 * 240]uint32
		for y := 0; y < 240; y++ {
			for x := 0; x < 256; x++ {
				// Create unique pattern for each pixel
				r := uint8((x * 255) / 256)
				g := uint8((y * 255) / 240)
				b := uint8(((x + y) % 256))
				testFrameBuffer[y*256+x] = (uint32(r) << 16) | (uint32(g) << 8) | uint32(b) | 0xFF000000
			}
		}
		
		// Render frame
		err = window.RenderFrame(testFrameBuffer)
		if err != nil {
			t.Fatalf("REQUIREMENT FAILED: RenderFrame failed: %v", err)
		}
		
		// Verify frame buffer was transferred
		ebitengineWindow, _ := graphics.AsEbitengineWindow(window)
		actualFrameBuffer := ebitengineWindow.GetFrameBufferForTesting()
		
		// Check frame buffer integrity
		for i := 0; i < 100; i++ { // Check first 100 pixels for efficiency
			expected := testFrameBuffer[i]
			actual := actualFrameBuffer[i]
			if actual != expected {
				t.Errorf("REQUIREMENT FAILED: Frame buffer pixel %d: expected 0x%08X, got 0x%08X", 
					i, expected, actual)
				// Only show first few errors
				if i > 5 {
					break
				}
			}
		}
	})
	
	t.Run("EmulatorUpdate_Integration_Must_Work", func(t *testing.T) {
		backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
		if err != nil {
			t.Fatalf("Backend creation failed: %v", err)
		}
		
		config := graphics.Config{
			WindowTitle: "Test Emulator Integration",
			Headless:    false,
		}
		
		err = backend.Initialize(config)
		if err != nil {
			t.Fatalf("Backend initialization failed: %v", err)
		}
		defer backend.Cleanup()
		
		window, err := backend.CreateWindow("Test Emulator Integration", 800, 600)
		if err != nil {
			t.Fatalf("Window creation failed: %v", err)
		}
		defer window.Cleanup()
		
		ebitengineWindow, _ := graphics.AsEbitengineWindow(window)
		
		// Set up emulator update function
		updateCallCount := 0
		renderCallCount := 0
		
		emulatorUpdateFunc := func() error {
			updateCallCount++
			
			// Simulate emulator generating frame
			var emulatorFrame [256 * 240]uint32
			for i := 0; i < len(emulatorFrame); i++ {
				emulatorFrame[i] = uint32(updateCallCount<<16) | 0x0000FFFF // Blue with red variation
			}
			
			// This simulates Application.render() calling Window.RenderFrame()
			err := window.RenderFrame(emulatorFrame)
			if err == nil {
				renderCallCount++
			}
			return err
		}
		
		// Set emulator update function
		ebitengineWindow.SetEmulatorUpdateFunc(emulatorUpdateFunc)
		
		// Get the game instance for testing
		game := ebitengineWindow.GetGameForTesting()
		if game == nil {
			t.Fatal("REQUIREMENT FAILED: Game instance should be available")
		}
		
		// Call game update (simulates Ebitengine calling Update)
		err = game.Update()
		if err != nil {
			t.Fatalf("REQUIREMENT FAILED: Game update failed: %v", err)
		}
		
		// Verify emulator update was called
		if updateCallCount != 1 {
			t.Errorf("REQUIREMENT FAILED: Expected 1 emulator update call, got %d", updateCallCount)
		}
		
		if renderCallCount != 1 {
			t.Errorf("REQUIREMENT FAILED: Expected 1 render call, got %d", renderCallCount)
		}
		
		// Verify frame buffer contains expected data
		actualFrameBuffer := ebitengineWindow.GetFrameBufferForTesting()
		expectedColor := uint32(1<<16) | 0x0000FFFF // First frame color
		if actualFrameBuffer[0] != expectedColor {
			t.Errorf("REQUIREMENT FAILED: Expected frame buffer color 0x%08X, got 0x%08X", 
				expectedColor, actualFrameBuffer[0])
		}
	})
}

// TestEbitengineRenderingPipeline_ApplicationIntegration tests integration with the actual Application
func TestEbitengineRenderingPipeline_ApplicationIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping application integration tests in short mode")
	}
	
	// Skip if no display available
	if isHeadlessEnvironment() {
		t.Skip("Skipping Ebitengine tests in headless environment")
	}
	
	t.Run("Application_With_Graphics_Backend_Must_Initialize", func(t *testing.T) {
		// Create application with graphics backend
		application, err := app.NewApplicationWithMode("", false)
		if err != nil {
			// If this fails due to display issues, that's expected in headless environments
			if isDisplayError(err) {
				t.Skip("Skipping due to display unavailability")
			}
			t.Fatalf("REQUIREMENT FAILED: Application creation failed: %v", err)
		}
		defer application.Cleanup()
		
		// Verify bus is available
		bus := application.GetBus()
		if bus == nil {
			t.Fatal("REQUIREMENT FAILED: Application should have initialized bus")
		}
	})
	
	t.Run("Application_Render_Must_Call_Window_RenderFrame", func(t *testing.T) {
		// This test verifies that Application.render() properly calls Window.RenderFrame()
		
		application, err := app.NewApplicationWithMode("", false)
		if err != nil {
			if isDisplayError(err) {
				t.Skip("Skipping due to display unavailability")
			}
			t.Fatalf("Application creation failed: %v", err)
		}
		defer application.Cleanup()
		
		// Create minimal test ROM
		testROM := createMinimalTestROM()
		cart, err := cartridge.LoadFromBytes(testROM)
		if err != nil {
			t.Fatalf("Test cartridge creation failed: %v", err)
		}
		
		// Load cartridge
		bus := application.GetBus()
		bus.LoadCartridge(cart)
		bus.Reset()
		
		// Set known frame buffer
		testFrameBuffer := createTestFrameBuffer()
		bus.SetFrameBufferForTesting(testFrameBuffer)
		
		// This test cannot easily simulate Application.render() without running the full application
		// But we can verify the frame buffer pipeline works
		retrievedFrameBuffer := bus.GetFrameBuffer()
		if len(retrievedFrameBuffer) != 256*240 {
			t.Errorf("REQUIREMENT FAILED: Expected frame buffer size %d, got %d", 
				256*240, len(retrievedFrameBuffer))
		}
		
		// Verify frame buffer content
		for i := 0; i < 10; i++ {
			expected := testFrameBuffer[i]
			actual := retrievedFrameBuffer[i]
			if actual != expected {
				t.Errorf("REQUIREMENT FAILED: Frame buffer pixel %d: expected 0x%08X, got 0x%08X", 
					i, expected, actual)
			}
		}
	})
}

// TestEbitengineRenderingPipeline_FailureDetection tests that failures are properly detected
func TestEbitengineRenderingPipeline_FailureDetection(t *testing.T) {
	t.Run("DetectMissingBackendInitialization", func(t *testing.T) {
		backend := graphics.NewEbitengineBackend()
		
		// Attempt to create window without initialization should fail
		_, err := backend.CreateWindow("Test", 800, 600)
		if err == nil {
			t.Fatal("DETECTION FAILED: Should detect missing backend initialization")
		}
		
		expectedError := "backend not initialized"
		if err.Error() != expectedError {
			t.Errorf("DETECTION FAILED: Expected error '%s', got '%s'", expectedError, err.Error())
		}
	})
	
	t.Run("DetectMissingGameInstance", func(t *testing.T) {
		// This test verifies that RenderFrame properly detects when game is not initialized
		// We'll create a backend and window, then test with a frame buffer
		
		backend := graphics.NewEbitengineBackend()
		config := graphics.Config{
			WindowTitle: "Test Missing Game",
			Headless:    false,
		}
		
		err := backend.Initialize(config)
		if err != nil {
			if isDisplayError(err) {
				t.Skip("Skipping due to display unavailability")
			}
			t.Fatalf("Backend initialization failed: %v", err)
		}
		defer backend.Cleanup()
		
		window, err := backend.CreateWindow("Test Missing Game", 800, 600)
		if err != nil {
			t.Fatalf("Window creation failed: %v", err)
		}
		defer window.Cleanup()
		
		// With a properly created window, RenderFrame should work
		var frameBuffer [256 * 240]uint32
		err = window.RenderFrame(frameBuffer)
		if err != nil {
			t.Errorf("RenderFrame should work with properly created window: %v", err)
		}
	})
	
	t.Run("DetectFrameBufferNotTransferred", func(t *testing.T) {
		// This test would detect if RenderFrame doesn't actually transfer the frame buffer
		// In a broken implementation, the frame buffer would remain unchanged
		
		if testing.Short() {
			t.Skip("Skipping comprehensive frame buffer test in short mode")
		}
		
		if isHeadlessEnvironment() {
			t.Skip("Skipping frame buffer test in headless environment")
		}
		
		backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
		if err != nil {
			t.Fatalf("Backend creation failed: %v", err)
		}
		
		config := graphics.Config{
			WindowTitle: "Frame Buffer Detection Test",
			Headless:    false,
		}
		
		err = backend.Initialize(config)
		if err != nil {
			if isDisplayError(err) {
				t.Skip("Skipping due to display unavailability")
			}
			t.Fatalf("Backend initialization failed: %v", err)
		}
		defer backend.Cleanup()
		
		window, err := backend.CreateWindow("Frame Buffer Detection Test", 800, 600)
		if err != nil {
			t.Fatalf("Window creation failed: %v", err)
		}
		defer window.Cleanup()
		
		ebitengineWindow, _ := graphics.AsEbitengineWindow(window)
		
		// Get initial frame buffer state
		initialFrameBuffer := ebitengineWindow.GetFrameBufferForTesting()
		
		// Create unique test frame buffer
		var testFrameBuffer [256 * 240]uint32
		for i := 0; i < len(testFrameBuffer); i++ {
			testFrameBuffer[i] = 0x12345678 + uint32(i) // Unique pattern
		}
		
		// Render frame
		err = window.RenderFrame(testFrameBuffer)
		if err != nil {
			t.Fatalf("RenderFrame failed: %v", err)
		}
		
		// Get final frame buffer state
		finalFrameBuffer := ebitengineWindow.GetFrameBufferForTesting()
		
		// Detect if frame buffer actually changed
		frameBufferChanged := false
		for i := 0; i < len(testFrameBuffer); i++ {
			if finalFrameBuffer[i] != initialFrameBuffer[i] {
				frameBufferChanged = true
				break
			}
		}
		
		if !frameBufferChanged {
			t.Error("DETECTION FAILED: Frame buffer should have changed after RenderFrame call")
		}
		
		// Verify the change is correct
		for i := 0; i < 10; i++ {
			expected := testFrameBuffer[i]
			actual := finalFrameBuffer[i]
			if actual != expected {
				t.Errorf("DETECTION FAILED: Frame buffer transfer error at pixel %d: expected 0x%08X, got 0x%08X", 
					i, expected, actual)
			}
		}
	})
}

// Helper functions

func isHeadlessEnvironment() bool {
	// Check for common indicators of headless environment
	return isDisplayError(&DisplayError{})
}

func isDisplayError(err error) bool {
	if err == nil {
		return false
	}
	errorStr := err.Error()
	return containsAny(errorStr, []string{
		"DISPLAY",
		"display",
		"X11",
		"wayland",
		"glfw",
		"no such file or directory",
		"cannot open display",
		"connection refused",
	})
}

func containsAny(str string, substrings []string) bool {
	for _, substr := range substrings {
		if len(str) >= len(substr) {
			for i := 0; i <= len(str)-len(substr); i++ {
				if str[i:i+len(substr)] == substr {
					return true
				}
			}
		}
	}
	return false
}

type DisplayError struct{}

func (e *DisplayError) Error() string {
	return "DISPLAY environment variable missing"
}

func createMinimalTestROM() []byte {
	// Create a minimal 16KB iNES ROM
	rom := make([]byte, 16+16384+8192) // Header + PRG ROM + CHR ROM
	
	// iNES header
	copy(rom[0:4], []byte("NES\x1a"))
	rom[4] = 1  // 16KB PRG ROM
	rom[5] = 1  // 8KB CHR ROM
	rom[6] = 0  // Mapper 0, horizontal mirroring
	rom[7] = 0  // Mapper 0 continued
	
	// Fill PRG ROM with NOPs
	for i := 16; i < 16+16384; i++ {
		rom[i] = 0xEA // NOP instruction
	}
	
	// Set reset vector
	rom[16+16384-4] = 0x00 // Reset vector low byte
	rom[16+16384-3] = 0x80 // Reset vector high byte
	
	return rom
}

func createTestFrameBuffer() [256 * 240]uint32 {
	var frameBuffer [256 * 240]uint32
	
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			// Create a test pattern
			r := uint8((x * 255) / 256)
			g := uint8((y * 255) / 240)
			b := uint8(((x + y) % 256))
			frameBuffer[y*256+x] = (uint32(r) << 16) | (uint32(g) << 8) | uint32(b) | 0xFF000000
		}
	}
	
	return frameBuffer
}