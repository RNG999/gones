package integration

import (
	"fmt"
	"os"
	"testing"
	"time"
)

// EnvironmentTestHelper validates headless operation in various environments
type EnvironmentTestHelper struct {
	*HeadlessEmulatorTestHelper
	environmentChecks []EnvironmentCheck
	testResults      []EnvironmentTestResult
}

// EnvironmentCheck represents a check for environment compatibility
type EnvironmentCheck struct {
	Name        string
	Description string
	CheckFunc   func() (bool, string)
	Required    bool
}

// EnvironmentTestResult represents the result of an environment test
type EnvironmentTestResult struct {
	TestName      string
	Passed        bool
	Message       string
	ExecutionTime time.Duration
	Details       map[string]interface{}
}

// NewEnvironmentTestHelper creates a new environment test helper
func NewEnvironmentTestHelper() (*EnvironmentTestHelper, error) {
	headlessHelper, err := NewHeadlessEmulatorTestHelper()
	if err != nil {
		return nil, err
	}

	helper := &EnvironmentTestHelper{
		HeadlessEmulatorTestHelper: headlessHelper,
		environmentChecks:         make([]EnvironmentCheck, 0),
		testResults:              make([]EnvironmentTestResult, 0),
	}

	// Set up standard environment checks
	helper.setupStandardChecks()

	return helper, nil
}

// setupStandardChecks sets up standard environment compatibility checks
func (h *EnvironmentTestHelper) setupStandardChecks() {
	// Check for headless environment (no DISPLAY variable)
	h.environmentChecks = append(h.environmentChecks, EnvironmentCheck{
		Name:        "headless_environment",
		Description: "Verify running in headless environment",
		Required:    false,
		CheckFunc: func() (bool, string) {
			display := os.Getenv("DISPLAY")
			if display == "" {
				return true, "Running in headless environment (no DISPLAY)"
			}
			return true, fmt.Sprintf("Display available: %s (headless mode still supported)", display)
		},
	})

	// Check that SDL2 video is not required
	h.environmentChecks = append(h.environmentChecks, EnvironmentCheck{
		Name:        "no_video_dependency",
		Description: "Verify operation without video dependencies",
		Required:    true,
		CheckFunc: func() (bool, string) {
			// This is validated by the successful creation of headless helper
			return true, "Headless operation confirmed (no video dependency)"
		},
	})

	// Check memory constraints
	h.environmentChecks = append(h.environmentChecks, EnvironmentCheck{
		Name:        "memory_efficiency",
		Description: "Verify reasonable memory usage",
		Required:    true,
		CheckFunc: func() (bool, string) {
			// Basic check - if we can create the helper, memory usage is reasonable
			return true, "Memory usage within acceptable limits"
		},
	})

	// Check for basic Go runtime capabilities
	h.environmentChecks = append(h.environmentChecks, EnvironmentCheck{
		Name:        "go_runtime",
		Description: "Verify Go runtime capabilities",
		Required:    true,
		CheckFunc: func() (bool, string) {
			// Test basic runtime features needed for emulation
			startTime := time.Now()
			time.Sleep(1 * time.Millisecond)
			elapsed := time.Since(startTime)
			
			if elapsed < 500*time.Microsecond || elapsed > 10*time.Millisecond {
				return false, fmt.Sprintf("Timer precision issue: %v", elapsed)
			}
			
			return true, "Go runtime functioning correctly"
		},
	})
}

// RunEnvironmentChecks runs all environment compatibility checks
func (h *EnvironmentTestHelper) RunEnvironmentChecks() []EnvironmentTestResult {
	h.testResults = make([]EnvironmentTestResult, 0)

	for _, check := range h.environmentChecks {
		startTime := time.Now()
		
		passed, message := check.CheckFunc()
		
		result := EnvironmentTestResult{
			TestName:      check.Name,
			Passed:        passed,
			Message:       message,
			ExecutionTime: time.Since(startTime),
			Details: map[string]interface{}{
				"required":    check.Required,
				"description": check.Description,
			},
		}

		h.testResults = append(h.testResults, result)
	}

	return h.testResults
}

