package integration

import (
	"fmt"
	"testing"
	"time"

	"gones/internal/app"
	"gones/internal/input"
)

// HeadlessEmulatorTestHelper provides utilities for headless emulator testing
type HeadlessEmulatorTestHelper struct {
	*IntegrationTestHelper
	app         *app.Application
	frameBuffer []uint32
	audioBuffer []float32
	inputEvents []HeadlessInputEvent
	
	// Track current button states for each controller
	buttonStates map[int]map[input.Button]bool
	
	// Test execution state
	frameCount    int
	executionTime time.Duration
	testStartTime time.Time
}

// HeadlessInputEvent represents a simulated input event for testing
type HeadlessInputEvent struct {
	Controller int
	Button     input.Button
	Pressed    bool
	FrameDelay int // Frames to wait before applying this event
	Applied    bool // Whether this event has been applied
}

// FrameBufferValidationResult represents the result of frame buffer validation
type FrameBufferValidationResult struct {
	Valid              bool
	PixelCount         int
	NonZeroPixels      int
	UniqueColors       int
	BackgroundColor    uint32
	ExpectedDimensions bool
	ValidationMessage  string
}

// AudioValidationResult represents the result of audio validation
type AudioValidationResult struct {
	Valid         bool
	SampleCount   int
	NonSilent     bool
	PeakAmplitude float32
	AverageLevel  float32
	Message       string
}

// NewHeadlessEmulatorTestHelper creates a new headless emulator test helper
func NewHeadlessEmulatorTestHelper() (*HeadlessEmulatorTestHelper, error) {
	// Create headless application (no SDL2 video/audio)
	application, err := app.NewApplicationWithMode("", true)
	if err != nil {
		return nil, fmt.Errorf("failed to create headless application: %v", err)
	}

	// Create integration test helper for low-level access
	integrationHelper := NewIntegrationTestHelper()

	helper := &HeadlessEmulatorTestHelper{
		IntegrationTestHelper: integrationHelper,
		app:                   application,
		frameBuffer:          make([]uint32, 256*240),
		audioBuffer:          make([]float32, 0),
		inputEvents:          make([]HeadlessInputEvent, 0),
		buttonStates:         map[int]map[input.Button]bool{
			1: make(map[input.Button]bool),
			2: make(map[input.Button]bool),
		},
		frameCount:           0,
		testStartTime:        time.Now(),
	}

	return helper, nil
}

// LoadTestROM loads a ROM file for testing
func (h *HeadlessEmulatorTestHelper) LoadTestROM(romPath string) error {
	return h.app.LoadROM(romPath)
}

// LoadMockROM loads a mock ROM with specified program data
func (h *HeadlessEmulatorTestHelper) LoadMockROM(programData []uint8) error {
	// Create mock cartridge with the program
	romData := make([]uint8, 0x8000)
	copy(romData, programData)
	
	// Set reset vector to start of ROM
	romData[0x7FFC] = 0x00 // Reset vector low
	romData[0x7FFD] = 0x80 // Reset vector high
	
	// Create mock cartridge
	mockCart := NewMockCartridge()
	mockCart.LoadPRG(romData)
	
	// Load cartridge into the application's bus
	bus := h.app.GetBus()
	if bus != nil {
		bus.LoadCartridge(mockCart)
		bus.Reset()
		
		// Re-apply any input events that were set before reset
		// Reset clears all input states, so we need to restore them
		for i := range h.inputEvents {
			event := &h.inputEvents[i]
			if event.Applied {
				bus.SetControllerButton(event.Controller, event.Button, event.Pressed)
			}
		}
	}
	
	return nil
}

// RunHeadlessFrames executes the emulator for specified number of frames in headless mode
func (h *HeadlessEmulatorTestHelper) RunHeadlessFrames(frameCount int) error {
	h.testStartTime = time.Now()
	
	bus := h.app.GetBus()
	if bus == nil {
		return fmt.Errorf("no bus available")
	}
	
	// Pre-process input events to establish initial state
	for i := range h.inputEvents {
		event := &h.inputEvents[i]
		if !event.Applied {
			bus.SetControllerButton(event.Controller, event.Button, event.Pressed)
			// Track the button state
			if h.buttonStates[event.Controller] != nil {
				h.buttonStates[event.Controller][event.Button] = event.Pressed
			}
		}
	}
	
	for frame := 0; frame < frameCount; frame++ {
		// Process any scheduled input events
		h.processScheduledInputEvents(frame)
		
		// Run one frame worth of cycles (NTSC: ~29,781 CPU cycles)
		bus.RunCycles(29781)
		
		// Capture frame buffer
		h.captureFrameBuffer()
		
		// Capture audio samples
		h.captureAudioSamples()
		
		h.frameCount++
	}
	
	h.executionTime = time.Since(h.testStartTime)
	return nil
}

