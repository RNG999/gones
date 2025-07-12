package integration

import (
	"os"
	"testing"

	"gones/internal/bus"
	"gones/internal/cartridge"
	"gones/internal/input"
)

// TestCompleteEmulatorValidation tests the complete emulator functionality
// with both background rendering and input processing working together
func TestCompleteEmulatorValidation(t *testing.T) {
	t.Run("Complete emulator workflow validation", func(t *testing.T) {
		// Load sample ROM if available
		romPath := "../../roms/sample.nes"
		if _, err := os.Stat(romPath); os.IsNotExist(err) {
			t.Skip("Sample ROM not available for complete validation")
		}

		file, err := os.Open(romPath)
		if err != nil {
			t.Skipf("Failed to open ROM file: %v", err)
		}
		defer file.Close()

		cart, err := cartridge.LoadFromReader(file)
		if err != nil {
			t.Fatalf("Failed to load cartridge: %v", err)
		}

		// Initialize the complete emulator system
		emulator := bus.New()
		emulator.LoadCartridge(cart)
		emulator.Reset()

		// Verify initial state
		if emulator.CPU == nil {
			t.Fatal("CPU not initialized")
		}
		if emulator.PPU == nil {
			t.Fatal("PPU not initialized")
		}
		if emulator.Memory == nil {
			t.Fatal("Memory not initialized")
		}
		if emulator.Input == nil {
			t.Fatal("Input not initialized")
		}

		// Test background rendering with input processing
		validateBackgroundRenderingWithInput(t, emulator)

		// Test system stability under combined load
		validateSystemStability(t, emulator)

		// Test input responsiveness during rendering
		validateInputResponsiveness(t, emulator)
	})
}

// validateBackgroundRenderingWithInput tests background rendering while processing input
func validateBackgroundRenderingWithInput(t *testing.T, emulator *bus.Bus) {
	t.Log("Validating background rendering with input processing...")

	inputState := emulator.Input

	// Simulate controller input sequence while rendering
	inputSequence := []struct {
		frame  int
		button input.Button
		state  bool
	}{
		{10, input.Start, true},    // Press Start at frame 10
		{20, input.Start, false},   // Release Start at frame 20
		{30, input.Right, true},    // Move right at frame 30
		{50, input.A, true},        // Press A button at frame 50
		{60, input.A, false},       // Release A button at frame 60
		{70, input.Right, false},   // Stop moving at frame 70
	}

	sequenceIndex := 0
	frameCount := 100

	for frame := 0; frame < frameCount; frame++ {
		// Apply input sequence
		if sequenceIndex < len(inputSequence) && frame >= inputSequence[sequenceIndex].frame {
			action := inputSequence[sequenceIndex]
			inputState.Controller1.SetButton(action.button, action.state)
			t.Logf("Frame %d: Set button %d to %v", frame, action.button, action.state)
			sequenceIndex++
		}

		// Update input strobe (simulate hardware read sequence)
		inputState.Write(0x4016, 0x01) // Set strobe
		inputState.Write(0x4016, 0x00) // Clear strobe

		// Run emulator for one frame
		for cycle := 0; cycle < 29780; cycle++ {
			emulator.Step()
		}

		// Verify background rendering every 20 frames
		if frame%20 == 19 {
			frameBuffer := emulator.GetFrameBuffer()
			if len(frameBuffer) == 0 {
				t.Errorf("Frame buffer empty at frame %d", frame)
				continue
			}

			// Analyze frame buffer content
			analysis := analyzeFrameContent(frameBuffer)
			
			if analysis.UniqueColors < 2 {
				t.Errorf("Frame %d: insufficient color variation (%d colors)", frame, analysis.UniqueColors)
			}

			// Verify PPU background rendering is enabled
			ppumask := emulator.Memory.Read(0x2001)
			backgroundEnabled := (ppumask & 0x08) != 0
			if !backgroundEnabled {
				t.Errorf("Frame %d: background rendering disabled", frame)
			}

			t.Logf("Frame %d: %d unique colors, %.1f%% black pixels", 
				frame, analysis.UniqueColors, analysis.BlackPercentage)
		}
	}

	// Final verification
	finalFrameBuffer := emulator.GetFrameBuffer()
	finalAnalysis := analyzeFrameContent(finalFrameBuffer)
	
	if finalAnalysis.UniqueColors < 2 {
		t.Error("Final frame has insufficient color variation")
	}

	if finalAnalysis.BlackPercentage > 95.0 {
		t.Error("Final frame is mostly black - potential rendering issue")
	}

	t.Log("Background rendering with input validation passed")
}