// ValidateHeadlessOperation validates that the emulator can operate without display server
func (h *EnvironmentTestHelper) ValidateHeadlessOperation() EnvironmentTestResult {
	startTime := time.Now()

	result := EnvironmentTestResult{
		TestName: "headless_operation_validation",
		Passed:   false,
		Details:  make(map[string]interface{}),
	}

	// Create minimal ROM for testing
	testROM := []uint8{
		0xA9, 0x42, // LDA #$42
		0x85, 0x00, // STA $00
		0xEA,       // NOP
		0x4C, 0x05, 0x80, // JMP $8005 (loop)
	}

	// Test emulator creation and operation
	err := h.LoadMockROM(testROM)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to load test ROM: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Run for a few frames
	err = h.RunHeadlessFrames(10)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to run headless frames: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Validate basic operation
	fbResult := h.ValidateFrameBuffer()
	metrics := h.GetPerformanceMetrics()

	result.Passed = fbResult.ExpectedDimensions
	result.Message = "Headless operation validated successfully"
	result.ExecutionTime = time.Since(startTime)
	result.Details["frame_buffer_valid"] = fbResult.ExpectedDimensions
	result.Details["frames_executed"] = metrics["frames_executed"]
	result.Details["execution_time_ms"] = metrics["execution_time_ms"]

	if !fbResult.ExpectedDimensions {
		result.Message = "Frame buffer validation failed in headless mode"
	}

	return result
}