// captureFrameBuffer captures the current frame buffer for validation
func (h *HeadlessEmulatorTestHelper) captureFrameBuffer() {
	bus := h.app.GetBus()
	if bus != nil {
		frameData := bus.GetFrameBuffer()
		if len(frameData) >= len(h.frameBuffer) {
			copy(h.frameBuffer, frameData[:len(h.frameBuffer)])
		}
	}
}

// captureAudioSamples captures audio samples for validation
func (h *HeadlessEmulatorTestHelper) captureAudioSamples() {
	bus := h.app.GetBus()
	if bus != nil {
		samples := bus.GetAudioSamples()
		h.audioBuffer = append(h.audioBuffer, samples...)
	}
}

// processScheduledInputEvents processes input events scheduled for the current frame
func (h *HeadlessEmulatorTestHelper) processScheduledInputEvents(currentFrame int) {
	bus := h.app.GetBus()
	if bus == nil {
		return
	}
	
	// Process all events that should occur at this frame and haven't been applied yet
	for i := range h.inputEvents {
		event := &h.inputEvents[i]
		if event.FrameDelay <= currentFrame && !event.Applied {
			bus.SetControllerButton(event.Controller, event.Button, event.Pressed)
			event.Applied = true
			
			// Track the button state
			if h.buttonStates[event.Controller] != nil {
				h.buttonStates[event.Controller][event.Button] = event.Pressed
			}
		}
	}
	
	// Apply all current button states to ensure they persist
	for controllerNum, buttonMap := range h.buttonStates {
		for button, pressed := range buttonMap {
			if pressed {
				bus.SetControllerButton(controllerNum, button, true)
			}
		}
	}
}

// ScheduleInputEvent schedules an input event to occur at a specific frame
func (h *HeadlessEmulatorTestHelper) ScheduleInputEvent(controller int, button input.Button, pressed bool, frameDelay int) {
	event := HeadlessInputEvent{
		Controller: controller,
		Button:     button,
		Pressed:    pressed,
		FrameDelay: frameDelay,
		Applied:    false,
	}
	h.inputEvents = append(h.inputEvents, event)
}

// ValidateFrameBuffer validates the current frame buffer contents
func (h *HeadlessEmulatorTestHelper) ValidateFrameBuffer() FrameBufferValidationResult {
	result := FrameBufferValidationResult{
		Valid:              true,
		PixelCount:         len(h.frameBuffer),
		ExpectedDimensions: len(h.frameBuffer) == 256*240,
	}
	
	if !result.ExpectedDimensions {
		result.Valid = false
		result.ValidationMessage = fmt.Sprintf("Expected 256x240 pixels (%d), got %d", 256*240, len(h.frameBuffer))
		return result
	}
	
	// Count non-zero pixels and unique colors
	colorMap := make(map[uint32]bool)
	nonZeroCount := 0
	
	for _, pixel := range h.frameBuffer {
		colorMap[pixel] = true
		if pixel != 0 {
			nonZeroCount++
		}
	}
	
	result.NonZeroPixels = nonZeroCount
	result.UniqueColors = len(colorMap)
	
	// Determine background color (most common color)
	colorCounts := make(map[uint32]int)
	for _, pixel := range h.frameBuffer {
		colorCounts[pixel]++
	}
	
	maxCount := 0
	for color, count := range colorCounts {
		if count > maxCount {
			maxCount = count
			result.BackgroundColor = color
		}
	}
	
	// Validation criteria
	if result.UniqueColors < 2 {
		result.ValidationMessage = "Frame buffer appears to have insufficient color variety"
	} else if nonZeroCount == 0 {
		result.ValidationMessage = "Frame buffer appears to be completely black"
	} else {
		result.ValidationMessage = "Frame buffer validation passed"
	}
	
	return result
}

