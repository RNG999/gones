package main

import (
	"fmt"

	"gones/internal/bus"
	"gones/internal/cartridge"
)

func main() {
	fmt.Println("=== SMB NMI ENABLE FIX ===")
	
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

	fmt.Println("Monitoring timer countdown and implementing NMI fix...")
	
	frameCount := 0
	lastTimer := uint8(0xFF)
	
	// Run until timer reaches 0 or 300 frames max
	for frameCount < 300 {
		systemBus.Run(1)
		frameCount++
		
		// Check timer every frame
		currentTimer := systemBus.Memory.Read(0x07)
		
		// Monitor timer countdown
		if currentTimer != lastTimer {
			fmt.Printf("[Frame %d] Timer changed: 0x%02X -> 0x%02X\n", 
				frameCount, lastTimer, currentTimer)
			lastTimer = currentTimer
			
			// CRITICAL FIX: When timer reaches 0, enable NMI manually
			if currentTimer == 0 {
				fmt.Printf("\n*** TIMER REACHED 0! ENABLING NMI ***\n")
				
				// Enable NMI in PPUCTRL (set bit 7)
				currentPPUCTRL := systemBus.PPU.ReadRegister(0x2000)
				newPPUCTRL := currentPPUCTRL | 0x80
				
				fmt.Printf("PPUCTRL fix: 0x%02X -> 0x%02X (NMI enabled)\n", 
					currentPPUCTRL, newPPUCTRL)
				
				// Force write the corrected PPUCTRL value
				systemBus.PPU.WriteRegister(0x2000, newPPUCTRL)
				
				// Verify the fix worked
				verifyPPUCTRL := systemBus.PPU.ReadRegister(0x2000)
				nmiEnabled := (verifyPPUCTRL & 0x80) != 0
				
				fmt.Printf("Verification: PPUCTRL=0x%02X, NMI enabled=%v\n", 
					verifyPPUCTRL, nmiEnabled)
				
				if nmiEnabled {
					fmt.Println("\n*** SUCCESS! NMI ENABLED! Game should start! ***\n")
					break
				} else {
					fmt.Println("\n*** FAILED! NMI still not enabled ***\n")
				}
			}
		}
		
		// Progress indicator
		if frameCount%60 == 0 {
			ppuCtrl := systemBus.PPU.ReadRegister(0x2000)
			nmiEnabled := (ppuCtrl & 0x80) != 0
			fmt.Printf("[%ds] Timer=0x%02X, NMI=%v\n", 
				frameCount/60, currentTimer, nmiEnabled)
		}
	}
	
	fmt.Printf("\nCompleted %d frames\n", frameCount)
	
	// Final status
	finalTimer := systemBus.Memory.Read(0x07)
	finalPPUCTRL := systemBus.PPU.ReadRegister(0x2000)
	finalNMI := (finalPPUCTRL & 0x80) != 0
	
	fmt.Printf("Final state: Timer=0x%02X, PPUCTRL=0x%02X, NMI=%v\n", 
		finalTimer, finalPPUCTRL, finalNMI)
}