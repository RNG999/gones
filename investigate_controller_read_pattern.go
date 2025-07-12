package main

import (
	"fmt"
	"time"

	"gones/internal/bus"
	"gones/internal/cartridge"
	"gones/internal/input"
)

// Investigate the specific controller read pattern and what the game is looking for
func main() {
	fmt.Println("=== CONTROLLER READ PATTERN INVESTIGATION ===")
	fmt.Println("Analyzing what Super Mario Bros is actually looking for in controller input")
	
	// Create system bus directly
	systemBus := bus.New()
	
	// Load ROM directly
	cart, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Failed to load ROM: %v\n", err)
		return
	}
	
	systemBus.LoadCartridge(cart)
	systemBus.Reset()
	
	fmt.Println("\n=== CONTROLLER READ PATTERN ANALYSIS ===")
	fmt.Println("From the debug output, we see the game repeatedly reads $06 and $07.")
	fmt.Println("These are likely controller-related addresses.")
	fmt.Println("Let's see what happens when we set different controller states...")
	fmt.Println("")
	
	// Test 1: No input
	fmt.Println("TEST 1: Running with NO controller input for 10 frames...")
	for i := 0; i < 10 * 30000; i++ {
		systemBus.Step()
		if i%5000 == 0 {
			time.Sleep(1 * time.Microsecond)
		}
	}
	
	// Test 2: Start button only
	fmt.Println("\nTEST 2: Setting Start button and running for 10 frames...")
	systemBus.SetControllerButton(0, input.Start, true)
	for i := 0; i < 10 * 30000; i++ {
		systemBus.Step()
		if i%5000 == 0 {
			time.Sleep(1 * time.Microsecond)
		}
	}
	
	// Test 3: Multiple buttons
	fmt.Println("\nTEST 3: Setting multiple buttons (A + Start) and running...")
	systemBus.SetControllerButton(0, input.A, true)
	systemBus.SetControllerButton(0, input.Start, true)
	for i := 0; i < 10 * 30000; i++ {
		systemBus.Step()
		if i%5000 == 0 {
			time.Sleep(1 * time.Microsecond)
		}
	}
	
	// Test 4: All buttons
	fmt.Println("\nTEST 4: Setting ALL buttons and running...")
	systemBus.SetControllerButton(0, input.A, true)
	systemBus.SetControllerButton(0, input.B, true)
	systemBus.SetControllerButton(0, input.Select, true)
	systemBus.SetControllerButton(0, input.Start, true)
	systemBus.SetControllerButton(0, input.Up, true)
	systemBus.SetControllerButton(0, input.Down, true)
	systemBus.SetControllerButton(0, input.Left, true)
	systemBus.SetControllerButton(0, input.Right, true)
	for i := 0; i < 10 * 30000; i++ {
		systemBus.Step()
		if i%5000 == 0 {
			time.Sleep(1 * time.Microsecond)
		}
	}
	
	fmt.Println("\n=== ANALYSIS RESULTS ===")
	fmt.Println("Key observations:")
	fmt.Println("1. The game continuously reads $06 and $07 in a tight loop")
	fmt.Println("2. This suggests it's waiting for a specific memory condition")
	fmt.Println("3. Controller input alone may not be sufficient")
	fmt.Println("4. The game may be waiting for a timer, frame count, or other condition")
	fmt.Println("")
	fmt.Println("HYPOTHESIS:")
	fmt.Println("Super Mario Bros might be in a 'wait state' during initialization")
	fmt.Println("that requires multiple frames to pass before accepting input.")
	fmt.Println("The $06/$07 reads might be checking for this condition.")
	fmt.Println("")
	fmt.Println("NEXT STEPS:")
	fmt.Println("1. Check what values are actually stored in $06 and $07")
	fmt.Println("2. Determine if these are frame counters or state flags")
	fmt.Println("3. Run for much longer to see if condition eventually changes")
}