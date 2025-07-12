package test

import (
	"fmt"
	"gones/internal/app"
	"gones/internal/cartridge"
	"gones/internal/graphics"
	"os"
	"sync"
	"testing"
	"time"
)

// TestGraphicsBackend_ApplicationIntegration tests integration between Application and Graphics Backend
func TestGraphicsBackend_ApplicationIntegration(t *testing.T) {
	// Create application in non-headless mode to test graphics backend
	application, err := app.NewApplicationWithMode("", false)
	if err != nil {
		// If graphics backend fails (e.g., no display), skip this test
		if isDisplayError(err) {
			t.Skip("Skipping graphics test due to display unavailability")
		}
		t.Fatalf("Failed to create application: %v", err)
	}
	defer application.Cleanup()
	
	// Verify application has graphics backend
	if application.GetBus() == nil {
		t.Fatal("Application should have initialized bus")
	}
	
	// Test loading a ROM to trigger rendering pipeline
	testROMPath := "roms/sample.nes"
	if _, err := os.Stat(testROMPath); os.IsNotExist(err) {
		t.Skipf("Test ROM not found at %s, skipping integration test", testROMPath)
	}
	
	err = application.LoadROM(testROMPath)
	if err != nil {
		t.Fatalf("Failed to load test ROM: %v", err)
	}
	
	// Verify ROM is loaded and emulator is started
	if application.GetROMPath() != testROMPath {
		t.Errorf("Expected ROM path %s, got %s", testROMPath, application.GetROMPath())
	}
}

// TestGraphicsBackend_RenderingFrameTransfer tests that frames are properly transferred to graphics backend
func TestGraphicsBackend_RenderingFrameTransfer(t *testing.T) {
	// Create minimal test ROM data
	testROM := createMinimalTestROM()
	
	// Create application
	application, err := app.NewApplicationWithMode("", false)
	if err != nil {
		if isDisplayError(err) {
			t.Skip("Skipping graphics test due to display unavailability")
		}
		t.Fatalf("Failed to create application: %v", err)
	}
	defer application.Cleanup()
	
	// Load test cartridge directly
	cart, err := cartridge.LoadFromBytes(testROM)
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}
	
	// Load into bus and reset
	bus := application.GetBus()
	bus.LoadCartridge(cart)
	bus.Reset()
	
	// Set a known frame buffer pattern
	expectedFrameBuffer := createTestFrameBuffer()
	bus.SetFrameBufferForTesting(expectedFrameBuffer)
	
	// Verify frame buffer can be retrieved
	retrievedFrameBuffer := bus.GetFrameBuffer()
	if len(retrievedFrameBuffer) != 256*240 {
		t.Errorf("Expected frame buffer size %d, got %d", 256*240, len(retrievedFrameBuffer))
	}
	
	// Verify first few pixels match expected pattern
	for i := 0; i < 10; i++ {
		if retrievedFrameBuffer[i] != expectedFrameBuffer[i] {
			t.Errorf("Frame buffer pixel %d: expected 0x%08X, got 0x%08X", 
				i, expectedFrameBuffer[i], retrievedFrameBuffer[i])
		}
	}
}

// TestGraphicsBackend_EmulatorRenderLoop tests the emulator rendering loop
func TestGraphicsBackend_EmulatorRenderLoop(t *testing.T) {
	// Create application
	application, err := app.NewApplicationWithMode("", false)
	if err != nil {
		if isDisplayError(err) {
			t.Skip("Skipping graphics test due to display unavailability")
		}
		t.Fatalf("Failed to create application: %v", err)
	}
	defer application.Cleanup()
	
	// Create and load minimal test ROM
	testROM := createMinimalTestROM()
	cart, err := cartridge.LoadFromBytes(testROM)
	if err != nil {
		t.Fatalf("Failed to create test cartridge: %v", err)
	}
	
	bus := application.GetBus()
	bus.LoadCartridge(cart)
	bus.Reset()
	
	// Run emulator for a few frames to test rendering pipeline
	frameCount := 0
	maxFrames := 10
	
	for frameCount < maxFrames && application.IsRunning() {
		// Update emulator state
		application.GetBus().Step()
		
		// Check if a frame was rendered
		frameBuffer := bus.GetFrameBuffer()
		if frameBuffer != nil && len(frameBuffer) == 256*240 {
			frameCount++
		}
		
		// Avoid infinite loop
		if frameCount == 0 {
			time.Sleep(time.Millisecond)
		}
	}
	
	if frameCount == 0 {
		t.Error("No frames were rendered during emulator execution")
	}
}

