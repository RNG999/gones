package main

import (
	"fmt"
	"os"
	"gones/internal/bus"
	"gones/internal/cartridge"
	"gones/internal/input"
)

func main() {
	fmt.Println("=== NMI TIMING FIX TEST ===")
	fmt.Println("Testing manual NMI re-enabling during title screen wait")

	// Load SMB ROM
	rom, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create system
	system := bus.New()
	system.LoadCartridge(rom)

	fmt.Println("\nRunning to stable title screen state...")
	
	// Run until we reach the stable title screen state
	for frame := 0; frame < 10; frame++ {
		system.Run(1)
	}
	
	fmt.Println("\nCurrent state:")
	ppuctrl := system.PPU.GetPPUCTRL()
	nmiEnabled := (ppuctrl & 0x80) != 0
	fmt.Printf("PPUCTRL: 0x%02X, NMI enabled: %t\n", ppuctrl, nmiEnabled)
	
	fmt.Println("\nTesting fix: Manual NMI re-enabling during input processing")
	
	// Strategy: Enable NMI manually before pressing Start button
	// This simulates what should happen if the timing was correct
	
	// Press Start button
	fmt.Println("\n--- Pressing Start button ---")
	system.Input.Controller1.SetButton(input.Start, true)
	
	// Manual NMI enable before processing input
	fmt.Println("Manually enabling NMI...")
	system.PPU.WriteRegister(0x2000, ppuctrl | 0x80) // Set bit 7 (NMI enable)
	
	newPPUCTRL := system.PPU.GetPPUCTRL()
	newNMIEnabled := (newPPUCTRL & 0x80) != 0
	fmt.Printf("After manual enable - PPUCTRL: 0x%02X, NMI enabled: %t\n", newPPUCTRL, newNMIEnabled)
	
	// Run a few frames to see if the game responds
	fmt.Println("\nRunning frames with NMI enabled and Start pressed...")
	for frame := 0; frame < 5; frame++ {
		fmt.Printf("\n=== Frame %d ===\n", frame+1)
		system.Run(1)
		
		// Check if game state changed
		currentPPUCTRL := system.PPU.GetPPUCTRL()
		currentNMI := (currentPPUCTRL & 0x80) != 0
		fmt.Printf("PPUCTRL: 0x%02X, NMI: %t\n", currentPPUCTRL, currentNMI)
		
		// If NMI gets disabled again, re-enable it
		if !currentNMI {
			fmt.Println("NMI was disabled, re-enabling...")
			system.PPU.WriteRegister(0x2000, currentPPUCTRL | 0x80)
		}
	}
	
	// Release button
	system.Input.Controller1.SetButton(input.Start, false)
	
	fmt.Println("\n=== RESULTS ===")
	fmt.Println("If the game responds to Start button with this fix,")
	fmt.Println("then the solution is to ensure NMI stays enabled during title screen input processing.")
}