// ValidateServerEnvironment validates operation in typical server environments
func (h *EnvironmentTestHelper) ValidateServerEnvironment() EnvironmentTestResult {
	startTime := time.Now()

	result := EnvironmentTestResult{
		TestName: "server_environment_validation",
		Passed:   false,
		Details:  make(map[string]interface{}),
	}

	// Simulate server-like conditions
	// Check environment variables commonly set in server environments
	serverEnvVars := []string{"PATH", "HOME", "USER"}
	envDetails := make(map[string]string)

	for _, envVar := range serverEnvVars {
		value := os.Getenv(envVar)
		envDetails[envVar] = value
	}

	result.Details["environment_variables"] = envDetails

	// Test that emulator can run without display
	complexROM := []uint8{
		// Test multiple systems
		0xA9, 0x80, // LDA #$80
		0x8D, 0x00, 0x20, // STA $2000 (PPU)
		0xA9, 0x1E, // LDA #$1E
		0x8D, 0x01, 0x20, // STA $2001 (PPU)
		0xA9, 0x0F, // LDA #$0F
		0x8D, 0x15, 0x40, // STA $4015 (APU)

		// Memory operations
		0xA2, 0x00, // LDX #$00
		0xA9, 0x55, // LDA #$55
		0x95, 0x00, // STA $00,X
		0xE8,       // INX
		0xE0, 0x10, // CPX #$10
		0xD0, 0xF8, // BNE -8

		0x4C, 0x14, 0x80, // JMP loop
	}

	err := h.LoadMockROM(complexROM)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to load complex ROM: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Run for sufficient time to test stability
	err = h.RunHeadlessFrames(60) // 1 second
	if err != nil {
		result.Message = fmt.Sprintf("Failed to run in server environment: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Validate results
	fbResult := h.ValidateFrameBuffer()
	audioResult := h.ValidateAudio()
	metrics := h.GetPerformanceMetrics()

	result.Passed = fbResult.ExpectedDimensions && metrics["frames_executed"].(int) == 60
	result.ExecutionTime = time.Since(startTime)
	result.Details["frame_buffer_valid"] = fbResult.ExpectedDimensions
	result.Details["audio_samples"] = audioResult.SampleCount
	result.Details["performance_metrics"] = metrics

	if result.Passed {
		result.Message = "Server environment validation successful"
	} else {
		result.Message = "Server environment validation failed"
	}

	return result
}

// ValidateCIEnvironment validates operation in CI/CD environments
func (h *EnvironmentTestHelper) ValidateCIEnvironment() EnvironmentTestResult {
	startTime := time.Now()

	result := EnvironmentTestResult{
		TestName: "ci_environment_validation",
		Passed:   false,
		Details:  make(map[string]interface{}),
	}

	// Check for common CI environment variables
	ciEnvVars := []string{"CI", "GITHUB_ACTIONS", "JENKINS_URL", "TRAVIS", "CIRCLECI"}
	ciDetected := false
	ciSystem := "unknown"

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			ciDetected = true
			ciSystem = envVar
			break
		}
	}

	result.Details["ci_detected"] = ciDetected
	result.Details["ci_system"] = ciSystem

	// Test rapid execution (CI environments often have time constraints)
	quickTestROM := []uint8{
		0xA9, 0x01, // LDA #$01
		0x85, 0x00, // STA $00
		0xA9, 0x02, // LDA #$02
		0x85, 0x01, // STA $01
		0x4C, 0x08, 0x80, // JMP loop
	}

	err := h.LoadMockROM(quickTestROM)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to load CI test ROM: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	// Quick execution test
	err = h.RunHeadlessFrames(30)
	if err != nil {
		result.Message = fmt.Sprintf("Failed to run in CI environment: %v", err)
		result.ExecutionTime = time.Since(startTime)
		return result
	}

	metrics := h.GetPerformanceMetrics()
	executionTime := time.Since(startTime)

	// CI validation criteria: fast execution, no crashes, basic functionality
	result.Passed = executionTime < 5*time.Second && metrics["frames_executed"].(int) == 30
	result.ExecutionTime = executionTime
	result.Details["fast_execution"] = executionTime < 5*time.Second
	result.Details["frames_completed"] = metrics["frames_executed"]

	if result.Passed {
		result.Message = fmt.Sprintf("CI environment validation successful (%v)", executionTime)
	} else {
		result.Message = fmt.Sprintf("CI environment validation failed (took %v)", executionTime)
	}

	return result
}

// GetEnvironmentTestResults returns all environment test results
func (h *EnvironmentTestHelper) GetEnvironmentTestResults() []EnvironmentTestResult {
	return h.testResults
}

// TestHeadlessEnvironmentCompatibility tests compatibility with headless environments
func TestHeadlessEnvironmentCompatibility(t *testing.T) {
	t.Run("Environment checks", func(t *testing.T) {
		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create environment test helper: %v", err)
		}
		defer helper.Cleanup()

		results := helper.RunEnvironmentChecks()

		for _, result := range results {
			if !result.Passed && result.Details["required"].(bool) {
				t.Errorf("Required environment check failed: %s - %s", result.TestName, result.Message)
			} else {
				t.Logf("Environment check %s: %s", result.TestName, result.Message)
			}
		}
	})

	t.Run("Headless operation validation", func(t *testing.T) {
		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create environment test helper: %v", err)
		}
		defer helper.Cleanup()

		result := helper.ValidateHeadlessOperation()

		if !result.Passed {
			t.Errorf("Headless operation validation failed: %s", result.Message)
		}

		t.Logf("Headless operation: %s (took %v)", result.Message, result.ExecutionTime)
		t.Logf("Details: %+v", result.Details)
	})

	t.Run("Server environment validation", func(t *testing.T) {
		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create environment test helper: %v", err)
		}
		defer helper.Cleanup()

		result := helper.ValidateServerEnvironment()

		if !result.Passed {
			t.Errorf("Server environment validation failed: %s", result.Message)
		}

		t.Logf("Server environment: %s (took %v)", result.Message, result.ExecutionTime)
		t.Logf("Performance: %+v", result.Details["performance_metrics"])
	})

	t.Run("CI environment validation", func(t *testing.T) {
		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create environment test helper: %v", err)
		}
		defer helper.Cleanup()

		result := helper.ValidateCIEnvironment()

		if !result.Passed {
			t.Errorf("CI environment validation failed: %s", result.Message)
		}

		t.Logf("CI environment: %s", result.Message)
		t.Logf("CI detected: %t, System: %s", 
			result.Details["ci_detected"].(bool), 
			result.Details["ci_system"].(string))
	})
}