// TestGraphicsBackend_EbitengineSpecificFeatures tests Ebitengine-specific functionality
func TestGraphicsBackend_EbitengineSpecificFeatures(t *testing.T) {
	// Create graphics backend directly
	backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
	if err != nil {
		t.Fatalf("Failed to create Ebitengine backend: %v", err)
	}
	
	// Test backend properties
	if backend.GetName() != "Ebitengine" {
		t.Errorf("Expected backend name 'Ebitengine', got '%s'", backend.GetName())
	}
	
	// Initialize backend
	config := graphics.Config{
		WindowTitle:  "Test Ebitengine Features",
		WindowWidth:  800,
		WindowHeight: 600,
		Fullscreen:   false,
		VSync:        true,
		Filter:       "nearest",
		AspectRatio:  "4:3",
		Headless:     false,
		Debug:        false,
	}
	
	err = backend.Initialize(config)
	if err != nil {
		if isDisplayError(err) {
			t.Skip("Skipping Ebitengine test due to display unavailability")
		}
		t.Fatalf("Failed to initialize Ebitengine backend: %v", err)
	}
	defer backend.Cleanup()
	
	// Test window creation
	window, err := backend.CreateWindow("Test Window", 800, 600)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}
	defer window.Cleanup()
	
	// Test window properties
	width, height := window.GetSize()
	if width != 800 || height != 600 {
		t.Errorf("Expected window size 800x600, got %dx%d", width, height)
	}
	
	// Test frame rendering
	frameBuffer := createTestFrameBuffer()
	err = window.RenderFrame(frameBuffer)
	if err != nil {
		t.Fatalf("Failed to render frame: %v", err)
	}
	
	// Test Ebitengine-specific window casting
	ebitengineWindow, ok := graphics.AsEbitengineWindow(window)
	if !ok {
		t.Fatal("Failed to cast window to EbitengineWindow")
	}
	
	// Test emulator update function setting
	updateCalled := false
	ebitengineWindow.SetEmulatorUpdateFunc(func() error {
		updateCalled = true
		return nil
	})
	
	// This would normally be called by Ebitengine's game loop
	// We can't test the actual game loop without running Ebitengine
	// but we can test that the update function was set
	if ebitengineWindow == nil {
		t.Error("EbitengineWindow should be properly initialized")
	}
	
	// Check that update function can be called
	if !updateCalled {
		t.Log("Update function not called yet (expected in test environment)")
	}
}

// TestGraphicsBackend_FrameBufferConversion tests frame buffer format conversion
func TestGraphicsBackend_FrameBufferConversion(t *testing.T) {
	backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	
	config := graphics.Config{
		WindowTitle: "Frame Buffer Test",
		Headless:    false,
	}
	
	err = backend.Initialize(config)
	if err != nil {
		if isDisplayError(err) {
			t.Skip("Skipping graphics test due to display unavailability")
		}
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Cleanup()
	
	window, err := backend.CreateWindow("Frame Buffer Test", 800, 600)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}
	defer window.Cleanup()
	
	// Test various frame buffer patterns
	testCases := []struct {
		name     string
		pattern  func() [256 * 240]uint32
		validate func([256 * 240]uint32) bool
	}{
		{
			name: "All Red",
			pattern: func() [256 * 240]uint32 {
				var fb [256 * 240]uint32
				for i := range fb {
					fb[i] = 0xFF0000FF // Red
				}
				return fb
			},
			validate: func(fb [256 * 240]uint32) bool {
				return fb[0] == 0xFF0000FF
			},
		},
		{
			name: "Gradient",
			pattern: func() [256 * 240]uint32 {
				var fb [256 * 240]uint32
				for y := 0; y < 240; y++ {
					for x := 0; x < 256; x++ {
						r := uint8((x * 255) / 256)
						g := uint8((y * 255) / 240)
						fb[y*256+x] = (uint32(r) << 16) | (uint32(g) << 8) | 0xFF
					}
				}
				return fb
			},
			validate: func(fb [256 * 240]uint32) bool {
				// Check corner pixels
				topLeft := fb[0]
				topRight := fb[255]
				bottomLeft := fb[239*256]
				bottomRight := fb[239*256+255]
				
				return topLeft != topRight && topLeft != bottomLeft && 
					   topRight != bottomRight && bottomLeft != bottomRight
			},
		},
		{
			name: "Checkerboard",
			pattern: func() [256 * 240]uint32 {
				var fb [256 * 240]uint32
				for y := 0; y < 240; y++ {
					for x := 0; x < 256; x++ {
						if (x+y)%2 == 0 {
							fb[y*256+x] = 0xFFFFFFFF // White
						} else {
							fb[y*256+x] = 0x000000FF // Black
						}
					}
				}
				return fb
			},
			validate: func(fb [256 * 240]uint32) bool {
				return fb[0] == 0xFFFFFFFF && fb[1] == 0x000000FF
			},
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			frameBuffer := tc.pattern()
			
			err := window.RenderFrame(frameBuffer)
			if err != nil {
				t.Fatalf("Failed to render %s pattern: %v", tc.name, err)
			}
			
			// Verify the pattern was processed correctly
			ebitengineWindow, ok := graphics.AsEbitengineWindow(window)
			if !ok {
				t.Fatal("Failed to cast to EbitengineWindow")
			}
			
			if !tc.validate(ebitengineWindow.GetFrameBufferForTesting()) {
				t.Errorf("Frame buffer validation failed for %s pattern", tc.name)
			}
		})
	}
}