// validateSystemStability tests system stability under combined rendering and input load
func validateSystemStability(t *testing.T, emulator *bus.Bus) {
	t.Log("Validating system stability...")

	inputState := emulator.Input
	stabilityFrames := 60 // Test for 60 frames

	// Rapid input changes to stress test the system
	for frame := 0; frame < stabilityFrames; frame++ {
		// Simulate rapid button presses (stress test)
		buttons := []input.Button{input.A, input.B, input.Up, input.Down, input.Left, input.Right}
		for i, button := range buttons {
			pressed := (frame+i)%4 < 2 // Rapid on/off pattern
			inputState.Controller1.SetButton(button, pressed)
		}

		// Update input registers
		inputState.Write(0x4016, 0x01)
		inputState.Write(0x4016, 0x00)

		// Run frame
		initialPC := emulator.CPU.PC
		for cycle := 0; cycle < 29780; cycle++ {
			emulator.Step()
		}

		// Verify system stability
		if emulator.CPU.PC < 0x8000 {
			t.Errorf("Frame %d: CPU PC moved outside ROM area: 0x%04X", frame, emulator.CPU.PC)
		}

		if emulator.CPU.SP > 0xFF {
			t.Errorf("Frame %d: CPU stack pointer corrupted: 0x%02X", frame, emulator.CPU.SP)
		}

		// Verify frame buffer is still being generated
		frameBuffer := emulator.GetFrameBuffer()
		if len(frameBuffer) == 0 {
			t.Errorf("Frame %d: frame buffer lost during stress test", frame)
		}

		// Check for infinite loops or hangs
		if frame > 10 && emulator.CPU.PC == initialPC {
			t.Logf("Frame %d: CPU may be in a tight loop at 0x%04X", frame, emulator.CPU.PC)
		}
	}

	t.Log("System stability validation passed")
}

// validateInputResponsiveness tests input responsiveness during active rendering
func validateInputResponsiveness(t *testing.T, emulator *bus.Bus) {
	t.Log("Validating input responsiveness...")

	inputState := emulator.Input

	// Test each button for responsiveness
	testButtons := []input.Button{
		input.A, input.B, input.Start, input.Select,
		input.Up, input.Down, input.Left, input.Right,
	}

	for _, button := range testButtons {
		buttonName := getButtonName(button)
		t.Run(buttonName, func(t *testing.T) {
			// Reset button state
			inputState.Controller1.Reset()

			// Press button
			inputState.Controller1.SetButton(button, true)
			inputState.Write(0x4016, 0x01) // Strobe
			inputState.Write(0x4016, 0x00) // Clear strobe

			// Run a few cycles to process input
			for cycle := 0; cycle < 100; cycle++ {
				emulator.Step()
			}

			// Test button read sequence
			inputState.Write(0x4016, 0x01) // Set strobe
			inputState.Write(0x4016, 0x00) // Clear strobe

			// Read button states in hardware order
			buttonValues := make([]uint8, 8)
			for i := 0; i < 8; i++ {
				buttonValues[i] = inputState.Read(0x4016) & 0x01
			}

			// Check if our button was read correctly
			buttonIndex := getButtonIndex(button)
			if buttonIndex >= 0 && buttonIndex < len(buttonValues) {
				if buttonValues[buttonIndex] != 1 {
					t.Errorf("Button %s not read correctly: expected 1, got %d", 
						buttonName, buttonValues[buttonIndex])
				}
			}

			// Release button
			inputState.Controller1.SetButton(button, false)
			inputState.Write(0x4016, 0x01) // Strobe
			inputState.Write(0x4016, 0x00) // Clear strobe

			// Run a few more cycles
			for cycle := 0; cycle < 100; cycle++ {
				emulator.Step()
			}

			// Verify button is released
			inputState.Write(0x4016, 0x01) // Set strobe
			inputState.Write(0x4016, 0x00) // Clear strobe

			if buttonIndex >= 0 && buttonIndex < 8 {
				value := inputState.Read(0x4016) & 0x01
				for i := 1; i <= buttonIndex; i++ {
					value = inputState.Read(0x4016) & 0x01
				}
				if value != 0 {
					t.Errorf("Button %s not released correctly", buttonName)
				}
			}
		})
	}

	t.Log("Input responsiveness validation passed")
}

// getButtonName returns a human-readable name for a button
func getButtonName(button input.Button) string {
	switch button {
	case input.A:
		return "A Button"
	case input.B:
		return "B Button"
	case input.Select:
		return "Select Button"
	case input.Start:
		return "Start Button"
	case input.Up:
		return "Up Button"
	case input.Down:
		return "Down Button"
	case input.Left:
		return "Left Button"
	case input.Right:
		return "Right Button"
	default:
		return "Unknown Button"
	}
}