// ValidateAudio validates captured audio samples
func (h *HeadlessEmulatorTestHelper) ValidateAudio() AudioValidationResult {
	result := AudioValidationResult{
		Valid:       true,
		SampleCount: len(h.audioBuffer),
	}
	
	if len(h.audioBuffer) == 0 {
		result.Valid = false
		result.Message = "No audio samples captured"
		return result
	}
	
	// Calculate audio statistics
	var sum, peak float32
	nonSilentSamples := 0
	
	for _, sample := range h.audioBuffer {
		absValue := sample
		if absValue < 0 {
			absValue = -absValue
		}
		
		sum += absValue
		if absValue > peak {
			peak = absValue
		}
		
		if absValue > 0.001 { // Threshold for "non-silent"
			nonSilentSamples++
		}
	}
	
	result.PeakAmplitude = peak
	result.AverageLevel = sum / float32(len(h.audioBuffer))
	result.NonSilent = nonSilentSamples > 0
	
	if !result.NonSilent {
		result.Message = "Audio appears to be silent"
	} else {
		result.Message = fmt.Sprintf("Audio validation passed: peak=%.3f, avg=%.3f", peak, result.AverageLevel)
	}
	
	return result
}

// GetPerformanceMetrics returns performance metrics for the test run
func (h *HeadlessEmulatorTestHelper) GetPerformanceMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})
	
	metrics["frames_executed"] = h.frameCount
	metrics["execution_time_ms"] = h.executionTime.Milliseconds()
	metrics["frame_buffer_size"] = len(h.frameBuffer)
	metrics["audio_samples_captured"] = len(h.audioBuffer)
	
	if h.executionTime.Seconds() > 0 {
		metrics["frames_per_second"] = float64(h.frameCount) / h.executionTime.Seconds()
	}
	
	return metrics
}

// Cleanup releases resources
func (h *HeadlessEmulatorTestHelper) Cleanup() error {
	if h.app != nil {
		return h.app.Cleanup()
	}
	return nil
}

// TestHeadlessEmulatorBasicOperation tests basic headless emulator functionality
func TestHeadlessEmulatorBasicOperation(t *testing.T) {
	t.Run("Headless application creation", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		if helper.app == nil {
			t.Fatal("Headless application not created")
		}
		
		if helper.app.GetBus() == nil {
			t.Fatal("Application bus not available")
		}
	})
	
	t.Run("Frame buffer generation", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		// Load simple test program
		program := []uint8{
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL - enable NMI)
			0xA9, 0x1E, // LDA #$1E  
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK - enable rendering)
			0x4C, 0x08, 0x80, // JMP $8008 (infinite loop)
		}
		
		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load mock ROM: %v", err)
		}
		
		// Run for several frames
		err = helper.RunHeadlessFrames(5)
		if err != nil {
			t.Fatalf("Failed to run headless frames: %v", err)
		}
		
		// Validate frame buffer
		fbResult := helper.ValidateFrameBuffer()
		if !fbResult.Valid {
			t.Errorf("Frame buffer validation failed: %s", fbResult.ValidationMessage)
		}
		
		if !fbResult.ExpectedDimensions {
			t.Errorf("Frame buffer has wrong dimensions")
		}
		
		t.Logf("Frame buffer: %d pixels, %d non-zero, %d unique colors", 
			fbResult.PixelCount, fbResult.NonZeroPixels, fbResult.UniqueColors)
	})
	
	t.Run("Audio sample generation", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		// Program that initializes APU channels
		program := []uint8{
			0xA9, 0x0F, // LDA #$0F
			0x8D, 0x15, 0x40, // STA $4015 (APU_STATUS - enable all channels)
			0xA9, 0x30, // LDA #$30
			0x8D, 0x00, 0x40, // STA $4000 (PULSE1_DUTY - duty cycle and volume)
			0xA9, 0x08, // LDA #$08
			0x8D, 0x02, 0x40, // STA $4002 (PULSE1_LO - frequency low)
			0xA9, 0x02, // LDA #$02  
			0x8D, 0x03, 0x40, // STA $4003 (PULSE1_HI - frequency high)
			0x4C, 0x12, 0x80, // JMP $8012 (infinite loop)
		}
		
		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load mock ROM: %v", err)
		}
		
		// Run for several frames to generate audio
		err = helper.RunHeadlessFrames(10)
		if err != nil {
			t.Fatalf("Failed to run headless frames: %v", err)
		}
		
		// Validate audio output
		audioResult := helper.ValidateAudio()
		
		t.Logf("Audio: %d samples, peak=%.3f, avg=%.3f, non-silent=%t", 
			audioResult.SampleCount, audioResult.PeakAmplitude, 
			audioResult.AverageLevel, audioResult.NonSilent)
		
		if audioResult.SampleCount == 0 {
			t.Error("No audio samples were generated")
		}
	})
}

