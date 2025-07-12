package main

import (
	"fmt"
	"os"
	"time"
	"gones/internal/bus"
	"gones/internal/cartridge"
)

func main() {
	fmt.Println("=== Super Mario Bros Timer Investigation ===")
	fmt.Println("Investigating why timer at $07 is stuck at 0x06...")

	// Load SMB ROM
	rom, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create system
	system := bus.New()
	system.LoadCartridge(rom)

	// Let the game run for some frames to see if timer changes
	fmt.Println("\nRunning for 30 frames to observe timer behavior...")
	
	var lastTimerValue uint8 = 0xFF
	var timerStuckCount int = 0
	
	for frame := 0; frame < 30; frame++ {
		// Run one frame
		system.Run(1)
		
		// Check timer value
		currentTimer := system.Memory.Read(0x07)
		gameState := system.Memory.Read(0x00)
		
		if currentTimer != lastTimerValue {
			fmt.Printf("Frame %d: Timer changed from 0x%02X to 0x%02X (GameState: 0x%02X)\n", 
				frame, lastTimerValue, currentTimer, gameState)
			lastTimerValue = currentTimer
			timerStuckCount = 0
		} else if frame > 0 {
			timerStuckCount++
		}
		
		// Check if timer has been stuck for too long
		if timerStuckCount > 10 {
			fmt.Printf("Frame %d: Timer stuck at 0x%02X for %d frames (GameState: 0x%02X)\n", 
				frame, currentTimer, timerStuckCount, gameState)
			
			// Let's check if the game is actually running by looking at some other memory locations
			fmt.Printf("  CPU PC: 0x%04X\n", system.CPU.PC)
			fmt.Printf("  Some other memory locations:\n")
			fmt.Printf("    $00: 0x%02X, $01: 0x%02X, $02: 0x%02X\n", 
				system.Memory.Read(0x00), system.Memory.Read(0x01), system.Memory.Read(0x02))
			break
		}
		
		// Brief pause to avoid overwhelming output
		time.Sleep(10 * time.Millisecond)
	}
	
	fmt.Println("\nAnalysis complete.")
}