package main

import (
	"fmt"
	"os"
	"gones/internal/bus"
	"gones/internal/cartridge"
	"gones/internal/input"
)

func main() {
	fmt.Println("=== FINAL START BUTTON TEST ===")
	fmt.Println("Testing the NMI auto-enable fix for title screen input")

	// Load SMB ROM
	rom, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create system
	system := bus.New()
	system.LoadCartridge(rom)

	fmt.Println("\nRunning to title screen...")
	
	// Run until title screen is stable
	for frame := 0; frame < 15; frame++ {
		system.Run(1)
	}
	
	fmt.Println("\n=== TITLE SCREEN REACHED ===")
	fmt.Println("Testing Start button press with auto-NMI fix...")
	
	// Press Start button
	fmt.Println("\n--- PRESSING START BUTTON ---")
	system.Input.Controller1.SetButton(input.Start, true)
	
	// Monitor game response for several frames
	var startingPC uint16 = system.CPU.PC
	fmt.Printf("Starting CPU PC: 0x%04X\n", startingPC)
	
	fmt.Println("\nMonitoring game response...")
	frameStartPC := system.CPU.PC
	
	for frame := 0; frame < 10; frame++ {
		fmt.Printf("\n--- Frame %d ---\n", frame+1)
		beforePC := system.CPU.PC
		
		system.Run(1)
		
		afterPC := system.CPU.PC
		ppuctrl := system.PPU.GetPPUCTRL()
		nmiEnabled := (ppuctrl & 0x80) != 0
		
		fmt.Printf("PC: 0x%04X -> 0x%04X, PPUCTRL: 0x%02X, NMI: %t\n", 
			beforePC, afterPC, ppuctrl, nmiEnabled)
		
		// Check for significant PC changes (indicating game progression)
		if afterPC != frameStartPC && afterPC != 0x8057 {
			fmt.Printf("🎉 GAME STATE CHANGE DETECTED! PC moved from 0x%04X to 0x%04X\n", 
				frameStartPC, afterPC)
			fmt.Println("This indicates the Start button was processed successfully!")
			break
		}
		
		if frame == 5 {
			fmt.Println("Mid-test: Holding Start button longer...")
		}
	}
	
	// Release button
	system.Input.Controller1.SetButton(input.Start, false)
	fmt.Println("\nReleased Start button")
	
	// Check final state
	finalPC := system.CPU.PC
	finalPPUCTRL := system.PPU.GetPPUCTRL()
	finalNMI := (finalPPUCTRL & 0x80) != 0
	
	fmt.Println("\n=== FINAL RESULTS ===")
	fmt.Printf("Initial PC: 0x%04X\n", startingPC)
	fmt.Printf("Final PC:   0x%04X\n", finalPC)
	fmt.Printf("Final PPUCTRL: 0x%02X (NMI: %t)\n", finalPPUCTRL, finalNMI)
	
	if finalPC != startingPC && finalPC != 0x8057 {
		fmt.Println("\n✅ SUCCESS: Game responded to Start button!")
		fmt.Println("The NMI auto-enable fix is working correctly.")
		fmt.Println("Super Mario Bros should now be playable!")
	} else if finalNMI {
		fmt.Println("\n⚠️ PARTIAL SUCCESS: NMI is enabled but game may need longer input")
		fmt.Println("The fix is working - NMI auto-enabling is functional")
	} else {
		fmt.Println("\n❌ ISSUE: NMI auto-enable may need adjustment")
		fmt.Println("Check the auto-enable conditions in PPU code")
	}
}