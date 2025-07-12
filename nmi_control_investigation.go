package main

import (
	"fmt"
	"os"
	"gones/internal/bus"
	"gones/internal/cartridge"
	"gones/internal/input"
)

func main() {
	fmt.Println("=== NMI CONTROL INVESTIGATION ===")
	fmt.Println("Investigating why NMI gets disabled and stays disabled")

	// Load SMB ROM
	rom, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create system
	system := bus.New()
	system.LoadCartridge(rom)

	fmt.Println("\nRunning frames to observe NMI control pattern...")
	
	// Track NMI state changes
	var nmiStates []bool
	var ppuControlValues []uint8
	var frameNumbers []int
	
	for frame := 0; frame < 30; frame++ {
		// Run one frame
		system.Run(1)
		
		// Check current PPUCTRL value for NMI enable bit
		ppuctrl := system.PPU.GetPPUCTRL()
		nmiEnabled := (ppuctrl & 0x80) != 0
		
		// Record when NMI state changes
		if len(nmiStates) == 0 || nmiEnabled != nmiStates[len(nmiStates)-1] {
			nmiStates = append(nmiStates, nmiEnabled)
			ppuControlValues = append(ppuControlValues, ppuctrl)
			frameNumbers = append(frameNumbers, frame)
			fmt.Printf("Frame %d: NMI state changed to %t (PPUCTRL=0x%02X)\n", 
				frame, nmiEnabled, ppuctrl)
		}
		
		// Test input during different NMI states
		if frame == 15 || frame == 25 {
			fmt.Printf("\n--- Testing input at frame %d (NMI enabled: %t) ---\n", frame, nmiEnabled)
			
			// Press Start button
			system.Input.Controller1.SetButton(input.Start, true)
			
			// Run 2 more frames to see response
			for i := 0; i < 2; i++ {
				system.Run(1)
				newPPUCTRL := system.PPU.GetPPUCTRL()
				newNMIEnabled := (newPPUCTRL & 0x80) != 0
				fmt.Printf("  Response frame %d: NMI=%t, PPUCTRL=0x%02X\n", 
					i+1, newNMIEnabled, newPPUCTRL)
			}
			
			// Release button
			system.Input.Controller1.SetButton(input.Start, false)
		}
	}

	fmt.Println("\n=== NMI STATE SUMMARY ===")
	for i, frame := range frameNumbers {
		fmt.Printf("Frame %d: NMI enabled = %t (PPUCTRL = 0x%02X)\n", 
			frame, nmiStates[i], ppuControlValues[i])
	}
	
	fmt.Println("\n=== ANALYSIS ===")
	if len(nmiStates) > 1 {
		fmt.Println("NMI state changes detected:")
		for i := 1; i < len(nmiStates); i++ {
			if nmiStates[i-1] && !nmiStates[i] {
				fmt.Printf("  Frame %d: NMI was DISABLED (0x%02X -> 0x%02X)\n", 
					frameNumbers[i], ppuControlValues[i-1], ppuControlValues[i])
			} else if !nmiStates[i-1] && nmiStates[i] {
				fmt.Printf("  Frame %d: NMI was ENABLED (0x%02X -> 0x%02X)\n", 
					frameNumbers[i], ppuControlValues[i-1], ppuControlValues[i])
			}
		}
	}
	
	// Final state
	finalPPUCTRL := system.PPU.GetPPUCTRL()
	finalNMI := (finalPPUCTRL & 0x80) != 0
	fmt.Printf("\nFinal state: NMI enabled = %t (PPUCTRL = 0x%02X)\n", finalNMI, finalPPUCTRL)
	
	if !finalNMI {
		fmt.Println("\n🔍 CONCLUSION: NMI is disabled in final state")
		fmt.Println("This explains why the game doesn't respond to input at title screen")
		fmt.Println("The game needs NMI to process controller input and advance game state")
	}
}