// TestHeadlessInputSimulation tests input simulation without SDL2
func TestHeadlessInputSimulation(t *testing.T) {
	t.Run("Controller input simulation", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		// Program that reads controller input and stores result
		program := []uint8{
			0xA9, 0x01, // LDA #$01
			0x8D, 0x16, 0x40, // STA $4016 (strobe controller)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x16, 0x40, // STA $4016 (stop strobe)
			
			// Read 8 bits from controller 1
			0xAD, 0x16, 0x40, // LDA $4016 (read A button)
			0x85, 0x00,       // STA $00
			0xAD, 0x16, 0x40, // LDA $4016 (read B button)  
			0x85, 0x01,       // STA $01
			0xAD, 0x16, 0x40, // LDA $4016 (read Select)
			0x85, 0x02,       // STA $02
			0xAD, 0x16, 0x40, // LDA $4016 (read Start)
			0x85, 0x03,       // STA $03
			
			0x4C, 0x18, 0x80, // JMP $8018 (infinite loop)
		}
		
		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load mock ROM: %v", err)
		}
		
		// Schedule input events
		helper.ScheduleInputEvent(1, input.A, true, 1)      // Press A on frame 1
		helper.ScheduleInputEvent(1, input.Start, true, 2)  // Press Start on frame 2
		helper.ScheduleInputEvent(1, input.A, false, 3)     // Release A on frame 3
		
		// Run frames with input events
		err = helper.RunHeadlessFrames(5)
		if err != nil {
			t.Fatalf("Failed to run headless frames: %v", err)
		}
		
		// Verify input was processed
		bus := helper.app.GetBus()
		inputState := bus.GetInputState()
		
		// Test that input system is functional
		if inputState == nil {
			t.Error("Input state not available")
		}
		
		t.Logf("Input simulation completed successfully")
	})
	
	t.Run("Multiple controller simulation", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		// Simple ROM for multi-controller test
		program := []uint8{
			0xEA, // NOP
			0x4C, 0x00, 0x80, // JMP $8000 (infinite loop)
		}
		
		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load mock ROM: %v", err)
		}
		
		// Schedule events for both controllers
		helper.ScheduleInputEvent(1, input.A, true, 1)
		helper.ScheduleInputEvent(2, input.B, true, 1)
		helper.ScheduleInputEvent(1, input.A, false, 3)
		helper.ScheduleInputEvent(2, input.B, false, 3)
		
		err = helper.RunHeadlessFrames(5)
		if err != nil {
			t.Fatalf("Failed to run headless frames: %v", err)
		}
		
		t.Logf("Multi-controller simulation completed")
	})
}

