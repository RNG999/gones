package main

import (
	"fmt"
	"time"

	"gones/internal/bus"
	"gones/internal/cartridge"
)

func main() {
	fmt.Println("=== TIMER FIX ANALYSIS ===")
	
	// Load ROM directly
	cart, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		return
	}

	// Create bus and load cartridge
	systemBus := bus.New()
	systemBus.LoadCartridge(cart)
	systemBus.Reset()

	fmt.Println("Bus initialized. Analyzing timer behavior...")
	
	// Monitor critical memory locations
	frameCount := 0
	startTime := time.Now()
	
	// Previous values for change detection
	lastTimer07 := uint8(0xFF)
	lastGameState00 := uint8(0xFF)
	lastTimer09 := uint8(0xFF)
	
	// Run analysis for limited time
	for frameCount < 180 { // 3 seconds at 60fps
		// Run one frame
		systemBus.Run(1)
		frameCount++
		
		// Check every 60 frames (1 second)
		if frameCount%60 == 0 {
			// Read critical memory addresses through Memory interface
			timer07 := systemBus.Memory.Read(0x07)
			gameState00 := systemBus.Memory.Read(0x00)
			timer09 := systemBus.Memory.Read(0x09)
			
			// Check for changes
			if timer07 != lastTimer07 || gameState00 != lastGameState00 || timer09 != lastTimer09 {
				fmt.Printf("[Frame %d] MEMORY CHANGE DETECTED!\n", frameCount)
				fmt.Printf("  Timer $07: 0x%02X -> 0x%02X\n", lastTimer07, timer07)
				fmt.Printf("  Game State $00: 0x%02X -> 0x%02X\n", lastGameState00, gameState00)
				fmt.Printf("  Timer $09: 0x%02X -> 0x%02X\n", lastTimer09, timer09)
				
				lastTimer07 = timer07
				lastGameState00 = gameState00
				lastTimer09 = timer09
			}
			
			// Check PPU NMI status
			ppuCtrl := systemBus.PPU.ReadRegister(0x2000)
			nmiEnabled := (ppuCtrl & 0x80) != 0
			
			fmt.Printf("[%ds] Timer=$07:0x%02X, GameState=$00:0x%02X, Timer=$09:0x%02X, NMI:%v\n",
				frameCount/60, timer07, gameState00, timer09, nmiEnabled)
				
			// Success condition
			if nmiEnabled {
				fmt.Printf("\n*** NMI ENABLED! Game should start soon! ***\n")
				break
			}
		}
	}
	
	elapsed := time.Since(startTime)
	fmt.Printf("\nAnalysis completed: %d frames in %.2fs (%.1f FPS)\n", 
		frameCount, elapsed.Seconds(), float64(frameCount)/elapsed.Seconds())
		
	// Final status
	if frameCount >= 180 {
		fmt.Println("\n*** ISSUE CONFIRMED: NMI never enabled after 3 seconds ***")
		fmt.Println("SMB initialization appears to be stuck in infinite loop")
		fmt.Println("Need to debug CPU execution path and timer countdown logic")
	}
}