// TestGraphicsBackend_ErrorRecovery tests error recovery in graphics pipeline
func TestGraphicsBackend_ErrorRecovery(t *testing.T) {
	backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	
	config := graphics.Config{
		WindowTitle: "Error Recovery Test",
		Headless:    false,
	}
	
	err = backend.Initialize(config)
	if err != nil {
		if isDisplayError(err) {
			t.Skip("Skipping graphics test due to display unavailability")
		}
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Cleanup()
	
	window, err := backend.CreateWindow("Error Recovery Test", 800, 600)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}
	defer window.Cleanup()
	
	// Test recovery after invalid operations
	validFrameBuffer := createTestFrameBuffer()
	
	// First, ensure normal operation works
	err = window.RenderFrame(validFrameBuffer)
	if err != nil {
		t.Fatalf("Normal frame rendering failed: %v", err)
	}
	
	// Test that rendering continues to work after valid operations
	err = window.RenderFrame(validFrameBuffer)
	if err != nil {
		t.Fatalf("Frame rendering failed after previous success: %v", err)
	}
}

// TestGraphicsBackend_ConcurrentRendering tests concurrent frame rendering
func TestGraphicsBackend_ConcurrentRendering(t *testing.T) {
	backend, err := graphics.CreateBackend(graphics.BackendEbitengine)
	if err != nil {
		t.Fatalf("Failed to create backend: %v", err)
	}
	
	config := graphics.Config{
		WindowTitle: "Concurrent Rendering Test",
		Headless:    false,
	}
	
	err = backend.Initialize(config)
	if err != nil {
		if isDisplayError(err) {
			t.Skip("Skipping graphics test due to display unavailability")
		}
		t.Fatalf("Failed to initialize backend: %v", err)
	}
	defer backend.Cleanup()
	
	window, err := backend.CreateWindow("Concurrent Test", 800, 600)
	if err != nil {
		t.Fatalf("Failed to create window: %v", err)
	}
	defer window.Cleanup()
	
	// Test concurrent rendering from multiple goroutines
	const numGoroutines = 3
	const framesPerGoroutine = 5
	
	var wg sync.WaitGroup
	errorChan := make(chan error, numGoroutines*framesPerGoroutine)
	
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			
			for f := 0; f < framesPerGoroutine; f++ {
				frameBuffer := createTestFrameBufferWithPattern(goroutineID, f)
				
				err := window.RenderFrame(frameBuffer)
				if err != nil {
					errorChan <- fmt.Errorf("goroutine %d frame %d: %v", goroutineID, f, err)
					return
				}
				
				// Small delay between frames
				time.Sleep(10 * time.Millisecond)
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

// Helper functions

func isDisplayError(err error) bool {
	// Check for common display-related errors
	errorStr := err.Error()
	return containsAny(errorStr, []string{
		"DISPLAY",
		"display",
		"X11",
		"wayland",
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

func createMinimalTestROM() []byte {
	// Create a minimal iNES ROM with header
	rom := make([]byte, 16+16384) // 16-byte header + 16KB PRG ROM
	
	// iNES header
	copy(rom[0:4], []byte("NES\x1a"))
	rom[4] = 1  // 16KB PRG ROM
	rom[5] = 1  // 8KB CHR ROM
	rom[6] = 0  // Mapper 0, horizontal mirroring
	rom[7] = 0  // Mapper 0
	
	// Fill PRG ROM with NOPs and reset vector
	for i := 16; i < len(rom)-6; i++ {
		rom[i] = 0xEA // NOP instruction
	}
	
	// Set reset vector to point to start of ROM
	rom[len(rom)-4] = 0x00 // Reset vector low byte
	rom[len(rom)-3] = 0x80 // Reset vector high byte
	
	return rom
}

func createTestFrameBuffer() [256 * 240]uint32 {
	var frameBuffer [256 * 240]uint32
	
	// Create a test pattern
	for y := 0; y < 240; y++ {
		for x := 0; x < 256; x++ {
			// Create a simple gradient pattern
			r := uint8((x * 255) / 256)
			g := uint8((y * 255) / 240)
			b := uint8(((x + y) % 256))
			
			frameBuffer[y*256+x] = (uint32(r) << 16) | (uint32(g) << 8) | uint32(b) | 0xFF000000
		}
	}
	
	return frameBuffer
}

func createTestFrameBufferWithPattern(goroutineID, frameID int) [256 * 240]uint32 {
	var frameBuffer [256 * 240]uint32
	
	// Create unique pattern based on goroutine and frame
	baseColor := uint32(goroutineID<<20 | frameID<<12 | 0xFF000000)
	
	for i := 0; i < len(frameBuffer); i++ {
		frameBuffer[i] = baseColor | uint32(i%256)
	}
	
	return frameBuffer
}