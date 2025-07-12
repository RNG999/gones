package main

import (
	"fmt"
	"time"

	"gones/internal/bus"
	"gones/internal/cartridge"
)

func main() {
	fmt.Println("=== DIRECT NMI FIX TEST ===")
	
	// Load ROM
	cart, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		return
	}

	// Create bus
	systemBus := bus.New()
	systemBus.LoadCartridge(cart)
	systemBus.Reset()

	fmt.Println("Running for a few frames, then applying direct NMI fix...")
	
	frameCount := 0
	startTime := time.Now()
	
	// Run for 10 frames to let initialization start
	for frameCount < 10 {
		systemBus.Run(1)
		frameCount++
	}
	
	// Check initial state
	timer := systemBus.Memory.Read(0x07)
	gameState := systemBus.Memory.Read(0x00)
	ppuCtrl := systemBus.PPU.ReadRegister(0x2000)
	nmiEnabled := (ppuCtrl & 0x80) != 0
	
	fmt.Printf("Before fix: Timer=0x%02X, GameState=0x%02X, NMI=%v\n", 
		timer, gameState, nmiEnabled)
	
	// Apply direct fix by forcing timer to 0
	fmt.Println("\n*** APPLYING DIRECT NMI FIX ***")
	systemBus.Memory.Write(0x07, 0)
	
	// Give it a few frames for the fix to take effect
	for frameCount < 20 {
		systemBus.Run(1)
		frameCount++
	}
	
	// Check state after fix
	timer = systemBus.Memory.Read(0x07)
	gameState = systemBus.Memory.Read(0x00)
	ppuCtrl = systemBus.PPU.ReadRegister(0x2000)
	nmiEnabled = (ppuCtrl & 0x80) != 0
	
	fmt.Printf("\nAfter fix: Timer=0x%02X, GameState=0x%02X, NMI=%v\n", 
		timer, gameState, nmiEnabled)
	
	// Run additional frames to see game progress
	for frameCount < 60 { // Run to 1 second total
		systemBus.Run(1)
		frameCount++
		
		if frameCount%20 == 0 {
			timer = systemBus.Memory.Read(0x07)
			gameState = systemBus.Memory.Read(0x00)
			ppuCtrl = systemBus.PPU.ReadRegister(0x2000)
			nmiEnabled = (ppuCtrl & 0x80) != 0
			
			fmt.Printf("[Frame %d] Timer=0x%02X, GameState=0x%02X, NMI=%v\n", 
				frameCount, timer, gameState, nmiEnabled)
		}
	}
	
	elapsed := time.Since(startTime)
	fmt.Printf("\nTest completed: %d frames in %.2fs (%.1f FPS)\n", 
		frameCount, elapsed.Seconds(), float64(frameCount)/elapsed.Seconds())
		
	// Final verification
	finalTimer := systemBus.Memory.Read(0x07)
	finalGameState := systemBus.Memory.Read(0x00)
	finalPPUCTRL := systemBus.PPU.ReadRegister(0x2000)
	finalNMI := (finalPPUCTRL & 0x80) != 0
	
	fmt.Printf("\nFinal state:\n")
	fmt.Printf("  Timer ($07): 0x%02X\n", finalTimer)
	fmt.Printf("  Game State ($00): 0x%02X\n", finalGameState)
	fmt.Printf("  PPUCTRL: 0x%02X (NMI: %v)\n", finalPPUCTRL, finalNMI)
	
	if finalGameState != 0 && finalNMI {
		fmt.Println("\n✅ SUCCESS: SMB fix working correctly!")
	} else if finalGameState == 0 {
		fmt.Println("\n❌ Game state still 0 - fix may need adjustment")
	} else {
		fmt.Println("\n⚠️  Partial success: Game progressed but NMI status unclear")
	}
}