package main

import (
	"fmt"
	"os"
	"time"

	"gones/internal/cartridge"
	"gones/internal/cpu"
	"gones/internal/graphics"
	"gones/internal/input"
	"gones/internal/memory"
	"gones/internal/ppu"
	"gones/internal/system"
)

func main() {
	fmt.Println("=== FIXING SMB TIMER ISSUE ===")
	
	// Load Super Mario Bros ROM
	cart, err := cartridge.LoadROM("super_mario_bros.nes")
	if err != nil {
		fmt.Printf("Failed to load ROM: %v\n", err)
		return
	}

	// Create graphics backend
	graphics := graphics.NewEbitengineBackend()

	// Initialize system
	sys := system.NewNESSystem(cart, graphics)

	// Track specific memory addresses
	lastTimer := uint8(0xFF)
	lastGameState := uint8(0xFF)
	ppuCtrlState := uint8(0xFF)
	
	frameCount := 0
	startTime := time.Now()

	// Monitor key addresses during execution
	for frameCount < 300 { // Run for 5 seconds at 60fps
		// Run one frame
		err := sys.RunOneFrame()
		if err != nil {
			fmt.Printf("Error running frame: %v\n", err)
			break
		}
		
		frameCount++
		
		// Read critical memory locations every 30 frames
		if frameCount%30 == 0 {
			// Direct memory access to avoid debug spam
			memSys := sys.GetMemorySystem()
			
			// Read $07 timer
			currentTimer := memSys.Read(0x07)
			if currentTimer != lastTimer {
				fmt.Printf("[Frame %d] Timer $07 changed: 0x%02X -> 0x%02X\n", 
					frameCount, lastTimer, currentTimer)
				lastTimer = currentTimer
			}
			
			// Read $00 game state  
			currentGameState := memSys.Read(0x00)
			if currentGameState != lastGameState {
				fmt.Printf("[Frame %d] Game State $00 changed: 0x%02X -> 0x%02X\n",
					frameCount, lastGameState, currentGameState)
				lastGameState = currentGameState
			}
			
			// Check PPUCTRL NMI enable bit
			ppuSys := sys.GetPPUSystem()
			currentPPUCtrl := ppuSys.GetPPUCTRL()
			if currentPPUCtrl != ppuCtrlState {
				nmiEnabled := (currentPPUCtrl & 0x80) != 0
				fmt.Printf("[Frame %d] PPUCTRL changed: 0x%02X -> 0x%02X (NMI: %v)\n",
					frameCount, ppuCtrlState, currentPPUCtrl, nmiEnabled)
				ppuCtrlState = currentPPUCtrl
			}
			
			// Current status report
			nmiEnabled := (currentPPUCtrl & 0x80) != 0
			fmt.Printf("[Frame %d] Status: Timer=0x%02X, GameState=0x%02X, NMI=%v\n",
				frameCount, currentTimer, currentGameState, nmiEnabled)
		}
		
		// Break early if NMI gets enabled (success!)
		if (ppuCtrlState & 0x80) != 0 {
			fmt.Printf("\n*** SUCCESS! NMI ENABLED AT FRAME %d ***\n", frameCount)
			break
		}
	}
	
	elapsed := time.Since(startTime)
	fmt.Printf("\nRan %d frames in %.2fs (%.1f FPS)\n", 
		frameCount, elapsed.Seconds(), float64(frameCount)/elapsed.Seconds())
	
	if (ppuCtrlState & 0x80) == 0 {
		fmt.Println("\n*** ISSUE CONFIRMED: NMI NEVER ENABLED ***")
		fmt.Println("Need to investigate why SMB initialization is stuck")
	}
}