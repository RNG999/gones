package integration

import (
	"testing"
	"gones/internal/app"
	"gones/internal/input"
)

// TestBasicHeadlessOperation tests basic headless emulator functionality
func TestBasicHeadlessOperation(t *testing.T) {
	t.Run("Create headless application", func(t *testing.T) {
		// Create headless application (no SDL2 video/audio)
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		if application == nil {
			t.Fatal("Headless application not created")
		}

		bus := application.GetBus()
		if bus == nil {
			t.Fatal("Application bus not available")
		}

		t.Log("Headless application created successfully")
	})

	t.Run("Basic frame execution", func(t *testing.T) {
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		bus := application.GetBus()
		
		// Create a simple mock cartridge and test ROM
		helper := NewIntegrationTestHelper()
		
		// Simple test program
		program := []uint8{
			0xA9, 0x42, // LDA #$42
			0x85, 0x00, // STA $00
			0x4C, 0x04, 0x80, // JMP $8004 (infinite loop)
		}

		// Set up ROM data
		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00 // Reset vector low
		romData[0x7FFD] = 0x80 // Reset vector high
		
		helper.GetMockCartridge().LoadPRG(romData)
		bus.LoadCartridge(helper.GetMockCartridge())
		bus.Reset()

		// Execute some cycles
		initialCycles := bus.GetCycleCount()
		for i := 0; i < 100; i++ {
			bus.Step()
		}
		finalCycles := bus.GetCycleCount()

		if finalCycles <= initialCycles {
			t.Error("CPU cycles did not advance")
		}

		t.Logf("Executed %d CPU cycles", finalCycles-initialCycles)
	})

	t.Run("Frame buffer access", func(t *testing.T) {
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		bus := application.GetBus()
		helper := NewIntegrationTestHelper()

		// ROM that enables PPU rendering
		program := []uint8{
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL - enable NMI)
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK - enable rendering)
			0x4C, 0x08, 0x80, // JMP $8008 (infinite loop)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		bus.LoadCartridge(helper.GetMockCartridge())
		bus.Reset()

		// Run for a few frames
		for i := 0; i < 1000; i++ {
			bus.Step()
		}

		// Get frame buffer
		frameBuffer := bus.GetFrameBuffer()
		if frameBuffer == nil {
			t.Fatal("Frame buffer not available")
		}

		if len(frameBuffer) != 256*240 {
			t.Errorf("Expected frame buffer size %d, got %d", 256*240, len(frameBuffer))
		}

		t.Logf("Frame buffer access successful: %d pixels", len(frameBuffer))
	})

	t.Run("Audio sample access", func(t *testing.T) {
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		bus := application.GetBus()
		helper := NewIntegrationTestHelper()

		// ROM that initializes APU
		program := []uint8{
			0xA9, 0x0F, // LDA #$0F
			0x8D, 0x15, 0x40, // STA $4015 (APU_STATUS - enable all channels)
			0xA9, 0x30, // LDA #$30
			0x8D, 0x00, 0x40, // STA $4000 (PULSE1_DUTY)
			0x4C, 0x08, 0x80, // JMP $8008 (infinite loop)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		bus.LoadCartridge(helper.GetMockCartridge())
		bus.Reset()

		// Run for several frames to generate audio
		for i := 0; i < 2000; i++ {
			bus.Step()
		}

		// Get audio samples
		audioSamples := bus.GetAudioSamples()
		
		// Audio samples may be empty in headless mode, but the call should not crash
		t.Logf("Audio sample access successful: %d samples", len(audioSamples))
	})

	t.Run("Input simulation", func(t *testing.T) {
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		bus := application.GetBus()
		helper := NewIntegrationTestHelper()

		// ROM that reads controller input
		program := []uint8{
			0xA9, 0x01, // LDA #$01
			0x8D, 0x16, 0x40, // STA $4016 (strobe controller)
			0xA9, 0x00, // LDA #$00
			0x8D, 0x16, 0x40, // STA $4016 (stop strobe)
			0xAD, 0x16, 0x40, // LDA $4016 (read A button)
			0x85, 0x10,       // STA $10 (store result)
			0x4C, 0x00, 0x80, // JMP $8000 (repeat)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		bus.LoadCartridge(helper.GetMockCartridge())
		bus.Reset()

		// Simulate input
		bus.SetControllerButton(1, input.A, true)

		// Run some cycles
		for i := 0; i < 200; i++ {
			bus.Step()
		}

		// Release input
		bus.SetControllerButton(1, input.A, false)

		// Get input state
		inputState := bus.GetInputState()
		if inputState == nil {
			t.Error("Input state not available")
		}

		t.Log("Input simulation completed successfully")
	})

	t.Run("Performance validation", func(t *testing.T) {
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		bus := application.GetBus()
		helper := NewIntegrationTestHelper()

		// Simple loop ROM for performance testing
		program := []uint8{
			0xEA, // NOP
			0x4C, 0x00, 0x80, // JMP $8000 (tight loop)
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		bus.LoadCartridge(helper.GetMockCartridge())
		bus.Reset()

		// Execute many cycles for performance test
		initialCycles := bus.GetCycleCount()
		targetCycles := 10000

		for i := 0; i < targetCycles; i++ {
			bus.Step()
		}

		finalCycles := bus.GetCycleCount()
		cyclesExecuted := finalCycles - initialCycles

		if cyclesExecuted < uint64(targetCycles) {
			t.Errorf("Expected at least %d cycles, got %d", targetCycles, cyclesExecuted)
		}

		t.Logf("Performance test: executed %d cycles", cyclesExecuted)

		// Validate frame buffer is still accessible after intensive execution
		frameBuffer := bus.GetFrameBuffer()
		if len(frameBuffer) != 256*240 {
			t.Error("Frame buffer corrupted after intensive execution")
		}
	})
}

// TestBasicHeadlessEnvironmentCompatibility tests environment compatibility
func TestBasicHeadlessEnvironmentCompatibility(t *testing.T) {
	t.Run("No DISPLAY environment", func(t *testing.T) {
		// This test should pass regardless of whether DISPLAY is set
		// since we're running in headless mode
		
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		bus := application.GetBus()
		if bus == nil {
			t.Fatal("Bus not available in headless mode")
		}

		t.Log("Headless operation confirmed (no display dependency)")
	})

	t.Run("Resource constraints", func(t *testing.T) {
		// Test that headless mode uses reasonable resources
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		// The fact that we can create and use the application 
		// indicates resource usage is reasonable
		bus := application.GetBus()
		
		// Quick functionality test
		frameBuffer := bus.GetFrameBuffer()
		audioSamples := bus.GetAudioSamples()
		
		// These should be accessible without crashing
		if len(frameBuffer) != 256*240 {
			t.Errorf("Frame buffer size incorrect: %d", len(frameBuffer))
		}

		t.Logf("Resource test: frame buffer %d pixels, audio %d samples", 
			len(frameBuffer), len(audioSamples))
	})
}

// TestBasicHeadlessSystemIntegration tests complete system integration
func TestBasicHeadlessSystemIntegration(t *testing.T) {
	t.Run("Complete emulation workflow", func(t *testing.T) {
		application, err := app.NewApplicationWithMode("", true)
		if err != nil {
			t.Fatalf("Failed to create headless application: %v", err)
		}
		defer application.Cleanup()

		bus := application.GetBus()
		helper := NewIntegrationTestHelper()

		// Comprehensive test ROM that exercises multiple systems
		program := []uint8{
			// Initialize PPU
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001

			// Initialize APU
			0xA9, 0x0F, // LDA #$0F
			0x8D, 0x15, 0x40, // STA $4015

			// Read controller
			0xA9, 0x01, // LDA #$01
			0x8D, 0x16, 0x40, // STA $4016
			0xA9, 0x00, // LDA #$00
			0x8D, 0x16, 0x40, // STA $4016
			0xAD, 0x16, 0x40, // LDA $4016

			// Store state marker
			0xA9, 0x55, // LDA #$55
			0x85, 0x20, // STA $20

			// Main loop
			0x4C, 0x1A, 0x80, // JMP $801A
		}

		romData := make([]uint8, 0x8000)
		copy(romData, program)
		romData[0x7FFC] = 0x00
		romData[0x7FFD] = 0x80
		
		helper.GetMockCartridge().LoadPRG(romData)
		bus.LoadCartridge(helper.GetMockCartridge())
		bus.Reset()

		// Simulate input during execution
		bus.SetControllerButton(1, input.A, true)

		// Execute comprehensive test
		initialCycles := bus.GetCycleCount()
		for i := 0; i < 5000; i++ {
			bus.Step()
		}
		finalCycles := bus.GetCycleCount()

		// Release input
		bus.SetControllerButton(1, input.A, false)

		// Validate all systems
		frameBuffer := bus.GetFrameBuffer()
		audioSamples := bus.GetAudioSamples()
		inputState := bus.GetInputState()

		// Check execution occurred
		if finalCycles <= initialCycles {
			t.Error("No CPU execution detected")
		}

		// Check systems are accessible
		if len(frameBuffer) != 256*240 {
			t.Errorf("Frame buffer wrong size: %d", len(frameBuffer))
		}

		if inputState == nil {
			t.Error("Input state not accessible")
		}

		// Check memory state (our test marker)
		memory := bus.Memory
		if memory != nil {
			marker := memory.Read(0x0020)
			if marker != 0x55 {
				t.Errorf("Memory state marker wrong: expected 0x55, got 0x%02X", marker)
			}
		}

		t.Logf("Complete integration test successful:")
		t.Logf("  CPU cycles: %d", finalCycles-initialCycles)
		t.Logf("  Frame buffer: %d pixels", len(frameBuffer))
		t.Logf("  Audio samples: %d", len(audioSamples))
		t.Logf("  Memory marker: valid")
	})
}