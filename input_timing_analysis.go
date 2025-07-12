package main

import (
	"fmt"
	"os"
	"time"
	"gones/internal/bus"
	"gones/internal/cartridge"
	"gones/internal/input"
)

func main() {
	fmt.Println("=== INPUT TIMING ANALYSIS ===")
	fmt.Println("Analyzing the timing relationship between input and game strobe sequences")

	// Load SMB ROM
	rom, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create system
	system := bus.New()
	system.LoadCartridge(rom)

	fmt.Println("\nRunning 10 frames to establish baseline behavior...")
	
	// Run a few frames to get to stable state
	for i := 0; i < 10; i++ {
		system.Run(1)
	}
	
	fmt.Println("\nNow testing Start button input timing...")
	
	// Test sequence: Press Start button and observe timing
	controller1 := system.Input.Controller1
	
	// Test 1: Press Start button before strobe
	fmt.Println("\n--- TEST 1: Press Start before next strobe ---")
	controller1.SetButton(input.Start, true)
	fmt.Printf("Start button set to PRESSED\n")
	
	// Run 2 frames to see what happens
	for frame := 0; frame < 2; frame++ {
		fmt.Printf("\n=== Frame %d ===\n", frame+1)
		system.Run(1)
	}
	
	// Release button
	controller1.SetButton(input.Start, false)
	fmt.Printf("Start button set to RELEASED\n")
	
	// Test 2: Press Start during strobe sequence
	fmt.Println("\n--- TEST 2: Press Start during strobe sequence ---")
	
	// Run one frame and try to time the button press during the strobe
	go func() {
		time.Sleep(16 * time.Millisecond) // Approximate timing of strobe in frame
		controller1.SetButton(input.Start, true)
		fmt.Printf("Start button set to PRESSED (during frame)\n")
		time.Sleep(5 * time.Millisecond)
		controller1.SetButton(input.Start, false)
		fmt.Printf("Start button set to RELEASED (during frame)\n")
	}()
	
	fmt.Printf("\n=== Frame with timed input ===\n")
	system.Run(1)
	
	// Test 3: Long press test
	fmt.Println("\n--- TEST 3: Long press test ---")
	controller1.SetButton(input.Start, true)
	fmt.Printf("Start button set to PRESSED (long press)\n")
	
	for frame := 0; frame < 3; frame++ {
		fmt.Printf("\n=== Frame %d (long press) ===\n", frame+1)
		system.Run(1)
	}
	
	controller1.SetButton(input.Start, false)
	fmt.Printf("Start button set to RELEASED\n")
	
	fmt.Println("\n=== ANALYSIS COMPLETE ===")
	fmt.Println("Check the TIMING_DEBUG logs above to see:")
	fmt.Println("1. When buttons are changed vs when strobe captures them")
	fmt.Println("2. Whether input timing affects game behavior")
	fmt.Println("3. How long button presses need to be held")
}