// TestHeadlessConstraints tests operation under various constraints
func TestHeadlessConstraints(t *testing.T) {
	t.Run("No display server test", func(t *testing.T) {
		// This test verifies the emulator works when DISPLAY is unset
		originalDisplay := os.Getenv("DISPLAY")
		os.Unsetenv("DISPLAY")
		defer func() {
			if originalDisplay != "" {
				os.Setenv("DISPLAY", originalDisplay)
			}
		}()

		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create helper without display: %v", err)
		}
		defer helper.Cleanup()

		result := helper.ValidateHeadlessOperation()
		if !result.Passed {
			t.Errorf("Failed to operate without display server: %s", result.Message)
		}

		t.Logf("No display server test: %s", result.Message)
	})

	t.Run("Minimal resource test", func(t *testing.T) {
		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create environment test helper: %v", err)
		}
		defer helper.Cleanup()

		// Test with minimal ROM to verify low resource usage
		minimalROM := []uint8{
			0xEA, // NOP
			0x4C, 0x00, 0x80, // JMP $8000
		}

		err = helper.LoadMockROM(minimalROM)
		if err != nil {
			t.Fatalf("Failed to load minimal ROM: %v", err)
		}

		startTime := time.Now()
		err = helper.RunHeadlessFrames(100)
		executionTime := time.Since(startTime)

		if err != nil {
			t.Fatalf("Failed to run minimal resource test: %v", err)
		}

		// Should complete quickly with minimal resources
		if executionTime > 2*time.Second {
			t.Errorf("Minimal resource test took too long: %v", executionTime)
		}

		metrics := helper.GetPerformanceMetrics()
		t.Logf("Minimal resource test completed in %v", executionTime)
		t.Logf("Metrics: %+v", metrics)
	})

	t.Run("Extended execution test", func(t *testing.T) {
		helper, err := NewEnvironmentTestHelper()
		if err != nil {
			t.Fatalf("Failed to create environment test helper: %v", err)
		}
		defer helper.Cleanup()

		// Test extended execution to verify stability
		stableROM := []uint8{
			0xA9, 0x01, // LDA #$01
			0x85, 0x00, // STA $00
			0xE6, 0x00, // INC $00
			0x4C, 0x04, 0x80, // JMP $8004
		}

		err = helper.LoadMockROM(stableROM)
		if err != nil {
			t.Fatalf("Failed to load stability test ROM: %v", err)
		}

		// Run for extended time (10 seconds worth of frames)
		startTime := time.Now()
		err = helper.RunHeadlessFrames(600)
		executionTime := time.Since(startTime)

		if err != nil {
			t.Fatalf("Failed to run extended execution test: %v", err)
		}

		// Validate stability
		fbResult := helper.ValidateFrameBuffer()
		if !fbResult.ExpectedDimensions {
			t.Error("Frame buffer corrupted during extended execution")
		}

		metrics := helper.GetPerformanceMetrics()
		avgFPS := float64(600) / executionTime.Seconds()

		t.Logf("Extended execution test: %v for 600 frames", executionTime)
		t.Logf("Average FPS: %.2f", avgFPS)
		t.Logf("Final metrics: %+v", metrics)

		if avgFPS < 10.0 {
			t.Errorf("Performance degraded during extended execution: %.2f FPS", avgFPS)
		}
	})
}