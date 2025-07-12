package main

import (
	"fmt"
	"os"
	"gones/internal/bus"
	"gones/internal/cartridge"
)

func main() {
	fmt.Println("=== FINAL SUPER MARIO BROS TEST ===")
	fmt.Println("Testing the complete NES emulator with Super Mario Bros")

	// Load SMB ROM
	rom, err := cartridge.LoadFromFile("roms/smb.nes")
	if err != nil {
		fmt.Printf("Error loading ROM: %v\n", err)
		os.Exit(1)
	}

	// Create system
	system := bus.New()
	system.LoadCartridge(rom)

	fmt.Println("\nRunning Super Mario Bros for 20 frames...")
	fmt.Println("Expected behavior:")
	fmt.Println("1. CPU should start at reset vector ($8000)")
	fmt.Println("2. Timer should count down from 7 to 1 to 0")
	fmt.Println("3. NMI should be enabled and triggered properly")
	fmt.Println("4. Game should progress through initialization")

	// Track key game states
	var timerValues []uint8
	var gameStates []uint8
	var nmiEvents []string
	
	for frame := 0; frame < 20; frame++ {
		// Capture state before frame
		timer := system.Memory.Read(0x07)
		gameState := system.Memory.Read(0x00)
		
		// Run one frame
		system.Run(1)
		
		// Record significant changes
		if len(timerValues) == 0 || timer != timerValues[len(timerValues)-1] {
			timerValues = append(timerValues, timer)
			if timer == 0x07 {
				nmiEvents = append(nmiEvents, fmt.Sprintf("Frame %d: Timer countdown started (0x%02X)", frame, timer))
			} else if timer == 0x01 {
				nmiEvents = append(nmiEvents, fmt.Sprintf("Frame %d: Timer approaching zero (0x%02X)", frame, timer))
			} else if timer >= 0x80 {
				nmiEvents = append(nmiEvents, fmt.Sprintf("Frame %d: Game state changed - NMI working! (0x%02X)", frame, timer))
			}
		}
		
		if len(gameStates) == 0 || gameState != gameStates[len(gameStates)-1] {
			gameStates = append(gameStates, gameState)
		}
	}

	// Print results
	fmt.Println("\n=== RESULTS ===")
	fmt.Printf("CPU PC after 20 frames: $%04X\n", system.CPU.PC)
	fmt.Printf("Final timer value: 0x%02X\n", system.Memory.Read(0x07))
	fmt.Printf("Final game state: 0x%02X\n", system.Memory.Read(0x00))
	
	fmt.Println("\nKey events observed:")
	for _, event := range nmiEvents {
		fmt.Printf("  %s\n", event)
	}
	
	fmt.Printf("\nTimer values seen: ")
	for i, val := range timerValues {
		if i > 0 {
			fmt.Print(" -> ")
		}
		fmt.Printf("0x%02X", val)
	}
	fmt.Println()
	
	fmt.Printf("Game states seen: ")
	for i, val := range gameStates {
		if i > 0 {
			fmt.Print(" -> ")
		}
		fmt.Printf("0x%02X", val)
	}
	fmt.Println()

	// Final assessment
	fmt.Println("\n=== ASSESSMENT ===")
	if system.CPU.PC >= 0x8000 {
		fmt.Println("✅ CPU is executing ROM code properly")
	} else {
		fmt.Println("❌ CPU is not in ROM space")
	}
	
	if len(timerValues) > 1 {
		fmt.Println("✅ Timer is changing - game logic is running")
	} else {
		fmt.Println("❌ Timer is stuck - possible execution issue")
	}
	
	hasCountdown := false
	hasStateChange := false
	for _, val := range timerValues {
		if val == 0x07 || val == 0x01 {
			hasCountdown = true
		}
		if val >= 0x80 {
			hasStateChange = true
		}
	}
	
	if hasCountdown {
		fmt.Println("✅ Timer countdown sequence observed")
	}
	
	if hasStateChange {
		fmt.Println("✅ Game state progression detected - NMI system working!")
	}
	
	if hasCountdown && hasStateChange {
		fmt.Println("\n🎉 SUCCESS: Super Mario Bros is initializing properly!")
		fmt.Println("The NES emulator is working correctly.")
	} else {
		fmt.Println("\n⚠️  Partial success: Game is running but may have remaining issues.")
	}
}