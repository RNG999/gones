package main

import (
	"fmt"
	"os"
	"gones/internal/bus"
	"gones/internal/cartridge"
	"gones/internal/input"
)

func main() {
	fmt.Println("=== PPUMASK RENDERING TEST ===")
	fmt.Println("Testing when Super Mario Bros enables rendering via PPUMASK")

	// Load SMB ROM
	rom, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create system
	system := bus.New()
	system.LoadCartridge(rom)

	fmt.Println("\n🎮 Phase 1: Running to title screen and checking PPUMASK...")
	
	// Run frames and monitor PPUMASK
	for frame := 0; frame < 30; frame++ {
		system.Run(1)
		afterMask := system.PPU.ReadRegister(0x2001)
		
		if frame % 5 == 0 {
			fmt.Printf("Frame %d: PPUMASK=0x%02X\n", frame, afterMask)
		}
	}

	fmt.Println("\n🎮 Phase 2: Pressing Start and monitoring PPUMASK changes...")
	
	// Press Start button
	system.Input.Controller1.SetButton(input.Start, true)
	
	// Monitor PPUMASK changes after Start button
	for frame := 0; frame < 20; frame++ {
		system.Run(1)
		afterMask := system.PPU.ReadRegister(0x2001)
		
		fmt.Printf("Frame %d: PPUMASK=0x%02X\n", frame+30, afterMask)
		
		// Check if rendering gets enabled
		if afterMask != 0x00 {
			fmt.Printf("🎉 RENDERING ENABLED! PPUMASK changed to 0x%02X\n", afterMask)
			break
		}
	}
	
	// Release Start button
	system.Input.Controller1.SetButton(input.Start, false)
	
	fmt.Println("\n🎮 Phase 3: Testing manual PPUMASK write...")
	
	// Test: Manually enable rendering to see if that fixes the display
	fmt.Println("Manually setting PPUMASK to 0x1E (enable background and sprites)...")
	system.PPU.WriteRegister(0x2001, 0x1E)
	
	finalMask := system.PPU.ReadRegister(0x2001)
	fmt.Printf("After manual write: PPUMASK=0x%02X\n", finalMask)
	
	// Run a few frames to see if anything renders
	fmt.Println("Running 5 frames with rendering enabled...")
	for frame := 0; frame < 5; frame++ {
		system.Run(1)
		mask := system.PPU.ReadRegister(0x2001)
		fmt.Printf("  Frame %d: PPUMASK=0x%02X\n", frame+1, mask)
	}

	fmt.Println("\n=== ANALYSIS ===")
	fmt.Println("If PPUMASK never changes from 0x00, the game logic isn't progressing")
	fmt.Println("to the point where it would enable rendering.")
	fmt.Println("This suggests the game state machine is stuck or not advancing properly.")
}