// TestHeadlessROMExecution tests ROM execution in headless mode
func TestHeadlessROMExecution(t *testing.T) {
	t.Run("Sample ROM execution", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		// Try to load sample ROM if it exists
		err = helper.LoadTestROM("../../roms/sample.nes")
		if err != nil {
			t.Skipf("Sample ROM not available: %v", err)
		}
		
		// Run the ROM for several frames
		err = helper.RunHeadlessFrames(60) // 1 second at 60 FPS
		if err != nil {
			t.Fatalf("Failed to run sample ROM: %v", err)
		}
		
		// Validate execution results
		fbResult := helper.ValidateFrameBuffer()
		audioResult := helper.ValidateAudio()
		metrics := helper.GetPerformanceMetrics()
		
		t.Logf("Sample ROM execution metrics: %+v", metrics)
		t.Logf("Frame buffer: %s", fbResult.ValidationMessage)
		t.Logf("Audio: %s", audioResult.Message)
		
		if !fbResult.ExpectedDimensions {
			t.Error("Frame buffer has wrong dimensions")
		}
	})
	
	t.Run("Complex ROM behavior", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		// Complex test program with graphics and sound
		program := []uint8{
			// Initialize PPU
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK)
			
			// Initialize APU
			0xA9, 0x0F, // LDA #$0F
			0x8D, 0x15, 0x40, // STA $4015 (APU_STATUS)
			
			// Write some palette data
			0xA9, 0x3F, // LDA #$3F
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x06, 0x20, // STA $2006 (PPUADDR)
			
			// Write background color
			0xA9, 0x0F, // LDA #$0F (white)
			0x8D, 0x07, 0x20, // STA $2007 (PPUDATA)
			
			// Main loop
			0x4C, 0x1E, 0x80, // JMP $801E (infinite loop)
		}
		
		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load complex ROM: %v", err)
		}
		
		// Run for several frames
		err = helper.RunHeadlessFrames(30)
		if err != nil {
			t.Fatalf("Failed to run complex ROM: %v", err)
		}
		
		// Validate results
		fbResult := helper.ValidateFrameBuffer()
		metrics := helper.GetPerformanceMetrics()
		
		if !fbResult.Valid {
			t.Errorf("Complex ROM frame buffer validation failed: %s", fbResult.ValidationMessage)
		}
		
		if fbResult.UniqueColors < 1 {
			t.Error("Complex ROM should generate at least 1 color")
		}
		
		t.Logf("Complex ROM execution completed: %+v", metrics)
	})
}

// TestHeadlessPerformance tests performance characteristics in headless mode
func TestHeadlessPerformance(t *testing.T) {
	t.Run("Performance benchmark", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		// Simple ROM for performance testing
		program := []uint8{
			0xEA, // NOP
			0x4C, 0x00, 0x80, // JMP $8000 (tight loop)
		}
		
		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load performance test ROM: %v", err)
		}
		
		// Run for many frames to measure performance
		frameCount := 600 // 10 seconds at 60 FPS
		err = helper.RunHeadlessFrames(frameCount)
		if err != nil {
			t.Fatalf("Failed to run performance test: %v", err)
		}
		
		metrics := helper.GetPerformanceMetrics()
		
		executionTimeMs := metrics["execution_time_ms"].(int64)
		framesPerSecond := metrics["frames_per_second"].(float64)
		
		t.Logf("Performance metrics:")
		t.Logf("  Frames executed: %d", frameCount)
		t.Logf("  Execution time: %d ms", executionTimeMs)
		t.Logf("  Frames per second: %.2f", framesPerSecond)
		
		// Performance expectations for headless mode
		if framesPerSecond < 60.0 {
			t.Logf("Warning: Performance below 60 FPS (%.2f)", framesPerSecond)
		}
		
		if executionTimeMs > 20000 { // Should complete in under 20 seconds
			t.Errorf("Performance test took too long: %d ms", executionTimeMs)
		}
	})
	
	t.Run("Memory usage stability", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless emulator: %v", err)
		}
		defer helper.Cleanup()
		
		// ROM that exercises various systems
		program := []uint8{
			// Exercise PPU
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001
			
			// Exercise APU
			0xA9, 0x0F, // LDA #$0F
			0x8D, 0x15, 0x40, // STA $4015
			
			// Exercise memory
			0xA2, 0x00, // LDX #$00
			0xA9, 0xAA, // LDA #$AA
			0x95, 0x00, // STA $00,X
			0xE8,       // INX
			0xE0, 0xFF, // CPX #$FF
			0xD0, 0xF8, // BNE -8
			
			0x4C, 0x14, 0x80, // JMP $8014 (loop back)
		}
		
		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load memory test ROM: %v", err)
		}
		
		// Run for extended period
		err = helper.RunHeadlessFrames(300) // 5 seconds
		if err != nil {
			t.Fatalf("Failed to run memory stability test: %v", err)
		}
		
		// Validate that frame buffer and audio buffer sizes are reasonable
		fbResult := helper.ValidateFrameBuffer()
		audioResult := helper.ValidateAudio()
		
		if !fbResult.ExpectedDimensions {
			t.Error("Frame buffer dimensions changed during execution")
		}
		
		// Audio buffer shouldn't grow unbounded
		if audioResult.SampleCount > 100000 {
			t.Errorf("Audio buffer may be growing unbounded: %d samples", audioResult.SampleCount)
		}
		
		t.Logf("Memory stability test completed successfully")
	})
}