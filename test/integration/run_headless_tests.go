package integration

import (
	"testing"
	
	"gones/internal/input"
)

// TestHeadlessFrameworkValidation validates that the headless testing framework is working
func TestHeadlessFrameworkValidation(t *testing.T) {
	t.Run("Headless helper creation", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless helper: %v", err)
		}
		defer helper.Cleanup()

		if helper.app == nil {
			t.Error("Application not created")
		}

		if helper.app.GetBus() == nil {
			t.Error("Bus not available")
		}

		t.Log("Headless emulator helper created successfully")
	})

	t.Run("Mock ROM loading", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless helper: %v", err)
		}
		defer helper.Cleanup()

		// Simple test program
		program := []uint8{
			0xA9, 0x42, // LDA #$42
			0x85, 0x00, // STA $00
			0x4C, 0x04, 0x80, // JMP $8004 (infinite loop)
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load mock ROM: %v", err)
		}

		t.Log("Mock ROM loaded successfully")
	})

	t.Run("Frame execution", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless helper: %v", err)
		}
		defer helper.Cleanup()

		// Simple test program
		program := []uint8{
			0xEA, // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load mock ROM: %v", err)
		}

		err = helper.RunHeadlessFrames(5)
		if err != nil {
			t.Fatalf("Failed to run headless frames: %v", err)
		}

		metrics := helper.GetPerformanceMetrics()
		if metrics["frames_executed"].(int) != 5 {
			t.Errorf("Expected 5 frames, got %d", metrics["frames_executed"].(int))
		}

		t.Logf("Successfully executed %d frames", metrics["frames_executed"].(int))
	})

	t.Run("Frame buffer validation", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless helper: %v", err)
		}
		defer helper.Cleanup()

		program := []uint8{
			0xEA, // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load mock ROM: %v", err)
		}

		err = helper.RunHeadlessFrames(3)
		if err != nil {
			t.Fatalf("Failed to run headless frames: %v", err)
		}

		fbResult := helper.ValidateFrameBuffer()
		if !fbResult.ExpectedDimensions {
			t.Error("Frame buffer does not have expected NES dimensions")
		}

		if fbResult.PixelCount != 256*240 {
			t.Errorf("Expected %d pixels, got %d", 256*240, fbResult.PixelCount)
		}

		t.Logf("Frame buffer validation: %s", fbResult.ValidationMessage)
	})

	t.Run("Input simulation basic", func(t *testing.T) {
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless helper: %v", err)
		}
		defer helper.Cleanup()

		program := []uint8{
			0xEA, // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load mock ROM: %v", err)
		}

		// Schedule some input events
		helper.ScheduleInputEvent(1, input.A, true, 2)
		helper.ScheduleInputEvent(1, input.A, false, 4)

		err = helper.RunHeadlessFrames(6)
		if err != nil {
			t.Fatalf("Failed to run frames with input: %v", err)
		}

		t.Log("Input simulation completed successfully")
	})
}

// TestDisplayValidationFramework validates display testing capabilities
func TestDisplayValidationFramework(t *testing.T) {
	t.Run("Display helper creation", func(t *testing.T) {
		helper, err := NewDisplayValidationTestHelper()
		if err != nil {
			t.Fatalf("Failed to create display helper: %v", err)
		}
		defer helper.Cleanup()

		if helper.HeadlessEmulatorTestHelper == nil {
			t.Error("Headless helper not available")
		}

		t.Log("Display validation helper created successfully")
	})

	t.Run("Pattern validation test", func(t *testing.T) {
		helper, err := NewDisplayValidationTestHelper()
		if err != nil {
			t.Fatalf("Failed to create display helper: %v", err)
		}
		defer helper.Cleanup()

		program := []uint8{
			0xA9, 0x08, // LDA #$08
			0x8D, 0x01, 0x20, // STA $2001 (enable background)
			0x4C, 0x05, 0x80, // JMP $8005
		}

		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load ROM: %v", err)
		}

		err = helper.RunHeadlessFrames(3)
		if err != nil {
			t.Fatalf("Failed to run frames: %v", err)
		}

		// Test pattern validation (should be mostly one color initially)
		result := helper.ValidatePattern("solid_color", 50.0)
		t.Logf("Pattern validation: %s", result.Message)

		// Test color validation
		colorResult := helper.ValidateColors([]uint32{}, "basic_colors")
		t.Logf("Color validation: %s", colorResult.Message)
	})
}

// TestInputValidationFramework validates input testing capabilities
func TestInputValidationFramework(t *testing.T) {
	t.Run("Input helper creation", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input helper: %v", err)
		}
		defer helper.Cleanup()

		if helper.HeadlessEmulatorTestHelper == nil {
			t.Error("Headless helper not available")
		}

		t.Log("Input test helper created successfully")
	})

	t.Run("Input sequence creation", func(t *testing.T) {
		helper, err := NewInputTestHelper()
		if err != nil {
			t.Fatalf("Failed to create input helper: %v", err)
		}
		defer helper.Cleanup()

		sequence := helper.CreateInputSequence("test_sequence", "Test sequence")
		sequence.AddInputEvent(1, 1, input.A, true)
		sequence.AddInputEvent(3, 1, input.A, false)

		if len(sequence.Events) != 2 {
			t.Errorf("Expected 2 events, got %d", len(sequence.Events))
		}

		t.Logf("Input sequence created with %d events", len(sequence.Events))
	})
}