// getButtonIndex returns the hardware read index for a button
func getButtonIndex(button input.Button) int {
	switch button {
	case input.A:
		return 0
	case input.B:
		return 1
	case input.Select:
		return 2
	case input.Start:
		return 3
	case input.Up:
		return 4
	case input.Down:
		return 5
	case input.Left:
		return 6
	case input.Right:
		return 7
	default:
		return -1
	}
}

// FrameContentAnalysis contains frame buffer analysis
type FrameContentAnalysis struct {
	TotalPixels     int
	BlackPixels     int
	WhitePixels     int
	ColorPixels     int
	UniqueColors    int
	BlackPercentage float64
	WhitePercentage float64
	ColorPercentage float64
}

// analyzeFrameContent analyzes frame buffer content
func analyzeFrameContent(frameBuffer []uint32) *FrameContentAnalysis {
	analysis := &FrameContentAnalysis{
		TotalPixels: len(frameBuffer),
	}

	colorMap := make(map[uint32]bool)

	for _, pixel := range frameBuffer {
		r := (pixel >> 16) & 0xFF
		g := (pixel >> 8) & 0xFF
		b := pixel & 0xFF

		colorMap[pixel] = true

		if r < 50 && g < 50 && b < 50 {
			analysis.BlackPixels++
		} else if r > 200 && g > 200 && b > 200 {
			analysis.WhitePixels++
		} else {
			analysis.ColorPixels++
		}
	}

	analysis.UniqueColors = len(colorMap)

	if analysis.TotalPixels > 0 {
		analysis.BlackPercentage = float64(analysis.BlackPixels) * 100.0 / float64(analysis.TotalPixels)
		analysis.WhitePercentage = float64(analysis.WhitePixels) * 100.0 / float64(analysis.TotalPixels)
		analysis.ColorPercentage = float64(analysis.ColorPixels) * 100.0 / float64(analysis.TotalPixels)
	}

	return analysis
}

// TestBackgroundRenderingStability tests background rendering stability over time
func TestBackgroundRenderingStability(t *testing.T) {
	t.Run("Background rendering stability over extended period", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)
		helper.SetupBasicCHR()

		// Run for extended period to test stability
		stableFrames := 300 // 5 seconds at 60 FPS
		var lastAnalysis *FrameContentAnalysis

		for frame := 0; frame < stableFrames; frame++ {
			// Run one frame
			helper.RunFrame()

			// Analyze every 30 frames
			if frame%30 == 29 {
				frameBuffer := helper.Bus.GetFrameBuffer()
				if len(frameBuffer) == 0 {
					t.Errorf("Frame %d: no frame buffer", frame)
					continue
				}

				analysis := analyzeFrameContent(frameBuffer)
				
				// Verify consistent rendering
				if analysis.UniqueColors < 2 {
					t.Errorf("Frame %d: insufficient colors", frame)
				}

				// Compare with previous analysis for stability
				if lastAnalysis != nil {
					colorDiff := abs(analysis.UniqueColors - lastAnalysis.UniqueColors)
					if colorDiff > 5 {
						t.Logf("Frame %d: color count variation: %d -> %d", 
							frame, lastAnalysis.UniqueColors, analysis.UniqueColors)
					}
				}

				lastAnalysis = analysis
			}
		}

		t.Log("Background rendering stability test passed")
	})
}

// TestInputProcessingStability tests input processing stability
func TestInputProcessingStability(t *testing.T) {
	t.Run("Input processing stability under load", func(t *testing.T) {
		helper := NewIntegrationTestHelper()
		helper.SetupBasicROM(0x8000)

		inputState := helper.Input

		// Test rapid input changes
		rapidInputFrames := 120
		
		for frame := 0; frame < rapidInputFrames; frame++ {
			// Rapid button state changes
			allButtons := []input.Button{
				input.A, input.B, input.Select, input.Start,
				input.Up, input.Down, input.Left, input.Right,
			}

			for i, button := range allButtons {
				// Create rapid pattern
				pressed := (frame+i)%3 == 0
				inputState.Controller1.SetButton(button, pressed)
			}

			// Process input
			inputState.Write(0x4016, 0x01)
			inputState.Write(0x4016, 0x00)

			// Run emulator
			helper.RunCycles(1000)

			// Verify input system stability every 20 frames
			if frame%20 == 19 {
				// Test input read sequence
				inputState.Write(0x4016, 0x01)
				inputState.Write(0x4016, 0x00)

				// Should be able to read 8 button states
				for i := 0; i < 8; i++ {
					value := inputState.Read(0x4016)
					if value != 0x40 && value != 0x41 {
						t.Errorf("Frame %d: unexpected input value 0x%02X at position %d", 
							frame, value, i)
					}
				}
			}
		}

		t.Log("Input processing stability test passed")
	})
}

// abs returns the absolute value
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}