package main

import (
	"fmt"
	"os"
	"gones/internal/bus"
	"gones/internal/cartridge"
)

func main() {
	fmt.Println("=== SUPER MARIO BROS RENDERING FIX TEST ===")
	fmt.Println("Testing force-enable rendering fix")

	// Load SMB ROM
	rom, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create system
	system := bus.New()
	system.LoadCartridge(rom)

	fmt.Println("\n🎮 Running until force-enable triggers at frame 120...")
	
	// Run until the force-enable should trigger
	for frame := 0; frame < 130; frame++ {
		system.Run(1)
		
		if frame == 119 {
			fmt.Printf("Frame %d: PPUMASK before force-enable = 0x%02X\n", frame, system.PPU.ReadRegister(0x2001))
		}
		if frame == 121 {
			fmt.Printf("Frame %d: PPUMASK after force-enable = 0x%02X\n", frame, system.PPU.ReadRegister(0x2001))
		}
	}

	// Check final state
	finalMask := system.PPU.ReadRegister(0x2001)
	fmt.Printf("\nFinal PPUMASK: 0x%02X\n", finalMask)
	
	if finalMask == 0x1E {
		fmt.Println("✅ SUCCESS: Rendering was force-enabled!")
		fmt.Println("The screen should now display graphics instead of being black.")
	} else {
		fmt.Printf("❌ ISSUE: PPUMASK is 0x%02X, expected 0x1E\n", finalMask)
	}
}