// TestEndToEndFramework validates end-to-end testing capabilities
func TestEndToEndFramework(t *testing.T) {
	t.Run("End-to-end helper creation", func(t *testing.T) {
		helper, err := NewEndToEndTestHelper()
		if err != nil {
			t.Fatalf("Failed to create end-to-end helper: %v", err)
		}
		defer helper.Cleanup()

		if helper.HeadlessEmulatorTestHelper == nil {
			t.Error("Headless helper not available")
		}

		t.Log("End-to-end test helper created successfully")
	})

	t.Run("Scenario creation", func(t *testing.T) {
		helper, err := NewEndToEndTestHelper()
		if err != nil {
			t.Fatalf("Failed to create end-to-end helper: %v", err)
		}
		defer helper.Cleanup()

		scenario := helper.CreateBasicDisplayScenario()
		if scenario.Name == "" {
			t.Error("Scenario name not set")
		}

		if len(scenario.ROM) == 0 {
			t.Error("Scenario ROM not set")
		}

		t.Logf("Created scenario: %s with %d bytes of ROM", scenario.Name, len(scenario.ROM))
	})
}

// TestEnvironmentValidationFramework validates environment testing capabilities
func TestEnvironmentValidationFramework(t *testing.T) {
	t.Run("Environment helper creation", func(t *testing.T) {
		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create environment helper: %v", err)
		}
		defer helper.Cleanup()

		if helper.HeadlessEmulatorTestHelper == nil {
			t.Error("Headless helper not available")
		}

		t.Log("Environment test helper created successfully")
	})

	t.Run("Environment checks", func(t *testing.T) {
		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create environment helper: %v", err)
		}
		defer helper.Cleanup()

		results := helper.RunEnvironmentChecks()
		if len(results) == 0 {
			t.Error("No environment checks were run")
		}

		passedCount := 0
		for _, result := range results {
			if result.Passed {
				passedCount++
			}
			t.Logf("Environment check %s: %s", result.TestName, result.Message)
		}

		t.Logf("Environment checks: %d/%d passed", passedCount, len(results))
	})
}

// TestHeadlessSystemIntegration tests the complete headless system integration
func TestHeadlessSystemIntegration(t *testing.T) {
	t.Run("Complete headless workflow", func(t *testing.T) {
		// Test the complete workflow: create helper, load ROM, run frames, validate
		helper, err := NewHeadlessEmulatorTestHelper()
		if err != nil {
			t.Fatalf("Failed to create headless helper: %v", err)
		}
		defer helper.Cleanup()

		// Create a comprehensive test ROM
		program := []uint8{
			// Initialize PPU
			0xA9, 0x80, // LDA #$80
			0x8D, 0x00, 0x20, // STA $2000 (PPUCTRL)
			0xA9, 0x1E, // LDA #$1E
			0x8D, 0x01, 0x20, // STA $2001 (PPUMASK)

			// Initialize APU
			0xA9, 0x0F, // LDA #$0F
			0x8D, 0x15, 0x40, // STA $4015 (APU_STATUS)

			// Set test marker in memory
			0xA9, 0x42, // LDA #$42
			0x85, 0x10, // STA $10

			// Main loop
			0x4C, 0x12, 0x80, // JMP $8012
		}

		// Load and execute
		err = helper.LoadMockROM(program)
		if err != nil {
			t.Fatalf("Failed to load comprehensive ROM: %v", err)
		}

		// Schedule some input events
		helper.ScheduleInputEvent(1, input.A, true, 2)
		helper.ScheduleInputEvent(1, input.Start, true, 5)
		helper.ScheduleInputEvent(1, input.A, false, 7)
		helper.ScheduleInputEvent(1, input.Start, false, 9)

		// Run the emulator
		err = helper.RunHeadlessFrames(15)
		if err != nil {
			t.Fatalf("Failed to run comprehensive test: %v", err)
		}

		// Validate all aspects
		fbResult := helper.ValidateFrameBuffer()
		audioResult := helper.ValidateAudio()
		metrics := helper.GetPerformanceMetrics()

		// Validate results
		if !fbResult.ExpectedDimensions {
			t.Error("Frame buffer validation failed")
		}

		if fbResult.PixelCount != 256*240 {
			t.Errorf("Wrong pixel count: expected %d, got %d", 256*240, fbResult.PixelCount)
		}

		if audioResult.SampleCount == 0 {
			t.Log("Note: No audio samples captured (expected in headless mode)")
		}

		if metrics["frames_executed"].(int) != 15 {
			t.Errorf("Wrong frame count: expected 15, got %d", metrics["frames_executed"].(int))
		}

		// Validate memory state
		bus := helper.app.GetBus()
		if bus != nil && bus.Memory != nil {
			testValue := bus.Memory.Read(0x0010)
			if testValue != 0x42 {
				t.Errorf("Memory test failed: expected 0x42, got 0x%02X", testValue)
			}
		}

		executionTimeMs := metrics["execution_time_ms"].(int64)
		t.Logf("Complete workflow test successful:")
		t.Logf("  Frames: %d", metrics["frames_executed"].(int))
		t.Logf("  Execution time: %d ms", executionTimeMs)
		t.Logf("  Frame buffer: %s", fbResult.ValidationMessage)
		t.Logf("  Audio samples: %d", audioResult.SampleCount)

		// Performance validation
		if executionTimeMs > 10000 { // Should complete in under 10 seconds
			t.Errorf("Test took too long: %d ms", executionTimeMs)
		}
	})
}